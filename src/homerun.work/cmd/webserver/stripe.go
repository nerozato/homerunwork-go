package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/charge"
	"github.com/stripe/stripe-go/checkout/session"
	"github.com/stripe/stripe-go/oauth"
	"github.com/stripe/stripe-go/webhook"
)

//stripe db tables
const (
	dbTableEventStripe = "stripe_event"
)

//stripe constants
const (
	StripeEventTypeCheckoutSessionCompleted = "checkout.session.completed"
	StripeEventTypePaymentIntentSucceeded   = "payment_intent.succeeded"
	StripeHeaderSignature                   = "Stripe-Signature"
	StripeOAuthURL                          = "https://connect.stripe.com/oauth/authorize"
	StripePaymentIntentStatusSuccess        = "succeeded"
	StripeURLParamSessionID                 = "{CHECKOUT_SESSION_ID}"
)

//TokenStripe : wrapper for a Stripe OAuth token
type TokenStripe struct {
	*stripe.OAuthToken
}

//EncryptKeys : protect internal token data
func (t *TokenStripe) EncryptKeys() error {
	var err error
	t.AccessToken, err = EncryptString(t.AccessToken)
	if err != nil {
		return errors.Wrap(err, "encrypt access token")
	}
	t.RefreshToken, err = EncryptString(t.RefreshToken)
	if err != nil {
		return errors.Wrap(err, "encrypt refresh token")
	}
	t.StripePublishableKey, err = EncryptString(t.StripePublishableKey)
	if err != nil {
		return errors.Wrap(err, "encrypt publishable key")
	}
	t.StripeUserID, err = EncryptString(t.StripeUserID)
	if err != nil {
		return errors.Wrap(err, "encrypt user id")
	}
	return nil
}

//GetStripeUserID : get the Stripe user id
func (t *TokenStripe) GetStripeUserID() (string, error) {
	id, err := DecryptString(t.StripeUserID)
	if err != nil {
		return "", errors.Wrap(err, "decrypt user id")
	}
	return id, nil
}

//SessionStripe : wrapper for a Stripe session
type SessionStripe struct {
	*stripe.CheckoutSession
}

//PaymentIntentStripe : wrapper for a Stripe payment intent
type PaymentIntentStripe struct {
	*stripe.PaymentIntent
}

//EventStripe : wrapper for a Stripe event
type EventStripe struct {
	*stripe.Event
}

//InitStripe : initialize stripe
func InitStripe(logger *Logger) {
	stripe.DefaultLeveledLogger = logger
	stripe.EnableTelemetry = false
	stripe.Key = GetStripeSecretKey()
}

//CreateOAuthURLStripe : create the Stripe URL used for OAuth
func CreateOAuthURLStripe(ctx context.Context, token string, redirectURL string, email string, firstName string, lastName string, providerName string, providerURL string) (string, error) {
	params := map[string]interface{}{
		"client_id":                  GetStripeClientID(),
		"redirect_uri":               redirectURL,
		"response_type":              "code",
		"scope":                      "read_write",
		"state":                      token,
		"stripe_user[business_name]": providerName,
		"stripe_user[email]":         email,
		"stripe_user[first_name]":    firstName,
		"stripe_user[last_name]":     lastName,
		"stripe_user[url]":           providerURL,
	}
	url, err := CreateURLRel(StripeOAuthURL, params)
	if err != nil {
		return "", errors.Wrap(err, "create url")
	}
	return url, nil
}

//RetrieveOAuthTokenStripe : retrieve the Stripe OAuth token based on the code
func RetrieveOAuthTokenStripe(ctx context.Context, code string) (*TokenStripe, error) {
	ctx, logger := GetLogger(ctx)
	start := time.Now()
	defer func() {
		logger.Debugw("stripe oauth token", "elapsedMS", FormatElapsedMS(start))
		AddCtxStatsAPI(ctx, ServerStatAPIStripe, "stripe oauth token", time.Since(start))
	}()

	//retrieve the token
	params := &stripe.OAuthTokenParams{
		GrantType: stripe.String("authorization_code"),
		Code:      stripe.String(code),
	}
	result, err := oauth.New(params)
	if err != nil {
		return nil, errors.Wrap(err, "stripe oauth token")
	}

	//encrypt the keys
	token := &TokenStripe{result}
	err = token.EncryptKeys()
	if err != nil {
		return nil, errors.Wrap(err, "stripe token encrypt keys")
	}
	return token, nil
}

//RevokeOAuthTokenStripe : revoke access to an account
func RevokeOAuthTokenStripe(ctx context.Context, token *TokenStripe) error {
	ctx, logger := GetLogger(ctx)
	start := time.Now()
	defer func() {
		logger.Debugw("stripe oauth revoke", "elapsedMS", FormatElapsedMS(start))
		AddCtxStatsAPI(ctx, ServerStatAPIStripe, "stripe oauth revoke", time.Since(start))
	}()

	//get the stripe user id
	stripeUserID, err := token.GetStripeUserID()
	if err != nil {
		return errors.Wrap(err, "stripe get user id")
	}

	//revoke access for the account
	params := &stripe.DeauthorizeParams{
		ClientID:     stripe.String(GetStripeClientID()),
		StripeUserID: stripe.String(stripeUserID),
	}
	_, err = oauth.Del(params)
	if err != nil {
		return errors.Wrap(err, "stripe oauth revoke")
	}
	return nil
}

//CreateSessionStripe : create a Stripe checkout session
func CreateSessionStripe(ctx context.Context, token *TokenStripe, providerName string, desc string, paymentID string, bookID string, amount int, url string) (*SessionStripe, error) {
	ctx, logger := GetLogger(ctx)
	start := time.Now()
	defer func() {
		logger.Debugw("stripe create session", "elapsedMS", FormatElapsedMS(start))
		AddCtxStatsAPI(ctx, ServerStatAPIStripe, "stripe create session", time.Since(start))
	}()

	//ensure the session id is returned on success
	var err error
	url, err = CreateURLAbs(ctx, url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "create url")
	}
	var urlSuccess string
	if strings.Contains(url, "?") {
		urlSuccess = fmt.Sprintf("%s&%s=%s", url, URLParams.StripeID, StripeURLParamSessionID)
	} else {
		urlSuccess = fmt.Sprintf("%s?%s=%s", url, URLParams.StripeID, StripeURLParamSessionID)
	}

	//prepare a session for checkout
	params := &stripe.CheckoutSessionParams{
		ClientReferenceID: stripe.String(paymentID),
		SuccessURL:        stripe.String(urlSuccess),
		CancelURL:         stripe.String(url),
		Mode:              stripe.String("payment"),
		SubmitType:        stripe.String("pay"),
		PaymentMethodTypes: stripe.StringSlice([]string{
			"card",
		}),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Amount:      stripe.Int64(int64(amount)),
				Currency:    stripe.String(string(stripe.CurrencyUSD)),
				Quantity:    stripe.Int64(1),
				Name:        stripe.String(providerName),
				Description: stripe.String(desc),
			},
		},
	}

	//get the stripe user id
	if token != nil {
		stripeUserID, err := token.GetStripeUserID()
		if err != nil {
			return nil, errors.Wrap(err, "stripe get user id")
		}
		params.SetStripeAccount(stripeUserID)
	}

	//create the session
	result, err := session.New(params)
	if err != nil {
		return nil, errors.Wrap(err, "stripe create session")
	}
	session := &SessionStripe{result}
	return session, nil
}

//ChargeStripe : Stripe charge
type ChargeStripe struct {
	*stripe.Charge
}

//CreateStripeCharge : create a Stripe direct charge
func CreateStripeCharge(ctx context.Context, token *TokenStripe, providerName string, desc string, paymentID string, amount int, tokenSrc string) (*ChargeStripe, error) {
	ctx, logger := GetLogger(ctx)
	start := time.Now()
	defer func() {
		logger.Debugw("stripe charge", "elapsedMS", FormatElapsedMS(start))
		AddCtxStatsAPI(ctx, ServerStatAPIStripe, "stripe charge", time.Since(start))
	}()

	//create the charge
	params := &stripe.ChargeParams{
		StatementDescriptor: stripe.String(providerName),
		Description:         stripe.String(desc),
		Amount:              stripe.Int64(int64(amount)),
		Currency:            stripe.String(string(stripe.CurrencyUSD)),
		Source: &stripe.SourceParams{
			Token: stripe.String(tokenSrc),
		},
	}
	if token != nil {
		stripeUserID, err := token.GetStripeUserID()
		if err != nil {
			return nil, errors.Wrap(err, "stripe get user id")
		}
		params.TransferData = &stripe.ChargeTransferDataParams{
			Destination: stripe.String(stripeUserID),
		}
		params.TransferGroup = stripe.String(paymentID)
	}
	result, err := charge.New(params)
	if err != nil {
		return nil, errors.Wrap(err, "stripe create charge")
	}
	charge := &ChargeStripe{result}
	return charge, nil
}

//VerifyWebHookSignatureStripe : verify the payload signature for a Stripe webhook event
func VerifyWebHookSignatureStripe(w http.ResponseWriter, r *http.Request, secret string) (*EventStripe, string, error) {
	ctx, logger := GetLogger(r.Context())
	start := time.Now()
	defer func() {
		logger.Debugw("stripe verify webhook", "elapsedMS", FormatElapsedMS(start))
		AddCtxStatsAPI(ctx, ServerStatAPIStripe, "stripe verify webhook", time.Since(start))
	}()

	//read the body
	const MaxBodyBytes = int64(65536)
	r.Body = http.MaxBytesReader(w, r.Body, int64(65536))
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, "", errors.Wrap(err, "stripe read body")
	}

	//verify the signature
	signature := r.Header.Get(StripeHeaderSignature)
	result, err := webhook.ConstructEvent(body, signature, secret)
	if err != nil {
		return nil, "", errors.Wrap(err, "stripe create event")
	}
	event := &EventStripe{&result}
	return event, string(body), nil
}

//ParseSessionStripe : parse a Stripe checkout session
func ParseSessionStripe(in []byte) (*SessionStripe, error) {
	var result stripe.CheckoutSession
	err := json.Unmarshal(in, &result)
	if err != nil {
		return nil, errors.Wrap(err, "parse stripe session")
	}
	intent := &SessionStripe{&result}
	return intent, nil
}

//ParsePaymentIntentStripe : parse a Stripe payment intent
func ParsePaymentIntentStripe(in []byte) (*PaymentIntentStripe, error) {
	var result stripe.PaymentIntent
	err := json.Unmarshal(in, &result)
	if err != nil {
		return nil, errors.Wrap(err, "parse stripe payment intent")
	}
	intent := &PaymentIntentStripe{&result}
	return intent, nil
}

//SaveEventStripe : save a Stripe event
func SaveEventStripe(ctx context.Context, db *DB, event *EventStripe) (context.Context, error) {
	//json encode the message data
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return ctx, errors.Wrap(err, "json event")
	}

	//save to the db
	stmt := fmt.Sprintf("INSERT INTO %s(id,data) VALUES (?,?)", dbTableEventStripe)
	ctx, result, err := db.Exec(ctx, stmt, event.ID, eventJSON)
	if err != nil {
		return ctx, errors.Wrap(err, "insert event stripe")
	}
	count, err := result.RowsAffected()
	if err != nil {
		return ctx, errors.Wrap(err, "insert event stripe rows affected")
	}
	if count == 0 {
		return ctx, fmt.Errorf("unable to insert event stripe: %s", event.ID)
	}
	return ctx, nil
}
