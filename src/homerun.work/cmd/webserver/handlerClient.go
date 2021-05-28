package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"path"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

//load a service based on the path
func (s *Server) loadServiceClient(w http.ResponseWriter, r *http.Request, tpl *template.Template, data templateData, provider *providerUI) (*serviceUI, bool) {
	ctx, logger := GetLogger(s.getCtx(r))

	//check for a provider url name
	providerURLName := GetCtxProviderURLName(ctx)
	if providerURLName == "" {
		logger.Warnw("no provider url name")
		s.redirectError(w, r.WithContext(ctx), Err)
		return nil, false
	}

	//check for a service id
	svcIDStr := GetCtxServiceID(ctx)
	if svcIDStr == "" {
		logger.Warnw("no service id")
		s.redirectError(w, r.WithContext(ctx), Err)
		return nil, false
	}
	svcID, err := uuid.FromString(svcIDStr)
	if err != nil {
		logger.Warnw("invalid service id", "error", err, "id", svcIDStr)
		s.redirectError(w, r.WithContext(ctx), Err)
		return nil, false
	}

	//load the service
	ctx, svc, err := LoadServiceByProviderIDAndID(ctx, s.getDB(), provider.ID, &svcID)
	if err != nil {
		logger.Errorw("load service", "error", err, "name", providerURLName, "id", svcID)
		s.redirectError(w, r.WithContext(ctx), Err)
		return nil, false
	}

	//populate the data
	svcUI := s.createServiceUI(provider, svc)
	data[TplParamSvc] = svcUI
	return svcUI, true
}

//load a client web template
func (s *Server) loadWebTemplateClient(ctx context.Context, templateFile string) *template.Template {
	files := []string{path.Join(BaseWebTemplatePathClient, "base.html"), path.Join(BaseWebTemplatePathClient, templateFile)}
	tpl, err := template.New(path.Base(files[0])).Funcs(s.createTemplateFuncs()).ParseFiles(files...)
	if err != nil {
		panic(errors.Wrap(err, fmt.Sprintf("parse template client: %s", templateFile)))
	}
	return tpl
}

//create basic client template data
func (s *Server) createTemplateDataClient(w http.ResponseWriter, r *http.Request, tpl *template.Template) (*providerUI, templateData, map[string]string, bool) {
	ctx, logger := GetLogger(s.getCtx(r))
	data := s.createTemplateData(r.WithContext(ctx))

	//load the provider
	ctx, provider, ok := s.loadProviderByURLName(w, r.WithContext(ctx), tpl, data)
	if !ok {
		return nil, nil, nil, false
	}
	data[TplParamMetaDesc] = provider.Description
	data[TplParamMetaKeywords] = fmt.Sprintf("%s, online, service, schedule, order, invoice, payment", provider.ServiceArea)
	data[TplParamPageTitle] = provider.Name
	data[TplParamProvider] = provider
	data[TplParamClientView] = false

	//load the faq count
	ctx, count, err := CountFaqsByProviderID(ctx, s.getDB(), provider.Provider)
	if err != nil {
		logger.Errorw("count faqs", "error", err, "id", provider.ID)
		data[TplParamErr] = GetErrText(Err)
		s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
		return nil, nil, nil, false
	}
	data[TplParamFaqCount] = count

	//add the errors
	errs := make(map[string]string)
	data[TplParamErrs] = errs

	//check user access
	userID := GetCtxUserID(ctx)
	ctx, providerUser, err := LoadProviderUserByProviderIDAndUserID(ctx, s.getDB(), provider.ID, userID)
	if err != nil {
		logger.Errorw("load provider user", "error", err, "id", provider.ID, "userId", userID)
		data[TplParamErr] = GetErrText(Err)
		s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
		return nil, nil, nil, false
	}
	data[TplParamIsAdmin] = provider.CheckAdmin(userID)
	data[TplParamHasAccess] = provider.CheckUserAccess(userID, providerUser)
	return provider, data, errs, true
}

//handle the client about page
func (s *Server) handleClientAbout() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, _ := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateClient(ctx, "about.html")
		})
		_, data, _, ok := s.createTemplateDataClient(w, r.WithContext(ctx), tpl)
		if !ok {
			return
		}
		s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
	}
}

//handle the client booking page
func (s *Server) handleClientBooking() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateClient(ctx, "order1.html")
		})
		provider, data, errs, ok := s.createTemplateDataClient(w, r.WithContext(ctx), tpl)
		if !ok {
			return
		}

		//read the form
		dateStr := r.FormValue(URLParams.Date)
		timeStr := r.FormValue(URLParams.Time)
		timeZone := r.FormValue(URLParams.TimeZone)
		userIDStr := r.FormValue(URLParams.UserID)

		//default the timezone if not set
		if timeZone == "" {
			timeZone = GetCtxTimeZone(ctx)
		}

		//default the date if not set
		if dateStr == "" && timeStr != "" {
			date := ParseTimeUnixLocal(timeStr, timeZone)
			dateStr = FormatDateLocal(date, timeZone)
		}

		//prepare the data
		data[TplParamDate] = dateStr
		data[TplParamTime] = timeStr
		data[TplParamTimeZone] = timeZone
		data[TplParamUserID] = userIDStr

		//load the service
		svc, ok := s.loadServiceClient(w, r.WithContext(ctx), tpl, data, provider)
		if !ok {
			return
		}
		data[TplParamFormAction] = svc.GetURLBooking()

		//load the users
		ctx, svcUsers, err := ListProviderUsersForService(ctx, s.getDB(), provider.ID, svc.ID)
		if err != nil {
			logger.Errorw("list users", "error", err, "id", provider.ID)
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//process to create a list of users, sorted by the login
		users := make([]*ProviderUser, 0, 2)
		for _, svcUser := range svcUsers {
			users = append(users, svcUser.User)
		}
		sort.Slice(users, func(i, j int) bool {
			return users[i].Login < users[j].Login
		})
		data[TplParamUsers] = users

		//check for a selected user
		if userIDStr != "" {
			userID := uuid.FromStringOrNil(userIDStr)
			if userID == uuid.Nil {
				logger.Errorw("parse user id", "id", userIDStr)
				errs[string(FieldErrUserID)] = GetFieldErrText(string(FieldErrUserID))
				s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
				return
			}
			var providerUser *ProviderUser
			for _, user := range users {
				if user.ID.String() == userIDStr {
					providerUser = user
					break
				}
			}
			provider.ProviderUser = providerUser
		}

		//load the service times
		now := data[TplParamCurrentTime].(time.Time)
		ctx, svcStartDate, _, svcTimes, ok := s.loadTemplateServiceTimes(w, r.WithContext(ctx), tpl, data, errs, provider, svc, dateStr, now, true)
		if !ok {
			return
		}
		svcStartDateStr := FormatDateLocal(svcStartDate, timeZone)
		data[TplParamSvcStartDate] = svcStartDateStr

		//create the structure used for the client ui
		type svcTimeSlot struct {
			Value    int64  `json:"value"`
			Label    string `json:"label"`
			Disabled bool   `json:"disabled"`
			Selected bool   `json:"selected"`
		}
		time := ParseTimeUnixUTC(timeStr)
		svcTimeSlots := make([]*svcTimeSlot, 0, len(svcTimes))
		for _, svcTime := range svcTimes {
			if !svcTime.Hidden {
				svcTimeSlots = append(svcTimeSlots, &svcTimeSlot{
					Value:    svcTime.Start.Unix(),
					Label:    svcTime.FormatPeriodLocal(svc.IsApptOnly(), timeZone),
					Disabled: svcTime.Unavailable,
					Selected: svcTime.Start.Unix() == time.Unix(),
				})
			}
		}
		svcTimesJSON, err := json.Marshal(svcTimeSlots)
		data[TplParamSvcBusyTimes] = string(svcTimesJSON)

		//default the date if necessary
		if dateStr == "" {
			dateStr = svcStartDateStr
			data[TplParamDate] = dateStr
		}

		//check the method
		if r.Method == http.MethodGet {
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//validate the form
		form := ClientBookingDateTimeForm{
			TimeUnixForm: TimeUnixForm{
				Time: timeStr,
			},
			TimeZoneForm: TimeZoneForm{
				TimeZone: timeZone,
			},
		}
		ok = s.validateForm(w, r.WithContext(ctx), tpl, data, errs, form, true)
		if !ok {
			return
		}

		//display the submit page
		url, err := CreateURLRelParams(svc.GetURLBookingSubmit(), URLParams.Time, timeStr, URLParams.TimeZone, timeZone, URLParams.UserID, userIDStr)
		if err != nil {
			logger.Errorw("redirect", "error", svc.GetURLBookingSubmit())
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}
		http.Redirect(w, r.WithContext(ctx), url, http.StatusSeeOther)
	}
}

//handle the client booking submit page
func (s *Server) handleClientBookingSubmit() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateClient(ctx, "order3.html")
		})
		provider, data, errs, ok := s.createTemplateDataClient(w, r.WithContext(ctx), tpl)
		if !ok {
			return
		}

		//read the form
		code := r.FormValue(URLParams.Code)
		desc := r.FormValue(URLParams.Desc)
		email := r.FormValue(URLParams.Email)
		enablePhone := r.FormValue(URLParams.EnablePhone) == "on"
		location := r.FormValue(URLParams.Location)
		name := r.FormValue(URLParams.Name)
		phone := r.FormValue(URLParams.Phone)
		timeStr := r.FormValue(URLParams.Time)
		timeZone := r.FormValue(URLParams.TimeZone)
		userIDStr := r.FormValue(URLParams.UserID)

		//prepare the data
		data[TplParamCode] = code
		data[TplParamDesc] = desc
		data[TplParamEmail] = email
		data[TplParamEnablePhone] = enablePhone
		data[TplParamLocation] = location
		data[TplParamName] = name
		data[TplParamPhone] = phone
		data[TplParamTime] = timeStr
		data[TplParamTimeZone] = timeZone
		data[TplParamUserID] = userIDStr

		//default the timezone if not set
		if timeZone == "" {
			timeZone = GetCtxTimeZone(ctx)
		}

		//load the service
		svc, ok := s.loadServiceClient(w, r.WithContext(ctx), tpl, data, provider)
		if !ok {
			return
		}
		data[TplParamFormAction] = svc.GetURLBookingSubmit()

		//validate the time
		formTime := ClientBookingDateTimeForm{
			TimeUnixForm: TimeUnixForm{
				Time: timeStr,
			},
			TimeZoneForm: TimeZoneForm{
				TimeZone: timeZone,
			},
		}
		ok = s.validateForm(w, r.WithContext(ctx), tpl, data, errs, formTime, true)
		if !ok {
			return
		}
		timeFrom := ParseTimeUnixLocal(timeStr, timeZone)
		data[TplParamSvcTime] = svc.FormatTime(timeFrom, timeZone)

		//create the url to go back
		var err error
		var url string
		if svc.IsApptOnly() {
			url, err = CreateURLRelParams(svc.GetURLBooking(), URLParams.Time, timeStr)
			if err != nil {
				logger.Errorw("create url", "error", err, "url", provider.GetURLBookings())
				data[TplParamErr] = GetErrText(Err)
				s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
				return
			}
		} else {
			url = svc.GetURLService()
		}
		data[TplParamURLPrev] = url

		//check the method
		if r.Method == http.MethodGet {
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//validate the location
		if svc.LocationType.IsLocationProvider() {
			//force the service location
			location = svc.Location
		} else if svc.LocationType.IsLocationClient() {
			//client location required
			if location == "" {
				errs[string(FieldErrLocation)] = GetFieldErrText(string(FieldErrLocation))
				s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
				return
			}
		}

		//process the incoming location
		location = svc.ProcessBookingLocationInput(location)

		//validate the booking
		form := &ClientBookingForm{
			ServiceID:      svc.ID.String(),
			Email:          strings.TrimSpace(email),
			Name:           name,
			Phone:          FormatPhone(phone),
			EnablePhone:    enablePhone,
			Location:       location,
			Description:    desc,
			DescriptionSet: true,
			ClientCreated:  true,
			ClientBookingDateTimeForm: ClientBookingDateTimeForm{
				TimeUnixForm: TimeUnixForm{
					Time: timeStr,
				},
				TimeZoneForm: TimeZoneForm{
					TimeZone: timeZone,
				},
			},
			ProviderNote:    svc.Note,
			ProviderNoteSet: true,
			Code:            code,
		}
		ok = s.validateForm(w, r.WithContext(ctx), tpl, data, errs, form, true)
		if !ok {
			return
		}

		//load the user if set
		var providerUser *ProviderUser
		if userIDStr != "" {
			userID := uuid.FromStringOrNil(userIDStr)
			if userID == uuid.Nil {
				logger.Errorw("invalid user id", "id", userIDStr)
				data[TplParamErr] = GetErrText(Err)
				s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
				return
			}
			ctx, providerUser, err = LoadProviderUserForServiceByProviderIDAndServiceIDAndUserID(ctx, s.getDB(), provider.ID, svc.ID, &userID)
			if err != nil {
				logger.Errorw("load provider user", "error", err, "id", userID)
				data[TplParamErr] = GetErrText(Err)
				s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
				return
			}
		}

		//create the booking
		now := data[TplParamCurrentTime].(time.Time)
		book, ok := s.saveBooking(w, r.WithContext(ctx), tpl, data, errs, provider, providerUser, svc, nil, now, false, form, true)
		if !ok {
			return
		}
		http.Redirect(w, r.WithContext(ctx), book.GetURLConfirmClient(), http.StatusSeeOther)
	}
}

//handle the client booking confirmation page
func (s *Server) handleClientBookingConfirm() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateClient(ctx, "order4.html")
		})
		provider, data, errs, ok := s.createTemplateDataClient(w, r.WithContext(ctx), tpl)
		if !ok {
			return
		}

		//check for the booking id
		bookIDStr := GetCtxBookID(ctx)
		if bookIDStr == "" {
			logger.Errorw("no booking id")
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//load the booking
		ctx, _, ok = s.loadTemplateBook(w, r.WithContext(ctx), tpl, data, errs, bookIDStr, false, false)
		if !ok {
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//load the service
		_, ok = s.loadServiceClient(w, r.WithContext(ctx), tpl, data, provider)
		if !ok {
			return
		}
		s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
	}
}

//handle the client booking cancel page
func (s *Server) handleClientBookingCancel() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateClient(ctx, "order5.html")
		})
		provider, data, errs, ok := s.createTemplateDataClient(w, r.WithContext(ctx), tpl)
		if !ok {
			return
		}
		data[TplParamClientView] = r.FormValue(URLParams.Client)

		//check for the booking id
		bookIDStr := GetCtxBookID(ctx)
		if bookIDStr == "" {
			logger.Errorw("no booking id")
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//load the booking
		ctx, book, ok := s.loadTemplateBook(w, r.WithContext(ctx), tpl, data, errs, bookIDStr, false, false)
		if !ok {
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//load the service
		svc, ok := s.loadServiceClient(w, r.WithContext(ctx), tpl, data, provider)
		if !ok {
			return
		}

		//check the method
		if r.Method == http.MethodGet {
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//ignore if already cancelled
		if book.Deleted {
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//cancel the booking
		now := data[TplParamCurrentTime].(time.Time)
		ok = s.cancelServiceBooking(w, r.WithContext(ctx), tpl, data, errs, provider, svc, book, now, false)
		if !ok {
			return
		}
		ctx = s.setCtxMsg(ctx, MsgBookingCancel)
		s.invokeHdlrGet(s.handleClientIndex(), w, r.WithContext(ctx))
	}
}

//handle the client view booking page
func (s *Server) handleClientBookingView() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateClient(ctx, "order-detail.html")
		})
		provider, data, errs, ok := s.createTemplateDataClient(w, r.WithContext(ctx), tpl)
		if !ok {
			return
		}
		data[TplParamClientView] = r.FormValue(URLParams.Client)

		//check for the booking id
		bookIDStr := GetCtxBookID(ctx)
		if bookIDStr == "" {
			logger.Errorw("no booking id")
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//load the booking
		ctx, book, ok := s.loadTemplateBook(w, r.WithContext(ctx), tpl, data, errs, bookIDStr, false, true)
		if !ok {
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}
		data[TplParamFormAction] = book.GetURLViewClient()

		//load the service
		_, ok = s.loadServiceClient(w, r.WithContext(ctx), tpl, data, provider)
		if !ok {
			return
		}

		//check the method
		if r.Method == http.MethodGet {
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//update the booking
		book.EnableClientPhone = r.FormValue(URLParams.EnablePhone) == "on"
		_, err := UpdateBookingData(ctx, s.getDB(), book.Booking)
		if err != nil {
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//success
		s.SetCookieMsg(w, MsgUpdateSuccess)
		http.Redirect(w, r.WithContext(ctx), book.GetURLViewClient(), http.StatusSeeOther)
	}
}

//handle the client contact page
func (s *Server) handleClientContact() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateClient(ctx, "contact.html")
		})
		provider, data, errs, ok := s.createTemplateDataClient(w, r.WithContext(ctx), tpl)
		if !ok {
			return
		}

		//read the form
		name := r.FormValue(URLParams.Name)
		email := r.FormValue(URLParams.Email)
		phone := r.FormValue(URLParams.Phone)
		text := r.FormValue(URLParams.Text)
		timeZone := r.FormValue(URLParams.TimeZone)

		//prepare the data
		data[TplParamFormAction] = provider.GetURLContactClient()
		data[TplParamName] = name
		data[TplParamEmail] = email
		data[TplParamPhone] = phone
		data[TplParamText] = text

		//check the method
		if r.Method == http.MethodGet {
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//validate the form
		form := ContactForm{
			ClientForm: ClientForm{
				ClientDataForm: ClientDataForm{
					EmailForm: EmailForm{
						Email: strings.TrimSpace(email),
					},
					NameForm: NameForm{
						Name: name,
					},
					Phone: FormatPhone(phone),
				},
				TimeZoneForm: TimeZoneForm{
					TimeZone: timeZone,
				},
			},
			Text: text,
		}
		ok = s.validateForm(w, r.WithContext(ctx), tpl, data, errs, form, true)
		if !ok {
			return
		}

		//populate from the form
		client := &Client{
			ProviderID: provider.ID,
			Email:      form.Email,
			Name:       form.Name,
			Phone:      form.Phone,
			TimeZone:   form.TimeZone,
		}

		//save the client
		ctx, err := SaveClient(ctx, s.getDB(), client)
		if err != nil {
			logger.Errorw("save client", "error", err, "client", client, "id", provider.ID)
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//queue the email
		ctx, err = s.queueEmailContact(ctx, provider, client, text)
		if err != nil {
			logger.Errorw("queue email contact", "error", err, "id", client.ID)
			ctx = s.setCtxErr(ctx, ErrClientInvite)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//success
		s.SetCookieMsg(w, MsgContact)
		http.Redirect(w, r.WithContext(ctx), provider.GetURLProvider(), http.StatusSeeOther)
	}
}

//handle the client direct payment page
func (s *Server) handleClientPaymentDirect() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateClient(ctx, "payment-direct.html")
		})
		provider, data, errs, ok := s.createTemplateDataClient(w, r.WithContext(ctx), tpl)
		if !ok {
			return
		}

		//read the form
		desc := r.FormValue(URLParams.Desc)
		email := r.FormValue(URLParams.Email)
		name := r.FormValue(URLParams.Name)
		phone := r.FormValue(URLParams.Phone)
		priceStr := r.FormValue(URLParams.Price)
		svcIDStr := r.FormValue(URLParams.SvcID)
		timeZone := r.FormValue(URLParams.TimeZone)

		//load the service if specified
		var err error
		var svc *Service
		if svcIDStr != "" {
			svcID := uuid.FromStringOrNil(svcIDStr)
			if svcID == uuid.Nil {
				logger.Errorw("invalid uuid", "id", svcIDStr)
				data[TplParamErr] = GetErrText(Err)
				s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
				return
			}
			ctx, svc, err = LoadServiceByProviderIDAndID(ctx, s.getDB(), provider.ID, &svcID)
			if err != nil {
				logger.Errorw("load service", "id", svcID)
				data[TplParamErr] = GetErrText(Err)
				s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
				return
			}
			data[TplParamSvc] = s.createServiceUI(provider, svc)
		}

		//prepare the data
		data[TplParamFormAction] = provider.GetURLPaymentDirectClient()
		data[TplParamDesc] = desc
		data[TplParamEmail] = email
		data[TplParamName] = name
		data[TplParamPhone] = phone
		data[TplParamPrice] = priceStr

		//check the method
		if r.Method == http.MethodGet {
			data[TplParamDesc] = ""
			data[TplParamEmail] = ""
			data[TplParamName] = ""
			data[TplParamPhone] = ""
			data[TplParamPrice] = ""
			if svc != nil {
				data[TplParamPrice] = svc.ComputePrice()
			}
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

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
			ClientInitiated: true,
			DirectCapture:   false,
		}
		ok = s.validateForm(w, r.WithContext(ctx), tpl, data, errs, form, true)
		if !ok {
			return
		}

		//save the payment
		now := data[TplParamCurrentTime].(time.Time)
		ctx, payment, err := s.savePaymentDirect(ctx, provider, svc, form, now, timeZone)
		if err != nil {
			logger.Errorw("save payment", "error", err)
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}
		http.Redirect(w, r.WithContext(ctx), payment.URL, http.StatusSeeOther)
	}
}

//handle the client faq page
func (s *Server) handleClientFaq() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, _ := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateClient(ctx, "faq.html")
		})
		provider, data, _, ok := s.createTemplateDataClient(w, r.WithContext(ctx), tpl)
		if !ok {
			return
		}

		//load the faqs
		ctx, _, ok = s.loadTemplateFaqs(w, r.WithContext(ctx), tpl, data, provider)
		if !ok {
			return
		}
		s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
	}
}

//handle the client index page
func (s *Server) handleClientIndex() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, _ := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateClient(ctx, "index.html")
		})
		provider, data, _, ok := s.createTemplateDataClient(w, r.WithContext(ctx), tpl)
		if !ok {
			return
		}

		//disable the nav
		data[TplParamNavDisable] = true
		data[TplParamClientView] = r.FormValue(URLParams.Client)

		//load the testimonials
		ctx, _, ok = s.loadTemplateTestimonials(w, r.WithContext(ctx), tpl, data, provider)
		if !ok {
			return
		}

		//load the faqs
		ctx, _, ok = s.loadTemplateFaqs(w, r.WithContext(ctx), tpl, data, provider)
		if !ok {
			return
		}

		//load the services
		ctx, _, ok = s.loadTemplateServices(w, r.WithContext(ctx), tpl, data, provider)
		if !ok {
			return
		}

		//load the schedule
		schedule1, schedule2 := provider.GetScheduleBuckets()
		data[TplParamSchedule1] = schedule1
		data[TplParamSchedule2] = schedule2
		s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
	}
}

//handle the client payment page
func (s *Server) handleClientPayment() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateClient(ctx, "payment.html")
		})
		provider, data, errs, ok := s.createTemplateDataClient(w, r.WithContext(ctx), tpl)
		if !ok {
			return
		}

		//check for a booking id
		var err error
		var payment *Payment
		bookIDStr := GetCtxBookID(ctx)
		if bookIDStr != "" {
			//load the booking
			ctx, book, ok := s.loadTemplateBook(w, r.WithContext(ctx), tpl, data, errs, bookIDStr, false, false)
			if !ok {
				data[TplParamErr] = GetErrText(Err)
				s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
				return
			}

			//check if a payment is supported, otherwise view the order
			if !book.SupportsPayment() {
				http.Redirect(w, r.WithContext(ctx), book.GetURLViewClient(), http.StatusSeeOther)
				return
			}
			data[TplParamFormAction] = book.GetURLPaymentClient()

			//load the service
			_, ok = s.loadServiceClient(w, r.WithContext(ctx), tpl, data, provider)
			if !ok {
				return
			}

			//load the payment
			ctx, payment, err = LoadPaymentByProviderIDAndSecondaryIDAndType(ctx, s.getDB(), provider.ID, book.ID, PaymentTypeBooking)
			if err != nil {
				logger.Errorw("load payment", "error", err, "id", book.ID)
				http.Error(w, "", http.StatusInternalServerError)
				return
			}
			if payment == nil {
				//create a payment
				form := &PaymentForm{
					EmailForm: EmailForm{
						Email: book.Client.Email,
					},
					NameForm: NameForm{
						Name: book.Client.Name,
					},
					Price:           strconv.FormatFloat(float64(book.ComputeServicePrice()), 'f', 2, 32),
					ClientInitiated: true,
					DirectCapture:   false,
				}
				now := data[TplParamCurrentTime].(time.Time)
				ctx, payment, err = s.savePaymentBooking(ctx, provider, book, form, now)
				if err != nil {
					logger.Errorw("save payment", "error", err)
					data[TplParamErr] = GetErrText(Err)
					s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
					return
				}
			}
		} else {
			//assume a payment id
			paymentIDStr := GetCtxPaymentID(ctx)
			if paymentIDStr == "" {
				logger.Errorw("no payment id")
				data[TplParamErr] = GetErrText(Err)
				s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
				return
			}
			paymentID := uuid.FromStringOrNil(paymentIDStr)
			if paymentID == uuid.Nil {
				logger.Errorw("invalid uuid", "id", paymentIDStr)
				data[TplParamErr] = GetErrText(Err)
				s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
				return
			}

			//load the payment
			ctx, payment, err = LoadPaymentByID(ctx, s.getDB(), &paymentID)
			if err != nil {
				logger.Errorw("load payment", "error", err, "id", paymentID)
				http.Error(w, "", http.StatusInternalServerError)
				return
			}

			//load the service
			if payment.ServiceID != "" {
				svcID := uuid.FromStringOrNil(payment.ServiceID)
				if svcID == uuid.Nil {
					logger.Errorw("invalid uuid", "id", payment.ServiceID)
					data[TplParamErr] = GetErrText(Err)
					s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
					return
				}
				ctx, svc, err := LoadServiceByProviderIDAndID(ctx, s.getDB(), provider.ID, &svcID)
				if err != nil {
					logger.Errorw("load service", "id", svcID)
					data[TplParamErr] = GetErrText(Err)
					s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
					return
				}
				data[TplParamSvc] = s.createServiceUI(provider, svc)
			}
		}
		paymentUI := s.createPaymentUI(payment)
		data[TplParamPayment] = paymentUI

		//check for the a payment confirmation
		now := data[TplParamCurrentTime].(time.Time)
		paypalID := r.FormValue(URLParams.PayPalID)
		stripeID := r.FormValue(URLParams.StripeID)
		status := r.FormValue(URLParams.State)
		plaidData := r.FormValue(URLParams.Data)
		if paypalID != "" || stripeID != "" || status != "" || plaidData != "" {
			data[TplParamSuccess] = true

			//process an plaid payment
			if plaidData != "" {
				err = s.createPaymentStripeACH(ctx, provider.StripeToken, payment, plaidData)
				if err != nil {
					logger.Errorw("create stripe ach", "error", err, "id", payment.ID)
					data[TplParamErr] = GetErrText(Err)
					s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
					return
				}
				payment.Captured = &now
			}

			//mark the payment
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

				//queue the emails
				ctx, err = s.queueEmailsPayment(ctx, provider, paymentUI)
				if err != nil {
					logger.Errorw("queue email payment", "error", err)
					data[TplParamErr] = GetErrText(Err)
					s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
					return
				}
			}
			s.SetCookieMsg(w, MsgPaymentClientSuccess)
			http.Redirect(w, r.WithContext(ctx), payment.URL, http.StatusSeeOther)
			return
		}

		//set-up payment
		if provider.PayPalEmail != nil {
			//create a paypal order if necessary
			if payment.PayPalID == nil {
				err = s.createPaymentPayPal(ctx, provider.PayPalEmail, payment)
				if err != nil {
					logger.Errorw("create paypal order", "error", err, "id", payment.ID)
					data[TplParamErr] = GetErrText(Err)
					s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
					return
				}
			}
			data[TplParamPayPalOrderID] = payment.PayPalID
		}
		if provider.StripeToken != nil {
			//create a stripe session if necessary
			if payment.StripeSessionID == nil {
				err = s.createPaymentStripe(ctx, provider.StripeToken, payment)
				if err != nil {
					logger.Errorw("create stripe session", "error", err, "id", payment.ID)
					data[TplParamErr] = GetErrText(Err)
					s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
					return
				}
			}
			data[TplParamStripeAccountID] = payment.StripeAccountID
			data[TplParamStripeSessionID] = payment.StripeSessionID

			//configure ach
			token, err := CreatePlaidLinkToken(ctx, payment.ID, payment.Email)
			if err != nil {
				logger.Errorw("create stripe link token", "error", err, "id", payment.ID)
				data[TplParamErr] = GetErrText(Err)
				s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
				return
			}
			data[TplParamPlaidToken] = token.LinkToken
		}
		if provider.ZelleID != nil {
			data[TplParamZelleID] = provider.ZelleID
		}
		s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
	}
}

//handle the client service page
func (s *Server) handleClientServiceIndex() http.HandlerFunc {
	var o sync.Once
	var tpl *template.Template
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))
		o.Do(func() {
			tpl = s.loadWebTemplateClient(ctx, "service.html")
		})
		provider, data, _, ok := s.createTemplateDataClient(w, r.WithContext(ctx), tpl)
		if !ok {
			return
		}

		//load the service
		svc, ok := s.loadServiceClient(w, r.WithContext(ctx), tpl, data, provider)
		if !ok {
			return
		}

		//create the direct payment url
		url, err := CreateURLRelParams(provider.GetURLPaymentDirectClient(), URLParams.SvcID, svc.ID)
		if err != nil {
			logger.Errorw("create payment url", "error", err)
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}
		data[TplParamURL] = ForceURLAbs(ctx, url)

		//load the services excluding the current one
		ctx, svcs, err := ListServicesExcludeID(ctx, s.getDB(), svc.Provider.URLName, svc.ID)
		if err != nil {
			logger.Errorw("load services", "error", err, "name", svc.Provider.URLName, "id", svc.ID)
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return
		}

		//load the services
		data[TplParamSvcs] = s.createServiceUIs(provider, svcs)
		s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
	}
}
