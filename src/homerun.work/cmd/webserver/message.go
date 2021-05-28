package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

//message db tables
const (
	dbTableMessage = "message"
)

//MsgType : message type
type MsgType string

//message types
const (
	MsgTypeBookingCancelClient         MsgType = "bookingCancelClient"
	MsgTypeBookingCancelProvider       MsgType = "bookingCancelProvider"
	MsgTypeBookingConfirmClient        MsgType = "bookingConfirmClient"
	MsgTypeBookingEditClient           MsgType = "bookingEditClient"
	MsgTypeBookingNewClient            MsgType = "bookingNewClient"
	MsgTypeBookingNewProvider          MsgType = "bookingNewProvider"
	MsgTypeBookingReminderClient       MsgType = "bookingReminderClient"
	MsgTypeBookingReminderProvider     MsgType = "bookingReminderProvider"
	MsgTypeCampaignAddNotification     MsgType = "campaignAddNotification"
	MsgTypeCampaignAddProvider         MsgType = "campaignAddProvider"
	MsgTypeCampaignPaymentNotification MsgType = "campaignPaymentNotification"
	MsgTypeCampaignStatusProvider      MsgType = "campaignStatusProvider"
	MsgTypeClientInvite                MsgType = "clientInvite"
	MsgTypeContact                     MsgType = "contact"
	MsgTypeDomainNotification          MsgType = "domainNotification"
	MsgTypeEmailVerify                 MsgType = "emailVerify"
	MsgTypeInvoice                     MsgType = "invoice"
	MsgTypeInvoiceInternal             MsgType = "invoiceInternal"
	MsgTypeMessage                     MsgType = "message"
	MsgTypePaymentClient               MsgType = "paymentClient"
	MsgTypePaymentProvider             MsgType = "paymentProvider"
	MsgTypePwdReset                    MsgType = "pwdReset"
	MsgTypeProviderUserInvite          MsgType = "providerUserInvite"
	MsgTypeWelcome                     MsgType = "welcome"
)

//Message : definition of a message
type Message struct {
	ID           *uuid.UUID `json:"-"`
	SecondaryID  *uuid.UUID `json:"-"`
	FromClientID *uuid.UUID `json:"-"`
	FromUserID   *uuid.UUID `json:"-"`
	ToClientID   *uuid.UUID `json:"-"`
	ToUserID     *uuid.UUID `json:"-"`
	ToEmail      string     `json:"-"`
	ToPhone      string     `json:"ToPhone"`
	SenderName   string     `json:"SenderName"`
	IsClient     bool       `json:"IsClient"`
	Type         MsgType    `json:"Type"`
	Subject      string     `json:"Subject"`
	BodyHTML     string     `json:"BodyHtml"`
	BodyText     string     `json:"BodyText"`
	Text         string     `json:"Text"`
	TokenURL     string     `json:"TokenUrl"`
}

//LoadMsgByID : load a message
func LoadMsgByID(ctx context.Context, db *DB, id *uuid.UUID) (context.Context, *Message, error) {
	stmt := fmt.Sprintf("SELECT BIN_TO_UUID(secondary_id),BIN_TO_UUID(from_user_id),BIN_TO_UUID(from_client_id),BIN_TO_UUID(to_user_id),BIN_TO_UUID(to_client_id),to_email,data FROM %s WHERE id=UUID_TO_BIN(?)", dbTableMessage)
	ctx, row, err := db.QueryRow(ctx, stmt, id)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "query row message")
	}

	//read the row
	var secondaryIDStr string
	var fromUserIDStr sql.NullString
	var fromClientIDStr sql.NullString
	var toUserIDStr sql.NullString
	var toClientIDStr sql.NullString
	var email string
	var dataStr string
	err = row.Scan(&secondaryIDStr, &fromUserIDStr, &fromClientIDStr, &toUserIDStr, &toClientIDStr, &email, &dataStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return ctx, nil, fmt.Errorf("no message: %s", id)
		}
		return ctx, nil, errors.Wrap(err, "select message")
	}

	//parse the uuid
	secondaryID, err := uuid.FromString(secondaryIDStr)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "parse uuid message secondary id")
	}
	var fromUserID *uuid.UUID
	if fromUserIDStr.Valid {
		id, err := uuid.FromString(fromUserIDStr.String)
		if err != nil {
			return ctx, nil, errors.Wrap(err, "parse uuid message from user id")
		}
		fromUserID = &id
	}
	var fromClientID *uuid.UUID
	if fromClientIDStr.Valid {
		id, err := uuid.FromString(fromClientIDStr.String)
		if err != nil {
			return ctx, nil, errors.Wrap(err, "parse uuid message from client id")
		}
		fromClientID = &id
	}
	var toUserID *uuid.UUID
	if toUserIDStr.Valid {
		id, err := uuid.FromString(toUserIDStr.String)
		if err != nil {
			return ctx, nil, errors.Wrap(err, "parse uuid message to user id")
		}
		toUserID = &id
	}
	var toClientID *uuid.UUID
	if toClientIDStr.Valid {
		id, err := uuid.FromString(toClientIDStr.String)
		if err != nil {
			return ctx, nil, errors.Wrap(err, "parse uuid message to client id")
		}
		toClientID = &id
	}

	//unmarshal the data
	var msg Message
	err = json.Unmarshal([]byte(dataStr), &msg)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "unjson message")
	}
	msg.ID = id
	msg.SecondaryID = &secondaryID
	msg.FromUserID = fromUserID
	msg.FromClientID = fromClientID
	msg.ToUserID = toUserID
	msg.ToClientID = toClientID
	msg.ToEmail = email
	return ctx, &msg, nil
}

//SaveMsg : save a message
func SaveMsg(ctx context.Context, db *DB, msg *Message) (context.Context, error) {
	//generate an id if necessary
	if msg.ID == nil {
		msgID, err := uuid.NewV4()
		if err != nil {
			return ctx, errors.Wrap(err, "new uuid message")
		}
		msg.ID = &msgID
	}

	//json encode the message data
	msgJSON, err := json.Marshal(msg)
	if err != nil {
		return ctx, errors.Wrap(err, "json message")
	}

	//save to the db
	stmt := fmt.Sprintf("INSERT INTO %s(id,secondary_id,from_user_id,from_client_id,to_user_id,to_client_id,to_email,data) VALUES (UUID_TO_BIN(?),UUID_TO_BIN(?),UUID_TO_BIN(?),UUID_TO_BIN(?),UUID_TO_BIN(?),UUID_TO_BIN(?),?,?)", dbTableMessage)
	ctx, result, err := db.Exec(ctx, stmt, msg.ID, msg.SecondaryID, msg.FromUserID, msg.FromClientID, msg.ToUserID, msg.ToClientID, msg.ToEmail, msgJSON)
	if err != nil {
		return ctx, errors.Wrap(err, "insert message")
	}
	count, err := result.RowsAffected()
	if err != nil {
		return ctx, errors.Wrap(err, "insert message rows affected")
	}
	if count == 0 {
		return ctx, fmt.Errorf("unable to insert message: %s", msg.ToEmail)
	}
	return ctx, nil
}

//ListMsgsToProcess : list messages to process
func ListMsgsToProcess(ctx context.Context, db *DB, limit int) (context.Context, []*Message, error) {
	ctx, logger := GetLogger(ctx)
	var err error
	var msgs []*Message
	ctx, err = db.ProcessTx(ctx, "list messages process", func(ctx context.Context, db *DB) (context.Context, error) {
		stmt := fmt.Sprintf("SELECT BIN_TO_UUID(id),BIN_TO_UUID(secondary_id),BIN_TO_UUID(from_user_id),BIN_TO_UUID(from_client_id),BIN_TO_UUID(to_user_id),BIN_TO_UUID(to_client_id),to_email,data FROM %s WHERE deleted=0 AND ((email_processing=0 AND email_processed=0) OR (sms_processing=0 AND sms_processed=0)) ORDER BY created LIMIT %d", dbTableMessage, limit)
		ctx, rows, err := db.Query(ctx, stmt)
		if err != nil {
			return ctx, errors.Wrap(err, "select messages")
		}
		defer func() {
			err := rows.Close()
			if err != nil {
				logger.Warnw("rows close", "error", err)
			}
		}()

		//read the rows
		msgs = make([]*Message, 0, 2)
		var idStr string
		var secondaryIDStr string
		var fromUserIDStr sql.NullString
		var fromClientIDStr sql.NullString
		var toUserIDStr sql.NullString
		var toClientIDStr sql.NullString
		var email string
		var dataStr string
		for rows.Next() {
			err := rows.Scan(&idStr, &secondaryIDStr, &fromUserIDStr, &fromClientIDStr, &toUserIDStr, &toClientIDStr, &email, &dataStr)
			if err != nil {
				return ctx, errors.Wrap(err, "rows scan messages")
			}

			//parse the uuid
			id, err := uuid.FromString(idStr)
			if err != nil {
				return ctx, errors.Wrap(err, "parse uuid message id")
			}
			secondaryID, err := uuid.FromString(secondaryIDStr)
			if err != nil {
				return ctx, errors.Wrap(err, "parse uuid message secondary id")
			}
			var fromUserID *uuid.UUID
			if fromUserIDStr.Valid {
				id, err := uuid.FromString(fromUserIDStr.String)
				if err != nil {
					return ctx, errors.Wrap(err, "parse uuid message from user id")
				}
				fromUserID = &id
			}
			var fromClientID *uuid.UUID
			if fromClientIDStr.Valid {
				id, err := uuid.FromString(fromClientIDStr.String)
				if err != nil {
					return ctx, errors.Wrap(err, "parse uuid message from client id")
				}
				fromClientID = &id
			}
			var toUserID *uuid.UUID
			if toUserIDStr.Valid {
				id, err := uuid.FromString(toUserIDStr.String)
				if err != nil {
					return ctx, errors.Wrap(err, "parse uuid message to user id")
				}
				toUserID = &id
			}
			var toClientID *uuid.UUID
			if toClientIDStr.Valid {
				id, err := uuid.FromString(toClientIDStr.String)
				if err != nil {
					return ctx, errors.Wrap(err, "parse uuid message to client id")
				}
				toClientID = &id
			}

			//unmarshal the data
			var msg Message
			err = json.Unmarshal([]byte(dataStr), &msg)
			if err != nil {
				return ctx, errors.Wrap(err, "unjson message")
			}
			msg.ID = &id
			msg.SecondaryID = &secondaryID
			msg.FromUserID = fromUserID
			msg.FromClientID = fromClientID
			msg.ToUserID = toUserID
			msg.ToClientID = toClientID
			msg.ToEmail = email
			msgs = append(msgs, &msg)
		}
		if len(msgs) == 0 {
			return ctx, nil
		}

		//mark the messages as being processed
		MarkMsgsProcessing(ctx, db, msgs)
		return ctx, nil
	})
	if err != nil {
		return ctx, nil, errors.Wrap(err, "list messages process")
	}
	return ctx, msgs, nil
}

//MarkMsgsProcessing : mark messages as processing
func MarkMsgsProcessing(ctx context.Context, db *DB, msgs []*Message) (context.Context, error) {
	lenMsgs := len(msgs)
	if lenMsgs == 0 {
		return ctx, fmt.Errorf("no messages to mark")
	}

	//prepare the ids
	args := make([]interface{}, lenMsgs)
	for i, img := range msgs {
		args[i] = img.ID.String()
	}

	//generate the list of parameters to use in the query
	paramsStr := fmt.Sprintf("(UUID_TO_BIN(?)%s)", strings.Repeat(",UUID_TO_BIN(?)", lenMsgs-1))

	//mark the messages
	stmt := fmt.Sprintf("UPDATE %s SET email_processing=1,email_processing_time=CURRENT_TIMESTAMP(),sms_processing=1,sms_processing_time=CURRENT_TIMESTAMP() WHERE (email_processing=0 OR sms_processing=0) AND id in %s", dbTableMessage, paramsStr)
	ctx, result, err := db.Exec(ctx, stmt, args...)
	if err != nil {
		return ctx, errors.Wrap(err, "mark messages processing")
	}
	count, err := result.RowsAffected()
	if err != nil {
		return ctx, errors.Wrap(err, "mark messages processing rows affected")
	}
	if int(count) != lenMsgs {
		return ctx, fmt.Errorf("unable to mark messages processing: %d: %d", count, lenMsgs)
	}
	return ctx, nil
}

//MarkMsgsProcessed : mark messages as processed
func MarkMsgsProcessed(ctx context.Context, db *DB, msgs []*Message) (context.Context, error) {
	lenMsgs := len(msgs)
	if lenMsgs == 0 {
		return ctx, fmt.Errorf("no messages to mark")
	}

	//prepare the ids
	args := make([]interface{}, lenMsgs)
	for i, img := range msgs {
		args[i] = img.ID.String()
	}

	//generate the list of parameters to use in the query
	paramsStr := fmt.Sprintf("(UUID_TO_BIN(?)%s)", strings.Repeat(",UUID_TO_BIN(?)", lenMsgs-1))

	//mark the messages
	stmt := fmt.Sprintf("UPDATE %s SET email_processing=0,email_processed=1,sms_processing=0,sms_processed=1 WHERE ((email_processing=1 AND email_processed=0) OR (sms_processing=1 AND sms_processed=0)) AND id in %s", dbTableMessage, paramsStr)
	ctx, result, err := db.Exec(ctx, stmt, args...)
	if err != nil {
		return ctx, errors.Wrap(err, "mark messages processed")
	}
	count, err := result.RowsAffected()
	if err != nil {
		return ctx, errors.Wrap(err, "mark messages processed rows affected")
	}
	if int(count) != lenMsgs {
		return ctx, fmt.Errorf("unable to mark messages processed: %d: %d", count, lenMsgs)
	}
	return ctx, nil
}
