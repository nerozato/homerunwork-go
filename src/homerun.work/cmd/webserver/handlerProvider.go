package main

import (
	"context"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

//load a provider web template
func (s *Server) loadWebTemplateProvider(ctx context.Context, templateFile string) *template.Template {
	files := []string{path.Join(BaseWebTemplatePathProvider, "base.html"), path.Join(BaseWebTemplatePathProvider, templateFile)}
	tpl, err := template.New(path.Base(files[0])).Funcs(s.createTemplateFuncs()).ParseFiles(files...)
	if err != nil {
		panic(errors.Wrap(err, fmt.Sprintf("parse template provider: %s", templateFile)))
	}
	return tpl
}

//load a provider landing page web template
func (s *Server) loadWebTemplateProviderLanding(ctx context.Context, subPath string, templateFile string) *template.Template {
	files := []string{path.Join(BaseWebTemplatePathProvider, "base.html"), path.Join(BaseWebTemplatePathProvider, BaseWebTemplatePathLanding, subPath, templateFile)}
	tpl, err := template.New(path.Base(files[0])).Funcs(s.createTemplateFuncs()).ParseFiles(files...)
	if err != nil {
		panic(errors.Wrap(err, fmt.Sprintf("parse template provider: %s", templateFile)))
	}
	return tpl
}

//create basic provider template data
func (s *Server) createTemplateDataProvider(r *http.Request) (templateData, map[string]string) {
	ctx, _ := GetLogger(s.getCtx(r))
	data := s.createTemplateData(r.WithContext(ctx))

	//populate the data
	data[TplParamSvcAreaStrs] = ListServiceAreaStrs()

	//add the errors
	errs := make(map[string]string)
	data[TplParamErrs] = errs
	return data, errs
}

//handle the about page
func (s *Server) handleProviderAbout() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, _ := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateProvider(ctx, "about.html")
		})
		data, _ := s.createTemplateDataProvider(r.WithContext(ctx))
		data[TplParamActiveNav] = URIAbout
		s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
	}
}

//handle the provider base page
func (s *Server) handleProviderBase() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, URIDefault, http.StatusTemporaryRedirect)
	}
}

//handle the campaign view external page
func (s *Server) handleProviderCampaignManage() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template

	//steps on the page
	steps := struct {
		StepInvoice string
		StepStatus  string
	}{
		StepInvoice: "stepInvoice",
		StepStatus:  "stepStatus",
	}
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateProvider(ctx, "campaign-manage.html")
		})
		data, errs := s.createTemplateDataProvider(r.WithContext(ctx))
		data[TplParamSteps] = steps

		//validate the id
		idStr := r.FormValue(URLParams.ID)
		campaignID := uuid.FromStringOrNil(idStr)
		if campaignID == uuid.Nil {
			logger.Warnw("invalid uuid", "id", idStr)
			s.SetCookieErr(w, Err)
			http.Redirect(w, r.WithContext(ctx), URIDefault, http.StatusSeeOther)
			return
		}

		//load the campaign
		ctx, campaign, err := LoadCampaignByExternalID(ctx, s.getDB(), &campaignID)
		if err != nil {
			logger.Errorw("load campaign", "error", err, "id", campaignID)
			s.SetCookieErr(w, Err)
			http.Redirect(w, r.WithContext(ctx), URIDefault, http.StatusSeeOther)
			return
		}
		campaignUI := s.createCampaignUI(campaign)
		data[TplParamCampaign] = campaignUI
		data[TplParamFormAction] = campaignUI.GetURLViewExternal()

		//load the provider
		ctx, provider, err := LoadProviderByID(ctx, s.getDB(), campaign.ProviderID)
		if err != nil {
			logger.Errorw("load provider", "error", err, "id", campaign.ProviderID)
			s.SetCookieErr(w, Err)
			http.Redirect(w, r.WithContext(ctx), URIDefault, http.StatusSeeOther)
			return
		}
		providerUI := s.createProviderUI(provider)
		data[TplParamProvider] = providerUI

		//load the payment
		ctx, payment, err := LoadPaymentByProviderIDAndSecondaryIDAndType(ctx, s.getDB(), campaign.ProviderID, campaign.ID, PaymentTypeCampaign)
		if err != nil {
			logger.Errorw("load payment", "error", err, "id", campaignID)
			s.SetCookieErr(w, Err)
			http.Redirect(w, r.WithContext(ctx), URIDefault, http.StatusSeeOther)
			return
		}
		if payment != nil {
			paymentUI := s.createPaymentUI(payment)
			data[TplParamPayment] = paymentUI
		}

		//read the form
		desc := r.FormValue(URLParams.Desc)
		priceStr := r.FormValue(URLParams.Price)
		status := r.FormValue(URLParams.Status)

		//prepare the data
		data[TplParamDesc] = desc
		data[TplParamPrice] = priceStr
		data[TplParamStatus] = status

		//check the method
		if r.Method == http.MethodGet {
			if payment != nil {
				data[TplParamDesc] = payment.Note
				data[TplParamPrice] = payment.GetAmount()
			}
			data[TplParamPrice] = strconv.FormatFloat(float64(campaign.GetFee()), 'f', 2, 32)
			data[TplParamStatus] = campaign.Status
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//process the step
		step := r.FormValue(URLParams.Step)
		if step != "" {
			switch step {
			case steps.StepInvoice:
				//check if the invoice has already been paid
				if payment != nil && payment.IsPaid() {
					s.SetCookieErr(w, ErrInvoicePaid)
					http.Redirect(w, r.WithContext(ctx), campaignUI.GetURLViewExternal(), http.StatusSeeOther)
					return
				}

				//validate the form
				form := &CampaignPaymentForm{
					Price:       priceStr,
					Description: desc,
				}
				ok := s.validateForm(w, r.WithContext(ctx), tpl, data, errs, form, true)
				if !ok {
					return
				}

				//save the payment
				now := data[TplParamCurrentTime].(time.Time)
				ctx, payment, err := s.savePaymentCampaign(ctx, campaignUI, form, &now)
				if err != nil {
					logger.Errorw("save payment", "error", err)
					data[TplParamErr] = GetErrText(Err)
					s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
					return
				}
				paymentUI := s.createPaymentUI(payment)
				data[TplParamPayment] = paymentUI

				//queue the email
				ctx, err = s.queueEmailInvoiceInternal(ctx, "HomeRun", paymentUI)
				if err != nil {
					logger.Errorw("queue email invoice", "error", err)
					data[TplParamErr] = GetErrText(Err)
					s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
					return
				}

				//success
				s.SetCookieMsg(w, MsgUpdateSuccess)
				http.Redirect(w, r.WithContext(ctx), campaignUI.GetURLViewExternal(), http.StatusSeeOther)
				return
			case steps.StepStatus:
				//process the status
				form := CampaignStatusForm{
					Status: status,
				}
				ok := s.validateForm(w, r.WithContext(ctx), tpl, data, errs, form, true)
				if !ok {
					return
				}
				campaign.Status = CampaignStatus(form.Status)

				//save the campaign
				ctx, err = SaveCampaign(ctx, s.getDB(), campaign)
				if err != nil {
					logger.Errorw("save campaign", "error", err, "id", campaign.ID)
					data[TplParamErr] = GetErrText(Err)
					s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
					return
				}

				//queue the emails
				ctx, err = s.queueEmailsCampaignStatus(ctx, campaignUI)
				if err != nil {
					logger.Errorw("queue email campaign status", "error", err)
					s.SetCookieErr(w, ErrEmailVerifyToken)
					http.Redirect(w, r.WithContext(ctx), URIDefault, http.StatusSeeOther)
					return
				}

				//success
				s.SetCookieMsg(w, MsgUpdateSuccess)
				http.Redirect(w, r.WithContext(ctx), campaignUI.GetURLViewExternal(), http.StatusSeeOther)
				return
			default:
				logger.Errorw("invalid step", "id", campaign.ID, "step", step)
				s.SetCookieErr(w, Err)
				http.Redirect(w, r.WithContext(ctx), URIDefault, http.StatusSeeOther)
				return
			}
		}
	}
}

//handle the email verification
func (s *Server) handleProviderEmailVerify() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))

		//read the form
		token := r.FormValue(URLParams.Token)

		//validate the token
		ctx, ok, userID, err := CheckEmailVerifyToken(ctx, s.getDB(), token, time.Now().Unix())
		if err != nil {
			logger.Errorw("check email verify token", "error", err)
			s.SetCookieErr(w, ErrEmailVerifyToken)
			http.Redirect(w, r.WithContext(ctx), URIDefault, http.StatusSeeOther)
			return
		}
		if !ok {
			s.SetCookieErr(w, ErrEmailVerifyToken)
			http.Redirect(w, r.WithContext(ctx), URIDefault, http.StatusSeeOther)
			return
		}

		//load the provider
		ctx, provider, ok := s.loadProvider(w, r.WithContext(ctx))
		if !ok {
			return
		}

		//mark the email verified
		user := provider.GetUser()
		if userID.String() != user.ID.String() {
			s.SetCookieErr(w, ErrEmailVerifyToken)
			http.Redirect(w, r.WithContext(ctx), URIDefault, http.StatusSeeOther)
			return
		}
		ctx, err = VerifyEmail(ctx, s.getDB(), user.ID, token)
		if err != nil {
			logger.Errorw("verify email", "error", err, "id", user.ID)
			s.SetCookieErr(w, ErrEmailVerifyToken)
			http.Redirect(w, r.WithContext(ctx), URIDefault, http.StatusSeeOther)
			return
		}

		//queue the welcome email
		ctx, err = s.queueEmailWelcome(ctx, provider)
		if err != nil {
			logger.Errorw("queue email welcome", "error", err)
			s.SetCookieErr(w, ErrEmailVerifyToken)
			http.Redirect(w, r.WithContext(ctx), URIDefault, http.StatusSeeOther)
			return
		}

		//display the home page
		s.SetCookieMsg(w, MsgEmailVerify)
		http.Redirect(w, r.WithContext(ctx), provider.GetURLDashboard(), http.StatusSeeOther)
	}
}

//handle the error page
func (s *Server) handleProviderErr() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, _ := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateProvider(ctx, "error.html")
		})
		data, _ := s.createTemplateDataProvider(r.WithContext(ctx))
		data[TplParamDisableNav] = true
		s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
	}
}

//handle the maintenance page
func (s *Server) handleProviderErrMaintenance() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, _ := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateProvider(ctx, "error-maintenance.html")
		})
		data, _ := s.createTemplateDataProvider(r.WithContext(ctx))
		data[TplParamDisableNav] = true
		s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
	}
}

//handle the 404 page
func (s *Server) handleProviderErr404() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, _ := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateProvider(ctx, "error-404.html")
		})
		data, _ := s.createTemplateDataProvider(r.WithContext(ctx))
		data[TplParamDisableNav] = true
		s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
	}
}

//handle the provider faq page
func (s *Server) handleProviderFaq() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, _ := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateProvider(ctx, "faq.html")
		})
		data, _ := s.createTemplateDataProvider(r.WithContext(ctx))
		data[TplParamActiveNav] = URIFaq
		data[TplParamMetaDesc] = "Answers to frequently asked questions about HomeRun product features."
		data[TplParamPageTitle] = "Frequently asked questions and answers for HomeRun product features"
		s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
	}
}

//handle the forgot password page
func (s *Server) handleProviderForgotPwd() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateProvider(ctx, "forgotpwd.html")
		})
		data, errs := s.createTemplateDataProvider(r.WithContext(ctx))

		//read the form
		email := r.FormValue(URLParams.Email)

		//prepare the data
		data[TplParamFormAction] = URIForgotPwd
		data[TplParamEmail] = email

		//check the method
		if r.Method == http.MethodGet {
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//check the email
		form := EmailForm{
			Email: strings.TrimSpace(email),
		}
		ok := s.validateForm(w, r.WithContext(ctx), tpl, data, errs, form, true)
		if !ok {
			return
		}

		//check if the email exists
		ctx, user, err := LoginExists(ctx, s.getDB(), email)
		if err != nil {
			logger.Errorw("email exists", "error", err, "email", email)
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		} else if user == nil {
			logger.Debugw("email not exists", "email", email)
			errs[string(FieldErrEmail)] = GetErrText(ErrEmailNotExist)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		} else if user.IsOAuth {
			logger.Debugw("email oauth", "email", email)
			data[TplParamErr] = GetErrOAuth(user.Login)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//create a reset token
		token, err := CreatePwdResetToken()
		if err != nil {
			logger.Errorw("password reset token", "error", err)
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//compute the expiration from now and store the token for verification
		expiration := time.Now().Unix() + int64(resetPwdTokenExpiration.Seconds())
		ctx, err = SavePwdResetToken(ctx, s.getDB(), user.ID, token, expiration)

		//queue the email
		ctx, err = s.queueEmailPwdReset(ctx, user, token)
		if err != nil {
			logger.Errorw("queue email password reset", "error", err)
			data[TplParamErr] = GetErrText(ErrEmailSend, email)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//signal success
		s.SetCookieMsg(w, MsgForgotPwd, user.Email)
		http.Redirect(w, r.WithContext(ctx), URILogin, http.StatusSeeOther)
	}
}

//handle the how it works page
func (s *Server) handleProviderHowItWorks() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, _ := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateProvider(ctx, "how_it_works.html")
		})
		data, _ := s.createTemplateDataProvider(r.WithContext(ctx))
		data[TplParamActiveNav] = URIHowItWorks
		data[TplParamMetaDesc] = "The list of HomeRun key features and explanation of how HomeRun can help service professionals build website, market services, schedule orders, send invoices and receive payments."
		data[TplParamPageTitle] = "Product features and benefits to service professionals"
		s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
	}
}

//handle the provider how-to page
func (s *Server) handleProviderHowTo() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, _ := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateProvider(ctx, "how-to.html")
		})
		data, _ := s.createTemplateDataProvider(r.WithContext(ctx))
		data[TplParamMetaDesc] = "Step-by-step instructions on using HomeRun to activate Zoom video conferencing and schedule live video meetings in order to offer services to remote clients."
		data[TplParamPageTitle] = "How to use the HomeRun for remote services"
		s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
	}
}

//handle the index page
func (s *Server) handleProviderIndex() http.HandlerFunc {
	var m sync.Mutex
	var o sync.Once
	var tpl *template.Template

	//local cache
	var expirationSec int64 = 600
	var expiration int64
	var alert *ContentAlert
	var tips *ContentTips
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateProvider(ctx, "index.html")
		})
		data, _ := s.createTemplateDataProvider(r.WithContext(ctx))
		data[TplParamActiveNav] = URIDefault

		//load the content
		now := data[TplParamCurrentTime].(time.Time)
		nowSec := now.Unix()
		if expiration < nowSec {
			m.Lock()
			if expiration < nowSec {
				var err error
				ctx, alert, err = LoadContentAlert(ctx, s.getDB())
				if err != nil {
					logger.Errorw("load alert", "error", err)
				}
				ctx, tips, err = LoadContentTips(ctx, s.getDB())
				if err != nil {
					logger.Errorw("load tips", "error", err)
				}
				expiration = nowSec + expirationSec
			}
			m.Unlock()
		}
		data[TplParamAlert] = alert
		data[TplParamTips] = tips
		s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
	}
}

func (s *Server) handleProviderLandingTutors() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, _ := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateProviderLanding(ctx, BaseWebTemplatePathLandingTutors, "landing.html")
		})
		data, _ := s.createTemplateDataProvider(r.WithContext(ctx))
		data[TplParamActiveNav] = URITutors
		data[TplParamTypeSignUp] = ServiceAreaEducationAndTraining
		data[TplParamMetaDesc] = "HomeRun helps tutors offer remote lessons to people from anywhere without paying commissions. It is an all-in-one platform for tutors to manage their service schedules, lessons, clients, invoices and payments in one place."
		data[TplParamPageTitle] = "Online scheduling, marketing, live meetings, invoices and payment for online tutors"
		s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
	}
}

//handle the login page
func (s *Server) handleProviderLogin() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateProvider(ctx, "login.html")
		})
		data, errs := s.createTemplateDataProvider(r.WithContext(ctx))

		//probe for the user id and go to the dashboard if possible
		userID := GetCtxUserID(ctx)
		if userID != nil {
			provider := providerUI{}
			http.Redirect(w, r.WithContext(ctx), provider.GetURLBookings(), http.StatusSeeOther)
			return
		}

		//read the form
		email := r.FormValue(URLParams.Email)
		oauth := r.FormValue(URLParams.OAuth)
		pwd := Secret(r.FormValue(URLParams.Password))

		//prepare the data
		data[TplParamFormAction] = URILogin
		data[TplParamEmail] = email

		//check the method
		if r.Method == http.MethodGet {
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//check if using oauth
		switch oauth {
		case OAuthFacebook:
			s.invokeHdlrGet(s.handleFacebookLogin(), w, r.WithContext(ctx))
			return
		case OAuthGoogle:
			s.invokeHdlrGet(s.handleGoogleLogin(), w, r.WithContext(ctx))
			return
		}

		//validate the data
		form := CredentialsForm{
			EmailForm: EmailForm{
				Email: strings.TrimSpace(email),
			},
			PasswordForm: PasswordForm{
				Password: pwd,
			},
		}
		ok := s.validateForm(w, r.WithContext(ctx), tpl, data, errs, form, true)
		if !ok {
			return
		}

		//check the password
		ctx, ok, userID, userLogin, userOAuth, err := CheckPasswordUser(ctx, s.getDB(), email, pwd)
		if err != nil {
			logger.Warnw("password check", "error", err)
			data[TplParamErr] = GetErrText(ErrCredentials)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		} else if userOAuth {
			logger.Debugw("email oauth", "email", email)
			data[TplParamErr] = GetErrOAuth(userLogin)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		} else if !ok {
			logger.Debugw("password check", "email", email)
			data[TplParamErr] = GetErrText(ErrCredentials)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//refresh the token and store in the cookie
		_, err = s.refreshToken(w, r.WithContext(ctx), userID)
		if err != nil {
			logger.Warnw("refresh token", "error", err)
			data[TplParamErr] = GetErrText(ErrCredentials)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//redirect after login
		ctx, err = s.redirectLogin(w, r.WithContext(ctx), userID, "")
		if err != nil {
			logger.Warnw("redirect login", "error", err)
			data[TplParamErr] = GetErrText(ErrCredentials)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}
	}
}

//handle the logout page
func (s *Server) handleProviderLogout() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, _ := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateProvider(ctx, "logout.html")
		})
		data, _ := s.createTemplateDataProvider(r.WithContext(ctx))

		//delete the token
		s.DeleteCookieToken(w)
		s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
	}
}

//handle the provider payment page
func (s *Server) handleProviderPayment() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateProvider(ctx, "payment.html")
		})
		data, _ := s.createTemplateDataProvider(r.WithContext(ctx))

		//load the payment
		idStr := r.FormValue(URLParams.ID)
		id, err := uuid.FromString(idStr)
		if err != nil {
			logger.Errorw("invalid id", "error", err, "id", idStr)
			http.Error(w, "", http.StatusInternalServerError)
			return
		}
		ctx, payment, err := LoadPaymentByID(ctx, s.getDB(), &id)
		if err != nil {
			logger.Errorw("load payment", "error", err, "id", id)
			http.Error(w, "", http.StatusInternalServerError)
			return
		}
		paymentUI := s.createPaymentUI(payment)
		data[TplParamPayment] = paymentUI

		//load the campaign
		ctx, campaign, err := LoadCampaignByID(ctx, s.getDB(), payment.SecondaryID)
		if err != nil {
			logger.Errorw("load campaign", "error", err, "id", payment.SecondaryID)
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}
		campaignUI := s.createCampaignUI(campaign)
		data[TplParamCampaign] = campaignUI

		//load the provider
		ctx, provider, err := LoadProviderByID(ctx, s.getDB(), campaign.ProviderID)
		if err != nil {
			logger.Errorw("load provider", "error", err, "id", campaign.ProviderID)
			s.SetCookieErr(w, Err)
			http.Redirect(w, r.WithContext(ctx), URIDefault, http.StatusSeeOther)
			return
		}
		providerUI := s.createProviderUI(provider)
		data[TplParamProvider] = providerUI

		//check for the a payment confirmation
		now := data[TplParamCurrentTime].(time.Time)
		paypalID := r.FormValue(URLParams.PayPalID)
		stripeID := r.FormValue(URLParams.StripeID)
		status := r.FormValue(URLParams.State)
		if paypalID != "" || stripeID != "" || status != "" {
			data[TplParamSuccess] = true
			if !payment.IsPaid() {
				//save the payment
				payment.Paid = &now
				ctx, err = UpdatePaymentPaid(ctx, s.getDB(), payment.ID, payment.Paid, payment.Captured)
				if err != nil {
					logger.Errorw("mark payment paid", "error", err, "id", payment.ID)
					data[TplParamErr] = GetErrText(Err)
					s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
					return
				}

				//mark the campaign as paid
				campaign.Paid = true
				ctx, err = SaveCampaign(ctx, s.getDB(), campaign)
				if err != nil {
					logger.Errorw("update campaign", "error", err, "id", campaign.ID)
					data[TplParamErr] = GetErrText(Err)
					s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
					return
				}

				//queue the emails
				ctx, err = s.queueEmailsCampaignPayment(ctx, providerUI, campaignUI)
				if err != nil {
					logger.Errorw("queue email campaign payment", "error", err)
					data[TplParamErr] = GetErrText(Err)
					s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
					return
				}
				http.Redirect(w, r.WithContext(ctx), campaignUI.GetURLPayment(payment.ID), http.StatusSeeOther)
				return
			}
		} else {
			url := campaignUI.GetURLPayment(payment.ID)

			//create a paypal order if necessary
			if payment.PayPalID == nil {
				err = s.createPaymentPayPal(ctx, nil, payment)
				if err != nil {
					logger.Errorw("create paypal order", "error", err, "id", payment.ID)
					s.SetCookieErr(w, Err)
					http.Redirect(w, r.WithContext(ctx), url, http.StatusSeeOther)
					return
				}
			}
			data[TplParamPayPalOrderID] = payment.PayPalID

			//create a stripe session if necessary
			if payment.StripeSessionID == nil {
				err = s.createPaymentStripe(ctx, nil, payment)
				if err != nil {
					logger.Errorw("create stripe session", "error", err, "id", payment.ID)
					s.SetCookieErr(w, Err)
					http.Redirect(w, r.WithContext(ctx), url, http.StatusSeeOther)
					return
				}
			}
			data[TplParamStripeSessionID] = payment.StripeSessionID
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}
		s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
	}
}

//list the providers
func (s *Server) handleProviderProviders() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateProvider(ctx, "providers.html")
		})
		data := s.createTemplateData(r.WithContext(ctx))

		//read the form
		prev := r.FormValue(URLParams.Prev)
		next := r.FormValue(URLParams.Next)
		serviceArea := r.FormValue(URLParams.Type)
		data[TplParamType] = serviceArea

		//load the providers
		ctx, providers, prev, next, err := ListProviders(ctx, s.getDB(), serviceArea, prev, next, GetPageSizeProviders())
		if err != nil {
			logger.Errorw("list providers", "error", err)
		}
		data[TplParamProviders] = s.createProviderUIs(providers)

		//handle pagination
		if prev != "" {
			urlPrev, err := CreateURLRelParams(URIProviders, URLParams.Prev, prev)
			if err != nil {
				logger.Errorw("create url", "error", err)
			}
			data[TplParamURLPrev] = urlPrev
		}
		if next != "" {
			urlNext, err := CreateURLRelParams(URIProviders, URLParams.Next, next)
			if err != nil {
				logger.Errorw("create url", "error", err)
			}
			data[TplParamURLNext] = urlNext
		}
		s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
	}
}

//handle the password reset page
func (s *Server) handleProviderPwdReset() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateProvider(ctx, "pwdreset.html")
		})
		data, errs := s.createTemplateDataProvider(r.WithContext(ctx))

		//prepare the data
		data[TplParamFormAction] = URIPwdReset

		//read the form
		pwd := Secret(r.FormValue(URLParams.Password))
		token := r.FormValue(URLParams.Token)

		//validate the token
		ctx, ok, userID, err := CheckPwdResetToken(ctx, s.getDB(), token, time.Now().Unix())
		if err != nil {
			logger.Errorw("check password reset token", "error", err)
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}
		if !ok {
			//display the forgot password page
			s.SetCookieErr(w, ErrPwdResetToken)
			http.Redirect(w, r.WithContext(ctx), URIForgotPwd, http.StatusSeeOther)
			return
		}
		data[TplParamToken] = token

		//check the method
		if r.Method == http.MethodGet {
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//check the password
		form := PasswordForm{
			Password: pwd,
		}
		ok = s.validateForm(w, r.WithContext(ctx), tpl, data, errs, form, true)
		if !ok {
			return
		}

		//update the password
		ctx, err = ResetPassword(ctx, s.getDB(), userID, pwd, token)
		if err != nil {
			logger.Errorw("save password", "error", err, "id", userID, "token", token)
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//success
		s.SetCookieMsg(w, MsgPwdReset)
		http.Redirect(w, r.WithContext(ctx), URILogin, http.StatusSeeOther)
	}
}

//handle the policy page
func (s *Server) handleProviderPolicy() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, _ := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateProvider(ctx, "privacy_policy.html")
		})
		data, _ := s.createTemplateDataProvider(r.WithContext(ctx))
		s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
	}
}

//handle the robots file
func (s *Server) handleProviderRobots() http.HandlerFunc {
	var o sync.Once
	var data []byte
	return func(w http.ResponseWriter, r *http.Request) {
		o.Do(func() {
			var err error
			file := path.Join(BaseWebAssetPath, "robots.txt")
			data, err = ioutil.ReadFile(file)
			if err != nil {
				panic(errors.Wrap(err, fmt.Sprintf("load file: %s", file)))
			}
		})
		w.Header().Set("Content-Type", "text/plain")
		w.Write(data)
	}
}

//list the provider service areas
func (s *Server) handlerProviderServiceAreas() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateProvider(ctx, "service-areas.html")
		})
		data := s.createTemplateData(r.WithContext(ctx))

		//load the service areas
		ctx, serviceAreas, err := ListProviderServiceAreas(ctx, s.getDB())
		if err != nil {
			logger.Errorw("list service areas", "error", err)
		}
		data[TplParamServiceAreas] = serviceAreas
		s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
	}
}

//handle the index page
func (s *Server) handleProviderSignUp() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateProvider(ctx, "signup.html")
		})
		data, errs := s.createTemplateDataProvider(r.WithContext(ctx))

		//probe for the signup type
		signUpType := r.FormValue(URLParams.Type)
		data[TplParamTypeSignUp] = signUpType
		s.SetCookieSignUpType(w, signUpType)

		//probe for the user id and go to the dashboard if possible
		userID := GetCtxUserID(ctx)
		if userID != nil {
			provider := providerUI{}
			http.Redirect(w, r.WithContext(ctx), provider.GetURLBookings(), http.StatusSeeOther)
			return
		}

		//read the form
		email := r.FormValue(URLParams.Email)
		firstName := r.FormValue(URLParams.FirstName)
		lastName := r.FormValue(URLParams.LastName)
		oauth := r.FormValue(URLParams.OAuth)
		providerIDStr := r.FormValue(URLParams.ProviderID)
		pwd := Secret(r.FormValue(URLParams.Password))
		timeZone := r.FormValue(URLParams.TimeZone)

		//prepare the data
		data[TplParamFormAction] = URISignUp
		data[TplParamEmail] = email
		data[TplParamNameFirst] = firstName
		data[TplParamNameLast] = lastName
		data[TplParamMetaDesc] = "Sign up for free to build own business website for services."
		data[TplParamPageTitle] = "Sign up for HomeRun"
		data[TplParamProviderID] = providerIDStr

		//check the method
		if r.Method == http.MethodGet {
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//check if using oauth
		if oauth != "" {
			ctx = SetCtxIsSignUp(ctx, true)
			ctx = SetCtxTimeZone(ctx, timeZone)
			ctx = SetCtxType(ctx, signUpType)
			switch oauth {
			case OAuthFacebook:
				s.invokeHdlrGet(s.handleFacebookLogin(), w, r.WithContext(ctx))
				return
			case OAuthGoogle:
				s.invokeHdlrGet(s.handleGoogleLogin(), w, r.WithContext(ctx))
				return
			}
		}

		//check the recaptcha
		googleRecaptchaResponse := r.FormValue(URLParams.GoogleRecaptchaResponse)
		ctx, err := VerifyRecaptchaResponseGoogle(ctx, googleRecaptchaResponse)
		if err != nil {
			logger.Errorw("google recaptcha", "error", err)
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//validate the user
		form := UserSignUpForm{
			EmailForm: EmailForm{
				Email: strings.TrimSpace(email),
			},
			PasswordForm: PasswordForm{
				Password: pwd,
			},
			UserForm: UserForm{
				FirstName: firstName,
				LastName:  lastName,
			},
			TimeZoneForm: TimeZoneForm{
				TimeZone: timeZone,
			},
		}
		ok := s.validateForm(w, r.WithContext(ctx), tpl, data, errs, form, true)
		if !ok {
			return
		}

		//check if the email exists
		ctx, user, err := LoginExists(ctx, s.getDB(), email)
		if err != nil {
			logger.Errorw("email exists", "error", err, "email", email)
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		} else if user != nil {
			if user.IsOAuth {
				logger.Debugw("email oauth", "email", email)
				data[TplParamErr] = GetErrOAuth(user.Login)
				s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
				return
			}
			logger.Debugw("email exists", "email", email)
			errs[string(FieldErrEmail)] = GetErrText(ErrEmailDup)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//save the user, assuming the login is the email
		user = &User{
			FirstName:  form.FirstName,
			LastName:   form.LastName,
			Email:      email,
			Login:      email,
			TimeZone:   form.TimeZone,
			SignUpType: signUpType,
		}
		ctx, err = s.saveUser(w, r.WithContext(ctx), user, pwd)
		if err != nil {
			logger.Errorw("save user", "error", err)
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//queue the email
		ctx, err = s.queueEmailVerify(ctx, user)
		if err != nil {
			logger.Errorw("queue email verify", "error", err)
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}
		http.Redirect(w, r.WithContext(ctx), createDashboardURLDashboard(), http.StatusSeeOther)
	}
}

//handle the signup link page
func (s *Server) handleProviderSignUpLink() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, _ := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateProvider(ctx, "signup-link.html")
		})
		data, _ := s.createTemplateDataProvider(r.WithContext(ctx))

		//probe for an existing provider and use the existing information if possible
		ctx, provider, ok := s.loadProvider(w, r.WithContext(ctx))
		if !ok {
			return
		}
		data[TplParamProvider] = provider
		s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
	}
}

//handle the signup main page
func (s *Server) handleProviderSignUpMain() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateProvider(ctx, "signup-main.html")
		})
		data, errs := s.createTemplateDataProvider(r.WithContext(ctx))
		now := data[TplParamCurrentTime].(time.Time)

		//probe for the user id
		userID := GetCtxUserID(ctx)
		if userID == nil {
			logger.Warnw("invalid user id")
			http.Redirect(w, r.WithContext(ctx), URILogin, http.StatusSeeOther)
			return
		}
		data[TplParamFormAction] = URISignUpMain
		data[TplParamDisableAuth] = true

		//probe for the signup type
		signUpType, err := s.GetCookieSignUpType(r)
		if err != nil {
			logger.Warnw("cookie signup type", "error", err)
			http.Redirect(w, r.WithContext(ctx), URIErr, http.StatusSeeOther)
			return
		}
		data[TplParamTypeSignUp] = signUpType

		//probe for an existing provider and use the existing information if possible
		sendWelcome := false
		ctx, provider, ok := s.loadProvider(w, r.WithContext(ctx))
		if !ok {
			return
		}
		if provider == nil {
			ctx, user, err := LoadUserByID(ctx, s.getDB(), userID)
			if err != nil {
				logger.Warnw("bad user id", "id", userID)
				http.Redirect(w, r.WithContext(ctx), URIErr, http.StatusSeeOther)
				return
			}
			ctx = SetCtxTimeZone(ctx, user.TimeZone)

			//create a new provider
			newProvider := &Provider{
				User: user,
				Name: user.FormatName(),
			}
			provider = s.createProviderUI(newProvider)
			sendWelcome = true
		}

		//check the method
		if r.Method == http.MethodGet {
			//default the provider
			data[TplParamProviderName] = provider.Name
			data[TplParamBio] = provider.Description
			data[TplParamEducation] = provider.Education
			data[TplParamExperience] = provider.Experience
			data[TplParamSvcArea] = provider.ServiceArea

			//default the service by looking for the first one
			ctx, svcs, ok := s.loadTemplateServices(w, r.WithContext(ctx), tpl, data, provider)
			if !ok {
				return
			}
			if len(svcs) > 0 {
				data[TplParamSvcID] = svcs[0].ID
				data[TplParamName] = svcs[0].Name
				data[TplParamDesc] = svcs[0].Description
				data[TplParamDuration] = strconv.Itoa(svcs[0].Duration)
				data[TplParamPrice] = svcs[0].Price
				data[TplParamPriceType] = svcs[0].PriceType
			} else {
				data[TplParamName] = ""
				data[TplParamDesc] = ""
				data[TplParamDuration] = strconv.Itoa(ServiceDurationDefault)
				data[TplParamPrice] = ""
				data[TplParamPriceType] = ""
			}

			//default the schedule
			start := ParseTimeLocal(ProviderDefaultScheduleStart, now, provider.User.TimeZone)
			duration := ProviderDefaultScheduleDuration
			schedule := provider.GetSchedule()
			if schedule == nil {
				data[TplParamCheckedMon] = true
				data[TplParamCheckedTue] = true
				data[TplParamCheckedWed] = true
				data[TplParamCheckedThu] = true
				data[TplParamCheckedFri] = true
				data[TplParamCheckedSat] = false
				data[TplParamCheckedSun] = false
			} else {
				data[TplParamCheckedMon] = !schedule.DaySchedules[time.Monday].Unavailable
				data[TplParamCheckedTue] = !schedule.DaySchedules[time.Tuesday].Unavailable
				data[TplParamCheckedWed] = !schedule.DaySchedules[time.Wednesday].Unavailable
				data[TplParamCheckedThu] = !schedule.DaySchedules[time.Thursday].Unavailable
				data[TplParamCheckedFri] = !schedule.DaySchedules[time.Friday].Unavailable
				data[TplParamCheckedSat] = !schedule.DaySchedules[time.Saturday].Unavailable
				data[TplParamCheckedSun] = !schedule.DaySchedules[time.Sunday].Unavailable

				//find the first valid start and duration
				for _, daySchedule := range schedule.DaySchedules {
					if daySchedule.Unavailable {
						continue
					}
					for _, timeDuration := range daySchedule.TimeDurations {
						start = timeDuration.Start
						duration = timeDuration.Duration
					}
				}
			}
			data[TplParamTime] = FormatTimeLocal(start, provider.User.TimeZone)
			data[TplParamScheduleDuration] = duration
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//read the form
		providerName := r.FormValue(URLParams.ProviderName)
		bio := r.FormValue(URLParams.Bio)
		education := r.FormValue(URLParams.Education)
		experience := r.FormValue(URLParams.Experience)
		subject := r.FormValue(URLParams.Name)
		desc := r.FormValue(URLParams.Desc)
		duration := r.FormValue(URLParams.Duration)
		price := r.FormValue(URLParams.Price)
		priceTypeStr := r.FormValue(URLParams.PriceType)
		checkedMon := r.FormValue(URLParams.CheckedMon) == "on"
		checkedTue := r.FormValue(URLParams.CheckedTue) == "on"
		checkedWed := r.FormValue(URLParams.CheckedWed) == "on"
		checkedThu := r.FormValue(URLParams.CheckedThu) == "on"
		checkedFri := r.FormValue(URLParams.CheckedFri) == "on"
		checkedSat := r.FormValue(URLParams.CheckedSat) == "on"
		checkedSun := r.FormValue(URLParams.CheckedSun) == "on"
		timeStr := r.FormValue(URLParams.Time)
		scheduleDurationStr := r.FormValue(URLParams.ScheduleDuration)
		svcArea := r.FormValue(URLParams.SvcArea)

		//prepare the data
		data[TplParamProviderName] = providerName
		data[TplParamBio] = bio
		data[TplParamEducation] = education
		data[TplParamExperience] = experience
		data[TplParamName] = subject
		data[TplParamDesc] = desc
		data[TplParamDuration] = duration
		data[TplParamPrice] = price
		data[TplParamPriceType] = priceTypeStr
		data[TplParamCheckedMon] = checkedMon
		data[TplParamCheckedTue] = checkedTue
		data[TplParamCheckedWed] = checkedWed
		data[TplParamCheckedThu] = checkedThu
		data[TplParamCheckedFri] = checkedFri
		data[TplParamCheckedSat] = checkedSat
		data[TplParamCheckedSun] = checkedSun
		data[TplParamSvcArea] = svcArea
		data[TplParamTime] = timeStr
		data[TplParamScheduleDuration] = scheduleDurationStr

		//handle the uploaded logo
		ctx, uploadLogo, err := s.processFileUploadBase64(r.WithContext(ctx), URLParams.ImgLogo, provider.FormatUserID())
		if err != nil {
			logger.Errorw("upload file", "error", err, "file")
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//validate the id
		var svcID uuid.UUID
		idStr := r.FormValue(URLParams.SvcID)
		if idStr != "" {
			svcID = uuid.FromStringOrNil(idStr)
			if svcID == uuid.Nil {
				logger.Warnw("invalid uuid", "id", idStr)
				data[TplParamErr] = GetErrText(Err)
				s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
				return
			}
		}

		//validate the data
		form := TutorForm{
			ProviderName: providerName,
			Biography:    bio,
			Education:    education,
			Experience:   experience,
			ServiceArea:  svcArea,
			Service: ServiceForm{
				ApptOnly:     true,
				Description:  desc,
				Duration:     duration,
				Interval:     ServiceIntervals[0].ValueStr,
				Location:     provider.Location,
				LocationType: string(ServiceLocationTypeRemote),
				NameForm: NameForm{
					Name: subject,
				},
				Padding:            "0",
				PaddingInitial:     "0",
				PaddingInitialUnit: string(PaddingUnitHours),
				Price:              price,
				PriceType:          priceTypeStr,
			},
			Time:             timeStr,
			ScheduleDuration: scheduleDurationStr,
		}
		ok = s.validateForm(w, r.WithContext(ctx), tpl, data, errs, form, true)
		if !ok {
			return
		}

		//save the provider
		provider.SetName(form.ProviderName)
		provider.Description = form.Biography
		provider.Education = form.Education
		provider.Experience = form.Experience
		provider.ServiceArea = form.ServiceArea
		if uploadLogo != nil {
			provider.SetImgLogo(uploadLogo.GetFile())
		}
		ctx, err = SaveProvider(ctx, s.getDB(), provider.Provider)
		if err != nil {
			logger.Errorw("save provider", "error", err, "provider", provider)
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//save the service
		svc := s.createService(provider, &form.Service)
		if svcID != uuid.Nil {
			svc.ID = &svcID
		}
		ctx, err = SaveService(ctx, s.getDB(), provider.Provider, svc, now)
		if err != nil {
			logger.Errorw("save service", "error", err, "service", svc, "id", provider.ID)
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}
		provider.ServiceCreated = &now

		//initalize the schedule
		scheduleDuration, _ := strconv.ParseInt(form.ScheduleDuration, 10, 32)
		err = s.createSchedule(provider, now, !checkedMon, !checkedTue, !checkedWed, !checkedThu, !checkedFri, !checkedSat, !checkedSun, form.Time, int(scheduleDuration))
		if err != nil {
			logger.Errorw("create schedule", "id", userID)
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}
		ctx, err = SaveProvider(ctx, s.getDB(), provider.Provider)
		if err != nil {
			logger.Errorw("save provider", "error", err, "id", provider.ID)
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//queue the welcome email if necessary
		if sendWelcome {
			ctx, err = s.queueEmailWelcome(ctx, provider)
			if err != nil {
				logger.Errorw("queue email welcome", "error", err)
				data[TplParamErr] = GetErrText(Err)
				s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
				return
			}
		}
		http.Redirect(w, r.WithContext(ctx), URISignUpLink, http.StatusSeeOther)
	}
}

//handle the signup pricing page
func (s *Server) handleProviderSignUpPricing() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, _ := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateProvider(ctx, "pricing.html")
		})
		data, _ := s.createTemplateDataProvider(r.WithContext(ctx))

		//probe for a type
		signUpType := r.FormValue(URLParams.Type)
		data[TplParamTypeSignUp] = signUpType
		s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
	}
}

//handle the signup success page
func (s *Server) handleProviderSignUpSuccess() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, _ := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateProvider(ctx, "signup-success.html")
		})
		data, _ := s.createTemplateDataProvider(r.WithContext(ctx))

		//probe for an existing provider and use the existing information if possible
		ctx, provider, ok := s.loadProvider(w, r.WithContext(ctx))
		if !ok {
			return
		}
		data[TplParamProvider] = provider
		s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
	}
}

//handle the sitemap
func (s *Server) handleProviderSiteMap() http.HandlerFunc {
	var o sync.Once
	var data []byte
	return func(w http.ResponseWriter, r *http.Request) {
		o.Do(func() {
			var err error
			file := path.Join(BaseWebAssetPath, "sitemap.xml")
			data, err = ioutil.ReadFile(file)
			if err != nil {
				panic(errors.Wrap(err, fmt.Sprintf("load file: %s", file)))
			}
		})
		w.Header().Set("Content-Type", "application/xml")
		w.Write(data)
	}
}

//handle the support page
func (s *Server) handleProviderSupport() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, _ := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateProvider(ctx, "support.html")
		})
		data, _ := s.createTemplateDataProvider(r.WithContext(ctx))
		data[TplParamActiveNav] = URIAbout
		s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
	}
}

//handle the provider terms page
func (s *Server) handleProviderTerms() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, _ := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateProvider(ctx, "terms_of_use.html")
		})
		data, _ := s.createTemplateDataProvider(r.WithContext(ctx))
		s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
	}
}

//handle the zoom support page
func (s *Server) handleProviderZoomSupport() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, _ := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateProvider(ctx, "zoom-support.html")
		})
		data, _ := s.createTemplateDataProvider(r.WithContext(ctx))
		data[TplParamActiveNav] = URIAbout
		s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
	}
}
