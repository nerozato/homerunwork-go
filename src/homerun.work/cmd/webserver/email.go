package main

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"path"
	"sync"

	"github.com/kvannotten/mailstrip"
	"github.com/pkg/errors"
)

//type used for email subject keys
type emailSubjectKey string

//email subject keys
const (
	EmailSubjectBookingCancelClient            emailSubjectKey = "bookingCancelClient"
	EmailSubjectBookingCancelProvider          emailSubjectKey = "bookingCancelProvider"
	EmailSubjectBookingConfirmClient           emailSubjectKey = "bookingConfirmClient"
	EmailSubjectBookingEdit                    emailSubjectKey = "bookingEdit"
	EmailSubjectBookingNewFromClientToClient   emailSubjectKey = "bookingFromClientToClient"
	EmailSubjectBookingNewFromClientToProvider emailSubjectKey = "bookingFromClientToProvider"
	EmailSubjectBookingNewFromProviderToClient emailSubjectKey = "bookingFromProviderToClient"
	EmailSubjectBookingReminder                emailSubjectKey = "bookingReminder"
	EmailSubjectCampaignAddNotification        emailSubjectKey = "campaignAddNotification"
	EmailSubjectCampaignAddProvider            emailSubjectKey = "campaignAddProvider"
	EmailSubjectCampaignPaymentNotification    emailSubjectKey = "campaignPaymentNotification"
	EmailSubjectCampaignStatusProvider         emailSubjectKey = "campaignStatusProvider"
	EmailSubjectClientInvite                   emailSubjectKey = "clientInvite"
	EmailSubjectContact                        emailSubjectKey = "contact"
	EmailSubjectDomainNotification             emailSubjectKey = "domainNotification"
	EmailSubjectInvoice                        emailSubjectKey = "invoice"
	EmailSubjectPaymentClient                  emailSubjectKey = "paymentClient"
	EmailSubjectPaymentProvider                emailSubjectKey = "paymentProvider"
	EmailSubjectProviderUserInvite             emailSubjectKey = "providerUserInvite"
	EmailSubjectPwdReset                       emailSubjectKey = "pwdReset"
	EmailSubjectVerify                         emailSubjectKey = "verify"
	EmailSubjectWelcome                        emailSubjectKey = "welcome"
)

//email subjects
var emailSubjectText = map[emailSubjectKey]string{
	EmailSubjectBookingCancelClient:            "Your order has been cancelled",
	EmailSubjectBookingCancelProvider:          "An order has been cancelled",
	EmailSubjectBookingConfirmClient:           "Your order has been confirmed",
	EmailSubjectBookingEdit:                    "Your order has been updated",
	EmailSubjectBookingNewFromClientToClient:   "Your order is pending confirmation",
	EmailSubjectBookingNewFromClientToProvider: "You have received a new order, please confirm",
	EmailSubjectBookingNewFromProviderToClient: "Your order has been created",
	EmailSubjectBookingReminder:                "Order Reminder",
	EmailSubjectCampaignAddNotification:        "New campaign from %s",
	EmailSubjectCampaignAddProvider:            "Your campaign has been submitted for review",
	EmailSubjectCampaignPaymentNotification:    "%s has paid the invoice for their campaign",
	EmailSubjectCampaignStatusProvider:         "Campaign is now %s",
	EmailSubjectClientInvite:                   "Invitiation from %s",
	EmailSubjectContact:                        "Contact Us Message",
	EmailSubjectDomainNotification:             "New custom domain from %s",
	EmailSubjectInvoice:                        "You have received an invoice for your order",
	EmailSubjectPaymentClient:                  "Your payment has been received",
	EmailSubjectProviderUserInvite:             "You have been added to the team",
	EmailSubjectPaymentProvider:                "You have received payment",
	EmailSubjectPwdReset:                       "Reset Your Password",
	EmailSubjectVerify:                         "Please Verify Your Email",
	EmailSubjectWelcome:                        "Welcome!",
}

//GetEmailSubjectText : returns the email subject text
func GetEmailSubjectText(key emailSubjectKey, args ...interface{}) string {
	v, ok := emailSubjectText[key]
	if !ok {
		return string(key)
	}
	if len(args) > 0 {
		return fmt.Sprintf(v, args...)
	}
	return v
}

//type used for email message keys
type emailMsgKey string

//email message keys
const (
	EmailMsgBookingNewFromClientToClient   emailMsgKey = "bookingFromClientToClient"
	EmailMsgBookingNewFromProviderToClient emailMsgKey = "bookingFromProviderToClient"
)

//email messages
var emailMsgText = map[emailMsgKey]string{
	EmailMsgBookingNewFromClientToClient:   "Your service order has been created. You will be notified after it is confirmed.",
	EmailMsgBookingNewFromProviderToClient: "Your service order has been created.",
}

//GetEmailMsgText : returns the email message text
func GetEmailMsgText(key emailMsgKey, args ...interface{}) string {
	v, ok := emailMsgText[key]
	if !ok {
		return string(key)
	}
	if len(args) > 0 {
		return fmt.Sprintf(v, args...)
	}
	return v
}

//BaseEmailTemplatePath : path to HTML templates for emails
const BaseEmailTemplatePath = "email/template"

//load an email template
func (s *Server) loadTemplateEmail(ctx context.Context, templateFile string) *template.Template {
	files := []string{path.Join(BaseEmailTemplatePath, "base.html"), path.Join(BaseEmailTemplatePath, templateFile)}
	tpl, err := template.New(path.Base(files[0])).Funcs(s.createTemplateFuncs()).ParseFiles(files...)
	if err != nil {
		panic(errors.Wrap(err, fmt.Sprintf("parse base template: %s", templateFile)))
	}
	return tpl
}

//create basic email template data
func (s *Server) createTemplateDataEmail() templateData {
	data := s.createTemplateData(nil)

	//populate the data
	data[TplParamEmailDefault] = GetEmailDefault()
	data[TplParamShowHomeRun] = false
	return data
}

//render a web template
func (s *Server) renderEmailTemplate(ctx context.Context, tpl *template.Template, data templateData) (string, error) {
	//write the template to a buffer
	var buffer bytes.Buffer

	//set the timezone based on the context
	timeZone := GetCtxTimeZone(ctx)
	data[TplParamTimeZone] = timeZone

	//force the template data to be keyed by string for use in the template
	m := data.CreateMap()
	err := tpl.Execute(&buffer, m)
	if err != nil {
		return "", errors.Wrap(err, "template execute")
	}
	return buffer.String(), nil
}

//create the email to the client for a booking cancellation
func (s *Server) createEmailBookingCancelClient(ctx context.Context, book *bookingUI) (context.Context, string, string, error) {
	var o sync.Once
	var tpl *template.Template
	o.Do(func() {
		tpl = s.loadTemplateEmail(ctx, "svcbookcancelclient.html")
	})
	subject := GetEmailSubjectText(EmailSubjectBookingCancelClient)
	data := s.createTemplateDataEmail()
	providerUI := s.createProviderUI(book.Provider)
	data[TplParamProvider] = providerUI
	data[TplParamSvc] = s.createServiceUI(providerUI, book.Service)
	data[TplParamBook] = book

	//use the booking timezone
	ctx = SetCtxTimeZone(ctx, book.Client.TimeZone)
	body, err := s.renderEmailTemplate(ctx, tpl, data)
	if err != nil {
		return ctx, "", "", errors.Wrap(err, "render service book cancel client")
	}
	return ctx, subject, body, nil
}

//create the email to the provider for a booking cancellation
func (s *Server) createEmailBookingCancelProvider(ctx context.Context, book *bookingUI) (context.Context, string, string, error) {
	var o sync.Once
	var tpl *template.Template
	o.Do(func() {
		tpl = s.loadTemplateEmail(ctx, "svcbookcancelprovider.html")
	})
	subject := GetEmailSubjectText(EmailSubjectBookingCancelProvider)
	data := s.createTemplateDataEmail()
	providerUI := s.createProviderUI(book.Provider)
	data[TplParamProvider] = providerUI
	data[TplParamSvc] = s.createServiceUI(providerUI, book.Service)
	data[TplParamBook] = book

	//use the provider timezone
	ctx = SetCtxTimeZone(ctx, book.Provider.User.TimeZone)
	body, err := s.renderEmailTemplate(ctx, tpl, data)
	if err != nil {
		return ctx, "", "", errors.Wrap(err, "render service book cancel provider")
	}
	return ctx, subject, body, nil
}

//create the email to the client for a booking confirmation
func (s *Server) createEmailBookingConfirmClient(ctx context.Context, book *bookingUI) (context.Context, string, string, error) {
	var o sync.Once
	var tpl *template.Template
	o.Do(func() {
		tpl = s.loadTemplateEmail(ctx, "svcbookconfirmclient.html")
	})
	subject := GetEmailSubjectText(EmailSubjectBookingConfirmClient)
	data := s.createTemplateDataEmail()
	providerUI := s.createProviderUI(book.Provider)
	data[TplParamProvider] = providerUI
	data[TplParamSvc] = s.createServiceUI(providerUI, book.Service)
	data[TplParamBook] = book

	//use the booking timezone
	ctx = SetCtxTimeZone(ctx, book.Client.TimeZone)
	body, err := s.renderEmailTemplate(ctx, tpl, data)
	if err != nil {
		return ctx, "", "", errors.Wrap(err, "render service book confirm client")
	}
	return ctx, subject, body, nil
}

//create the email to the client for a booking edit
func (s *Server) createEmailBookingEditClient(ctx context.Context, book *bookingUI) (context.Context, string, string, error) {
	var o sync.Once
	var tpl *template.Template
	o.Do(func() {
		tpl = s.loadTemplateEmail(ctx, "svcbookeditclient.html")
	})
	subject := GetEmailSubjectText(EmailSubjectBookingEdit)
	data := s.createTemplateDataEmail()
	providerUI := s.createProviderUI(book.Provider)
	data[TplParamProvider] = providerUI
	data[TplParamSvc] = s.createServiceUI(providerUI, book.Service)
	data[TplParamBook] = book

	//use the booking timezone
	ctx = SetCtxTimeZone(ctx, book.Client.TimeZone)
	body, err := s.renderEmailTemplate(ctx, tpl, data)
	if err != nil {
		return ctx, "", "", errors.Wrap(err, "render service book edit client")
	}
	return ctx, subject, body, nil
}

//create the email to the client for a new booking
func (s *Server) createEmailBookingNewClient(ctx context.Context, book *bookingUI, isClient bool) (context.Context, string, string, error) {
	var o sync.Once
	var tpl *template.Template
	o.Do(func() {
		tpl = s.loadTemplateEmail(ctx, "svcbooknewclient.html")
	})

	//find the correct subject
	var subject string
	var msg string
	if isClient {
		subject = GetEmailSubjectText(EmailSubjectBookingNewFromClientToClient)
		msg = GetEmailMsgText(EmailMsgBookingNewFromClientToClient)
	} else {
		subject = GetEmailSubjectText(EmailSubjectBookingNewFromProviderToClient)
		msg = GetEmailMsgText(EmailMsgBookingNewFromProviderToClient)
	}

	//generate the email
	data := s.createTemplateDataEmail()
	providerUI := s.createProviderUI(book.Provider)
	data[TplParamProvider] = providerUI
	data[TplParamSvc] = s.createServiceUI(providerUI, book.Service)
	data[TplParamBook] = book
	data[TplParamMsg] = msg

	//use the booking timezone
	ctx = SetCtxTimeZone(ctx, book.Client.TimeZone)
	body, err := s.renderEmailTemplate(ctx, tpl, data)
	if err != nil {
		return ctx, "", "", errors.Wrap(err, "render service book client")
	}
	return ctx, subject, body, nil
}

//create the email to the provider for a new booking
func (s *Server) createEmailBookingNewProvider(ctx context.Context, book *bookingUI, isClient bool) (context.Context, string, string, bool, error) {
	var o sync.Once
	var tpl *template.Template
	o.Do(func() {
		tpl = s.loadTemplateEmail(ctx, "svcbooknewprovider.html")
	})

	//provider should not receive the email
	if !isClient {
		return ctx, "", "", false, nil
	}

	//generate the email
	subject := GetEmailSubjectText(EmailSubjectBookingNewFromClientToProvider)
	data := s.createTemplateDataEmail()
	providerUI := s.createProviderUI(book.Provider)
	data[TplParamProvider] = providerUI
	data[TplParamSvc] = s.createServiceUI(providerUI, book.Service)
	data[TplParamBook] = book

	//use the provider timezone
	ctx = SetCtxTimeZone(ctx, book.Provider.User.TimeZone)
	body, err := s.renderEmailTemplate(ctx, tpl, data)
	if err != nil {
		return ctx, "", "", false, errors.Wrap(err, "render service book provider")
	}
	return ctx, subject, body, true, nil
}

//create the email to the client for a booking reminder
func (s *Server) createEmailBookingReminderClient(ctx context.Context, book *bookingUI) (context.Context, string, string, error) {
	var o sync.Once
	var tpl *template.Template
	o.Do(func() {
		tpl = s.loadTemplateEmail(ctx, "svcbookreminderclient.html")
	})
	subject := GetEmailSubjectText(EmailSubjectBookingReminder)
	data := s.createTemplateDataEmail()
	providerUI := s.createProviderUI(book.Provider)
	data[TplParamProvider] = providerUI
	data[TplParamSvc] = s.createServiceUI(providerUI, book.Service)
	data[TplParamBook] = book

	//use the booking timezone
	ctx = SetCtxTimeZone(ctx, book.Client.TimeZone)
	body, err := s.renderEmailTemplate(ctx, tpl, data)
	if err != nil {
		return ctx, "", "", errors.Wrap(err, "render service book client")
	}
	return ctx, subject, body, nil
}

//create the email to the provider for a booking reminder
func (s *Server) createEmailBookingReminderProvider(ctx context.Context, book *bookingUI) (context.Context, string, string, error) {
	var o sync.Once
	var tpl *template.Template
	o.Do(func() {
		tpl = s.loadTemplateEmail(ctx, "svcbookreminderprovider.html")
	})
	subject := GetEmailSubjectText(EmailSubjectBookingReminder)
	data := s.createTemplateDataEmail()
	providerUI := s.createProviderUI(book.Provider)
	data[TplParamProvider] = providerUI
	data[TplParamSvc] = s.createServiceUI(providerUI, book.Service)
	data[TplParamBook] = book

	//use the provider timezone
	ctx = SetCtxTimeZone(ctx, book.Provider.User.TimeZone)
	body, err := s.renderEmailTemplate(ctx, tpl, data)
	if err != nil {
		return ctx, "", "", errors.Wrap(err, "render service book provider")
	}
	return ctx, subject, body, nil
}

//create the add campaign notification email
func (s *Server) createEmailCampaignAddNotification(ctx context.Context, provider *providerUI, campaign *campaignUI) (context.Context, string, string, error) {
	var o sync.Once
	var tpl *template.Template
	o.Do(func() {
		tpl = s.loadTemplateEmail(ctx, "campaignaddnotification.html")
	})
	subject := GetEmailSubjectText(EmailSubjectCampaignAddNotification, provider.Name)
	data := s.createTemplateDataEmail()
	data[TplParamProvider] = provider
	data[TplParamCampaign] = campaign

	//use the client timezone
	ctx = SetCtxTimeZone(ctx, provider.User.TimeZone)
	body, err := s.renderEmailTemplate(ctx, tpl, data)
	if err != nil {
		return ctx, "", "", errors.Wrap(err, "render campaign add notification")
	}
	return ctx, subject, body, nil
}

//create the add campaign provider email
func (s *Server) createEmailCampaignAddProvider(ctx context.Context, provider *providerUI, campaign *campaignUI) (context.Context, string, string, error) {
	var o sync.Once
	var tpl *template.Template
	o.Do(func() {
		tpl = s.loadTemplateEmail(ctx, "campaignaddprovider.html")
	})
	subject := GetEmailSubjectText(EmailSubjectCampaignAddProvider)
	data := s.createTemplateDataEmail()
	data[TplParamProvider] = provider
	data[TplParamCampaign] = campaign

	//use the client timezone
	ctx = SetCtxTimeZone(ctx, provider.User.TimeZone)
	body, err := s.renderEmailTemplate(ctx, tpl, data)
	if err != nil {
		return ctx, "", "", errors.Wrap(err, "render campaign add provider")
	}
	return ctx, subject, body, nil
}

//create the payment campaign notification email
func (s *Server) createEmailCampaignPaymentNotification(ctx context.Context, provider *providerUI, campaign *campaignUI) (context.Context, string, string, error) {
	var o sync.Once
	var tpl *template.Template
	o.Do(func() {
		tpl = s.loadTemplateEmail(ctx, "campaignpaymentnotification.html")
	})
	subject := GetEmailSubjectText(EmailSubjectCampaignPaymentNotification, provider.Name)
	data := s.createTemplateDataEmail()
	data[TplParamProvider] = provider
	data[TplParamCampaign] = campaign

	//use the client timezone
	ctx = SetCtxTimeZone(ctx, provider.User.TimeZone)
	body, err := s.renderEmailTemplate(ctx, tpl, data)
	if err != nil {
		return ctx, "", "", errors.Wrap(err, "render campaign payment notification")
	}
	return ctx, subject, body, nil
}

//create the status campaign provider email
func (s *Server) createEmailCampaignStatusProvider(ctx context.Context, provider *providerUI, campaign *campaignUI) (context.Context, string, string, error) {
	var o sync.Once
	var tpl *template.Template
	o.Do(func() {
		tpl = s.loadTemplateEmail(ctx, "campaignstatusprovider.html")
	})
	subject := GetEmailSubjectText(EmailSubjectCampaignStatusProvider, campaign.Status)
	data := s.createTemplateDataEmail()
	data[TplParamProvider] = provider
	data[TplParamCampaign] = campaign

	//use the client timezone
	ctx = SetCtxTimeZone(ctx, provider.User.TimeZone)
	body, err := s.renderEmailTemplate(ctx, tpl, data)
	if err != nil {
		return ctx, "", "", errors.Wrap(err, "render campaign status provider")
	}
	return ctx, subject, body, nil
}

//create the client invite email
func (s *Server) createEmailClientInvite(ctx context.Context, provider *providerUI, client *Client) (context.Context, string, string, error) {
	var o sync.Once
	var tpl *template.Template
	o.Do(func() {
		tpl = s.loadTemplateEmail(ctx, "invite.html")
	})
	subject := GetEmailSubjectText(EmailSubjectClientInvite, provider.Name)
	data := s.createTemplateDataEmail()
	data[TplParamProvider] = provider
	data[TplParamClient] = client

	//use the client timezone
	ctx = SetCtxTimeZone(ctx, client.TimeZone)
	body, err := s.renderEmailTemplate(ctx, tpl, data)
	if err != nil {
		return ctx, "", "", errors.Wrap(err, "render invite")
	}
	return ctx, subject, body, nil
}

//create the contact email
func (s *Server) createEmailContact(ctx context.Context, provider *providerUI, client *Client, text string) (context.Context, string, string, error) {
	var o sync.Once
	var tpl *template.Template
	o.Do(func() {
		tpl = s.loadTemplateEmail(ctx, "contact.html")
	})
	subject := GetEmailSubjectText(EmailSubjectContact)
	data := s.createTemplateDataEmail()
	data[TplParamProvider] = provider
	data[TplParamClient] = client

	//convert newlines to breaks
	text = ConvertNewLinesToBreaks(text)
	data[TplParamText] = template.HTML(text)

	//use the client timezone
	ctx = SetCtxTimeZone(ctx, client.TimeZone)
	body, err := s.renderEmailTemplate(ctx, tpl, data)
	if err != nil {
		return ctx, "", "", errors.Wrap(err, "render contact")
	}
	return ctx, subject, body, nil
}

//create the domain notification email
func (s *Server) createEmailDomainNotification(ctx context.Context, provider *providerUI) (context.Context, string, string, error) {
	var o sync.Once
	var tpl *template.Template
	o.Do(func() {
		tpl = s.loadTemplateEmail(ctx, "domainnotification.html")
	})
	subject := GetEmailSubjectText(EmailSubjectDomainNotification, provider.Name)
	data := s.createTemplateDataEmail()
	data[TplParamProvider] = provider

	//use the client timezone
	ctx = SetCtxTimeZone(ctx, provider.User.TimeZone)
	body, err := s.renderEmailTemplate(ctx, tpl, data)
	if err != nil {
		return ctx, "", "", errors.Wrap(err, "render domain notification")
	}
	return ctx, subject, body, nil
}

//create the invoice email
func (s *Server) createEmailInvoice(ctx context.Context, provider *providerUI, payment *paymentUI) (context.Context, string, string, error) {
	var o sync.Once
	var tpl *template.Template
	o.Do(func() {
		tpl = s.loadTemplateEmail(ctx, "invoice.html")
	})
	subject := GetEmailSubjectText(EmailSubjectInvoice)
	data := s.createTemplateDataEmail()
	data[TplParamPayment] = payment
	data[TplParamProvider] = provider
	body, err := s.renderEmailTemplate(ctx, tpl, data)
	if err != nil {
		return ctx, "", "", errors.Wrap(err, "render invoice")
	}
	return ctx, subject, body, nil
}

//create the internal invoice email
func (s *Server) createEmailInvoiceInternal(ctx context.Context, provider *providerUI, payment *paymentUI) (context.Context, string, string, error) {
	var o sync.Once
	var tpl *template.Template
	o.Do(func() {
		tpl = s.loadTemplateEmail(ctx, "invoiceinternal.html")
	})
	subject := GetEmailSubjectText(EmailSubjectInvoice)
	data := s.createTemplateDataEmail()
	data[TplParamPayment] = payment
	data[TplParamProvider] = provider
	body, err := s.renderEmailTemplate(ctx, tpl, data)
	if err != nil {
		return ctx, "", "", errors.Wrap(err, "render invoice internal")
	}
	return ctx, subject, body, nil
}

//create the payment email to the client
func (s *Server) createEmailPaymentClient(ctx context.Context, provider *providerUI, payment *paymentUI) (context.Context, string, string, error) {
	var o sync.Once
	var tpl *template.Template
	o.Do(func() {
		tpl = s.loadTemplateEmail(ctx, "paymentclient.html")
	})
	subject := GetEmailSubjectText(EmailSubjectPaymentClient)
	data := s.createTemplateDataEmail()
	data[TplParamPayment] = payment
	data[TplParamProvider] = provider
	body, err := s.renderEmailTemplate(ctx, tpl, data)
	if err != nil {
		return ctx, "", "", errors.Wrap(err, "render payment client")
	}
	return ctx, subject, body, nil
}

//create the payment email to the provider
func (s *Server) createEmailPaymentProvider(ctx context.Context, provider *providerUI, payment *paymentUI) (context.Context, string, string, error) {
	var o sync.Once
	var tpl *template.Template
	o.Do(func() {
		tpl = s.loadTemplateEmail(ctx, "paymentprovider.html")
	})
	subject := GetEmailSubjectText(EmailSubjectPaymentProvider)
	data := s.createTemplateDataEmail()
	data[TplParamPayment] = payment
	data[TplParamProvider] = provider
	body, err := s.renderEmailTemplate(ctx, tpl, data)
	if err != nil {
		return ctx, "", "", errors.Wrap(err, "render payment provider")
	}
	return ctx, subject, body, nil
}

//create the invite for a provider user
func (s *Server) createEmailProviderUserInvite(ctx context.Context, provider *providerUI, email string) (context.Context, string, string, error) {
	var o sync.Once
	var tpl *template.Template
	o.Do(func() {
		tpl = s.loadTemplateEmail(ctx, "provideruserinvite.html")
	})
	subject := GetEmailSubjectText(EmailSubjectProviderUserInvite)
	data := s.createTemplateDataEmail()
	data[TplParamProvider] = provider
	data[TplParamEmail] = email
	body, err := s.renderEmailTemplate(ctx, tpl, data)
	if err != nil {
		return ctx, "", "", errors.Wrap(err, "render provider user invite")
	}
	return ctx, subject, body, nil
}

//create the password reset email
func (s *Server) createEmailPwdReset(ctx context.Context, tokenURL string) (context.Context, string, string, error) {
	var o sync.Once
	var tpl *template.Template
	o.Do(func() {
		tpl = s.loadTemplateEmail(ctx, "pwdreset.html")
	})
	subject := GetEmailSubjectText(EmailSubjectPwdReset)
	data := s.createTemplateDataEmail()
	data[TplParamURLPwdReset] = tokenURL
	body, err := s.renderEmailTemplate(ctx, tpl, data)
	if err != nil {
		return ctx, "", "", errors.Wrap(err, "render password reset")
	}
	return ctx, subject, body, nil
}

//create the verification email
func (s *Server) createEmailVerify(ctx context.Context, user *User, verifyURL string) (context.Context, string, string, error) {
	var o sync.Once
	var tpl *template.Template
	o.Do(func() {
		tpl = s.loadTemplateEmail(ctx, "verify.html")
	})
	subject := GetEmailSubjectText(EmailSubjectVerify)
	data := s.createTemplateDataEmail()
	data[TplParamUser] = user
	data[TplParamURLEmailVerify] = verifyURL
	body, err := s.renderEmailTemplate(ctx, tpl, data)
	if err != nil {
		return ctx, "", "", errors.Wrap(err, "render verify")
	}
	return ctx, subject, body, nil
}

//create the wlecome email
func (s *Server) createEmailWelcome(ctx context.Context, provider *providerUI) (context.Context, string, string, error) {
	var o sync.Once
	var tpl *template.Template
	o.Do(func() {
		tpl = s.loadTemplateEmail(ctx, "welcome.html")
	})
	subject := GetEmailSubjectText(EmailSubjectWelcome)
	data := s.createTemplateDataEmail()
	data[TplParamProvider] = provider
	data[TplParamShowHomeRun] = true
	body, err := s.renderEmailTemplate(ctx, tpl, data)
	if err != nil {
		return ctx, "", "", errors.Wrap(err, "render welcome")
	}
	return ctx, subject, body, nil
}

//StripEmail : strip the email plain text of signatures and reply quotes
func StripEmail(text string) string {
	email := mailstrip.Parse(text)
	return email.String()
}
