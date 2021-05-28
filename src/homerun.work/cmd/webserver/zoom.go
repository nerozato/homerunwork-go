package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/pkg/errors"
)

//zoom db tables
const (
	dbTableEventZoom = "zoom_event"
)

//zoom constants
const (
	ZoomAccessTokenExpirationPadding = 300 //5 minutes in seconds
	ZoomHeaderAuth                   = "authorization"
	ZoomRequestTimeOut               = 1 * time.Second
	ZoomURLDataCompliance            = "https://api.zoom.us/oauth/data/compliance"
	ZoomURLMeeting                   = "https://api.zoom.us/v2/users/me/meetings"
	ZoomURLMeetingModify             = "https://api.zoom.us/v2/meetings/%s"
	ZoomURLOAuth                     = "https://zoom.us/oauth/authorize"
	ZoomURLOAuthToken                = "https://zoom.us/oauth/token?grant_type=authorization_code&code=%s&redirect_uri=%s"
	ZoomURLOAuthAccessToken          = "https://zoom.us/oauth/token?grant_type=refresh_token&refresh_token=%s"
	ZoomURLUser                      = "https://api.zoom.us/v2/users/me"
)

//TokenZoom : Zoom OAuth token
type TokenZoom struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	Scope        string `json:"scope"`
	Expiration   int64  `json:"expiration"` //unix expiration time
}

//EncryptKeys : protect internal token data
func (t *TokenZoom) EncryptKeys() error {
	var err error
	t.AccessToken, err = EncryptString(t.AccessToken)
	if err != nil {
		return errors.Wrap(err, "encrypt access token")
	}
	t.RefreshToken, err = EncryptString(t.RefreshToken)
	if err != nil {
		return errors.Wrap(err, "encrypt refresh token")
	}
	return nil
}

//GetAccessToken : get the access token
func (t *TokenZoom) GetAccessToken() (string, error) {
	token, err := DecryptString(t.AccessToken)
	if err != nil {
		return "", errors.Wrap(err, "decrypt access token")
	}
	return token, nil
}

//GetRefreshToken : get the refresh token
func (t *TokenZoom) GetRefreshToken() (string, error) {
	token, err := DecryptString(t.RefreshToken)
	if err != nil {
		return "", errors.Wrap(err, "decrypt refresh token")
	}
	return token, nil
}

//UserZoom : Zoom user
type UserZoom struct {
	ID                 string   `json:"id"`
	FirstName          string   `json:"first_name"`
	LastName           string   `json:"last_name"`
	Email              string   `json:"email"`
	Type               int      `json:"type"`
	RoleName           string   `json:"role_name"`
	PMI                int      `json:"pmi"`
	UsePMI             bool     `json:"use_pmi"`
	TimeZone           string   `json:"timezone"`
	Dept               string   `json:"dept"`
	CreatedAt          string   `json:"created_at"`
	LastLoginTime      string   `json:"last_login_time"`
	LastClientVersion  string   `json:"last_client_version"`
	Language           string   `json:"language"`
	PhoneCountry       string   `json:"phone_country"`
	PhoneNumber        string   `json:"phone_number"`
	VanityURL          string   `json:"vanity_url"`
	PersonalMeetingURL string   `json:"personal_meeting_url"`
	Verified           int      `json:"verified"`
	PicURL             string   `json:"pic_url"`
	CMSUserID          string   `json:"cms_user_id"`
	AccountID          string   `json:"account_id"`
	HostKey            string   `json:"host_key"`
	Status             string   `json:"status"`
	GroupIDs           []string `json:"group_ids"`
	IMGroupIDs         []string `json:"im_group_ids"`
	JID                string   `json:"jid"`
	JobTitle           string   `json:"job_title"`
	Company            string   `json:"company"`
	Location           string   `json:"location"`
}

//MeetingDialInNumbersZoom : Zoom meeting numbers
type MeetingDialInNumbersZoom struct {
	Country     string `json:"country"`
	CountryName string `json:"country_name"`
	City        string `json:"city"`
	Number      string `json:"number"`
	Type        string `json:"type"`
}

//MeetingOccurrenceZoom : Zoom meeting occurrence
type MeetingOccurrenceZoom struct {
	ID       string `json:"occurrence_id"`
	Start    string `json:"start_time"`
	Duration int    `json:"duration"`
	Status   string `json:"status"`
}

//MeetingRecurrenceZoom : Zoom meeting recurrence
type MeetingRecurrenceZoom struct {
	Type           int    `json:"type"`
	RepeatInterval int    `json:"repeat_interval"`
	WeeklyDays     string `json:"weekly_days"`
	MonthlyDay     int    `json:"monthly_day"`
	MonthlyWeek    int    `json:"monthly_week"`
	MonthlyWeekDay int    `json:"monthly_week_day"`
	EndTimes       int    `json:"end_times"`
	EndDateTime    string `json:"end_date_time"`
}

//MeetingSettingsZoom : Zoom meeting settings
type MeetingSettingsZoom struct {
	HostVideo                    bool                        `json:"host_video"`
	ParticipantVideo             bool                        `json:"participant_video"`
	CNMeeting                    bool                        `json:"cn_meeting"`
	INMeeting                    bool                        `json:"in_meeting"`
	JoinBeforeHost               bool                        `json:"join_before_host"`
	MuteUponEntry                bool                        `json:"mute_upon_entry"`
	Watermark                    bool                        `json:"watermark"`
	UsePMI                       bool                        `json:"use_pmi"`
	ApprovalType                 int                         `json:"approval_type"`
	RegistrationType             int                         `json:"registration_type"`
	Audio                        string                      `json:"audio"`
	AutoRecording                string                      `json:"auto_recording"`
	EnforceLogin                 bool                        `json:"enforce_login"`
	EnforceLoginDomains          string                      `json:"enforce_login_domains"`
	AlternativeHosts             string                      `json:"alternative_hosts"`
	CloseRegistration            bool                        `json:"close_registration"`
	WaitingRoom                  bool                        `json:"waiting_room"`
	GlobalDialInCountries        []string                    `json:"global_dial_in_countries"`
	GlobalDialInNumbers          []*MeetingDialInNumbersZoom `json:"global_dial_in_numbers"`
	ContactName                  string                      `json:"contact_name"`
	ContactEmail                 string                      `json:"contact_email"`
	RegistrantsConfirmationEmail bool                        `json:"registrants_confirmation_email"`
	RegistrantsEmailNotification bool                        `json:"registrants_email_notification"`
	MeetingAuthentication        bool                        `json:"meeting_authentication"`
	AuthenticationOption         string                      `json:"authentication_option"`
	AuthenticationDomains        string                      `json:"authentication_domains"`
}

//MeetingTrackingFieldZoom : Zoom meeting tracking field
type MeetingTrackingFieldZoom struct {
	Field string `json:"field"`
	Value string `json:"value"`
}

//MeetingInputZoom : Zoom input meeting
type MeetingInputZoom struct {
	Topic          string                      `json:"topic"`
	Agenda         string                      `json:"agenda"`
	Type           int                         `json:"type"`
	Duration       int                         `json:"duration"`
	Start          string                      `json:"start_time"`
	TimeZone       string                      `json:"timezone"`
	SchedueFor     string                      `json:"schedule_for"`
	Recurrence     *MeetingRecurrenceZoom      `json:"recurrence"`
	Settings       *MeetingSettingsZoom        `json:"settings"`
	TrackingFields []*MeetingTrackingFieldZoom `json:"tracking_fields"`
}

//MeetingZoom : Zoom meeting
type MeetingZoom struct {
	ID             int                         `json:"id"`
	Topic          string                      `json:"topic"`
	Agenda         string                      `json:"agenda"`
	Type           int                         `json:"type"`
	Duration       int                         `json:"duration"`
	Start          string                      `json:"start_time"`
	TimeZone       string                      `json:"timezone"`
	Created        string                      `json:"created_at"`
	URLStart       string                      `json:"start_url"`
	URLJoin        string                      `json:"join_url"`
	Password       string                      `json:"password"`
	PasswordH323   string                      `json:"h323_password"`
	PMI            int                         `json:"pmi"`
	Occurrences    []*MeetingOccurrenceZoom    `json:"occurrences"`
	Recurrence     *MeetingRecurrenceZoom      `json:"recurrence"`
	Settings       *MeetingSettingsZoom        `json:"settings"`
	TrackingFields []*MeetingTrackingFieldZoom `json:"tracking_fields"`
}

//PayloadDeauthorizationZoom : Zoom deauthorization payload
type PayloadDeauthorizationZoom struct {
	ClientID            string `json:"client_id"`
	UserID              string `json:"user_id"`
	AccountID           string `json:"account_id"`
	DataRetention       string `json:"user_data_retention"`
	DeauthorizationTime string `json:"deauthorization_time"`
	Signature           string `json:"signature"`
}

//EventZoom : Zoom event
type EventZoom struct {
	Type    string                      `json:"event"`
	Payload *PayloadDeauthorizationZoom `json:"payload"`
}

//DeauthorizationZoom : Zoo deauthorization request
type DeauthorizationZoom struct {
	ClientID            string                      `json:"client_id"`
	UserID              string                      `json:"user_id"`
	AccountID           string                      `json:"account_id"`
	ComplianceCompleted bool                        `json:"compliance_completed"`
	Event               *PayloadDeauthorizationZoom `json:"deauthorization_event_received"`
}

//zoom timezones
var zoomTimeZones map[string]string = map[string]string{
	"Africa/Algiers":                 "Africa/Algiers",
	"Africa/Bangui":                  "Africa/Bangui",
	"Africa/Cairo":                   "Africa/Cairo",
	"Africa/Casablanca":              "Africa/Casablanca",
	"Africa/Djibouti":                "Africa/Djibouti",
	"Africa/Harare":                  "Africa/Harare",
	"Africa/Johannesburg":            "Africa/Johannesburg",
	"Africa/Khartoum":                "Africa/Khartoum",
	"Africa/Mogadishu":               "Africa/Mogadishu",
	"Africa/Nairobi":                 "Africa/Nairobi",
	"Africa/Nouakchott":              "Africa/Nouakchott",
	"Africa/Tripoli":                 "Africa/Tripoli",
	"Africa/Tunis":                   "Africa/Tunis",
	"America/Anchorage":              "America/Anchorage",
	"America/Araguaina":              "America/Araguaina",
	"America/Argentina/Buenos_Aires": "America/Argentina/Buenos_Aires",
	"America/Bogota":                 "America/Bogota",
	"America/Caracas":                "America/Caracas",
	"America/Chicago":                "America/Chicago",
	"America/Costa_Rica":             "America/Costa_Rica",
	"America/Denver":                 "America/Denver",
	"America/Edmonton":               "America/Edmonton",
	"America/El_Salvador":            "America/El_Salvador",
	"America/Godthab":                "America/Godthab",
	"America/Guatemala":              "America/Guatemala",
	"America/Halifax":                "America/Halifax",
	"America/Indianapolis":           "America/Indianapolis",
	"America/Lima":                   "America/Lima",
	"America/Los_Angeles":            "America/Los_Angeles",
	"America/Managua":                "America/Managua",
	"America/Mazatlan":               "America/Mazatlan",
	"America/Mexico_City":            "America/Mexico_City",
	"America/Montevideo":             "America/Montevideo",
	"America/Montreal":               "America/Montreal",
	"America/New_York":               "America/New_York",
	"America/Panama":                 "America/Panama",
	"America/Phoenix":                "America/Phoenix",
	"America/Puerto_Rico":            "America/Puerto_Rico",
	"America/Regina":                 "America/Regina",
	"America/Santiago":               "America/Santiago",
	"America/Sao_Paulo":              "America/Sao_Paulo",
	"America/St_Johns":               "America/St_Johns",
	"America/Tegucigalpa":            "America/Tegucigalpa",
	"America/Tijuana":                "America/Tijuana",
	"America/Vancouver":              "America/Vancouver",
	"America/Winnipeg":               "America/Winnipeg",
	"Asia/Aden":                      "Asia/Aden",
	"Asia/Almaty":                    "Asia/Almaty",
	"Asia/Amman":                     "Asia/Amman",
	"Asia/Baghdad":                   "Asia/Baghdad",
	"Asia/Bahrain":                   "Asia/Bahrain",
	"Asia/Baku":                      "Asia/Baku",
	"Asia/Bangkok":                   "Asia/Bangkok",
	"Asia/Beirut":                    "Asia/Beirut",
	"Asia/Calcutta":                  "Asia/Calcutta",
	"Asia/Dacca":                     "Asia/Dacca",
	"Asia/Damascus":                  "Asia/Damascus",
	"Asia/Dhaka":                     "Asia/Dhaka",
	"Asia/Dubai":                     "Asia/Dubai",
	"Asia/Hong_Kong":                 "Asia/Hong_Kong",
	"Asia/Irkutsk":                   "Asia/Irkutsk",
	"Asia/Jakarta":                   "Asia/Jakarta",
	"Asia/Jerusalem":                 "Asia/Jerusalem",
	"Asia/Kabul":                     "Asia/Kabul",
	"Asia/Kamchatka":                 "Asia/Kamchatka",
	"Asia/Kathmandu":                 "Asia/Kathmandu",
	"Asia/Kolkata":                   "Asia/Kolkata",
	"Asia/Krasnoyarsk":               "Asia/Krasnoyarsk",
	"Asia/Kuala_Lumpur":              "Asia/Kuala_Lumpur",
	"Asia/Kuwait":                    "Asia/Kuwait",
	"Asia/Magadan":                   "Asia/Magadan",
	"Asia/Muscat":                    "Asia/Muscat",
	"Asia/Nicosia":                   "Asia/Nicosia",
	"Asia/Novosibirsk":               "Asia/Novosibirsk",
	"Asia/Qatar":                     "Asia/Qatar",
	"Asia/Riyadh":                    "Asia/Riyadh",
	"Asia/Saigon":                    "Asia/Saigon",
	"Asia/Seoul":                     "Asia/Seoul",
	"Asia/Shanghai":                  "Asia/Shanghai",
	"Asia/Singapore":                 "Asia/Singapore",
	"Asia/Taipei":                    "Asia/Taipei",
	"Asia/Tashkent":                  "Asia/Tashkent",
	"Asia/Tehran":                    "Asia/Tehran",
	"Asia/Tokyo":                     "Asia/Tokyo",
	"Asia/Vladivostok":               "Asia/Vladivostok",
	"Asia/Yakutsk":                   "Asia/Yakutsk",
	"Asia/Yekaterinburg":             "Asia/Yekaterinburg",
	"Atlantic/Azores":                "Atlantic/Azores",
	"Atlantic/Cape_Verde":            "Atlantic/Cape_Verde",
	"Atlantic/Reykjavik":             "Atlantic/Reykjavik",
	"Australia/Adelaide":             "Australia/Adelaide",
	"Australia/Brisbane":             "Australia/Brisbane",
	"Australia/Darwin":               "Australia/Darwin",
	"Australia/Hobart":               "Australia/Hobart",
	"Australia/Perth":                "Australia/Perth",
	"Australia/Sydney":               "Australia/Sydney",
	"Canada/Atlantic":                "Canada/Atlantic",
	"CET":                            "CET",
	"Etc/Greenwich":                  "Etc/Greenwich",
	"Europe/Amsterdam":               "Europe/Amsterdam",
	"Europe/Athens":                  "Europe/Athens",
	"Europe/Belgrade":                "Europe/Belgrade",
	"Europe/Berlin":                  "Europe/Berlin",
	"Europe/Brussels":                "Europe/Brussels",
	"Europe/Bucharest":               "Europe/Bucharest",
	"Europe/Budapest":                "Europe/Budapest",
	"Europe/Copenhagen":              "Europe/Copenhagen",
	"Europe/Dublin":                  "Europe/Dublin",
	"Europe/Helsinki":                "Europe/Helsinki",
	"Europe/Istanbul":                "Europe/Istanbul",
	"Europe/Kiev":                    "Europe/Kiev",
	"Europe/Lisbon":                  "Europe/Lisbon",
	"Europe/London":                  "Europe/London",
	"Europe/Luxembourg":              "Europe/Luxembourg",
	"Europe/Madrid":                  "Europe/Madrid",
	"Europe/Moscow":                  "Europe/Moscow",
	"Europe/Oslo":                    "Europe/Oslo",
	"Europe/Paris":                   "Europe/Paris",
	"Europe/Prague":                  "Europe/Prague",
	"Europe/Rome":                    "Europe/Rome",
	"Europe/Sofia":                   "Europe/Sofia",
	"Europe/Stockholm":               "Europe/Stockholm",
	"Europe/Vienna":                  "Europe/Vienna",
	"Europe/Warsaw":                  "Europe/Warsaw",
	"Europe/Zurich":                  "Europe/Zurich",
	"Pacific/Apia":                   "Pacific/Apia",
	"Pacific/Auckland":               "Pacific/Auckland",
	"Pacific/Fiji":                   "Pacific/Fiji",
	"Pacific/Honolulu":               "Pacific/Honolulu",
	"Pacific/Midway":                 "Pacific/Midway",
	"Pacific/Noumea":                 "Pacific/Noumea",
	"Pacific/Pago_Pago":              "Pacific/Pago_Pago",
	"Pacific/Port_Moresby":           "Pacific/Port_Moresby",
	"SST":                            "SST",
	"UTC":                            "UTC",
}

//find the zoom timezone based on the incoming timezone
func findTimeZoneZoom(timeZone string) string {
	tz, ok := zoomTimeZones[timeZone]
	if !ok {
		return zoomTimeZones["UTC"]
	}
	return tz
}

//CreateOAuthURLZoom : create the Zoom URL used for OAuth
func CreateOAuthURLZoom(ctx context.Context, token string, redirectURL string) (string, error) {
	params := map[string]interface{}{
		"client_id":     GetZoomClientID(),
		"redirect_uri":  redirectURL,
		"response_type": "code",
		"state":         token,
	}
	url, err := CreateURLRel(ZoomURLOAuth, params)
	if err != nil {
		return "", errors.Wrap(err, "create url")
	}
	return url, nil
}

//create a zoom http client
func createClientZoom() *http.Client {
	client := &http.Client{
		Timeout: ZoomRequestTimeOut,
	}
	return client
}

//create the authorization header for retrieving an oauth token
func createAuthHeaderOAuth() string {
	key := fmt.Sprintf("%s:%s", GetZoomClientID(), GetZoomClientSecret())
	out := base64.StdEncoding.EncodeToString([]byte(key))
	header := fmt.Sprintf("Basic %s", out)
	return header
}

//make a call to the zoom oauth api
func makeRequestOAuthZoom(request *http.Request) (*http.Response, error) {
	//create the auth header
	header := createAuthHeaderOAuth()
	request.Header.Set("Authorization", header)

	//make the request
	client := createClientZoom()
	response, err := client.Do(request)
	if err != nil {
		return nil, errors.Wrap(err, "zoom token http request")
	}
	return response, nil
}

//create the authorization header based on a token
func createAuthHeader(ctx context.Context, token *TokenZoom) (*TokenZoom, string, error) {
	//make sure the token is up-to-date
	token, new, err := refreshTokenZoom(ctx, token)
	if err != nil {
		return nil, "", errors.Wrap(err, "zoom token refresh")
	}

	//create the auth header
	key, err := token.GetAccessToken()
	if err != nil {
		return nil, "", errors.Wrap(err, "zoom access token")
	}
	header := fmt.Sprintf("Bearer %s", key)

	//if the token is new, return the token
	if new {
		return token, header, nil
	}
	return nil, header, nil
}

//make a call to the zoom api
func makeRequestAPIZoom(ctx context.Context, token *TokenZoom, request *http.Request) (*TokenZoom, *http.Response, error) {
	//get the token for auth
	token, header, err := createAuthHeader(ctx, token)
	if err != nil {
		return nil, nil, errors.Wrap(err, "zoom auth header")
	}
	request.Header.Set("Authorization", header)

	//make the request
	client := createClientZoom()
	response, err := client.Do(request)
	if err != nil {
		return nil, nil, errors.Wrap(err, "zoom token http request")
	}
	return token, response, nil
}

//SignalDataComplianceZoom : signal data compliance with zoom
func SignalDataComplianceZoom(ctx context.Context, deauthPayload *PayloadDeauthorizationZoom) error {
	ctx, logger := GetLogger(ctx)
	start := time.Now()
	defer func() {
		logger.Debugw("zoom data compliance", "elapsedMS", FormatElapsedMS(start))
		AddCtxStatsAPI(ctx, ServerStatAPIZoom, "zoom data compliance", time.Since(start))
	}()

	//signal a successful deauthorization
	deauth := DeauthorizationZoom{
		ClientID:            deauthPayload.ClientID,
		UserID:              deauthPayload.UserID,
		AccountID:           deauthPayload.AccountID,
		ComplianceCompleted: true,
		Event:               deauthPayload,
	}
	deauthData, err := json.Marshal(deauth)
	if err != nil {
		return errors.Wrap(err, "zoom deauth json")
	}

	//make the request
	request, err := http.NewRequest("POST", ZoomURLDataCompliance, bytes.NewBuffer(deauthData))
	if err != nil {
		return errors.Wrap(err, "zoom deauthorization http request")
	}
	request.Header.Set(HeaderContentType, "application/json")
	response, err := makeRequestOAuthZoom(request)
	if err != nil {
		return errors.Wrap(err, "zoom deauthorization http")
	}
	if response.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(response.Body)
		return fmt.Errorf("status: %d: %s", response.StatusCode, body)
	}
	defer response.Body.Close()
	return nil
}

//RetrieveOAuthTokenZoom : retrieve the Zoom OAuth token based on the code
func RetrieveOAuthTokenZoom(ctx context.Context, code string, redirectURL string) (*TokenZoom, error) {
	ctx, logger := GetLogger(ctx)
	start := time.Now()
	defer func() {
		logger.Debugw("zoom oauth token", "elapsedMS", FormatElapsedMS(start))
		AddCtxStatsAPI(ctx, ServerStatAPIZoom, "zoom oauth token", time.Since(start))
	}()

	//request the token
	url := fmt.Sprintf(ZoomURLOAuthToken, code, redirectURL)
	request, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "zoom token http request")
	}
	response, err := makeRequestOAuthZoom(request)
	if err != nil {
		return nil, errors.Wrap(err, "zoom token http")
	}
	if response.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(response.Body)
		return nil, fmt.Errorf("status: %d: %s", response.StatusCode, body)
	}
	defer response.Body.Close()

	//decode the data
	var token TokenZoom
	err = json.NewDecoder(response.Body).Decode(&token)
	if err != nil {
		return nil, errors.Wrap(err, "zoom token json decode")
	}

	//compute the expiration, including the padding
	token.Expiration = time.Now().Unix() + token.ExpiresIn - ZoomAccessTokenExpirationPadding

	//encrypt the keys
	err = token.EncryptKeys()
	if err != nil {
		return nil, errors.Wrap(err, "zoom token encrypt keys")
	}
	return &token, nil
}

//RefreshAccessTokenZoom : refresh the Zoom access token
func RefreshAccessTokenZoom(ctx context.Context, token *TokenZoom) (*TokenZoom, error) {
	ctx, logger := GetLogger(ctx)
	start := time.Now()
	defer func() {
		logger.Debugw("zoom refresh token", "elapsedMS", FormatElapsedMS(start))
		AddCtxStatsAPI(ctx, ServerStatAPIZoom, "zoom refresh token", time.Since(start))
	}()

	//read the refresh token
	refreshToken, err := token.GetRefreshToken()
	if err != nil {
		return nil, errors.Wrap(err, "zoom refresh token")
	}

	//request the token
	url := fmt.Sprintf(ZoomURLOAuthAccessToken, refreshToken)
	request, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "zoom token http request")
	}
	response, err := makeRequestOAuthZoom(request)
	if err != nil {
		return nil, errors.Wrap(err, "zoom token http")
	}
	if response.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(response.Body)
		return nil, fmt.Errorf("status: %d: %s", response.StatusCode, body)
	}
	defer response.Body.Close()

	//decode the data
	var responseToken TokenZoom
	err = json.NewDecoder(response.Body).Decode(&responseToken)
	if err != nil {
		return nil, errors.Wrap(err, "zoom token json decode")
	}

	//compute the expiration, including the padding
	responseToken.Expiration = time.Now().Unix() + responseToken.ExpiresIn - ZoomAccessTokenExpirationPadding

	//encrypt the keys
	err = responseToken.EncryptKeys()
	if err != nil {
		return nil, errors.Wrap(err, "zoom token encrypt keys")
	}
	return &responseToken, nil
}

//validate and refresh token
func refreshTokenZoom(ctx context.Context, token *TokenZoom) (*TokenZoom, bool, error) {
	//check for a valid token
	if time.Now().Unix() < token.Expiration {
		return token, false, nil
	}

	//refresh the token
	token, err := RefreshAccessTokenZoom(ctx, token)
	if err != nil {
		return nil, false, errors.Wrap(err, "zoom token refresh")
	}
	return token, true, nil
}

//GetUserZoom : retrieve the Zoom user
func GetUserZoom(ctx context.Context, token *TokenZoom) (*TokenZoom, *UserZoom, error) {
	ctx, logger := GetLogger(ctx)
	start := time.Now()
	defer func() {
		logger.Debugw("zoom get user", "elapsedMS", FormatElapsedMS(start))
		AddCtxStatsAPI(ctx, ServerStatAPIZoom, "zoom get user", time.Since(start))
	}()

	//make the request
	request, err := http.NewRequest("GET", ZoomURLUser, nil)
	if err != nil {
		return nil, nil, errors.Wrap(err, "zoom user http request")
	}
	tokenRefresh, response, err := makeRequestAPIZoom(ctx, token, request)
	if err != nil {
		return nil, nil, errors.Wrap(err, "zoom user http")
	}
	if response.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(response.Body)
		return nil, nil, fmt.Errorf("status: %d: %s", response.StatusCode, body)
	}
	defer response.Body.Close()

	//decode the data
	var responseUser UserZoom
	err = json.NewDecoder(response.Body).Decode(&responseUser)
	if err != nil {
		return nil, nil, errors.Wrap(err, "zoom meeting json decode")
	}
	return tokenRefresh, &responseUser, nil
}

//CreateMeetingZoom : create a Zoom meeting
func CreateMeetingZoom(ctx context.Context, token *TokenZoom, bookingID string, topic string, agenda string, meetingStart time.Time, meetingTimeZone string, duration int, recurrence RecurrenceInterval) (*TokenZoom, *MeetingZoom, error) {
	ctx, logger := GetLogger(ctx)
	start := time.Now()
	defer func() {
		logger.Debugw("zoom create meeting", "elapsedMS", FormatElapsedMS(start))
		AddCtxStatsAPI(ctx, ServerStatAPIZoom, "zoom create meeting", time.Since(start))
	}()

	//create the meeting
	meeting := &MeetingInputZoom{
		Type:     2,
		Topic:    topic,
		Agenda:   agenda,
		Start:    meetingStart.UTC().Format(time.RFC3339),
		TimeZone: findTimeZoneZoom(meetingTimeZone),
		Duration: duration,
		Settings: &MeetingSettingsZoom{
			HostVideo:      true,
			JoinBeforeHost: true,
			WaitingRoom:    true,
		},
		TrackingFields: []*MeetingTrackingFieldZoom{
			{
				Field: URLParams.BookID,
				Value: bookingID,
			},
		},
	}

	//set-up the recurrence
	switch recurrence {
	case RecurrenceIntervalOnce:
	case RecurrenceIntervalDaily:
		meetingRecurrence := &MeetingRecurrenceZoom{}
		meetingRecurrence.Type = 1
		meeting.Recurrence = meetingRecurrence
	case RecurrenceIntervalWeekly:
		meetingRecurrence := &MeetingRecurrenceZoom{}
		meetingRecurrence.Type = 2
		meetingRecurrence.WeeklyDays = strconv.Itoa(int(meetingStart.Weekday()) + 1)
		meeting.Recurrence = meetingRecurrence
	case RecurrenceIntervalEveryTwoWeeks:
		meetingRecurrence := &MeetingRecurrenceZoom{}
		meetingRecurrence.Type = 2
		meetingRecurrence.RepeatInterval = 2
		meetingRecurrence.WeeklyDays = strconv.Itoa(int(meetingStart.Weekday()) + 1)
		meeting.Recurrence = meetingRecurrence
	case RecurrenceIntervalMonthly:
		meetingRecurrence := &MeetingRecurrenceZoom{}
		meetingRecurrence.Type = 3
		recurrenceDay := FindRecurrenceRuleByDay(meetingStart)
		meetingRecurrence.MonthlyWeek = recurrenceDay.Offset
		meetingRecurrence.MonthlyWeekDay = int(recurrenceDay.Weekday) + 1
		meeting.Recurrence = meetingRecurrence
	}
	meetingData, err := json.Marshal(meeting)
	if err != nil {
		return nil, nil, errors.Wrap(err, "zoom meeting json")
	}

	//make the request
	request, err := http.NewRequest("POST", ZoomURLMeeting, bytes.NewBuffer(meetingData))
	if err != nil {
		return nil, nil, errors.Wrap(err, "zoom meeting http request")
	}
	request.Header.Set(HeaderContentType, "application/json")
	tokenRefresh, response, err := makeRequestAPIZoom(ctx, token, request)
	if err != nil {
		return nil, nil, errors.Wrap(err, "zoom meeting http")
	}
	if response.StatusCode != http.StatusCreated {
		body, _ := ioutil.ReadAll(response.Body)
		return nil, nil, fmt.Errorf("status: %d: %s", response.StatusCode, body)
	}
	defer response.Body.Close()

	//decode the data
	var responseMeeting MeetingZoom
	err = json.NewDecoder(response.Body).Decode(&responseMeeting)
	if err != nil {
		return nil, nil, errors.Wrap(err, "zoom meeting json decode")
	}
	return tokenRefresh, &responseMeeting, nil
}

//GetMeetingZoom : get a Zoom meeting
func GetMeetingZoom(ctx context.Context, token *TokenZoom, meetingID *string) (*TokenZoom, *MeetingZoom, error) {
	ctx, logger := GetLogger(ctx)
	start := time.Now()
	defer func() {
		logger.Debugw("zoom get meeting", "elapsedMS", FormatElapsedMS(start))
		AddCtxStatsAPI(ctx, ServerStatAPIZoom, "zoom get meeting", time.Since(start))
	}()
	if meetingID == nil {
		return nil, nil, fmt.Errorf("null meeting id")
	}

	//make the request
	url := fmt.Sprintf(ZoomURLMeetingModify, *meetingID)
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, nil, errors.Wrap(err, "zoom meeting http request")
	}
	tokenRefresh, response, err := makeRequestAPIZoom(ctx, token, request)
	if err != nil {
		return nil, nil, errors.Wrap(err, "zoom meeting http")
	}
	if response.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(response.Body)
		return nil, nil, fmt.Errorf("status: %d: %s", response.StatusCode, body)
	}
	defer response.Body.Close()

	//decode the data
	var responseMeeting MeetingZoom
	err = json.NewDecoder(response.Body).Decode(&responseMeeting)
	if err != nil {
		return nil, nil, errors.Wrap(err, "zoom meeting json decode")
	}
	return tokenRefresh, &responseMeeting, nil
}

//UpdateMeetingZoom : update a Zoom meeting
func UpdateMeetingZoom(ctx context.Context, token *TokenZoom, meetingID *string, bookingID string, topic string, agenda string, meetingStart time.Time, meetingTimeZone string, duration int, recurrence RecurrenceInterval) (*TokenZoom, *MeetingZoom, error) {
	ctx, logger := GetLogger(ctx)
	start := time.Now()
	defer func() {
		logger.Debugw("zoom update meeting", "elapsedMS", FormatElapsedMS(start))
		AddCtxStatsAPI(ctx, ServerStatAPIZoom, "zoom update meeting", time.Since(start))
	}()
	if meetingID == nil {
		return nil, nil, fmt.Errorf("null meeting id")
	}

	//create the meeting
	meeting := &MeetingInputZoom{
		Type:     2,
		Topic:    topic,
		Agenda:   agenda,
		Start:    meetingStart.UTC().Format(time.RFC3339),
		TimeZone: findTimeZoneZoom(meetingTimeZone),
		Duration: duration,
		Settings: &MeetingSettingsZoom{
			HostVideo:      true,
			JoinBeforeHost: true,
			WaitingRoom:    true,
		},
		TrackingFields: []*MeetingTrackingFieldZoom{
			{
				Field: URLParams.BookID,
				Value: bookingID,
			},
		},
	}

	//set-up the recurrence
	switch recurrence {
	case RecurrenceIntervalOnce:
	case RecurrenceIntervalDaily:
		meetingRecurrence := &MeetingRecurrenceZoom{}
		meetingRecurrence.Type = 1
		meeting.Recurrence = meetingRecurrence
	case RecurrenceIntervalWeekly:
		meetingRecurrence := &MeetingRecurrenceZoom{}
		meetingRecurrence.Type = 2
		meetingRecurrence.WeeklyDays = strconv.Itoa(int(meetingStart.Weekday()) + 1)
		meeting.Recurrence = meetingRecurrence
	case RecurrenceIntervalEveryTwoWeeks:
		meetingRecurrence := &MeetingRecurrenceZoom{}
		meetingRecurrence.Type = 2
		meetingRecurrence.RepeatInterval = 2
		meetingRecurrence.WeeklyDays = strconv.Itoa(int(meetingStart.Weekday()) + 1)
		meeting.Recurrence = meetingRecurrence
	case RecurrenceIntervalMonthly:
		meetingRecurrence := &MeetingRecurrenceZoom{}
		meetingRecurrence.Type = 3
		recurrenceDay := FindRecurrenceRuleByDay(meetingStart)
		meetingRecurrence.MonthlyWeek = recurrenceDay.Offset
		meetingRecurrence.MonthlyWeekDay = int(recurrenceDay.Weekday) + 1
		meeting.Recurrence = meetingRecurrence
	}
	meetingData, err := json.Marshal(meeting)
	if err != nil {
		return nil, nil, errors.Wrap(err, "zoom meeting json")
	}

	//make the request
	url := fmt.Sprintf(ZoomURLMeetingModify, *meetingID)
	request, err := http.NewRequest("PATCH", url, bytes.NewBuffer(meetingData))
	if err != nil {
		return nil, nil, errors.Wrap(err, "zoom meeting http request")
	}
	request.Header.Set(HeaderContentType, "application/json")
	tokenRefresh, response, err := makeRequestAPIZoom(ctx, token, request)
	if err != nil {
		return nil, nil, errors.Wrap(err, "zoom meeting http")
	}
	if response.StatusCode != http.StatusNoContent {
		body, _ := ioutil.ReadAll(response.Body)
		return nil, nil, fmt.Errorf("status: %d: %s", response.StatusCode, body)
	}
	defer response.Body.Close()

	//retrieve the meeting
	if tokenRefresh != nil {
		token = tokenRefresh
	}
	tokenRefresh, responseMeeting, err := GetMeetingZoom(ctx, token, meetingID)
	if err != nil {
		return nil, nil, errors.Wrap(err, "zoom meeting get")
	}
	return tokenRefresh, responseMeeting, nil
}

//DeleteMeetingZoom : delete a Zoom meeting
func DeleteMeetingZoom(ctx context.Context, token *TokenZoom, meetingID *string) (*TokenZoom, error) {
	ctx, logger := GetLogger(ctx)
	start := time.Now()
	defer func() {
		logger.Debugw("zoom delete meeting", "elapsedMS", FormatElapsedMS(start))
		AddCtxStatsAPI(ctx, ServerStatAPIZoom, "zoom delete meeting", time.Since(start))
	}()
	if meetingID == nil {
		return nil, fmt.Errorf("null meeting id")
	}

	//delete the meeting
	url := fmt.Sprintf(ZoomURLMeetingModify, *meetingID)
	request, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "zoom meeting http request")
	}
	tokenRefresh, response, err := makeRequestAPIZoom(ctx, token, request)
	if err != nil {
		return nil, errors.Wrap(err, "zoom meeting http")
	}
	if response.StatusCode != http.StatusNoContent {
		body, _ := ioutil.ReadAll(response.Body)
		return nil, fmt.Errorf("status: %d: %s", response.StatusCode, body)
	}
	return tokenRefresh, nil
}

//VerifyWebHookZoom : verify the payload for a Zoom webhook event
func VerifyWebHookZoom(w http.ResponseWriter, r *http.Request) (*EventZoom, error) {
	ctx, logger := GetLogger(r.Context())
	start := time.Now()
	defer func() {
		logger.Debugw("zoom verify webhook", "elapsedMS", FormatElapsedMS(start))
		AddCtxStatsAPI(ctx, ServerStatAPIZoom, "zoom verify webhook", time.Since(start))
	}()

	//read the body
	const MaxBodyBytes = int64(65536)
	r.Body = http.MaxBytesReader(w, r.Body, int64(65536))
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, errors.Wrap(err, "zoom read body")
	}

	//verify the header
	token := r.Header.Get(ZoomHeaderAuth)
	if token != GetZoomVerificationToken() {
		return nil, fmt.Errorf("zoom verification token")
	}

	//read the event
	var event EventZoom
	err = json.Unmarshal(body, &event)
	if err != nil {
		return nil, errors.Wrap(err, "unjson zoom event")
	}
	return &event, nil
}

//SaveEventZoom : save a Zoom event
func SaveEventZoom(ctx context.Context, db *DB, event *EventZoom) (context.Context, error) {
	//json encode the message data
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return ctx, errors.Wrap(err, "json event")
	}

	//save to the db
	stmt := fmt.Sprintf("INSERT INTO %s(id,data) VALUES (?,?)", dbTableEventZoom)
	ctx, result, err := db.Exec(ctx, stmt, event.Payload.UserID, eventJSON)
	if err != nil {
		return ctx, errors.Wrap(err, "insert event zoom")
	}
	count, err := result.RowsAffected()
	if err != nil {
		return ctx, errors.Wrap(err, "insert event zoom rows affected")
	}
	if count == 0 {
		return ctx, fmt.Errorf("unable to insert event zoom: %s", event.Payload.UserID)
	}
	return ctx, nil
}
