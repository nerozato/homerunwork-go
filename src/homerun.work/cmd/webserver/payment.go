package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

//payment db tables
const (
	dbTablePayment = "payment"
)

//payment constants
const (
	PaymentFriendlyIDLength = 10
)

//PaymentType : payment type
type PaymentType int

//payment types
const (
	PaymentTypeBooking PaymentType = iota + 1
	PaymentTypeCampaign
	PaymentTypeDirect
)

//Payment : definition of a Payment
type Payment struct {
	ID              *uuid.UUID  `json:"-"`
	ProviderID      *uuid.UUID  `json:"-"`
	SecondaryID     *uuid.UUID  `json:"-"`
	FriendlyID      string      `json:"-"`
	Type            PaymentType `json:"-"`
	Amount          int         `json:"-"` //non-decimal, pennies in USD
	PayPalID        *string     `json:"-"`
	StripeAccountID *string     `json:"-"`
	StripeSessionID *string     `json:"-"`
	StripeID        *string     `json:"-"`
	ExternalData    *string     `json:"-"`
	Invoiced        *time.Time  `json:"-"`
	Paid            *time.Time  `json:"-"`
	Captured        *time.Time  `json:"-"`
	Client          *Client     `json:"-"`
	Name            string      `json:"Name"`
	Email           string      `json:"Email"`
	Phone           string      `json:"Phone"`
	ProviderName    string      `json:"ProviderName"`
	Description     string      `json:"Description"`
	Note            string      `json:"Note"`
	URL             string      `json:"Url"`
	ClientInitiated bool        `json:"ClientInitiated"`
	DirectCapture   bool        `json:"DirectCapture"`
	Internal        bool        `json:"Internal"`
	ServiceID       string      `json:"ServiceID"`
}

//SetAmount : set the payment amount as a fractionless number
func (p *Payment) SetAmount(amount float32) {
	//convert to the lowest unit
	amount = amount * 100
	p.Amount = int(math.Ceil(float64(amount)))
}

//GetAmount : get the payment amount
func (p *Payment) GetAmount() float32 {
	return float32(p.Amount) / 100
}

//IsCaptured : check if a payment has been captured
func (p *Payment) IsCaptured() bool {
	return p.Captured != nil
}

//IsInvoiced : check if a payment has been invoiced
func (p *Payment) IsInvoiced() bool {
	return p.Invoiced != nil
}

//IsPaid : check if a payment has been paid
func (p *Payment) IsPaid() bool {
	return p.Paid != nil
}

//AllowUnPay : check if a payment can be marked as unpaid
func (p *Payment) AllowUnPay() bool {
	return p.IsCaptured() && (p.StripeID == nil || p.PayPalID == nil)
}

//FormatCaptured : format the payment captured date
func (p *Payment) FormatCaptured(timeZone string) string {
	if p.Captured == nil {
		return ""
	}
	return FormatDateTimeLocal(*p.Captured, timeZone)
}

//FormatInvoiced : format the payment invoice date and time
func (p *Payment) FormatInvoiced(timeZone string) string {
	if p.Invoiced == nil {
		return ""
	}
	return FormatDateTimeLocal(*p.Invoiced, timeZone)
}

//FormatInvoicedDate : format the payment invoice date
func (p *Payment) FormatInvoicedDate(timeZone string) string {
	if p.Invoiced == nil {
		return ""
	}
	return FormatDateLocal(*p.Invoiced, timeZone)
}

//FormatPaid : format the payment paid date
func (p *Payment) FormatPaid(timeZone string) string {
	if p.Paid == nil {
		return ""
	}
	return FormatDateTimeLocal(*p.Paid, timeZone)
}

//create the statement to load a payment
func paymentQueryCreate(whereStmt string) string {
	stmt := fmt.Sprintf("SELECT BIN_TO_UUID(p.id),BIN_TO_UUID(p.provider_id),BIN_TO_UUID(p.secondary_id),p.friendly_id,p.type,p.amount,p.invoiced,p.paid,p.captured,p.stripe_id,p.stripe_session_id,p.stripe_account_id,p.paypal_id,p.data,c.email,c.data FROM %s p LEFT JOIN %s b ON b.id=p.secondary_id LEFT JOIN %s c ON c.id=b.client_id WHERE %s ORDER BY p.invoiced DESC", dbTablePayment, dbTableBooking, dbTableClient, whereStmt)
	return stmt
}

//parse a payment
func paymentQueryParse(rowFn ScanFn) (*Payment, error) {
	//read the row
	var paymentIDStr string
	var providerIDStr string
	var secondaryIDStr string
	var friendlyID string
	var paymentType int
	var amount int
	var invoiced sql.NullTime
	var paid sql.NullTime
	var captured sql.NullTime
	var stripeID sql.NullString
	var stripeSessionID sql.NullString
	var stripeAccountID sql.NullString
	var paypalID sql.NullString
	var dataStr string
	var clientEmail sql.NullString
	var clientDataStr sql.NullString
	err := rowFn(&paymentIDStr, &providerIDStr, &secondaryIDStr, &friendlyID, &paymentType, &amount, &invoiced, &paid, &captured, &stripeID, &stripeSessionID, &stripeAccountID, &paypalID, &dataStr, &clientEmail, &clientDataStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, errors.Wrap(err, "scan payment")
	}

	//parse the uuid
	paymentID, err := uuid.FromString(paymentIDStr)
	if err != nil {
		return nil, errors.Wrap(err, "parse uuid payment id")
	}
	providerID, err := uuid.FromString(providerIDStr)
	if err != nil {
		return nil, errors.Wrap(err, "parse uuid provider id")
	}
	secondaryID, err := uuid.FromString(secondaryIDStr)
	if err != nil {
		return nil, errors.Wrap(err, "parse uuid secondary id")
	}

	//unmarshal the payment
	var payment Payment
	err = json.Unmarshal([]byte(dataStr), &payment)
	if err != nil {
		return nil, errors.Wrap(err, "unjson payment")
	}
	payment.ID = &paymentID
	payment.ProviderID = &providerID
	payment.SecondaryID = &secondaryID
	payment.FriendlyID = friendlyID
	payment.Type = PaymentType(paymentType)
	payment.Amount = amount
	if invoiced.Valid {
		payment.Invoiced = &invoiced.Time
	}
	if paid.Valid {
		payment.Paid = &paid.Time
	}
	if captured.Valid {
		payment.Captured = &captured.Time
	}
	if stripeID.Valid {
		payment.StripeID = &stripeID.String
	}
	if stripeSessionID.Valid {
		payment.StripeSessionID = &stripeSessionID.String
	}
	if stripeAccountID.Valid {
		payment.StripeAccountID = &stripeAccountID.String
	}
	if paypalID.Valid {
		payment.PayPalID = &paypalID.String
	}

	//unmarshal the client data
	if clientDataStr.Valid {
		var client Client
		err = json.Unmarshal([]byte(clientDataStr.String), &client)
		if err != nil {
			return nil, errors.Wrap(err, "unjson client")
		}
		payment.Client = &client

		if clientEmail.Valid {
			client.Email = clientEmail.String
		}
	}
	return &payment, nil
}

//loadPayment : load a payment
func loadPayment(ctx context.Context, db *DB, whereStmt string, args ...interface{}) (context.Context, *Payment, error) {
	stmt := paymentQueryCreate(whereStmt)
	ctx, row, err := db.QueryRow(ctx, stmt, args...)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "query row payment")
	}
	payment, err := paymentQueryParse(row.Scan)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "payment parse")
	}
	return ctx, payment, nil
}

//LoadPaymentByID : load a payment by the id
func LoadPaymentByID(ctx context.Context, db *DB, id *uuid.UUID) (context.Context, *Payment, error) {
	whereStmt := "p.deleted=0 AND p.id=UUID_TO_BIN(?)"
	ctx, payment, err := loadPayment(ctx, db, whereStmt, id)
	if err != nil {
		return ctx, nil, err
	}
	if payment == nil {
		return ctx, nil, errors.Wrap(err, fmt.Sprintf("no payment: %s", id))
	}
	return ctx, payment, nil
}

//LoadPaymentByProviderIDAndSecondaryIDAndType : load a payment by the provider id and secondary id and type
func LoadPaymentByProviderIDAndSecondaryIDAndType(ctx context.Context, db *DB, providerID *uuid.UUID, id *uuid.UUID, paymentType PaymentType) (context.Context, *Payment, error) {
	whereStmt := "p.deleted=0 AND p.provider_id=UUID_TO_BIN(?) AND p.secondary_id=UUID_TO_BIN(?) AND p.type=?"
	return loadPayment(ctx, db, whereStmt, providerID, id, paymentType)
}

//SavePayment : save a payment
func SavePayment(ctx context.Context, db *DB, payment *Payment, isDirectCapture bool, deletePrevious bool) (context.Context, error) {
	ctx, err := db.ProcessTx(ctx, "save payment", func(ctx context.Context, db *DB) (context.Context, error) {
		//delete any previous payments
		if deletePrevious {
			stmt := fmt.Sprintf("UPDATE %s SET deleted=1 WHERE deleted=0 AND provider_id=UUID_TO_BIN(?) AND secondary_id=UUID_TO_BIN(?)", dbTablePayment)
			ctx, _, err := db.Exec(ctx, stmt, payment.ProviderID, payment.SecondaryID)
			if err != nil {
				return ctx, errors.Wrap(err, "update payment")
			}
		}

		//generate an id if necessary
		if payment.ID == nil {
			id, err := uuid.NewV4()
			if err != nil {
				return ctx, errors.Wrap(err, "new uuid payment")
			}
			payment.ID = &id
		}

		//generate a friendly id if necessary
		if payment.FriendlyID == "" {
			payment.FriendlyID = GenURLStringRndm(PaymentFriendlyIDLength)
		}

		//json encode the data
		paymentJSON, err := json.Marshal(payment)
		if err != nil {
			return ctx, errors.Wrap(err, "json payment")
		}

		//check for dates
		var paid *time.Time
		if payment.Paid != nil {
			utc := payment.Paid.UTC()
			paid = &utc
		}
		var captured *time.Time
		if payment.Captured != nil {
			utc := payment.Captured.UTC()
			captured = &utc
		}

		//save to the db, updating the paid and captured dates if necessary
		stmt := fmt.Sprintf("INSERT INTO %s(id,provider_id,secondary_id,friendly_id,type,amount,invoiced,paid,captured,data) VALUES (UUID_TO_BIN(?),UUID_TO_BIN(?),UUID_TO_BIN(?),?,?,?,?,?,?,?)", dbTablePayment)
		ctx, result, err := db.Exec(ctx, stmt, payment.ID, payment.ProviderID, payment.SecondaryID, payment.FriendlyID, payment.Type, payment.Amount, payment.Invoiced.UTC(), paid, captured, paymentJSON)
		if err != nil {
			return ctx, errors.Wrap(err, "insert payment")
		}
		count, err := result.RowsAffected()
		if err != nil {
			return ctx, errors.Wrap(err, "insert payment rows affected")
		}
		if count == 0 {
			return ctx, fmt.Errorf("unable to insert payment: %s: %s", payment.ProviderID, payment.SecondaryID)
		}
		return ctx, nil
	})
	if err != nil {
		return ctx, errors.Wrap(err, "save payment")
	}
	return ctx, nil
}

//UpdatePaymentUnPaid : update the unpaid state of the payment
func UpdatePaymentUnPaid(ctx context.Context, db *DB, id *uuid.UUID) (context.Context, error) {
	stmt := fmt.Sprintf("UPDATE %s SET paid=NULL,captured=NULL WHERE id=UUID_TO_BIN(?)", dbTablePayment)
	ctx, _, err := db.Exec(ctx, stmt, id)
	if err != nil {
		return ctx, errors.Wrap(err, "update payment unpaid")
	}
	return ctx, nil
}

//UpdatePaymentPaid : update the paid state of the payment
func UpdatePaymentPaid(ctx context.Context, db *DB, id *uuid.UUID, paid *time.Time, captured *time.Time) (context.Context, error) {
	var stmt string
	var args []interface{}
	if captured == nil {
		stmt = fmt.Sprintf("UPDATE %s SET paid=? WHERE id=UUID_TO_BIN(?)", dbTablePayment)
		args = []interface{}{paid, id}
	} else {
		stmt = fmt.Sprintf("UPDATE %s SET paid=?,captured=? WHERE id=UUID_TO_BIN(?)", dbTablePayment)
		args = []interface{}{paid, captured, id}
	}
	ctx, _, err := db.Exec(ctx, stmt, args...)
	if err != nil {
		return ctx, errors.Wrap(err, "update payment paid")
	}
	return ctx, nil
}

//UpdatePaymentDirectCapture : update a payment as directly captured
func UpdatePaymentDirectCapture(ctx context.Context, db *DB, id *uuid.UUID, captured *time.Time) (context.Context, error) {
	stmt := fmt.Sprintf("UPDATE %s SET captured=?,data=JSON_SET(data,'$.DirectCapture',true) WHERE id=UUID_TO_BIN(?)", dbTablePayment)
	ctx, _, err := db.Exec(ctx, stmt, captured, id)
	if err != nil {
		return ctx, errors.Wrap(err, "update payment direct capture")
	}
	return ctx, nil
}

//UpdatePaymentStripeID : update the Stripe id and associated data for a payment
func UpdatePaymentStripeID(ctx context.Context, db *DB, id *uuid.UUID, stripeID *string, stripeSessionID *string, stripeAccountID *string, data *string) (context.Context, error) {
	stmt := fmt.Sprintf("UPDATE %s SET stripe_id=?,stripe_session_id=?,stripe_account_id=?,stripe_data=? WHERE id=UUID_TO_BIN(?)", dbTablePayment)
	ctx, _, err := db.Exec(ctx, stmt, stripeID, stripeSessionID, stripeAccountID, data, id)
	if err != nil {
		return ctx, errors.Wrap(err, "update payment stripe id")
	}
	return ctx, nil
}

//UpdatePaymentPayPalID : update the PayPal id and associated data for a payment
func UpdatePaymentPayPalID(ctx context.Context, db *DB, id *uuid.UUID, paypalID *string, data *string) (context.Context, error) {
	stmt := fmt.Sprintf("UPDATE %s SET paypal_id=?,paypal_data=? WHERE id=UUID_TO_BIN(?)", dbTablePayment)
	ctx, _, err := db.Exec(ctx, stmt, paypalID, data, id)
	if err != nil {
		return ctx, errors.Wrap(err, "update payment paypal id")
	}
	return ctx, nil
}

//UpdatePaymentCaptured : update the captured state of the payment
func UpdatePaymentCaptured(ctx context.Context, db *DB, id *string, externalData *string, now *time.Time) (context.Context, error) {
	stmt := fmt.Sprintf("UPDATE %s SET external_data=?,captured=?,deleted=0 WHERE id=UUID_TO_BIN(?)", dbTablePayment)
	ctx, _, err := db.Exec(ctx, stmt, externalData, now, id)
	if err != nil {
		return ctx, errors.Wrap(err, "update payment captured")
	}
	return ctx, nil
}

//UpdatePaymentCapturedByExternalID : update the captured state of the payment by the external id
func UpdatePaymentCapturedByExternalID(ctx context.Context, db *DB, id *string, externalData *string, now *time.Time) (context.Context, error) {
	stmt := fmt.Sprintf("UPDATE %s SET external_data=?,captured=?,deleted=0 WHERE stripe_id=? OR paypal_id=?", dbTablePayment)
	ctx, _, err := db.Exec(ctx, stmt, externalData, now, id, id)
	if err != nil {
		return ctx, errors.Wrap(err, "update payment captured by external id")
	}
	return ctx, nil
}

//UpdatePaymentData : update the payment data
func UpdatePaymentData(ctx context.Context, db *DB, payment *Payment) (context.Context, error) {
	//json encode the data
	dataJSON, err := json.Marshal(payment)
	if err != nil {
		return ctx, errors.Wrap(err, "json payment")
	}

	//update
	stmt := fmt.Sprintf("UPDATE %s SET data=? WHERE id=UUID_TO_BIN(?)", dbTablePayment)
	ctx, _, err = db.Exec(ctx, stmt, dataJSON, payment.ID)
	if err != nil {
		return ctx, errors.Wrap(err, "update payment")
	}
	return ctx, nil
}

//DeletePayment : delete a payment
func DeletePayment(ctx context.Context, db *DB, id *uuid.UUID) (context.Context, error) {
	stmt := fmt.Sprintf("UPDATE %s SET deleted=1 WHERE deleted=0 AND captured IS NULL AND id=UUID_TO_BIN(?)", dbTablePayment)
	ctx, _, err := db.Exec(ctx, stmt, id)
	if err != nil {
		return ctx, errors.Wrap(err, "delete payment")
	}
	return ctx, nil
}

//ListPaymentsByProviderIDAndFilter : list bookings by the provider and filter
func ListPaymentsByProviderIDAndFilter(ctx context.Context, db *DB, providerID *uuid.UUID, filter PaymentFilter) (context.Context, []*Payment, error) {
	ctx, logger := GetLogger(ctx)

	//load the payments based on the filter
	whereStmt := "p.deleted=0 AND p.provider_id=UUID_TO_BIN(?) AND (p.type=? OR (p.type=? AND p.paid IS NOT NULL))"
	switch filter {
	case PaymentFilterAll:
	case PaymentFilterUnPaid:
		whereStmt = fmt.Sprintf("%s AND p.captured IS NULL", whereStmt)
	default:
		return ctx, nil, fmt.Errorf("invalid filter: %s", filter)
	}
	stmt := paymentQueryCreate(whereStmt)
	ctx, rows, err := db.Query(ctx, stmt, providerID, PaymentTypeBooking, PaymentTypeDirect)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "select payments")
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			logger.Warnw("rows close select payments", "error", err)
		}
	}()

	//read the bookings
	payments := make([]*Payment, 0, 2)
	for rows.Next() {
		payment, err := paymentQueryParse(rows.Scan)
		if err != nil {
			return ctx, nil, errors.Wrap(err, "payment parse")
		}
		payments = append(payments, payment)
	}
	return ctx, payments, nil
}

//CountPaymentsByProviderIDAndFilter : count the payments for a provider based on the filter
func CountPaymentsByProviderIDAndFilter(ctx context.Context, db *DB, providerID *uuid.UUID, filter PaymentFilter) (context.Context, int, error) {
	whereStmt := "deleted=0 AND provider_id=UUID_TO_BIN(?) AND (type=? OR (type=? AND paid IS NOT NULL))"
	switch filter {
	case PaymentFilterAll:
	case PaymentFilterUnPaid:
		whereStmt = fmt.Sprintf("%s AND captured IS NULL", whereStmt)
	default:
		return ctx, 0, fmt.Errorf("invalid filter: %s", filter)
	}

	//count the payments
	stmt := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s", dbTablePayment, whereStmt)
	ctx, row, err := db.QueryRow(ctx, stmt, providerID, PaymentTypeBooking, PaymentTypeDirect)
	if err != nil {
		return ctx, 0, errors.Wrap(err, "query row payments count")
	}

	//read the row
	var count int
	err = row.Scan(&count)
	if err != nil {
		if err == sql.ErrNoRows {
			return ctx, 0, nil
		}
		return ctx, 0, errors.Wrap(err, "select payments count")
	}
	return ctx, count, nil
}

//CountPayments : count payments
func CountPayments(ctx context.Context, db *DB) (context.Context, int, error) {
	stmt := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE deleted=0", dbTablePayment)
	ctx, row, err := db.QueryRow(ctx, stmt)
	if err != nil {
		return ctx, 0, errors.Wrap(err, "query row payment count")
	}

	//read the row
	var count int
	err = row.Scan(&count)
	if err != nil {
		if err == sql.ErrNoRows {
			return ctx, 0, nil
		}
		return ctx, 0, errors.Wrap(err, "select payment count")
	}
	return ctx, count, nil
}

//FindLatestPayment : find the latest payment create time
func FindLatestPayment(ctx context.Context, db *DB) (context.Context, *Provider, *time.Time, error) {
	stmt := fmt.Sprintf("SELECT p.data,b.created FROM %s b INNER JOIN %s p ON p.id=b.provider_id AND p.deleted=0 INNER JOIN %s u ON u.id=p.user_id AND u.deleted=0 AND u.test=0 WHERE b.deleted=0 ORDER BY b.created DESC LIMIT 1", dbTablePayment, dbTableProvider, dbTableUser)
	ctx, row, err := db.QueryRow(ctx, stmt)
	if err != nil {
		return ctx, nil, nil, errors.Wrap(err, "query row payment time create")
	}

	//read the row
	var providerDataStr string
	var t time.Time
	err = row.Scan(&providerDataStr, &t)
	if err != nil {
		if err == sql.ErrNoRows {
			return ctx, nil, nil, nil
		}
		return ctx, nil, nil, errors.Wrap(err, "select payment time create")
	}

	//unmarshal the data
	var provider Provider
	err = json.Unmarshal([]byte(providerDataStr), &provider)
	if err != nil {
		return ctx, nil, nil, errors.Wrap(err, "unjson provider")
	}
	return ctx, &provider, &t, nil
}
