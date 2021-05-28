package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"

	clientOAuth2 "google.golang.org/api/oauth2/v2"
)

//google constants
const (
	clientSAExpirationSec = 1800
	requestTimeOut        = 10 * time.Second

	//parameters
	paramSecret   = "secret"
	paramResponse = "response"

	//urls
	calendarURL        = "https://calendar.google.com/calendar?cid=%s"
	calendarIcalURL    = "https://calendar.google.com/calendar/ical/%s/public/basic.ics"
	recaptchaVerifyURL = "https://www.google.com/recaptcha/api/siteverify"
)

//UserGoogle : user data from Google
type UserGoogle struct {
	ID            string
	InternalID    string
	Email         string
	EmailVerified bool
	FirstName     string
	LastName      string
}

//RecaptchaVerificationGoogle : verification response for Google Recaptcha
type RecaptchaVerificationGoogle struct {
	Success     bool   `json:"success"`
	ChallengeTS string `json:"challenge_ts"`
	Hostname    string `json:"hostname"`
}

//get the oauth configuration
func getOAuthCfg(ctx context.Context, isCalendar bool) *oauth2.Config {
	var o sync.Once
	var oauthCfg *oauth2.Config
	o.Do(func() {
		//construct the redirect url, making sure to always use the base server host, which can be different due to custom provider domains
		urlCallback := URICallback
		if isCalendar {
			urlCallback = URICallbackCal
		}
		url, err := createGoogleURL(urlCallback)
		if err != nil {
			panic(errors.Wrap(err, "google redirect url"))
		}
		url, err = CreateURLAbs(ctx, url, nil)
		if err != nil {
			panic(errors.Wrap(err, "google redirect url absolute"))
		}

		//configure the oauth request
		oauthCfg = &oauth2.Config{
			ClientID:     GetGoogleOAuthClientID(),
			ClientSecret: GetGoogleOAuthClientSecret(),
			Endpoint:     google.Endpoint,
			RedirectURL:  url,
		}
		if isCalendar {
			oauthCfg.Scopes = []string{
				"https://www.googleapis.com/auth/calendar.readonly",
			}
		} else {
			oauthCfg.Scopes = []string{
				"https://www.googleapis.com/auth/userinfo.email",
				"https://www.googleapis.com/auth/userinfo.profile",
			}
		}
	})
	return oauthCfg
}

//GetURLOAuth : get the URL used for OAuth
func GetURLOAuth(ctx context.Context, token string, isCalendar bool) string {
	url := getOAuthCfg(ctx, isCalendar).AuthCodeURL(token, oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	return url
}

//TokenGoogle : definition of a Google token
type TokenGoogle struct {
	*oauth2.Token
}

//GetGoogleOAuthToken : get the user data from Google
func GetGoogleOAuthToken(ctx context.Context, code string, isCalendar bool) (*TokenGoogle, error) {
	token, err := getOAuthCfg(ctx, isCalendar).Exchange(ctx, code)
	if err != nil {
		return nil, errors.Wrap(err, "invalid google token exchange")
	}
	tokenGoogle := &TokenGoogle{token}
	return tokenGoogle, nil
}

//GetGoogleUserData : get the user data from Google
func GetGoogleUserData(ctx context.Context, code string, isCalendar bool) (*UserGoogle, error) {
	token, err := GetGoogleOAuthToken(ctx, code, isCalendar)
	if err != nil {
		return nil, errors.Wrap(err, "google oauth token")
	}
	tokenSrc := getOAuthCfg(ctx, isCalendar).TokenSource(ctx, token.Token)
	client, err := clientOAuth2.NewService(ctx, option.WithTokenSource(tokenSrc))
	if err != nil {
		return nil, errors.Wrap(err, "google api service")
	}
	userData, err := client.Userinfo.Get().Do()
	if err != nil {
		return nil, errors.Wrap(err, "google user info")
	}
	user := &UserGoogle{
		ID:            FormatGoogleID(userData.Id),
		InternalID:    userData.Id,
		Email:         userData.Email,
		EmailVerified: userData.VerifiedEmail != nil && *userData.VerifiedEmail,
		FirstName:     userData.GivenName,
		LastName:      userData.FamilyName,
	}
	return user, nil
}

//read the service account credentials json
func getClientSAOption(ctx context.Context) *option.ClientOption {
	var o sync.Once
	var opt option.ClientOption
	o.Do(func() {
		data, err := ioutil.ReadFile(GetGoogleSAFile())
		if err != nil {
			panic(errors.Wrap(err, fmt.Sprintf("read file: %s", GetGoogleSAFile())))
		}
		opt = option.WithCredentialsJSON(data)
	})
	return &opt
}

//get the calendar client
func getCalendarClient(ctx context.Context) *calendar.Service {
	var o sync.Once
	var err error
	var client *calendar.Service
	o.Do(func() {
		opt := getClientSAOption(ctx)
		client, err = calendar.NewService(ctx, *opt)
		if err != nil {
			panic(errors.Wrap(err, "google calendar client"))
		}
	})
	return client
}

//CalendarGoogle : definition of a Google calendar
type CalendarGoogle struct {
	*calendar.Calendar
}

//CreateCalendarGoogle : create a Google calendar
func CreateCalendarGoogle(ctx context.Context, title string) (*CalendarGoogle, error) {
	ctx, logger := GetLogger(ctx)
	start := time.Now()
	var err error
	var cal *calendar.Calendar
	var calResult *calendar.Calendar
	defer func() {
		logger.Debugw("google calendar insert", "calendar", cal, "result", calResult, "elapsedMS", FormatElapsedMS(start))
		AddCtxStatsAPI(ctx, ServerStatAPIGoogle, "google calendar insert", time.Since(start))
	}()

	//create the calendar
	client := getCalendarClient(ctx)
	cal = &calendar.Calendar{
		Summary: title,
	}
	calResult, err = client.Calendars.Insert(cal).Do()
	if err != nil {
		return nil, errors.Wrap(err, "google calendar insert")
	}

	//make the calendar public
	rule := &calendar.AclRule{
		Role: "reader",
		Scope: &calendar.AclRuleScope{
			Type: "default",
		},
	}
	aclResult, err := client.Acl.Insert(calResult.Id, rule).Do()
	if err != nil {
		logger.Warnw("google calendar acl public", "error", err, "rule", rule, "id", calResult.Id)
	}
	logger.Debugw("google acl public insert", "rule", rule, "result", aclResult)

	//add an alternate owner
	rule = &calendar.AclRule{
		Role: "owner",
		Scope: &calendar.AclRuleScope{
			Type:  "user",
			Value: GetGoogleCalendarEmail(),
		},
	}
	aclResult, err = client.Acl.Insert(calResult.Id, rule).Do()
	if err != nil {
		logger.Warnw("google calendar acl owner", "error", err, "rule", rule, "id", calResult.Id)
	}
	logger.Debugw("google acl owner insert", "rule", rule, "result", aclResult)
	calWrapper := &CalendarGoogle{calResult}
	return calWrapper, nil
}

//UpdateCalendarGoogle : update a Google calendar
func UpdateCalendarGoogle(ctx context.Context, calendarID *string, title string) (*CalendarGoogle, error) {
	ctx, logger := GetLogger(ctx)
	start := time.Now()
	var err error
	var cal *calendar.Calendar
	var result *calendar.Calendar
	defer func() {
		logger.Debugw("google calendar update", "id", calendarID, "result", result, "elapsedMS", FormatElapsedMS(start))
		AddCtxStatsAPI(ctx, ServerStatAPIGoogle, "google calendar update", time.Since(start))
	}()
	if calendarID == nil {
		return nil, fmt.Errorf("null calendar id")
	}

	//update the calendar
	client := getCalendarClient(ctx)
	cal = &calendar.Calendar{
		Summary: title,
	}
	result, err = client.Calendars.Update(*calendarID, cal).Do()
	if err != nil {
		return nil, errors.Wrap(err, "google calendar update")
	}
	calWrapper := &CalendarGoogle{cal}
	return calWrapper, nil
}

//DeleteCalendarGoogle : delete a Google calendar
func DeleteCalendarGoogle(ctx context.Context, calendarID *string) error {
	ctx, logger := GetLogger(ctx)
	start := time.Now()
	defer func() {
		logger.Debugw("google calendar delete", "id", calendarID, "elapsedMS", FormatElapsedMS(start))
		AddCtxStatsAPI(ctx, ServerStatAPIGoogle, "google calendar delete", time.Since(start))
	}()
	if calendarID == nil {
		return fmt.Errorf("null calendar id")
	}

	//delete the calendar
	client := getCalendarClient(ctx)
	err := client.Calendars.Delete(*calendarID).Do()
	if err != nil {
		return errors.Wrap(err, "google calendar delete")
	}
	return nil
}

//CheckFreeCalendarGoogle : check if the time is free on the Google calendar
func CheckFreeCalendarGoogle(ctx context.Context, calendarID *string, startTime time.Time, endTime time.Time, eventID string) (bool, error) {
	events, err := ListEventsAndInstancesGoogle(ctx, calendarID, startTime, endTime)
	if err != nil {
		return false, errors.Wrap(err, fmt.Sprintf("google calendar list: %s: %s: %s", *calendarID, startTime, endTime))
	}
	count := 0
	for _, event := range events {
		if event.Id != eventID {
			count++
		}
	}
	return count == 0, nil
}

//ListBusyCalendarGoogle : list the busy times on the Google calendar
func ListBusyCalendarGoogle(ctx context.Context, token *TokenGoogle, startTime time.Time, endTime time.Time) (*TokenGoogle, []*TimePeriod, error) {
	ctx, logger := GetLogger(ctx)
	start := time.Now()
	var err error
	var request *calendar.FreeBusyRequest
	var result *calendar.FreeBusyResponse
	defer func() {
		logger.Debugw("google calendar check free", "request", request, "result", result, "elapsedMS", FormatElapsedMS(start))
		AddCtxStatsAPI(ctx, ServerStatAPIGoogle, "google calendar check free", time.Since(start))
	}()

	//refresh the token as necessary
	tokenSrc := getOAuthCfg(ctx, true).TokenSource(ctx, token.Token)
	tokenRefreshed, err := tokenSrc.Token()
	if err != nil {
		return nil, nil, errors.Wrap(err, "google token source")
	}
	var tokenNew *TokenGoogle
	if tokenRefreshed.AccessToken != token.Token.AccessToken {
		tokenNew = &TokenGoogle{tokenRefreshed}
	}

	//query the calendar
	client, err := calendar.NewService(ctx, option.WithTokenSource(tokenSrc))
	if err != nil {
		return tokenNew, nil, errors.Wrap(err, "google api service")
	}
	request = &calendar.FreeBusyRequest{
		TimeMin: startTime.UTC().Format(time.RFC3339),
		TimeMax: endTime.UTC().Format(time.RFC3339),
		Items: []*calendar.FreeBusyRequestItem{
			{
				Id: "primary",
			},
		},
	}
	result, err = client.Freebusy.Query(request).Do()
	if err != nil {
		return tokenNew, nil, errors.Wrap(err, "google calendar free/busy")
	}

	//process the busy times
	periods := make([]*TimePeriod, 0, 1)
	for _, cal := range result.Calendars {
		for _, timePeriod := range cal.Busy {
			start, err := time.Parse(time.RFC3339, timePeriod.Start)
			if err != nil {
				return tokenNew, nil, errors.Wrap(err, "invalid start")
			}
			end, err := time.Parse(time.RFC3339, timePeriod.End)
			if err != nil {
				return tokenNew, nil, errors.Wrap(err, "invalid end")
			}
			period := &TimePeriod{
				Start: start,
				End:   end,
			}
			periods = append(periods, period)
		}
	}
	return tokenNew, periods, nil
}

//CalendarGoogleEntry : definition of a Google calendar list entry
type CalendarGoogleEntry struct {
	*calendar.CalendarListEntry
}

//ListCalendarsGoogle : list google calendars
func ListCalendarsGoogle(ctx context.Context) ([]*CalendarGoogleEntry, error) {
	ctx, logger := GetLogger(ctx)
	start := time.Now()
	defer func() {
		logger.Debugw("google calendar list", "elapsedMS", FormatElapsedMS(start))
		AddCtxStatsAPI(ctx, ServerStatAPIGoogle, "google calendar list", time.Since(start))
	}()

	//list the calendars
	client := getCalendarClient(ctx)
	cals, err := client.CalendarList.List().Do()
	if err != nil {
		return nil, errors.Wrap(err, "google calendar list")
	}
	googleCals := make([]*CalendarGoogleEntry, 0, 2)
	for _, cal := range cals.Items {
		googleCal := &CalendarGoogleEntry{cal}
		googleCals = append(googleCals, googleCal)
	}
	return googleCals, nil
}

//DeleteCalendarsGoogle : delete all Google calendars
func DeleteCalendarsGoogle(ctx context.Context) (int, error) {
	ctx, logger := GetLogger(ctx)
	start := time.Now()
	var err error
	var cals []*CalendarGoogleEntry
	defer func() {
		logger.Debugw("google calendars delete", "count", len(cals), "elapsedMS", FormatElapsedMS(start))
		AddCtxStatsAPI(ctx, ServerStatAPIGoogle, "google calendars delete", time.Since(start))
	}()

	//list calendars
	cals, err = ListCalendarsGoogle(ctx)
	if err != nil {
		return 0, errors.Wrap(err, "list calendars")
	}

	//delete calendars
	for _, cal := range cals {
		err = DeleteCalendarGoogle(ctx, &cal.Id)
		if err != nil {
			return 0, errors.Wrap(err, fmt.Sprintf("delete calendar: %s", cal.Id))
		}
	}
	return len(cals), nil
}

//EventGoogle : wrapper for a Google calendar event
type EventGoogle struct {
	*calendar.Event
}

//GetStartTime : get the start time
func (g *EventGoogle) GetStartTime() (time.Time, error) {
	t, err := time.Parse(time.RFC3339, g.Start.DateTime)
	if err != nil {
		return time.Time{}, errors.Wrap(err, fmt.Sprintf("invalid time: %s", g.Start.DateTime))
	}
	return t, nil
}

//GetEndTime : get the end time
func (g *EventGoogle) GetEndTime() (time.Time, error) {
	t, err := time.Parse(time.RFC3339, g.End.DateTime)
	if err != nil {
		return time.Time{}, errors.Wrap(err, fmt.Sprintf("invalid time: %s", g.Start.DateTime))
	}
	return t, nil
}

//CreateEventGoogle : create a Google calendar event
func CreateEventGoogle(ctx context.Context, calendarID *string, externalID string, startTime time.Time, endTime time.Time, title string, desc string, location string, recurrenceRules []string) (*EventGoogle, error) {
	ctx, logger := GetLogger(ctx)
	start := time.Now()
	var err error
	var event *calendar.Event
	var result *calendar.Event
	defer func() {
		logger.Debugw("google event insert", "event", event, "result", result, "elapsedMS", FormatElapsedMS(start))
		AddCtxStatsAPI(ctx, ServerStatAPIGoogle, "google event insert", time.Since(start))
	}()
	if calendarID == nil {
		return nil, fmt.Errorf("null calendar id")
	}

	//create the event
	client := getCalendarClient(ctx)
	event = &calendar.Event{
		Start: &calendar.EventDateTime{
			DateTime: startTime.UTC().Format(time.RFC3339),
			TimeZone: time.UTC.String(),
		},
		End: &calendar.EventDateTime{
			DateTime: endTime.UTC().Format(time.RFC3339),
			TimeZone: time.UTC.String(),
		},
		Summary:     title,
		Description: desc,
		Location:    location,
		ExtendedProperties: &calendar.EventExtendedProperties{
			Private: make(map[string]string, 2),
		},
	}
	SetEventGoogleExternalID(event, externalID)

	//check for recurrence
	if len(recurrenceRules) > 0 {
		event.Recurrence = recurrenceRules
	}

	//create the event
	result, err = client.Events.Insert(*calendarID, event).Do()
	if err != nil {
		return nil, errors.Wrap(err, "google event insert")
	}
	eventWrapper := &EventGoogle{result}
	return eventWrapper, nil
}

//GetEventGoogle : get a Google calendar event for a recurring event
func GetEventGoogle(ctx context.Context, calendarID *string, eventID *string, externalID string) (*EventGoogle, error) {
	ctx, logger := GetLogger(ctx)
	start := time.Now()
	var err error
	var result *calendar.Event
	defer func() {
		logger.Debugw("google event get", "calendarId", calendarID, "id", eventID, "externalId", externalID, "result", result, "elapsedMS", FormatElapsedMS(start))
		AddCtxStatsAPI(ctx, ServerStatAPIGoogle, "google event get", time.Since(start))
	}()
	if calendarID == nil {
		return nil, fmt.Errorf("null calendar id")
	}
	if eventID == nil {
		return nil, fmt.Errorf("null event id")
	}

	//get the event
	client := getCalendarClient(ctx)
	result, err = client.Events.Get(*calendarID, *eventID).Do()
	if err != nil {
		return nil, errors.Wrap(err, "google event get")
	}

	//sanity check the external id
	event := &EventGoogle{result}
	if externalID != "" {
		eventExternalID := GetEventGoogleExternalID(event)
		if externalID != eventExternalID {
			return nil, fmt.Errorf("invalid instance: %s: %s: %s", *calendarID, *eventID, externalID)
		}
	}
	return event, nil
}

//UpdateEventGoogle : update a Google calendar event
func UpdateEventGoogle(ctx context.Context, calendarID *string, eventID *string, externalID string, startTime time.Time, endTime time.Time, title string, desc string, location string, recurrenceRules []string) (*EventGoogle, error) {
	ctx, logger := GetLogger(ctx)
	start := time.Now()
	var err error
	var event *calendar.Event
	var result *calendar.Event
	defer func() {
		logger.Debugw("google event update", "calendarId", calendarID, "id", eventID, "event", event, "result", result, "elapsedMS", FormatElapsedMS(start))
		AddCtxStatsAPI(ctx, ServerStatAPIGoogle, "google event update", time.Since(start))
	}()
	if calendarID == nil {
		return nil, fmt.Errorf("null calendar id")
	}
	if eventID == nil {
		return nil, fmt.Errorf("null event id")
	}

	//create the event
	client := getCalendarClient(ctx)
	event = &calendar.Event{
		Start: &calendar.EventDateTime{
			DateTime: startTime.Format(time.RFC3339),
			TimeZone: time.UTC.String(),
		},
		End: &calendar.EventDateTime{
			DateTime: endTime.Format(time.RFC3339),
			TimeZone: time.UTC.String(),
		},
		Summary:     title,
		Description: desc,
		Location:    location,
		ExtendedProperties: &calendar.EventExtendedProperties{
			Private: make(map[string]string, 1),
		},
	}
	SetEventGoogleExternalID(event, externalID)

	//check for recurrence
	if len(recurrenceRules) > 0 {
		event.Recurrence = recurrenceRules
	}

	//update the event
	result, err = client.Events.Update(*calendarID, *eventID, event).Do()
	if err != nil {
		return nil, errors.Wrap(err, "google event update")
	}
	eventWrapper := &EventGoogle{result}
	return eventWrapper, nil
}

//DeleteEventGoogle : delete a Google calendar event
func DeleteEventGoogle(ctx context.Context, calendarID *string, eventID *string) error {
	ctx, logger := GetLogger(ctx)
	start := time.Now()
	defer func() {
		logger.Debugw("google event delete", "calendarId", calendarID, "id", eventID, "elapsedMS", FormatElapsedMS(start))
		AddCtxStatsAPI(ctx, ServerStatAPIGoogle, "google event delete", time.Since(start))
	}()
	if calendarID == nil {
		return fmt.Errorf("null calendar id")
	}
	if eventID == nil {
		return fmt.Errorf("null event id")
	}

	//delete the event
	client := getCalendarClient(ctx)
	err := client.Events.Delete(*calendarID, *eventID).Do()
	if err != nil {
		return errors.Wrap(err, "google event delete")
	}
	return nil
}

//CancelEventGoogle : cancel a Google calendar event
func CancelEventGoogle(ctx context.Context, calendarID *string, eventID *string, startTime time.Time, endTime time.Time) (*EventGoogle, error) {
	ctx, logger := GetLogger(ctx)
	start := time.Now()
	var err error
	var event *calendar.Event
	var result *calendar.Event
	defer func() {
		logger.Debugw("google event cancel", "calendarId", calendarID, "id", eventID, "event", event, "result", result, "elapsedMS", FormatElapsedMS(start))
		AddCtxStatsAPI(ctx, ServerStatAPIGoogle, "google event cancel", time.Since(start))
	}()
	if calendarID == nil {
		return nil, fmt.Errorf("null calendar id")
	}
	if eventID == nil {
		return nil, fmt.Errorf("null event id")
	}

	//create the event
	client := getCalendarClient(ctx)
	event = &calendar.Event{
		Start: &calendar.EventDateTime{
			DateTime: startTime.Format(time.RFC3339),
		},
		End: &calendar.EventDateTime{
			DateTime: endTime.Format(time.RFC3339),
		},
		Status: "cancelled",
	}
	result, err = client.Events.Update(*calendarID, *eventID, event).Do()
	if err != nil {
		return nil, errors.Wrap(err, "google event cancel")
	}
	eventWrapper := &EventGoogle{result}
	return eventWrapper, nil
}

//TerminateEventRecurringGoogle : terminate a Google calendar recurring event
func TerminateEventRecurringGoogle(ctx context.Context, calendarID *string, eventID *string, externalID string, endDate time.Time) ([]string, error) {
	ctx, logger := GetLogger(ctx)
	var err error
	var event *EventGoogle
	var result *calendar.Event
	start := time.Now()
	defer func() {
		logger.Debugw("google event terminate recurring", "calendarId", calendarID, "id", eventID, "externalId", externalID, "event", event, "result", result, "elapsedMS", FormatElapsedMS(start))
		AddCtxStatsAPI(ctx, ServerStatAPIGoogle, "google event terminate recurring", time.Since(start))
	}()
	if calendarID == nil {
		return nil, fmt.Errorf("null calendar id")
	}
	if eventID == nil {
		return nil, fmt.Errorf("null instance id")
	}

	//load the event
	event, err = GetEventGoogle(ctx, calendarID, eventID, externalID)
	if err != nil {
		return nil, errors.Wrap(err, "google event get")
	}

	//update the recurrence rules and add the "until" to the rules
	rules := event.Recurrence
	for i, rule := range rules {
		parsedRule, err := ParseRecurrenceRule(rule)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("parse recurrence rule: %s", rule))
		}
		parsedRule.Until = endDate
		rules[i] = parsedRule.RecurString()
	}
	event.Recurrence = rules

	//update the event
	client := getCalendarClient(ctx)
	result, err = client.Events.Update(*calendarID, event.Id, event.Event).Do()
	if err != nil {
		return nil, errors.Wrap(err, "google event update")
	}
	return rules, nil
}

//list Google calendar events
func listEventsGoogle(ctx context.Context, calendarID *string, singleEvents bool, startTime time.Time, endTime time.Time) ([]*EventGoogle, error) {
	ctx, logger := GetLogger(ctx)
	start := time.Now()
	var err error
	var result *calendar.Events
	defer func() {
		count := 0
		if result != nil {
			count = len(result.Items)
			result.Items = nil
		}
		logger.Debugw("google event list", "id", calendarID, "start", startTime, "end", endTime, "result", result, "count", count, "elapsedMS", FormatElapsedMS(start))
		AddCtxStatsAPI(ctx, ServerStatAPIGoogle, "google event list", time.Since(start))
	}()
	if calendarID == nil {
		return nil, fmt.Errorf("null calendar id")
	}

	//list the events
	client := getCalendarClient(ctx)
	call := client.Events.List(*calendarID)

	//include recurring instances
	if singleEvents {
		call = call.SingleEvents(true)
	}

	//time window
	if !startTime.IsZero() {
		call = call.TimeMin(startTime.Format(time.RFC3339))
	}
	if !endTime.IsZero() {
		call = call.TimeMax(endTime.Format(time.RFC3339))
	}
	if !startTime.IsZero() || !endTime.IsZero() {
		call = call.OrderBy("startTime")
	}

	//execute the call
	result, err = call.Do()
	if err != nil {
		return nil, errors.Wrap(err, "google event list")
	}

	//process the events
	events := make([]*EventGoogle, 0, len(result.Items))
	for _, item := range result.Items {
		event := &EventGoogle{item}
		events = append(events, event)
	}
	return events, nil
}

//ListEventsAndInstancesGoogle : list Google calendar events, including recurring event instances
func ListEventsAndInstancesGoogle(ctx context.Context, calendarID *string, startTime time.Time, endTime time.Time) ([]*EventGoogle, error) {
	return listEventsGoogle(ctx, calendarID, true, startTime, endTime)
}

//ListEventsGoogle : list Google calendar events, not including recurring event instances
func ListEventsGoogle(ctx context.Context, calendarID *string, startTime time.Time, endTime time.Time) ([]*EventGoogle, error) {
	return listEventsGoogle(ctx, calendarID, false, startTime, endTime)
}

//CheckInstancesGoogle : check if a Google calendar recurring event has any instances
func CheckInstancesGoogle(ctx context.Context, calendarID *string, recurringID *string) (bool, error) {
	ctx, logger := GetLogger(ctx)
	start := time.Now()
	var err error
	var result *calendar.Events
	var count int
	defer func() {
		if result != nil {
			result.Items = nil
		}
		logger.Debugw("google check instances", "calendarId", calendarID, "id", recurringID, "result", result, "count", count, "elapsedMS", FormatElapsedMS(start))
		AddCtxStatsAPI(ctx, ServerStatAPIGoogle, "google check instances", time.Since(start))
	}()
	if calendarID == nil {
		return false, fmt.Errorf("null calendar id")
	}
	if recurringID == nil {
		return false, fmt.Errorf("null recurring id")
	}

	//list the instances
	client := getCalendarClient(ctx)
	result, err = client.Events.Instances(*calendarID, *recurringID).MaxResults(1).Do()
	if err != nil {
		return false, errors.Wrap(err, "google check instances")
	}

	//check if there are any instances
	count = len(result.Items)
	return count > 0, nil
}

//SetEventGoogleExternalID : set the external id in the Google event
func SetEventGoogleExternalID(event *calendar.Event, v string) {
	if event == nil {
		return
	}
	if event.ExtendedProperties == nil {
		return
	}
	event.ExtendedProperties.Private[URLParams.ExternalID] = v
}

//GetEventGoogleExternalID : get the external id from the Google event
func GetEventGoogleExternalID(event *EventGoogle) string {
	if event == nil {
		return ""
	}
	if event.ExtendedProperties == nil {
		return ""
	}
	v, ok := event.ExtendedProperties.Private[URLParams.ExternalID]
	if !ok {
		return ""
	}
	return v
}

//FormatGoogleID : format the Google OAuth id
func FormatGoogleID(id string) string {
	return fmt.Sprintf("%s:%s", OAuthGoogle, id)
}

//FormatGoogleCalendarURL : format the Google calendar URL
func FormatGoogleCalendarURL(id *string) string {
	if id == nil {
		return ""
	}
	idEncoded := url.QueryEscape(*id)
	return fmt.Sprintf(calendarURL, idEncoded)
}

//FormatGoogleCalendarIcalURL : format the Google calendar Ical URL
func FormatGoogleCalendarIcalURL(id *string) string {
	if id == nil {
		return ""
	}
	idEncoded := url.QueryEscape(*id)
	return fmt.Sprintf(calendarIcalURL, idEncoded)
}

//VerifyRecaptchaResponseGoogle : verify the Google Recaptcha response
func VerifyRecaptchaResponseGoogle(ctx context.Context, in string) (context.Context, error) {
	ctx, logger := GetLogger(ctx)
	start := time.Now()
	defer func() {
		logger.Debugw("google recaptcha verification", "elapsedMS", FormatElapsedMS(start))
		AddCtxStatsAPI(ctx, ServerStatAPIGoogle, "google recaptcha verification", time.Since(start))
	}()

	//create the request
	data := url.Values{}
	data.Set(paramSecret, GetGoogleRecaptchaSecretKey())
	data.Set(paramResponse, in)

	//make the request
	request, err := http.NewRequest("POST", recaptchaVerifyURL, strings.NewReader(data.Encode()))
	if err != nil {
		return ctx, errors.Wrap(err, "google recaptcha http request")
	}
	client := &http.Client{
		Timeout: requestTimeOut,
	}
	request.Header.Set(HeaderContentType, "application/x-www-form-urlencoded")
	response, err := client.Do(request)
	if err != nil {
		return ctx, errors.Wrap(err, "google recaptcha http request")
	}
	if response.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(response.Body)
		return ctx, fmt.Errorf("status: %d: %s", response.StatusCode, body)
	}

	//process the response
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return ctx, errors.Wrap(err, "google recaptcha read body")
	}
	var out RecaptchaVerificationGoogle
	err = json.Unmarshal(body, &out)
	if err != nil {
		return ctx, errors.Wrap(err, "google recaptcha unjson")
	}
	defer response.Body.Close()
	if !out.Success {
		return ctx, fmt.Errorf("google recaptcha invalid")
	}
	return ctx, nil
}
