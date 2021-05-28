package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

//notification tables
const (
	dbTableNotification = "notification"
)

//NotificationType : notification type
type NotificationType int

//notification types
const (
	NotificationTypeBookingReminder NotificationType = iota + 1
)

//Notification : definition of a notification
type Notification struct {
	ID          *uuid.UUID       `json:"-"`
	UserID      *uuid.UUID       `json:"-"`
	SecondaryID *uuid.UUID       `json:"-"`
	ExternalID  *string          `json:"-"`
	Type        NotificationType `json:"-"`
	TimeStart   string           `json:"TimeStart"`
	TimeEnd     string           `json:"TimeEnd"`
	Booking     *Booking         `json:"-"`
}

//CreateNotification : create a notification
func CreateNotification(ctx context.Context, db *DB, notification *Notification, sendTime time.Time) (context.Context, error) {
	//json encode the notification data
	notificationJSON, err := json.Marshal(notification)
	if err != nil {
		return ctx, errors.Wrap(err, "json notification")
	}

	//save the notification
	stmt := fmt.Sprintf("INSERT INTO %s(id,user_id,secondary_id,external_id,type,send_date,data) VALUES (UUID_TO_BIN(UUID()),UUID_TO_BIN(?),UUID_TO_BIN(?),?,?,?,?)", dbTableNotification)
	ctx, result, err := db.Exec(ctx, stmt, notification.UserID, notification.SecondaryID, notification.ExternalID, notification.Type, sendTime, notificationJSON)
	if err != nil {
		return ctx, errors.Wrap(err, "insert notification")
	}
	count, err := result.RowsAffected()
	if err != nil {
		return ctx, errors.Wrap(err, "insert notification rows affected")
	}
	if count != 1 {
		return ctx, fmt.Errorf("unable to insert notification: %s: %s", notification.UserID, notification.SecondaryID)
	}
	return ctx, nil
}

//CreateBookingNotifications : create the notifications for bookings that should be processed
func CreateBookingNotifications(ctx context.Context, db *DB, now time.Time, notificationType NotificationType, limit int) (context.Context, error) {
	ctx, err := db.ProcessTx(ctx, "create booking notifications", func(ctx context.Context, db *DB) (context.Context, error) {
		ctx, logger := GetLogger(ctx)

		//compute the times used to find bookings
		duration := time.Duration(GetNotificationBookingReminderMin()) * time.Minute
		checkBookingTime := now.UTC().Add(duration)
		checkCreateTime := now.UTC().Add(-2 * duration)

		//process bookings
		stmt := fmt.Sprintf("SELECT BIN_TO_UUID(p.user_id),BIN_TO_UUID(b.id),b.time_start,b.time_end FROM %s b INNER JOIN %s s ON s.id=b.service_id INNER JOIN %s p ON p.id=s.provider_id INNER JOIN %s u on u.id=p.user_id LEFT JOIN %s n ON n.deleted=0 AND n.secondary_id=b.id AND n.type=%d WHERE u.disable_emails=0 AND b.deleted=0 AND b.confirmed=1 AND n.id IS NULL AND ?>b.time_start AND ?<b.time_end AND ?>b.created ORDER BY b.time_start LIMIT %d", dbTableBooking, dbTableService, dbTableProvider, dbTableUser, dbTableNotification, notificationType, limit)
		ctx, rows, err := db.Query(ctx, stmt, checkBookingTime, checkBookingTime, checkCreateTime)
		if err != nil {
			return ctx, errors.Wrap(err, "select booking notifications")
		}
		defer func() {
			err := rows.Close()
			if err != nil {
				logger.Warnw("rows close", "error", err)
			}
		}()

		//read the rows
		notifications := make([]*Notification, 0, 2)
		var userIDStr string
		var bookIDStr string
		var timeStart time.Time
		var timeEnd time.Time
		for rows.Next() {
			err := rows.Scan(&userIDStr, &bookIDStr, &timeStart, &timeEnd)
			if err != nil {
				return ctx, errors.Wrap(err, "rows scan booking notifications")
			}

			//parse the uuid
			userID, err := uuid.FromString(userIDStr)
			if err != nil {
				return ctx, errors.Wrap(err, "parse uuid user id")
			}
			bookID, err := uuid.FromString(bookIDStr)
			if err != nil {
				return ctx, errors.Wrap(err, "parse uuid booking id")
			}

			//prepare to store the notification
			notification := &Notification{
				UserID:      &userID,
				SecondaryID: &bookID,
				Type:        notificationType,
				TimeStart:   timeStart.Format(time.RFC3339),
				TimeEnd:     timeEnd.Format(time.RFC3339),
			}
			notifications = append(notifications, notification)
		}

		//save the notifications
		for _, notification := range notifications {
			ctx, err = CreateNotification(ctx, db, notification, now)
			if err != nil {
				return ctx, errors.Wrap(err, fmt.Sprintf("create notification: %s: %s", notification.UserID, notification.SecondaryID))
			}
		}
		return ctx, nil
	})
	if err != nil {
		return ctx, errors.Wrap(err, "create booking notifications")
	}
	return ctx, nil
}

//state tracking the booking
type bookingState struct {
	ID        *uuid.UUID
	CheckDate time.Time
}

//ListNotificationsToProcess : list notifications to process
func ListNotificationsToProcess(ctx context.Context, db *DB, now time.Time, limit int) (context.Context, []*Notification, error) {
	ctx, logger := GetLogger(ctx)
	var err error
	var notifications []*Notification
	ctx, err = db.ProcessTx(ctx, "list notifications process", func(ctx context.Context, db *DB) (context.Context, error) {
		stmt := fmt.Sprintf("SELECT BIN_TO_UUID(id),BIN_TO_UUID(user_id),BIN_TO_UUID(secondary_id),external_id,type,data FROM %s WHERE deleted=0 AND processing=0 AND processed=0 AND ?>=send_date ORDER BY created LIMIT %d", dbTableNotification, limit)
		ctx, rows, err := db.Query(ctx, stmt, now)
		if err != nil {
			return ctx, errors.Wrap(err, "select notifications")
		}
		defer func() {
			err := rows.Close()
			if err != nil {
				logger.Warnw("rows close", "error", err)
			}
		}()

		//read the rows
		notifications = make([]*Notification, 0, 2)
		var idStr string
		var userIDStr string
		var secondaryIDStr string
		var externalID sql.NullString
		var notificationType NotificationType
		var dataStr string
		for rows.Next() {
			err := rows.Scan(&idStr, &userIDStr, &secondaryIDStr, &externalID, &notificationType, &dataStr)
			if err != nil {
				return ctx, errors.Wrap(err, "rows scan notifications")
			}

			//parse the uuid
			id, err := uuid.FromString(idStr)
			if err != nil {
				return ctx, errors.Wrap(err, "parse uuid notification id")
			}
			userID, err := uuid.FromString(userIDStr)
			if err != nil {
				return ctx, errors.Wrap(err, "parse uuid user id")
			}
			secondaryID, err := uuid.FromString(secondaryIDStr)
			if err != nil {
				return ctx, errors.Wrap(err, "parse uuid secondary id")
			}

			//unmarshal the data
			var notification Notification
			err = json.Unmarshal([]byte(dataStr), &notification)
			if err != nil {
				return ctx, errors.Wrap(err, "unjson notification")
			}
			notification.ID = &id
			notification.UserID = &userID
			notification.SecondaryID = &secondaryID
			if externalID.Valid {
				notification.ExternalID = &externalID.String
			}
			notification.Type = notificationType
			notifications = append(notifications, &notification)
		}
		if len(notifications) == 0 {
			return ctx, nil
		}

		//mark the notifications as being processed
		MarkNotificationsProcessing(ctx, db, notifications)

		//load data for the notifications
		for _, notification := range notifications {
			switch notification.Type {
			case NotificationTypeBookingReminder:
				//load the booking
				ctx, book, err := LoadBookingByID(ctx, db, notification.SecondaryID, false, false)
				if err != nil {
					return ctx, errors.Wrap(err, fmt.Sprintf("load booking: %s", notification.SecondaryID))
				}
				notification.Booking = book

				//use the notification times
				timeFrom, err := time.Parse(time.RFC3339, notification.TimeStart)
				if err != nil {
					return ctx, errors.Wrap(err, fmt.Sprintf("parse time: %s", notification.TimeStart))
				}
				timeTo, err := time.Parse(time.RFC3339, notification.TimeEnd)
				if err != nil {
					return ctx, errors.Wrap(err, fmt.Sprintf("parse time: %s", notification.TimeEnd))
				}
				book.TimeFrom = timeFrom
				book.TimeTo = timeTo
			default:
				return ctx, fmt.Errorf("invalid notification type: %d", notification.Type)
			}
		}
		return ctx, nil
	})
	if err != nil {
		return ctx, nil, errors.Wrap(err, "list notifications process")
	}
	return ctx, notifications, nil
}

//DeleteUnSentNotifications : delete unsent notifications
func DeleteUnSentNotifications(ctx context.Context, db *DB, secondaryID *uuid.UUID) (context.Context, error) {
	stmt := fmt.Sprintf("UPDATE %s SET deleted=1 WHERE deleted=0 AND processed=0 AND send_date>CURRENT_TIMESTAMP() AND secondary_id=UUID_TO_BIN(?)", dbTableNotification)
	ctx, _, err := db.Exec(ctx, stmt, secondaryID)
	if err != nil {
		return ctx, errors.Wrap(err, "delete unsent notification")
	}
	return ctx, nil
}

//MarkNotificationsProcessing : mark notifications as processing
func MarkNotificationsProcessing(ctx context.Context, db *DB, notifications []*Notification) (context.Context, error) {
	lenNotifications := len(notifications)
	if lenNotifications == 0 {
		return ctx, fmt.Errorf("no notficiations to mark")
	}

	//prepare the ids
	args := make([]interface{}, lenNotifications)
	for i, img := range notifications {
		args[i] = img.ID.String()
	}

	//generate the list of parameters to use in the query
	paramsStr := fmt.Sprintf("(UUID_TO_BIN(?)%s)", strings.Repeat(",UUID_TO_BIN(?)", lenNotifications-1))

	//mark the notifications
	stmt := fmt.Sprintf("UPDATE %s SET processing=1,processing_time=CURRENT_TIMESTAMP() WHERE processing=0 AND id in %s", dbTableNotification, paramsStr)
	ctx, result, err := db.Exec(ctx, stmt, args...)
	if err != nil {
		return ctx, errors.Wrap(err, "mark notifications processing")
	}
	count, err := result.RowsAffected()
	if err != nil {
		return ctx, errors.Wrap(err, "mark notifications processing rows affected")
	}
	if int(count) != lenNotifications {
		return ctx, fmt.Errorf("unable to mark notifications processing: %d: %d", count, lenNotifications)
	}
	return ctx, nil
}

//MarkNotificationsProcessed : mark notifications as processed
func MarkNotificationsProcessed(ctx context.Context, db *DB, notifications []*Notification) (context.Context, error) {
	lenNotifications := len(notifications)
	if lenNotifications == 0 {
		return ctx, fmt.Errorf("no notifications to mark")
	}

	//prepare the ids
	args := make([]interface{}, lenNotifications)
	for i, img := range notifications {
		args[i] = img.ID.String()
	}

	//generate the list of parameters to use in the query
	paramsStr := fmt.Sprintf("(UUID_TO_BIN(?)%s)", strings.Repeat(",UUID_TO_BIN(?)", lenNotifications-1))

	//mark the notifications
	stmt := fmt.Sprintf("UPDATE %s SET processing=0,processed=1 WHERE processing=1 AND processed=0 AND id in %s", dbTableNotification, paramsStr)
	ctx, result, err := db.Exec(ctx, stmt, args...)
	if err != nil {
		return ctx, errors.Wrap(err, "mark notifications processed")
	}
	count, err := result.RowsAffected()
	if err != nil {
		return ctx, errors.Wrap(err, "mark notifications processed rows affected")
	}
	if int(count) != lenNotifications {
		return ctx, fmt.Errorf("unable to mark notifications processed: %d: %d", count, lenNotifications)
	}
	return ctx, nil
}
