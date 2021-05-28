package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"runtime/debug"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/robfig/cron/v3"
)

//length of the buffer for channels
const channelBufferLen = 100

//InitScheduler : initialize the scheduler
func InitScheduler(ctx context.Context, server *Server) *Scheduler {
	scheduler := &Scheduler{
		ctx:    ctx,
		label:  "main",
		server: server,
	}
	scheduler.Executor = cron.New(
		cron.WithChain(scheduler.panicHdlr()),
		cron.WithLogger(scheduler),
	)

	//add jobs
	cron := GetCronProcessGoogle()
	if cron != "" {
		scheduler.Executor.AddFunc(cron, scheduler.ProcessGoogle)
	}
	cron = GetCronProcessImgs()
	if cron != "" {
		scheduler.Executor.AddFunc(cron, scheduler.ProcessImgs)
	}
	cron = GetCronProcessMsgs()
	if cron != "" {
		scheduler.Executor.AddFunc(cron, scheduler.ProcessMsgs)
	}
	cron = GetCronProcessNotifications()
	if cron != "" {
		scheduler.Executor.AddFunc(cron, scheduler.ProcessNotifications)
	}
	cron = GetCronProcessRecurringBookings()
	if cron != "" {
		scheduler.Executor.AddFunc(cron, scheduler.ProcessRecurringBookings)
	}
	cron = GetCronProcessZoom()
	if cron != "" {
		scheduler.Executor.AddFunc(cron, scheduler.ProcessZoom)
	}
	return scheduler
}

//Scheduler : cron scheduler
type Scheduler struct {
	Executor *cron.Cron
	ctx      context.Context
	label    string
	server   *Server
}

//Info : log informational message
func (s *Scheduler) Info(msg string, keysAndValues ...interface{}) {
	s.server.logger.Debugw(msg, keysAndValues...)
}

//Error : log error
func (s *Scheduler) Error(err error, msg string, keysAndValues ...interface{}) {
	keysAndValues = append(keysAndValues, "error", err)
	s.server.logger.Errorw(msg, keysAndValues...)
}

//handle job panics
func (s *Scheduler) panicHdlr() cron.JobWrapper {
	return func(j cron.Job) cron.Job {
		return cron.FuncJob(func() {
			defer func() {
				if !GetPanicHandlerDisable() {
					err := recover()
					if err != nil {
						s.server.logger.Warnw("job panic", "error", err, "stack", string(debug.Stack()))
						AddCtxStatsCount(s.ctx, ServerStatLogPanics, 1)
					}
				}
			}()
			j.Run()
		})
	}
}

//Start : start the scheduler
func (s *Scheduler) Start() {
	s.Executor.Start()
	s.server.logger.Infow("start scheduler", "label", s.label)
}

//Stop : stop the scheduler
func (s *Scheduler) Stop() {
	s.Executor.Stop()
	s.server.logger.Infow("stop scheduler", "label", s.label)
}

//ProcessGoogle : process Google calendars and events
func (s *Scheduler) ProcessGoogle() {
	start := time.Now()
	defer func() {
		s.server.logger.Debugw("process google", "elapsedMS", FormatElapsedMS(start))
	}()
	s.server.stats.AddTime(ServerStatProcessGoogle, start)
	db := s.server.getDB()
	s.processGoogleCalendars(db)
	s.processGoogleEvents(db)
}

//process Google calendars
func (s *Scheduler) processGoogleCalendars(db *DB) {
	//list the providers to process
	ctx, providers, err := ListProviderCalendarsToProcessForGoogle(s.ctx, db, GetBatchSizeProcessGoogleCalendars())
	if err != nil {
		s.server.logger.Errorw("list providers process", "error", err)
		return
	}

	//process the providers
	for _, provider := range providers {
		//check if creating or updating a calendar
		if provider.GoogleCalendarID == nil {
			data, err := s.server.createProviderCalendarGoogle(ctx, provider)
			if err != nil {
				s.server.logger.Errorw("create google calendar", "error", err)
				continue
			}
			provider.GoogleCalendarID = &data.Id
			provider.GoogleCalendarData = data
		} else {
			data, err := s.server.updateProviderCalendarGoogle(ctx, provider)
			if err != nil {
				s.server.logger.Errorw("update google calendar", "error", err)
				continue
			}
			provider.GoogleCalendarData = data
		}

		//update the provider
		ctx, err = UpdateProviderCalendar(ctx, db, provider)
		if err != nil {
			s.server.logger.Errorw("update provider calendar", "error", err)
		}
	}
}

//process Google events
func (s *Scheduler) processGoogleEvents(db *DB) {
	//list the bookings to process
	ctx, books, err := ListBookingEventsToProcessForGoogle(s.ctx, db, GetBatchSizeProcessGoogleEvents())
	if err != nil {
		s.server.logger.Errorw("list bookings process google", "error", err)
		return
	}

	//process the bookings
	var data *EventGoogle
	for _, book := range books {
		bookUI := s.server.createBookingUI(book)

		//check if creating or delete an event
		if book.EventGoogleID != nil {
			if book.EventGoogleDelete {
				data, err = s.server.cancelEventGoogle(ctx, bookUI)
				if err != nil {
					s.server.logger.Errorw("cancel google event", "error", err)
					continue
				}
			} else if book.EventGoogleUpdate {
				ctx, data, err = s.server.updateEventGoogle(ctx, bookUI)
				if err != nil {
					s.server.logger.Errorw("update google event", "error", err)
					continue
				}
			}
		} else {
			ctx, data, err = s.server.createEventGoogle(ctx, bookUI)
			if err != nil {
				s.server.logger.Errorw("create google event", "error", err)
				continue
			}
			book.EventGoogleID = &data.Id
		}

		//update the booking
		ctx, err = UpdateBookingEventGoogle(ctx, db, bookUI.Booking, data)
		if err != nil {
			s.server.logger.Errorw("update booking google event", "error", err)
		}
	}
}

//ProcessImgs : process images for resizing
func (s *Scheduler) ProcessImgs() {
	start := time.Now()
	defer func() {
		s.server.logger.Debugw("process imgs", "elapsedMS", FormatElapsedMS(start))
	}()
	s.server.stats.AddTime(ServerStatProcessImgs, start)
	db := s.server.getDB()

	//list the images to process
	ctx, imgs, err := ListImgsToProcess(s.ctx, db, GetBatchSizeProcessImgs())
	if err != nil {
		s.server.logger.Errorw("list images process", "error", err)
		return
	}

	//process the images
	for _, img := range imgs {
		var inFile *os.File
		var outFile *os.File
		var buffer *bytes.Buffer
		var reader io.Reader
		var writer io.Writer

		//determine the target dimensions
		width, height, doCrop, err := GetTargetDimensions(img.Type)
		if err != nil {
			s.server.logger.Warnw("target dimensions", "error", err, "type", img.Type)
			continue
		}

		//generate the target file name, appending the target dimensions
		targetName := fmt.Sprintf("%s.%d.%d.%d.jpg", FileNoExt(img.FileSrc), img.Type, width, height)
		targetDir := path.Join(URLAssetUpload, img.Path)

		//get the image to process
		if GetAWSS3Enable() {
			//download the image from s3
			ctx, reader, err = s.server.awsSession.DownloadS3(ctx, targetDir, img.FileSrc)
			if err != nil {
				s.server.logger.Warnw("download image", "error", err, "path", targetDir, "file", img.GetFile())
				continue
			}

			//prepare the writer to the buffer
			buffer = new(bytes.Buffer)
			writer = buffer
		} else {
			//load the file from the local system
			localFile := path.Join(UploadAssetPathLocal, img.GetFile())
			inFile, err = os.Open(localFile)
			if err != nil {
				s.server.logger.Warnw("open image read", "error", err, "file", localFile)
				continue
			}
			reader = inFile

			//prepare the output file
			targetFile := path.Join(UploadAssetPathLocal, img.Path, targetName)
			outFile, err = os.Create(targetFile)
			if err != nil {
				s.server.logger.Warnw("open image write", "error", err, "path", img.Path, "file", targetFile)
				continue
			}
			writer = outFile
		}

		//resize the image
		ctx, err = ResizeImg(ctx, reader, width, height, doCrop, writer)
		if err != nil {
			s.server.logger.Errorw("resize image", "error", err, "img", img.GetFile())
			continue
		}

		//persist the image
		if GetAWSS3Enable() {
			data := buffer.Bytes()
			dataReader := bytes.NewReader(data)
			contentType := http.DetectContentType(data)
			ctx, err = s.server.awsSession.UploadS3(ctx, targetDir, targetName, dataReader, contentType)
			if err != nil {
				s.server.logger.Errorw("upload img", "error", err, "img", targetName)
				continue
			}
		} else {
			//close the files
			err = inFile.Close()
			if err != nil {
				s.server.logger.Errorw("close file in", "error", err, "path", targetDir, "file", img.GetFile())
			}
			err = outFile.Close()
			if err != nil {
				s.server.logger.Errorw("close file out", "error", err, "path", targetDir, "file", img.GetFile())
			}
		}

		//save the image in the db
		img.FileResized = targetName
		ctx, err = SaveImg(ctx, db, img)
		if err != nil {
			s.server.logger.Errorw("save image", "error", err, "file", targetName)
		}
	}
}

//ProcessMsgs : process messages
func (s *Scheduler) ProcessMsgs() {
	start := time.Now()
	defer func() {
		s.server.logger.Debugw("process emails", "elapsedMS", FormatElapsedMS(start))
	}()
	s.server.stats.AddTime(ServerStatProcessMsgs, start)
	db := s.server.getDB()

	//list the messages to process
	ctx, msgs, err := ListMsgsToProcess(s.ctx, db, GetBatchSizeProcessEmails())
	if err != nil {
		s.server.logger.Errorw("list messages process", "error", err)
		return
	}
	if len(msgs) == 0 {
		return
	}

	//process the messages
	processedMsgs := make([]*Message, 0, 2)
	for _, msg := range msgs {
		ctx, err = s.sendMsg(ctx, msg)
		if err != nil {
			s.server.logger.Errorw("send email", "error", err, "id", msg.ID)
			continue
		}
		processedMsgs = append(processedMsgs, msg)
	}

	//mark the messages
	ctx, err = MarkMsgsProcessed(ctx, db, processedMsgs)
	if err != nil {
		s.server.logger.Errorw("mark messages", "error", err)
	}
}

//send messages
func (s *Scheduler) sendMsg(ctx context.Context, msg *Message) (context.Context, error) {
	db := s.server.getDB()

	//load data for the message
	var err error
	var bookUI *bookingUI
	var campaignUI *campaignUI
	var client *Client
	var paymentUI *paymentUI
	var providerUI *providerUI
	var user *User
	switch msg.Type {
	//booking-related
	case MsgTypeBookingCancelClient:
		fallthrough
	case MsgTypeBookingCancelProvider:
		fallthrough
	case MsgTypeBookingConfirmClient:
		fallthrough
	case MsgTypeBookingEditClient:
		fallthrough
	case MsgTypeBookingNewClient:
		fallthrough
	case MsgTypeBookingNewProvider:
		fallthrough
	case MsgTypeBookingReminderClient:
		fallthrough
	case MsgTypeBookingReminderProvider:
		var book *Booking
		ctx, book, err = LoadBookingByID(ctx, db, msg.SecondaryID, false, false)
		if err != nil {
			return ctx, errors.Wrap(err, fmt.Sprintf("load booking: %s", msg.SecondaryID))
		}
		bookUI = s.server.createBookingUI(book)
		ctx, provider, err := LoadProviderByID(ctx, db, book.Provider.ID)
		if err != nil {
			return ctx, errors.Wrap(err, fmt.Sprintf("load provider: %s", book.Provider.ID))
		}
		bookUI.Provider = provider
		ctx, svc, err := LoadServiceByProviderIDAndID(ctx, db, provider.ID, book.Service.ID)
		if err != nil {
			return ctx, errors.Wrap(err, fmt.Sprintf("load service: %s", book.Service.ID))
		}
		bookUI.Service = svc

	//client-related
	case MsgTypeClientInvite:
		fallthrough
	case MsgTypeContact:
		ctx, client, err = LoadClientByID(ctx, db, msg.SecondaryID)
		if err != nil {
			return ctx, errors.Wrap(err, fmt.Sprintf("load client: %s", msg.SecondaryID))
		}
		var provider *Provider
		ctx, provider, err = LoadProviderByID(ctx, db, client.ProviderID)
		if err != nil {
			return ctx, errors.Wrap(err, fmt.Sprintf("load provider: %s", client.ProviderID))
		}
		providerUI = s.server.createProviderUI(provider)

	//payment-related
	case MsgTypeInvoice:
		fallthrough
	case MsgTypeInvoiceInternal:
		fallthrough
	case MsgTypePaymentClient:
		fallthrough
	case MsgTypePaymentProvider:
		var payment *Payment
		ctx, payment, err = LoadPaymentByID(ctx, db, msg.SecondaryID)
		if err != nil {
			return ctx, errors.Wrap(err, fmt.Sprintf("load payment: %s", msg.SecondaryID))
		}
		paymentUI = s.server.createPaymentUI(payment)
		var provider *Provider
		ctx, provider, err = LoadProviderByID(ctx, db, payment.ProviderID)
		if err != nil {
			return ctx, errors.Wrap(err, fmt.Sprintf("load provider: %s", payment.ProviderID))
		}
		providerUI = s.server.createProviderUI(provider)

	//provider-related
	case MsgTypeDomainNotification:
		fallthrough
	case MsgTypeProviderUserInvite:
		fallthrough
	case MsgTypeWelcome:
		var provider *Provider
		ctx, provider, err = LoadProviderByID(ctx, db, msg.SecondaryID)
		if err != nil {
			return ctx, errors.Wrap(err, fmt.Sprintf("load provider: %s", msg.SecondaryID))
		}
		providerUI = s.server.createProviderUI(provider)

	//user-related
	case MsgTypeEmailVerify:
		ctx, user, err = LoadUserByID(ctx, db, msg.SecondaryID)
		if err != nil {
			return ctx, errors.Wrap(err, fmt.Sprintf("load user: %s", msg.SecondaryID))
		}

	//campaign-related
	case MsgTypeCampaignAddNotification:
		fallthrough
	case MsgTypeCampaignAddProvider:
		fallthrough
	case MsgTypeCampaignPaymentNotification:
		fallthrough
	case MsgTypeCampaignStatusProvider:
		var campaign *Campaign
		ctx, campaign, err = LoadCampaignByID(ctx, db, msg.SecondaryID)
		if err != nil {
			return ctx, errors.Wrap(err, fmt.Sprintf("load campaign: %s", msg.SecondaryID))
		}
		campaignUI = s.server.createCampaignUI(campaign)
		var provider *Provider
		ctx, provider, err = LoadProviderByID(ctx, db, campaign.ProviderID)
		if err != nil {
			return ctx, errors.Wrap(err, fmt.Sprintf("load provider: %s", campaign.ProviderID))
		}
		providerUI = s.server.createProviderUI(provider)

	case MsgTypePwdReset:
	default:
		return ctx, fmt.Errorf("invalid message type: %s", msg.Type)
	}

	//create the subject and body
	send := true
	var subject string
	var bodyHTML string
	var bodyText string
	switch msg.Type {
	case MsgTypeBookingCancelClient:
		ctx, subject, bodyHTML, err = s.server.createEmailBookingCancelClient(ctx, bookUI)
		if err != nil {
			return ctx, errors.Wrap(err, fmt.Sprintf("create booking email cancel client: %s", msg.ID))
		}

		//set-up the SMS text
		if msg.ToPhone != "" {
			url := ForceURLAbs(ctx, bookUI.GetURLViewClient())
			ctx, urlShort, err := ShortenURLBitly(ctx, url)
			if err != nil {
				return ctx, errors.Wrap(err, fmt.Sprintf("create booking sms cancel client url shorten: %s", msg.ID))
			}
			bodyText = GetSMSText(MsgTypeBookingCancelClient, urlShort.URL)
		}
	case MsgTypeBookingCancelProvider:
		ctx, subject, bodyHTML, err = s.server.createEmailBookingCancelProvider(ctx, bookUI)
		if err != nil {
			return ctx, errors.Wrap(err, fmt.Sprintf("create booking email cancel provider: %s", msg.ID))
		}

		//set-up the SMS text
		if msg.ToPhone != "" {
			url := ForceURLAbs(ctx, bookUI.GetURLView())
			ctx, urlShort, err := ShortenURLBitly(ctx, url)
			if err != nil {
				return ctx, errors.Wrap(err, fmt.Sprintf("create booking sms cancel provider url shorten: %s", msg.ID))
			}
			bodyText = GetSMSText(MsgTypeBookingCancelProvider, urlShort.URL)
		}
	case MsgTypeBookingConfirmClient:
		ctx, subject, bodyHTML, err = s.server.createEmailBookingConfirmClient(ctx, bookUI)
		if err != nil {
			return ctx, errors.Wrap(err, fmt.Sprintf("create booking email confirm client: %s", msg.ID))
		}

		//set-up the SMS text
		if msg.ToPhone != "" {
			url := ForceURLAbs(ctx, bookUI.GetURLViewClient())
			ctx, urlShort, err := ShortenURLBitly(ctx, url)
			if err != nil {
				return ctx, errors.Wrap(err, fmt.Sprintf("create booking sms confirm client url shorten: %s", msg.ID))
			}
			bodyText = GetSMSText(MsgTypeBookingConfirmClient, urlShort.URL)
		}
	case MsgTypeBookingEditClient:
		ctx, subject, bodyHTML, err = s.server.createEmailBookingEditClient(ctx, bookUI)
		if err != nil {
			return ctx, errors.Wrap(err, fmt.Sprintf("create booking email edit client: %s", msg.ID))
		}

		//set-up the SMS text
		if msg.ToPhone != "" {
			url := ForceURLAbs(ctx, bookUI.GetURLViewClient())
			ctx, urlShort, err := ShortenURLBitly(ctx, url)
			if err != nil {
				return ctx, errors.Wrap(err, fmt.Sprintf("create booking sms edit client url shorten: %s", msg.ID))
			}
			bodyText = GetSMSText(MsgTypeBookingEditClient, urlShort.URL)
		}
	case MsgTypeBookingNewClient:
		ctx, subject, bodyHTML, err = s.server.createEmailBookingNewClient(ctx, bookUI, msg.IsClient)
		if err != nil {
			return ctx, errors.Wrap(err, fmt.Sprintf("create booking email new client: %s", msg.ID))
		}

		//set-up the SMS text
		if msg.ToPhone != "" {
			url := ForceURLAbs(ctx, bookUI.GetURLViewClient())
			ctx, urlShort, err := ShortenURLBitly(ctx, url)
			if err != nil {
				return ctx, errors.Wrap(err, fmt.Sprintf("create booking sms new client url shorten: %s", msg.ID))
			}
			bodyText = GetSMSText(MsgTypeBookingNewClient, urlShort.URL)
		}
	case MsgTypeBookingNewProvider:
		ctx, subject, bodyHTML, send, err = s.server.createEmailBookingNewProvider(ctx, bookUI, msg.IsClient)
		if err != nil {
			return ctx, errors.Wrap(err, fmt.Sprintf("create booking email new provider: %s", msg.ID))
		}

		//set-up the SMS text
		if msg.ToPhone != "" {
			url := ForceURLAbs(ctx, bookUI.GetURLView())
			ctx, urlShort, err := ShortenURLBitly(ctx, url)
			if err != nil {
				return ctx, errors.Wrap(err, fmt.Sprintf("create booking sms new provider url shorten: %s", msg.ID))
			}
			bodyText = GetSMSText(MsgTypeBookingNewProvider, urlShort.URL)
		}
	case MsgTypeBookingReminderClient:
		ctx, subject, bodyHTML, err = s.server.createEmailBookingReminderClient(ctx, bookUI)
		if err != nil {
			return ctx, errors.Wrap(err, fmt.Sprintf("create booking email reminder client: %s", msg.ID))
		}

		//set-up the SMS text
		if msg.ToPhone != "" {
			url := ForceURLAbs(ctx, bookUI.GetURLViewClient())
			ctx, urlShort, err := ShortenURLBitly(ctx, url)
			if err != nil {
				return ctx, errors.Wrap(err, fmt.Sprintf("create booking sms reminder client url shorten: %s", msg.ID))
			}
			bodyText = GetSMSText(MsgTypeBookingReminderClient, urlShort.URL)
		}
	case MsgTypeBookingReminderProvider:
		ctx, subject, bodyHTML, err = s.server.createEmailBookingReminderProvider(ctx, bookUI)
		if err != nil {
			return ctx, errors.Wrap(err, fmt.Sprintf("create booking email reminder provider: %s", msg.ID))
		}

		//set-up the SMS text
		if msg.ToPhone != "" {
			url := ForceURLAbs(ctx, bookUI.GetURLView())
			ctx, urlShort, err := ShortenURLBitly(ctx, url)
			if err != nil {
				return ctx, errors.Wrap(err, fmt.Sprintf("create booking sms reminder url shorten: %s", msg.ID))
			}
			bodyText = GetSMSText(MsgTypeBookingReminderProvider, urlShort.URL)
		}
	case MsgTypeCampaignAddNotification:
		ctx, subject, bodyHTML, err = s.server.createEmailCampaignAddNotification(ctx, providerUI, campaignUI)
		if err != nil {
			return ctx, errors.Wrap(err, fmt.Sprintf("create email campaign add notification: %s", msg.ID))
		}
	case MsgTypeCampaignAddProvider:
		ctx, subject, bodyHTML, err = s.server.createEmailCampaignAddProvider(ctx, providerUI, campaignUI)
		if err != nil {
			return ctx, errors.Wrap(err, fmt.Sprintf("create email campaign add provider: %s", msg.ID))
		}
	case MsgTypeCampaignPaymentNotification:
		ctx, subject, bodyHTML, err = s.server.createEmailCampaignPaymentNotification(ctx, providerUI, campaignUI)
		if err != nil {
			return ctx, errors.Wrap(err, fmt.Sprintf("create email campaign payment notification: %s", msg.ID))
		}
	case MsgTypeCampaignStatusProvider:
		ctx, subject, bodyHTML, err = s.server.createEmailCampaignStatusProvider(ctx, providerUI, campaignUI)
		if err != nil {
			return ctx, errors.Wrap(err, fmt.Sprintf("create email campaign add provider: %s", msg.ID))
		}
	case MsgTypeClientInvite:
		ctx, subject, bodyHTML, err = s.server.createEmailClientInvite(ctx, providerUI, client)
		if err != nil {
			return ctx, errors.Wrap(err, fmt.Sprintf("create email invite client: %s", msg.ID))
		}
	case MsgTypeContact:
		ctx, subject, bodyHTML, err = s.server.createEmailContact(ctx, providerUI, client, msg.Text)
		if err != nil {
			return ctx, errors.Wrap(err, fmt.Sprintf("create email contact: %s", msg.ID))
		}
	case MsgTypeDomainNotification:
		ctx, subject, bodyHTML, err = s.server.createEmailDomainNotification(ctx, providerUI)
		if err != nil {
			return ctx, errors.Wrap(err, fmt.Sprintf("create email domain notification: %s", msg.ID))
		}
	case MsgTypeEmailVerify:
		ctx, subject, bodyHTML, err = s.server.createEmailVerify(ctx, user, msg.TokenURL)
		if err != nil {
			return ctx, errors.Wrap(err, fmt.Sprintf("create email verify: %s", msg.ID))
		}
	case MsgTypeInvoice:
		ctx, subject, bodyHTML, err = s.server.createEmailInvoice(ctx, providerUI, paymentUI)
		if err != nil {
			return ctx, errors.Wrap(err, fmt.Sprintf("create email invoice: %s", msg.ID))
		}

		//set-up the SMS text
		if msg.ToPhone != "" {
			url := ForceURLAbs(ctx, paymentUI.URL)
			ctx, urlShort, err := ShortenURLBitly(ctx, url)
			if err != nil {
				return ctx, errors.Wrap(err, fmt.Sprintf("create sms invoice url shorten: %s", msg.ID))
			}
			bodyText = GetSMSText(MsgTypeInvoice, urlShort.URL)
		}
	case MsgTypeInvoiceInternal:
		ctx, subject, bodyHTML, err = s.server.createEmailInvoiceInternal(ctx, providerUI, paymentUI)
		if err != nil {
			return ctx, errors.Wrap(err, fmt.Sprintf("create email invoice: %s", msg.ID))
		}

		//set-up the SMS text
		if msg.ToPhone != "" {
			url := ForceURLAbs(ctx, paymentUI.URL)
			ctx, urlShort, err := ShortenURLBitly(ctx, url)
			if err != nil {
				return ctx, errors.Wrap(err, fmt.Sprintf("create sms invoice url shorten: %s", msg.ID))
			}
			bodyText = GetSMSText(MsgTypeInvoiceInternal, urlShort.URL)
		}
	case MsgTypeMessage:
		subject = msg.Subject
		bodyText = msg.BodyText
	case MsgTypePaymentClient:
		ctx, subject, bodyHTML, err = s.server.createEmailPaymentClient(ctx, providerUI, paymentUI)
		if err != nil {
			return ctx, errors.Wrap(err, fmt.Sprintf("create email payment client: %s", msg.ID))
		}
	case MsgTypePaymentProvider:
		ctx, subject, bodyHTML, err = s.server.createEmailPaymentProvider(ctx, providerUI, paymentUI)
		if err != nil {
			return ctx, errors.Wrap(err, fmt.Sprintf("create email payment provider: %s", msg.ID))
		}

		//set-up the SMS
		if msg.ToPhone != "" {
			url := ForceURLAbs(ctx, paymentUI.GetURLView())
			ctx, urlShort, err := ShortenURLBitly(ctx, url)
			if err != nil {
				return ctx, errors.Wrap(err, fmt.Sprintf("create sms payment provider url shorten: %s", msg.ID))
			}
			bodyText = GetSMSText(MsgTypePaymentProvider, paymentUI.Name, urlShort.URL)
		}
	case MsgTypePwdReset:
		ctx, subject, bodyHTML, err = s.server.createEmailPwdReset(ctx, msg.TokenURL)
		if err != nil {
			return ctx, errors.Wrap(err, fmt.Sprintf("create email password reset: %s", msg.ID))
		}
	case MsgTypeProviderUserInvite:
		ctx, subject, bodyHTML, err = s.server.createEmailProviderUserInvite(ctx, providerUI, msg.ToEmail)
		if err != nil {
			return ctx, errors.Wrap(err, fmt.Sprintf("create email provider user invite: %s", msg.ID))
		}
	case MsgTypeWelcome:
		ctx, subject, bodyHTML, err = s.server.createEmailWelcome(ctx, providerUI)
		if err != nil {
			return ctx, errors.Wrap(err, fmt.Sprintf("create email welcome: %s", msg.ID))
		}

	default:
		return ctx, fmt.Errorf("invalid message type: %s", msg.Type)
	}

	//send an email
	msg.Subject = subject
	msg.BodyHTML = bodyHTML
	msg.BodyText = bodyText
	if send {
		//send the email
		if msg.ToEmail != "" {
			_, err := s.server.awsSession.SendEmail(ctx, msg)
			if err != nil {
				return ctx, errors.Wrap(err, fmt.Sprintf("send email: %s", msg.ID))
			}
		}

		//send an SMS text if possible
		if msg.ToPhone != "" && msg.BodyText != "" {
			_, err := s.server.awsSession.SendSMS(ctx, msg)
			if err != nil {
				return ctx, errors.Wrap(err, fmt.Sprintf("send sms: %s", msg.ID))
			}
		}
	}
	return ctx, nil
}

//ProcessNotifications : process notifications
func (s *Scheduler) ProcessNotifications() {
	start := time.Now()
	defer func() {
		s.server.logger.Debugw("process notifications", "elapsedMS", FormatElapsedMS(start))
	}()
	s.server.stats.AddTime(ServerStatProcessNotifications, start)
	db := s.server.getDB()

	//create notifications for bookings
	now := GetTimeNow("")
	ctx, err := CreateBookingNotifications(s.ctx, db, now, NotificationTypeBookingReminder, GetBatchSizeProcessNotifications())
	if err != nil {
		s.server.logger.Errorw("create notifications", "error", err)
		return
	}

	//list the notifications to process
	ctx, notifications, err := ListNotificationsToProcess(ctx, db, now, GetBatchSizeProcessNotifications())
	if err != nil {
		s.server.logger.Errorw("list notifications process", "error", err)
		return
	}
	if len(notifications) == 0 {
		return
	}

	//process the notifications
	processedNotifications := make([]*Notification, 0, 2)
	for _, notification := range notifications {
		switch notification.Type {
		case NotificationTypeBookingReminder:
			bookUI := s.server.createBookingUI(notification.Booking)
			ctx, err = s.server.queueEmailsBookingReminder(ctx, bookUI)
			if err != nil {
				s.server.logger.Errorw("queue email booking reminder", "error", err, "id", bookUI.ID)
				continue
			}
			processedNotifications = append(processedNotifications, notification)
		default:
			s.server.logger.Errorw("invalid notification type", "type", notification.Type)
		}
	}

	//mark the notifications
	ctx, err = MarkNotificationsProcessed(ctx, db, processedNotifications)
	if err != nil {
		s.server.logger.Errorw("mark notifications", "error", err)
	}
}

//ProcessRecurringBookings : process recurring bookings
func (s *Scheduler) ProcessRecurringBookings() {
	start := time.Now()
	defer func() {
		s.server.logger.Debugw("process recurring bookings", "elapsedMS", FormatElapsedMS(start))
	}()
	s.server.stats.AddTime(ServerStatRecurringOrders, start)
	db := s.server.getDB()

	//list the bookings to process
	now := time.Now()
	ctx, books, err := ListRecurringToProcess(s.ctx, db, now, GetBatchSizeProcessRecurringBookings())
	if err != nil {
		s.server.logger.Errorw("list bookings process", "error", err)
	}

	//process the bookings
	for _, book := range books {
		//generate the next batch of recurring events
		ctx, ruleEnd, err := SaveBookingsRecurring(ctx, db, book, *book.RecurrenceInstanceEnd, book.Confirmed, true)
		if err != nil {
			s.server.logger.Errorw("save recurring", "error", err, "id", book.ID)
			continue
		}
		book.RecurrenceInstanceEnd = &ruleEnd

		//update the booking
		ctx, err = UpdateBookingRecurring(ctx, db, book)
		if err != nil {
			s.server.logger.Errorw("update booking recurring", "error", err)
		}
	}
}

//ProcessZoom : process Zoom meetings
func (s *Scheduler) ProcessZoom() {
	start := time.Now()
	defer func() {
		s.server.logger.Debugw("process zoom", "elapsedMS", FormatElapsedMS(start))
	}()
	s.server.stats.AddTime(ServerStatZoom, start)
	db := s.server.getDB()

	//list the bookings to process
	ctx, books, err := ListBookingEventsToProcessForZoom(s.ctx, db, GetBatchSizeProcessZoomMeetings())
	if err != nil {
		s.server.logger.Errorw("list bookings process zoom", "error", err)
		return
	}

	//track refreshed tokens to allow re-use of that token for this run
	tokens := make(map[string]*TokenZoom, 1)

	//process the bookings
	var data *MeetingZoom
	for _, book := range books {
		bookUI := s.server.createBookingUI(book)

		//check for a new token
		user := book.GetUser()
		token, ok := tokens[user.ID.String()]
		if ok {
			user.ZoomToken = token
		}

		//check if creating or delete an event
		if book.EnableZoom && user.ZoomToken != nil {
			if book.MeetingZoomID != nil {
				if book.MeetingZoomDelete {
					token, err = s.server.cancelBookingMeetingZoom(ctx, bookUI)
					if err != nil {
						s.server.logger.Errorw("create zoom meeting", "error", err)
						continue
					}
					data = book.MeetingZoomData
				} else if book.MeetingZoomUpdate {
					ctx, token, data, err = s.server.updateMeetingZoom(ctx, bookUI)
					if err != nil {
						s.server.logger.Errorw("update zoom meeting", "error", err)
						continue
					}
				}
			} else {
				ctx, token, data, err = s.server.createMeetingZoom(ctx, bookUI)
				if err != nil {
					s.server.logger.Errorw("cancel zoom meeting", "error", err)
					continue
				}
				meetingID := strconv.Itoa(data.ID)
				book.MeetingZoomID = &meetingID
			}
		}

		//update the booking
		ctx, err = UpdateBookingMeetingZoom(ctx, db, bookUI.Booking, data)
		if err != nil {
			s.server.logger.Errorw("update booking zoom meeting", "error", err)
		}

		//update the user if the token has been refreshed
		if token != nil {
			user.ZoomToken = token
			ctx, err = UpdateUserTokenZoom(ctx, db, user)
			if err != nil {
				s.server.logger.Errorw("update user zoom token", "error", err)
			}
			tokens[user.ID.String()] = token
		}
	}
}
