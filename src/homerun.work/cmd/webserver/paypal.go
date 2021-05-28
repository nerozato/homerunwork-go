package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/plutov/paypal/v3"
)

//paypal db tables
const (
	dbTableEventPayPal = "paypal_event"
)

//paypal constants
const (
	PayPalResourceTypeCapture              = "capture"
	PayPalResourceVersion                  = "2.0"
	PayPalEventTypePaymentCaptureCompleted = "PAYMENT.CAPTURE.COMPLETED"
	PayPalEventVersion                     = "1.0"
	PayPalVerificationStatusSuccess        = "SUCCESS"
)

//OrderPayPal : wrapper for a PayPal order
type OrderPayPal struct {
	*paypal.Order
}

//LinkPayPal : PayPal link
type LinkPayPal struct {
	Href   string `json:"href"`
	Rel    string `json:"rel"`
	Method string `json:"method"`
}

//MoneyPayPal : PayPal money
type MoneyPayPal struct {
	CurrencyCode string `json:"currency_code"`
	Value        string `json:"value"`
}

//SellerProtectionPayPal : PayPal seller protection
type SellerProtectionPayPal struct {
	Status string `json:"status"`
}

//SellerBreakdownPayPal : PayPal seller breakdown
type SellerBreakdownPayPal struct {
	GrossAmount MoneyPayPal `json:"gross_amount"`
	PayPalFee   MoneyPayPal `json:"paypal_fee"`
	NetAmount   MoneyPayPal `json:"net_amount"`
}

//ResourcePaymentPayPal : PayPal payment resource
type ResourcePaymentPayPal struct {
	ID               string                 `json:"id"`
	InvoiceID        string                 `json:"invoice_id"`
	CustomID         string                 `json:"custom_id"`
	Status           string                 `json:"status"`
	CreateTime       string                 `json:"create_time"`
	UpdateTime       string                 `json:"update_time"`
	FinalCapture     bool                   `json:"final_capture"`
	Links            []LinkPayPal           `json:"links"`
	Amount           MoneyPayPal            `json:"amount"`
	SellerProtection SellerProtectionPayPal `json:"seller_protection"`
	SellerBreakdown  SellerBreakdownPayPal  `json:"seller_receivable_breakdown"`
}

//EventPayPal : PayPal event
type EventPayPal struct {
	ID              string          `json:"id"`
	CreateTime      string          `json:"create_time"`
	EventType       string          `json:"event_type"`
	EventVersion    string          `json:"event_version"`
	ResourceType    string          `json:"resource_type"`
	ResourceVersion string          `json:"resource_version"`
	Summary         string          `json:"summary"`
	Links           []LinkPayPal    `json:"links"`
	Resource        json.RawMessage `json:"resource"`
}

//ParseEventPayPal : parse PayPal event JSON
func ParseEventPayPal(in []byte) (*EventPayPal, error) {
	var event EventPayPal
	err := json.Unmarshal(in, &event)
	if err != nil {
		return nil, errors.Wrap(err, "parse paypal event")
	}
	return &event, nil
}

//ParseResourcePaymentPayPal : parse a PayPal payment resource
func ParseResourcePaymentPayPal(in []byte) (*ResourcePaymentPayPal, error) {
	var resource ResourcePaymentPayPal
	err := json.Unmarshal(in, &resource)
	if err != nil {
		return nil, errors.Wrap(err, "parse paypal payment resource")
	}
	return &resource, nil
}

//logger wrapper
type loggerPayPal struct {
	logger *Logger
}

//write to the logger
func (l *loggerPayPal) Write(data []byte) (int, error) {
	l.logger.Debugw("paypal client", "data", string(data))
	return len(data), nil
}

//create a paypal client
func createClientPayPal() (*paypal.Client, error) {
	client, err := paypal.NewClient(GetPayPalClientID(), GetPayPalSecret(), GetPayPalURLApi())
	if err != nil {
		return nil, errors.Wrap(err, "paypal client")
	}
	_, err = client.GetAccessToken()
	if err != nil {
		return nil, errors.Wrap(err, "paypal access token")
	}

	//set the logger
	_, logger := GetLogger(nil)
	clientLogger := &loggerPayPal{logger}
	client.SetLog(clientLogger)
	return client, nil
}

//CreateOrderPayPal : create a PayPal order
func CreateOrderPayPal(ctx context.Context, payeeEmail *string, providerName string, desc string, paymentID string, customID string, amount float32) (*OrderPayPal, error) {
	ctx, logger := GetLogger(ctx)
	start := time.Now()
	defer func() {
		logger.Debugw("paypal create order", "elapsedMS", FormatElapsedMS(start))
		AddCtxStatsAPI(ctx, ServerStatAPIPayPal, "paypal create order", time.Since(start))
	}()

	//set-up the order parameters
	request := paypal.PurchaseUnitRequest{
		Description: desc,
		InvoiceID:   paymentID,
		CustomID:    customID,
		Amount: &paypal.PurchaseUnitAmount{
			Value:    strconv.FormatFloat(float64(amount), 'f', 2, 32),
			Currency: "USD",
		},
	}
	if payeeEmail != nil {
		request.Payee = &paypal.PayeeForOrders{
			EmailAddress: *payeeEmail,
		}
	}

	//create an order context
	appCtx := &paypal.ApplicationContext{
		BrandName:          providerName,
		ShippingPreference: "NO_SHIPPING",
		UserAction:         "PAY_NOW",
	}
	client, err := createClientPayPal()
	if err != nil {
		return nil, errors.Wrap(err, "paypal create client")
	}
	result, err := client.CreateOrder(paypal.OrderIntentCapture, []paypal.PurchaseUnitRequest{request}, nil, appCtx)
	if err != nil {
		return nil, errors.Wrap(err, "paypal create order")
	}
	order := &OrderPayPal{result}
	return order, nil
}

//VerifyWebHookSignaturePayPal : verify the payload signature for a PayPal webhook event
func VerifyWebHookSignaturePayPal(r *http.Request, webhookID string) (bool, error) {
	ctx, logger := GetLogger(r.Context())
	start := time.Now()
	defer func() {
		logger.Debugw("paypal verify webhook", "elapsedMS", FormatElapsedMS(start))
		AddCtxStatsAPI(ctx, ServerStatAPIPayPal, "paypal verify webhook", time.Since(start))
	}()

	//verify the signature
	client, err := createClientPayPal()
	if err != nil {
		return false, errors.Wrap(err, "paypal create client")
	}
	result, err := client.VerifyWebhookSignature(r.WithContext(ctx), webhookID)
	if err != nil {
		return false, errors.Wrap(err, "paypal verify signature")
	}
	ok := result.VerificationStatus == PayPalVerificationStatusSuccess
	return ok, nil
}

//SaveEventPayPal : save a PayPal event
func SaveEventPayPal(ctx context.Context, db *DB, event *EventPayPal) (context.Context, error) {
	//json encode the message data
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return ctx, errors.Wrap(err, "json event")
	}

	//save to the db
	stmt := fmt.Sprintf("INSERT INTO %s(id,data) VALUES (?,?)", dbTableEventPayPal)
	ctx, result, err := db.Exec(ctx, stmt, event.ID, eventJSON)
	if err != nil {
		return ctx, errors.Wrap(err, "insert event paypal")
	}
	count, err := result.RowsAffected()
	if err != nil {
		return ctx, errors.Wrap(err, "insert event paypal rows affected")
	}
	if count == 0 {
		return ctx, fmt.Errorf("unable to insert event paypal: %s", event.ID)
	}
	return ctx, nil
}
