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

//booking db tables
const (
	dbTableBooking = "service_booking"
)

//booking constants
const (
	recurringGenerateMonths  = 2 //how far out to generate recurring events
	recurringLookAheadMonths = 1 //how far out to look ahead for recurring events
)

//Booking : definition of a booking
type Booking struct {
	ID                    *uuid.UUID          `json:"-"`
	ParentID              *uuid.UUID          `json:"-"`
	RecurrenceStart       *time.Time          `json:"-"`
	RecurrenceFreq        RecurrenceInterval  `json:"RecurrenceFreq"`
	RecurrenceFreqChange  bool                `json:"-"`
	RecurrenceFreqLabel   string              `json:"RecurrenceFreqLabel"`
	RecurrenceRules       []string            `json:"-"`
	RecurrenceInstanceEnd *time.Time          `json:"-"`
	Location              string              `json:"Location"`
	LocationType          ServiceLocationType `json:"LocationType"`
	ClientCreated         bool                `json:"ClientCreated"`
	Description           string              `json:"Description"`
	EnableClientPhone     bool                `json:"EnableClientPhone"`
	ProviderNote          string              `json:"ProviderNote"`
	ServiceName           string              `json:"ServiceName"`
	ServicePadding        int                 `json:"ServicePadding"`
	ServicePrice          float32             `json:"ServicePrice"`
	ServicePriceOriginal  float32             `json:"ServicePriceOriginal"`
	ServicePriceType      PriceType           `json:"ServicePriceType"`
	ServiceDuration       int                 `json:"ServiceDuration"`
	ServiceDurationLabel  string              `json:"ServiceDurationLabel"`
	CouponCode            string              `json:"CouponCode"`
	CouponCodeChange      bool                `json:"-"`
	Coupon                *Coupon             `json:"Coupon"`
	TimeFrom              time.Time           `json:"-"`
	TimeTo                time.Time           `json:"-"`
	TimeFromPadded        time.Time           `json:"-"`
	TimeToPadded          time.Time           `json:"-"`
	TimeChange            bool                `json:"-"`
	Confirmed             bool                `json:"-"`
	Deleted               bool                `json:"-"`
	Created               time.Time           `json:"-"`

	//google metadata
	EventGoogleID     *string `json:"-"`
	EventGoogleDelete bool    `json:"-"`
	EventGoogleUpdate bool    `json:"-"`
	MeetingZoomID     *string `json:"-"`

	//zoom metadata
	EnableZoom        bool         `json:"EnableZoom"`
	MeetingZoomDelete bool         `json:"-"`
	MeetingZoomUpdate bool         `json:"-"`
	MeetingZoomData   *MeetingZoom `json:"-"`

	//linked data
	ProviderUserID *uuid.UUID    `json:"-"`
	ProviderUser   *ProviderUser `json:"-"`
	Provider       *Provider     `json:"-"`
	Service        *Service      `json:"-"`
	ServiceType    ServiceType   `json:"-"`
	Client         *Client       `json:"-"`
	Payment        *Payment      `json:"-"`
}

//GetUser : return the user that created the booking
func (b *Booking) GetUser() *User {
	if b.ProviderUser != nil {
		return b.ProviderUser.User
	}
	return b.Provider.User
}

//SetProviderUser : set the provider user that created the booking
func (b *Booking) SetProviderUser(user *ProviderUser) {
	if user == nil {
		b.ProviderUserID = nil
		b.ProviderUser = nil
		return
	}
	b.ProviderUserID = user.ID
	b.ProviderUser = user
}

//IsMappable : check if the location is mappable
func (b *Booking) IsMappable() bool {
	if b.Location == "" {
		return false
	}
	return IsMappable(b.Location)
}

//GetClientPhoneSMS : get the client phone to use for SMS
func (b *Booking) GetClientPhoneSMS() string {
	if b.EnableClientPhone {
		return b.Client.Phone
	}
	return ""
}

//IsApptOnly : check if appointment-only
func (b *Booking) IsApptOnly() bool {
	return IsApptOnly(b.ServiceType)
}

//IsRecurring : flag indicating if the booking is recurring
func (b *Booking) IsRecurring() bool {
	return b.RecurrenceFreq != RecurrenceIntervalOnce
}

//IsCancelled : check if a booking has been cancelled
func (b *Booking) IsCancelled() bool {
	return b.Deleted
}

//IsInvoiced : check if a payment has been invoiced
func (b *Booking) IsInvoiced() bool {
	return b.Payment != nil && b.Payment.IsInvoiced()
}

//IsPaid : check if a payment has been paid
func (b *Booking) IsPaid() bool {
	return b.Payment != nil && b.Payment.IsPaid()
}

//IsCaptured : check if a payment has been captured
func (b *Booking) IsCaptured() bool {
	return b.Payment != nil && b.Payment.IsCaptured()
}

//IsEditable : check if a payment is editable
func (b *Booking) IsEditable(now time.Time) bool {
	return (b.Payment == nil || !b.Payment.IsCaptured()) && now.Before(b.TimeFrom)
}

//SupportsPayment : check if payments are supported
func (b *Booking) SupportsPayment() bool {
	return b.ServicePrice != 0 && b.Provider.SupportsPayment()
}

//SetRecurrenceFreq : set the recurrence frequency
func (b *Booking) SetRecurrenceFreq(freq *RecurrenceFreq, resetStart bool) error {
	//default the frequency if not set
	if freq == nil {
		freq = &RecurrenceFreqOnce
		b.RecurrenceStart = nil
	}

	//update the frequency
	if b.RecurrenceFreq != freq.Value {
		b.RecurrenceFreqChange = true
		b.EventGoogleUpdate = true
		b.MeetingZoomUpdate = b.EnableZoom
	}
	b.RecurrenceFreq = freq.Value
	b.RecurrenceFreqLabel = freq.Label

	//process the recurrence frequency
	recurrenceRule, err := CreateRecurrenceRule(freq, b.TimeFrom)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("create rule: %v", freq))
	}
	if recurrenceRule == "" {
		b.RecurrenceStart = nil
		b.RecurrenceRules = nil
		return nil
	}
	if b.RecurrenceStart == nil || resetStart {
		b.RecurrenceStart = &b.TimeFrom
	}
	b.RecurrenceRules = []string{recurrenceRule}
	return nil
}

//GetRecurrenceRules : get the recurrence rules as a string pointer
func (b *Booking) GetRecurrenceRules() *string {
	if len(b.RecurrenceRules) == 0 {
		return nil
	}
	s := strings.Join(b.RecurrenceRules, RecurrenceRuleSeparator)
	return &s
}

//GenerateRecurring : check if more recurring bookings should be created
func (b *Booking) GenerateRecurring(now time.Time) bool {
	if !b.IsRecurring() {
		return false
	}
	if b.RecurrenceInstanceEnd == nil {
		return true
	}
	return now.After(*b.RecurrenceInstanceEnd)
}

//SetProvider : set the provider
func (b *Booking) SetProvider(provider *Provider) {
	b.Provider = provider
}

//SetService : set the service
func (b *Booking) SetService(svc *Service) {
	b.Service = svc
	b.EnableZoom = svc.EnableZoom
	b.LocationType = svc.LocationType
	b.ServicePadding = svc.Padding
	b.ServicePriceType = svc.PriceType

	//check for the type changing
	if b.ServiceType != svc.Type {
		b.EventGoogleUpdate = true
		b.MeetingZoomUpdate = b.EnableZoom
	}
	b.ServiceType = svc.Type

	//check for the name changing
	if b.ServiceName != svc.Name {
		b.EventGoogleUpdate = true
		b.MeetingZoomUpdate = b.EnableZoom
	}
	b.ServiceName = svc.Name

	//check for the price changing
	if b.ServicePrice != svc.Price {
		b.EventGoogleUpdate = true
		b.MeetingZoomUpdate = b.EnableZoom
	}
	b.ServicePrice = svc.Price

	//check for the duration changing
	if b.ServiceDuration != svc.Duration {
		b.EventGoogleUpdate = true
		b.MeetingZoomUpdate = b.EnableZoom
	}
	b.ServiceDuration = svc.Duration
	b.ServiceDurationLabel = svc.FormatDuration()
}

//SetTimeFrom : set the from-time
func (b *Booking) SetTimeFrom(t time.Time) {
	if !b.TimeFrom.Equal(t) {
		b.TimeChange = true
		b.EventGoogleUpdate = true
		b.MeetingZoomUpdate = b.EnableZoom
	}
	b.TimeFrom = t
	b.TimeFromPadded = b.TimeFrom.Add(-time.Duration(b.ServicePadding) * time.Minute)

	//default the to-time
	if b.IsApptOnly() {
		b.SetTimeTo(b.TimeFrom.Add(time.Duration(b.ServiceDuration) * time.Minute))
	} else {
		b.SetTimeTo(b.TimeFrom)
	}
}

//SetTimeTo : set the to-time
func (b *Booking) SetTimeTo(t time.Time) {
	if !b.TimeTo.Equal(t) {
		b.TimeChange = true
		b.EventGoogleUpdate = true
		b.MeetingZoomUpdate = b.EnableZoom
	}
	b.TimeTo = t
	b.TimeToPadded = b.TimeTo.Add(time.Duration(b.ServicePadding) * time.Minute)
}

//SetLocation : set the location
func (b *Booking) SetLocation(location string) {
	if b.Location != location {
		b.EventGoogleUpdate = true
		b.MeetingZoomUpdate = b.EnableZoom
	}
	b.Location = location
}

//SetDescription : set the description
func (b *Booking) SetDescription(desc string) {
	if b.Description != desc {
		b.EventGoogleUpdate = true
		b.MeetingZoomUpdate = b.EnableZoom
	}
	b.Description = desc
}

//SetProviderNote : set the provider note
func (b *Booking) SetProviderNote(providerNote string) {
	if b.ProviderNote != providerNote {
		b.EventGoogleUpdate = true
		b.MeetingZoomUpdate = b.EnableZoom
	}
	b.ProviderNote = providerNote
}

//FormatCaptured : format the payment captured date
func (b *Booking) FormatCaptured(timeZone string) string {
	if b.Payment == nil {
		return ""
	}
	return b.Payment.FormatCaptured(timeZone)
}

//FormatInvoiced : format the payment invoice date
func (b *Booking) FormatInvoiced(timeZone string) string {
	if b.Payment == nil {
		return ""
	}
	return b.Payment.FormatInvoiced(timeZone)
}

//FormatCreated : format the created date
func (b *Booking) FormatCreated(timeZone string) string {
	return FormatDateTimeLocal(b.Created, timeZone)
}

//FormatDateTime : format the booking date and time
func (b *Booking) FormatDateTime(timeZone string) string {
	//return the appointment time
	if b.IsApptOnly() {
		return fmt.Sprintf("%s - %s", FormatDateTimeLocal(b.TimeFrom, timeZone), FormatTimeLocal(b.TimeTo, timeZone))
	}
	return FormatDateTimeLocal(b.TimeFrom, timeZone)
}

//FormatTime : format the booking time
func (b *Booking) FormatTime(timeZone string) string {
	//return the appointment time
	if b.IsApptOnly() {
		return fmt.Sprintf("%s - %s", FormatTimeLocal(b.TimeFrom, timeZone), FormatTimeLocal(b.TimeTo, timeZone))
	}
	return FormatTimeLocal(b.TimeFrom, timeZone)
}

//FormatTimeTo : format the to-time
func (b *Booking) FormatTimeTo(timeZone string) string {
	return FormatTimeLocal(b.TimeTo, timeZone)
}

//FormatTimeFrom : format the from-time
func (b *Booking) FormatTimeFrom(timeZone string) string {
	return FormatTimeLocal(b.TimeFrom, timeZone)
}

//FormatTimeFromDate : format the from-time date
func (b *Booking) FormatTimeFromDate(timeZone string) string {
	return FormatDateLocal(b.TimeFrom, timeZone)
}

//FormatRecurrenceFreq : format the recurrency frequency
func (b *Booking) FormatRecurrenceFreq() string {
	if b.RecurrenceFreq == RecurrenceIntervalOnce {
		return ""
	}
	return b.RecurrenceFreqLabel
}

//FormatServicePaymentDescription : format the service description for payment
func (b *Booking) FormatServicePaymentDescription(timeZone string) string {
	return fmt.Sprintf("%s on %s", b.ServiceName, b.FormatDateTime(timeZone))
}

//AllowUnPay : check if a payment can be marked as unpaid
func (b *Booking) AllowUnPay() bool {
	return b.Payment != nil && b.Payment.AllowUnPay()
}

//FormatServicePrice : format the price
func (b *Booking) FormatServicePrice() string {
	return b.ServicePriceType.Format(b.ServicePrice)
}

//ComputeServicePrice : compute the price of the service
func (b *Booking) ComputeServicePrice() float32 {
	return b.ServicePriceType.Compute(b.ServicePrice, b.ServiceDuration)
}

//SetCouponCode : set the coupon code
func (b *Booking) SetCouponCode(code string) {
	if b.CouponCode != code {
		b.CouponCodeChange = true
	}
	b.CouponCode = strings.ToUpper(code)
}

//CouponApplied : flag indicating if the coupon was applied
func (b *Booking) CouponApplied() bool {
	return b.Coupon != nil
}

//FormatCoupon : display the coupon
func (b *Booking) FormatCoupon() string {
	if b.Coupon != nil {
		return fmt.Sprintf("%s - %s", b.Coupon.Code, b.Coupon.FormatValue())
	}
	return ""
}

//create the statement to load a booking
func bookingQueryCreate(whereStmt string, orderStmt string, limit int) string {
	if orderStmt == "" {
		orderStmt = "b.time_start,b.updated"
	}
	stmt := fmt.Sprintf("SELECT BIN_TO_UUID(p.id),p.url_name,p.url_name_friendly,p.calendar_google_id,p.calendar_google_update,p.calendar_google_data,p.data,BIN_TO_UUID(u.id),u.email,u.token_zoom_data,u.data,s.type,BIN_TO_UUID(s.id),s.data,BIN_TO_UUID(c.id),c.email,c.disable_emails,c.data,BIN_TO_UUID(b.id),BIN_TO_UUID(b.parent_id),b.service_type,b.time_start,b.time_end,b.time_start_padded,b.time_end_padded,b.confirmed,b.client_created,b.recurrence_start,b.recurrence_rules,b.recurrence_instance_end,b.event_google_id,b.event_google_update,b.event_google_delete,b.meeting_zoom_id,b.meeting_zoom_update,b.meeting_zoom_delete,b.meeting_zoom_data,b.deleted,b.created,b.data,BIN_TO_UUID(pmt.id),pmt.friendly_id,pmt.type,pmt.amount,pmt.invoiced,pmt.paid,pmt.captured,pmt.stripe_id,pmt.paypal_id,pmt.data,BIN_TO_UUID(pu.id),pu.login,pu.data,BIN_TO_UUID(puu.id),puu.email,puu.token_zoom_data,puu.data FROM %s b INNER JOIN %s s ON s.id=b.service_id INNER JOIN %s p ON p.id=s.provider_id INNER JOIN %s c ON c.id=b.client_id INNER JOIN %s u ON u.id=p.user_id LEFT JOIN %s pmt ON pmt.secondary_id=b.id AND pmt.deleted=0 LEFT JOIN %s pu ON pu.id=b.provider_user_id AND pu.deleted=0 LEFT JOIN %s puu ON puu.id=pu.user_id AND puu.deleted=0 WHERE %s ORDER BY %s", dbTableBooking, dbTableService, dbTableProvider, dbTableClient, dbTableUser, dbTablePayment, dbTableProviderUser, dbTableUser, whereStmt, orderStmt)
	if limit > 0 {
		stmt = fmt.Sprintf("%s LIMIT %d", stmt, limit)
	}
	return stmt
}

//create the statement to count bookings
func bookingCountCreate(whereStmt string) string {
	stmt := fmt.Sprintf("SELECT COUNT(*) FROM %s b INNER JOIN %s s ON s.id=b.service_id INNER JOIN %s p ON p.id=s.provider_id INNER JOIN %s c ON c.id=b.client_id INNER JOIN %s u ON u.id=p.user_id LEFT JOIN %s pmt ON pmt.secondary_id=b.id AND pmt.deleted=0 LEFT JOIN %s pu ON pu.id=b.provider_user_id AND pu.deleted=0 LEFT JOIN %s puu ON puu.id=pu.user_id AND puu.deleted=0 WHERE %s", dbTableBooking, dbTableService, dbTableProvider, dbTableClient, dbTableUser, dbTablePayment, dbTableProviderUser, dbTableUser, whereStmt)
	return stmt
}

//parse a booking
func bookingQueryParse(rowFn ScanFn) (*Booking, error) {
	//read the row
	var providerIDStr string
	var providerURLName string
	var providerURLNameFriendly string
	var providerCalenderGoogleID sql.NullString
	var providerCalendarGoogleUpdateBit string
	var providerCalendarGoogleDataStr sql.NullString
	var providerDataStr string
	var userIDStr string
	var userEmail string
	var userTokenZoomDataStr sql.NullString
	var userDataStr string
	var svcType int
	var svcIDStr string
	var svcDataStr string
	var clientIDStr string
	var clientEmail string
	var clientDisableEmailsBit string
	var clientDataStr string
	var idStr string
	var parentIDStr sql.NullString
	var bookingSvcType int
	var timeFrom time.Time
	var timeTo time.Time
	var timeFromPadded time.Time
	var timeToPadded time.Time
	var confirmedBit string
	var clientCreatedBit string
	var recurrenceStart sql.NullTime
	var recurrenceRules sql.NullString
	var recurrenceInstanceEnd sql.NullTime
	var eventGoogleID sql.NullString
	var eventGoogleUpdateBit string
	var eventGoogleDeleteBit string
	var meetingZoomID sql.NullString
	var meetingZoomUpdateBit string
	var meetingZoomDeleteBit string
	var meetingZoomData sql.NullString
	var deletedBit string
	var created time.Time
	var dataStr string
	var paymentIDStr sql.NullString
	var friendlyID sql.NullString
	var paymentType sql.NullInt32
	var amount sql.NullInt32
	var invoiced sql.NullTime
	var paid sql.NullTime
	var captured sql.NullTime
	var paymentStripeID sql.NullString
	var paymentPayPalID sql.NullString
	var paymentData sql.NullString
	var providerUserIDStr sql.NullString
	var providerUserLogin sql.NullString
	var providerUserData sql.NullString
	var providerUserUserIDStr sql.NullString
	var providerUserUserEmail sql.NullString
	var providerUserUserTokenZoomDataStr sql.NullString
	var providerUserUserData sql.NullString
	err := rowFn(
		//provider
		&providerIDStr,
		&providerURLName,
		&providerURLNameFriendly,
		&providerCalenderGoogleID,
		&providerCalendarGoogleUpdateBit,
		&providerCalendarGoogleDataStr,
		&providerDataStr,

		//user
		&userIDStr,
		&userEmail,
		&userTokenZoomDataStr,
		&userDataStr,

		//service
		&svcType,
		&svcIDStr,
		&svcDataStr,

		//client
		&clientIDStr,
		&clientEmail,
		&clientDisableEmailsBit,
		&clientDataStr,

		//booking
		&idStr,
		&parentIDStr,
		&bookingSvcType,
		&timeFrom,
		&timeTo,
		&timeFromPadded,
		&timeToPadded,
		&confirmedBit,
		&clientCreatedBit,
		&recurrenceStart,
		&recurrenceRules,
		&recurrenceInstanceEnd,
		&eventGoogleID,
		&eventGoogleUpdateBit,
		&eventGoogleDeleteBit,
		&meetingZoomID,
		&meetingZoomUpdateBit,
		&meetingZoomDeleteBit,
		&meetingZoomData,
		&deletedBit,
		&created,
		&dataStr,

		//payment
		&paymentIDStr,
		&friendlyID,
		&paymentType,
		&amount,
		&invoiced,
		&paid,
		&captured,
		&paymentStripeID,
		&paymentPayPalID,
		&paymentData,

		//provider user
		&providerUserIDStr,
		&providerUserLogin,
		&providerUserData,
		&providerUserUserIDStr,
		&providerUserUserEmail,
		&providerUserUserTokenZoomDataStr,
		&providerUserUserData,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, errors.Wrap(err, "select booking")
	}

	//parse the uuid
	providerID, err := uuid.FromString(providerIDStr)
	if err != nil {
		return nil, errors.Wrap(err, "parse uuid provider")
	}
	userID, err := uuid.FromString(userIDStr)
	if err != nil {
		return nil, errors.Wrap(err, "parse uuid user")
	}
	svcID, err := uuid.FromString(svcIDStr)
	if err != nil {
		return nil, errors.Wrap(err, "parse uuid service")
	}
	clientID, err := uuid.FromString(clientIDStr)
	if err != nil {
		return nil, errors.Wrap(err, "parse uuid client")
	}
	id, err := uuid.FromString(idStr)
	if err != nil {
		return nil, errors.Wrap(err, "parse uuid booking")
	}
	var parentID *uuid.UUID
	if parentIDStr.Valid {
		parentUUID, err := uuid.FromString(parentIDStr.String)
		if err != nil {
			return nil, errors.Wrap(err, "parse uuid booking parent")
		}
		parentID = &parentUUID
	}

	//unmarshal the user
	var user User
	err = json.Unmarshal([]byte(userDataStr), &user)
	if err != nil {
		return nil, errors.Wrap(err, "unjson user")
	}
	user.ID = &userID
	user.Email = userEmail

	//unmarshal the zoom token data
	if userTokenZoomDataStr.Valid {
		var token TokenZoom
		err = json.Unmarshal([]byte(userTokenZoomDataStr.String), &token)
		if err != nil {
			return nil, errors.Wrap(err, "unjson zoom")
		}
		user.ZoomToken = &token
	}

	//unmarshal the provider
	var provider Provider
	err = json.Unmarshal([]byte(providerDataStr), &provider)
	if err != nil {
		return nil, errors.Wrap(err, "unjson provider")
	}
	provider.User = &user
	provider.ID = &providerID
	provider.URLName = providerURLName
	provider.URLNameFriendly = providerURLNameFriendly
	if providerCalenderGoogleID.Valid {
		provider.GoogleCalendarID = &providerCalenderGoogleID.String
	}
	provider.GoogleCalendarUpdate = providerCalendarGoogleUpdateBit == "\x01"

	//unmarshal the calendar google data
	if providerCalendarGoogleDataStr.Valid {
		var calendarGoogle CalendarGoogle
		err = json.Unmarshal([]byte(providerCalendarGoogleDataStr.String), &calendarGoogle)
		if err != nil {
			return nil, errors.Wrap(err, "unjson calendar")
		}
		provider.GoogleCalendarData = &calendarGoogle
	}

	//unmarshal the service
	var svc Service
	err = json.Unmarshal([]byte(svcDataStr), &svc)
	if err != nil {
		return nil, errors.Wrap(err, "unjson service")
	}
	svc.Type = ServiceType(svcType)
	svc.ID = &svcID

	//unmarshal the client
	var client Client
	err = json.Unmarshal([]byte(clientDataStr), &client)
	if err != nil {
		return nil, errors.Wrap(err, "unjson client")
	}
	client.ID = &clientID
	client.ProviderID = &providerID
	client.Email = clientEmail
	client.DisableEmails = clientDisableEmailsBit == "\x01"

	//unmarshal the booking
	var book Booking
	err = json.Unmarshal([]byte(dataStr), &book)
	if err != nil {
		return nil, errors.Wrap(err, "unjson booking")
	}
	book.Client = &client
	book.Provider = &provider
	book.Service = &svc
	book.ServiceType = ServiceType(bookingSvcType)
	book.ID = &id
	book.ParentID = parentID
	book.TimeFrom = timeFrom
	book.TimeTo = timeTo
	book.TimeFromPadded = timeFromPadded
	book.TimeToPadded = timeToPadded
	book.Confirmed = confirmedBit == "\x01"
	book.ClientCreated = clientCreatedBit == "\x01"
	if recurrenceStart.Valid {
		book.RecurrenceStart = &recurrenceStart.Time
	}
	if recurrenceRules.Valid {
		book.RecurrenceRules = strings.Split(recurrenceRules.String, RecurrenceRuleSeparator)
	}
	if recurrenceInstanceEnd.Valid {
		book.RecurrenceInstanceEnd = &recurrenceInstanceEnd.Time
	}
	if eventGoogleID.Valid {
		book.EventGoogleID = &eventGoogleID.String
	}
	book.EventGoogleUpdate = eventGoogleUpdateBit == "\x01"
	book.EventGoogleDelete = eventGoogleDeleteBit == "\x01"
	if meetingZoomID.Valid {
		book.MeetingZoomID = &meetingZoomID.String
	}
	book.MeetingZoomUpdate = meetingZoomUpdateBit == "\x01"
	book.MeetingZoomDelete = meetingZoomDeleteBit == "\x01"
	if meetingZoomData.Valid {
		var meetingZoom MeetingZoom
		err = json.Unmarshal([]byte(meetingZoomData.String), &meetingZoom)
		if err != nil {
			return nil, errors.Wrap(err, "unjson zoom meeting")
		}
		book.MeetingZoomData = &meetingZoom
	}
	book.Deleted = deletedBit == "\x01"
	book.Created = created

	//unmarshal the payment
	var payment Payment
	if paymentData.Valid {
		err = json.Unmarshal([]byte(paymentData.String), &payment)
		if err != nil {
			return nil, errors.Wrap(err, "unjson payment")
		}
		if paymentIDStr.Valid {
			paymentUUID, err := uuid.FromString(paymentIDStr.String)
			if err != nil {
				return nil, errors.Wrap(err, "parse uuid payment")
			}
			payment.ID = &paymentUUID
		}
		if friendlyID.Valid {
			payment.FriendlyID = friendlyID.String
		}
		if paymentType.Valid {
			payment.Type = PaymentType(paymentType.Int32)
		}
		if amount.Valid {
			payment.Amount = int(amount.Int32)
		}
		if invoiced.Valid {
			payment.Invoiced = &invoiced.Time
		}
		if paid.Valid {
			payment.Paid = &paid.Time
		}
		if captured.Valid {
			payment.Captured = &captured.Time
		}
		if paymentStripeID.Valid {
			payment.StripeID = &paymentStripeID.String
		}
		if paymentPayPalID.Valid {
			payment.PayPalID = &paymentPayPalID.String
		}
		book.Payment = &payment
	}

	//unmarshal the provider user
	var providerUser ProviderUser
	if providerUserData.Valid {
		err = json.Unmarshal([]byte(providerUserData.String), &providerUser)
		if err != nil {
			return nil, errors.Wrap(err, "unjson provider user")
		}
		if providerUserIDStr.Valid {
			providerUserID, err := uuid.FromString(providerUserIDStr.String)
			if err != nil {
				return nil, errors.Wrap(err, "parse uuid provider user")
			}
			providerUser.ID = &providerUserID
		}
		if providerUserLogin.Valid {
			providerUser.Login = providerUserLogin.String
		}
		book.ProviderUserID = providerUser.ID
		book.ProviderUser = &providerUser

		//unmarshal the associated user
		var providerUserUser User
		if providerUserUserData.Valid {
			err = json.Unmarshal([]byte(providerUserUserData.String), &providerUserUser)
			if err != nil {
				return nil, errors.Wrap(err, "unjson provider user user")
			}
			if providerUserUserIDStr.Valid {
				providerUserUserID, err := uuid.FromString(providerUserUserIDStr.String)
				if err != nil {
					return nil, errors.Wrap(err, "parse uuid provider user user")
				}
				providerUserUser.ID = &providerUserUserID
			}
			if providerUserUserEmail.Valid {
				providerUserUser.Email = providerUserUserEmail.String
			}
			if providerUserUserTokenZoomDataStr.Valid {
				var token TokenZoom
				err = json.Unmarshal([]byte(providerUserUserTokenZoomDataStr.String), &token)
				if err != nil {
					return nil, errors.Wrap(err, "unjson provider user zoom token")
				}
				providerUserUser.ZoomToken = &token
			}
			providerUser.User = &providerUserUser
		}
	}
	return &book, nil
}

//load a booking
func loadBooking(ctx context.Context, db *DB, updateViewed bool, whereStmt string, args ...interface{}) (context.Context, *Booking, error) {
	//create the final query
	stmt := bookingQueryCreate(whereStmt, "", 0)
	ctx, row, err := db.QueryRow(ctx, stmt, args...)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "query row booking")
	}

	//read the booking
	book, err := bookingQueryParse(row.Scan)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "bookng parse")
	}

	//mark the booking as read
	if updateViewed {
		ctx, err = MarkBookingViewed(ctx, db, book.ID)
		if err != nil {
			return ctx, nil, errors.Wrap(err, "mark booking read")
		}
	}
	return ctx, book, nil
}

//LoadBookingByID : load a booking by id
func LoadBookingByID(ctx context.Context, db *DB, id *uuid.UUID, updateViewed bool, includeDeleted bool) (context.Context, *Booking, error) {
	var whereStmt string
	if includeDeleted {
		whereStmt = fmt.Sprintf("b.id=UUID_TO_BIN(?)")
	} else {
		whereStmt = fmt.Sprintf("(b.deleted=0 OR b.event_google_delete=1 OR b.meeting_zoom_delete=1) AND b.id=UUID_TO_BIN(?)")
	}
	ctx, booking, err := loadBooking(ctx, db, updateViewed, whereStmt, id)
	if err != nil {
		return ctx, nil, errors.Wrap(err, fmt.Sprintf("no booking: %s", id))
	}
	if booking == nil {
		return ctx, nil, fmt.Errorf("no booking: %s", id)
	}
	return ctx, booking, err
}

//save a booking
func saveBooking(ctx context.Context, db *DB, book *Booking, confirmed bool, isClient bool, deleted bool) (context.Context, error) {
	//create the booking id if necessary
	if book.ID == nil {
		id, err := uuid.NewV4()
		if err != nil {
			return ctx, errors.Wrap(err, "new uuid booking")
		}
		book.ID = &id
	}

	//propagate the deleted flag
	book.EventGoogleDelete = deleted
	book.MeetingZoomDelete = deleted

	//json encode the meeting data
	var err error
	var meetingJSON []byte
	if book.MeetingZoomData != nil {
		meetingJSON, err = json.Marshal(book.MeetingZoomData)
		if err != nil {
			return ctx, errors.Wrap(err, "json meeting")
		}
	}

	//json encode the booking data
	dataJSON, err := json.Marshal(book)
	if err != nil {
		return ctx, errors.Wrap(err, "json booking")
	}

	//insert the booking
	stmt := fmt.Sprintf("INSERT INTO %s(id,parent_id,provider_id,provider_user_id,service_type,service_id,client_id,time_start,time_end,time_start_padded,time_end_padded,confirmed,client_created,recurrence_start,recurrence_rules,recurrence_instance_end,event_google_delete,event_google_update,meeting_zoom_id,meeting_zoom_update,meeting_zoom_delete,meeting_zoom_data,deleted,data) VALUES (UUID_TO_BIN(?),UUID_TO_BIN(?),UUID_TO_BIN(?),UUID_TO_BIN(?),?,UUID_TO_BIN(?),UUID_TO_BIN(?),?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?) ON DUPLICATE KEY UPDATE parent_id=VALUES(parent_id),provider_id=VALUES(provider_id),provider_user_id=VALUES(provider_user_id),service_type=VALUES(service_type),service_id=VALUES(service_id),client_id=VALUES(client_id),time_start=VALUES(time_start),time_end=VALUES(time_end),time_start_padded=VALUES(time_start_padded),time_end_padded=VALUES(time_end_padded),confirmed=VALUES(confirmed),client_created=VALUES(client_created),recurrence_start=VALUES(recurrence_start),recurrence_rules=VALUES(recurrence_rules),recurrence_instance_end=VALUES(recurrence_instance_end),event_google_delete=VALUES(event_google_delete),event_google_update=VALUES(event_google_update),meeting_zoom_id=VALUES(meeting_zoom_id),meeting_zoom_update=VALUES(meeting_zoom_update),meeting_zoom_delete=VALUES(meeting_zoom_delete),meeting_zoom_data=VALUES(meeting_zoom_data),deleted=VALUES(deleted),data=VALUES(data)", dbTableBooking)
	ctx, result, err := db.Exec(ctx, stmt, book.ID, book.ParentID, book.Provider.ID, book.ProviderUserID, book.ServiceType, book.Service.ID, book.Client.ID, book.TimeFrom.UTC(), book.TimeTo.UTC(), book.TimeFromPadded.UTC(), book.TimeToPadded.UTC(), confirmed, isClient, book.RecurrenceStart, book.GetRecurrenceRules(), book.RecurrenceInstanceEnd, book.EventGoogleDelete, book.EventGoogleUpdate, book.MeetingZoomID, book.MeetingZoomUpdate, book.MeetingZoomDelete, meetingJSON, deleted, dataJSON)
	if err != nil {
		return ctx, errors.Wrap(err, "insert booking")
	}
	count, err := result.RowsAffected()
	if err != nil {
		return ctx, errors.Wrap(err, "insert booking rows affected")
	}

	//0 indicated no update, 1 an insert, 2 an update
	if count < 0 || count > 2 {
		return ctx, fmt.Errorf("unable to insert booking: %s: %s", book.Service.ID, book.Client.Email)
	}
	return ctx, nil
}

//SaveBooking : save a booking
func SaveBooking(ctx context.Context, db *DB, provider *Provider, svc *Service, book *Booking, now time.Time, changeAllFollowing bool, confirmed bool, isClient bool, deleted bool) (context.Context, error) {
	var err error

	//create the booking id if necessary
	create := false
	if book.ID == nil {
		id, err := uuid.NewV4()
		if err != nil {
			return ctx, errors.Wrap(err, "new uuid booking")
		}
		book.ID = &id
		if book.IsRecurring() {
			book.ParentID = book.ID
		}
		create = true
	}
	ctx, err = db.ProcessTx(ctx, "save booking", func(ctx context.Context, tx *DB) (context.Context, error) {
		//use the specified client id if set, otherwise probe for a client
		isNewClient := false
		if book.Client.ID == nil {
			//probe for a client by email
			ctx, client, err := LoadClientByProviderIDAndEmail(ctx, db, svc.Provider.ID, book.Client.Email)
			if err != nil {
				return ctx, errors.Wrap(err, "booking load client email")
			}

			//create or update the client
			if client == nil {
				client = &Client{
					ProviderID: svc.Provider.ID,
					Email:      book.Client.Email,
					TimeZone:   book.Client.TimeZone,
				}
				isNewClient = true
			}
			client.Name = book.Client.Name
			client.Phone = book.Client.Phone
			if book.LocationType.IsLocationClient() {
				client.Location = book.Client.Location
			}

			//associate the user if available
			ctx, user, err := LoadUserByLogin(ctx, db, book.Client.Email)
			if err != nil {
				return ctx, errors.Wrap(err, "booking load user email")
			}
			if user != nil {
				client.UserID = user.ID
			}

			//save
			ctx, err = SaveClient(ctx, db, client)
			if err != nil {
				return ctx, errors.Wrap(err, "booking save client")
			}
			book.Client = client
		} else {
			//load the client
			ctx, client, err := LoadClientByID(ctx, db, book.Client.ID)
			if err != nil {
				return ctx, errors.Wrap(err, "booking load client")
			}
			if client == nil {
				return ctx, errors.Wrap(err, fmt.Sprintf("no booking load client: %s", book.Client.ID))
			}
			if book.LocationType.IsLocationClient() && client.Location != book.Client.Location {
				client.Location = book.Client.Location
				ctx, err := SaveClient(ctx, db, client)
				if err != nil {
					return ctx, errors.Wrap(err, "booking update client location")
				}
			}
			book.Client = client
		}

		//apply the coupon
		if book.Coupon != nil {
			book.ServicePriceOriginal = svc.Price
			book.ServicePrice = book.Coupon.AdjustPrice(svc.Price, book.Service.ID, isNewClient, now)
		}

		//load the parent
		saveParentBook := false
		parentBook := book
		if book.ParentID != nil && book.ID.String() != book.ParentID.String() {
			ctx, parentBook, err = LoadBookingByID(ctx, db, book.ParentID, false, true)
			if err != nil {
				return ctx, errors.Wrap(err, fmt.Sprintf("load booking: %s", book.ParentID))
			}
		}

		//check for a time change, in which case regenerate the recurring bookings
		if !create && (book.TimeChange || deleted) && changeAllFollowing {
			DeleteBookingsByParentID(ctx, db, book)

			//reset the instance end date to force regeneration
			parentBook.RecurrenceInstanceEnd = nil
			saveParentBook = true
		}

		//save recurring bookings, checking if new ones should be generated
		if create && !deleted && book.GenerateRecurring(now) {
			ctx, ruleEnd, err := SaveBookingsRecurring(ctx, db, book, book.TimeFrom, book.Confirmed, isClient)
			if err != nil {
				return ctx, errors.Wrap(err, "save bookings recurring")
			}
			parentBook.RecurrenceInstanceEnd = &ruleEnd
			saveParentBook = true
		} else if changeAllFollowing {
			UpdateBookingsByParent(ctx, db, book)

			//create new recurring bookings if the time changes
			if book.TimeChange {
				ctx, ruleEnd, err := SaveBookingsRecurring(ctx, db, book, book.TimeFrom, book.Confirmed, isClient)
				if err != nil {
					return ctx, errors.Wrap(err, "save bookings recurring")
				}
				book.RecurrenceInstanceEnd = &ruleEnd
				parentBook.RecurrenceInstanceEnd = nil
				saveParentBook = true
			}
		}

		//save the original booking
		ctx, err := saveBooking(ctx, db, book, confirmed, isClient, deleted)
		if err != nil {
			return ctx, errors.Wrap(err, "save booking")
		}

		//save the parent booking if necessary
		if saveParentBook && book.ID.String() != parentBook.ID.String() {
			ctx, err := saveBooking(ctx, db, parentBook, parentBook.Confirmed, isClient, parentBook.Deleted)
			if err != nil {
				return ctx, errors.Wrap(err, "save booking parent")
			}
		}
		return ctx, nil
	})
	if err != nil {
		return ctx, errors.Wrap(err, "save booking")
	}
	return ctx, nil
}

//SaveBookingsRecurring : generate and save recurring bookings
func SaveBookingsRecurring(ctx context.Context, db *DB, book *Booking, ruleStart time.Time, confirmed bool, isClient bool) (context.Context, time.Time, error) {
	ctx, logger := GetLogger(ctx)
	if !book.IsRecurring() {
		return ctx, time.Time{}, nil
	}

	//prepare to generate the recurring series of bookings
	rules := book.RecurrenceRules
	ruleDurationMin := time.Duration(book.ServiceDuration) * time.Minute
	ruleEnd := ruleStart.AddDate(0, recurringGenerateMonths, 0)
	ruleBook := *book
	ruleBook.RecurrenceRules = nil
	ruleBook.RecurrenceStart = nil
	ruleBook.RecurrenceInstanceEnd = nil

	//track the last instance end date
	bookEnd := ruleEnd

	//pad the times based on the service padding
	padding := time.Duration(book.ServicePadding) * time.Minute

	//process the recurrence rules across a time window and find all times that match the recurrence rules
	count := 0
	for _, rule := range rules {
		parsedRule, err := ParseRecurrenceRule(rule)
		if err != nil {
			return ctx, time.Time{}, errors.Wrap(err, fmt.Sprintf("parse recurrence rule: %s", rule))
		}
		parsedRule.DtStart = ruleStart
		logger.Debugw("recurrence rule", "rule", parsedRule, "start", ruleStart, "end", ruleEnd)

		//find the event times based on the rule
		var ruleTime time.Time
		ruleIterator := parsedRule.Iterator().Between(ruleStart, ruleEnd)
		for ruleIterator.Step(&ruleTime) {
			count++

			//save a booking for the new times, ignoring the current booking times
			if ruleTime.Equal(book.TimeFrom) {
				continue
			}
			ruleBook.ID = nil
			ruleBook.TimeFrom = ruleTime
			ruleBook.TimeFromPadded = ruleBook.TimeFrom.Add(-padding)
			ruleBook.TimeTo = ruleTime.Add(ruleDurationMin)
			ruleBook.TimeToPadded = ruleBook.TimeTo.Add(padding)
			logger.Debugw("recurring instance", "from", ruleBook.TimeFrom, "to", ruleBook.TimeTo)
			ctx, err = saveBooking(ctx, db, &ruleBook, confirmed, isClient, false)
			if err != nil {
				return ctx, time.Time{}, errors.Wrap(err, fmt.Sprintf("save recurring booking: %s: %s-%s", book.ID, ruleBook.TimeFrom, ruleBook.TimeTo))
			}
			bookEnd = ruleBook.TimeTo
		}
	}

	//if the count is zero, then there are no more recurring events, so use the maximum date to avoid further processing
	if count == 0 {
		ruleEnd = MaxTime
	}
	return ctx, bookEnd, nil
}

//UpdateBookingsByParent : update child booking by the parent
func UpdateBookingsByParent(ctx context.Context, db *DB, book *Booking) (context.Context, error) {
	//json encode the booking data
	dataJSON, err := json.Marshal(book)
	if err != nil {
		return ctx, errors.Wrap(err, "json booking")
	}

	//update
	stmt := fmt.Sprintf("UPDATE %s SET service_type=?,service_id=UUID_TO_BIN(?),client_id=UUID_TO_BIN(?),data=? WHERE deleted=0 AND time_start>? AND parent_id=UUID_TO_BIN(?)", dbTableBooking)
	ctx, _, err = db.Exec(ctx, stmt, book.ServiceType, book.Service.ID, book.Client.ID, dataJSON, book.TimeFrom, book.ParentID)
	if err != nil {
		return ctx, errors.Wrap(err, "update booking parent")
	}
	return ctx, nil
}

//DeleteBooking : delete a booking
func DeleteBooking(ctx context.Context, db *DB, id *uuid.UUID) (context.Context, error) {
	stmt := fmt.Sprintf("UPDATE %s SET deleted=1,event_google_delete=1,meeting_zoom_delete=1 WHERE deleted=0 AND id=UUID_TO_BIN(?)", dbTableBooking)
	ctx, _, err := db.Exec(ctx, stmt, id)
	if err != nil {
		return ctx, errors.Wrap(err, "delete booking")
	}
	return ctx, nil
}

//DeleteBookingsByParentID : delete a booking by the parent id
func DeleteBookingsByParentID(ctx context.Context, db *DB, book *Booking) (context.Context, error) {
	stmt := fmt.Sprintf("UPDATE %s SET deleted=1,event_google_delete=1,meeting_zoom_delete=1 WHERE deleted=0 AND time_start>? AND parent_id=UUID_TO_BIN(?)", dbTableBooking)
	ctx, _, err := db.Exec(ctx, stmt, book.TimeFrom, book.ParentID)
	if err != nil {
		return ctx, errors.Wrap(err, "delete booking by parent")
	}
	return ctx, nil
}

//list bookings by the specified where clause and ordering
func listBookings(ctx context.Context, db *DB, whereStmt string, orderStmt string, args ...interface{}) (context.Context, []*Booking, error) {
	ctx, logger := GetLogger(ctx)

	//load the bookings
	stmt := bookingQueryCreate(whereStmt, orderStmt, 0)
	ctx, rows, err := db.Query(ctx, stmt, args...)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "select bookings")
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			logger.Warnw("rows close select bookings", "error", err)
		}
	}()

	//read the bookings
	books := make([]*Booking, 0, 2)
	for rows.Next() {
		book, err := bookingQueryParse(rows.Scan)
		if err != nil {
			return ctx, nil, errors.Wrap(err, "bookng parse")
		}
		books = append(books, book)
	}
	return ctx, books, nil
}

//count bookings by the specified where clause
func countBookings(ctx context.Context, db *DB, whereStmt string, args ...interface{}) (context.Context, int, error) {
	//count the bookings
	stmt := bookingCountCreate(whereStmt)
	ctx, row, err := db.QueryRow(ctx, stmt, args...)
	if err != nil {
		return ctx, 0, errors.Wrap(err, "query row bookings count client")
	}

	//read the row
	var count int
	err = row.Scan(&count)
	if err != nil {
		if err == sql.ErrNoRows {
			return ctx, 0, nil
		}
		return ctx, 0, errors.Wrap(err, "select bookings count client")
	}
	return ctx, count, nil
}

//ListBookingsByClientID : load the bookings for a client
func ListBookingsByClientID(ctx context.Context, db *DB, clientID *uuid.UUID) (context.Context, []*Booking, error) {
	stmt := "b.deleted=0 AND b.client_id=UUID_TO_BIN(?)"
	return listBookings(ctx, db, stmt, "", clientID)
}

//ListBookingsByProviderID : load the bookings for a provider
func ListBookingsByProviderID(ctx context.Context, db *DB, providerID *uuid.UUID) (context.Context, []*Booking, error) {
	whereStmt := "b.deleted=0 AND c.provider_id=UUID_TO_BIN(?)"
	return listBookings(ctx, db, whereStmt, "", providerID)
}

//ListBookingsByProviderIDAndTime : load the bookings for a provider over a time span
func ListBookingsByProviderIDAndTime(ctx context.Context, db *DB, providerID *uuid.UUID, user *User, fromTime time.Time, toTime time.Time) (context.Context, []*Booking, error) {
	whereStmt := "b.deleted=0 AND c.provider_id=UUID_TO_BIN(?) AND ((b.time_start_padded>=? AND b.time_start_padded<=?) OR (b.time_end_padded>=? AND b.time_end_padded<=?))"
	args := []interface{}{providerID, fromTime.UTC(), toTime.UTC(), fromTime.UTC(), toTime.UTC()}

	//match the user if set
	if user != nil {
		whereStmt = fmt.Sprintf("%s AND puu.id=UUID_TO_BIN(?)", whereStmt)
		args = append(args, user.ID)
	}
	return listBookings(ctx, db, whereStmt, "", args...)
}

//ListBookingsByProviderIDAndTypeAndTime : load the bookings for a provider or a specified type over a time span
func ListBookingsByProviderIDAndTypeAndTime(ctx context.Context, db *DB, providerID *uuid.UUID, user *User, serviceType ServiceType, fromTime time.Time, toTime time.Time) (context.Context, []*Booking, error) {
	whereStmt := "b.deleted=0 AND c.provider_id=UUID_TO_BIN(?) AND service_type=? AND ((b.time_start_padded>=? AND b.time_start_padded<=?) OR (b.time_end_padded>=? AND b.time_end_padded<=?))"
	args := []interface{}{providerID, serviceType, fromTime.UTC(), toTime.UTC(), fromTime.UTC(), toTime.UTC()}

	//match the user if set
	if user != nil {
		whereStmt = fmt.Sprintf("%s AND puu.id=UUID_TO_BIN(?)", whereStmt)
		args = append(args, user.ID)
	}
	return listBookings(ctx, db, whereStmt, "", args...)
}

//ListBookingsByProviderIDAndFilter : load the bookings for a provider based on the filter
func ListBookingsByProviderIDAndFilter(ctx context.Context, db *DB, providerID *uuid.UUID, user *User, filter BookingFilter, filterSub BookingFilter, now time.Time) (context.Context, []*Booking, error) {
	whereStmt := "b.deleted=0 AND c.provider_id=UUID_TO_BIN(?)"
	orderStmt := ""
	args := []interface{}{providerID, now}
	switch filter {
	case BookingFilterUnPaid:
		whereStmt = fmt.Sprintf("%s AND pmt.captured IS NULL", whereStmt)
		return listBookings(ctx, db, whereStmt, orderStmt, providerID)
	case BookingFilterUpcoming:
		whereStmt = fmt.Sprintf("%s AND b.time_start >= ?", whereStmt)
		switch filterSub {
		case BookingFilterNew:
			whereStmt = fmt.Sprintf("%s AND (b.confirmed=0 OR b.viewed=0) AND (b.parent_id IS NULL OR b.recurrence_rules IS NOT NULL)", whereStmt)
		case BookingFilterAll:
		default:
			return ctx, nil, fmt.Errorf("invalid sub-filter: %s", filterSub)
		}
	case BookingFilterPast:
		whereStmt = fmt.Sprintf("%s AND b.time_start < ?", whereStmt)
		orderStmt = "b.time_start DESC,b.updated DESC"
		switch filterSub {
		case BookingFilterInvoiced:
			whereStmt = fmt.Sprintf("%s AND pmt.invoiced IS NOT NULL", whereStmt)
		case BookingFilterPaid:
			whereStmt = fmt.Sprintf("%s AND pmt.captured IS NOT NULL", whereStmt)
		case BookingFilterAll:
		default:
			return ctx, nil, fmt.Errorf("invalid sub-filter: %s", filterSub)
		}
	default:
		return ctx, nil, fmt.Errorf("invalid filter: %s", filter)
	}

	//match the user if set
	if user != nil {
		whereStmt = fmt.Sprintf("%s AND puu.id=UUID_TO_BIN(?)", whereStmt)
		args = append(args, user.ID)
	}
	return listBookings(ctx, db, whereStmt, orderStmt, args...)
}

//CountBookingsByProviderIDAndFilter : count the bookings for a provider based on the filter
func CountBookingsByProviderIDAndFilter(ctx context.Context, db *DB, providerID *uuid.UUID, user *User, filter BookingFilter, filterSub BookingFilter, now time.Time) (context.Context, int, error) {
	whereStmt := "b.deleted=0 AND c.provider_id=UUID_TO_BIN(?)"
	args := []interface{}{providerID}
	switch filter {
	case BookingFilterNew:
		whereStmt = fmt.Sprintf("%s AND b.time_start>=? AND (b.confirmed=0 OR b.viewed=0) AND (b.parent_id IS NULL OR b.recurrence_rules IS NOT NULL)", whereStmt)
		args = append(args, now)
	case BookingFilterUnPaid:
		whereStmt = fmt.Sprintf("%s AND pmt.captured IS NULL", whereStmt)
	case BookingFilterUpcoming:
		whereStmt = fmt.Sprintf("%s AND b.time_start>=?", whereStmt)
		switch filterSub {
		case BookingFilterNew:
			whereStmt = fmt.Sprintf("%s AND (b.confirmed=0 OR b.viewed=0) AND (b.parent_id IS NULL OR b.recurrence_rules IS NOT NULL)", whereStmt)
		case BookingFilterAll:
		default:
			return ctx, 0, fmt.Errorf("invalid sub-filter: %s", filterSub)
		}
		args = append(args, now)
	case BookingFilterPast:
		whereStmt = fmt.Sprintf("%s AND b.time_start<?", whereStmt)
		switch filterSub {
		case BookingFilterInvoiced:
			whereStmt = fmt.Sprintf("%s AND pmt.invoiced IS NOT NULL", whereStmt)
		case BookingFilterPaid:
			whereStmt = fmt.Sprintf("%s AND pmt.captured IS NOT NULL", whereStmt)
		case BookingFilterAll:
		default:
			return ctx, 0, fmt.Errorf("invalid sub-filter: %s", filterSub)
		}
		args = append(args, now)
	default:
		return ctx, 0, fmt.Errorf("invalid filter: %s", filter)
	}

	//match the user if set
	if user != nil {
		whereStmt = fmt.Sprintf("%s AND puu.id=UUID_TO_BIN(?)", whereStmt)
		args = append(args, user.ID)
	}
	return countBookings(ctx, db, whereStmt, args...)
}

//MarkBookingViewed : mark a booking as viewed
func MarkBookingViewed(ctx context.Context, db *DB, id *uuid.UUID) (context.Context, error) {
	stmt := fmt.Sprintf("UPDATE %s SET viewed=1 WHERE id=UUID_TO_BIN(?)", dbTableBooking)
	ctx, _, err := db.Exec(ctx, stmt, id)
	if err != nil {
		return ctx, errors.Wrap(err, "mark booking read")
	}
	return ctx, nil
}

//MarkBookingConfirmed : mark a booking as confirmed
func MarkBookingConfirmed(ctx context.Context, db *DB, svc *Service, book *Booking, now time.Time) (context.Context, error) {
	ctx, err := db.ProcessTx(ctx, "mark booking confirmed", func(ctx context.Context, db *DB) (context.Context, error) {
		//check for a coupon
		if book.CouponCodeChange {
			_, coupon, err := LoadCouponByProviderIDAndCode(ctx, db, book.Provider.ID, book.CouponCode, &now)
			if err != nil {
				return nil, errors.Wrap(err, fmt.Sprintf("load coupon: %s", book.CouponCode))
			}
			book.Coupon = coupon
		}

		//apply the coupon
		if book.Coupon != nil {
			book.ServicePriceOriginal = svc.Price
			book.ServicePrice = book.Coupon.AdjustPrice(svc.Price, book.Service.ID, false, now)
		}

		//json encode the booking data
		dataJSON, err := json.Marshal(book)
		if err != nil {
			return ctx, errors.Wrap(err, "json booking")
		}

		//update the booking, triggering a calendar update
		stmt := fmt.Sprintf("UPDATE %s SET confirmed=1,event_google_update=1,data=? WHERE id=UUID_TO_BIN(?) OR parent_id=UUID_TO_BIN(?)", dbTableBooking)
		ctx, _, err = db.Exec(ctx, stmt, dataJSON, book.ID, book.ID)
		if err != nil {
			return ctx, errors.Wrap(err, "update booking confirmed")
		}
		book.Confirmed = true
		return ctx, nil
	})
	if err != nil {
		return ctx, errors.Wrap(err, "mark booking confirmed")
	}
	return ctx, nil
}

//CountBookings : count bookings
func CountBookings(ctx context.Context, db *DB) (context.Context, int, error) {
	stmt := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE deleted=0", dbTableBooking)
	ctx, row, err := db.QueryRow(ctx, stmt)
	if err != nil {
		return ctx, 0, errors.Wrap(err, "query row bookings count")
	}

	//read the row
	var count int
	err = row.Scan(&count)
	if err != nil {
		if err == sql.ErrNoRows {
			return ctx, 0, nil
		}
		return ctx, 0, errors.Wrap(err, "select bookings count")
	}
	return ctx, count, nil
}

//CountBookingsForService : find the number bookings for a service
func CountBookingsForService(ctx context.Context, db *DB, svcID *uuid.UUID) (context.Context, int, error) {
	stmt := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE deleted=0 AND service_id=UUID_TO_BIN(?)", dbTableBooking)
	ctx, row, err := db.QueryRow(ctx, stmt, svcID)
	if err != nil {
		return ctx, 0, errors.Wrap(err, "query row bookings count service")
	}

	//read the row
	var count int
	err = row.Scan(&count)
	if err != nil {
		if err == sql.ErrNoRows {
			return ctx, 0, nil
		}
		return ctx, 0, errors.Wrap(err, "select bookings count service")
	}
	return ctx, count, nil
}

//CountBookingsForClient : find the number bookings for a client
func CountBookingsForClient(ctx context.Context, db *DB, clientID *uuid.UUID) (context.Context, int, error) {
	stmt := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE deleted=0 AND client_id=UUID_TO_BIN(?)", dbTableBooking)
	ctx, row, err := db.QueryRow(ctx, stmt, clientID)
	if err != nil {
		return ctx, 0, errors.Wrap(err, "query row bookings count client")
	}

	//read the row
	var count int
	err = row.Scan(&count)
	if err != nil {
		if err == sql.ErrNoRows {
			return ctx, 0, nil
		}
		return ctx, 0, errors.Wrap(err, "select bookings count client")
	}
	return ctx, count, nil
}

//CountBookingsForProviderAndTime : find the number bookings for a provider over the given time
func CountBookingsForProviderAndTime(ctx context.Context, db *DB, providerID *uuid.UUID, user *User, start time.Time, end time.Time, serviceType ServiceType) (context.Context, int, error) {
	start = start.UTC()
	end = end.UTC()
	stmt := fmt.Sprintf("SELECT COUNT(*) FROM %s b INNER JOIN %s s ON s.id=b.service_id INNER JOIN %s p ON p.id=s.provider_id LEFT JOIN %s pu ON pu.id=b.provider_user_id AND pu.deleted=0 LEFT JOIN %s puu ON puu.id=pu.user_id AND puu.deleted=0 WHERE b.deleted=0 AND p.id=UUID_TO_BIN(?) AND ((b.time_start_padded>=? AND b.time_start_padded<?) OR (b.time_end_padded>? AND b.time_end_padded<=?)) AND b.service_type=?", dbTableBooking, dbTableService, dbTableProvider, dbTableProviderUser, dbTableUser)
	args := []interface{}{providerID, start.UTC(), end.UTC(), start.UTC(), end.UTC(), serviceType}

	//match the user if set
	if user != nil {
		stmt = fmt.Sprintf("%s AND puu.id=UUID_TO_BIN(?)", stmt)
		args = append(args, user.ID)
	}
	ctx, row, err := db.QueryRow(ctx, stmt, args...)
	if err != nil {
		return ctx, 0, errors.Wrap(err, "query row bookings time")
	}

	//read the row
	var count int
	err = row.Scan(&count)
	if err != nil {
		if err == sql.ErrNoRows {
			return ctx, 0, nil
		}
		return ctx, 0, errors.Wrap(err, "select bookings time")
	}
	return ctx, count, nil
}

//ListBookingEventsToProcessForGoogle : list booking events to process for Google
func ListBookingEventsToProcessForGoogle(ctx context.Context, db *DB, limit int) (context.Context, []*Booking, error) {
	ctx, logger := GetLogger(ctx)
	var err error
	var bookings []*Booking
	ctx, err = db.ProcessTx(ctx, "list booking events process google", func(ctx context.Context, db *DB) (context.Context, error) {
		//list the bookings
		stmt := fmt.Sprintf("SELECT BIN_TO_UUID(b.id) FROM %s b INNER JOIN %s s ON s.id=b.service_id INNER JOIN %s p on p.id=s.provider_id WHERE (b.deleted=0 OR b.event_google_delete=1) AND b.event_google_processing=0 AND (b.event_google_id IS NULL OR b.event_google_update=1 OR b.event_google_delete=1) AND p.calendar_google_id IS NOT NULL ORDER BY b.created LIMIT %d", dbTableBooking, dbTableService, dbTableProvider, limit)
		ctx, rows, err := db.Query(ctx, stmt)
		if err != nil {
			return ctx, errors.Wrap(err, "select bookings")
		}
		defer func() {
			err := rows.Close()
			if err != nil {
				logger.Warnw("rows close", "error", err)
			}
		}()

		//read the rows
		bookingIds := make([]*uuid.UUID, 0, 2)
		var idStr string
		for rows.Next() {
			err := rows.Scan(&idStr)
			if err != nil {
				return ctx, errors.Wrap(err, "rows scan bookings")
			}

			//parse the uuid
			id, err := uuid.FromString(idStr)
			if err != nil {
				return ctx, errors.Wrap(err, "parse uuid id")
			}
			bookingIds = append(bookingIds, &id)
		}
		if len(bookingIds) == 0 {
			return ctx, nil
		}

		//mark the bookings as being processed
		MarkBookingsProcessingGoogle(ctx, db, bookingIds)

		//load data for the bookings
		for _, bookingID := range bookingIds {
			ctx, booking, err := LoadBookingByID(ctx, db, bookingID, false, false)
			if err != nil {
				return ctx, errors.Wrap(err, fmt.Sprintf("load booking: %s", bookingID))
			}
			bookings = append(bookings, booking)
		}
		return ctx, nil
	})
	if err != nil {
		return ctx, nil, errors.Wrap(err, "list booking events process google")
	}
	return ctx, bookings, nil
}

//MarkBookingsProcessingGoogle : mark bookings as processing for Google
func MarkBookingsProcessingGoogle(ctx context.Context, db *DB, bookingIds []*uuid.UUID) (context.Context, error) {
	lenIds := len(bookingIds)
	if lenIds == 0 {
		return ctx, fmt.Errorf("no bookings to mark")
	}

	//prepare the ids
	args := make([]interface{}, lenIds)
	for i, id := range bookingIds {
		args[i] = id.String()
	}

	//generate the list of parameters to use in the query
	paramsStr := fmt.Sprintf("(UUID_TO_BIN(?)%s)", strings.Repeat(",UUID_TO_BIN(?)", lenIds-1))

	//mark the bookings
	stmt := fmt.Sprintf("UPDATE %s SET event_google_processing=1,event_google_processing_time=CURRENT_TIMESTAMP() WHERE event_google_processing=0 AND id in %s", dbTableBooking, paramsStr)
	ctx, result, err := db.Exec(ctx, stmt, args...)
	if err != nil {
		return ctx, errors.Wrap(err, "mark bookings processing google")
	}
	count, err := result.RowsAffected()
	if err != nil {
		return ctx, errors.Wrap(err, "mark bookings processing google rows affected")
	}
	if int(count) != lenIds {
		return ctx, fmt.Errorf("unable to mark bookings processing google: %d: %d", count, lenIds)
	}
	return ctx, nil
}

//UpdateBookingEventGoogle : update the booking Google calendar event information
func UpdateBookingEventGoogle(ctx context.Context, db *DB, book *Booking, eventData *EventGoogle) (context.Context, error) {
	//json encode the booking data
	dataJSON, err := json.Marshal(book)
	if err != nil {
		return ctx, errors.Wrap(err, "json booking")
	}

	//json encode the event data
	eventJSON, err := json.Marshal(eventData)
	if err != nil {
		return ctx, errors.Wrap(err, "json event")
	}

	//update the event fields for the booking
	stmt := fmt.Sprintf("UPDATE %s SET event_google_id=?,event_google_update=0,event_google_delete=0,event_google_processing=0,event_google_data=?,data=? WHERE id=UUID_TO_BIN(?)", dbTableBooking)
	ctx, result, err := db.Exec(ctx, stmt, book.EventGoogleID, eventJSON, dataJSON, book.ID)
	if err != nil {
		return ctx, errors.Wrap(err, "update booking")
	}
	count, err := result.RowsAffected()
	if err != nil {
		return ctx, errors.Wrap(err, "update booking rows affected")
	}
	if count != 1 {
		return ctx, fmt.Errorf("unable to update booking: %s", book.ID)
	}
	return ctx, nil
}

//ListBookingEventsToProcessForZoom : list booking events to process for Zoom
func ListBookingEventsToProcessForZoom(ctx context.Context, db *DB, limit int) (context.Context, []*Booking, error) {
	ctx, logger := GetLogger(ctx)
	var err error
	var bookings []*Booking
	ctx, err = db.ProcessTx(ctx, "list booking events process zoom", func(ctx context.Context, db *DB) (context.Context, error) {
		//list the bookings
		stmt := fmt.Sprintf("SELECT BIN_TO_UUID(b.id) FROM %s b INNER JOIN %s s ON s.id=b.service_id INNER JOIN %s p on p.id=s.provider_id WHERE (b.deleted=0 OR b.meeting_zoom_delete=1) AND b.meeting_zoom_processing=0 AND (b.meeting_zoom_update=1 OR b.meeting_zoom_delete=1) ORDER BY b.created LIMIT %d", dbTableBooking, dbTableService, dbTableProvider, limit)
		ctx, rows, err := db.Query(ctx, stmt)
		if err != nil {
			return ctx, errors.Wrap(err, "select bookings")
		}
		defer func() {
			err := rows.Close()
			if err != nil {
				logger.Warnw("rows close", "error", err)
			}
		}()

		//read the rows
		bookingIds := make([]*uuid.UUID, 0, 2)
		var idStr string
		for rows.Next() {
			err := rows.Scan(&idStr)
			if err != nil {
				return ctx, errors.Wrap(err, "rows scan bookings")
			}

			//parse the uuid
			id, err := uuid.FromString(idStr)
			if err != nil {
				return ctx, errors.Wrap(err, "parse uuid id")
			}
			bookingIds = append(bookingIds, &id)
		}
		if len(bookingIds) == 0 {
			return ctx, nil
		}

		//mark the bookings as being processed
		MarkBookingsProcessingZoom(ctx, db, bookingIds)

		//load data for the bookings
		for _, bookingID := range bookingIds {
			ctx, booking, err := LoadBookingByID(ctx, db, bookingID, false, false)
			if err != nil {
				return ctx, errors.Wrap(err, fmt.Sprintf("load booking: %s", bookingID))
			}
			bookings = append(bookings, booking)
		}
		return ctx, nil
	})
	if err != nil {
		return ctx, nil, errors.Wrap(err, "list booking events process zoom")
	}
	return ctx, bookings, nil
}

//MarkBookingsProcessingZoom : mark bookings as processing for Zoom
func MarkBookingsProcessingZoom(ctx context.Context, db *DB, bookingIds []*uuid.UUID) (context.Context, error) {
	lenIds := len(bookingIds)
	if lenIds == 0 {
		return ctx, fmt.Errorf("no bookings to mark")
	}

	//prepare the ids
	args := make([]interface{}, lenIds)
	for i, id := range bookingIds {
		args[i] = id.String()
	}

	//generate the list of parameters to use in the query
	paramsStr := fmt.Sprintf("(UUID_TO_BIN(?)%s)", strings.Repeat(",UUID_TO_BIN(?)", lenIds-1))

	//mark the bookings
	stmt := fmt.Sprintf("UPDATE %s SET meeting_zoom_processing=1,meeting_zoom_processing_time=CURRENT_TIMESTAMP() WHERE meeting_zoom_processing=0 AND id in %s", dbTableBooking, paramsStr)
	ctx, result, err := db.Exec(ctx, stmt, args...)
	if err != nil {
		return ctx, errors.Wrap(err, "mark bookings processing zoom")
	}
	count, err := result.RowsAffected()
	if err != nil {
		return ctx, errors.Wrap(err, "mark bookings processing zoom rows affected")
	}
	if int(count) != lenIds {
		return ctx, fmt.Errorf("unable to mark bookings processing zoom: %d: %d", count, lenIds)
	}
	return ctx, nil
}

//UpdateBookingMeetingZoom : update the booking Zoom meeting information
func UpdateBookingMeetingZoom(ctx context.Context, db *DB, book *Booking, meetingData *MeetingZoom) (context.Context, error) {
	//json encode the event data
	var err error
	var meetingJSON []byte
	if meetingData != nil {
		meetingJSON, err = json.Marshal(meetingData)
		if err != nil {
			return ctx, errors.Wrap(err, "json meeting")
		}
	}

	//update the meeting fields for the booking
	var stmt string
	var args []interface{}
	if meetingJSON == nil {
		stmt = fmt.Sprintf("UPDATE %s SET meeting_zoom_id=?,meeting_zoom_update=0,meeting_zoom_delete=0,meeting_zoom_processing=0 WHERE id=UUID_TO_BIN(?)", dbTableBooking)
		args = []interface{}{book.MeetingZoomID, book.ID}
	} else {
		stmt = fmt.Sprintf("UPDATE %s SET meeting_zoom_id=?,meeting_zoom_update=0,meeting_zoom_delete=0,meeting_zoom_processing=0,meeting_zoom_data=? WHERE id=UUID_TO_BIN(?)", dbTableBooking)
		args = []interface{}{book.MeetingZoomID, meetingJSON, book.ID}
	}
	ctx, result, err := db.Exec(ctx, stmt, args...)
	if err != nil {
		return ctx, errors.Wrap(err, "update booking")
	}
	count, err := result.RowsAffected()
	if err != nil {
		return ctx, errors.Wrap(err, "update booking rows affected")
	}
	if count != 1 {
		return ctx, fmt.Errorf("unable to update booking: %s", book.ID)
	}
	return ctx, nil
}

//ListRecurringToProcess : list recurring bookings to process
func ListRecurringToProcess(ctx context.Context, db *DB, now time.Time, limit int) (context.Context, []*Booking, error) {
	ctx, logger := GetLogger(ctx)
	var err error
	var bookings []*Booking
	ctx, err = db.ProcessTx(ctx, "list booking events process", func(ctx context.Context, db *DB) (context.Context, error) {
		//lean out the time based on the look-ahead
		now = now.AddDate(0, recurringLookAheadMonths, 0)

		//list the bookings
		stmt := fmt.Sprintf("SELECT BIN_TO_UUID(id) FROM %s WHERE deleted=0 AND recurrence_processing=0 AND recurrence_instance_end<? AND recurrence_instance_end!=? ORDER BY created LIMIT %d", dbTableBooking, limit)
		ctx, rows, err := db.Query(ctx, stmt, now, MaxTime)
		if err != nil {
			return ctx, errors.Wrap(err, "select bookings")
		}
		defer func() {
			err := rows.Close()
			if err != nil {
				logger.Warnw("rows close", "error", err)
			}
		}()

		//read the rows
		bookingIds := make([]*uuid.UUID, 0, 2)
		var idStr string
		for rows.Next() {
			err := rows.Scan(&idStr)
			if err != nil {
				return ctx, errors.Wrap(err, "rows scan bookings")
			}

			//parse the uuid
			id, err := uuid.FromString(idStr)
			if err != nil {
				return ctx, errors.Wrap(err, "parse uuid id")
			}
			bookingIds = append(bookingIds, &id)
		}
		if len(bookingIds) == 0 {
			return ctx, nil
		}

		//mark the bookings as being processed
		MarkBookingsProcessingRecurring(ctx, db, bookingIds)

		//load data for the bookings
		for _, bookingID := range bookingIds {
			ctx, booking, err := LoadBookingByID(ctx, db, bookingID, false, false)
			if err != nil {
				return ctx, errors.Wrap(err, fmt.Sprintf("load booking: %s", bookingID))
			}
			bookings = append(bookings, booking)
		}
		return ctx, nil
	})
	if err != nil {
		return ctx, nil, errors.Wrap(err, "list booking events process")
	}
	return ctx, bookings, nil
}

//MarkBookingsProcessingRecurring : mark bookings as processing for recurrence
func MarkBookingsProcessingRecurring(ctx context.Context, db *DB, bookingIds []*uuid.UUID) (context.Context, error) {
	lenIds := len(bookingIds)
	if lenIds == 0 {
		return ctx, fmt.Errorf("no bookings to mark")
	}

	//prepare the ids
	args := make([]interface{}, lenIds)
	for i, id := range bookingIds {
		args[i] = id.String()
	}

	//generate the list of parameters to use in the query
	paramsStr := fmt.Sprintf("(UUID_TO_BIN(?)%s)", strings.Repeat(",UUID_TO_BIN(?)", lenIds-1))

	//mark the bookings
	stmt := fmt.Sprintf("UPDATE %s SET recurrence_processing=1,recurrence_processing_time=CURRENT_TIMESTAMP() WHERE recurrence_processing=0 AND id in %s", dbTableBooking, paramsStr)
	ctx, result, err := db.Exec(ctx, stmt, args...)
	if err != nil {
		return ctx, errors.Wrap(err, "mark bookings processing")
	}
	count, err := result.RowsAffected()
	if err != nil {
		return ctx, errors.Wrap(err, "mark bookings processing rows affected")
	}
	if int(count) != lenIds {
		return ctx, fmt.Errorf("unable to mark bookings processing: %d: %d", count, lenIds)
	}
	return ctx, nil
}

//UpdateBookingRecurring : update the booking recurring information
func UpdateBookingRecurring(ctx context.Context, db *DB, book *Booking) (context.Context, error) {
	stmt := fmt.Sprintf("UPDATE %s SET recurrence_processing=0,recurrence_instance_end=? WHERE id=UUID_TO_BIN(?)", dbTableBooking)
	ctx, result, err := db.Exec(ctx, stmt, book.RecurrenceInstanceEnd, book.ID)
	if err != nil {
		return ctx, errors.Wrap(err, "update booking")
	}
	count, err := result.RowsAffected()
	if err != nil {
		return ctx, errors.Wrap(err, "update booking rows affected")
	}
	if count != 1 {
		return ctx, fmt.Errorf("unable to update booking: %s", book.ID)
	}
	return ctx, nil
}

//UpdateBookingPaddingForService : update the booking padding
func UpdateBookingPaddingForService(ctx context.Context, db *DB, svcID *uuid.UUID, padding int, now time.Time) (context.Context, error) {
	stmt := fmt.Sprintf("UPDATE %s SET time_start_padded=DATE_SUB(time_start, INTERVAL ? MINUTE),time_end_padded=DATE_ADD(time_end, INTERVAL ? MINUTE) WHERE deleted=0 AND service_id=UUID_TO_BIN(?) AND time_start>?", dbTableBooking)
	ctx, result, err := db.Exec(ctx, stmt, padding, padding, svcID, now)
	if err != nil {
		return ctx, errors.Wrap(err, "update booking padding")
	}
	count, err := result.RowsAffected()
	if err != nil {
		return ctx, errors.Wrap(err, "update booking padding rows affected")
	}
	if count != 1 {
		return ctx, fmt.Errorf("unable to update booking padding: %s", svcID)
	}
	return ctx, nil
}

//UpdateBookingData : update the booking data
func UpdateBookingData(ctx context.Context, db *DB, book *Booking) (context.Context, error) {
	//json encode the booking data
	dataJSON, err := json.Marshal(book)
	if err != nil {
		return ctx, errors.Wrap(err, "json booking")
	}

	//update
	stmt := fmt.Sprintf("UPDATE %s SET data=? WHERE id=UUID_TO_BIN(?)", dbTableBooking)
	ctx, result, err := db.Exec(ctx, stmt, dataJSON, book.ID)
	if err != nil {
		return ctx, errors.Wrap(err, "update booking")
	}
	_, err = result.RowsAffected()
	if err != nil {
		return ctx, errors.Wrap(err, "update booking rows affected")
	}
	return ctx, nil
}

//FindLatestBooking : find the latest booking create time
func FindLatestBooking(ctx context.Context, db *DB) (context.Context, *Provider, *time.Time, error) {
	stmt := fmt.Sprintf("SELECT p.data,b.created FROM %s b INNER JOIN %s p ON p.id=b.provider_id AND p.deleted=0 INNER JOIN %s u ON u.id=p.user_id AND u.deleted=0 AND u.test=0 WHERE b.deleted=0 ORDER BY b.created DESC LIMIT 1", dbTableBooking, dbTableProvider, dbTableUser)
	ctx, row, err := db.QueryRow(ctx, stmt)
	if err != nil {
		return ctx, nil, nil, errors.Wrap(err, "query row booking time create")
	}

	//read the row
	var providerDataStr string
	var t time.Time
	err = row.Scan(&providerDataStr, &t)
	if err != nil {
		if err == sql.ErrNoRows {
			return ctx, nil, nil, nil
		}
		return ctx, nil, nil, errors.Wrap(err, "select booking time create")
	}

	//unmarshal the data
	var provider Provider
	err = json.Unmarshal([]byte(providerDataStr), &provider)
	if err != nil {
		return ctx, nil, nil, errors.Wrap(err, "unjson provider")
	}
	return ctx, &provider, &t, nil
}
