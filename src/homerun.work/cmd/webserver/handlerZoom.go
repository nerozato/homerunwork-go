package main

import (
	"net/http"
)

//handle the zoom login
func (s *Server) handleZoomLogin() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))

		//probe for a provider
		ctx, provider, ok := s.loadProvider(w, r.WithContext(ctx))
		if !ok {
			s.SetCookieErr(w, ErrOAuthZoom)
			http.Redirect(w, r.WithContext(ctx), provider.GetURLAddOns(), http.StatusSeeOther)
			return
		}

		//read the host
		host, err := s.GetCookieHost(r)
		if err != nil {
			logger.Warnw("get cookie host", "error", err)
			http.Redirect(w, r.WithContext(ctx), URIErr, http.StatusSeeOther)
			return
		}

		//generate a token used to validate the callback
		timeZone := GetCtxTimeZone(ctx)
		isSignUp := GetCtxIsSignUp(ctx)
		token, err := GenerateOAuthToken(isSignUp, timeZone, "", host)
		if err != nil {
			logger.Errorw("zoom token", "error", err)
			http.Redirect(w, r.WithContext(ctx), provider.GetURLAddOns(), http.StatusSeeOther)
			return
		}

		//create the url for the zoom oauth callback
		url, err := createZoomURL(URICallback)
		if err != nil {
			logger.Errorw("zoom callback url", "error", err)
			http.Redirect(w, r.WithContext(ctx), provider.GetURLAddOns(), http.StatusSeeOther)
			return
		}

		//create url for the zoom oauth
		url, err = CreateOAuthURLZoom(ctx, token, url)
		if err != nil {
			logger.Errorw("zoom oauth url", "error", err)
			http.Redirect(w, r.WithContext(ctx), provider.GetURLAddOns(), http.StatusSeeOther)
			return
		}
		http.Redirect(w, r.WithContext(ctx), url, http.StatusSeeOther)
	}
}

//handle the zoom callback
func (s *Server) handleZoomOAuthCallback() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))

		//probe for a provider
		ctx, provider, ok := s.loadProvider(w, r.WithContext(ctx))
		if !ok {
			s.SetCookieErr(w, ErrOAuthZoom)
			s.redirectAbs(w, r.WithContext(ctx), provider.GetURLAddOns())
			return
		}

		//check for an error
		oauthErr := r.FormValue(URLParams.Error)
		if oauthErr != "" {
			oauthErrDesc := r.FormValue(URLParams.ErrorDesc)
			logger.Errorw("zoom oauth", "error", oauthErr, "description", oauthErrDesc)
			s.SetCookieErr(w, ErrOAuthZoom)
			s.redirectAbs(w, r.WithContext(ctx), provider.GetURLAddOns())
			return
		}

		//verify the token
		state := r.FormValue(URLParams.State)
		_, _, _, host, ok, err := ValidateOAuthToken(state)
		if err != nil {
			logger.Errorw("invalid zoom state", "error", err)
			s.SetCookieErr(w, ErrOAuthZoom)
			s.redirectAbs(w, r.WithContext(ctx), provider.GetURLAddOns())
			return
		}
		if !ok {
			logger.Errorw("invalid zoom state token", "token", state)
			s.SetCookieErr(w, ErrOAuthZoom)
			s.redirectAbs(w, r.WithContext(ctx), provider.GetURLAddOns())
			return
		}

		//store the host
		if host != "" {
			ctx = SetCtxCustomHost(ctx, host)
		}

		//create the url for the zoom oauth callback
		url, err := createZoomURL(URICallback)
		if err != nil {
			logger.Errorw("zoom callback url", "error", err)
			s.redirectAbs(w, r.WithContext(ctx), provider.GetURLAddOns())
			return
		}

		//retrieve the oauth token from zoom
		code := r.FormValue(URLParams.Code)
		token, err := RetrieveOAuthTokenZoom(ctx, code, url)
		if err != nil {
			logger.Errorw("zoom token", "error", err)
			s.SetCookieErr(w, ErrOAuthZoom)
			s.redirectAbs(w, r.WithContext(ctx), provider.GetURLAddOns())
			return
		}
		user := provider.GetUser()
		user.ZoomToken = token

		//get the zoom user
		token, zoomUser, err := GetUserZoom(ctx, token)
		if err != nil {
			logger.Errorw("zoom user", "error", err)
			s.SetCookieErr(w, ErrOAuthZoom)
			s.redirectAbs(w, r.WithContext(ctx), provider.GetURLAddOns())
			return
		}
		user.ZoomUser = zoomUser
		if token != nil {
			user.ZoomToken = token
		}

		//save the user
		ctx, err = SaveUser(ctx, s.getDB(), user, "")
		if err != nil {
			logger.Errorw("save user", "error", err, "id", user.ID)
			s.SetCookieErr(w, ErrOAuthZoom)
			s.redirectAbs(w, r.WithContext(ctx), provider.GetURLAddOns())
			return
		}

		//success
		s.redirectAbs(w, r.WithContext(ctx), provider.GetURLAddOns(), URLParams.MsgKey, string(MsgZoomSuccess))
	}
}

//handle the zoom webhook callback
func (s *Server) handleZoomWebHookCallback() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))

		//verify a valid event
		event, err := VerifyWebHookZoom(w, r.WithContext(ctx))
		if err != nil {
			logger.Errorw("verify zoom", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		//store the event
		ctx, err = SaveEventZoom(ctx, s.getDB(), event)
		if err != nil {
			logger.Errorw("save event zoom", "error", err, "event", event)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		//process the event and remove the zoom information from the user
		ctx, err = DeleteUserZoom(ctx, s.getDB(), event.Payload.UserID)
		if err != nil {
			logger.Errorw("delete provider zoom", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		//signal compliance
		err = SignalDataComplianceZoom(ctx, event.Payload)
		if err != nil {
			logger.Errorw("signal compliance zoom", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}
