package main

import (
	"context"
	"encoding/json"
	"time"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/plaid/plaid-go/plaid"
)

//PlaidAccount : Plaid account definition
type PlaidAccount struct {
	ID      string `json:"id"`
	Mask    string `json:"mask"`
	Name    string `json:"name"`
	SubType string `json:"subtype"`
	Type    string `json:"type"`
}

//PlaidInstitution : Plaid institution definition
type PlaidInstitution struct {
	ID   string `json:"institution_id"`
	Name string `json:"name"`
}

//PlaidLinkData : definition of a Plaid link
type PlaidLinkData struct {
	Account     *PlaidAccount     `json:"account"`
	AccountID   string            `json:"account_id"`
	Accounts    []*PlaidAccount   `json:"accounts"`
	Institution *PlaidInstitution `json:"institution"`
	SessionID   string            `json:"link_session_id"`
	PublicToken string            `json:"public_token"`
}

//ParsePlaidLinkData : parse the Plaid link data
func ParsePlaidLinkData(in string) (*PlaidLinkData, error) {
	var data PlaidLinkData
	err := json.Unmarshal([]byte(in), &data)
	if err != nil {
		return nil, errors.Wrap(err, "unjson data")
	}
	return &data, nil
}

//create a paypal client
func createClientPlaid() (*plaid.Client, error) {
	options := plaid.ClientOptions{
		ClientID: GetPlaidClientID(),
		Secret:   GetPlaidSecret(),
	}

	//set the environment
	if GetPlaidSandboxDisable() {
		options.Environment = plaid.Production
	} else {
		options.Environment = plaid.Sandbox
	}
	client, err := plaid.NewClient(options)
	if err != nil {
		return nil, errors.Wrap(err, "plaid client")
	}
	return client, nil
}

//PlaidLinkToken : Plaid link token
type PlaidLinkToken struct {
	*plaid.CreateLinkTokenResponse
}

//CreatePlaidLinkToken : create a Plaid link token
func CreatePlaidLinkToken(ctx context.Context, orderID *uuid.UUID, email string) (*PlaidLinkToken, error) {
	ctx, logger := GetLogger(ctx)
	start := time.Now()
	defer func() {
		logger.Debugw("plaid create link token", "elapsedMS", FormatElapsedMS(start))
		AddCtxStatsAPI(ctx, ServerStatAPIPlaid, "plaid create link token", time.Since(start))
	}()

	//create a link token
	client, err := createClientPlaid()
	if err != nil {
		return nil, errors.Wrap(err, "client link token")
	}
	options := plaid.LinkTokenConfigs{
		ClientName:   GetPlaidName(),
		CountryCodes: []string{"US"},
		Language:     "en",
		Products:     []string{"auth"},
		User: &plaid.LinkTokenUser{
			ClientUserID: orderID.String(),
			EmailAddress: email,
		},
	}
	response, err := client.CreateLinkToken(options)
	if err != nil {
		return nil, errors.Wrap(err, "create link token")
	}
	data := &PlaidLinkToken{&response}
	return data, nil
}

//PlaidAccessToken : Plaid access token
type PlaidAccessToken struct {
	*plaid.ExchangePublicTokenResponse
}

//ExchangePlaidToken : exchange a Plaid public token for an access token
func ExchangePlaidToken(ctx context.Context, token string) (*PlaidAccessToken, error) {
	ctx, logger := GetLogger(ctx)
	start := time.Now()
	defer func() {
		logger.Debugw("plaid exchange token", "elapsedMS", FormatElapsedMS(start))
		AddCtxStatsAPI(ctx, ServerStatAPIPlaid, "plaid exchange token", time.Since(start))
	}()

	//exchange the token
	client, err := createClientPlaid()
	if err != nil {
		return nil, errors.Wrap(err, "client exchange token")
	}
	response, err := client.ExchangePublicToken(token)
	if err != nil {
		return nil, errors.Wrap(err, "exchange token")
	}
	data := &PlaidAccessToken{&response}
	return data, nil
}

//PlaidStripeToken : Plaid token for Stripe
type PlaidStripeToken struct {
	*plaid.CreateStripeTokenResponse
}

//CreatePlaidStripeToken : create a Plaid token for use with Stripe
func CreatePlaidStripeToken(ctx context.Context, token string, accountID string) (*PlaidStripeToken, error) {
	ctx, logger := GetLogger(ctx)
	start := time.Now()
	defer func() {
		logger.Debugw("plaid stripe token", "elapsedMS", FormatElapsedMS(start))
		AddCtxStatsAPI(ctx, ServerStatAPIPlaid, "plaid stripe token", time.Since(start))
	}()

	//create a stripe token
	client, err := createClientPlaid()
	if err != nil {
		return nil, errors.Wrap(err, "client stripe token")
	}
	response, err := client.CreateStripeToken(token, accountID)
	if err != nil {
		return nil, errors.Wrap(err, "create stripe token")
	}
	data := &PlaidStripeToken{&response}
	return data, nil
}
