package main

import (
	"sync"
	"time"

	"github.com/spf13/viper"
)

//prefix for environment configuration values
const cfgPrefix = "HR"

//configuration keys
const (
	cfgKeyAWSEmailDisable                   = "AWS_EMAIL_DISABLE"
	cfgKeyAWSSMSDisable                     = "AWS_SMS_DISABLE"
	cfgKeyAWSRegion                         = "AWS_REGION"
	cfgKeyAWSS3Bucket                       = "AWS_S3_BUCKET"
	cfgKeyAWSS3Enable                       = "AWS_S3_ENABLE"
	cfgKeyAWSS3KeyBase                      = "AWS_S3_KEY_BASE"
	cfgKeyAWSSMSSenderID                    = "AWS_SMS_SENDER_ID"
	cfgKeyBatchSizeProcessEmails            = "BATCH_SIZE_PROCESS_EMAILS"
	cfgKeyBatchSizeProcessGoogleCalendars   = "BATCH_SIZE_PROCESS_GOOGLE_CALENDARS"
	cfgKeyBatchSizeProcessGoogleEvents      = "BATCH_SIZE_PROCESS_GOOGLE_EVENTS"
	cfgKeyBatchSizeProcessImgs              = "BATCH_SIZE_PROCESS_IMGS"
	cfgKeyBatchSizeProcessNotifications     = "BATCH_SIZE_PROCESS_NOTIFICATIONS"
	cfgKeyBatchSizeProcessRecurringBookings = "BATCH_SIZE_PROCESS_RECURRING_BOOKINGS"
	cfgKeyBatchSizeProcessZoomMeetings      = "BATCH_SIZE_PROCESS_ZOOM_MEETINGS"
	cfgKeyBitlyAccessToken                  = "BITLY_ACCESS_TOKEN"
	cfgKeyCronProcessGoogle                 = "CRON_PROCESS_GOOGLE"
	cfgKeyCronProcessImgs                   = "CRON_PROCESS_IMGS"
	cfgKeyCronProcessMsgs                   = "CRON_PROCESS_MSGS"
	cfgKeyCronProcessNotifications          = "CRON_PROCESS_NOTIFICATIONS"
	cfgKeyCronProcessRecurringBookings      = "CRON_PROCESS_RECURRING_BOOKINGS"
	cfgKeyCronProcessZoom                   = "CRON_PROCESS_ZOOM"
	cfgKeyDBAddress                         = "DB_ADDRESS"
	cfgKeyDBMaxIdleConnections              = "DB_MAX_IDLE"
	cfgKeyDBMaxOpenConnections              = "DB_MAX_OPEN"
	cfgKeyDBName                            = "DB_NAME"
	cfgKeyDBPwd                             = "DB_PWD"
	cfgKeyDBUser                            = "DB_USER"
	cfgKeyDevAPIToken                       = "DEV_API_TOKEN"
	cfgKeyDevModeEnable                     = "DEV_MODE_ENABLE"
	cfgKeyDomain                            = "DOMAIN"
	cfgKeyDomainRoot                        = "DOMAIN_ROOT"
	cfgKeyEmailDefault                      = "EMAIL_DEFAULT"
	cfgKeyEmailReplyTo                      = "EMAIL_REPLY_TO"
	cfgKeyEmailSender                       = "EMAIL_SENDER"
	cfgKeyEmailSenderName                   = "EMAIL_SENDER_NAME"
	cfgKeyEmailSubjectPrefix                = "EMAIL_SUBJECT_PREFIX"
	cfgKeyFacebookAPIVersion                = "FACEBOOK_API_VERSION"
	cfgKeyFacebookAppID                     = "FACEBOOK_APP_ID"
	cfgKeyFacebookConversionCost            = "FACEBOOK_CONVERSION_COST"
	cfgKeyFacebookTrackingID                = "FACEBOOK_TRACKING_ID"
	cfgKeyFileCSS                           = "FILE_CSS"
	cfgKeyFileJS                            = "FILE_JS"
	cfgKeyGoogleCalenderEmail               = "GOOGLE_CALENDAR_EMAIL"
	cfgKeyGoogleOAuthClientID               = "GOOGLE_OAUTH_CLIENT_ID"
	cfgKeyGoogleOAuthClientSecret           = "GOOGLE_OAUTH_CLIENT_SECRET"
	cfgKeyGoogleRecaptchaSecretKey          = "GOOGLE_RECAPTCHA_SECRET_KEY"
	cfgKeyGoogleRecaptchaSiteKey            = "GOOGLE_RECAPTCHA_SITE_KEY"
	cfgKeyGoogleSAFile                      = "GOOGLE_SA_FILE"
	cfgKeyGoogleTagManagerID                = "GOOGLE_TAG_MANAGER_ID"
	cfgKeyGoogleTrackingID                  = "GOOGLE_TRACKING_ID"
	cfgKeyGoogleURLMap                      = "GOOGLE_URL_MAP"
	cfgKeyJWTKey                            = "JWT_KEY"
	cfgKeyLogDevEnable                      = "LOG_DEV_ENABLE"
	cfgKeyLogFileEnable                     = "LOG_FILE_ENABLE"
	cfgKeyLogLevel                          = "LOG_LEVEL"
	cfgKeyNotificationBookingReminderMin    = "NOTIFICATION_BOOKING_REMINDER_MIN"
	cfgKeyNotificationEmails                = "NOTIFICATION_EMAILS"
	cfgKeyPageSizeProviders                 = "PAGE_SIZE_PROVIDERS"
	cfgKeyPanicHandlerDisable               = "PANIC_HANDLER_DISABLE"
	cfgKeyPayPalURLApi                      = "PAYPAL_URL_API"
	cfgKeyPayPalClientID                    = "PAYPAL_CLIENT_ID"
	cfgKeyPayPalSecret                      = "PAYPAL_SECRET"
	cfgKeyPayPalWebHookID                   = "PAYPAL_WEBHOOK_ID"
	cfgKeyPlaidClientID                     = "PLAID_CLIENT_ID"
	cfgKeyPlaidName                         = "PLAID_NAME"
	cfgKeyPlaidSandboxDisable               = "PLAID_SANDBOX_DISABLE"
	cfgKeyPlaidSecret                       = "PLAID_SECRET"
	cfgKeyPProfEnable                       = "PPROF_ENABLE"
	cfgKeySQSQueueURLEmail                  = "SQS_QUEUE_URL_EMAIL"
	cfgKeySQSWorkerCount                    = "SQS_WORKER_COUNT"
	cfgKeyServerAddressHTTP                 = "SVR_ADDRESS_HTTP"
	cfgKeyServerAddressHTTPS                = "SVR_ADDRESS_HTTPS"
	cfgKeyServerAddressPublic               = "SVR_ADDRESS_PUBLIC"
	cfgKeyServerAddressPublicIP             = "SVR_ADDRESS_PUBLIC_IP"
	cfgKeyServerSchemePublic                = "SVR_SCHEME_PUBLIC"
	cfgKeyServerTimeOutHandlerSec           = "SVR_TIMEOUT_HANDLER_SEC"
	cfgKeyServerTimeOutReadSec              = "SVR_TIMEOUT_READ_SEC"
	cfgKeyServerTimeOutWriteSec             = "SVR_TIMEOUT_WRITE_SEC"
	cfgKeyServerTLSCert                     = "SVR_TLS_CERT"
	cfgKeyServerTLSKey                      = "SVR_TLS_KEY"
	cfgKeyServerUseHTTPS                    = "SVR_USE_HTTPS"
	cfgKeyStripeClientID                    = "STRIPE_CLIENT_ID"
	cfgKeyStripeLive                        = "STRIPE_LIVE"
	cfgKeyStripePublicKey                   = "STRIPE_PUBLIC_KEY"
	cfgKeyStripeSecretKey                   = "STRIPE_SECRET_KEY"
	cfgKeyStripeWebHookSecretCheckout       = "STRIPE_WEBHOOK_SECRET_CHECKOUT"
	cfgKeyStripeWebHookSecretConnect        = "STRIPE_WEBHOOK_SECRET_CONNECT"
	cfgKeyTimeNow                           = "TIME_NOW"
	cfgKeyTokenKey                          = "TOKEN_KEY"
	cfgKeyURLAssets                         = "URL_ASSETS"
	cfgKeyURLFacebook                       = "URL_FACEBOOK"
	cfgKeyURLInstagram                      = "URL_INSTAGRAM"
	cfgKeyURLLinkedIn                       = "URL_LINKEDIN"
	cfgKeyURLTwitter                        = "URL_TWITTER"
	cfgKeyURLUploads                        = "URL_UPLOADS"
	cfgKeyURLYouTube                        = "URL_YOUTUBE"
	cfgKeyZoomClientID                      = "ZOOM_CLIENT_ID"
	cfgKeyZoomClientSecret                  = "ZOOM_CLIENT_SECRET"
	cfgKeyZoomVerificationToken             = "ZOOM_VERIFICATION_TOKEN"
)

//initialize
func init() {
	viper.SetDefault(cfgKeyAWSEmailDisable, false)
	viper.SetDefault(cfgKeyAWSSMSDisable, false)
	viper.SetDefault(cfgKeyAWSRegion, "us-west-2")
	viper.SetDefault(cfgKeyAWSS3Bucket, "homerun.work")
	viper.SetDefault(cfgKeyAWSS3Enable, "false")
	viper.SetDefault(cfgKeyAWSS3KeyBase, "/dev/asset")
	viper.SetDefault(cfgKeyAWSSMSSenderID, "HomeRunDev")
	viper.SetDefault(cfgKeyBatchSizeProcessEmails, 10)
	viper.SetDefault(cfgKeyBatchSizeProcessGoogleCalendars, 10)
	viper.SetDefault(cfgKeyBatchSizeProcessGoogleEvents, 10)
	viper.SetDefault(cfgKeyBatchSizeProcessImgs, 10)
	viper.SetDefault(cfgKeyBatchSizeProcessNotifications, 10)
	viper.SetDefault(cfgKeyBatchSizeProcessRecurringBookings, 10)
	viper.SetDefault(cfgKeyBatchSizeProcessZoomMeetings, 10)
	viper.SetDefault(cfgKeyBitlyAccessToken, "18b94bbe6eb8c4d10dcf6ff28e71e6f15ef707b7")
	viper.SetDefault(cfgKeyCronProcessGoogle, "*/1 * * * *")          //once a minute
	viper.SetDefault(cfgKeyCronProcessImgs, "*/1 * * * *")            //once a minute
	viper.SetDefault(cfgKeyCronProcessMsgs, "*/1 * * * *")            //once a minute
	viper.SetDefault(cfgKeyCronProcessNotifications, "0,30 * * * *")  //every 30 minute
	viper.SetDefault(cfgKeyCronProcessRecurringBookings, "0 0 * * *") //once a day at beginning of the day
	viper.SetDefault(cfgKeyCronProcessZoom, "*/1 * * * *")            //once a minute
	viper.SetDefault(cfgKeyDBAddress, "172.31.28.102")
	viper.SetDefault(cfgKeyDBMaxIdleConnections, 1)
	viper.SetDefault(cfgKeyDBMaxOpenConnections, 2)
	viper.SetDefault(cfgKeyDBName, "homerundb_dev")
	viper.SetDefault(cfgKeyDBPwd, "test1234")
	viper.SetDefault(cfgKeyDBUser, "devuser")
	viper.SetDefault(cfgKeyDevAPIToken, "f206239131d443b7afc82643db1b4642")
	viper.SetDefault(cfgKeyDevModeEnable, false)
	viper.SetDefault(cfgKeyEmailDefault, "info@homerun.work")
	viper.SetDefault(cfgKeyEmailSender, "homerun-dev@mail.homerun.work")
	viper.SetDefault(cfgKeyEmailSenderName, "HomeRun Dev")
	viper.SetDefault(cfgKeyEmailSubjectPrefix, "dev")
	viper.SetDefault(cfgKeyEmailReplyTo, "dev@ops.homerun.work")
	viper.SetDefault(cfgKeyFacebookAPIVersion, "v7.0")
	viper.SetDefault(cfgKeyFacebookAppID, "1091762801224843")
	viper.SetDefault(cfgKeyFacebookConversionCost, 2)
	viper.SetDefault(cfgKeyFacebookTrackingID, "")
	viper.SetDefault(cfgKeyFileCSS, "main.min.css")
	viper.SetDefault(cfgKeyFileJS, "main.min.js")
	viper.SetDefault(cfgKeyGoogleCalenderEmail, "ops@homerun.work")
	viper.SetDefault(cfgKeyGoogleOAuthClientID, "416676865938-3v8mg1bulqtfr2gf0348tjjqp4lf25i3.apps.googleusercontent.com")
	viper.SetDefault(cfgKeyGoogleOAuthClientSecret, "cQ9Cw-7hK03rcvxJu7CeDrBu")
	viper.SetDefault(cfgKeyGoogleRecaptchaSecretKey, "6LfvUJ8aAAAAAElEz-Qt3hhoJp8KO5TGX6_eV_4D")
	viper.SetDefault(cfgKeyGoogleRecaptchaSiteKey, "6LfvUJ8aAAAAABmzBRBssVeiznQA2lpU6BpetceI")
	viper.SetDefault(cfgKeyGoogleSAFile, "/home/dev/homerun/src/homerun.work/cmd/webserver/googleSA.json")
	viper.SetDefault(cfgKeyGoogleTagManagerID, "")
	viper.SetDefault(cfgKeyGoogleTrackingID, "UA-158878222-4")
	viper.SetDefault(cfgKeyGoogleURLMap, "https://www.google.com/maps/search/?api=1&query=%s")
	viper.SetDefault(cfgKeyJWTKey, "dev1234")
	viper.SetDefault(cfgKeyLogDevEnable, false)
	viper.SetDefault(cfgKeyLogFileEnable, false)
	viper.SetDefault(cfgKeyLogLevel, "info")
	viper.SetDefault(cfgKeyNotificationBookingReminderMin, 60) //1 hour
	viper.SetDefault(cfgKeyNotificationEmails, "dev@homerun.work")
	viper.SetDefault(cfgKeyPageSizeProviders, 10)
	viper.SetDefault(cfgKeyPanicHandlerDisable, false)
	viper.SetDefault(cfgKeyPayPalURLApi, "https://api.sandbox.paypal.com")
	viper.SetDefault(cfgKeyPayPalClientID, "ASeFG5cnx0w8UAm37a1zAYHg_nK3DMzkIuCTKxMagni-5f2Vijv1oVD9ZCG5cmWRoDfS8MBN0ZuH8F1C")
	viper.SetDefault(cfgKeyPayPalSecret, "EKK0BM_aMqWM5SeO5uvxGAmTZijTeg_RPtXNEglrFGBB-_fPiRY9dxaFVB5vqVyFzjrnUINF7F4113yg")
	viper.SetDefault(cfgKeyPayPalWebHookID, "72376260LB232923J")
	viper.SetDefault(cfgKeyPlaidClientID, "5f727080219f3b0011e5b326")
	viper.SetDefault(cfgKeyPlaidName, "HomeRun Dev")
	viper.SetDefault(cfgKeyPlaidSandboxDisable, false)
	viper.SetDefault(cfgKeyPlaidSecret, "1095d3f11be8d82b623143bfad30c2")
	viper.SetDefault(cfgKeyPProfEnable, false)
	viper.SetDefault(cfgKeySQSQueueURLEmail, "https://sqs.us-west-2.amazonaws.com/967135508685/ses-incoming-dev")
	viper.SetDefault(cfgKeySQSWorkerCount, 1)
	viper.SetDefault(cfgKeyServerAddressHTTP, ":8080")
	viper.SetDefault(cfgKeyServerAddressPublic, ":8080")
	viper.SetDefault(cfgKeyServerAddressPublicIP, "44.229.18.168")
	viper.SetDefault(cfgKeyServerSchemePublic, "http")
	viper.SetDefault(cfgKeyServerTimeOutHandlerSec, 60)
	viper.SetDefault(cfgKeyServerTimeOutReadSec, 65)
	viper.SetDefault(cfgKeyServerTimeOutWriteSec, 65)
	viper.SetDefault(cfgKeyServerUseHTTPS, false)
	viper.SetDefault(cfgKeyStripeClientID, "ca_H0cinHoMTN3ZeIUVwQ4MHrpob3GYldFz")
	viper.SetDefault(cfgKeyStripeLive, false)
	viper.SetDefault(cfgKeyStripePublicKey, "pk_test_tOd7aTR5VkV96X052sMuEXUr")
	viper.SetDefault(cfgKeyStripeSecretKey, "rk_test_IWc5TnSF1OOCFA0eKVYnotnw003mhuisUW")
	viper.SetDefault(cfgKeyStripeWebHookSecretCheckout, "whsec_yEXd94qS6ntf6SMwoT7IHuDsUSNSn242")
	viper.SetDefault(cfgKeyStripeWebHookSecretConnect, "whsec_FnYHv5Jq9HwIt43O3WhQOvTiWRimfbaY")
	viper.SetDefault(cfgKeyTimeNow, "")
	viper.SetDefault(cfgKeyTokenKey, "ba46754fdc70416bb58473a56016d784")
	viper.SetDefault(cfgKeyURLAssets, "/asset")
	viper.SetDefault(cfgKeyDomain, "dev.homerun.work")
	viper.SetDefault(cfgKeyDomainRoot, "homerun.work")
	viper.SetDefault(cfgKeyURLFacebook, "https://www.facebook.com/homerunworkpro")
	viper.SetDefault(cfgKeyURLInstagram, "https://www.instagram.com/homerunworkpro/")
	viper.SetDefault(cfgKeyURLLinkedIn, "")
	viper.SetDefault(cfgKeyURLTwitter, "https://twitter.com/homerunworkpro")
	viper.SetDefault(cfgKeyURLUploads, "/asset")
	viper.SetDefault(cfgKeyURLYouTube, "")
	viper.SetDefault(cfgKeyZoomClientID, "aND6YdfgTVeCjAu_1MqWcQ")
	viper.SetDefault(cfgKeyZoomClientSecret, "TDPi9P5zGz5vW12ZwHfHv6b6mpY5S3BQ")
	viper.SetDefault(cfgKeyZoomVerificationToken, "")

	//enable reading of environment variables
	viper.SetEnvPrefix(cfgPrefix)
	viper.AllowEmptyEnv(true)
	viper.AutomaticEnv()
}

//GetAWSEmailDisable : flag indicating if email should be sent
func GetAWSEmailDisable() bool {
	return viper.GetBool(cfgKeyAWSEmailDisable)
}

//GetAWSSMSDisable : flag indicating if an SMS text should be sent
func GetAWSSMSDisable() bool {
	return viper.GetBool(cfgKeyAWSSMSDisable)
}

//GetAWSRegion : AWS region
func GetAWSRegion() string {
	return viper.GetString(cfgKeyAWSRegion)
}

//GetAWSS3Bucket : AWS S3 bucket
func GetAWSS3Bucket() string {
	return viper.GetString(cfgKeyAWSS3Bucket)
}

//GetAWSS3Enable : flag indicating if AWS S3 should be used for uploads
func GetAWSS3Enable() bool {
	return viper.GetBool(cfgKeyAWSS3Enable)
}

//GetAWSS3KeyBase : base AWS S3 key
func GetAWSS3KeyBase() string {
	return viper.GetString(cfgKeyAWSS3KeyBase)
}

//GetAWSSMSSenderID : SMS sender id
func GetAWSSMSSenderID() string {
	return viper.GetString(cfgKeyAWSSMSSenderID)
}

//GetBatchSizeProcessEmails : batch size for processing emails
func GetBatchSizeProcessEmails() int {
	return viper.GetInt(cfgKeyBatchSizeProcessEmails)
}

//GetBatchSizeProcessGoogleEvents : batch size for processing Google calendar events
func GetBatchSizeProcessGoogleEvents() int {
	return viper.GetInt(cfgKeyBatchSizeProcessGoogleEvents)
}

//GetBatchSizeProcessGoogleCalendars : batch size for processing Google calendars
func GetBatchSizeProcessGoogleCalendars() int {
	return viper.GetInt(cfgKeyBatchSizeProcessGoogleCalendars)
}

//GetBatchSizeProcessImgs : batch size for processing images
func GetBatchSizeProcessImgs() int {
	return viper.GetInt(cfgKeyBatchSizeProcessImgs)
}

//GetBatchSizeProcessNotifications : batch size for processing notifications
func GetBatchSizeProcessNotifications() int {
	return viper.GetInt(cfgKeyBatchSizeProcessNotifications)
}

//GetBatchSizeProcessRecurringBookings : batch size for processing recurring bookings
func GetBatchSizeProcessRecurringBookings() int {
	return viper.GetInt(cfgKeyBatchSizeProcessRecurringBookings)
}

//GetBatchSizeProcessZoomMeetings : batch size for processing Zoom meetings
func GetBatchSizeProcessZoomMeetings() int {
	return viper.GetInt(cfgKeyBatchSizeProcessZoomMeetings)
}

//GetBitlyAccessToken : Bitly access token
func GetBitlyAccessToken() string {
	return viper.GetString(cfgKeyBitlyAccessToken)
}

//GetCronProcessGoogle : cron schedule for processing Google calendars and events
func GetCronProcessGoogle() string {
	return viper.GetString(cfgKeyCronProcessGoogle)
}

//GetCronProcessImgs : cron schedule for processing images
func GetCronProcessImgs() string {
	return viper.GetString(cfgKeyCronProcessImgs)
}

//GetCronProcessMsgs : cron schedule for processing messages
func GetCronProcessMsgs() string {
	return viper.GetString(cfgKeyCronProcessMsgs)
}

//GetCronProcessNotifications : cron schedule for processing notifications
func GetCronProcessNotifications() string {
	return viper.GetString(cfgKeyCronProcessNotifications)
}

//GetCronProcessRecurringBookings : cron schedule for processing recurring bookings
func GetCronProcessRecurringBookings() string {
	return viper.GetString(cfgKeyCronProcessRecurringBookings)
}

//GetCronProcessZoom : cron schedule for processing Zoom meetings
func GetCronProcessZoom() string {
	return viper.GetString(cfgKeyCronProcessZoom)
}

//GetDBAddress : database address
func GetDBAddress() string {
	return viper.GetString(cfgKeyDBAddress)
}

//GetDBMaxIdleConnections : database maximum idle connections
func GetDBMaxIdleConnections() int {
	return viper.GetInt(cfgKeyDBMaxIdleConnections)
}

//GetDBMaxOpenConnections : database maximum idle connections
func GetDBMaxOpenConnections() int {
	return viper.GetInt(cfgKeyDBMaxOpenConnections)
}

//GetDBName : database name
func GetDBName() string {
	return viper.GetString(cfgKeyDBName)
}

//GetDBPwd : database password
func GetDBPwd() Secret {
	return Secret(viper.GetString(cfgKeyDBPwd))
}

//GetDBUser : database user
func GetDBUser() string {
	return viper.GetString(cfgKeyDBUser)
}

//GetDevAPIToken : development API token
func GetDevAPIToken() string {
	return viper.GetString(cfgKeyDevAPIToken)
}

//GetDevModeEnable : flag indicating if the development mode is enabled
func GetDevModeEnable() bool {
	return viper.GetBool(cfgKeyDevModeEnable)
}

//GetDomain : base domain
func GetDomain() string {
	return viper.GetString(cfgKeyDomain)
}

//GetDomainRoot : root domain
func GetDomainRoot() string {
	return viper.GetString(cfgKeyDomainRoot)
}

//GetEmailDefault : default email to use
func GetEmailDefault() string {
	return viper.GetString(cfgKeyEmailDefault)
}

//GetEmailReplyTo : email to use for the reply-to
func GetEmailReplyTo() string {
	return viper.GetString(cfgKeyEmailReplyTo)
}

//GetEmailSender : email sender
func GetEmailSender() string {
	return viper.GetString(cfgKeyEmailSender)
}

//GetEmailSenderName : email sender name
func GetEmailSenderName() string {
	return viper.GetString(cfgKeyEmailSenderName)
}

//GetEmailSubjectPrefix : email prefix to use for the subject
func GetEmailSubjectPrefix() string {
	return viper.GetString(cfgKeyEmailSubjectPrefix)
}

//GetFacebookAPIVersion : Facebook api version
func GetFacebookAPIVersion() string {
	return viper.GetString(cfgKeyFacebookAPIVersion)
}

//GetFacebookAppID : Facebook application id
func GetFacebookAppID() string {
	return viper.GetString(cfgKeyFacebookAppID)
}

//GetFacebookConversionCost : Facebook tracking conversion cost
func GetFacebookConversionCost() int {
	return viper.GetInt(cfgKeyFacebookConversionCost)
}

//GetFacebookTrackingID : Facebook tracking id
func GetFacebookTrackingID() string {
	return viper.GetString(cfgKeyFacebookTrackingID)
}

//GetFileCSS : name of the CSS file
func GetFileCSS() string {
	return viper.GetString(cfgKeyFileCSS)
}

//GetFileJS : name of the JS file
func GetFileJS() string {
	return viper.GetString(cfgKeyFileJS)
}

//GetGoogleCalendarEmail : email of the owner of all Google calendars
func GetGoogleCalendarEmail() string {
	return viper.GetString(cfgKeyGoogleCalenderEmail)
}

//GetGoogleOAuthClientID : Google client id
func GetGoogleOAuthClientID() string {
	return viper.GetString(cfgKeyGoogleOAuthClientID)
}

//GetGoogleOAuthClientSecret : Google client secret
func GetGoogleOAuthClientSecret() string {
	return viper.GetString(cfgKeyGoogleOAuthClientSecret)
}

//GetGoogleRecaptchaSecretKey : Google Recaptcha secret key
func GetGoogleRecaptchaSecretKey() string {
	return viper.GetString(cfgKeyGoogleRecaptchaSecretKey)
}

//GetGoogleRecaptchaSiteKey : Google Recaptcha site key
func GetGoogleRecaptchaSiteKey() string {
	return viper.GetString(cfgKeyGoogleRecaptchaSiteKey)
}

//GetGoogleSAFile : Google service account credentials file
func GetGoogleSAFile() string {
	return viper.GetString(cfgKeyGoogleSAFile)
}

//GetGoogleTagManagerID : Google tag manager id
func GetGoogleTagManagerID() string {
	return viper.GetString(cfgKeyGoogleTagManagerID)
}

//GetGoogleTrackingID : Google tracking id
func GetGoogleTrackingID() string {
	return viper.GetString(cfgKeyGoogleTrackingID)
}

//GetGoogleURLMap : base Google map URL
func GetGoogleURLMap() string {
	return viper.GetString(cfgKeyGoogleURLMap)
}

//GetJWTKey : secret used to sign the JWT
func GetJWTKey() Secret {
	return Secret(viper.GetString(cfgKeyJWTKey))
}

//GetLogDevEnable : flag indicating if the logger should be in development mode
func GetLogDevEnable() bool {
	return viper.GetBool(cfgKeyLogDevEnable)
}

//GetLogFileEnable : flag indicating if log file should be enabled
func GetLogFileEnable() bool {
	return viper.GetBool(cfgKeyLogFileEnable)
}

//GetLogLevel : minimum log level
func GetLogLevel() string {
	return viper.GetString(cfgKeyLogLevel)
}

//GetNotificationBookingReminderMin : notification booking reminder minutes
func GetNotificationBookingReminderMin() int {
	return viper.GetInt(cfgKeyNotificationBookingReminderMin)
}

//GetNotificationEmails : notification emails
func GetNotificationEmails() string {
	return viper.GetString(cfgKeyNotificationEmails)
}

//GetPageSizeProviders : default provider list page size
func GetPageSizeProviders() int {
	return viper.GetInt(cfgKeyPageSizeProviders)
}

//GetPanicHandlerDisable : flag indicating if the panic handler should be disabled
func GetPanicHandlerDisable() bool {
	return viper.GetBool(cfgKeyPanicHandlerDisable)
}

//GetPayPalURLApi : URL for the PayPal API
func GetPayPalURLApi() string {
	return viper.GetString(cfgKeyPayPalURLApi)
}

//GetPayPalClientID : PayPal client id
func GetPayPalClientID() string {
	return viper.GetString(cfgKeyPayPalClientID)
}

//GetPayPalSecret : PayPal secret
func GetPayPalSecret() string {
	return viper.GetString(cfgKeyPayPalSecret)
}

//GetPayPalWebHookID : PayPal webhook ID
func GetPayPalWebHookID() string {
	return viper.GetString(cfgKeyPayPalWebHookID)
}

//GetPlaidClientID : Plaid client id
func GetPlaidClientID() string {
	return viper.GetString(cfgKeyPlaidClientID)
}

//GetPlaidName : Plaid application name
func GetPlaidName() string {
	return viper.GetString(cfgKeyPlaidName)
}

//GetPlaidSandboxDisable : flag indicating if Plaid sandbox is disabled
func GetPlaidSandboxDisable() bool {
	return viper.GetBool(cfgKeyPlaidSandboxDisable)
}

//GetPlaidSecret : Plaid secret
func GetPlaidSecret() string {
	return viper.GetString(cfgKeyPlaidSecret)
}

//GetPProfEnable : flag indicating if the profiler is enabled
func GetPProfEnable() bool {
	return viper.GetBool(cfgKeyPProfEnable)
}

//GetSQSQueueURLEmail : queue URL for the SQS queue for email
func GetSQSQueueURLEmail() string {
	return viper.GetString(cfgKeySQSQueueURLEmail)
}

//GetSQSWorkerCount : count of workers to process the SQS queue
func GetSQSWorkerCount() int {
	return viper.GetInt(cfgKeySQSWorkerCount)
}

//GetServerAddressHTTP : server address to use for HTTP
func GetServerAddressHTTP() string {
	return viper.GetString(cfgKeyServerAddressHTTP)
}

//GetServerAddressHTTPS : server address to use for HTTPS
func GetServerAddressHTTPS() string {
	return viper.GetString(cfgKeyServerAddressHTTPS)
}

//GetServerAddressPublic : server address to use for public consumption
func GetServerAddressPublic() string {
	return viper.GetString(cfgKeyServerAddressPublic)
}

//GetServerAddressPublicIP : server IP address to use for public consumption
func GetServerAddressPublicIP() string {
	return viper.GetString(cfgKeyServerAddressPublicIP)
}

//GetServerSchemePublic : server scheme to use for public consumption
func GetServerSchemePublic() string {
	return viper.GetString(cfgKeyServerSchemePublic)
}

//GetServerTimeOutHandlerSec : server timeout for handling a request
func GetServerTimeOutHandlerSec() int {
	return viper.GetInt(cfgKeyServerTimeOutHandlerSec)
}

//GetServerTimeOutReadSec : server timeout for reading the request
func GetServerTimeOutReadSec() int {
	return viper.GetInt(cfgKeyServerTimeOutReadSec)
}

//GetServerTimeOutWriteSec : server timeout for writing the response
func GetServerTimeOutWriteSec() int {
	return viper.GetInt(cfgKeyServerTimeOutWriteSec)
}

//GetServerTLSCert : server certificate file to use for HTTPS
func GetServerTLSCert() string {
	return viper.GetString(cfgKeyServerTLSCert)
}

//GetServerTLSKey : server key file to use for HTTPS
func GetServerTLSKey() string {
	return viper.GetString(cfgKeyServerTLSKey)
}

//GetServerUseHTTPS : flag indicating if the server should use HTTPS
func GetServerUseHTTPS() bool {
	return viper.GetBool(cfgKeyServerUseHTTPS)
}

//GetStripeClientID : Stripe client id
func GetStripeClientID() string {
	return viper.GetString(cfgKeyStripeClientID)
}

//GetStripeLive : Stripe live flag
func GetStripeLive() bool {
	return viper.GetBool(cfgKeyStripeLive)
}

//GetStripePublicKey : Stripe public key
func GetStripePublicKey() string {
	return viper.GetString(cfgKeyStripePublicKey)
}

//GetStripeSecretKey : Stripe secret key
func GetStripeSecretKey() string {
	return viper.GetString(cfgKeyStripeSecretKey)
}

//GetStripeWebHookSecretCheckout : Stripe checkout webhook secret
func GetStripeWebHookSecretCheckout() string {
	return viper.GetString(cfgKeyStripeWebHookSecretCheckout)
}

//GetStripeWebHookSecretConnect : Stripe connect webhook secret
func GetStripeWebHookSecretConnect() string {
	return viper.GetString(cfgKeyStripeWebHookSecretConnect)
}

//GetTimeNow : get the current time, using the override if set
func GetTimeNow(timeZone string) time.Time {
	now := time.Now()
	var o sync.Once
	var cfgNow time.Time
	o.Do(func() {
		cfgTime := viper.GetString(cfgKeyTimeNow)
		if cfgTime != "" {
			cfgNow, _ = time.Parse(time.RFC3339, cfgTime)
		}
	})
	if !cfgNow.IsZero() {
		now = cfgNow
	}
	if timeZone == "" {
		return now
	}
	return GetTimeLocal(now, timeZone)
}

//GetTokenKey : key used for securing tokens
func GetTokenKey() []byte {
	return []byte(viper.GetString(cfgKeyTokenKey))
}

//GetURLAssets : base assets URL
func GetURLAssets() string {
	return viper.GetString(cfgKeyURLAssets)
}

//GetURLFacebook : URL for Facebook
func GetURLFacebook() string {
	return viper.GetString(cfgKeyURLFacebook)
}

//GetURLInstagram : URL for Instagram
func GetURLInstagram() string {
	return viper.GetString(cfgKeyURLInstagram)
}

//GetURLLinkedIn : URL for LinkedIn
func GetURLLinkedIn() string {
	return viper.GetString(cfgKeyURLLinkedIn)
}

//GetURLTwitter : URL for Twitter
func GetURLTwitter() string {
	return viper.GetString(cfgKeyURLTwitter)
}

//GetURLUploads : base uploads URL
func GetURLUploads() string {
	return viper.GetString(cfgKeyURLUploads)
}

//GetURLYouTube : URL for YouTube
func GetURLYouTube() string {
	return viper.GetString(cfgKeyURLYouTube)
}

//GetZoomClientID : Zoom client id
func GetZoomClientID() string {
	return viper.GetString(cfgKeyZoomClientID)
}

//GetZoomClientSecret : Zoom client secret
func GetZoomClientSecret() string {
	return viper.GetString(cfgKeyZoomClientSecret)
}

//GetZoomVerificationToken : Zoom event verification token
func GetZoomVerificationToken() string {
	return viper.GetString(cfgKeyZoomVerificationToken)
}
