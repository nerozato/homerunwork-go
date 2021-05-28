package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

//number of months so show when showing all
const ordersAllMonths = 2

//Breadcrumb : definition of a breadcrumb
type breadcrumb struct {
	Name string
	URL  string
}

//load a dashboard web template
func (s *Server) loadWebTemplateDashboard(ctx context.Context, templateFile string) *template.Template {
	files := []string{path.Join(BaseWebTemplatePathDashboard, "base.html"), path.Join(BaseWebTemplatePathDashboard, templateFile)}
	tpl, err := template.New(path.Base(files[0])).Funcs(s.createTemplateFuncs()).ParseFiles(files...)
	if err != nil {
		panic(errors.Wrap(err, fmt.Sprintf("parse template dashboard: %s", templateFile)))
	}
	return tpl
}

//create basic dashboard template data
func (s *Server) createTemplateDataDashboard(w http.ResponseWriter, r *http.Request, tpl *template.Template, requiresAdmin bool) (context.Context, *providerUI, templateData, map[string]string, bool) {
	ctx, _ := GetLogger(s.getCtx(r))
	data := s.createTemplateData(r.WithContext(ctx))

	//probe for a provider
	ctx, provider, ok := s.loadProvider(w, r.WithContext(ctx))
	if !ok {
		return ctx, nil, nil, nil, false
	}

	//check if sign-up was completed
	if provider == nil {
		http.Redirect(w, r.WithContext(ctx), URISignUpMain, http.StatusSeeOther)
		return ctx, nil, nil, nil, false
	} else if provider.ServiceCreated == nil {
		http.Redirect(w, r.WithContext(ctx), URISignUpMain, http.StatusSeeOther)
		return ctx, nil, nil, nil, false
	} else if provider.GetSchedule() == nil {
		http.Redirect(w, r.WithContext(ctx), URISignUpMain, http.StatusSeeOther)
		return ctx, nil, nil, nil, false
	}
	data[TplParamProvider] = provider

	//check permissions
	ok = s.checkPermission(w, r.WithContext(ctx), provider, requiresAdmin)
	if !ok {
		return ctx, nil, nil, nil, false
	}

	//populate the data
	data[TplParamURLDefaultProvider] = BaseClientURL
	data[TplParamURLFacebookProvider] = provider.URLFacebook
	data[TplParamURLInstagramProvider] = provider.URLInstagram
	data[TplParamURLLinkedInProvider] = provider.URLLinkedIn
	data[TplParamURLTwitterProvider] = provider.URLTwitter
	data[TplParamURLWebProvider] = provider.URLWeb

	//add the errors
	errs := make(map[string]string)
	data[TplParamErrs] = errs
	return ctx, provider, data, errs, true
}

//handle the about page
func (s *Server) handleDashboardAbout() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateDashboard(ctx, "about.html")
		})
		ctx, provider, data, errs, ok := s.createTemplateDataDashboard(w, r.WithContext(ctx), tpl, true)
		if !ok {
			return
		}

		//setup the breadcrumbs
		breadcrumbs := []breadcrumb{
			{"Our Story", ""},
		}
		data[TplParamBreadcrumbs] = breadcrumbs
		data[TplParamActiveNav] = provider.GetURLAbout()
		data[TplParamFormAction] = provider.GetURLAbout()
		data[TplParamClientView] = r.FormValue(URLParams.Client)

		//check the method
		if r.Method == http.MethodGet {
			data[TplParamText] = provider.About
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//handle the input
		text := r.FormValue(URLParams.Text)

		//treat "<br>" by itself as empty
		if text == "<br>" {
			text = ""
		}

		//prepare the data
		data[TplParamText] = text

		//validate the data
		form := TextLongForm{
			Text: text,
		}
		ok = s.validateForm(w, r.WithContext(ctx), tpl, data, errs, form, true)
		if !ok {
			return
		}

		//populate from the form
		provider.About = form.Text

		//save the provider
		ctx, err := SaveProvider(ctx, s.getDB(), provider.Provider)
		if err != nil {
			logger.Errorw("save provider", "error", err, "provider", provider)
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//success
		s.SetCookieMsg(w, MsgUpdateSuccess)
		url := s.checkClientView(data, provider, provider.GetURLAbout())
		http.Redirect(w, r.WithContext(ctx), url, http.StatusSeeOther)
	}
}

//handle the account page
func (s *Server) handleDashboardAccount() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template

	//steps on the page
	steps := struct {
		StepUpd    string
		StepVerify string
	}{
		StepUpd:    "stepUpd",
		StepVerify: "stepVerify",
	}
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateDashboard(ctx, "my-account.html")
		})
		ctx, provider, data, errs, ok := s.createTemplateDataDashboard(w, r.WithContext(ctx), tpl, true)
		if !ok {
			return
		}

		//setup the breadcrumbs
		breadcrumbs := []breadcrumb{
			{"Account", ""},
		}
		data[TplParamBreadcrumbs] = breadcrumbs
		data[TplParamActiveNav] = provider.GetURLAccount()
		data[TplParamFormAction] = provider.GetURLAccount()
		data[TplParamSteps] = steps

		//check the method
		if r.Method == http.MethodGet {
			data[TplParamDisablePhone] = provider.User.DisablePhone
			data[TplParamNameFirst] = provider.User.FirstName
			data[TplParamNameLast] = provider.User.LastName
			data[TplParamPhone] = provider.User.Phone
			data[TplParamTimeZone] = provider.User.TimeZone
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//handle the input
		disablePhone := r.FormValue(URLParams.DisablePhone) == "on"
		firstName := r.FormValue(URLParams.FirstName)
		lastName := r.FormValue(URLParams.LastName)
		phone := r.FormValue(URLParams.Phone)
		pwd := r.FormValue(URLParams.Password)
		step := r.FormValue(URLParams.Step)
		timeZone := r.FormValue(URLParams.TimeZone)

		//ignore certain fields if oauth
		if provider.User.IsOAuth {
			firstName = provider.User.FirstName
			lastName = provider.User.LastName
			pwd = ""
		}

		//execute the correct operation
		switch step {
		case steps.StepVerify:
			//send a request email to verify the email
			ctx, err := s.queueEmailVerify(ctx, provider.User)
			if err != nil {
				logger.Errorw("queue email verify", "error", err)
				s.SetCookieErr(w, ErrEmailVerifyEmail, provider.User.Email)
			} else {
				s.SetCookieMsg(w, MsgEmailVerifySent)
			}
			http.Redirect(w, r.WithContext(ctx), provider.GetURLAccount(), http.StatusSeeOther)
			return
		case steps.StepUpd:
			//prepare the data
			data[TplParamDisablePhone] = disablePhone
			data[TplParamNameFirst] = firstName
			data[TplParamNameLast] = lastName
			data[TplParamPhone] = phone
			data[TplParamTimeZone] = timeZone

			//validate the data
			form := UserUpdateForm{
				DisablePhone: disablePhone,
				Password:     Secret(pwd),
				Phone:        FormatPhone(phone),
				TimeZoneForm: TimeZoneForm{
					TimeZone: timeZone,
				},
				UserForm: UserForm{
					FirstName: firstName,
					LastName:  lastName,
				},
			}
			ok = s.validateForm(w, r.WithContext(ctx), tpl, data, errs, form, true)
			if !ok {
				return
			}

			//update the hours to use the new timezone if necessary
			if provider.User.TimeZone != form.TimeZone {
				now := data[TplParamCurrentTime].(time.Time)
				schedule := provider.GetSchedule()
				if schedule == nil {
					logger.Errorw("no schedule", "id", provider.ID)
					data[TplParamErr] = GetErrText(Err)
					s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
					return
				}
				schedule.Adjust(now, provider.User.TimeZone, form.TimeZone)
				ctx, err := SaveProvider(ctx, s.getDB(), provider.Provider)
				if err != nil {
					logger.Errorw("save provider", "error", err, "id", provider.ID)
					data[TplParamErr] = GetErrText(Err)
					s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
					return
				}
			}

			//populate from the form
			user := provider.User
			user.DisablePhone = form.DisablePhone
			user.FirstName = form.FirstName
			user.LastName = form.LastName
			user.Phone = form.Phone
			user.TimeZone = form.TimeZone

			//save the user
			ctx, err := SaveUser(ctx, s.getDB(), user, form.Password)
			if err != nil {
				logger.Errorw("save user", "error", err, "user", user)
				data[TplParamErr] = GetErrText(Err)
				s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
				return
			}

			//success
			s.SetCookieMsg(w, MsgUpdateSuccess)
			http.Redirect(w, r.WithContext(ctx), provider.GetURLAccount(), http.StatusSeeOther)
			return
		default:
			logger.Errorw("invalid step", "id", provider.ID, "step", step)
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}
	}
}

//handle the booking add page
func (s *Server) handleDashboardBookingAdd() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateDashboard(ctx, "order-add.html")
		})
		ctx, provider, data, errs, ok := s.createTemplateDataDashboard(w, r.WithContext(ctx), tpl, false)
		if !ok {
			return
		}

		//setup the breadcrumbs
		breadcrumbs := []breadcrumb{
			{"Orders", provider.GetURLBookings()},
			{"Add Order", ""},
		}
		data[TplParamBreadcrumbs] = breadcrumbs
		data[TplParamActiveNav] = provider.GetURLBookings()
		data[TplParamFormAction] = provider.GetURLBookingAdd()

		//read the form
		clientIDStr := r.FormValue(URLParams.ClientID)
		code := r.FormValue(URLParams.Code)
		dateStr := r.FormValue(URLParams.Date)
		desc := r.FormValue(URLParams.Desc)
		email := r.FormValue(URLParams.Email)
		freqStr := r.FormValue(URLParams.Freq)
		location := r.FormValue(URLParams.Location)
		name := r.FormValue(URLParams.Name)
		phone := r.FormValue(URLParams.Phone)
		svcIDStr := r.FormValue(URLParams.SvcID)
		timeStr := r.FormValue(URLParams.Time)
		userIDStr := r.FormValue(URLParams.UserID)

		//prepare the data
		data[TplParamClientID] = clientIDStr
		data[TplParamCode] = code
		data[TplParamDesc] = desc
		data[TplParamEmail] = email
		data[TplParamFreq] = freqStr
		data[TplParamLocation] = location
		data[TplParamName] = name
		data[TplParamPhone] = phone
		data[TplParamUserID] = userIDStr

		//load the services
		ctx, svcs, ok := s.loadTemplateServices(w, r.WithContext(ctx), tpl, data, provider)
		if !ok {
			return
		}

		//load the clients
		ctx, clients, ok := s.loadTemplateClients(w, r.WithContext(ctx), tpl, data, provider.ID)
		if !ok {
			return
		}

		//find the selected client
		var selectedClient *Client
		if clientIDStr != "" {
			for _, client := range clients {
				if client.ID.String() == clientIDStr {
					selectedClient = client
					break
				}
			}
		}

		//pre-select the service if there's only one
		if svcIDStr == "" && len(svcs) > 0 {
			svcIDStr = svcs[0].ID.String()
		}
		data[TplParamSvcID] = svcIDStr

		//validate the id
		var svcStartDateStr string
		now := data[TplParamCurrentTime].(time.Time)
		timeZone := provider.User.TimeZone
		var svc *serviceUI
		if svcIDStr != "" {
			//find the service in the list
			for _, s := range svcs {
				if s.ID.String() == svcIDStr {
					svc = s
				}
			}
			if svc == nil {
				logger.Errorw("invalid service id", "id", svcIDStr)
				data[TplParamErr] = GetErrText(Err)
				s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
				return
			}
			data[TplParamSvc] = svc

			//load the service times
			_, svcStartDate, firstAvailableTime, svcTimes, ok := s.loadTemplateServiceTimes(w, r.WithContext(ctx), tpl, data, errs, provider, svc, dateStr, now, false)
			if !ok {
				return
			}
			svcStartDateStr = FormatDateLocal(svcStartDate, timeZone)
			data[TplParamSvcStartDate] = svcStartDateStr
			data[TplParamSvcBusyTimes] = svcTimes

			//default the time if necessary
			if timeStr == "" && !firstAvailableTime.IsZero() {
				timeStr = strconv.FormatInt(firstAvailableTime.Unix(), 10)
			}
		} else {
			logger.Errorw("no service id")
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//default the date if necessary
		if dateStr == "" {
			dateStr = svcStartDateStr
		}
		data[TplParamDate] = dateStr
		data[TplParamTime] = timeStr

		//load the users
		var err error
		var users []*ProviderUser
		if provider.IsAdmin() {
			ctx, users, err = ListProviderUsersByProviderID(ctx, s.getDB(), provider.ID, true)
			if err != nil {
				logger.Errorw("list users", "error", err, "id", provider.ID)
				data[TplParamErr] = GetErrText(Err)
				s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
				return
			}
			data[TplParamUsers] = users
		}

		//check the method
		if r.Method == http.MethodGet {
			data[TplParamDesc] = svc.Note

			//determine which location to use
			if location == "" {
				location = svc.Location
				if selectedClient != nil && svc.LocationType.IsLocationClient() && selectedClient.Location != "" {
					location = selectedClient.Location
				}
			}
			data[TplParamLocation] = location
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//validate the booking
		form := &ClientBookingForm{
			ServiceID:       svcIDStr,
			ClientID:        clientIDStr,
			Email:           strings.TrimSpace(email),
			Name:            name,
			Phone:           FormatPhone(phone),
			ProviderNote:    desc,
			ProviderNoteSet: true,
			Freq:            freqStr,
			FreqSet:         true,
			ClientCreated:   false,
			Confirmed:       true,
			Location:        location,
			ClientBookingDateTimeForm: ClientBookingDateTimeForm{
				TimeUnixForm: TimeUnixForm{
					Time: timeStr,
				},
				TimeZoneForm: TimeZoneForm{
					TimeZone: timeZone,
				},
			},
			Code: code,
		}
		ok = s.validateForm(w, r.WithContext(ctx), tpl, data, errs, form, true)
		if !ok {
			return
		}

		//check for a selected user
		var providerUser *ProviderUser
		if userIDStr != "" {
			userID := uuid.FromStringOrNil(userIDStr)
			if userID == uuid.Nil {
				logger.Errorw("parse user id", "id", userIDStr)
				errs[string(FieldErrUserID)] = GetFieldErrText(string(FieldErrUserID))
				s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
				return
			}
			for _, user := range users {
				if user.ID.String() == userIDStr {
					providerUser = user
					break
				}
			}
		}

		//create the booking, prioritizing the entered client information
		book, ok := s.saveBooking(w, r.WithContext(ctx), tpl, data, errs, provider, providerUser, svc, nil, now, false, form, false)
		if !ok {
			return
		}
		http.Redirect(w, r.WithContext(ctx), provider.GetURLBookingAddSuccess(book.ID), http.StatusSeeOther)
	}
}

//handle the booking add success page
func (s *Server) handleDashboardBookingAddSuccess() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, _ := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateDashboard(ctx, "order-add-success.html")
		})
		ctx, provider, data, errs, ok := s.createTemplateDataDashboard(w, r.WithContext(ctx), tpl, false)
		if !ok {
			return
		}
		data[TplParamActiveNav] = provider.GetURLBookings()

		//load the booking
		idStr := r.FormValue(URLParams.BookID)
		ctx, _, ok = s.loadTemplateBook(w, r.WithContext(ctx), tpl, data, errs, idStr, false, false)
		if !ok {
			s.SetCookieErr(w, Err)
			http.Redirect(w, r.WithContext(ctx), provider.GetURLBookings(), http.StatusSeeOther)
			return
		}

		//setup the breadcrumbs
		breadcrumbs := []breadcrumb{
			{"Orders", provider.GetURLBookings()},
			{"Add Order", ""},
		}
		data[TplParamBreadcrumbs] = breadcrumbs
		s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
	}
}

//handle the other services page
func (s *Server) handleDashboardAddOns() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template

	//steps on the page
	steps := struct {
		StepGoogleTracking    string
		StepGoogleTrackingDel string
		StepGoogleTrackingUpd string
		StepZoomDel           string
		StepZoomUpd           string
	}{
		StepGoogleTracking:    "stepGoogleTracking",
		StepGoogleTrackingDel: "stepGoogleTrackingDel",
		StepGoogleTrackingUpd: "stepGoogleTrackingUpd",
		StepZoomDel:           "stepZoomDel",
		StepZoomUpd:           "stepZoomUpd",
	}
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateDashboard(ctx, "add-ons.html")
		})
		ctx, provider, data, errs, ok := s.createTemplateDataDashboard(w, r.WithContext(ctx), tpl, false)
		if !ok {
			return
		}

		//setup the breadcrumbs
		breadcrumbs := []breadcrumb{
			{"Add-Ons", ""},
		}
		data[TplParamBreadcrumbs] = breadcrumbs
		data[TplParamActiveNav] = provider.GetURLAddOns()
		data[TplParamFormAction] = provider.GetURLAddOns()
		data[TplParamSteps] = steps

		//handle the input
		id := r.FormValue(URLParams.ID)
		step := r.FormValue(URLParams.Step)

		//prepare the data
		data[TplParamID] = id
		data[TplParamStep] = step

		//check the method
		if r.Method == http.MethodGet {
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//execute the correct operation
		switch step {
		case steps.StepGoogleTracking:
			ok := s.checkPermission(w, r.WithContext(ctx), provider, true)
			if !ok {
				return
			}
			if provider.GoogleTrackingID != nil {
				data[TplParamID] = provider.GoogleTrackingID
			} else {
				data[TplParamID] = ""
			}
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		case steps.StepGoogleTrackingDel:
			ok := s.checkPermission(w, r.WithContext(ctx), provider, true)
			if !ok {
				return
			}

			//delete the tracking id
			provider.GoogleTrackingID = nil
			ctx, err := SaveProvider(ctx, s.getDB(), provider.Provider)
			if err != nil {
				logger.Errorw("save provider", "error", err, "provider", provider)
				data[TplParamErr] = GetErrText(Err)
				s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
				return
			}
		case steps.StepGoogleTrackingUpd:
			ok := s.checkPermission(w, r.WithContext(ctx), provider, true)
			if !ok {
				return
			}

			//validate the tracking id
			form := &GoogleTrackingIDForm{
				ID: id,
			}
			ok = s.validateForm(w, r.WithContext(ctx), tpl, data, errs, form, true)
			if !ok {
				return
			}

			//save the provider
			provider.GoogleTrackingID = &form.ID
			ctx, err := SaveProvider(ctx, s.getDB(), provider.Provider)
			if err != nil {
				logger.Errorw("save provider", "error", err, "provider", provider)
				data[TplParamErr] = GetErrText(Err)
				s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
				return
			}
		case steps.StepZoomDel:
			user := provider.GetUser()
			user.ZoomToken = nil
			user.ZoomUser = nil

			//save the user
			ctx, err := SaveUser(ctx, s.getDB(), user, "")
			if err != nil {
				logger.Errorw("save provider", "error", err, "id", user.ID)
				data[TplParamErr] = GetErrText(Err)
				s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
				return
			}
		case steps.StepZoomUpd:
			s.invokeHdlrGet(s.handleZoomLogin(), w, r.WithContext(ctx))
			return
		default:
			logger.Errorw("invalid step", "step", step)
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//success
		s.SetCookieMsg(w, MsgUpdateSuccess)
		http.Redirect(w, r.WithContext(ctx), provider.GetURLAddOns(), http.StatusSeeOther)
	}
}

//handle the booking edit page
func (s *Server) handleDashboardBookingEdit() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template

	//steps on the page
	steps := struct {
		StepConfirm string
		StepUpd     string
		StepUpdAll  string
	}{
		StepConfirm: "stepConfirm",
		StepUpd:     "stepUpd",
		StepUpdAll:  "stepUpdAll",
	}
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateDashboard(ctx, "order-edit.html")
		})
		ctx, provider, data, errs, ok := s.createTemplateDataDashboard(w, r.WithContext(ctx), tpl, false)
		if !ok {
			return
		}
		data[TplParamActiveNav] = provider.GetURLBookings()
		data[TplParamFormAction] = provider.GetURLBookingEdit()
		data[TplParamSteps] = steps

		//handle the input
		code := r.FormValue(URLParams.Code)
		dateStr := r.FormValue(URLParams.Date)
		desc := r.FormValue(URLParams.Desc)
		idStr := r.FormValue(URLParams.BookID)
		location := r.FormValue(URLParams.Location)
		svcIDStr := r.FormValue(URLParams.SvcID)
		step := r.FormValue(URLParams.Step)
		timeStr := r.FormValue(URLParams.Time)
		timeZone := provider.User.TimeZone

		//load the booking
		ctx, book, ok := s.loadTemplateBook(w, r.WithContext(ctx), tpl, data, errs, idStr, true, false)
		if !ok {
			s.SetCookieErr(w, Err)
			http.Redirect(w, r.WithContext(ctx), provider.GetURLBookings(), http.StatusSeeOther)
			return
		}

		//check permissions
		ok = s.checkOwner(w, r.WithContext(ctx), provider, book)
		if !ok {
			return
		}

		//default if not set or if client-created
		if svcIDStr == "" || book.ClientCreated {
			svcIDStr = book.Service.ID.String()
		}
		if dateStr == "" || book.ClientCreated {
			dateStr = FormatDateLocal(book.TimeFrom, timeZone)
		}
		if timeStr == "" || book.ClientCreated {
			timeStr = strconv.FormatInt(book.TimeFrom.Unix(), 10)
		}

		//prepare the data
		data[TplParamCode] = code
		data[TplParamDate] = dateStr
		data[TplParamDesc] = desc
		data[TplParamLocation] = location
		data[TplParamRecurrenceFreq] = book.FormatRecurrenceFreq()
		data[TplParamSvcID] = svcIDStr
		data[TplParamTime] = timeStr

		//setup the breadcrumbs
		breadcrumbs := []breadcrumb{
			{"Orders", provider.GetURLBookings()},
			{"Edit Order", ""},
		}
		data[TplParamBreadcrumbs] = breadcrumbs

		//load the services
		ctx, svcs, ok := s.loadTemplateServices(w, r.WithContext(ctx), tpl, data, provider)
		if !ok {
			return
		}

		//find the service in the list
		var svc *serviceUI
		for _, s := range svcs {
			if s.ID.String() == svcIDStr {
				svc = s
			}
		}
		if svc == nil {
			logger.Errorw("invalid service id", "id", book.Service.ID)
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}
		data[TplParamSvc] = svc

		//check the method
		now := data[TplParamCurrentTime].(time.Time)
		if r.Method == http.MethodGet {
			//load the service times
			ctx, svcStartDate, _, svcTimes, ok := s.loadTemplateServiceTimes(w, r.WithContext(ctx), tpl, data, errs, provider, svc, dateStr, now, false)
			if !ok {
				return
			}
			svcStartDateStr := FormatDateLocal(svcStartDate, timeZone)
			data[TplParamSvcStartDate] = svcStartDateStr
			data[TplParamSvcBusyTimes] = svcTimes

			//default the date if necessary
			if dateStr == "" {
				dateStr = svcStartDateStr
				data[TplParamDate] = dateStr
			}
			data[TplParamCode] = book.CouponCode
			data[TplParamDesc] = book.ProviderNote
			data[TplParamLocation] = book.Location
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//execute the correct operation
		changeAllFollowing := false
		switch step {
		case steps.StepUpdAll:
			changeAllFollowing = true
			fallthrough
		case steps.StepUpd:
			//no edits allowed for captured bookings
			if !book.IsEditable(now) {
				http.Redirect(w, r.WithContext(ctx), book.GetURLView(), http.StatusSeeOther)
				return
			}

			//validate the booking
			form := &ClientBookingForm{
				ServiceID:       svcIDStr,
				ClientID:        book.Client.ID.String(),
				Email:           book.Client.Email,
				Name:            book.Client.Name,
				Phone:           book.Client.Phone,
				ProviderNote:    desc,
				ProviderNoteSet: true,
				ClientCreated:   book.ClientCreated,
				Confirmed:       true,
				Location:        location,
				ClientBookingDateTimeForm: ClientBookingDateTimeForm{
					TimeUnixForm: TimeUnixForm{
						Time: timeStr,
					},
					TimeZoneForm: TimeZoneForm{
						TimeZone: timeZone,
					},
				},
				Code: code,
			}
			ok = s.validateForm(w, r.WithContext(ctx), tpl, data, errs, form, true)
			if !ok {
				return
			}

			//update the booking
			book, ok = s.saveBooking(w, r.WithContext(ctx), tpl, data, errs, provider, book.ProviderUser, svc, book, now, changeAllFollowing, form, false)
			if !ok {
				return
			}
			http.Redirect(w, r.WithContext(ctx), provider.GetURLBookingEditSuccess(book.ID), http.StatusSeeOther)
			return
		case steps.StepConfirm:
			if !book.Confirmed {
				//validate the note
				form := &ClientConfirmForm{
					Text: desc,
					Code: code,
				}
				ok = s.validateForm(w, r.WithContext(ctx), tpl, data, errs, form, true)
				if !ok {
					return
				}

				//mark the booking confirmed
				book.SetCouponCode(form.Code)
				book.SetProviderNote(form.Text)
				ctx, err := MarkBookingConfirmed(ctx, s.getDB(), svc.Service, book.Booking, now)
				if err != nil {
					logger.Errorw("confirm booking", "error", err)
					data[TplParamErr] = GetErrText(Err)
					s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
					return
				}

				//queue the emails
				ctx, err = s.queueEmailsBookingConfirm(ctx, book)
				if err != nil {
					logger.Errorw("queue email booking confirm", "error", err)
					data[TplParamErr] = GetErrText(Err)
					s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
					return
				}
			}
			http.Redirect(w, r.WithContext(ctx), provider.GetURLBookingEditSuccess(book.ID), http.StatusSeeOther)
			return
		}
		logger.Errorw("invalid step", "id", book.ID, "step", step)
		data[TplParamErr] = GetErrText(Err)
		s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
		return
	}
}

//handle the booking cancel success page
func (s *Server) handleDashboardBookingCancelSuccess() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, _ := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateDashboard(ctx, "order-cancel-success.html")
		})
		ctx, provider, data, errs, ok := s.createTemplateDataDashboard(w, r.WithContext(ctx), tpl, false)
		if !ok {
			return
		}
		data[TplParamActiveNav] = provider.GetURLBookings()

		//load the booking
		idStr := r.FormValue(URLParams.BookID)
		ctx, _, ok = s.loadTemplateBook(w, r.WithContext(ctx), tpl, data, errs, idStr, true, true)
		if !ok {
			s.SetCookieErr(w, Err)
			http.Redirect(w, r.WithContext(ctx), provider.GetURLBookings(), http.StatusSeeOther)
			return
		}

		//setup the breadcrumbs
		breadcrumbs := []breadcrumb{
			{"Orders", provider.GetURLBookings()},
			{"Edit Order", ""},
		}
		data[TplParamBreadcrumbs] = breadcrumbs
		s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
	}
}

//handle the booking edit success page
func (s *Server) handleDashboardBookingEditSuccess() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, _ := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateDashboard(ctx, "order-edit-success.html")
		})
		ctx, provider, data, errs, ok := s.createTemplateDataDashboard(w, r.WithContext(ctx), tpl, false)
		if !ok {
			return
		}
		data[TplParamActiveNav] = provider.GetURLBookings()

		//load the booking
		idStr := r.FormValue(URLParams.BookID)
		ctx, _, ok = s.loadTemplateBook(w, r.WithContext(ctx), tpl, data, errs, idStr, true, false)
		if !ok {
			s.SetCookieErr(w, Err)
			http.Redirect(w, r.WithContext(ctx), provider.GetURLBookings(), http.StatusSeeOther)
			return
		}

		//setup the breadcrumbs
		breadcrumbs := []breadcrumb{
			{"Orders", provider.GetURLBookings()},
			{"Edit Order", ""},
		}
		data[TplParamBreadcrumbs] = breadcrumbs
		s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
	}
}

//handle the booking view page
func (s *Server) handleDashboardBookingView() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template

	//steps on the page
	steps := struct {
		StepDel        string
		StepDelAll     string
		StepMarkPaid   string
		StepMarkUnPaid string
	}{
		StepDel:        "stepDel",
		StepDelAll:     "stepDelAll",
		StepMarkPaid:   "stepMarkPaid",
		StepMarkUnPaid: "stepMarkUnPaid",
	}
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateDashboard(ctx, "order-view.html")
		})
		ctx, provider, data, errs, ok := s.createTemplateDataDashboard(w, r.WithContext(ctx), tpl, false)
		if !ok {
			return
		}
		data[TplParamActiveNav] = provider.GetURLBookings()
		data[TplParamFormAction] = provider.GetURLBookingView()
		data[TplParamSteps] = steps

		//load the booking
		idStr := r.FormValue(URLParams.BookID)
		ctx, book, ok := s.loadTemplateBook(w, r.WithContext(ctx), tpl, data, errs, idStr, true, false)
		if !ok {
			s.SetCookieErr(w, Err)
			http.Redirect(w, r.WithContext(ctx), provider.GetURLBookings(), http.StatusSeeOther)
			return
		}

		//check permissions
		ok = s.checkOwner(w, r.WithContext(ctx), provider, book)
		if !ok {
			return
		}

		//load the service
		now := data[TplParamCurrentTime].(time.Time)
		ctx, svc, ok := s.loadTemplateService(w, r.WithContext(ctx), tpl, data, provider, book.Service.ID, now)
		if !ok {
			return
		}

		//setup the breadcrumbs
		breadcrumbs := []breadcrumb{
			{"Orders", provider.GetURLBookings()},
			{"View Order", ""},
		}
		data[TplParamBreadcrumbs] = breadcrumbs

		//prepare the confirmation modal
		if book.AllowUnPay() {
			data[TplParamConfirmMsg] = GetMsgText(MsgPaymentMarkUnPaid)
			data[TplParamConfirmSubmitName] = URLParams.Step
			data[TplParamConfirmSubmitValue] = steps.StepMarkUnPaid
		} else {
			data[TplParamConfirmMsg] = GetMsgText(MsgPaymentMarkPaid)
			data[TplParamConfirmSubmitName] = URLParams.Step
			data[TplParamConfirmSubmitValue] = steps.StepMarkPaid
		}

		//check the method
		if r.Method == http.MethodGet {
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//execute the correct operation
		changeAllFollowing := false
		step := r.FormValue(URLParams.Step)
		switch step {
		case steps.StepMarkPaid:
			//save the payment
			form := &PaymentForm{
				EmailForm: EmailForm{
					Email: book.Client.Email,
				},
				NameForm: NameForm{
					Name: book.Client.Name,
				},
				Price:           strconv.FormatFloat(float64(book.ComputeServicePrice()), 'f', 2, 32),
				ClientInitiated: false,
				DirectCapture:   true,
			}
			ctx, _, err := s.savePaymentBooking(ctx, provider, book, form, now)
			if err != nil {
				logger.Errorw("save payment", "error", err)
				data[TplParamErr] = GetErrText(Err)
				s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
				return
			}
		case steps.StepMarkUnPaid:
			//sanity check the operation
			if !book.AllowUnPay() || book.Payment == nil {
				logger.Errorw("invalid mark unpaid")
				data[TplParamErr] = GetErrText(Err)
				s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
				return
			}

			//mark the payment as unpaid
			ctx, err := UpdatePaymentUnPaid(ctx, s.getDB(), book.Payment.ID)
			if err != nil {
				logger.Errorw("mark payment unpaid", "error", err)
				data[TplParamErr] = GetErrText(Err)
				s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
				return
			}
		case steps.StepDelAll:
			changeAllFollowing = true
			fallthrough
		case steps.StepDel:
			//delete the booking
			now := data[TplParamCurrentTime].(time.Time)
			ok = s.cancelServiceBooking(w, r.WithContext(ctx), tpl, data, errs, provider, svc, book, now, changeAllFollowing)
			if !ok {
				return
			}
			http.Redirect(w, r.WithContext(ctx), provider.GetURLBookingCancelSuccess(book.ID), http.StatusSeeOther)
			return
		default:
			logger.Errorw("invalid step", "id", book.ID, "step", step)
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//success
		s.SetCookieMsg(w, MsgUpdateSuccess)
		http.Redirect(w, r.WithContext(ctx), provider.GetURLBookings(), http.StatusSeeOther)
	}
}

//handle the bookings page
func (s *Server) handleDashboardBookings() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateDashboard(ctx, "orders.html")
		})
		ctx, provider, data, errs, ok := s.createTemplateDataDashboard(w, r.WithContext(ctx), tpl, false)
		if !ok {
			return
		}
		user := provider.GetProviderUser()

		//setup the breadcrumbs
		breadcrumbs := []breadcrumb{
			{"Orders", ""},
		}
		data[TplParamBreadcrumbs] = breadcrumbs
		data[TplParamActiveNav] = provider.GetURLBookings()
		data[TplParamFormAction] = provider.GetURLBookings()

		//url for the calendar
		url := createDashboardAPIURL(URIOrdersCalendar)
		data[TplParamURL] = url

		//load the counts
		now := data[TplParamCurrentTime].(time.Time)
		ctx, countNew, err := CountBookingsByProviderIDAndFilter(ctx, s.getDB(), provider.ID, user, BookingFilterNew, "", now)
		if err != nil {
			logger.Errorw("count bookings new", "error", err, "id", provider.ID)
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}
		data[TplParamCountNew] = countNew
		ctx, countUnPaid, err := CountBookingsByProviderIDAndFilter(ctx, s.getDB(), provider.ID, user, BookingFilterUnPaid, "", now)
		if err != nil {
			logger.Errorw("count bookings unpaid", "error", err, "id", provider.ID)
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}
		data[TplParamCountUnPaid] = countUnPaid
		ctx, countUpcoming, err := CountBookingsByProviderIDAndFilter(ctx, s.getDB(), provider.ID, user, BookingFilterUpcoming, BookingFilterAll, now)
		if err != nil {
			logger.Errorw("count bookings upcoming", "error", err, "id", provider.ID)
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}
		data[TplParamCountUpcoming] = countUpcoming

		//read the form
		timeStr := r.FormValue(URLParams.Time)
		filterStr := r.FormValue(URLParams.Filter)
		filterSubStr := r.FormValue(URLParams.FilterSub)
		if filterSubStr == "" {
			//default to the new filter if possible
			if countNew > 0 {
				filterSubStr = string(BookingFilterNew)
			} else {
				filterSubStr = string(BookingFilterAll)
			}
		}

		//prepare the data
		data[TplParamFilter] = filterStr
		data[TplParamFilterSub] = filterSubStr
		data[TplParamTime] = timeStr

		//validate the filter
		if filterStr != "" {
			filter, err := ParseBookingFilter(filterStr)
			if err != nil {
				logger.Errorw("parse filter", "error", err, "filter", filterStr)
				data[TplParamErr] = GetErrText(Err)
				s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			}
			filterSub, err := ParseBookingFilter(filterSubStr)
			if err != nil {
				logger.Errorw("parse filter", "error", err, "filter", filterSubStr)
				data[TplParamErr] = GetErrText(Err)
				s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			}

			//check permissions
			switch filter {
			case BookingFilterUpcoming:
			default:
				ok := s.checkPermission(w, r.WithContext(ctx), provider, true)
				if !ok {
					return
				}
			}

			//validate the time
			var timeCurrent time.Time
			if timeStr != "" {
				form := TimeUnixForm{
					Time: timeStr,
				}
				ok = s.validateForm(w, r.WithContext(ctx), tpl, data, errs, form, false)
				if !ok {
					s.SetCookieErr(w, Err)
					http.Redirect(w, r.WithContext(ctx), provider.GetURLBookings(), http.StatusSeeOther)
					return
				}
				timeCurrent = ParseTimeUnixLocal(form.Time, provider.User.TimeZone)
				if timeCurrent.IsZero() {
					errs[string(FieldErrDate)] = GetFieldErrText(string(FieldErrDate))
					s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
					return
				}
			} else {
				timeCurrent = data[TplParamCurrentTime].(time.Time)
			}
			timeNext := timeCurrent.AddDate(0, ordersAllMonths, 0)

			//load the bookings
			ctx, books, err := ListBookingsByProviderIDAndFilter(ctx, s.getDB(), provider.ID, user, filter, filterSub, timeCurrent)
			if err != nil {
				logger.Errorw("load bookings", "error", err, "id", provider.ID, "from", timeCurrent, "to", timeNext)
				data[TplParamErr] = GetErrText(Err)
				s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
				return
			}

			//organize the bookings by day of week
			booksByDate := &bookingsByDate{}
			for _, book := range books {
				bookUI := s.createBookingUI(book)
				booksByDate.AddBooking(bookUI, provider.User.TimeZone)
			}
			data[TplParamBooks] = booksByDate
		}
		s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
	}
}

//handle the calendars page
func (s *Server) handleDashboardCalendars() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template

	//steps on the page
	steps := struct {
		StepGoogleCalDel string
		StepGoogleCalUpd string
	}{
		StepGoogleCalDel: "stepGoogleCalDel",
		StepGoogleCalUpd: "stepGoogleCalUpd",
	}
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateDashboard(ctx, "calendars.html")
		})
		ctx, provider, data, _, ok := s.createTemplateDataDashboard(w, r.WithContext(ctx), tpl, true)
		if !ok {
			return
		}

		//setup the breadcrumbs
		breadcrumbs := []breadcrumb{
			{"Calendars", ""},
		}
		data[TplParamBreadcrumbs] = breadcrumbs
		data[TplParamActiveNav] = provider.GetURLCalendar()
		data[TplParamFormAction] = provider.GetURLCalendar()
		data[TplParamSteps] = steps

		//handle the input
		step := r.FormValue(URLParams.Step)

		//prepare the data
		data[TplParamStep] = step

		//check the method
		if r.Method == http.MethodGet {
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//execute the correct operation
		switch step {
		case steps.StepGoogleCalDel:
			//save the user
			ctx, user, err := LoadUserByID(ctx, s.db, provider.User.ID)
			if err != nil {
				logger.Errorw("google user load", "error", err, "id", provider.User.ID)
				data[TplParamErr] = GetErrText(Err)
				s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
				return
			}
			user.GoogleCalendarToken = nil
			ctx, err = s.saveUser(w, r.WithContext(ctx), user, "")
			if err != nil {
				logger.Errorw("save user", "error", err, "id", user.ID)
				data[TplParamErr] = GetErrText(Err)
				s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
				return
			}
		case steps.StepGoogleCalUpd:
			s.invokeHdlrGet(s.handleGoogleCalLogin(), w, r.WithContext(ctx))
			return
		default:
			logger.Errorw("invalid step", "step", step)
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//success
		s.SetCookieMsg(w, MsgUpdateSuccess)
		http.Redirect(w, r.WithContext(ctx), provider.GetURLCalendar(), http.StatusSeeOther)
		s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
	}
}

//handle the campaign add step 1 page
func (s *Server) handleDashboardCampaignAddStep1() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateDashboard(ctx, "campaign-add-step1.html")
		})
		ctx, provider, data, errs, ok := s.createTemplateDataDashboard(w, r.WithContext(ctx), tpl, true)
		if !ok {
			return
		}

		//setup the breadcrumbs
		breadcrumbs := []breadcrumb{
			{"Campaigns", provider.GetURLCampaigns()},
			{"Add Campaign", ""},
		}
		data[TplParamBreadcrumbs] = breadcrumbs
		data[TplParamActiveNav] = provider.GetURLCampaigns()
		data[TplParamFormAction] = provider.GetURLCampaignAddStep1(nil)

		//handle the input
		ageMinStr := r.FormValue(URLParams.AgeMin)
		ageMaxStr := r.FormValue(URLParams.AgeMax)
		budgetStr := r.FormValue(URLParams.Budget)
		gender := r.FormValue(URLParams.Gender)
		idStr := r.FormValue(URLParams.ID)
		interests := r.FormValue(URLParams.Interests)
		locations := r.FormValue(URLParams.Locations)
		startStr := r.FormValue(URLParams.Start)
		endStr := r.FormValue(URLParams.End)
		svcIDStr := r.FormValue(URLParams.SvcID)
		text := r.FormValue(URLParams.Text)
		timeZone := r.FormValue(URLParams.TimeZone)

		//prepare the data
		data[TplParamAgeMin] = ageMinStr
		data[TplParamAgeMax] = ageMaxStr
		data[TplParamBudget] = budgetStr
		data[TplParamGender] = gender
		data[TplParamID] = idStr
		data[TplParamInterests] = interests
		data[TplParamLocations] = locations
		data[TplParamStart] = startStr
		data[TplParamEnd] = endStr
		data[TplParamSvcID] = svcIDStr
		data[TplParamText] = text

		//load the campaign if necessary
		var campaign *campaignUI
		if idStr != "" {
			var ok bool
			_, campaign, ok = s.loadTemplateCampaign(w, r.WithContext(ctx), tpl, data, provider, true)
			if !ok {
				return
			}
		}

		//load the services
		ctx, svcs, ok := s.loadTemplateServices(w, r.WithContext(ctx), tpl, data, provider)
		if !ok {
			return
		}

		//find the matching service
		var matchedSvc *serviceUI
		if svcIDStr != "" {
			for _, svc := range svcs {
				if svc.ID.String() == svcIDStr {
					matchedSvc = svc
					break
				}
			}
		}

		//check the method
		if r.Method == http.MethodGet {
			//default the campaign
			if campaign != nil {
				data[TplParamAgeMin] = strconv.FormatInt(int64(campaign.AgeMin), 10)
				data[TplParamAgeMax] = strconv.FormatInt(int64(campaign.AgeMax), 10)
				data[TplParamBudget] = strconv.FormatFloat(float64(campaign.Budget), 'f', 2, 32)
				data[TplParamGender] = string(campaign.Gender)
				data[TplParamStart] = FormatDateLocal(campaign.Start, timeZone)
				data[TplParamEnd] = FormatDateLocal(campaign.End, timeZone)
			} else {
				now := GetTimeNow(timeZone).Add(CampaignStartPadding)
				data[TplParamAgeMin] = strconv.FormatInt(AgeMin, 10)
				data[TplParamAgeMax] = strconv.FormatInt(AgeMax, 10)
				data[TplParamBudget] = strconv.FormatFloat(CampaignBudgetDefault, 'f', 2, 32)
				data[TplParamGender] = string(GenderAll)
				data[TplParamStart] = FormatDateLocal(now, timeZone)
				data[TplParamEnd] = FormatDateLocal(now.Add(CampaignDurationDefault), timeZone)
			}
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//validate the data
		form := CampaignForm{
			AgeMin:    ageMinStr,
			AgeMax:    ageMaxStr,
			Budget:    budgetStr,
			Gender:    gender,
			Interests: interests,
			Locations: locations,
			ServiceID: svcIDStr,
			Start:     startStr,
			End:       endStr,
			Text:      text,
		}
		ok = s.validateForm(w, r.WithContext(ctx), tpl, data, errs, form, true)
		if !ok {
			return
		}

		//parse the data
		ageMin, _ := strconv.ParseInt(form.AgeMin, 10, 32)
		ageMax, _ := strconv.ParseInt(form.AgeMax, 10, 32)
		budget, _ := strconv.ParseFloat(form.Budget, 32)
		start := ParseDateLocal(startStr, timeZone)
		end := ParseDateLocal(endStr, timeZone)

		//handle the uploaded files
		ctx, uploadImg, err := s.processFileUpload(r.WithContext(ctx), URLParams.Img, provider.FormatUserID())
		if err != nil {
			logger.Errorw("upload file", "error", err)
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}
		if uploadImg == nil && (campaign == nil || campaign.Img == nil) {
			logger.Errorw("save campaign", "error", err)
			errs[string(FieldErrImg)] = GetFieldErrText(string(FieldErrImg))
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//populate from the form
		if campaign == nil {
			campaign = s.createCampaignUI(&Campaign{})
		}
		campaign.UserID = provider.User.ID
		campaign.ProviderID = provider.ID
		campaign.ProviderName = provider.Name
		campaign.AgeMin = int(ageMin)
		campaign.AgeMax = int(ageMax)
		campaign.Budget = float32(budget)
		campaign.Gender = Gender(form.Gender)
		campaign.Interests = form.Interests
		campaign.Locations = form.Locations
		campaign.Platform = CampaignPlatformFacebook
		campaign.Start = start
		campaign.End = end
		campaign.Text = form.Text
		campaign.Status = CampaignStatusSubmitted
		if matchedSvc != nil {
			campaign.SetService(matchedSvc.Service)
		}
		if uploadImg != nil {
			campaign.SetImg(uploadImg.GetFile())
		}

		//mark as deleted, which will be modified after step 2
		campaign.Deleted = true

		//save the campaign
		ctx, err = SaveCampaign(ctx, s.getDB(), campaign.Campaign)
		if err != nil {
			logger.Errorw("save campaign", "error", err, "campaign", campaign, "id", provider.ID)
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//go to step2
		url := provider.GetURLCampaignAddStep2(campaign.ID)
		http.Redirect(w, r.WithContext(ctx), url, http.StatusSeeOther)
	}
}

//handle the campaign add step 2 page
func (s *Server) handleDashboardCampaignAddStep2() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateDashboard(ctx, "campaign-add-step2.html")
		})
		ctx, provider, data, errs, ok := s.createTemplateDataDashboard(w, r.WithContext(ctx), tpl, true)
		if !ok {
			return
		}

		//setup the breadcrumbs
		breadcrumbs := []breadcrumb{
			{"Campaigns", provider.GetURLCampaigns()},
			{"Add Campaign", ""},
		}
		data[TplParamBreadcrumbs] = breadcrumbs
		data[TplParamActiveNav] = provider.GetURLCampaigns()
		data[TplParamFormAction] = provider.GetURLCampaignAddStep2(nil)

		//handle the input
		hasFacebookAdAccount := r.FormValue(URLParams.HasFacebookAdAccount) == "true"
		hasFacebookPage := r.FormValue(URLParams.HasFacebookPage) == "true"
		urlFacebook := r.FormValue(URLParams.URLFacebook)

		//prepare the data
		data[TplParamHasFacebookAdAccount] = hasFacebookAdAccount
		data[TplParamHasFacebookPage] = hasFacebookPage
		data[TplParamURLFacebookProvider] = urlFacebook

		//load the campaign if necessary
		ctx, campaign, ok := s.loadTemplateCampaign(w, r.WithContext(ctx), tpl, data, provider, true)
		if !ok {
			return
		}

		//check the method
		if r.Method == http.MethodGet {
			data[TplParamHasFacebookAdAccount] = true
			data[TplParamHasFacebookPage] = true
			data[TplParamURLFacebookProvider] = provider.URLFacebook
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//validate the data
		form := CampaignFacebookForm{
			HasFacebookAdAccount: hasFacebookAdAccount,
			HasFacebookPage:      hasFacebookPage,
			URLFacebook:          urlFacebook,
		}
		ok = s.validateForm(w, r.WithContext(ctx), tpl, data, errs, form, true)
		if !ok {
			return
		}

		//populate from the form
		campaign.HasFacebookAdAccount = hasFacebookAdAccount
		campaign.HasFacebookPage = hasFacebookPage
		campaign.URLFacebook = urlFacebook

		//campaign is no longer deleted
		campaign.Deleted = false

		//save the campaign
		ctx, err := SaveCampaign(ctx, s.getDB(), campaign.Campaign)
		if err != nil {
			logger.Errorw("save campaign", "error", err, "campaign", campaign, "id", provider.ID)
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//queue the emails
		ctx, err = s.queueEmailsCampaignAdd(ctx, provider, campaign)
		if err != nil {
			logger.Errorw("queue email campaign add", "error", err)
			s.SetCookieErr(w, ErrEmailVerifyToken)
			http.Redirect(w, r.WithContext(ctx), URIDefault, http.StatusSeeOther)
			return
		}

		//go to step3
		url := provider.GetURLCampaignAddStep3()
		http.Redirect(w, r.WithContext(ctx), url, http.StatusSeeOther)
	}
}

//handle the campaign add step 3 page
func (s *Server) handleDashboardCampaignAddStep3() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, _ := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateDashboard(ctx, "campaign-add-step3.html")
		})
		ctx, provider, data, _, ok := s.createTemplateDataDashboard(w, r.WithContext(ctx), tpl, true)
		if !ok {
			return
		}

		//setup the breadcrumbs
		breadcrumbs := []breadcrumb{
			{"Campaigns", provider.GetURLCampaigns()},
			{"Add Campaign", ""},
		}
		data[TplParamBreadcrumbs] = breadcrumbs
		data[TplParamActiveNav] = provider.GetURLCampaigns()
		s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
	}
}

//handle the campaign view page
func (s *Server) handleDashboardCampaignView() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateDashboard(ctx, "campaign-view.html")
		})
		ctx, provider, data, _, ok := s.createTemplateDataDashboard(w, r.WithContext(ctx), tpl, true)
		if !ok {
			return
		}

		//setup the breadcrumbs
		breadcrumbs := []breadcrumb{
			{"Campaigns", provider.GetURLCampaigns()},
			{"View Campaign", ""},
		}
		data[TplParamBreadcrumbs] = breadcrumbs
		data[TplParamActiveNav] = provider.GetURLCampaigns()
		data[TplParamFormAction] = provider.GetURLCampaignView(nil)

		//load the campaign
		ctx, campaign, ok := s.loadTemplateCampaign(w, r.WithContext(ctx), tpl, data, provider, false)
		if !ok {
			return
		}

		//load the payment
		ctx, payment, err := LoadPaymentByProviderIDAndSecondaryIDAndType(ctx, s.getDB(), campaign.ProviderID, campaign.ID, PaymentTypeCampaign)
		if err != nil {
			logger.Errorw("load payment", "error", err, "id", campaign.ID)
			s.SetCookieErr(w, Err)
			http.Redirect(w, r.WithContext(ctx), URIDefault, http.StatusSeeOther)
			return
		}
		if payment != nil {
			paymentUI := s.createPaymentUI(payment)
			data[TplParamPayment] = paymentUI
		}
		s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
	}
}

//handle the campaigns page
func (s *Server) handleDashboardCampaigns() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateDashboard(ctx, "campaigns.html")
		})
		ctx, provider, data, _, ok := s.createTemplateDataDashboard(w, r.WithContext(ctx), tpl, true)
		if !ok {
			return
		}

		//setup the breadcrumbs
		breadcrumbs := []breadcrumb{
			{"Campaigns", ""},
		}
		data[TplParamBreadcrumbs] = breadcrumbs
		data[TplParamActiveNav] = provider.GetURLCampaigns()

		//load the campaigns
		ctx, campaigns, err := ListCampaignsByProviderID(ctx, s.getDB(), provider.Provider)
		if err != nil {
			logger.Errorw("list campaigns", "error", err, "id", provider.ID)
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}
		data[TplParamCampaigns] = campaigns
		s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
	}
}

//handle the client add page
func (s *Server) handleDashboardClientAdd() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateDashboard(ctx, "client-add.html")
		})
		ctx, provider, data, errs, ok := s.createTemplateDataDashboard(w, r.WithContext(ctx), tpl, true)
		if !ok {
			return
		}

		//setup the breadcrumbs
		breadcrumbs := []breadcrumb{
			{"Clients", provider.GetURLClients()},
			{"Add Client", ""},
		}
		data[TplParamBreadcrumbs] = breadcrumbs
		data[TplParamActiveNav] = provider.GetURLClients()
		data[TplParamClientView] = r.FormValue(URLParams.Client)

		//check the method
		if r.Method == http.MethodGet {
			data[TplParamEmail] = ""
			data[TplParamLocation] = ""
			data[TplParamName] = ""
			data[TplParamPhone] = ""
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//handle the input
		email := r.FormValue(URLParams.Email)
		location := r.FormValue(URLParams.Location)
		name := r.FormValue(URLParams.Name)
		phone := r.FormValue(URLParams.Phone)

		//prepare the data
		data[TplParamEmail] = email
		data[TplParamLocation] = location
		data[TplParamName] = name
		data[TplParamPhone] = phone

		//validate the data
		form := ClientForm{
			ClientDataForm: ClientDataForm{
				EmailForm: EmailForm{
					Email: strings.TrimSpace(email),
				},
				NameForm: NameForm{
					Name: name,
				},
				Location: location,
				Phone:    FormatPhone(phone),
			},
			TimeZoneForm: TimeZoneForm{
				TimeZone: provider.User.TimeZone,
			},
		}
		ok = s.validateForm(w, r.WithContext(ctx), tpl, data, errs, form, true)
		if !ok {
			return
		}

		//check for an existing email
		ctx, existingClient, err := LoadClientByProviderIDAndEmail(ctx, s.getDB(), provider.ID, form.Email)
		if err != nil {
			logger.Errorw("existing client", "error", err, "email", form.Email)
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}
		if existingClient != nil {
			errs[string(FieldErrEmail)] = GetErrText(ErrClientEmailDup)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//populate from the form
		client := &Client{
			ProviderID: provider.ID,
			Email:      form.Email,
			Name:       form.Name,
			Location:   form.Location,
			Phone:      form.Phone,
			TimeZone:   form.TimeZone,
		}

		//save the client
		ctx, err = SaveClient(ctx, s.getDB(), client)
		if err != nil {
			logger.Errorw("save client", "error", err, "client", client, "id", provider.ID)
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//success
		s.SetCookieMsg(w, MsgClientAdd, client.Name)
		url := s.checkClientView(data, provider, provider.GetURLClients())
		http.Redirect(w, r.WithContext(ctx), url, http.StatusSeeOther)
	}
}

//handle the client edit page
func (s *Server) handleDashboardClientEdit() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template

	//steps on the page
	steps := struct {
		StepDel string
		StepUpd string
	}{
		StepDel: "stepDel",
		StepUpd: "stepUpd",
	}
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateDashboard(ctx, "client-edit.html")
		})
		ctx, provider, data, errs, ok := s.createTemplateDataDashboard(w, r.WithContext(ctx), tpl, true)
		if !ok {
			return
		}

		//setup the breadcrumbs
		breadcrumbs := []breadcrumb{
			{"Clients", provider.GetURLClients()},
			{"Edit Client", ""},
		}
		data[TplParamBreadcrumbs] = breadcrumbs
		data[TplParamActiveNav] = provider.GetURLClients()
		data[TplParamFormAction] = provider.GetURLClientEdit()
		data[TplParamSteps] = steps

		//handle the input
		email := r.FormValue(URLParams.Email)
		location := r.FormValue(URLParams.Location)
		name := r.FormValue(URLParams.Name)
		phone := r.FormValue(URLParams.Phone)
		step := r.FormValue(URLParams.Step)

		//prepare the data
		data[TplParamEmail] = email
		data[TplParamLocation] = location
		data[TplParamName] = name
		data[TplParamPhone] = phone

		//load the client
		ctx, client, ok := s.loadTemplateClient(w, r.WithContext(ctx), tpl, data, errs, provider)
		if !ok {
			return
		}

		//prepare the confirmation modal
		data[TplParamConfirmMsg] = GetMsgText(MsgClientDelConfirm)
		data[TplParamConfirmSubmitName] = URLParams.Step
		data[TplParamConfirmSubmitValue] = steps.StepDel

		//check the method
		if r.Method == http.MethodGet {
			data[TplParamEmail] = client.Email
			data[TplParamLocation] = client.Location
			data[TplParamName] = client.Name
			data[TplParamPhone] = client.Phone
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//validate the data
		form := ClientDataForm{
			EmailForm: EmailForm{
				Email: strings.TrimSpace(email),
			},
			NameForm: NameForm{
				Name: name,
			},
			Location: location,
			Phone:    FormatPhone(phone),
		}
		ok = s.validateForm(w, r.WithContext(ctx), tpl, data, errs, form, true)
		if !ok {
			return
		}

		//execute the correct operation
		var msgKey MsgKey
		switch step {
		case steps.StepDel:
			//delete the client
			ctx, count, err := DeleteClient(ctx, s.getDB(), provider.ID, client.ID)
			if err != nil {
				logger.Errorw("delete client", "error", err, "id", client.ID)
				data[TplParamErr] = GetErrText(Err)
				s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
				return
			}

			//check success
			if count > 0 {
				data[TplParamErr] = GetErrText(ErrBookingExist, count)
				s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
				return
			}
			msgKey = MsgClientDel
		case steps.StepUpd:
			//check for an existing email
			if client.Email != form.Email {
				ctx, existingClient, err := LoadClientByProviderIDAndEmail(ctx, s.getDB(), provider.ID, form.Email)
				if err != nil {
					logger.Errorw("existing client", "error", err, "email", form.Email)
					data[TplParamErr] = GetErrText(Err)
					s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
					return
				}
				if existingClient != nil {
					errs[string(FieldErrEmail)] = GetErrText(ErrClientEmailDup)
					s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
					return
				}
			}

			//populate from the form
			client.SetEmail(form.Email)
			client.Name = form.Name
			client.Location = form.Location
			client.Phone = form.Phone

			//update the client
			ctx, err := SaveClient(ctx, s.getDB(), client)
			if err != nil {
				logger.Errorw("save client", "error", err, "client", client)
				data[TplParamErr] = GetErrText(Err)
				s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
				return
			}
			msgKey = MsgClientEdit
		default:
			logger.Errorw("invalid step", "id", client.ID, "step", step)
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//success
		s.SetCookieMsg(w, msgKey, client.Name)
		http.Redirect(w, r.WithContext(ctx), provider.GetURLClients(), http.StatusSeeOther)
	}
}

//handle the clients page
func (s *Server) handleDashboardClients() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateDashboard(ctx, "clients.html")
		})
		ctx, provider, data, errs, ok := s.createTemplateDataDashboard(w, r.WithContext(ctx), tpl, true)
		if !ok {
			return
		}

		//setup the breadcrumbs
		breadcrumbs := []breadcrumb{
			{"Clients", ""},
		}
		data[TplParamBreadcrumbs] = breadcrumbs
		data[TplParamActiveNav] = provider.GetURLClients()
		data[TplParamFormAction] = provider.GetURLClientEdit()
		data[TplParamFormAction2] = provider.GetURLClients()

		//load the clients
		ctx, _, ok = s.loadTemplateClients(w, r.WithContext(ctx), tpl, data, provider.ID)
		if !ok {
			return
		}

		//check the method
		if r.Method == http.MethodGet {
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//load the client
		ctx, client, ok := s.loadTemplateClient(w, r.WithContext(ctx), tpl, data, errs, provider)
		if !ok {
			return
		}

		//update the client invited
		ctx, err := UpdateClientInvited(ctx, s.getDB(), client.ID)
		if err != nil {
			logger.Errorw("update client invited", "error", err, "id", client.ID)
			ctx = s.setCtxErr(ctx, ErrClientInvite)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//send the email
		ctx, err = s.queueEmailClientInvite(ctx, provider, client)
		if err != nil {
			logger.Errorw("queue email client invited", "error", err, "id", client.ID)
			ctx = s.setCtxErr(ctx, ErrClientInvite)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//re-load the clients
		ctx, _, ok = s.loadTemplateClients(w, r.WithContext(ctx), tpl, data, provider.ID)
		if !ok {
			return
		}

		//success
		ctx = s.setCtxMsg(ctx, MsgClientInviteSuccess)
		s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
	}
}

//handle the coupon add page
func (s *Server) handleDashboardCouponAdd() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateDashboard(ctx, "coupon-add.html")
		})
		ctx, provider, data, errs, ok := s.createTemplateDataDashboard(w, r.WithContext(ctx), tpl, true)
		if !ok {
			return
		}

		//setup the breadcrumbs
		breadcrumbs := []breadcrumb{
			{"Coupons", provider.GetURLCoupons()},
			{"Add Coupon", ""},
		}
		data[TplParamBreadcrumbs] = breadcrumbs
		data[TplParamActiveNav] = provider.GetURLCoupons()

		//handle the input
		couponTypeStr := r.FormValue(URLParams.Type)
		code := strings.ToUpper(r.FormValue(URLParams.Code))
		valStr := r.FormValue(URLParams.Value)
		startStr := r.FormValue(URLParams.Start)
		endStr := r.FormValue(URLParams.End)
		desc := r.FormValue(URLParams.Desc)
		svcIDStr := r.FormValue(URLParams.SvcID)
		newClients := r.FormValue(URLParams.Flag) == "on"
		timeZone := r.FormValue(URLParams.TimeZone)

		//prepare the data
		data[TplParamType] = couponTypeStr
		data[TplParamCode] = code
		data[TplParamValue] = valStr
		data[TplParamStart] = startStr
		data[TplParamEnd] = endStr
		data[TplParamDesc] = desc
		data[TplParamSvcID] = svcIDStr
		data[TplParamFlag] = newClients

		//load the services
		ctx, svcs, ok := s.loadTemplateServices(w, r.WithContext(ctx), tpl, data, provider)
		if !ok {
			return
		}

		//find the matching service
		var matchedSvc *serviceUI
		if svcIDStr != "" {
			for _, svc := range svcs {
				if svc.ID.String() == svcIDStr {
					matchedSvc = svc
					break
				}
			}
		}

		//check the method
		if r.Method == http.MethodGet {
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//validate the data
		form := CouponForm{
			Type:        couponTypeStr,
			Code:        code,
			Value:       valStr,
			Start:       startStr,
			End:         endStr,
			Description: desc,
			ServiceID:   svcIDStr,
			NewClients:  newClients,
		}
		ok = s.validateForm(w, r.WithContext(ctx), tpl, data, errs, form, true)
		if !ok {
			return
		}

		//check for an existing code
		ctx, id, err := LoadCouponByProviderIDAndCode(ctx, s.getDB(), provider.ID, form.Code, nil)
		if err != nil {
			logger.Errorw("existing coupon", "error", err, "email", form.Code)
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}
		if id != nil {
			errs[string(FieldErrCode)] = GetErrText(ErrCouponCodeDup)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//parse the data
		couponType := ParseCouponType(couponTypeStr)
		val, _ := strconv.ParseFloat(valStr, 32)
		start := ParseDateLocal(startStr, timeZone)
		end := ParseDateLocal(endStr, timeZone)

		//populate from the form
		coupon := &Coupon{
			ProviderID:  provider.ID,
			Type:        *couponType,
			Code:        strings.ToUpper(form.Code),
			Value:       float32(val),
			Start:       start,
			End:         end,
			Description: desc,
			NewClients:  form.NewClients,
		}
		if matchedSvc != nil {
			coupon.SetService(matchedSvc.Service)
		}

		//save the coupon
		ctx, err = SaveCoupon(ctx, s.getDB(), coupon)
		if err != nil {
			logger.Errorw("save coupon", "error", err, "coupon", coupon, "id", provider.ID)
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//success
		s.SetCookieMsg(w, MsgCouponAdd)
		http.Redirect(w, r.WithContext(ctx), provider.GetURLCoupons(), http.StatusSeeOther)
	}
}

//handle the coupon edit page
func (s *Server) handleDashboardCouponEdit() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template

	//steps on the page
	steps := struct {
		StepDel string
		StepUpd string
	}{
		StepDel: "stepDel",
		StepUpd: "stepUpd",
	}
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateDashboard(ctx, "coupon-edit.html")
		})
		ctx, provider, data, errs, ok := s.createTemplateDataDashboard(w, r.WithContext(ctx), tpl, true)
		if !ok {
			return
		}

		//setup the breadcrumbs
		breadcrumbs := []breadcrumb{
			{"Coupons", provider.GetURLCoupons()},
			{"Edit Coupon", ""},
		}
		data[TplParamBreadcrumbs] = breadcrumbs
		data[TplParamActiveNav] = provider.GetURLCoupons()
		data[TplParamFormAction] = provider.GetURLCouponEdit(nil)
		data[TplParamSteps] = steps

		//handle the input
		couponTypeStr := r.FormValue(URLParams.Type)
		code := strings.ToUpper(r.FormValue(URLParams.Code))
		valStr := r.FormValue(URLParams.Value)
		startStr := r.FormValue(URLParams.Start)
		endStr := r.FormValue(URLParams.End)
		desc := r.FormValue(URLParams.Desc)
		svcIDStr := r.FormValue(URLParams.SvcID)
		newClients := r.FormValue(URLParams.Flag) == "on"
		timeZone := r.FormValue(URLParams.TimeZone)
		step := r.FormValue(URLParams.Step)

		//prepare the data
		data[TplParamType] = couponTypeStr
		data[TplParamCode] = code
		data[TplParamValue] = valStr
		data[TplParamStart] = startStr
		data[TplParamEnd] = endStr
		data[TplParamDesc] = desc
		data[TplParamSvcID] = svcIDStr
		data[TplParamFlag] = newClients

		//validate the id
		idStr := r.FormValue(URLParams.ID)
		couponID := uuid.FromStringOrNil(idStr)
		if couponID == uuid.Nil {
			logger.Warnw("invalid uuid", "id", idStr)
			s.SetCookieErr(w, Err)
			http.Redirect(w, r.WithContext(ctx), provider.GetURLCoupons(), http.StatusSeeOther)
			return
		}

		//load the coupon
		ctx, coupon, err := LoadCouponByProviderIDAndID(ctx, s.getDB(), provider.Provider, &couponID)
		if err != nil {
			logger.Errorw("load coupon", "error", err, "id", couponID)
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}
		data[TplParamCoupon] = coupon

		//load the services
		ctx, svcs, ok := s.loadTemplateServices(w, r.WithContext(ctx), tpl, data, provider)
		if !ok {
			return
		}

		//find the matching service
		var matchedSvc *serviceUI
		if svcIDStr != "" {
			for _, svc := range svcs {
				if svc.ID.String() == svcIDStr {
					matchedSvc = svc
					break
				}
			}
		}

		//prepare the confirmation modal
		data[TplParamConfirmMsg] = GetMsgText(MsgCouponDelConfirm)
		data[TplParamConfirmSubmitName] = URLParams.Step
		data[TplParamConfirmSubmitValue] = steps.StepDel

		//check the method
		if r.Method == http.MethodGet {
			data[TplParamType] = coupon.Type
			data[TplParamCode] = coupon.Code
			data[TplParamValue] = coupon.Value
			data[TplParamStart] = coupon.FormatStart(timeZone)
			data[TplParamEnd] = coupon.FormatEnd(timeZone)
			data[TplParamDesc] = coupon.Description
			data[TplParamFlag] = coupon.NewClients
			if coupon.ServiceID != nil {
				data[TplParamSvcID] = coupon.ServiceID.String()
			} else {
				data[TplParamSvcID] = ""
			}
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//execute the correct operation
		var msgKey MsgKey
		switch step {
		case steps.StepDel:
			//delete the coupon
			ctx, err := DeleteCoupon(ctx, s.getDB(), provider.ID, coupon.ID)
			if err != nil {
				logger.Errorw("delete coupon", "error", err, "id", coupon.ID)
				data[TplParamErr] = GetErrText(Err)
				s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
				return
			}
			msgKey = MsgCouponDel
		case steps.StepUpd:
			//validate the data
			form := CouponForm{
				Type:        couponTypeStr,
				Code:        code,
				Value:       valStr,
				Start:       startStr,
				End:         endStr,
				Description: desc,
				ServiceID:   svcIDStr,
				NewClients:  newClients,
			}
			ok = s.validateForm(w, r.WithContext(ctx), tpl, data, errs, form, true)
			if !ok {
				return
			}

			//check for an existing code
			if coupon.Code != form.Code {
				ctx, id, err := LoadCouponByProviderIDAndCode(ctx, s.getDB(), provider.ID, form.Code, nil)
				if err != nil {
					logger.Errorw("existing coupon", "error", err, "email", form.Code)
					data[TplParamErr] = GetErrText(Err)
					s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
					return
				}
				if id != nil {
					errs[string(FieldErrCode)] = GetErrText(ErrCouponCodeDup)
					s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
					return
				}
			}

			//parse the data
			couponType := ParseCouponType(couponTypeStr)
			val, _ := strconv.ParseFloat(valStr, 32)
			start := ParseDateLocal(startStr, timeZone)
			end := ParseDateLocal(endStr, timeZone)

			//populate from the form
			coupon.Type = *couponType
			coupon.Code = form.Code
			coupon.Value = float32(val)
			coupon.Start = start
			coupon.End = end
			coupon.Description = desc
			coupon.NewClients = newClients
			if matchedSvc != nil {
				coupon.SetService(matchedSvc.Service)
			}

			//update the coupon
			ctx, err := SaveCoupon(ctx, s.getDB(), coupon)
			if err != nil {
				logger.Errorw("save coupon", "error", err, "coupon", coupon)
				data[TplParamErr] = GetErrText(Err)
				s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
				return
			}
			msgKey = MsgCouponEdit
		default:
			logger.Errorw("invalid step", "id", coupon.ID, "step", step)
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//success
		s.SetCookieMsg(w, msgKey)
		http.Redirect(w, r.WithContext(ctx), provider.GetURLCoupons(), http.StatusSeeOther)
	}
}

//handle the coupons page
func (s *Server) handleDashboardCoupons() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateDashboard(ctx, "coupons.html")
		})
		ctx, provider, data, _, ok := s.createTemplateDataDashboard(w, r.WithContext(ctx), tpl, true)
		if !ok {
			return
		}

		//setup the breadcrumbs
		breadcrumbs := []breadcrumb{
			{"Coupons", ""},
		}
		data[TplParamBreadcrumbs] = breadcrumbs
		data[TplParamActiveNav] = provider.GetURLCoupons()

		//load the coupons
		ctx, coupons, err := ListCouponsByProviderID(ctx, s.getDB(), provider.Provider)
		if err != nil {
			logger.Errorw("load coupons", "error", err)
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}
		data[TplParamCoupons] = coupons
		s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
	}
}

//handle the faq add page
func (s *Server) handleDashboardFaqAdd() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateDashboard(ctx, "faq-add.html")
		})
		ctx, provider, data, errs, ok := s.createTemplateDataDashboard(w, r.WithContext(ctx), tpl, true)
		if !ok {
			return
		}

		//setup the breadcrumbs
		breadcrumbs := []breadcrumb{
			{"FAQs", provider.GetURLFaqs()},
			{"Add FAQ", ""},
		}
		data[TplParamBreadcrumbs] = breadcrumbs
		data[TplParamActiveNav] = provider.GetURLFaqs()
		data[TplParamClientView] = r.FormValue(URLParams.Client)

		//check the method
		if r.Method == http.MethodGet {
			data[TplParamName] = ""
			data[TplParamText] = ""
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//handle the input
		name := r.FormValue(URLParams.Name)
		text := r.FormValue(URLParams.Text)

		//prepare the data
		data[TplParamName] = name
		data[TplParamText] = text

		//validate the data
		form := FaqForm{
			Question: name,
			Answer:   text,
		}
		ok = s.validateForm(w, r.WithContext(ctx), tpl, data, errs, form, true)
		if !ok {
			return
		}

		//populate from the form
		faq := &Faq{
			ProviderID: provider.ID,
			Question:   form.Question,
			Answer:     form.Answer,
		}

		//save the faq
		ctx, err := SaveFaq(ctx, s.getDB(), faq)
		if err != nil {
			logger.Errorw("save faq", "error", err, "faq", faq, "id", provider.ID)
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//success
		s.SetCookieMsg(w, MsgFaqAdd)
		url := s.checkClientView(data, provider, provider.GetURLFaqs())
		http.Redirect(w, r.WithContext(ctx), url, http.StatusSeeOther)
	}
}

//handle the faq edit page
func (s *Server) handleDashboardFaqEdit() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template

	//steps on the page
	steps := struct {
		StepDel string
		StepUpd string
	}{
		StepDel: "stepDel",
		StepUpd: "stepUpd",
	}
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateDashboard(ctx, "faq-edit.html")
		})
		ctx, provider, data, errs, ok := s.createTemplateDataDashboard(w, r.WithContext(ctx), tpl, true)
		if !ok {
			return
		}

		//setup the breadcrumbs
		breadcrumbs := []breadcrumb{
			{"FAQs", provider.GetURLFaqs()},
			{"Edit FAQ", ""},
		}
		data[TplParamBreadcrumbs] = breadcrumbs
		data[TplParamActiveNav] = provider.GetURLFaqs()
		data[TplParamFormAction] = provider.GetURLFaqEdit(nil)
		data[TplParamSteps] = steps
		data[TplParamClientView] = r.FormValue(URLParams.Client)

		//validate the id
		idStr := r.FormValue(URLParams.ID)
		faqID := uuid.FromStringOrNil(idStr)
		if faqID == uuid.Nil {
			logger.Warnw("invalid uuid", "id", idStr)
			s.SetCookieErr(w, Err)
			http.Redirect(w, r.WithContext(ctx), provider.GetURLFaqs(), http.StatusSeeOther)
			return
		}

		//load the faq
		ctx, faq, err := LoadFaqByProviderIDAndID(ctx, s.getDB(), provider.Provider, &faqID)
		if err != nil {
			logger.Errorw("load faq", "error", err, "id", faqID)
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}
		data[TplParamFaq] = s.createFaqUI(faq)

		//prepare the confirmation modal
		data[TplParamConfirmMsg] = GetMsgText(MsgFaqDelConfirm)
		data[TplParamConfirmSubmitName] = URLParams.Step
		data[TplParamConfirmSubmitValue] = steps.StepDel

		//check the method
		if r.Method == http.MethodGet {
			data[TplParamName] = faq.Question
			data[TplParamText] = faq.Answer
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//handle the input
		name := r.FormValue(URLParams.Name)
		step := r.FormValue(URLParams.Step)
		text := r.FormValue(URLParams.Text)

		//prepare the data
		data[TplParamName] = name
		data[TplParamText] = text

		//validate the data
		form := FaqForm{
			Question: name,
			Answer:   text,
		}
		ok = s.validateForm(w, r.WithContext(ctx), tpl, data, errs, form, true)
		if !ok {
			return
		}

		//execute the correct operation
		var msgKey MsgKey
		switch step {
		case steps.StepDel:
			//delete the faq
			ctx, err := DeleteFaq(ctx, s.getDB(), provider.ID, faq.ID)
			if err != nil {
				logger.Errorw("delete faq", "error", err, "id", faq.ID)
				data[TplParamErr] = GetErrText(Err)
				s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
				return
			}
			msgKey = MsgFaqDel
		case steps.StepUpd:
			//populate from the form
			faq.Question = form.Question
			faq.Answer = form.Answer

			//update the faq
			ctx, err = SaveFaq(ctx, s.getDB(), faq)
			if err != nil {
				logger.Errorw("save testimonial", "error", err, "faq", faq)
				data[TplParamErr] = GetErrText(Err)
				s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
				return
			}
			msgKey = MsgFaqEdit
		default:
			logger.Errorw("invalid step", "id", faq.ID, "step", step)
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//success
		s.SetCookieMsg(w, msgKey)
		url := s.checkClientView(data, provider, provider.GetURLFaqs())
		http.Redirect(w, r.WithContext(ctx), url, http.StatusSeeOther)
	}
}

//handle the faqs page
func (s *Server) handleDashboardFaqs() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template

	//steps on the page
	steps := struct {
		StepUp   string
		StepDown string
	}{
		StepUp:   "stepUp",
		StepDown: "stepDown",
	}
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateDashboard(ctx, "faqs.html")
		})
		ctx, provider, data, _, ok := s.createTemplateDataDashboard(w, r.WithContext(ctx), tpl, true)
		if !ok {
			return
		}

		//setup the breadcrumbs
		breadcrumbs := []breadcrumb{
			{"FAQs", ""},
		}
		data[TplParamBreadcrumbs] = breadcrumbs
		data[TplParamActiveNav] = provider.GetURLFaqs()
		data[TplParamFormAction] = provider.GetURLFaqs()
		data[TplParamSteps] = steps
		data[TplParamClientView] = r.FormValue(URLParams.Client)

		//load the faqs
		ctx, faqs, ok := s.loadTemplateFaqs(w, r.WithContext(ctx), tpl, data, provider)
		if !ok {
			return
		}
		count := len(faqs) - 1
		data[TplParamCount] = count

		//check the method
		if r.Method == http.MethodGet {
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//handle the input
		step := r.FormValue(URLParams.Step)

		//validate the id
		idStr := r.FormValue(URLParams.ID)
		faqID := uuid.FromStringOrNil(idStr)
		if faqID == uuid.Nil {
			logger.Warnw("invalid uuid", "id", idStr)
			s.SetCookieErr(w, Err)
			http.Redirect(w, r.WithContext(ctx), provider.GetURLFaqs(), http.StatusSeeOther)
			return
		}

		//find the index of the faq
		faqIDs := make([]*uuid.UUID, len(faqs))
		faqIdx := -1
		for idx, faq := range faqs {
			if faq.ID.String() == idStr {
				faqIdx = idx
			}
			faqIDs[idx] = faq.ID
		}
		if faqIdx == -1 {
			logger.Errorw("invalid faq id", "id", faqID)
			s.SetCookieErr(w, Err)
			http.Redirect(w, r.WithContext(ctx), provider.GetURLFaqs(), http.StatusSeeOther)
			return
		}

		//check the step
		switch step {
		case steps.StepUp:
			if faqIdx == 0 {
				logger.Errorw("invalid step up", "id", faqID)
				s.SetCookieErr(w, Err)
				http.Redirect(w, r.WithContext(ctx), provider.GetURLFaqs(), http.StatusSeeOther)
				return
			}

			//swap faqs
			faqID := faqIDs[faqIdx-1]
			faqIDs[faqIdx-1] = faqIDs[faqIdx]
			faqIDs[faqIdx] = faqID
		case steps.StepDown:
			if faqIdx == count {
				logger.Errorw("invalid step down", "id", faqID)
				s.SetCookieErr(w, Err)
				http.Redirect(w, r.WithContext(ctx), provider.GetURLFaqs(), http.StatusSeeOther)
				return
			}

			//swap faqs
			faqID := faqIDs[faqIdx+1]
			faqIDs[faqIdx+1] = faqIDs[faqIdx]
			faqIDs[faqIdx] = faqID
		default:
			logger.Errorw("invalid step", "id", faqID, "step", step)
			s.SetCookieErr(w, Err)
			http.Redirect(w, r.WithContext(ctx), provider.GetURLFaqs(), http.StatusSeeOther)
			return
		}

		//update the faq indices
		_, err := UpdateFaqIndices(ctx, s.getDB(), faqIDs)
		if err != nil {
			logger.Errorw("update faq indices", "error", err)
			s.SetCookieErr(w, Err)
			http.Redirect(w, r.WithContext(ctx), provider.GetURLFaqs(), http.StatusSeeOther)
			return
		}
		http.Redirect(w, r.WithContext(ctx), provider.GetURLFaqs(), http.StatusSeeOther)
	}
}

//handle the hours page
func (s *Server) handleDashboardHours() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateDashboard(ctx, "service-hours.html")
		})
		ctx, provider, data, _, ok := s.createTemplateDataDashboard(w, r.WithContext(ctx), tpl, false)
		if !ok {
			return
		}

		//setup the breadcrumbs
		breadcrumbs := []breadcrumb{
			{"Schedule", ""},
		}
		data[TplParamBreadcrumbs] = breadcrumbs
		data[TplParamActiveNav] = provider.GetURLHours()
		data[TplParamFormAction] = provider.GetURLHours()
		data[TplParamClientView] = r.FormValue(URLParams.Client)

		//check the method
		if r.Method == http.MethodGet {
			ok = s.loadTemplateProviderSchedule(w, r.WithContext(ctx), tpl, data, provider)
			if !ok {
				return
			}
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//handle the input
		schedule := r.FormValue(URLParams.Schedule)

		//prepare the data
		data[TplParamSchedule] = schedule

		//validate the data
		now := data[TplParamCurrentTime].(time.Time)
		errDays, err := s.setSchedule(ctx, provider, schedule, now, provider.User.TimeZone)
		if err != nil {
			logger.Warnw("set schedule", "error", errDays, "schedule", schedule)
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}
		if len(errDays) > 0 {
			logger.Warnw("invalid provider schedule", "error", errDays, "schedule", schedule)
			errDaysJSON, err := json.Marshal(errDays)
			if err != nil {
				logger.Warnw("error json", "error", errDays)
				data[TplParamErr] = GetErrText(Err)
				s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
				return
			}
			data[TplParamDaysOfWeek] = string(errDaysJSON)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//save the provider
		ctx, err = SaveProvider(ctx, s.getDB(), provider.Provider)
		if err != nil {
			logger.Errorw("save provider", "error", err, "provider", provider)
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//success
		s.SetCookieMsg(w, MsgUpdateSuccess)
		url := s.checkClientView(data, provider, provider.GetURLHours())
		http.Redirect(w, r.WithContext(ctx), url, http.StatusSeeOther)
	}
}

//handle the dashboard page
func (s *Server) handleDashboardIndex() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, _ := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateDashboard(ctx, "index.html")
		})
		ctx, _, data, _, ok := s.createTemplateDataDashboard(w, r.WithContext(ctx), tpl, false)
		if !ok {
			return
		}
		s.SetCookieSignUp(w)
		s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
	}
}

//handle the links page
func (s *Server) handleDashboardLinks() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateDashboard(ctx, "links.html")
		})
		ctx, provider, data, errs, ok := s.createTemplateDataDashboard(w, r.WithContext(ctx), tpl, true)
		if !ok {
			return
		}

		//setup the breadcrumbs
		breadcrumbs := []breadcrumb{
			{"Links", ""},
		}
		data[TplParamBreadcrumbs] = breadcrumbs
		data[TplParamActiveNav] = provider.GetURLLinks()
		data[TplParamFormAction] = provider.GetURLLinks()
		data[TplParamClientView] = r.FormValue(URLParams.Client)

		//check the method
		if r.Method == http.MethodGet {
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//handle the input
		urlFacebook := r.FormValue(URLParams.URLFacebook)
		urlInstagram := r.FormValue(URLParams.URLInstagram)
		urlLinkedIn := r.FormValue(URLParams.URLLinkedIn)
		urlTwitter := r.FormValue(URLParams.URLTwitter)
		urlWeb := r.FormValue(URLParams.URLWeb)

		//prepare the data
		data[TplParamURLFacebookProvider] = urlFacebook
		data[TplParamURLInstagramProvider] = urlInstagram
		data[TplParamURLLinkedInProvider] = urlLinkedIn
		data[TplParamURLTwitterProvider] = urlTwitter
		data[TplParamURLWebProvider] = urlWeb

		//validate the data
		form := ProviderLinksForm{
			URLFacebook:  urlFacebook,
			URLInstagram: urlInstagram,
			URLLinkedIn:  urlLinkedIn,
			URLTwitter:   urlTwitter,
			URLWeb:       urlWeb,
		}
		ok = s.validateForm(w, r.WithContext(ctx), tpl, data, errs, form, true)
		if !ok {
			return
		}

		//populate from the form
		provider.URLFacebook = form.URLFacebook
		provider.URLInstagram = form.URLInstagram
		provider.URLLinkedIn = form.URLLinkedIn
		provider.URLTwitter = form.URLTwitter
		provider.URLWeb = form.URLWeb

		//save the provider
		ctx, err := SaveProvider(ctx, s.getDB(), provider.Provider)
		if err != nil {
			logger.Errorw("save provider", "error", err, "provider", provider)
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//success
		s.SetCookieMsg(w, MsgUpdateSuccess)
		url := s.checkClientView(data, provider, provider.GetURLLinks())
		http.Redirect(w, r.WithContext(ctx), url, http.StatusSeeOther)
	}
}

//handle the payment page
func (s *Server) handleDashboardPayment() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateDashboard(ctx, "payment.html")
		})
		ctx, provider, data, errs, ok := s.createTemplateDataDashboard(w, r.WithContext(ctx), tpl, true)
		if !ok {
			return
		}
		data[TplParamActiveNav] = provider.GetURLBookings()

		//load the booking
		idStr := r.FormValue(URLParams.BookID)
		ctx, book, ok := s.loadTemplateBook(w, r.WithContext(ctx), tpl, data, errs, idStr, true, false)
		if !ok {
			s.SetCookieErr(w, Err)
			http.Redirect(w, r.WithContext(ctx), provider.GetURLBookings(), http.StatusSeeOther)
			return
		}
		data[TplParamFormAction] = book.GetURLPayment()

		//check if a payment is supported, otherwise view the order
		if !book.SupportsPayment() {
			http.Redirect(w, r.WithContext(ctx), book.GetURLView(), http.StatusSeeOther)
			return
		}

		//check if already paid, in which case just view the payment
		if book.IsPaid() {
			http.Redirect(w, r.WithContext(ctx), book.GetURLPaymentView(), http.StatusSeeOther)
			return
		}

		//load the service
		now := data[TplParamCurrentTime].(time.Time)
		ctx, _, ok = s.loadTemplateService(w, r.WithContext(ctx), tpl, data, provider, book.Service.ID, now)
		if !ok {
			return
		}

		//check the method
		if r.Method == http.MethodGet {
			data[TplParamDesc] = ""
			data[TplParamEmail] = book.Client.Email
			data[TplParamName] = book.Client.Name
			data[TplParamPhone] = book.Client.Phone
			data[TplParamPrice] = book.ComputeServicePrice()
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//read the form
		desc := r.FormValue(URLParams.Desc)
		email := r.FormValue(URLParams.Email)
		name := r.FormValue(URLParams.Name)
		phone := r.FormValue(URLParams.Phone)
		priceStr := r.FormValue(URLParams.Price)

		//prepare the data
		data[TplParamDesc] = desc
		data[TplParamEmail] = email
		data[TplParamName] = name
		data[TplParamPhone] = phone
		data[TplParamPrice] = priceStr

		//validate the form
		form := &PaymentForm{
			EmailForm: EmailForm{
				Email: strings.TrimSpace(email),
			},
			NameForm: NameForm{
				Name: name,
			},
			Phone:           FormatPhone(phone),
			Price:           priceStr,
			Description:     desc,
			ClientInitiated: false,
			DirectCapture:   false,
		}
		ok = s.validateForm(w, r.WithContext(ctx), tpl, data, errs, form, true)
		if !ok {
			return
		}

		//save the payment
		ctx, payment, err := s.savePaymentBooking(ctx, provider, book, form, now)
		if err != nil {
			logger.Errorw("save payment", "error", err)
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//queue the email
		paymentUI := s.createPaymentUI(payment)
		ctx, err = s.queueEmailInvoice(ctx, provider.Name, paymentUI)
		if err != nil {
			logger.Errorw("queue email invoice", "error", err)
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//success
		s.SetCookieMsg(w, MsgPaymentSuccess)
		http.Redirect(w, r.WithContext(ctx), book.GetURLView(), http.StatusSeeOther)
	}
}

//handle the payment view page
func (s *Server) handleDashboardPaymentView() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template

	//steps on the page
	steps := struct {
		StepDel      string
		StepMarkPaid string
	}{
		StepDel:      "stepDel",
		StepMarkPaid: "stepMarkPaid",
	}
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateDashboard(ctx, "payment-view.html")
		})
		ctx, provider, data, errs, ok := s.createTemplateDataDashboard(w, r.WithContext(ctx), tpl, true)
		if !ok {
			return
		}
		data[TplParamActiveNav] = provider.GetURLPayments()
		data[TplParamSteps] = steps

		//load the booking
		now := data[TplParamCurrentTime].(time.Time)
		var paymentUI *paymentUI
		bookIDStr := r.FormValue(URLParams.BookID)
		if bookIDStr != "" {
			ctx, book, ok := s.loadTemplateBook(w, r.WithContext(ctx), tpl, data, errs, bookIDStr, false, false)
			if !ok {
				s.SetCookieErr(w, Err)
				http.Redirect(w, r.WithContext(ctx), provider.GetURLBookings(), http.StatusSeeOther)
				return
			}
			data[TplParamFormAction] = book.GetURLPaymentView()

			//load the service
			ctx, _, ok = s.loadTemplateService(w, r.WithContext(ctx), tpl, data, provider, book.Service.ID, now)
			if !ok {
				return
			}

			//probe for a payment
			ctx, payment, err := LoadPaymentByProviderIDAndSecondaryIDAndType(ctx, s.getDB(), provider.ID, book.ID, PaymentTypeBooking)
			if err != nil {
				logger.Errorw("load payment", "error", err, "id", book.ID)
				s.SetCookieErr(w, Err)
				http.Redirect(w, r.WithContext(ctx), provider.GetURLBookings(), http.StatusSeeOther)
				return
			}
			if payment == nil {
				s.SetCookieErr(w, Err)
				http.Redirect(w, r.WithContext(ctx), provider.GetURLBookings(), http.StatusSeeOther)
				return
			}
			paymentUI = s.createPaymentUI(payment)
		} else {
			//load the payment directly
			idStr := r.FormValue(URLParams.PaymentID)
			if idStr == "" {
				s.SetCookieErr(w, Err)
				http.Redirect(w, r.WithContext(ctx), provider.GetURLPayments(), http.StatusSeeOther)
				return
			}
			id := uuid.FromStringOrNil(idStr)
			if id == uuid.Nil {
				logger.Errorw("invalid uuid", "id", idStr)
				data[TplParamErr] = GetErrText(Err)
				s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
				return
			}
			ctx, payment, err := LoadPaymentByID(ctx, s.getDB(), &id)
			if err != nil {
				logger.Errorw("load payment", "error", err, "id", id)
				s.SetCookieErr(w, Err)
				http.Redirect(w, r.WithContext(ctx), provider.GetURLPayments(), http.StatusSeeOther)
				return
			}
			paymentUI = s.createPaymentUI(payment)
			data[TplParamFormAction] = paymentUI.GetURLView()

			//probe for a booking
			ctx, book, ok := s.loadTemplateBook(w, r.WithContext(ctx), tpl, data, errs, payment.SecondaryID.String(), false, false)
			if ok {
				ctx, _, _ = s.loadTemplateService(w, r.WithContext(ctx), tpl, data, provider, book.Service.ID, now)
			} else if paymentUI.ServiceID != "" {
				svcID := uuid.FromStringOrNil(paymentUI.ServiceID)
				if svcID == uuid.Nil {
					logger.Errorw("invalid uuid", "id", paymentUI.ServiceID)
					data[TplParamErr] = GetErrText(Err)
					s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
					return
				}
				ctx, _, _ = s.loadTemplateService(w, r.WithContext(ctx), tpl, data, provider, &svcID, now)
			}
		}
		data[TplParamPayment] = paymentUI

		//set-up the confirmation
		data[TplParamConfirmMsg] = GetMsgText(MsgPaymentMarkPaid)
		data[TplParamConfirmSubmitName] = URLParams.Step
		data[TplParamConfirmSubmitValue] = steps.StepMarkPaid

		//check the method
		if r.Method == http.MethodGet {
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//process the step
		step := r.FormValue(URLParams.Step)
		switch step {
		case steps.StepDel:
			ctx, err := DeletePayment(ctx, s.getDB(), paymentUI.ID)
			if err != nil {
				logger.Errorw("delete payment", "error", err, "id", paymentUI.ID)
				s.SetCookieErr(w, Err)
				http.Redirect(w, r.WithContext(ctx), provider.GetURLPayments(), http.StatusSeeOther)
				return
			}
		case steps.StepMarkPaid:
			ctx, err := UpdatePaymentDirectCapture(ctx, s.getDB(), paymentUI.ID, &now)
			if err != nil {
				logger.Errorw("update payment captured", "error", err, "id", paymentUI.ID)
				s.SetCookieErr(w, Err)
				http.Redirect(w, r.WithContext(ctx), provider.GetURLPayments(), http.StatusSeeOther)
				return
			}
		default:
			logger.Errorw("invalid step", "id", paymentUI.ID, "step", step)
			s.SetCookieErr(w, Err)
			http.Redirect(w, r.WithContext(ctx), provider.GetURLPayments(), http.StatusSeeOther)
			return
		}
		s.SetCookieMsg(w, MsgUpdateSuccess)
		http.Redirect(w, r.WithContext(ctx), provider.GetURLPayments(), http.StatusSeeOther)
	}
}

//handle the payment settings page
func (s *Server) handleDashboardPaymentSettings() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template

	//steps on the page
	steps := struct {
		StepDel string
		StepUpd string
	}{
		StepDel: "stepDel",
		StepUpd: "stepUpd",
	}
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateDashboard(ctx, "payment-settings.html")
		})
		ctx, provider, data, errs, ok := s.createTemplateDataDashboard(w, r.WithContext(ctx), tpl, true)
		if !ok {
			return
		}

		//setup the breadcrumbs
		breadcrumbs := []breadcrumb{
			{"Payment Settings", ""},
		}
		data[TplParamBreadcrumbs] = breadcrumbs
		data[TplParamActiveNav] = provider.GetURLPaymentSettings()
		data[TplParamFormAction] = provider.GetURLPaymentSettings()
		data[TplParamSteps] = steps
		data[TplParamTypes] = PaymentTypes

		//handle the input
		email := r.FormValue(URLParams.Email)
		id := r.FormValue(URLParams.ID)
		step := r.FormValue(URLParams.Step)
		paymentType := r.FormValue(URLParams.Type)

		//prepare the data
		data[TplParamEmail] = email
		data[TplParamID] = id
		data[TplParamType] = paymentType

		//prepare the confirmation modal
		switch paymentType {
		case PaymentTypes.TypePayPal:
			if provider.PayPalEmail != nil {
				data[TplParamConfirmMsg] = GetMsgText(MsgPayPalRemove)
				data[TplParamConfirmSubmitValue] = steps.StepDel
			} else {
				data[TplParamConfirmMsg] = GetMsgText(MsgPayPalActivate)
				data[TplParamConfirmSubmitValue] = steps.StepUpd
			}
			data[TplParamConfirmSubmitName] = URLParams.Step
		case PaymentTypes.TypeStripe:
			if provider.StripeToken != nil {
				data[TplParamConfirmMsg] = GetMsgText(MsgStripeRemove)
				data[TplParamConfirmSubmitValue] = steps.StepDel
			} else {
				data[TplParamConfirmMsg] = GetMsgText(MsgStripeActivate)
				data[TplParamConfirmSubmitValue] = steps.StepUpd
			}
			data[TplParamConfirmSubmitName] = URLParams.Step
		case PaymentTypes.TypeZelle:
			if provider.ZelleID != nil {
				data[TplParamConfirmMsg] = GetMsgText(MsgZelleRemove)
				data[TplParamConfirmSubmitValue] = steps.StepDel
			} else {
				data[TplParamConfirmMsg] = GetMsgText(MsgZelleActivate)
				data[TplParamConfirmSubmitValue] = steps.StepUpd
			}
			data[TplParamConfirmSubmitName] = URLParams.Step
		}

		//check the method
		if r.Method == http.MethodGet {
			//default the data
			if provider.PayPalEmail != nil {
				data[TplParamEmail] = *provider.PayPalEmail
			}
			if provider.ZelleID != nil {
				data[TplParamID] = *provider.ZelleID
			}
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//execute the correct operation
		switch step {
		case steps.StepDel:
			switch paymentType {
			case PaymentTypes.TypePayPal:
				provider.PayPalEmail = nil
			case PaymentTypes.TypeStripe:
				//revoke access
				err := RevokeOAuthTokenStripe(ctx, provider.StripeToken)
				if err != nil {
					logger.Errorw("revoke stripe", "error", err)
					data[TplParamErr] = GetErrText(Err)
					s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
					return
				}
				provider.StripeToken = nil
			case PaymentTypes.TypeZelle:
				provider.ZelleID = nil
			}

			//save the provider
			ctx, err := SaveProvider(ctx, s.getDB(), provider.Provider)
			if err != nil {
				logger.Errorw("save provider", "error", err, "provider", provider)
				data[TplParamErr] = GetErrText(Err)
				s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
				return
			}
		case steps.StepUpd:
			switch paymentType {
			case PaymentTypes.TypePayPal:
				//validate the data
				form := EmailForm{
					Email: strings.TrimSpace(email),
				}
				ok = s.validateForm(w, r.WithContext(ctx), tpl, data, errs, form, true)
				if !ok {
					return
				}

				//populate from the form
				provider.PayPalEmail = &form.Email

				//save the provider
				ctx, err := SaveProvider(ctx, s.getDB(), provider.Provider)
				if err != nil {
					logger.Errorw("save provider", "error", err, "provider", provider)
					data[TplParamErr] = GetErrText(Err)
					s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
					return
				}
			case PaymentTypes.TypeStripe:
				s.invokeHdlrGet(s.handleStripeLogin(), w, r.WithContext(ctx))
				return
			case PaymentTypes.TypeZelle:
				//validate the data
				form := ZelleIDForm{
					ZelleID: id,
				}
				ok = s.validateForm(w, r.WithContext(ctx), tpl, data, errs, form, true)
				if !ok {
					return
				}

				//populate from the form
				provider.ZelleID = &form.ZelleID

				//save the provider
				ctx, err := SaveProvider(ctx, s.getDB(), provider.Provider)
				if err != nil {
					logger.Errorw("save provider", "error", err, "provider", provider)
					data[TplParamErr] = GetErrText(Err)
					s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
					return
				}
			}
		default:
			logger.Errorw("invalid step", "step", step)
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//success
		s.SetCookieMsg(w, MsgUpdateSuccess)
		http.Redirect(w, r.WithContext(ctx), provider.GetURLPaymentSettings(), http.StatusSeeOther)
	}
}

//handle the payments page
func (s *Server) handleDashboardPayments() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateDashboard(ctx, "payments.html")
		})
		ctx, provider, data, _, ok := s.createTemplateDataDashboard(w, r.WithContext(ctx), tpl, true)
		if !ok {
			return
		}

		//setup the breadcrumbs
		breadcrumbs := []breadcrumb{
			{"Invoices", ""},
		}
		data[TplParamBreadcrumbs] = breadcrumbs
		data[TplParamActiveNav] = provider.GetURLPayments()
		data[TplParamFormAction] = provider.GetURLPayments()

		//read the form
		filterStr := r.FormValue(URLParams.Filter)

		//prepare the data
		data[TplParamFilter] = filterStr

		//validate the filter
		var err error
		filter := PaymentFilterAll
		if filterStr != "" {
			filter, err = ParsePaymentFilter(filterStr)
			if err != nil {
				logger.Errorw("parse filter", "error", err, "filter", filterStr)
				data[TplParamErr] = GetErrText(Err)
				s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			}
		}

		//load the payments
		ctx, payments, err := ListPaymentsByProviderIDAndFilter(ctx, s.getDB(), provider.ID, filter)
		if err != nil {
			logger.Errorw("load payments", "error", err, "id", provider.ID)
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}
		data[TplParamPayments] = s.createPaymentUIs(payments)

		//load the count
		ctx, countUnPaid, err := CountPaymentsByProviderIDAndFilter(ctx, s.getDB(), provider.ID, PaymentFilterUnPaid)
		if err != nil {
			logger.Errorw("count payments unpaid", "error", err, "id", provider.ID)
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}
		data[TplParamCountUnPaid] = countUnPaid
		s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
	}
}

//handle the profile page
func (s *Server) handleDashboardProfile() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateDashboard(ctx, "profile.html")
		})
		ctx, provider, data, errs, ok := s.createTemplateDataDashboard(w, r.WithContext(ctx), tpl, true)
		data[TplParamSvcAreaStrs] = ListServiceAreaStrs()
		if !ok {
			return
		}

		//setup the breadcrumbs
		breadcrumbs := []breadcrumb{
			{"Profile", ""},
		}
		data[TplParamBreadcrumbs] = breadcrumbs
		data[TplParamActiveNav] = provider.GetURLProfile()
		data[TplParamFormAction] = provider.GetURLProfile()
		data[TplParamSteps] = stepsProfileDomain
		data[TplParamTypes] = ImgViewTypes
		data[TplParamClientView] = r.FormValue(URLParams.Client)

		//check the method
		if r.Method == http.MethodGet {
			data[TplParamAbout] = provider.About
			data[TplParamDesc] = provider.Description
			data[TplParamDisablePhone] = provider.User.DisablePhone
			data[TplParamEducation] = provider.Education
			data[TplParamExperience] = provider.Experience
			data[TplParamLocation] = provider.Location
			data[TplParamName] = provider.Name
			data[TplParamSvcArea] = provider.ServiceArea
			data[TplParamType] = r.FormValue(URLParams.Type)
			data[TplParamURLName] = provider.URLNameFriendly
			if provider.Domain != nil {
				data[TplParamDomain] = &provider.Domain
			} else {
				data[TplParamDomain] = ""
			}
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//handle the input
		desc := r.FormValue(URLParams.Desc)
		education := r.FormValue(URLParams.Education)
		experience := r.FormValue(URLParams.Experience)
		imgDelBanner := r.FormValue(URLParams.ImgDelBanner) == "true"
		imgDelLogo := r.FormValue(URLParams.ImgDelLogo) == "true"
		location := r.FormValue(URLParams.Location)
		name := r.FormValue(URLParams.Name)
		svcArea := r.FormValue(URLParams.SvcArea)
		urlName := r.FormValue(URLParams.URLName)
		viewType := r.FormValue(URLParams.Type)

		//prepare the data
		data[TplParamDesc] = desc
		data[TplParamEducation] = education
		data[TplParamExperience] = experience
		data[TplParamLocation] = location
		data[TplParamName] = name
		data[TplParamSvcArea] = svcArea
		data[TplParamType] = viewType
		data[TplParamURLName] = urlName

		//validate the data
		form := ProviderForm{
			Description: desc,
			Education:   education,
			Experience:  experience,
			Location:    location,
			NameForm: NameForm{
				Name: name,
			},
			ServiceArea: svcArea,
			URLName:     urlName,
		}
		ok = s.validateForm(w, r.WithContext(ctx), tpl, data, errs, form, true)
		if !ok {
			return
		}

		//handle the uploaded files
		ctx, uploadBanner, err := s.processFileUploadBase64(r.WithContext(ctx), URLParams.ImgBanner, provider.FormatUserID())
		if err != nil {
			logger.Errorw("upload file", "error", err, "file")
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}
		ctx, uploadLogo, err := s.processFileUploadBase64(r.WithContext(ctx), URLParams.ImgLogo, provider.FormatUserID())
		if err != nil {
			logger.Errorw("upload file", "error", err, "file")
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//populate from the form
		provider.Description = form.Description
		provider.Education = form.Education
		provider.Experience = form.Experience
		provider.Location = form.Location
		provider.SetName(form.Name)

		//check if the friendly url name exists
		if form.URLName != "" && provider.URLNameFriendly != form.URLName {
			ctx, ok, err = URLNameFriendlyExists(ctx, s.db, form.URLName)
			if err != nil {
				logger.Errorw("provider url name exists", "error", err, "provider", provider)
				data[TplParamErr] = GetErrText(Err)
				s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
				return
			}
			if ok {
				logger.Debugw("url name exists", "name", form.URLName)
				errs[string(FieldErrURLName)] = GetErrText(ErrURLNameDup)
				s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
				return
			}
		}
		provider.URLNameFriendly = form.URLName

		//convert the service area
		provider.ServiceArea = form.ServiceArea

		//check if delete or explicitly set
		if uploadBanner != nil {
			provider.SetImgBanner(uploadBanner.GetFile())
		} else if imgDelBanner {
			provider.DeleteImgBanner()
		}
		if uploadLogo != nil {
			provider.SetImgLogo(uploadLogo.GetFile())
		} else if imgDelLogo {
			provider.DeleteImgLogo()
		}

		//save the provider
		ctx, err = SaveProvider(ctx, s.getDB(), provider.Provider)
		if err != nil {
			logger.Errorw("save provider", "error", err, "provider", provider)
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//success
		s.SetCookieMsg(w, MsgUpdateSuccess)
		url := s.checkClientView(data, provider, provider.GetURLProfileType(viewType))
		http.Redirect(w, r.WithContext(ctx), url, http.StatusSeeOther)
	}
}

//profile domain steps
var stepsProfileDomain = struct {
	StepDel string
	StepUpd string
}{
	StepDel: "stepDel",
	StepUpd: "stepUpd",
}

//handle the profile domain page
func (s *Server) handleDashboardProfileDomain() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateDashboard(ctx, "profile-domain.html")
		})
		ctx, provider, data, errs, ok := s.createTemplateDataDashboard(w, r.WithContext(ctx), tpl, true)
		if !ok {
			return
		}
		data[TplParamSvcAreaStrs] = ListServiceAreaStrs()

		//setup the breadcrumbs
		breadcrumbs := []breadcrumb{
			{"Profile", ""},
		}
		data[TplParamBreadcrumbs] = breadcrumbs
		data[TplParamActiveNav] = provider.GetURLProfile()
		data[TplParamFormAction] = provider.GetURLProfileDomain()
		data[TplParamSteps] = stepsProfileDomain

		//handle the input
		domain := r.FormValue(URLParams.Domain)
		flag := r.FormValue(URLParams.Flag) == "true"
		step := r.FormValue(URLParams.Step)

		//prepare the data
		data[TplParamDomain] = domain
		data[TplParamFlag] = flag

		//check the method
		if step == "" && r.Method == http.MethodGet {
			if provider.Domain != nil {
				data[TplParamFlag] = strings.Count(*provider.Domain, ".") == 1
				data[TplParamDomain] = &provider.Domain
			} else {
				data[TplParamFlag] = true
				data[TplParamDomain] = ""
			}
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//execute the correct operation
		domainChanged := false
		switch step {
		case stepsProfileDomain.StepDel:
			//delete the domain
			domainChanged = provider.Domain != nil
			provider.SetDomain(nil)
			ctx, err := SaveProvider(ctx, s.getDB(), provider.Provider)
			if err != nil {
				logger.Errorw("save provider", "error", err, "provider", provider)
				data[TplParamErr] = GetErrText(Err)
				s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
				return
			}
			s.SetCookieMsg(w, MsgUpdateSuccess)
		case stepsProfileDomain.StepUpd:
			//validate the data
			form := ProviderDomainForm{
				Domain: domain,
			}
			ok = s.validateForm(w, r.WithContext(ctx), tpl, data, errs, form, true)
			if !ok {
				return
			}

			//check if the domain exists
			if form.Domain != "" {
				domainChanged = provider.Domain == nil || *provider.Domain != form.Domain
				if provider.Domain == nil || (provider.Domain != nil && *provider.Domain != form.Domain) {
					ctx, ok, err := DomainExists(ctx, s.db, form.Domain)
					if err != nil {
						logger.Errorw("provider domain exists", "error", err, "provider", provider)
						data[TplParamErr] = GetErrText(Err)
						s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
						return
					}
					if ok {
						logger.Debugw("domain exists", "name", form.Domain)
						errs[string(FieldErrDomain)] = GetErrText(ErrDomainDup)
						s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
						return
					}
				}
				provider.SetDomain(&form.Domain)
			} else {
				domainChanged = provider.Domain != nil
				provider.SetDomain(nil)
			}

			//save the provider
			ctx, err := SaveProvider(ctx, s.getDB(), provider.Provider)
			if err != nil {
				logger.Errorw("save provider", "error", err, "provider", provider)
				data[TplParamErr] = GetErrText(Err)
				s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
				return
			}
			s.SetCookieFlag(w, "success")
		default:
			logger.Errorw("invalid step", "id", provider.ID, "step", step)
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//queue the emails
		if domainChanged {
			ctx, err := s.queueEmailsDomain(ctx, provider)
			if err != nil {
				logger.Errorw("queue email domain", "error", err)
				data[TplParamErr] = GetErrText(Err)
				s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
				return
			}
		}

		//success
		http.Redirect(w, r.WithContext(ctx), provider.GetURLProfile(), http.StatusSeeOther)
	}
}

//handle the service add page
func (s *Server) handleDashboardServiceAdd() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateDashboard(ctx, "service-add.html")
		})
		ctx, provider, data, errs, ok := s.createTemplateDataDashboard(w, r.WithContext(ctx), tpl, true)
		if !ok {
			return
		}

		//setup the breadcrumbs
		breadcrumbs := []breadcrumb{
			{"Services", provider.GetURLServices()},
			{"Add Service", ""},
		}
		data[TplParamBreadcrumbs] = breadcrumbs
		data[TplParamActiveNav] = provider.GetURLServices()

		//check the method
		if r.Method == http.MethodGet {
			data[TplParamApptOnly] = true
			data[TplParamDesc] = ""
			data[TplParamDuration] = ""
			data[TplParamEnableZoom] = false
			data[TplParamInterval] = strconv.Itoa(ServiceIntervalDefault)
			data[TplParamLocation] = provider.Location
			data[TplParamLocationType] = ServiceLocationTypeRemote
			data[TplParamName] = ""
			data[TplParamNote] = ""
			data[TplParamPadding] = 0
			data[TplParamPaddingInitial] = 1
			data[TplParamPaddingInitialUnit] = PaddingUnitHours
			data[TplParamPrice] = ""
			data[TplParamPriceType] = ""
			data[TplParamURLVideo] = ""
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//handle the input
		apptOnly := r.FormValue(URLParams.ApptOnly) == "on"
		desc := r.FormValue(URLParams.Desc)
		durationStr := r.FormValue(URLParams.Duration)
		enableZoom := r.FormValue(URLParams.EnableZoom) == "on"
		intervalStr := r.FormValue(URLParams.Interval)
		location := r.FormValue(URLParams.Location)
		locationTypeStr := r.FormValue(URLParams.LocationType)
		name := r.FormValue(URLParams.Name)
		note := r.FormValue(URLParams.Note)
		paddingStr := r.FormValue(URLParams.Padding)
		paddingInitialStr := r.FormValue(URLParams.PaddingInitial)
		paddingInitialUnitStr := r.FormValue(URLParams.PaddingInitialUnit)
		priceStr := r.FormValue(URLParams.Price)
		priceTypeStr := r.FormValue(URLParams.PriceType)
		urlVideo := r.FormValue(URLParams.URLVideo)

		//prepare the data
		data[TplParamApptOnly] = apptOnly
		data[TplParamDesc] = desc
		data[TplParamDuration] = durationStr
		data[TplParamEnableZoom] = enableZoom
		data[TplParamInterval] = intervalStr
		data[TplParamLocation] = location
		data[TplParamLocationType] = locationTypeStr
		data[TplParamName] = name
		data[TplParamNote] = note
		data[TplParamPadding] = paddingStr
		data[TplParamPaddingInitial] = paddingInitialStr
		data[TplParamPaddingInitialUnit] = paddingInitialUnitStr
		data[TplParamPrice] = priceStr
		data[TplParamPriceType] = priceTypeStr
		data[TplParamURLVideo] = urlVideo

		//validate the data
		user := provider.GetUser()
		form := ServiceForm{
			ApptOnly:     apptOnly,
			Description:  desc,
			Duration:     durationStr,
			EnableZoom:   user.ZoomToken != nil && enableZoom,
			Interval:     intervalStr,
			Location:     location,
			LocationType: locationTypeStr,
			NameForm: NameForm{
				Name: name,
			},
			Note:               note,
			Padding:            paddingStr,
			PaddingInitial:     paddingInitialStr,
			PaddingInitialUnit: paddingInitialUnitStr,
			Price:              priceStr,
			PriceType:          priceTypeStr,
			URLVideo:           urlVideo,
		}
		ok = s.validateForm(w, r.WithContext(ctx), tpl, data, errs, form, true)
		if !ok {
			return
		}

		//handle the uploaded files
		ctx, uploads, ok, err := s.processFileUploads(r.WithContext(ctx), URLParams.Img, provider.FormatUserID())
		if err != nil {
			logger.Errorw("upload file", "error", err)
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//populate from the form
		svc := s.createService(provider, &form)

		//process the images
		if ok {
			svc.SetImgs(uploads)
		}

		//check for the maximum number of images
		if len(svc.Imgs) > MaxImgSvcCount {
			data[TplParamErr] = GetErrText(ErrSvcImgCount, MaxImgSvcCount)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//save the service
		now := data[TplParamCurrentTime].(time.Time)
		ctx, err = SaveService(ctx, s.getDB(), provider.Provider, svc, now)
		if err != nil {
			logger.Errorw("save service", "error", err, "service", svc, "id", provider.ID)
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//success
		s.SetCookieMsg(w, MsgSvcAdd)
		http.Redirect(w, r.WithContext(ctx), provider.GetURLServices(), http.StatusSeeOther)
	}
}

//handle the service edit page
func (s *Server) handleDashboardServiceEdit() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template

	//steps on the page
	steps := struct {
		StepDel string
		StepUpd string
	}{
		StepDel: "stepDel",
		StepUpd: "stepUpd",
	}
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateDashboard(ctx, "service-edit.html")
		})
		ctx, provider, data, errs, ok := s.createTemplateDataDashboard(w, r.WithContext(ctx), tpl, true)
		if !ok {
			return
		}

		//setup the breadcrumbs
		breadcrumbs := []breadcrumb{
			{"Services", provider.GetURLServices()},
			{"Edit Service", ""},
		}
		data[TplParamBreadcrumbs] = breadcrumbs
		data[TplParamActiveNav] = provider.GetURLServices()
		data[TplParamFormAction] = provider.GetURLServiceEdit(nil)
		data[TplParamSteps] = steps
		data[TplParamTypes] = ImgViewTypes
		data[TplParamClientView] = r.FormValue(URLParams.Client)

		//validate the id
		idStr := r.FormValue(URLParams.SvcID)
		svcID := uuid.FromStringOrNil(idStr)
		if svcID == uuid.Nil {
			logger.Warnw("invalid uuid", "id", idStr)
			s.SetCookieErr(w, Err)
			http.Redirect(w, r.WithContext(ctx), provider.GetURLServices(), http.StatusSeeOther)
			return
		}

		//load the service
		now := data[TplParamCurrentTime].(time.Time)
		ctx, svc, ok := s.loadTemplateService(w, r.WithContext(ctx), tpl, data, provider, &svcID, now)
		if !ok {
			return
		}

		//prepare the confirmation modal
		data[TplParamConfirmMsg] = GetMsgText(MsgSvcDelConfirm)
		data[TplParamConfirmSubmitName] = URLParams.Step
		data[TplParamConfirmSubmitValue] = steps.StepDel

		//load the provider users
		ctx, users, err := ListProviderUsersByProviderID(ctx, s.getDB(), provider.ID, true)
		if err != nil {
			logger.Errorw("load provider users", "error", err, "providerId", provider.ID, "id", svcID)
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}
		data[TplParamUsers] = users

		//check the method
		if r.Method == http.MethodGet {
			data[TplParamApptOnly] = svc.IsApptOnly()
			data[TplParamDesc] = svc.Description
			data[TplParamDuration] = strconv.Itoa(svc.Duration)
			data[TplParamEnableZoom] = svc.EnableZoom
			data[TplParamInterval] = strconv.Itoa(svc.Interval)
			data[TplParamName] = svc.Name
			data[TplParamNote] = svc.Note
			data[TplParamPadding] = strconv.Itoa(svc.Padding)
			data[TplParamPaddingInitial] = strconv.Itoa(svc.PaddingInitial)
			data[TplParamPaddingInitialUnit] = svc.PaddingInitialUnit
			data[TplParamPrice] = svc.Price
			data[TplParamPriceType] = svc.PriceType
			data[TplParamURLVideo] = svc.URLVideo
			data[TplParamType] = r.FormValue(URLParams.Type)

			//default the location if not set
			data[TplParamLocationType] = svc.LocationType
			data[TplParamLocation] = svc.Location
			if svc.Location == "" {
				data[TplParamLocation] = provider.Location
			}
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//handle the input
		apptOnly := r.FormValue(URLParams.ApptOnly) == "on"
		desc := r.FormValue(URLParams.Desc)
		durationStr := r.FormValue(URLParams.Duration)
		enableZoom := r.FormValue(URLParams.EnableZoom) == "on"
		imgIdxs := r.Form[URLParams.ImgIdx]
		intervalStr := r.FormValue(URLParams.Interval)
		location := r.FormValue(URLParams.Location)
		locationTypeStr := r.FormValue(URLParams.LocationType)
		name := r.FormValue(URLParams.Name)
		note := r.FormValue(URLParams.Note)
		paddingStr := r.FormValue(URLParams.Padding)
		paddingInitialStr := r.FormValue(URLParams.PaddingInitial)
		paddingInitialUnitStr := r.FormValue(URLParams.PaddingInitialUnit)
		priceStr := r.FormValue(URLParams.Price)
		priceTypeStr := r.FormValue(URLParams.PriceType)
		urlVideo := r.FormValue(URLParams.URLVideo)
		step := r.FormValue(URLParams.Step)
		viewType := r.FormValue(URLParams.Type)

		//prepare the data
		data[TplParamApptOnly] = apptOnly
		data[TplParamDesc] = desc
		data[TplParamDuration] = durationStr
		data[TplParamEnableZoom] = enableZoom
		data[TplParamInterval] = intervalStr
		data[TplParamLocation] = location
		data[TplParamLocationType] = locationTypeStr
		data[TplParamName] = name
		data[TplParamNote] = note
		data[TplParamPadding] = paddingStr
		data[TplParamPaddingInitial] = paddingInitialStr
		data[TplParamPaddingInitialUnit] = paddingInitialUnitStr
		data[TplParamPrice] = priceStr
		data[TplParamPriceType] = priceTypeStr
		data[TplParamURLVideo] = urlVideo
		data[TplParamType] = viewType

		//validate the data
		user := provider.GetUser()
		form := ServiceForm{
			ApptOnly:     apptOnly,
			Description:  desc,
			Duration:     durationStr,
			EnableZoom:   user.ZoomToken != nil && enableZoom,
			Interval:     intervalStr,
			Location:     location,
			LocationType: locationTypeStr,
			NameForm: NameForm{
				Name: name,
			},
			Note:               note,
			Padding:            paddingStr,
			PaddingInitial:     paddingInitialStr,
			PaddingInitialUnit: paddingInitialUnitStr,
			Price:              priceStr,
			PriceType:          priceTypeStr,
			URLVideo:           urlVideo,
		}
		ok = s.validateForm(w, r.WithContext(ctx), tpl, data, errs, form, true)
		if !ok {
			return
		}

		//execute the correct operation
		switch step {
		case steps.StepDel:
			//delete the service
			ctx, count, err := DeleteService(ctx, s.getDB(), provider.ID, svc.ID)
			if err != nil {
				logger.Errorw("delete service", "error", err, "providerId", provider.ID, "id", svc.ID)
				data[TplParamErr] = GetErrText(Err)
				s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
				return
			}

			//check for an error
			if count > 0 {
				data[TplParamErr] = GetErrText(ErrSvcExist, count)
				s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
				return
			}
		case steps.StepUpd:
			//populate from the form
			svc.SetFields(apptOnly, form.Name, form.Description, form.Note, form.Price, form.PriceType, form.Duration, form.LocationType, form.Location, form.Padding, form.PaddingInitial, form.PaddingInitialUnit, form.Interval, form.EnableZoom, form.URLVideo)

			//handle the delete and re-ordering of any images
			svc.ProcessImgIndices(imgIdxs)

			//handle the uploaded files
			ctx, uploads, ok, err := s.processFileUploads(r.WithContext(ctx), URLParams.Img, provider.FormatUserID())
			if err != nil {
				logger.Errorw("upload file", "error", err)
				data[TplParamErr] = GetErrText(Err)
				s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
				return
			}
			if ok {
				svc.AddImgs(uploads)
			}

			//check for the maximum number of images
			if len(svc.Imgs) > MaxImgSvcCount {
				data[TplParamErr] = GetErrText(ErrSvcImgCount, MaxImgSvcCount)
				s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
				return
			}

			//update the service
			now := data[TplParamCurrentTime].(time.Time)
			ctx, err = SaveService(ctx, s.getDB(), provider.Provider, svc.Service, now)
			if err != nil {
				logger.Errorw("save service", "error", err, "service", svc, "providerId", provider.ID, "id", svc.ID)
				data[TplParamErr] = GetErrText(Err)
				s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
				return
			}
		default:
			logger.Errorw("invalid step", "id", svc.ID, "step", step)
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//success
		s.SetCookieMsg(w, MsgUpdateSuccess)
		url := s.checkClientView(data, provider, provider.GetURLServices())
		http.Redirect(w, r.WithContext(ctx), url, http.StatusSeeOther)
	}
}

//handle the service users page
func (s *Server) handleDashboardServiceUsers() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template

	//steps on the page
	steps := struct {
		StepAdd string
		StepDel string
	}{
		StepAdd: "stepAdd",
		StepDel: "stepDel",
	}
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateDashboard(ctx, "service-users.html")
		})
		ctx, provider, data, _, ok := s.createTemplateDataDashboard(w, r.WithContext(ctx), tpl, true)
		if !ok {
			return
		}

		//setup the breadcrumbs
		breadcrumbs := []breadcrumb{
			{"Services", provider.GetURLServices()},
			{"Edit Service", provider.GetURLServiceEdit(nil)},
			{"Manage Team Members", ""},
		}
		data[TplParamBreadcrumbs] = breadcrumbs
		data[TplParamActiveNav] = provider.GetURLServices()
		data[TplParamFormAction] = provider.GetURLServiceUsers(nil)
		data[TplParamSteps] = steps

		//validate the id
		svcIDStr := r.FormValue(URLParams.SvcID)
		svcID := uuid.FromStringOrNil(svcIDStr)
		if svcID == uuid.Nil {
			logger.Warnw("invalid uuid", "id", svcIDStr)
			s.SetCookieErr(w, Err)
			http.Redirect(w, r.WithContext(ctx), provider.GetURLServices(), http.StatusSeeOther)
			return
		}
		breadcrumbs[1].URL = provider.GetURLServiceEdit(&svcID)

		//load the service
		now := data[TplParamCurrentTime].(time.Time)
		ctx, svc, ok := s.loadTemplateService(w, r.WithContext(ctx), tpl, data, provider, &svcID, now)
		if !ok {
			return
		}

		//load the provider users
		ctx, users, err := ListProviderUsersByProviderID(ctx, s.getDB(), provider.ID, false)
		if err != nil {
			logger.Errorw("load provider users", "error", err, "providerId", provider.ID, "id", svc.ID)
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}
		data[TplParamUsers] = users

		//load the users associated with the service
		ctx, svcUsers, err := ListProviderUsersForService(ctx, s.getDB(), provider.ID, svc.ID)
		if err != nil {
			logger.Errorw("load service provider users", "error", err, "providerId", provider.ID, "id", svc.ID)
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//process the users data
		type SvcUserData struct {
			*ProviderUser
			ServiceProviderUserID *uuid.UUID
		}
		svcUserDatas := make([]*SvcUserData, 0, 2)
		for _, user := range users {
			svcUserData := &SvcUserData{
				ProviderUser: user,
			}
			svcUser, ok := svcUsers[*svcUserData.ID]
			if ok {
				svcUserData.ServiceProviderUserID = svcUser.ID
			}
			svcUserDatas = append(svcUserDatas, svcUserData)
		}
		data[TplParamSvcUsers] = svcUserDatas

		//check the method
		if r.Method == http.MethodGet {
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//handle the input
		userIDStr := r.FormValue(URLParams.UserID)
		step := r.FormValue(URLParams.Step)

		//parse the user id
		userID, err := uuid.FromString(userIDStr)
		if err != nil {
			logger.Errorw("parse user id", "error", err, "id", userIDStr)
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//execute the correct operation
		switch step {
		case steps.StepAdd:
			ctx, err := AddProviderUserToService(ctx, s.getDB(), provider.ID, svc.ID, &userID)
			if err != nil {
				logger.Errorw("add service provider user", "error", err, "providerId", provider.ID, "serviceId", svc.ID, "id", userID)
				data[TplParamErr] = GetErrText(Err)
				s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
				return
			}
		case steps.StepDel:
			ctx, err := DeleteProviderUserFromService(ctx, s.getDB(), provider.ID, svc.ID, &userID)
			if err != nil {
				logger.Errorw("delete service provider user", "error", err, "providerId", provider.ID, "serviceId", svc.ID, "id", userID)
				data[TplParamErr] = GetErrText(Err)
				s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
				return
			}
		default:
			logger.Errorw("invalid step", "id", svc.ID, "step", step)
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//success
		s.SetCookieMsg(w, MsgUpdateSuccess)
		http.Redirect(w, r.WithContext(ctx), provider.GetURLServiceUsers(svc.ID), http.StatusSeeOther)
	}
}

//handle the services page
func (s *Server) handleDashboardServices() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template

	//steps on the page
	steps := struct {
		StepUp   string
		StepDown string
	}{
		StepUp:   "stepUp",
		StepDown: "stepDown",
	}
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateDashboard(ctx, "services.html")
		})
		ctx, provider, data, _, ok := s.createTemplateDataDashboard(w, r.WithContext(ctx), tpl, true)
		if !ok {
			return
		}

		//setup the breadcrumbs
		breadcrumbs := []breadcrumb{
			{"Services", ""},
		}
		data[TplParamBreadcrumbs] = breadcrumbs
		data[TplParamActiveNav] = provider.GetURLServices()
		data[TplParamFormAction] = provider.GetURLServices()
		data[TplParamSteps] = steps
		data[TplParamClientView] = r.FormValue(URLParams.Client)

		//load the services
		ctx, svcs, ok := s.loadTemplateServices(w, r.WithContext(ctx), tpl, data, provider)
		if !ok {
			return
		}
		count := len(svcs) - 1
		data[TplParamCount] = count

		//check the method
		if r.Method == http.MethodGet {
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//handle the input
		step := r.FormValue(URLParams.Step)

		//validate the id
		idStr := r.FormValue(URLParams.SvcID)
		svcID := uuid.FromStringOrNil(idStr)
		if svcID == uuid.Nil {
			logger.Warnw("invalid uuid", "id", idStr)
			s.SetCookieErr(w, Err)
			http.Redirect(w, r.WithContext(ctx), provider.GetURLServices(), http.StatusSeeOther)
			return
		}

		//find the index of the service
		svcIDs := make([]*uuid.UUID, len(svcs))
		svcIdx := -1
		for idx, service := range svcs {
			if service.ID.String() == idStr {
				svcIdx = idx
			}
			svcIDs[idx] = service.ID
		}
		if svcIdx == -1 {
			logger.Errorw("invalid service id", "id", svcID)
			s.SetCookieErr(w, Err)
			http.Redirect(w, r.WithContext(ctx), provider.GetURLServices(), http.StatusSeeOther)
			return
		}

		//check the step
		switch step {
		case steps.StepUp:
			if svcIdx == 0 {
				logger.Errorw("invalid step up", "id", svcID)
				s.SetCookieErr(w, Err)
				http.Redirect(w, r.WithContext(ctx), provider.GetURLServices(), http.StatusSeeOther)
				return
			}

			//swap services
			svcID := svcIDs[svcIdx-1]
			svcIDs[svcIdx-1] = svcIDs[svcIdx]
			svcIDs[svcIdx] = svcID
		case steps.StepDown:
			if svcIdx == count {
				logger.Errorw("invalid step down", "id", svcID)
				s.SetCookieErr(w, Err)
				http.Redirect(w, r.WithContext(ctx), provider.GetURLServices(), http.StatusSeeOther)
				return
			}

			//swap services
			svcID := svcIDs[svcIdx+1]
			svcIDs[svcIdx+1] = svcIDs[svcIdx]
			svcIDs[svcIdx] = svcID
		default:
			logger.Errorw("invalid step", "id", svcID, "step", step)
			s.SetCookieErr(w, Err)
			http.Redirect(w, r.WithContext(ctx), provider.GetURLServices(), http.StatusSeeOther)
			return
		}

		//update the service indices
		_, err := UpdateServiceIndices(ctx, s.getDB(), svcIDs)
		if err != nil {
			logger.Errorw("update service indices", "error", err)
			s.SetCookieErr(w, Err)
			http.Redirect(w, r.WithContext(ctx), provider.GetURLServices(), http.StatusSeeOther)
			return
		}
		http.Redirect(w, r.WithContext(ctx), provider.GetURLServices(), http.StatusSeeOther)
	}
}

//handle the testimonial add page
func (s *Server) handleDashboardTestimonialAdd() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateDashboard(ctx, "testimonial-add.html")
		})
		ctx, provider, data, errs, ok := s.createTemplateDataDashboard(w, r.WithContext(ctx), tpl, true)
		if !ok {
			return
		}

		//setup the breadcrumbs
		breadcrumbs := []breadcrumb{
			{"Testimonials", provider.GetURLTestimonials()},
			{"Add Testimonial", ""},
		}
		data[TplParamBreadcrumbs] = breadcrumbs
		data[TplParamActiveNav] = provider.GetURLTestimonials()
		data[TplParamClientView] = r.FormValue(URLParams.Client)

		//check the method
		if r.Method == http.MethodGet {
			data[TplParamCity] = ""
			data[TplParamName] = ""
			data[TplParamText] = ""
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//handle the input
		city := r.FormValue(URLParams.City)
		imgDel := r.FormValue(URLParams.ImgDel) == "true"
		name := r.FormValue(URLParams.Name)
		text := r.FormValue(URLParams.Text)

		//prepare the data
		data[TplParamCity] = city
		data[TplParamName] = name
		data[TplParamText] = text

		//validate the data
		form := TestimonialForm{
			NameForm: NameForm{
				Name: name,
			},
			City: city,
			Text: text,
		}
		ok = s.validateForm(w, r.WithContext(ctx), tpl, data, errs, form, true)
		if !ok {
			return
		}

		//handle the uploaded files
		ctx, uploadImg, err := s.processFileUploadBase64(r.WithContext(ctx), URLParams.Img, provider.FormatUserID())
		if err != nil {
			logger.Errorw("upload file", "error", err, "file")
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//populate from the form
		testimonial := &Testimonial{
			ProviderID: provider.ID,
			UserID:     provider.User.ID,
			Name:       form.Name,
			City:       form.City,
			Text:       form.Text,
		}

		//check if delete or explicitly set
		if uploadImg != nil {
			testimonial.SetImg(uploadImg.GetFile())
		} else if imgDel {
			testimonial.DeleteImg()
		}

		//save the testimonial
		ctx, err = SaveTestimonial(ctx, s.getDB(), testimonial)
		if err != nil {
			logger.Errorw("save testimonial", "error", err, "testimonial", testimonial, "id", provider.ID)
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//success
		s.SetCookieMsg(w, MsgTestimonialAdd, testimonial.Name)
		url := s.checkClientView(data, provider, provider.GetURLTestimonials())
		http.Redirect(w, r.WithContext(ctx), url, http.StatusSeeOther)
	}
}

//handle the testimonial edit page
func (s *Server) handleDashboardTestimonialEdit() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template

	//steps on the page
	steps := struct {
		StepDel string
		StepUpd string
	}{
		StepDel: "stepDel",
		StepUpd: "stepUpd",
	}
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateDashboard(ctx, "testimonial-edit.html")
		})
		ctx, provider, data, errs, ok := s.createTemplateDataDashboard(w, r.WithContext(ctx), tpl, true)
		if !ok {
			return
		}

		//setup the breadcrumbs
		breadcrumbs := []breadcrumb{
			{"Testimonials", provider.GetURLTestimonials()},
			{"Edit Testimonial", ""},
		}
		data[TplParamBreadcrumbs] = breadcrumbs
		data[TplParamActiveNav] = provider.GetURLTestimonials()
		data[TplParamFormAction] = provider.GetURLTestimonialEdit()
		data[TplParamSteps] = steps
		data[TplParamClientView] = r.FormValue(URLParams.Client)

		//validate the id
		idStr := r.FormValue(URLParams.ID)
		testimonialID := uuid.FromStringOrNil(idStr)
		if testimonialID == uuid.Nil {
			logger.Warnw("invalid uuid", "id", idStr)
			s.SetCookieErr(w, Err)
			http.Redirect(w, r.WithContext(ctx), provider.GetURLTestimonials(), http.StatusSeeOther)
			return
		}

		//load the testimonial
		ctx, testimonial, err := LoadTestimonialByProviderIDAndID(ctx, s.getDB(), provider.Provider, &testimonialID)
		if err != nil {
			logger.Errorw("load testimonial", "error", err, "id", testimonialID)
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}
		data[TplParamTestimonial] = s.createTestimonialUI(testimonial)

		//prepare the confirmation modal
		data[TplParamConfirmMsg] = GetMsgText(MsgTestimonialDelConfirm)
		data[TplParamConfirmSubmitName] = URLParams.Step
		data[TplParamConfirmSubmitValue] = steps.StepDel

		//check the method
		if r.Method == http.MethodGet {
			data[TplParamCity] = testimonial.City
			data[TplParamName] = testimonial.Name
			data[TplParamText] = testimonial.Text
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//handle the input
		city := r.FormValue(URLParams.City)
		imgDel := r.FormValue(URLParams.ImgDel) == "true"
		name := r.FormValue(URLParams.Name)
		text := r.FormValue(URLParams.Text)
		step := r.FormValue(URLParams.Step)

		//prepare the data
		data[TplParamCity] = city
		data[TplParamName] = name
		data[TplParamText] = text

		//validate the data
		form := TestimonialForm{
			NameForm: NameForm{
				Name: name,
			},
			City: city,
			Text: text,
		}
		ok = s.validateForm(w, r.WithContext(ctx), tpl, data, errs, form, true)
		if !ok {
			return
		}

		//execute the correct operation
		var msgKey MsgKey
		switch step {
		case steps.StepDel:
			//delete the testimonial
			ctx, err := DeleteTestimonial(ctx, s.getDB(), provider.ID, testimonial.ID)
			if err != nil {
				logger.Errorw("delete testimonial", "error", err, "id", testimonial.ID)
				data[TplParamErr] = GetErrText(Err)
				s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
				return
			}
			msgKey = MsgTestimonialDel
		case steps.StepUpd:
			//populate from the form
			testimonial.Name = form.Name
			testimonial.City = form.City
			testimonial.Text = form.Text

			//handle the uploaded files
			ctx, uploadImg, err := s.processFileUploadBase64(r.WithContext(ctx), URLParams.Img, provider.FormatUserID())
			if err != nil {
				logger.Errorw("upload file", "error", err)
				data[TplParamErr] = GetErrText(Err)
				s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
				return
			}

			//check if delete or explicitly set
			if uploadImg != nil {
				testimonial.SetImg(uploadImg.GetFile())
			} else if imgDel {
				testimonial.DeleteImg()
			}

			//update the testimonial
			ctx, err = SaveTestimonial(ctx, s.getDB(), testimonial)
			if err != nil {
				logger.Errorw("save testimonial", "error", err, "testimonial", testimonial)
				data[TplParamErr] = GetErrText(Err)
				s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
				return
			}
			msgKey = MsgTestimonialEdit
		default:
			logger.Errorw("invalid step", "id", testimonial.ID, "step", step)
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//success
		s.SetCookieMsg(w, msgKey, testimonial.Name)
		url := s.checkClientView(data, provider, provider.GetURLTestimonials())
		http.Redirect(w, r.WithContext(ctx), url, http.StatusSeeOther)
	}
}

//handle the testimonials page
func (s *Server) handleDashboardTestimonials() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, _ := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateDashboard(ctx, "testimonials.html")
		})
		ctx, provider, data, _, ok := s.createTemplateDataDashboard(w, r.WithContext(ctx), tpl, true)
		if !ok {
			return
		}

		//setup the breadcrumbs
		breadcrumbs := []breadcrumb{
			{"Testimonials", ""},
		}
		data[TplParamBreadcrumbs] = breadcrumbs
		data[TplParamActiveNav] = provider.GetURLTestimonials()
		data[TplParamFormAction] = provider.GetURLTestimonialEdit()
		data[TplParamClientView] = r.FormValue(URLParams.Client)
		data[TplParamURLTestimonialAdd] = s.checkClientView(data, provider, provider.GetURLTestimonialAdd())

		//load the testimonials
		ctx, _, ok = s.loadTemplateTestimonials(w, r.WithContext(ctx), tpl, data, provider)
		if !ok {
			return
		}
		s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
	}
}

//handle the user add page
func (s *Server) handleDashboardUserAdd() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateDashboard(ctx, "user-add.html")
		})
		ctx, provider, data, errs, ok := s.createTemplateDataDashboard(w, r.WithContext(ctx), tpl, true)
		if !ok {
			return
		}

		//setup the breadcrumbs
		breadcrumbs := []breadcrumb{
			{"Users", provider.GetURLUsers()},
			{"Add Team Member", ""},
		}
		data[TplParamBreadcrumbs] = breadcrumbs
		data[TplParamActiveNav] = provider.GetURLUsers()

		//check the method
		if r.Method == http.MethodGet {
			data[TplParamEmail] = ""
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//handle the input
		email := r.FormValue(URLParams.Email)

		//prepare the data
		data[TplParamEmail] = email

		//validate the user
		form := EmailForm{
			Email: strings.TrimSpace(email),
		}
		ok = s.validateForm(w, r.WithContext(ctx), tpl, data, errs, form, true)
		if !ok {
			return
		}

		//check if the email exists
		ctx, ok, userID, err := ProviderLoginExists(ctx, s.getDB(), provider.ID, form.Email)
		if err != nil {
			logger.Errorw("email exists", "error", err, "email", form.Email)
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		} else if ok {
			logger.Debugw("email exists", "email", form.Email)
			errs[string(FieldErrEmail)] = GetErrText(ErrEmailDup)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//save the provider user
		user := &ProviderUser{
			ProviderID: provider.ID,
			Login:      form.Email,
			UserID:     userID,
			Schedule:   provider.Schedule,
		}
		ctx, err = SaveProviderUser(ctx, s.getDB(), user, true)
		if err != nil {
			logger.Errorw("save provider user", "error", err)
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//queue the email
		ctx, err = s.queueEmailProviderUserInvite(ctx, provider, user)
		if err != nil {
			logger.Errorw("queue email provider user invite", "error", err)
			data[TplParamErr] = GetErrText(ErrEmailSend, email)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//success
		msgKey := MsgUserAdd
		if userID == nil {
			msgKey = MsgUserAddNew
		}
		s.SetCookieMsg(w, msgKey, form.Email)
		http.Redirect(w, r.WithContext(ctx), provider.GetURLUsers(), http.StatusSeeOther)
	}
}

//handle the user edit page
func (s *Server) handleDashboardUserEdit() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template

	//steps on the page
	steps := struct {
		StepDel string
	}{
		StepDel: "stepDel",
	}
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateDashboard(ctx, "user-edit.html")
		})
		ctx, provider, data, _, ok := s.createTemplateDataDashboard(w, r.WithContext(ctx), tpl, true)
		if !ok {
			return
		}

		//setup the breadcrumbs
		breadcrumbs := []breadcrumb{
			{"Users", provider.GetURLUsers()},
			{"Edit Team Member", ""},
		}
		data[TplParamBreadcrumbs] = breadcrumbs
		data[TplParamActiveNav] = provider.GetURLUsers()
		data[TplParamFormAction] = provider.GetURLUserEdit()
		data[TplParamSteps] = steps

		//handle the input
		idStr := r.FormValue(URLParams.UserID)
		step := r.FormValue(URLParams.Step)

		//prepare the data
		data[TplParamUserID] = idStr

		//load the provider user
		id := uuid.FromStringOrNil(idStr)
		if id == uuid.Nil {
			logger.Warnw("invalid uuid", "id", idStr)
			s.SetCookieErr(w, Err)
			http.Redirect(w, r.WithContext(ctx), provider.GetURLUsers(), http.StatusSeeOther)
			return
		}
		ctx, providerUser, err := LoadProviderUserByProviderIDAndID(ctx, s.getDB(), provider.ID, &id)
		if err != nil {
			logger.Errorw("load provider user", "error", err, "id", id)
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}
		if providerUser == nil {
			logger.Errorw("no provider user", "id", id)
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}
		data[TplParamUser] = providerUser

		//prepare the confirmation modal
		data[TplParamConfirmMsg] = GetMsgText(MsgUserDelConfirm)
		data[TplParamConfirmSubmitName] = URLParams.Step
		data[TplParamConfirmSubmitValue] = steps.StepDel

		//check the method
		if r.Method == http.MethodGet {
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//execute the correct operation
		var msgKey MsgKey
		switch step {
		case steps.StepDel:
			//delete the provider user
			ctx, err := DeleteUserProvider(ctx, s.getDB(), provider.ID, providerUser.ID)
			if err != nil {
				logger.Errorw("delete provider user", "error", err, "id", providerUser.ID)
				data[TplParamErr] = GetErrText(Err)
				s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
				return
			}
			msgKey = MsgUserDel
		default:
			logger.Errorw("invalid step", "id", providerUser.ID, "step", step)
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//success
		s.SetCookieMsg(w, msgKey, providerUser.Login)
		http.Redirect(w, r.WithContext(ctx), provider.GetURLUsers(), http.StatusSeeOther)
	}
}

//handle the users page
func (s *Server) handleDashboardUsers() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateDashboard(ctx, "users.html")
		})
		ctx, provider, data, _, ok := s.createTemplateDataDashboard(w, r.WithContext(ctx), tpl, true)
		if !ok {
			return
		}

		//setup the breadcrumbs
		breadcrumbs := []breadcrumb{
			{"Users", ""},
		}
		data[TplParamBreadcrumbs] = breadcrumbs
		data[TplParamActiveNav] = provider.GetURLUsers()
		data[TplParamFormAction] = provider.GetURLUserEdit()

		//load the users
		ctx, users, err := ListProviderUsersByProviderID(ctx, s.getDB(), provider.ID, false)
		if err != nil {
			logger.Errorw("list users", "error", err, "id", provider.ID)
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}
		data[TplParamUsers] = users
		s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
	}
}
