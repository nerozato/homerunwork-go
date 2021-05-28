package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/aws/aws-sdk-go/service/ses"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

//character encoding for email
const (
	emailCharSet                     = "UTF-8"
	snsAttributeSenderID             = "AWS.SNS.SMS.SenderID"
	snsAttributeSMSType              = "AWS.SNS.SMS.SMSType"
	snsAttributeSMSTypeTransactional = "Transactional"
)

//AWSSQSMsg : wrapper for an SQS message
type AWSSQSMsg struct {
	*sqs.Message
}

//AWSSQSMsgData : message data for an email received via SQS
type AWSSQSMsgData struct {
	MessageID string    `json:"MessageId"`
	TimeStamp time.Time `json:"Timestamp"`
	Mail      struct {
		MessageID   string    `json:"messageId"`
		TimeStamp   time.Time `json:"timestamp"`
		Source      string    `json:"source"`
		Destination []string  `json:"destination"`
		Headers     []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		} `json:"headers"`
		CommonHeaders struct {
			MessageID  string   `json:"messageId"`
			Date       string   `json:"date"`
			To         []string `json:"to"`
			Cc         []string `json:"cc"`
			Bcc        []string `json:"bcc"`
			From       []string `json:"from"`
			Sender     []string `json:"sender"`
			ReturnPath string   `json:"returnPath"`
			ReplyTo    []string `json:"reply-to"`
			Subject    string   `json:"subject"`
		} `json:"commonHeaders"`
	} `json:"mail"`
	Content string `json:"content"`
}

//SQSMessageFunc : definition of the function to process an SQS message
type SQSMessageFunc func(ctx context.Context, msg *AWSSQSMsg) error

//InitAWS : initialize AWS
func InitAWS(ctx context.Context) (*AWSSession, error) {
	_, logger := GetLogger(ctx)
	awsSession := &AWSSession{
		ctx:    ctx,
		logger: logger,
	}

	//initialize the session
	cfg := &aws.Config{
		Region: aws.String(GetAWSRegion()),
	}
	sess, err := session.NewSession(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "aws session")
	}
	awsSession.Session = sess

	//initalize the clients
	awsSession.clientSES = ses.New(sess)
	awsSession.clientSNS = sns.New(sess)
	awsSession.clientSQS = sqs.New(sess)
	awsSession.downloaderS3 = s3manager.NewDownloader(sess)
	awsSession.uploaderS3 = s3manager.NewUploader(sess)
	return awsSession, nil
}

//CreateReplyToAddress : create the reply-to address for emails, embedding the ids
func CreateReplyToAddress(id *uuid.UUID) (string, error) {
	//extract the user and domain from the email
	replyTo := GetEmailReplyTo()
	tokens := strings.Split(replyTo, "@")

	//create an email token
	token, err := GenerateEmailToken(id)
	if err != nil {
		return "", errors.Wrap(err, "generate email token")
	}

	//embed the token in the reply-to email address
	return fmt.Sprintf("%s+%s@%s", tokens[0], token, tokens[1]), nil
}

//function for splitting an email
func splitEmail(r rune) bool {
	return r == '+' || r == '@'
}

//ParseReplyToAddress : parse the reply-to email address, assuming a token is embedded
func ParseReplyToAddress(replyTo string) (*EmailID, error) {
	//extract the id components of the email
	tokens := strings.FieldsFunc(replyTo, splitEmail)

	//validate the token
	if len(tokens) != 3 {
		return nil, fmt.Errorf("invalid email: %s", replyTo)
	}
	ok, id, err := ValidateEmailToken(tokens[1])
	if err != nil {
		return nil, errors.Wrap(err, "validate email token")
	}
	if !ok {
		return nil, fmt.Errorf("invalid email token")
	}
	return id, nil
}

//AWSSession : AWS session and services
type AWSSession struct {
	Session      *session.Session
	clientSES    *ses.SES
	clientSNS    *sns.SNS
	clientSQS    *sqs.SQS
	downloaderS3 *s3manager.Downloader
	uploaderS3   *s3manager.Uploader
	ctx          context.Context
	logger       *Logger
	stopping     bool
}

//Stop : stop the session
func (s *AWSSession) Stop() {
	s.stopping = true
}

//ProcessSQSEmail : process the SQS queue for email
func (s *AWSSession) ProcessSQSEmail(fn SQSMessageFunc) {
	if fn == nil {
		return
	}
	count := GetSQSWorkerCount()
	for i := 0; i < count; i++ {
		go s.processSQSEmail(i, fn)
	}
	s.logger.Debugw("sqs email workers", "count", count)
}

//process the SQS queue for email
func (s *AWSSession) processSQSEmail(idx int, fn SQSMessageFunc) {
	s.logger.Infow("start sqs worker", "index", idx)
	for {
		//read messages from the queue
		queueURL := GetSQSQueueURLEmail()
		ctx, msgs, err := s.ReceiveSQSMsg(s.ctx, queueURL)
		if err != nil {
			s.logger.Errorw("receive sqs msg", "error", err, "index", idx, "queue", queueURL)
			return
		}
		start := time.Now()
		AddCtxStatsTime(ctx, ServerStatProcessIncomingEmail, start)
		AddCtxStatsCount(ctx, ServerStatProcessIncomingEmailCount, len(msgs))

		//process the messages
		count := 0
		var wg sync.WaitGroup
		for _, msg := range msgs {
			wg.Add(1)
			go func(m *sqs.Message, logger *Logger) {
				defer wg.Done()
				msg := &AWSSQSMsg{m}
				err := fn(ctx, msg)
				if err != nil {
					//log the error and allow the message to be removed
					logger.Errorw("process sqs msg", "error", err, "index", idx, "id", msg.MessageId)
				}
				s.DeleteSQSMsg(ctx, queueURL, m)
				count++
			}(msg, s.logger)
		}
		wg.Wait()
		s.logger.Debugw("process sql email", "count", count, "elapsedMS", FormatElapsedMS(start))
		if s.stopping {
			break
		}
	}
	s.logger.Infow("stop sqs worker", "index", idx)
}

//SendEmail : send an email
func (s *AWSSession) SendEmail(ctx context.Context, msg *Message) (context.Context, error) {
	if GetAWSEmailDisable() {
		return ctx, nil
	}
	ctx, logger := GetLogger(ctx)
	var in *ses.SendEmailInput
	var result *ses.SendEmailOutput
	start := time.Now()
	defer func() {
		//clear the email contents
		if in.Message.Body.Html != nil {
			in.Message.Body.Html.Data = aws.String(fmt.Sprintf("length: %d", len(*in.Message.Body.Html.Data)))
		}
		if in.Message.Body.Text != nil {
			in.Message.Body.Text.Data = aws.String(fmt.Sprintf("length: %d", len(*in.Message.Body.Text.Data)))
		}
		logger.Debugw("send email", "input", in, "result", result, "elapsedMS", FormatElapsedMS(start))
		AddCtxStatsAPI(ctx, ServerStatAPIAWS, "send email", time.Since(start))
	}()

	//prepare the subject
	if msg.Subject != "" {
		//generate the prefix
		prefix := GetEmailSubjectPrefix()
		if prefix != "" {
			prefix = fmt.Sprintf("%s: ", prefix)
		}

		//only prepend the prefix if the subject doesn't already
		//contain the prefix, especially for replies
		if !strings.Contains(msg.Subject, prefix) {
			msg.Subject = fmt.Sprintf("%s%s", prefix, msg.Subject)
		}
	}

	//prepare the email
	sourceEmail := GetEmailSender()
	if msg.SenderName != "" {
		sourceEmail = fmt.Sprintf("%s <%s>", msg.SenderName, sourceEmail)
	} else {
		sourceEmail = fmt.Sprintf("%s <%s>", GetEmailSenderName(), sourceEmail)
	}
	in = &ses.SendEmailInput{
		Source: aws.String(sourceEmail),
		Destination: &ses.Destination{
			ToAddresses: []*string{
				aws.String(msg.ToEmail),
			},
		},
		Message: &ses.Message{
			Subject: &ses.Content{
				Charset: aws.String(emailCharSet),
				Data:    aws.String(msg.Subject),
			},
			Body: &ses.Body{},
		},
	}

	//check for html content
	if msg.BodyHTML != "" {
		in.Message.Body.Html = &ses.Content{
			Charset: aws.String(emailCharSet),
			Data:    aws.String(msg.BodyHTML),
		}
	}

	//check for text content
	if msg.BodyText != "" {
		in.Message.Body.Text = &ses.Content{
			Charset: aws.String(emailCharSet),
			Data:    aws.String(msg.BodyText),
		}
	}

	//specify the reply-to address if a reply can be handled, when there is from-user or from-client
	if msg.FromUserID != nil || msg.FromClientID != nil {
		//generate an id if necessary
		if msg.ID == nil {
			msgID, err := uuid.NewV4()
			if err != nil {
				return ctx, errors.Wrap(err, "new uuid message")
			}
			msg.ID = &msgID
		}

		//prepare the reply-to address
		replyTo, err := CreateReplyToAddress(msg.ID)
		if err != nil {
			return ctx, errors.Wrap(err, "create reply-to")
		}
		in.ReplyToAddresses = []*string{aws.String(replyTo)}
	}

	//send the email
	var err error
	result, err = s.clientSES.SendEmail(in)
	if err != nil {
		return ctx, errors.Wrap(err, "send email")
	}
	return ctx, nil
}

//SendSMS : send an SMS message
func (s *AWSSession) SendSMS(ctx context.Context, msg *Message) (context.Context, error) {
	if GetAWSSMSDisable() {
		return ctx, nil
	}
	ctx, logger := GetLogger(ctx)
	var in *sns.PublishInput
	var result *sns.PublishOutput
	start := time.Now()
	defer func() {
		logger.Debugw("send sms", "input", in, "result", result, "elapsedMS", FormatElapsedMS(start))
		AddCtxStatsAPI(ctx, ServerStatAPIAWS, "send sms", time.Since(start))
	}()

	//prepare the message
	in = &sns.PublishInput{
		PhoneNumber: aws.String(msg.ToPhone),
		Message:     aws.String(msg.BodyText),
		MessageAttributes: map[string]*sns.MessageAttributeValue{
			snsAttributeSenderID: {
				StringValue: aws.String(GetAWSSMSSenderID()),
				DataType:    aws.String("String"),
			},
			snsAttributeSMSType: {
				StringValue: aws.String(snsAttributeSMSTypeTransactional),
				DataType:    aws.String("String"),
			},
		},
	}

	//send the sms text
	var err error
	result, err = s.clientSNS.Publish(in)
	if err != nil {
		return ctx, errors.Wrap(err, "send sms")
	}
	return ctx, nil
}

//ReceiveSQSMsg : receive an SQS message from the specified queue
func (s *AWSSession) ReceiveSQSMsg(ctx context.Context, queueURL string) (context.Context, []*sqs.Message, error) {
	ctx, logger := GetLogger(ctx)
	start := time.Now()
	var in *sqs.ReceiveMessageInput
	var result *sqs.ReceiveMessageOutput
	defer func() {
		//clear the messages
		count := len(result.Messages)
		result.Messages = nil
		logger.Debugw("sqs receive", "queue", queueURL, "in", in, "result", result, "count", count, "elapsedMS", FormatElapsedMS(start))
		AddCtxStatsAPI(ctx, ServerStatAPIAWS, "sqs receive", time.Since(start))
	}()

	//probe for messages in the queue
	in = &sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(queueURL),
		MaxNumberOfMessages: aws.Int64(10),
		WaitTimeSeconds:     aws.Int64(20),
		AttributeNames: aws.StringSlice([]string{
			"All",
		}),
		MessageAttributeNames: aws.StringSlice([]string{
			"All",
		}),
	}
	var err error
	result, err = s.clientSQS.ReceiveMessage(in)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "sqs receive")
	}
	return ctx, result.Messages, nil
}

//DeleteSQSMsg : delete a message from an SQS queue
func (s *AWSSession) DeleteSQSMsg(ctx context.Context, queueURL string, msg *sqs.Message) (context.Context, error) {
	ctx, logger := GetLogger(ctx)
	start := time.Now()
	var in *sqs.DeleteMessageInput
	var result *sqs.DeleteMessageOutput
	defer func() {
		logger.Debugw("sqs delete", "in", in, "result", result, "elapsedMS", FormatElapsedMS(start))
		AddCtxStatsAPI(ctx, ServerStatAPIAWS, "sqs delete", time.Since(start))
	}()

	//delete the message by its handle
	in = &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(queueURL),
		ReceiptHandle: msg.ReceiptHandle,
	}
	var err error
	result, err = s.clientSQS.DeleteMessage(in)
	if err != nil {
		return ctx, errors.Wrap(err, "sqs delete")
	}
	return ctx, nil
}

//UploadS3 : upload a file to S3 using a reader
func (s *AWSSession) UploadS3(ctx context.Context, location string, key string, reader io.Reader, contentType string) (context.Context, error) {
	ctx, logger := GetLogger(ctx)
	start := time.Now()
	var in *s3manager.UploadInput
	var result *s3manager.UploadOutput
	defer func() {
		logger.Debugw("file uploaded", "in", in, "result", result, "type", contentType, "elapsedMS", FormatElapsedMS(start))
		AddCtxStatsAPI(ctx, ServerStatAPIAWS, "file uploaded", time.Since(start))
	}()

	//upload the file
	bucket := GetAWSS3Bucket()
	fullKey := path.Join(GetAWSS3KeyBase(), location, key)
	in = &s3manager.UploadInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(fullKey),
		ACL:         aws.String("public-read"),
		ContentType: aws.String(contentType),
		Body:        reader,
	}
	var err error
	result, err = s.uploaderS3.Upload(in)
	if err != nil {
		return ctx, errors.Wrap(err, fmt.Sprintf("file upload: %s: %s", bucket, fullKey))
	}
	return ctx, nil
}

//UploadS3File : upload a file to S3
func (s *AWSSession) UploadS3File(ctx context.Context, location string, fileName string, contentType string) (context.Context, error) {
	ctx, logger := GetLogger(ctx)

	//open the file
	file, err := os.Open(fileName)
	if err != nil {
		return ctx, errors.Wrap(err, fmt.Sprintf("file open: %s", fileName))
	}
	defer func() {
		err = file.Close()
		if err != nil {
			logger.Warnw("file close", "error", err, "file", fileName)
		}
	}()
	key := path.Base(fileName)
	return s.UploadS3(ctx, location, key, file, contentType)
}

//DownloadS3 : download a file from S3 to a buffer
func (s *AWSSession) DownloadS3(ctx context.Context, keyBase string, key string) (context.Context, *bytes.Reader, error) {
	ctx, logger := GetLogger(ctx)
	start := time.Now()
	var in *s3.GetObjectInput
	var size int64
	defer func() {
		logger.Debugw("file downloaded", "in", in, "size", size, "elapsedMS", FormatElapsedMS(start))
		AddCtxStatsAPI(ctx, ServerStatAPIAWS, "file downloaded", time.Since(start))
	}()

	//download the file to a buffer
	buffer := &aws.WriteAtBuffer{}
	bucket := GetAWSS3Bucket()
	fullKey := path.Join(GetAWSS3KeyBase(), keyBase, key)
	in = &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(fullKey),
	}
	var err error
	size, err = s.downloaderS3.Download(buffer, in)
	if err != nil {
		return ctx, nil, errors.Wrap(err, fmt.Sprintf("file download: %s: %s %s", key, bucket, key))
	}

	//use a reader for the data
	reader := bytes.NewReader(buffer.Bytes())
	return ctx, reader, nil
}
