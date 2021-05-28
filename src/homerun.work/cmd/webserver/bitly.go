package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

//bitly constants
const (
	//general
	BitlyRequestTimeOut = 10 * time.Second

	//urls
	BitlyURLShorten = "https://api-ssl.bitly.com/v4/shorten"
)

//create a bitly http client
func createClientBitly() *http.Client {
	client := &http.Client{
		Timeout: BitlyRequestTimeOut,
	}
	return client
}

//BitlyURLInput : input for shortening a URL
type BitlyURLInput struct {
	URL string `json:"long_url"`
}

//BitlyURL : representation of a shortened URL from Bitly
type BitlyURL struct {
	ID          string      `json:"id"`
	ClientID    string      `json:"client_id"`
	URL         string      `json:"link"`
	URLLong     string      `json:"long_url"`
	Title       string      `json:"title"`
	Archived    bool        `json:"archived"`
	CreatedAt   string      `json:"created_at"`
	CreatedBy   string      `json:"created_by"`
	Tags        []string    `json:"tags"`
	CustomLinks []string    `json:"custom_bitlinks"`
	DeepLinks   interface{} `json:"deeplinks"`
	References  interface{} `json:"references"`
}

//ShortenURLBitly : shorten a URL using Bitly
func ShortenURLBitly(ctx context.Context, urlIn string) (context.Context, *BitlyURL, error) {
	ctx, logger := GetLogger(ctx)
	start := time.Now()
	defer func() {
		logger.Debugw("bitly shorten url", "elapsedMS", FormatElapsedMS(start))
		AddCtxStatsAPI(ctx, ServerStatAPIBitly, "bitly shorten url", time.Since(start))
	}()

	//create the request
	in := BitlyURLInput{
		URL: urlIn,
	}
	data, err := json.Marshal(in)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "input json")
	}

	//setup the request
	request, err := http.NewRequest("POST", BitlyURLShorten, bytes.NewBuffer(data))
	if err != nil {
		return ctx, nil, errors.Wrap(err, "bitly shorten url create http request")
	}

	//set the token
	header := fmt.Sprintf("Bearer %s", GetBitlyAccessToken())
	request.Header.Set("Authorization", header)

	//make the request
	request.Header.Set(HeaderContentType, "application/json")
	client := createClientBitly()
	response, err := client.Do(request)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "bitly shorten url http request")
	}
	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusCreated {
		body, _ := ioutil.ReadAll(response.Body)
		return ctx, nil, fmt.Errorf("status: %d: %s", response.StatusCode, body)
	}

	//process the response
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "bitly shorten url read body")
	}
	var urlData BitlyURL
	err = json.Unmarshal(body, &urlData)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "bitly shorten url unjson")
	}
	defer response.Body.Close()
	return ctx, &urlData, nil
}
