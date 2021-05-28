package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

//paths
const (
	BaseWebAssetPath                 = "web/asset"
	BaseWebTemplatePathClient        = "web/template/client"
	BaseWebTemplatePathDashboard     = "web/template/dashboard"
	BaseWebTemplatePathProvider      = "web/template/provider"
	BaseWebTemplatePathLanding       = "landing"
	BaseWebTemplatePathLandingTutors = "tutors"
)

//http constants
const (
	HeaderAPIToken      = "X-HR-Token"
	HeaderCacheControl  = "Cache-Control"
	HeaderContentType   = "Content-Type"
	HeaderForwardedHost = "X-Forwarded-Host"
	HeaderRequestID     = "X-Request-Id"
)

//default images
const (
	ImgDefaultProviderBanner  = "/dashboard/img/defaultbanner.png"
	ImgDefaultProviderFavIcon = "/dashboard/img/favicon.ico"
	ImgDefaultProviderLogo    = "/dashboard/img/defaultlogo.jpg"
	ImgDefaultService         = "/dashboard/img/defaultservice.png"
)

//UploadAssetPathLocal : local path to uploaded files
const UploadAssetPathLocal = "web/asset/upload"

//URLAssetUpload : url for uploaded files
const URLAssetUpload = "/upload"

//http paths
const (
	URIAccount              = "/my-account.html"
	URIAbout                = "/about.html"
	URIAddOns               = "/add-ons.html"
	URIAssets               = "/asset"
	URIBooking              = "/order1.html"
	URIBookingAdd           = "/add-order.html"
	URIBookingAddSuccess    = "/add-order-success.html"
	URIBookingCancel        = "/cancel"
	URIBookingCancelSuccess = "/cancel-order-success.html"
	URIBookingSubmit        = "/order3.html"
	URIBookingConfirm       = "/order4.html"
	URIBookingEdit          = "/edit-order.html"
	URIBookingEditSuccess   = "/edit-order-success.html"
	URIBookingView          = "/view-order.html"
	URIBookings             = "/orders.html"
	URICalendars            = "/calendar.html"
	URICallback             = "/callback"
	URICallbackCal          = "/callback-calendar"
	URICampaignAddStep1     = "/add-campaign-step1.html"
	URICampaignAddStep2     = "/add-campaign-step2.html"
	URICampaignAddStep3     = "/add-campaign-step3.html"
	URICampaignManage       = "/manage-campaign.html"
	URICampaignView         = "/view-campaign.html"
	URICampaigns            = "/campaigns.html"
	URICancel               = "/cancel"
	URIClientAdd            = "/add-client.html"
	URIClientEdit           = "/edit-client.html"
	URIClients              = "/clients.html"
	URIContact              = "/contact.html"
	URIContent              = "/content"
	URICouponAdd            = "/add-coupons.html"
	URICouponEdit           = "/edit-coupons.html"
	URICoupons              = "/coupons.html"
	URIDefault              = "/"
	URIEmail                = "/email"
	URIEmailVerify          = "/emailverify"
	URIFaq                  = "/faq.html"
	URIFaqAdd               = "/add-faq.html"
	URIFaqEdit              = "/edit-faq.html"
	URIFaqs                 = "/faqs.html"
	URIErr                  = "/error.html"
	URIForgotPwd            = "/forgotpwd.html"
	URIGoogle               = "/google"
	URIHours                = "/service-hours.html"
	URIHowItWorks           = "/how-it-works.html"
	URIHowTo                = "/how-to.html"
	URIIndex                = "/index.html"
	URILinks                = "/links.html"
	URILogin                = "/login.html"
	URILogout               = "/logout.html"
	URIMaintenance          = "/maintenance"
	URIOAuthLogin           = "/login"
	URIOrdersCalendar       = "/orders/calendar"
	URIPayment              = "/payment.html"
	URIPaymentView          = "/view-payment.html"
	URIPayPal               = "/paypal"
	URIPaymentDirect        = "/payment-direct.html"
	URIPaymentSettings      = "/payment-settings.html"
	URIPayments             = "/invoices.html"
	URIPolicy               = "/policy.html"
	URIProfile              = "/profile.html"
	URIProfileDomain        = "/profile-domain.html"
	URIProviders            = "/providers.html"
	URIPwdReset             = "/pwdreset.html"
	URIReport               = "/report"
	URIRobots               = "/robots.txt"
	URIServiceAreas         = "/service-areas.html"
	URISignUp               = "/signup.html"
	URISignUpLink           = "/signup-link.html"
	URISignUpPricing        = "/pricing.html"
	URISignUpSuccess        = "/signup-success.html"
	URISignUpMain           = "/signup-main.html"
	URISiteMap              = "/sitemap.xml"
	URIStats                = "/stats"
	URIStripe               = "/stripe"
	URISupport              = "/support.html"
	URISvcAdd               = "/add-service.html"
	URISvcEdit              = "/edit-service.html"
	URISvcUsers             = "/service-members.html"
	URISvcs                 = "/services.html"
	URITestimonialAdd       = "/add-testimonial.html"
	URITestimonialEdit      = "/edit-testimonial.html"
	URITestimonials         = "/testimonials.html"
	URITerms                = "/terms.html"
	URITutors               = "/tutors"
	URIUserAdd              = "/add-member.html"
	URIUserEdit             = "/edit-member.html"
	URIUsers                = "/members.html"
	URIZoom                 = "/zoom"
	URIZoomSupport          = "/zoom-support.html"
)

//url parameters type
type urlParams struct {
	AgeMin                  string
	AgeMax                  string
	ApptOnly                string
	AuthToken               string
	Bio                     string
	BookID                  string
	Budget                  string
	CampaignID              string
	CheckedMon              string
	CheckedTue              string
	CheckedWed              string
	CheckedThu              string
	CheckedFri              string
	CheckedSat              string
	CheckedSun              string
	City                    string
	Client                  string
	ClientID                string
	Code                    string
	Data                    string
	Date                    string
	Desc                    string
	DisablePhone            string
	Domain                  string
	Duration                string
	Education               string
	Email                   string
	EnablePhone             string
	EnableZoom              string
	End                     string
	Error                   string
	ErrorDesc               string
	Experience              string
	ExternalID              string
	Filter                  string
	FilterSub               string
	FirstName               string
	Flag                    string
	Freq                    string
	Gender                  string
	GoogleRecaptchaResponse string
	HasFacebookAdAccount    string
	HasFacebookPage         string
	ID                      string
	Img                     string
	ImgBanner               string
	ImgDel                  string
	ImgDelBanner            string
	ImgDelLogo              string
	ImgIdx                  string
	ImgLogo                 string
	Interests               string
	Interval                string
	LastName                string
	Location                string
	LocationType            string
	Locations               string
	MsgKey                  string
	Name                    string
	Next                    string
	Note                    string
	OAuth                   string
	Padding                 string
	PaddingInitial          string
	PaddingInitialUnit      string
	Password                string
	PayPalID                string
	PaymentID               string
	Phone                   string
	Prev                    string
	Price                   string
	PriceType               string
	ProviderID              string
	ProviderName            string
	ProviderURLName         string
	Schedule                string
	ScheduleDuration        string
	Start                   string
	State                   string
	Status                  string
	Step                    string
	StripeID                string
	SvcArea                 string
	SvcDesc                 string
	SvcID                   string
	SvcName                 string
	Text                    string
	Time                    string
	TimeZone                string
	Token                   string
	Type                    string
	URLFacebook             string
	URLInstagram            string
	URLLinkedIn             string
	URLName                 string
	URLShort                string
	URLTwitter              string
	URLVideo                string
	URLWeb                  string
	UserID                  string
	Value                   string
	Version                 string
}

//URLParams : url parameters, which are also used in the templates
var URLParams urlParams = urlParams{
	AgeMin:                  "ageMin",
	AgeMax:                  "ageMax",
	ApptOnly:                "apptOnly",
	AuthToken:               "authToken",
	Bio:                     "bio",
	BookID:                  "bookId",
	Budget:                  "budget",
	CampaignID:              "campaignId",
	CheckedMon:              "checkedMon",
	CheckedTue:              "checkedTue",
	CheckedWed:              "checkedWed",
	CheckedThu:              "checkedThu",
	CheckedFri:              "checkedFri",
	CheckedSat:              "checkedSat",
	CheckedSun:              "checkedSun",
	City:                    "city",
	Client:                  "client",
	ClientID:                "clientId",
	Code:                    "code",
	Data:                    "data",
	Date:                    "date",
	Desc:                    "desc",
	DisablePhone:            "disablePhone",
	Domain:                  "domain",
	Duration:                "duration",
	Education:               "education",
	Email:                   "email",
	EnablePhone:             "enablePhone",
	EnableZoom:              "enableZoom",
	End:                     "end",
	Error:                   "error",
	ErrorDesc:               "error_description",
	Experience:              "experience",
	ExternalID:              "externalId",
	Filter:                  "filter",
	FilterSub:               "filterSub",
	FirstName:               "firstName",
	Flag:                    "flag",
	Freq:                    "freq",
	Gender:                  "gender",
	GoogleRecaptchaResponse: "g-recaptcha-response",
	HasFacebookAdAccount:    "hasFacebookAdAccount",
	HasFacebookPage:         "hasFacebookPage",
	ID:                      "id",
	Img:                     "img",
	ImgBanner:               "imgBanner",
	ImgDel:                  "imgDel",
	ImgDelBanner:            "imgDelBanner",
	ImgDelLogo:              "imgDelLogo",
	ImgIdx:                  "imgIdx",
	ImgLogo:                 "imgLogo",
	Interests:               "interests",
	Interval:                "interval",
	LastName:                "lastName",
	Location:                "location",
	LocationType:            "locationType",
	Locations:               "locations",
	MsgKey:                  "msgKey",
	Name:                    "name",
	Next:                    "next",
	Note:                    "note",
	OAuth:                   "oauth",
	Padding:                 "padding",
	PaddingInitial:          "paddingInitial",
	PaddingInitialUnit:      "paddingInitialUnit",
	Password:                "password",
	PayPalID:                "paypalId",
	PaymentID:               "paymentId",
	Phone:                   "phone",
	Prev:                    "prev",
	Price:                   "price",
	PriceType:               "priceType",
	ProviderID:              "providerId",
	ProviderName:            "providerName",
	ProviderURLName:         "providerUrlName",
	Schedule:                "schedule",
	ScheduleDuration:        "scheduleDuration",
	State:                   "state",
	Status:                  "status",
	Start:                   "start",
	Step:                    "step",
	StripeID:                "stripeId",
	SvcArea:                 "svcArea",
	SvcDesc:                 "svcDesc",
	SvcID:                   "svcId",
	SvcName:                 "svcName",
	Text:                    "text",
	Time:                    "time",
	TimeZone:                "timeZone",
	Token:                   "token",
	Type:                    "type",
	URLFacebook:             "urlFacebook",
	URLInstagram:            "urlInstagram",
	URLLinkedIn:             "urlLinkedIn",
	URLName:                 "urlName",
	URLShort:                "urlShort",
	URLTwitter:              "urlTwitter",
	URLVideo:                "urlVideo",
	URLWeb:                  "urlWeb",
	UserID:                  "userId",
	Value:                   "value",
	Version:                 "v",
}

//template types used in the html template
type (
	templateDataKey string
	templateData    map[templateDataKey]interface{}
)

//CreateMap : create a map keyed by a string, which is required for use in templates
func (t templateData) CreateMap() map[string]interface{} {
	m := make(map[string]interface{})
	for k, v := range t {
		m[string(k)] = v
	}
	return m
}

//template parameters
const (
	TplParamAbout                  templateDataKey = "About"
	TplParamActiveNav              templateDataKey = "ActiveNav"
	TplParamAgeMin                 templateDataKey = "AgeMin"
	TplParamAgeMax                 templateDataKey = "AgeMax"
	TplParamAlert                  templateDataKey = "Alert"
	TplParamApptOnly               templateDataKey = "ApptOnly"
	TplParamBio                    templateDataKey = "Bio"
	TplParamBook                   templateDataKey = "Book"
	TplParamBooks                  templateDataKey = "Books"
	TplParamBreadcrumbs            templateDataKey = "Breadcrumbs"
	TplParamBudget                 templateDataKey = "Budget"
	TplParamCampaign               templateDataKey = "Campaign"
	TplParamCampaigns              templateDataKey = "Campaigns"
	TplParamCheckedMon             templateDataKey = "CheckedMon"
	TplParamCheckedTue             templateDataKey = "CheckedTue"
	TplParamCheckedWed             templateDataKey = "CheckedWed"
	TplParamCheckedThu             templateDataKey = "CheckedThu"
	TplParamCheckedFri             templateDataKey = "CheckedFri"
	TplParamCheckedSat             templateDataKey = "CheckedSat"
	TplParamCheckedSun             templateDataKey = "CheckedSun"
	TplParamCity                   templateDataKey = "City"
	TplParamClientID               templateDataKey = "ClientId"
	TplParamClient                 templateDataKey = "Client"
	TplParamClientView             templateDataKey = "ClientView"
	TplParamClients                templateDataKey = "Clients"
	TplParamCode                   templateDataKey = "Code"
	TplParamConfirm                templateDataKey = "Confirm"
	TplParamConfirmMsg             templateDataKey = "ConfirmMsg"
	TplParamConfirmSubmitName      templateDataKey = "ConfirmSubmitName"
	TplParamConfirmSubmitValue     templateDataKey = "ConfirmSubmitValue"
	TplParamConstants              templateDataKey = "Constants"
	TplParamContext                templateDataKey = "Ctx"
	TplParamCookieFlag             templateDataKey = "CookieFlag"
	TplParamCount                  templateDataKey = "Count"
	TplParamCountNew               templateDataKey = "CountNew"
	TplParamCountUnPaid            templateDataKey = "CountUnPaid"
	TplParamCountUpcoming          templateDataKey = "CountUpcoming"
	TplParamCouponTypes            templateDataKey = "CouponTypes"
	TplParamCoupon                 templateDataKey = "Coupon"
	TplParamCoupons                templateDataKey = "Coupons"
	TplParamCurrentTime            templateDataKey = "CurrentTime"
	TplParamCurrentYear            templateDataKey = "CurrentYear"
	TplParamDate                   templateDataKey = "Date"
	TplParamDaysOfWeek             templateDataKey = "DaysOfWeek"
	TplParamDesc                   templateDataKey = "Desc"
	TplParamDevModeEnable          templateDataKey = "DevModeEnable"
	TplParamDisableAuth            templateDataKey = "DisableAuth"
	TplParamDisableNav             templateDataKey = "DisableNav"
	TplParamDisablePhone           templateDataKey = "DisablePhone"
	TplParamDomain                 templateDataKey = "Domain"
	TplParamDomainPublic           templateDataKey = "DomainPublic"
	TplParamDuration               templateDataKey = "Duration"
	TplParamDurationsBooking       templateDataKey = "DurationsBooking"
	TplParamDurationsOrder         templateDataKey = "DurationsOrder"
	TplParamEducation              templateDataKey = "Education"
	TplParamEmail                  templateDataKey = "Email"
	TplParamEmailDefault           templateDataKey = "EmailDefault"
	TplParamEnablePhone            templateDataKey = "EnablePhone"
	TplParamEnableZoom             templateDataKey = "EnableZoom"
	TplParamEnd                    templateDataKey = "End"
	TplParamErr                    templateDataKey = "Err"
	TplParamErrs                   templateDataKey = "Errs"
	TplParamExperience             templateDataKey = "Experience"
	TplParamFacebookAPIVersion     templateDataKey = "FacebookAPIVersion"
	TplParamFacebookAppID          templateDataKey = "FacebookAppId"
	TplParamFacebookConversionCost templateDataKey = "FacebookConversionCost"
	TplParamFacebookTrackingID     templateDataKey = "FacebookTrackingId"
	TplParamFaqCount               templateDataKey = "FaqCount"
	TplParamFaq                    templateDataKey = "Faq"
	TplParamFaqs                   templateDataKey = "Faqs"
	TplParamFileCSS                templateDataKey = "FileCss"
	TplParamFileJS                 templateDataKey = "FileJs"
	TplParamFilter                 templateDataKey = "Filter"
	TplParamFilterSub              templateDataKey = "FilterSub"
	TplParamFlag                   templateDataKey = "Flag"
	TplParamFormAction             templateDataKey = "FormAction"
	TplParamFormAction2            templateDataKey = "FormAction2"
	TplParamFreq                   templateDataKey = "Freq"
	TplParamGender                 templateDataKey = "Gender"
	TplParamGoogleRecaptchaSiteKey templateDataKey = "GoogleRecaptchaSiteKey"
	TplParamGoogleTagManagerID     templateDataKey = "GoogleTagManagerId"
	TplParamGoogleTrackingID       templateDataKey = "GoogleTrackingId"
	TplParamHasAccess              templateDataKey = "HasAccess"
	TplParamHasFacebookAdAccount   templateDataKey = "HasFacebookAdAccount"
	TplParamHasFacebookPage        templateDataKey = "HasFacebookPage"
	TplParamID                     templateDataKey = "Id"
	TplParamInputs                 templateDataKey = "Inputs"
	TplParamInterests              templateDataKey = "Interests"
	TplParamInterval               templateDataKey = "Interval"
	TplParamIPPublic               templateDataKey = "IpPublic"
	TplParamIsAdmin                templateDataKey = "IsAdmin"
	TplParamLocation               templateDataKey = "Location"
	TplParamLocationType           templateDataKey = "LocationType"
	TplParamLocations              templateDataKey = "Locations"
	TplParamMarquee                templateDataKey = "Marquee"
	TplParamMetaDesc               templateDataKey = "MetaDesc"
	TplParamMetaKeywords           templateDataKey = "MetaKeywords"
	TplParamMsg                    templateDataKey = "Msg"
	TplParamName                   templateDataKey = "Name"
	TplParamNameFirst              templateDataKey = "FirstName"
	TplParamNameLast               templateDataKey = "LastName"
	TplParamNavDisable             templateDataKey = "NavDisable"
	TplParamNote                   templateDataKey = "Note"
	TplParamPadding                templateDataKey = "Padding"
	TplParamPaddingInitial         templateDataKey = "PaddingInitial"
	TplParamPaddingInitialUnit     templateDataKey = "PaddingInitialUnit"
	TplParamPageTitle              templateDataKey = "PageTitle"
	TplParamPaddingUnits           templateDataKey = "PaddingUnits"
	TplParamPayment                templateDataKey = "Payment"
	TplParamPayments               templateDataKey = "Payments"
	TplParamPayPalClientID         templateDataKey = "PayPalClientId"
	TplParamPayPalOrderID          templateDataKey = "PayPalOrderId"
	TplParamPhone                  templateDataKey = "Phone"
	TplParamPlaidToken             templateDataKey = "PlaidToken"
	TplParamPrice                  templateDataKey = "Price"
	TplParamPriceType              templateDataKey = "PriceType"
	TplParamPriceTypes             templateDataKey = "PriceTypes"
	TplParamProvider               templateDataKey = "Provider"
	TplParamProviderID             templateDataKey = "ProviderId"
	TplParamProviders              templateDataKey = "Providers"
	TplParamProviderName           templateDataKey = "ProviderName"
	TplParamRecurrenceFreq         templateDataKey = "RecurrenceFreq"
	TplParamRecurrenceFreqs        templateDataKey = "RecurrenceFreqs"
	TplParamReport                 templateDataKey = "Report"
	TplParamSchedule               templateDataKey = "Schedule"
	TplParamSchedule1              templateDataKey = "Schedule1"
	TplParamSchedule2              templateDataKey = "Schedule2"
	TplParamScheduleDuration       templateDataKey = "ScheduleDuration"
	TplParamServiceAreas           templateDataKey = "ServiceAreas"
	TplParamServiceIntervals       templateDataKey = "ServiceIntervals"
	TplParamServiceLocations       templateDataKey = "ServiceLocations"
	TplParamShowHomeRun            templateDataKey = "ShowHomeRun"
	TplParamStart                  templateDataKey = "Start"
	TplParamStatus                 templateDataKey = "Status"
	TplParamStep                   templateDataKey = "Step"
	TplParamSteps                  templateDataKey = "Steps"
	TplParamStripeAccountID        templateDataKey = "StripeAccountId"
	TplParamStripePublicKey        templateDataKey = "StripePublicKey"
	TplParamStripeSessionID        templateDataKey = "StripeSessionId"
	TplParamSuccess                templateDataKey = "Success"
	TplParamSvc                    templateDataKey = "Svc"
	TplParamSvcArea                templateDataKey = "SvcArea"
	TplParamSvcAreaStrs            templateDataKey = "SvcAreaStrs"
	TplParamSvcBusyTimes           templateDataKey = "SvcBusyTimes"
	TplParamSvcDesc                templateDataKey = "SvcDesc"
	TplParamSvcID                  templateDataKey = "SvcId"
	TplParamSvcName                templateDataKey = "SvcName"
	TplParamSvcStartDate           templateDataKey = "SvcStartDate"
	TplParamSvcTime                templateDataKey = "SvcTime"
	TplParamSvcs                   templateDataKey = "Svcs"
	TplParamTestimonial            templateDataKey = "Testimonial"
	TplParamTestimonials           templateDataKey = "Testimonials"
	TplParamText                   templateDataKey = "Text"
	TplParamTime                   templateDataKey = "Time"
	TplParamTimeZone               templateDataKey = "TimeZone"
	TplParamTimeZones              templateDataKey = "TimeZones"
	TplParamTips                   templateDataKey = "Tips"
	TplParamTitleAlert             templateDataKey = "TitleAlert"
	TplParamToken                  templateDataKey = "Token"
	TplParamType                   templateDataKey = "Type"
	TplParamTypes                  templateDataKey = "Types"
	TplParamTypeSignUp             templateDataKey = "TypeSignUp"
	TplParamURL                    templateDataKey = "Url"
	TplParamURLAbout               templateDataKey = "UrlAbout"
	TplParamURLAssets              templateDataKey = "UrlAssets"
	TplParamURLCampaignAdd         templateDataKey = "UrlCampaignAdd"
	TplParamURLDashboard           templateDataKey = "UrlDashboard"
	TplParamURLDefault             templateDataKey = "UrlDefault"
	TplParamURLDefaultProvider     templateDataKey = "UrlDefaultProvider"
	TplParamURLEmailVerify         templateDataKey = "UrlEmailVerify"
	TplParamURLFacebook            templateDataKey = "UrlFacebook"
	TplParamURLFacebookProvider    templateDataKey = "UrlFacebookProvider"
	TplParamURLFaq                 templateDataKey = "UrlFaq"
	TplParamURLForgotPwd           templateDataKey = "UrlForgotPwd"
	TplParamURLHowItWorks          templateDataKey = "UrlHowItWorks"
	TplParamURLHowTo               templateDataKey = "UrlHowTo"
	TplParamURLInstagram           templateDataKey = "UrlInstagram"
	TplParamURLInstagramProvider   templateDataKey = "UrlInstagramProvider"
	TplParamURLLinkedIn            templateDataKey = "UrlLinkedIn"
	TplParamURLLinkedInProvider    templateDataKey = "UrlLinkedInProvider"
	TplParamURLLogin               templateDataKey = "UrlLogin"
	TplParamURLLogout              templateDataKey = "UrlLogout"
	TplParamURLName                templateDataKey = "UrlName"
	TplParamURLNext                templateDataKey = "UrlNext"
	TplParamURLPolicy              templateDataKey = "UrlPolicy"
	TplParamURLPrev                templateDataKey = "UrlPrev"
	TplParamURLProviders           templateDataKey = "UrlProviders"
	TplParamURLPwdReset            templateDataKey = "UrlPwdReset"
	TplParamURLSignUp              templateDataKey = "UrlSignUp"
	TplParamURLSignUpPricing       templateDataKey = "UrlSignUpPricing"
	TplParamURLSignUpSuccess       templateDataKey = "UrlSignUpSuccess"
	TplParamURLSupport             templateDataKey = "UrlSupport"
	TplParamSvcUsers               templateDataKey = "SvcUsers"
	TplParamURLTerms               templateDataKey = "UrlTerms"
	TplParamURLTestimonialAdd      templateDataKey = "UrlTestimonialAdd"
	TplParamURLTutors              templateDataKey = "UrlTutors"
	TplParamURLTwitter             templateDataKey = "UrlTwitter"
	TplParamURLTwitterProvider     templateDataKey = "UrlTwitterProvider"
	TplParamURLUploads             templateDataKey = "UrlUploads"
	TplParamURLUsers               templateDataKey = "UrlUsers"
	TplParamURLVideo               templateDataKey = "UrlVideo"
	TplParamURLView                templateDataKey = "UrlView"
	TplParamURLWebProvider         templateDataKey = "UrlWebProvider"
	TplParamURLYouTube             templateDataKey = "UrlYouTube"
	TplParamUserID                 templateDataKey = "UserId"
	TplParamUser                   templateDataKey = "User"
	TplParamUsers                  templateDataKey = "Users"
	TplParamValue                  templateDataKey = "Value"
	TplParamZelleID                templateDataKey = "ZelleId"
)

//CreateURLRel : create a relative URL to the specified location
func CreateURLRel(path string, params map[string]interface{}) (string, error) {
	//set-up the basic url
	newURL, err := url.Parse(path)
	if err != nil {
		return "", errors.Wrap(err, fmt.Sprintf("invalid url: %s", path))
	}

	//add any query parameters
	queryParams := newURL.Query()
	for k, v := range params {
		queryParams.Add(k, fmt.Sprintf("%v", v))
	}
	newURL.RawQuery = queryParams.Encode()
	return newURL.String(), nil
}

//CreateURLRelParams : create a relative URL with a single parameters
func CreateURLRelParams(path string, params ...interface{}) (string, error) {
	//sanity check the number of parameters, since an even number is required
	lenParams := len(params)
	if lenParams%2 != 0 {
		return "", fmt.Errorf("invalid number of parameters: %d", lenParams)
	}

	//read the key/value pairs
	count := lenParams / 2
	p := make(map[string]interface{}, count)
	for i := 0; i < count; i++ {
		k, ok := params[2*i].(string)
		if !ok {
			return "", fmt.Errorf("invalid key: %v", params[i])
		}
		p[k] = params[(2*i)+1]
	}
	return CreateURLRel(path, p)
}

//CreateURLAbs : create an absolute URL to the specified location
func CreateURLAbs(ctx context.Context, path string, params map[string]interface{}) (string, error) {
	newURL, err := url.Parse(path)
	if err != nil {
		return "", errors.Wrap(err, fmt.Sprintf("invalid url: %s", path))
	}

	//set-up the url if necessary
	if !newURL.IsAbs() {
		//determine the port to use
		newURL.Scheme = GetServerSchemePublic()
		port := GetServerAddressPublic()

		//probe for a host override
		host := GetDomain()
		if ctx != nil {
			hostCtx := GetCtxCustomHost(ctx)
			if hostCtx != "" {
				host = hostCtx
			}
		}

		//ignore passed-in ports or default ports
		if strings.Contains(host, ":") {
			port = ""
		} else if port == ":80" || port == ":443" {
			port = ""
		}
		newURL.Host = fmt.Sprintf("%s%s", host, port)
	}

	//add any query parameters
	queryParams := newURL.Query()
	for k, v := range params {
		queryParams.Add(k, fmt.Sprintf("%v", v))
	}
	newURL.RawQuery = queryParams.Encode()
	return newURL.String(), nil
}

//CreateURLAbsParams : create an absolute URL with a single parameters
func CreateURLAbsParams(ctx context.Context, path string, params ...string) (string, error) {
	//sanity check the number of parameters, since an even number is required
	lenParams := len(params)
	if lenParams%2 != 0 {
		return "", fmt.Errorf("invalid number of parameters: %d", lenParams)
	}

	//read the key/value pairs
	count := lenParams / 2
	p := make(map[string]interface{}, count)
	for i := 0; i < count; i++ {
		p[params[2*i]] = params[(2*i)+1]
	}
	return CreateURLAbs(ctx, path, p)
}

//ForceURLAbs : force a URL to be absolute
func ForceURLAbs(ctx context.Context, url string) string {
	newURL, err := CreateURLAbs(ctx, url, nil)
	if err != nil {
		_, logger := GetLogger(ctx)
		logger.Warnw("create url", "error", err)
		return ""
	}
	return newURL
}

//AddURLEmailProviderID : add email and provider id parameters to a URL
func AddURLEmailProviderID(url string, email string, providerID *uuid.UUID) string {
	if email == "" && providerID == nil {
		return url
	}
	newURL, err := CreateURLRelParams(url, URLParams.Email, email, URLParams.ProviderID, providerID)
	if err != nil {
		return ""
	}
	return newURL
}

//AddURLStep : add a step parameter to a URL
func AddURLStep(url string, val string) string {
	if val == "" {
		return url
	}
	newURL, err := CreateURLRelParams(url, URLParams.Step, val)
	if err != nil {
		return ""
	}
	return newURL
}

//AddURLType : add a type parameter to a URL
func AddURLType(url string, val string) string {
	if val == "" {
		return url
	}
	newURL, err := CreateURLRelParams(url, URLParams.Type, val)
	if err != nil {
		return ""
	}
	return newURL
}

//URLJoin : join paths to a URL
func URLJoin(baseURL string, paths ...string) string {
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		params := append([]string{baseURL}, paths...)
		return path.Join(params...)
	}
	params := append([]string{parsedURL.Path}, paths...)
	parsedURL.Path = path.Join(params...)
	return parsedURL.String()
}

//FileUpload : information about the uploaded file
type FileUpload struct {
	FullPath    string
	Path        string
	Name        string
	ContentType string
	Size        int64
}

//GetFile : return the file based on the path and name
func (f *FileUpload) GetFile() string {
	return path.Join(f.Path, f.Name)
}

//save the uploaded file
func processFileUpload(ctx context.Context, size int64, mimeType string, ext string, outPath string, in io.Reader) (context.Context, *FileUpload, error) {
	ctx, logger := GetLogger(ctx)
	if size > 0 {
		//create a temporary file and save
		outFile, err := ioutil.TempFile("", fmt.Sprintf("upload-*%s", ext))
		if err != nil {
			return ctx, nil, errors.Wrap(err, "out file")
		}
		defer func() {
			err = outFile.Close()
			if err != nil {
				logger.Warnw("out file close", "error", err)
			}
		}()

		//save the data
		size, err = io.Copy(outFile, in)
		if err != nil {
			return ctx, nil, errors.Wrap(err, "copy file")
		}

		//move the file to the upload directory
		uploadDir := path.Join(UploadAssetPathLocal, outPath)
		err = os.MkdirAll(uploadDir, os.ModePerm)
		if err != nil {
			return ctx, nil, errors.Wrap(err, "mkdirall file")
		}
		newFile := path.Join(uploadDir, path.Base(outFile.Name()))
		err = os.Rename(outFile.Name(), newFile)
		if err != nil {
			return ctx, nil, errors.Wrap(err, "rename file")
		}
		file := FileUpload{
			FullPath:    newFile,
			Path:        outPath,
			Name:        path.Base(newFile),
			ContentType: mimeType,
			Size:        size,
		}
		return ctx, &file, nil
	}
	return ctx, nil, nil
}

//save the uploaded file
func processFileUploadMultipart(ctx context.Context, inHdr *multipart.FileHeader, inFile multipart.File, outPath string) (context.Context, *FileUpload, error) {
	return processFileUpload(ctx, inHdr.Size, inHdr.Header.Get(HeaderContentType), path.Ext(inHdr.Filename), outPath, inFile)
}

//ProcessFileUploads : upload files from a multipart form
func ProcessFileUploads(r *http.Request, fileName string, outPath string) (context.Context, []*FileUpload, bool, error) {
	ctx, logger := GetLogger(r.Context())

	//load the uploaded files
	inHdrs, ok := r.MultipartForm.File[fileName]
	if !ok {
		return ctx, nil, false, nil
	}
	lenInHdrs := len(inHdrs)
	if lenInHdrs == 0 {
		return ctx, nil, false, nil
	}
	files := make([]*FileUpload, lenInHdrs)
	for idx, inHdr := range inHdrs {
		//load the uploaded file
		inFile, err := inHdr.Open()
		if err != nil {
			return ctx, nil, false, errors.Wrap(err, "in file open")
		}
		defer func() {
			err = inFile.Close()
			if err != nil {
				logger.Warnw("form file close", "error", err)
			}
		}()
		ctx, file, err := processFileUploadMultipart(ctx, inHdr, inFile, outPath)
		if err != nil {
			return ctx, nil, false, errors.Wrap(err, "process file upload")
		}
		if file == nil {
			return ctx, nil, false, nil
		}
		files[idx] = file
	}
	return ctx, files, true, nil
}

//ProcessFileUpload : upload a file from a multipart form
func ProcessFileUpload(r *http.Request, fileName string, outPath string) (context.Context, *FileUpload, bool, error) {
	ctx, logger := GetLogger(r.Context())

	//load the uploaded file
	inFile, inHdr, err := r.FormFile(fileName)
	if err != nil {
		if err == http.ErrMissingFile {
			return ctx, nil, false, nil
		}
		return ctx, nil, false, errors.Wrap(err, "form file")
	}
	defer func() {
		err = inFile.Close()
		if err != nil {
			logger.Warnw("form file close", "error", err)
		}
	}()
	ctx, file, err := processFileUploadMultipart(ctx, inHdr, inFile, outPath)
	if err != nil {
		return ctx, nil, false, errors.Wrap(err, "process file upload")
	}
	if file == nil {
		return ctx, nil, false, nil
	}
	return ctx, file, true, nil
}

//ProcessFileUploadBase64 : upload a file base64-encoded
func ProcessFileUploadBase64(r *http.Request, fileName string, outPath string) (context.Context, *FileUpload, error) {
	ctx, _ := GetLogger(r.Context())

	//extract the data
	in := r.FormValue(fileName)
	if len(in) == 0 {
		return ctx, nil, nil
	}
	tokens := strings.Split(in, ",")
	header := tokens[0]
	headerTokens := strings.FieldsFunc(header, func(r rune) bool {
		return r == ':' || r == ';'
	})
	data := tokens[1]
	decodedData, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "base64 decode")
	}
	inFile := bytes.NewReader(decodedData)
	decodedDataLen := int64(len(decodedData))
	mimeType := headerTokens[1]
	ext := fmt.Sprintf(".%s", path.Base(mimeType))

	//process the file
	ctx, file, err := processFileUpload(ctx, decodedDataLen, mimeType, ext, outPath, inFile)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "process file upload")
	}
	if file == nil {
		return ctx, nil, nil
	}
	return ctx, file, nil
}

//ExtractURLYouTubeVideoID : extract the video id from a YouTube URL
func ExtractURLYouTubeVideoID(in string) (string, bool) {
	urlVideo, err := url.Parse(in)
	if err != nil {
		return "", false
	}
	host := strings.ToLower(urlVideo.Host)

	//check for youtube
	ok := strings.Contains(host, "youtube")
	if !ok {
		return "", false
	}

	//check for a video id
	values, err := url.ParseQuery(urlVideo.RawQuery)
	if err != nil {
		return "", false
	}
	id, ok := values["v"]
	if !ok {
		return "", false
	}
	if len(id) == 0 {
		return "", false
	}
	return id[0], true
}

//GenerateYouTubePlayerHTML : generate the code to embed the player for a YouTube video based on a url
func GenerateYouTubePlayerHTML(in string) string {
	id, ok := ExtractURLYouTubeVideoID(in)
	if !ok {
		return ""
	}
	html := "<iframe src='https://www.youtube.com/embed/%s' frameborder='0' allowfullscreen></iframe>"
	return fmt.Sprintf(html, id)
}
