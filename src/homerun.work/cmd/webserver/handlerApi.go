package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/gofrs/uuid"
)

//handle the content api
func (s *Server) handleAPIContent() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))

		//check the method
		if r.Method != http.MethodPost {
			http.Error(w, "", http.StatusBadRequest)
			return
		}

		//handle the input
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			logger.Errorw("read body", "error", err)
			http.Error(w, "", http.StatusBadRequest)
			return
		}

		//determine the content type
		var content Content
		err = json.Unmarshal([]byte(body), &content)
		if err != nil {
			logger.Errorw("unjson content", "error", err)
			http.Error(w, "", http.StatusBadRequest)
			return
		}

		//process the content
		switch content.Type {
		case ContentTypeAlert:
			var contentAlert ContentAlert
			err = json.Unmarshal([]byte(body), &contentAlert)
			if err != nil {
				logger.Errorw("unjson content alert", "error", err)
				http.Error(w, "", http.StatusBadRequest)
				return
			}
			ctx, err = SaveContentAlert(ctx, s.getDB(), &contentAlert)
			if err != nil {
				logger.Errorw("save content", "error", err)
				http.Error(w, "", http.StatusBadRequest)
				return
			}
		case ContentTypeTips:
			var contentTips ContentTips
			err = json.Unmarshal([]byte(body), &contentTips)
			if err != nil {
				logger.Errorw("unjson content tips", "error", err)
				http.Error(w, "", http.StatusBadRequest)
				return
			}
			ctx, err = SaveContentTips(ctx, s.getDB(), &contentTips)
			if err != nil {
				logger.Errorw("save content", "tips", err)
				http.Error(w, "", http.StatusBadRequest)
				return
			}
		default:
			logger.Errorw("invalid content", "type", content.Type)
			http.Error(w, "", http.StatusBadRequest)
			return
		}
		w.Header().Set(HeaderCacheControl, "no-store")
		w.WriteHeader(http.StatusOK)
	}
}

//handle the email api
func (s *Server) handleAPIEmail() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))

		//check the method
		if r.Method != http.MethodGet {
			http.Error(w, "", http.StatusBadRequest)
			return
		}

		//handle the input
		bookIDStr := r.FormValue(URLParams.BookID)
		campaignIDStr := r.FormValue(URLParams.CampaignID)
		clientIDStr := r.FormValue(URLParams.ClientID)
		isClient := r.FormValue(URLParams.State) == "true"
		msgTypeStr := r.FormValue(URLParams.Type)
		paymentIDStr := r.FormValue(URLParams.PaymentID)
		providerIDStr := r.FormValue(URLParams.ProviderID)
		userIDStr := r.FormValue(URLParams.UserID)

		//determine the message type
		msgType := MsgType(msgTypeStr)

		//load data based on the id
		var bookUI *bookingUI
		var book *Booking
		if bookIDStr != "" {
			bookID, err := uuid.FromString(bookIDStr)
			if err != nil {
				logger.Errorw("invalid id", "error", err, "id", bookIDStr)
				http.Error(w, "", http.StatusBadRequest)
				return
			}
			ctx, book, err = LoadBookingByID(ctx, s.getDB(), &bookID, false, false)
			if err != nil {
				logger.Errorw("load booking", "error", err, "id", bookID)
				http.Error(w, "", http.StatusInternalServerError)
				return
			}
			bookUI = s.createBookingUI(book)
		}
		var client *Client
		if clientIDStr != "" {
			clientID, err := uuid.FromString(clientIDStr)
			if err != nil {
				logger.Errorw("invalid id", "error", err, "id", clientIDStr)
				http.Error(w, "", http.StatusBadRequest)
				return
			}
			ctx, client, err = LoadClientByID(ctx, s.getDB(), &clientID)
			if err != nil {
				logger.Errorw("load client", "error", err, "id", clientID)
				http.Error(w, "", http.StatusInternalServerError)
				return
			}
			if client == nil {
				logger.Errorw("no client", "error", err, "id", clientID)
				http.Error(w, "", http.StatusInternalServerError)
				return
			}
		}
		var paymentUI *paymentUI
		var payment *Payment
		if paymentIDStr != "" {
			paymentID, err := uuid.FromString(paymentIDStr)
			if err != nil {
				logger.Errorw("invalid id", "error", err, "id", paymentIDStr)
				http.Error(w, "", http.StatusBadRequest)
				return
			}
			ctx, payment, err = LoadPaymentByID(ctx, s.getDB(), &paymentID)
			if err != nil {
				logger.Errorw("load payment", "error", err, "id", paymentID)
				http.Error(w, "", http.StatusInternalServerError)
				return
			}
			paymentUI = s.createPaymentUI(payment)
		}
		var providerUI *providerUI
		var provider *Provider
		if providerIDStr != "" {
			providerID, err := uuid.FromString(providerIDStr)
			if err != nil {
				logger.Errorw("invalid id", "error", err, "id", providerIDStr)
				http.Error(w, "", http.StatusBadRequest)
				return
			}
			ctx, provider, err = LoadProviderByID(ctx, s.getDB(), &providerID)
			if err != nil {
				logger.Errorw("load provider", "error", err, "id", providerID)
				http.Error(w, "", http.StatusInternalServerError)
				return
			}
			providerUI = s.createProviderUI(provider)
		}
		var user *User
		if userIDStr != "" {
			userID, err := uuid.FromString(userIDStr)
			if err != nil {
				logger.Errorw("invalid id", "error", err, "id", userIDStr)
				http.Error(w, "", http.StatusBadRequest)
				return
			}
			ctx, user, err = LoadUserByID(ctx, s.getDB(), &userID)
			if err != nil {
				logger.Errorw("load user", "error", err, "id", userID)
				http.Error(w, "", http.StatusInternalServerError)
				return
			}
		}
		var campaignUI *campaignUI
		var campaign *Campaign
		if campaignIDStr != "" {
			campaignID, err := uuid.FromString(campaignIDStr)
			if err != nil {
				logger.Errorw("invalid id", "error", err, "id", campaignIDStr)
				http.Error(w, "", http.StatusBadRequest)
				return
			}
			ctx, campaign, err = LoadCampaignByID(ctx, s.getDB(), &campaignID)
			if err != nil {
				logger.Errorw("load campaign", "error", err, "id", campaignID)
				http.Error(w, "", http.StatusInternalServerError)
				return
			}
			campaignUI = s.createCampaignUI(campaign)
		}

		//render the email based on the message type
		var err error
		var subject string
		var body string
		var send bool
		switch msgType {
		case MsgTypeBookingCancelClient:
			ctx, subject, body, err = s.createEmailBookingCancelClient(ctx, bookUI)
		case MsgTypeBookingCancelProvider:
			ctx, subject, body, err = s.createEmailBookingCancelProvider(ctx, bookUI)
		case MsgTypeBookingConfirmClient:
			ctx, subject, body, err = s.createEmailBookingConfirmClient(ctx, bookUI)
		case MsgTypeBookingEditClient:
			ctx, subject, body, err = s.createEmailBookingEditClient(ctx, bookUI)
		case MsgTypeBookingNewClient:
			ctx, subject, body, err = s.createEmailBookingNewClient(ctx, bookUI, isClient)
		case MsgTypeBookingNewProvider:
			ctx, subject, body, send, err = s.createEmailBookingNewProvider(ctx, bookUI, isClient)
		case MsgTypeBookingReminderClient:
			ctx, subject, body, err = s.createEmailBookingReminderClient(ctx, bookUI)
		case MsgTypeBookingReminderProvider:
			ctx, subject, body, err = s.createEmailBookingReminderProvider(ctx, bookUI)
		case MsgTypeCampaignAddNotification:
			ctx, subject, body, err = s.createEmailCampaignAddNotification(ctx, providerUI, campaignUI)
		case MsgTypeCampaignAddProvider:
			ctx, subject, body, err = s.createEmailCampaignAddProvider(ctx, providerUI, campaignUI)
		case MsgTypeCampaignPaymentNotification:
			ctx, subject, body, err = s.createEmailCampaignPaymentNotification(ctx, providerUI, campaignUI)
		case MsgTypeCampaignStatusProvider:
			ctx, subject, body, err = s.createEmailCampaignStatusProvider(ctx, providerUI, campaignUI)
		case MsgTypeClientInvite:
			ctx, subject, body, err = s.createEmailClientInvite(ctx, providerUI, client)
		case MsgTypeContact:
			ctx, subject, body, err = s.createEmailContact(ctx, providerUI, client, "text")
		case MsgTypeDomainNotification:
			ctx, subject, body, err = s.createEmailDomainNotification(ctx, providerUI)
		case MsgTypeEmailVerify:
			ctx, subject, body, err = s.createEmailVerify(ctx, user, "url")
		case MsgTypeInvoice:
			ctx, subject, body, err = s.createEmailInvoice(ctx, providerUI, paymentUI)
		case MsgTypeInvoiceInternal:
			ctx, subject, body, err = s.createEmailInvoiceInternal(ctx, providerUI, paymentUI)
		case MsgTypePaymentClient:
			ctx, subject, body, err = s.createEmailPaymentClient(ctx, providerUI, paymentUI)
		case MsgTypePaymentProvider:
			ctx, subject, body, err = s.createEmailPaymentProvider(ctx, providerUI, paymentUI)
		case MsgTypePwdReset:
			ctx, subject, body, err = s.createEmailPwdReset(ctx, "url")
		case MsgTypeProviderUserInvite:
			ctx, subject, body, err = s.createEmailProviderUserInvite(ctx, providerUI, "email")
		case MsgTypeWelcome:
			ctx, subject, body, err = s.createEmailWelcome(ctx, providerUI)
		default:
			logger.Warnw("invalid message type", "type", msgType)
			http.Error(w, "", http.StatusBadRequest)
			return
		}
		if err != nil {
			logger.Errorw("create email", "error", err)
			http.Error(w, "", http.StatusInternalServerError)
			return
		}
		logger.Debugw("email", "subject", subject, "send", send)

		//assume html output
		w.Header().Set(HeaderCacheControl, "no-store")
		w.Header().Set(HeaderContentType, "text/html; charset=utf-8")
		fmt.Fprintf(w, body)
	}
}

//handle the email delete api
func (s *Server) handleAPIEmailDelete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))

		//check the method
		if r.Method != http.MethodDelete {
			http.Error(w, "", http.StatusBadRequest)
			return
		}

		//handle the input
		email := r.FormValue(URLParams.Email)

		//delete the user
		ctx, count, err := DeleteUserByEmail(ctx, s.getDB(), email)
		if err != nil {
			logger.Errorw("delete user", "error", err)
			http.Error(w, "", http.StatusBadRequest)
			return
		}
		if count == 0 {
			logger.Errorw("delete user count", "error", err)
			http.Error(w, "", http.StatusBadRequest)
			return
		}
		w.Header().Set(HeaderCacheControl, "no-store")
		w.WriteHeader(http.StatusOK)
	}
}

//handle the maintenance api
func (s *Server) handleAPIMaintenance() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, logger := GetLogger(s.getCtx(r))

		//check the method
		if r.Method != http.MethodPost {
			http.Error(w, "", http.StatusBadRequest)
			return
		}

		//handle the input
		state := r.FormValue(URLParams.State)
		enable, err := strconv.ParseBool(state)
		if err != nil {
			logger.Errorw("parse bool", "error", err)
			http.Error(w, "", http.StatusBadRequest)
			return
		}

		//update the maintenance flag
		if s.maintenanceEnable != enable {
			//start the appropriate mode of the server
			logger.Infow("maintenance mode change", "enabled", enable)
			if enable {
				go func() {
					s.Stop(s.ctx)
					s.StartMaintenance()
				}()
			} else {
				go func() {
					s.StopMaintenance(s.ctx)
					s.Start()
				}()
			}

			//update the state
			s.maintenanceEnable = enable
			AddCtxStatsData(s.ctx, ServerStatMaintenanceEnabled, s.maintenanceEnable)
		}
		w.Header().Set(HeaderCacheControl, "no-store")
		w.WriteHeader(http.StatusOK)
	}
}

//handle the server orders api supporting the calendar
func (s *Server) handleAPIOrdersCalendar() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))

		//check the method
		if r.Method != http.MethodGet {
			http.Error(w, "", http.StatusBadRequest)
			return
		}

		//load the provider
		ctx, provider, ok := s.loadProvider(w, r.WithContext(ctx))
		if !ok {
			logger.Errorw("load provider")
			http.Error(w, "", http.StatusInternalServerError)
			return
		}

		//read the form and parse the dates
		startStr := r.FormValue(URLParams.Start)
		start := ParseDateTimeRFC3339(startStr)
		endStr := r.FormValue(URLParams.End)
		end := ParseDateTimeRFC3339(endStr)

		//load orders for the month
		user := provider.GetProviderUser()
		ctx, orders, err := ListBookingsByProviderIDAndTime(ctx, s.getDB(), provider.ID, user, start, end)
		if err != nil {
			logger.Errorw("load orders", "error", err, "id", provider.ID, "start", start, "end", end)
			http.Error(w, "", http.StatusInternalServerError)
			return
		}

		//create json for the calendar widget
		type event struct {
			Start       string `json:"start"`
			End         string `json:"end"`
			Title       string `json:"title"`
			URL         string `json:"url"`
			BorderColor string `json:"borderColor"`
		}
		lenOrders := len(orders)
		events := make([]*event, 0, lenOrders)
		if lenOrders > 0 {
			for _, order := range orders {
				orderUI := s.createBookingUI(order)
				event := &event{
					Start: orderUI.TimeFrom.Format(time.RFC3339),
					End:   orderUI.TimeTo.Format(time.RFC3339),
					Title: orderUI.GetEventTitle(),
					URL:   orderUI.GetURLView(),
				}

				//mark unconfirmed orders
				if !orderUI.Confirmed {
					event.BorderColor = "red"
				}
				events = append(events, event)
			}
		}
		logger.Debugw("report", "events", events)
		w.Header().Set(HeaderCacheControl, "no-store")
		w.Header().Set(HeaderContentType, "application/json")
		err = json.NewEncoder(w).Encode(events)
		if err != nil {
			logger.Errorw("write json", "error", err)
		}
	}
}

//handle the server report api
func (s *Server) handleAPIReport() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))

		//check the method
		if r.Method != http.MethodGet {
			http.Error(w, "", http.StatusBadRequest)
			return
		}

		//load report data
		ctx, countUsers, err := CountUsers(ctx, s.getDB())
		if err != nil {
			logger.Errorw("count users", "error", err)
			http.Error(w, "", http.StatusInternalServerError)
			return
		}
		ctx, user, userCreateTime, err := FindLatestUser(ctx, s.getDB())
		if err != nil {
			logger.Errorw("create time user", "error", err)
			http.Error(w, "", http.StatusInternalServerError)
			return
		}
		ctx, countProviders, err := CountProviders(ctx, s.getDB())
		if err != nil {
			logger.Errorw("count providers", "error", err)
			http.Error(w, "", http.StatusInternalServerError)
			return
		}
		ctx, provider, providerCreateTime, err := FindLatestProvider(ctx, s.getDB())
		if err != nil {
			logger.Errorw("latest provider", "error", err)
			http.Error(w, "", http.StatusInternalServerError)
			return
		}
		ctx, countBookings, err := CountBookings(ctx, s.getDB())
		if err != nil {
			logger.Errorw("count bookings", "error", err)
			http.Error(w, "", http.StatusInternalServerError)
			return
		}
		ctx, orderProvider, orderCreateTime, err := FindLatestBooking(ctx, s.getDB())
		if err != nil {
			logger.Errorw("latest booking", "error", err)
			http.Error(w, "", http.StatusInternalServerError)
			return
		}
		ctx, countPayments, err := CountPayments(ctx, s.getDB())
		if err != nil {
			logger.Errorw("count payments", "error", err)
			http.Error(w, "", http.StatusInternalServerError)
			return
		}
		ctx, paymentProvider, paymentCreateTime, err := FindLatestPayment(ctx, s.getDB())
		if err != nil {
			logger.Errorw("latest payment", "error", err)
			http.Error(w, "", http.StatusInternalServerError)
			return
		}

		//send the data
		type entryData struct {
			Login string `json:"Login,omitempty"`
			Email string `json:"Email,omitempty"`
			Name  string `json:"Name"`
			Time  string `json:"Time"`
		}
		type reportData struct {
			UserCount     int        `json:"UserCount"`
			LastUser      *entryData `json:"LastUser"`
			ProviderCount int        `json:"ProviderCount"`
			LastProvider  *entryData `json:"LastProvider"`
			OrderCount    int        `json:"OrderCount"`
			LastOrder     *entryData `json:"LastOrder"`
			PaymentCount  int        `json:"PaymentCount"`
			LastPayment   *entryData `json:"LastPaymen"`
		}
		data := reportData{
			UserCount:     countUsers,
			ProviderCount: countProviders,
			OrderCount:    countBookings,
			PaymentCount:  countPayments,
		}
		if user != nil && userCreateTime != nil {
			data.LastUser = &entryData{
				Login: user.Login,
				Email: user.Email,
				Name:  user.FormatName(),
				Time:  userCreateTime.Format(time.RFC3339),
			}
		}
		if provider != nil && providerCreateTime != nil {
			data.LastProvider = &entryData{
				Name: provider.Name,
				Time: providerCreateTime.Format(time.RFC3339),
			}
		}
		if orderProvider != nil && orderCreateTime != nil {
			data.LastOrder = &entryData{
				Name: orderProvider.Name,
				Time: orderCreateTime.Format(time.RFC3339),
			}
		}
		if paymentCreateTime != nil && paymentProvider != nil {
			data.LastPayment = &entryData{
				Name: paymentProvider.Name,
				Time: paymentCreateTime.Format(time.RFC3339),
			}
		}
		logger.Debugw("report", "data", data)
		w.Header().Set(HeaderCacheControl, "no-store")
		w.Header().Set(HeaderContentType, "application/json")
		err = json.NewEncoder(w).Encode(data)
		if err != nil {
			logger.Errorw("write json", "error", err)
		}
	}
}

//handle the server statistics api
func (s *Server) handleAPIStats() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, logger := GetLogger(s.getCtx(r))

		//check the method
		if r.Method != http.MethodGet {
			http.Error(w, "", http.StatusBadRequest)
			return
		}

		//read the input and update the timing stats based on the timezone
		timeZone := r.FormValue(URLParams.TimeZone)
		s.stats.DisplayTimes(timeZone)

		//send the data
		logger.Debugw("statistics", "data", s.stats)
		data, err := s.stats.CreateJSON()
		if err != nil {
			logger.Errorw("create json", "error", err)
			http.Error(w, "", http.StatusInternalServerError)
			return
		}
		w.Header().Set(HeaderCacheControl, "no-store")
		w.Header().Set(HeaderContentType, "application/json")
		_, err = w.Write(data)
		if err != nil {
			logger.Errorw("write json", "error", err)
		}
	}
}

//handle the shortened url
func (s *Server) handleAPIURLShort() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))

		//check for a shortened url
		urlShort := GetCtxURLShort(ctx)
		if urlShort == "" {
			logger.Warnw("no url short")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		//load the full url
		ctx, url, err := LoadURL(ctx, s.getDB(), urlShort)
		if err != nil {
			logger.Warnw("load url", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if url == "" {
			logger.Warnw("no url short")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set(HeaderCacheControl, "no-store")
		http.Redirect(w, r.WithContext(ctx), url, http.StatusMovedPermanently)
	}
}
