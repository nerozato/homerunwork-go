package main

import (
	"fmt"
	"strings"
)

//MsgKey : type used for message keys
type MsgKey string

//message keys
const (
	MsgBookingCancel         MsgKey = "bookingCancel"
	MsgBookingNewSingle      MsgKey = "bookingNewSingle"
	MsgBookingNewMultiple    MsgKey = "bookingNewMultiple"
	MsgClientAdd             MsgKey = "clientAdd"
	MsgClientDel             MsgKey = "clientDel"
	MsgClientDelConfirm      MsgKey = "clientDelConfirm"
	MsgClientEdit            MsgKey = "clientEdit"
	MsgClientInviteSuccess   MsgKey = "clientInviteSuccess"
	MsgContact               MsgKey = "contact"
	MsgCouponAdd             MsgKey = "couponAdd"
	MsgCouponDel             MsgKey = "couponDel"
	MsgCouponDelConfirm      MsgKey = "couponDelConfirm"
	MsgCouponEdit            MsgKey = "couponEdit"
	MsgEmailDelete           MsgKey = "emailDelete"
	MsgEmailVerify           MsgKey = "emailVerify"
	MsgEmailVerifySent       MsgKey = "emailVerifySent"
	MsgFaqAdd                MsgKey = "faqAdd"
	MsgFaqDel                MsgKey = "faqDel"
	MsgFaqDelConfirm         MsgKey = "faqDelConfirm"
	MsgFaqEdit               MsgKey = "faqEdit"
	MsgForgotPwd             MsgKey = "fogotPwd"
	MsgMetaDesc              MsgKey = "metaDesc"
	MsgMetaKeywords          MsgKey = "metaKeywords"
	MsgPageTitle             MsgKey = "pageTitle"
	MsgPaymentMarkPaid       MsgKey = "paymentMarkPaid"
	MsgPaymentMarkUnPaid     MsgKey = "paymentMarkUnPaid"
	MsgPaymentClientSuccess  MsgKey = "paymentClientSuccess"
	MsgPaymentSuccess        MsgKey = "paymentSuccess"
	MsgPayPalActivate        MsgKey = "paypalActivate"
	MsgPayPalRemove          MsgKey = "paypalRemove"
	MsgPwdReset              MsgKey = "passwordReset"
	MsgSignUpEmailErr        MsgKey = "signUpEmailErr"
	MsgSignUpSuccess         MsgKey = "signUpSuccess"
	MsgStripeActivate        MsgKey = "stripeActivate"
	MsgStripeRemove          MsgKey = "stripeRemove"
	MsgStripeSuccess         MsgKey = "stripeSuccess"
	MsgSvcAdd                MsgKey = "svcAdd"
	MsgSvcDelConfirm         MsgKey = "svcDelConfirm"
	MsgTestimonialAdd        MsgKey = "testimonialAdd"
	MsgTestimonialDel        MsgKey = "testimonialDel"
	MsgTestimonialDelConfirm MsgKey = "testimonialDelConfirm"
	MsgTestimonialEdit       MsgKey = "testimonialEdit"
	MsgUnavailable           MsgKey = "unavailable"
	MsgUpdateSuccess         MsgKey = "updateSuccess"
	MsgUserAdd               MsgKey = "userAdd"
	MsgUserAddNew            MsgKey = "userAddNew"
	MsgUserDel               MsgKey = "userDel"
	MsgUserDelConfirm        MsgKey = "userDelConfirm"
	MsgZelleActivate         MsgKey = "zelleActivate"
	MsgZelleRemove           MsgKey = "zelleRemove"
	MsgZoomSuccess           MsgKey = "zoomSuccess"
)

//messages
var msgText = map[MsgKey]string{
	MsgBookingCancel:         "Service order has been cancelled.",
	MsgBookingNewSingle:      "You have a new order.",
	MsgBookingNewMultiple:    "You have %d new orders.",
	MsgClientAdd:             "%s has been added.",
	MsgClientDel:             "%s has been deleted.",
	MsgClientDelConfirm:      "Are you sure you want to delete the client?",
	MsgClientEdit:            "%s has been updated.",
	MsgClientInviteSuccess:   "An invitation was successfully sent.",
	MsgContact:               "We will respond to your message as soon as possible.",
	MsgCouponAdd:             "Coupon has been added.",
	MsgCouponDel:             "Coupon has been deleted.",
	MsgCouponDelConfirm:      "Are you sure you want to delete the coupon?",
	MsgCouponEdit:            "Coupon has been updated.",
	MsgEmailDelete:           "Email has been deleted.",
	MsgEmailVerify:           "Thank you! Your email has now been verified.",
	MsgEmailVerifySent:       "An email has been sent. Please follow the enclosed instructions to verify your email.",
	MsgFaqAdd:                "FAQ has been added.",
	MsgFaqDel:                "FAQ has been deleted.",
	MsgFaqDelConfirm:         "Are you sure you want to delete the FAQ?",
	MsgFaqEdit:               "FAQ has been updated.",
	MsgForgotPwd:             "An email to reset your password has been sent to %s. Please check your email and follow the steps to reset your password.",
	MsgMetaDesc:              "HomeRun helps service professionals manage their service schedules, orders, invoices and payments in one place. It provides the essential tools to run service business without paying commissions.",
	MsgMetaKeywords:          "Service, professional, independent, self-employed, home-based, freelancer, worker, business, local, online, client, appointment, schedule, order, website builder, on-demand, invoice, payment, remote, live meeting, zoom, management, all-in-one, platform, marketing, designer, consultant, landscaper, trainer, teacher, tutor, handyman, repair, cleaning, cleaner, caretaker, caregiver, gardener, babysitter, nurse, specialist, developer, marketer, locksmith, roofer, artist, translator, assistant, copywriter, doctor, therapist, storyteller, musician, accountant, expert, agent, broker, carpenter, driver, delivery, manager, dietitian, hygienist, hairdresser, hair stylist, instructor, administrator, planner, maker, cook, chef, contractor, actor, entertainer, lawyer, support, technician, engineer, narrator, writer, photographer, producer, composer, pianist, singer, model, painter, carpenter, electrician, beautician, manicures, manicurist, adviser, florist, bookkeeper, strategist, seamstress, connoisseur, sommelier, blogger, tailor, buyer, builder, paralegal, coach, concierge, shopper, guide, caterer, mechanic, editor, architect, printer, plumber, massager, attorney, auditor, assessor, interpreter, veterinarian, nutritionist, courier",
	MsgPageTitle:             "Online scheduling, invoices and payment tools for service professionals",
	MsgPaymentMarkPaid:       "Are you sure you want to mark the order as paid?",
	MsgPaymentMarkUnPaid:     "Are you sure you want to mark the order as unpaid?",
	MsgPaymentClientSuccess:  "Your payment has been submitted.",
	MsgPaymentSuccess:        "The invoice has been sent to the recipient for payment.",
	MsgPayPalActivate:        "Are you sure you want to activate PayPal?",
	MsgPayPalRemove:          "Are you sure you want to deactivate PayPal?",
	MsgPwdReset:              "Password has been reset.",
	MsgSignUpEmailErr:        "Your account has been created, but a confirmation email could not be sent to %s. Please make sure to verify your email later.",
	MsgSignUpSuccess:         "Your account has been created, and a confirmation email has been sent to %s. Please check your email and follow the steps in the confirmation email to confirm your account.",
	MsgStripeActivate:        "Are you sure you want to activate Stripe?",
	MsgStripeRemove:          "Are you sure you want to deactivate Stripe?",
	MsgStripeSuccess:         "Your Stripe account has been activated.",
	MsgSvcAdd:                "The service is added and now available for ordering on your web page. To support direct ordering, you can get the order URL by editing the service.",
	MsgSvcDelConfirm:         "Are you sure you want to delete the service?",
	MsgTestimonialAdd:        "Testimonial by %s has been added.",
	MsgTestimonialDel:        "Testimonial by %s has been deleted.",
	MsgTestimonialDelConfirm: "Are you sure you want to delete the testimonial?",
	MsgTestimonialEdit:       "Testimonial by %s has been updated.",
	MsgUnavailable:           "Unavailable",
	MsgUpdateSuccess:         "Your changes have been saved successfully.",
	MsgUserAdd:               "%s has been added and notified by email.",
	MsgUserAddNew:            "%s has been added and notified by email. The user needs to follow the instructions in the email to register and complete the setup.",
	MsgUserDel:               "%s has been deleted.",
	MsgUserDelConfirm:        "Are you sure you want to delete the user?",
	MsgZelleActivate:         "Are you sure you want to activate Zelle?",
	MsgZelleRemove:           "Are you sure you want to deactivate Zelle?",
	MsgZoomSuccess:           "Your Zoom account has been activated.",
}

//titles
var msgTitleText = map[MsgKey]string{
	MsgContact: "Thank you for contacting us!",
}

//GetMsgText : returns the message
func GetMsgText(key MsgKey, args ...interface{}) string {
	v, ok := msgText[key]
	if !ok {
		return string(key)
	}
	if len(args) > 0 {
		return fmt.Sprintf(v, args...)
	}
	return v
}

//GetMsgTitle : returns the title for the message
func GetMsgTitle(key MsgKey) string {
	v, ok := msgTitleText[key]
	if !ok {
		return "Success"
	}
	return v
}

//ErrKey : type used for error keys
type ErrKey string

//error keys
const (
	Err                    ErrKey = "error"
	ErrBookingExist        ErrKey = "bookingExist"
	ErrBookingTime         ErrKey = "bookingTime"
	ErrClientEmailDup      ErrKey = "clientEmailDup"
	ErrClientInvite        ErrKey = "clientInvite"
	ErrCouponCodeDup       ErrKey = "couponCodeDup"
	ErrCredentials         ErrKey = "credentials"
	ErrDomainDup           ErrKey = "domainDup"
	ErrEmailDelete         ErrKey = "emailDelete"
	ErrEmailDup            ErrKey = "emailDup"
	ErrEmailNotExist       ErrKey = "emailNotExist"
	ErrEmailSend           ErrKey = "emailSend"
	ErrEmailVerify         ErrKey = "emailVerify"
	ErrEmailVerifyEmail    ErrKey = "emailVerifyEmail"
	ErrEmailVerifyToken    ErrKey = "emailVerifyToken"
	ErrInvoicePaid         ErrKey = "invoicePaid"
	ErrOAuthFacebook       ErrKey = "oauthFacebook"
	ErrOAuthFacebookEmail  ErrKey = "oauthFacebookEmail"
	ErrOAuthFacebookSignUp ErrKey = "oauthFacebookSignUp"
	ErrOAuthGoogle         ErrKey = "oauthGoogle"
	ErrOAuthGoogleEmail    ErrKey = "oauthGoogleEmail"
	ErrOAuthGoogleSignUp   ErrKey = "oauthGoogleSignUp"
	ErrOAuthStripe         ErrKey = "oauthStripe"
	ErrOAuthZoom           ErrKey = "oauthZoom"
	ErrID                  ErrKey = "id"
	ErrPayPalEmail         ErrKey = "paypalEmail"
	ErrPwdResetToken       ErrKey = "resetPasswordToken"
	ErrSvcExist            ErrKey = "svcExist"
	ErrSvcImgCount         ErrKey = "svcImgCount"
	ErrURLNameDup          ErrKey = "urlNameDup"
)

//errors
var errText = map[ErrKey]string{
	Err:                    "We are experiencing technical difficulties. Please try again.",
	ErrBookingExist:        "The client cannot be deleted due to having %d booking(s).",
	ErrBookingTime:         "Unforutanely, the selected time is already taken. Please try again.",
	ErrClientEmailDup:      "Client email already exists.",
	ErrClientInvite:        "We have encountered an error sending the invitation. Please try again.",
	ErrCouponCodeDup:       "Coupon code already exists.",
	ErrCredentials:         "Please enter a valid email address and password.",
	ErrDomainDup:           "The domain already exists. Please use a different domain.",
	ErrEmailDelete:         "Email address does not have an account.",
	ErrEmailDup:            "Email address has already been registered. Please enter another email address.",
	ErrEmailNotExist:       "Email address has not been registered.",
	ErrEmailSend:           "We are currently unable to send an email to %s. Please try again later.",
	ErrEmailVerify:         "We encountered technical difficulties when sending a verification email. Please try again later.",
	ErrEmailVerifyEmail:    "A confirmation email could not be sent to %s. Please make sure to verify your email later.",
	ErrEmailVerifyToken:    "Your email verification is no longer valid. Please try again.",
	ErrID:                  "We have encountered technical difficulties. Please try again.",
	ErrInvoicePaid:         "Invoice has already been paid.",
	ErrOAuthFacebook:       "We have encountered an error logging-in with Facebook. Please try again.",
	ErrOAuthFacebookEmail:  "Please login using Facebook.",
	ErrOAuthFacebookSignUp: "Please sign up using your Facebook account. Thanks.",
	ErrOAuthGoogle:         "We have encountered an error logging-in with Google. Please try again.",
	ErrOAuthGoogleEmail:    "Please login using Google.",
	ErrOAuthGoogleSignUp:   "Please sign up using your Google account. Thanks.",
	ErrOAuthStripe:         "We have encountered an error logging-in with Stripe. Please try again.",
	ErrOAuthZoom:           "We have encountered an error logging-in with Zoom. Please try again.",
	ErrPayPalEmail:         "Please use a valid PayPal email address.",
	ErrPwdResetToken:       "Your reset password request is no longer valid. Please try again.",
	ErrSvcExist:            "The service cannot be deleted due to having %d booking(s).",
	ErrSvcImgCount:         "You have too many images for your service. The maximum number of images allowed is %d.",
	ErrURLNameDup:          "The name already exists. Please use a different name.",
}

//GetErrOAuth : returns the appropraite OAuth error
func GetErrOAuth(login string) string {
	if strings.Contains(login, OAuthFacebook) {
		return GetErrText(ErrOAuthFacebookEmail)
	} else if strings.Contains(login, OAuthGoogle) {
		return GetErrText(ErrOAuthGoogleEmail)
	}
	return GetErrText(Err)
}

//GetErrText : returns the error
func GetErrText(key ErrKey, args ...interface{}) string {
	v, ok := errText[key]
	if !ok {
		return string(key)
	}
	if len(args) > 0 {
		return fmt.Sprintf(v, args...)
	}
	return v
}

//GetErrTitle : returns the title for the error
func GetErrTitle(key ErrKey) string {
	return "Error"
}

//type used for field error keys
type fieldErrKey string

//field error keys
const (
	FieldErrAgeMin             fieldErrKey = "AgeMin"
	FieldErrAgeMax             fieldErrKey = "AgeMax"
	FieldErrAnswer             fieldErrKey = "Answer"
	FieldErrBiography          fieldErrKey = "Biography"
	FieldErrBudget             fieldErrKey = "Budget"
	FieldErrClientID           fieldErrKey = "ClientID"
	FieldErrCode               fieldErrKey = "Code"
	FieldErrDate               fieldErrKey = "Date"
	FieldErrDesc               fieldErrKey = "Description"
	FieldErrDomain             fieldErrKey = "Domain"
	FieldErrDuration           fieldErrKey = "Duration"
	FieldErrEducation          fieldErrKey = "Education"
	FieldErrExperience         fieldErrKey = "Experience"
	FieldErrEmail              fieldErrKey = "Email"
	FieldErrEnd                fieldErrKey = "End"
	FieldErrFreq               fieldErrKey = "Freq"
	FieldErrFirstName          fieldErrKey = "FirstName"
	FieldErrGender             fieldErrKey = "Gender"
	FieldErrID                 fieldErrKey = "ID"
	FieldErrImg                fieldErrKey = "Img"
	FieldErrLastName           fieldErrKey = "LastName"
	FieldErrLocation           fieldErrKey = "Location"
	FieldErrLocationType       fieldErrKey = "LocationType"
	FieldErrName               fieldErrKey = "Name"
	FieldErrPadding            fieldErrKey = "Padding"
	FieldErrPaddingInitial     fieldErrKey = "PaddingInitial"
	FieldErrPaddingInitialUnit fieldErrKey = "PaddingInitialUnit"
	FieldErrPhone              fieldErrKey = "Phone"
	FieldErrProviderName       fieldErrKey = "ProviderName"
	FieldErrPrice              fieldErrKey = "Price"
	FieldErrPriceType          fieldErrKey = "PriceType"
	FieldErrPwd                fieldErrKey = "Password"
	FieldErrQuestion           fieldErrKey = "Question"
	FieldErrStart              fieldErrKey = "Start"
	FieldErrSvcID              fieldErrKey = "ServiceID"
	FieldErrSvcArea            fieldErrKey = "ServiceArea"
	FieldErrText               fieldErrKey = "Text"
	FieldErrTime               fieldErrKey = "Time"
	FieldErrTimeZone           fieldErrKey = "TimeZone"
	FieldErrTitle              fieldErrKey = "Title"
	FieldErrType               fieldErrKey = "Type"
	FieldErrURLFacebook        fieldErrKey = "URLFacebook"
	FieldErrURLInstagram       fieldErrKey = "URLInstagram"
	FieldErrURLLinkedIn        fieldErrKey = "URLLinkedIn"
	FieldErrURLName            fieldErrKey = "URLName"
	FieldErrURLTwitter         fieldErrKey = "URLTwitter"
	FieldErrURLVideo           fieldErrKey = "URLVideo"
	FieldErrURLWeb             fieldErrKey = "URLWeb"
	FieldErrUserID             fieldErrKey = "UserID"
	FieldErrValue              fieldErrKey = "Value"
	FieldErrZelleID            fieldErrKey = "ZelleID"
)

//field error messages
var fieldErrText = map[fieldErrKey]string{
	FieldErrAgeMin:             "Please enter a valid minimum age.",
	FieldErrAgeMax:             "Please enter a valid maximum age.",
	FieldErrAnswer:             "Please enter a valid question.",
	FieldErrBiography:          "Please enter a valid biography.",
	FieldErrBudget:             "Please enter a valid value for the budget.",
	FieldErrClientID:           "Please choose a client.",
	FieldErrCode:               "Please enter a valid code.",
	FieldErrDate:               "Please enter a valid date.",
	FieldErrDesc:               "Please enter a valid description.",
	FieldErrDomain:             "Please enter a valid domain.",
	FieldErrDuration:           "Please enter a valid duration.",
	FieldErrEducation:          "Please enter valid education text",
	FieldErrExperience:         "Please enter valid experience text",
	FieldErrEmail:              "Please enter a valid email address.",
	FieldErrEnd:                "Please enter a valid end date.",
	FieldErrFreq:               "Please enter a valid repeat frequency.",
	FieldErrFirstName:          "Please enter a valid first name.",
	FieldErrGender:             "Please choose a valid gender.",
	FieldErrID:                 "Please enter a valid ID.",
	FieldErrImg:                "Please select an image.",
	FieldErrLastName:           "Please enter a valid last name.",
	FieldErrLocation:           "Please enter a valid location.",
	FieldErrLocationType:       "Please enter a valid location type.",
	FieldErrName:               "Please enter a valid name.",
	FieldErrPadding:            "Please enter valid padding.",
	FieldErrPaddingInitial:     "Please enter valid advance notice.",
	FieldErrPaddingInitialUnit: "Please enter valid advance notice units.",
	FieldErrPhone:              "Please enter a valid phone number.",
	FieldErrPrice:              "Please enter a valid price.",
	FieldErrPriceType:          "Please enter a valid price type.",
	FieldErrProviderName:       "Please enter a valid name.",
	FieldErrPwd:                "Please enter a password that is at least 8 characters long, including lower-case and upper-case letters, at least one number, and a symbol.",
	FieldErrQuestion:           "Please enter a valid question.",
	FieldErrStart:              "Please enter a valid start date.",
	FieldErrSvcID:              "Please choose a service.",
	FieldErrSvcArea:            "Please select a valid service area.",
	FieldErrText:               "Please enter valid text.",
	FieldErrTime:               "Please enter a valid time.",
	FieldErrTimeZone:           "We are having problems detecting your timezone. Please try again.",
	FieldErrTitle:              "Please use a valid title.",
	FieldErrType:               "Please use a valid type.",
	FieldErrURLFacebook:        "Please use a valid URL.",
	FieldErrURLInstagram:       "Please use a valid URL.",
	FieldErrURLLinkedIn:        "Please use a valid URL.",
	FieldErrURLName:            "Please enter a valid name that is at least 3 characters long.",
	FieldErrURLTwitter:         "Please use a valid URL.",
	FieldErrURLVideo:           "Please use a valid YouTube URL.",
	FieldErrURLWeb:             "Please use a valid URL.",
	FieldErrUserID:             "Please choose a user.",
	FieldErrValue:              "Please use a valid value.",
	FieldErrZelleID:            "Please enter a valid email or phone number.",
}

//GetFieldErrText : returns the error message for a field error
func GetFieldErrText(field string) string {
	v, ok := fieldErrText[fieldErrKey(field)]
	if !ok {
		return string(field)
	}
	return v
}

//sms text messages
var smsText = map[MsgType]string{
	MsgTypeBookingCancelClient:     "Your service order has been cancelled. See the order here: %s",
	MsgTypeBookingCancelProvider:   "A service order with a client has been cancelled. See the order here: %s",
	MsgTypeBookingConfirmClient:    "Your service order has been confirmed. See the order here: %s",
	MsgTypeBookingEditClient:       "Your service order has been updated. See the order here: %s",
	MsgTypeBookingNewClient:        "Your service order has been created. You will be notified after it is confirmed. See the order here: %s",
	MsgTypeBookingNewProvider:      "You have received a new service order. Please review and confirm the order. See the order here: %s",
	MsgTypeBookingReminderClient:   "Your service order is coming up. See the order here: %s",
	MsgTypeBookingReminderProvider: "A service order is coming up with a client. See the order here: %s",
	MsgTypeClientInvite:            "",
	MsgTypeContact:                 "",
	MsgTypeEmailVerify:             "",
	MsgTypeInvoice:                 "Your service order invoice: %s",
	MsgTypeInvoiceInternal:         "Your service order invoice: %s",
	MsgTypeMessage:                 "",
	MsgTypePaymentClient:           "",
	MsgTypePaymentProvider:         "You have received the payment from %s. See the payment here: %s",
	MsgTypePwdReset:                "",
	MsgTypeWelcome:                 "",
}

//GetSMSText : returns the SMS text for the message type
func GetSMSText(msgType MsgType, args ...interface{}) string {
	v, ok := smsText[msgType]
	if !ok {
		return string(msgType)
	}
	if len(args) > 0 {
		return fmt.Sprintf(v, args...)
	}
	return v
}
