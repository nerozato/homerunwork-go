package main

import (
	"net/http"
)

//handle the stripe login
func (s *Server) handleStripeLogin() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))

		//probe for a provider
		ctx, provider, ok := s.loadProvider(w, r.WithContext(ctx))
		if !ok {
			s.SetCookieErr(w, ErrOAuthStripe)
			http.Redirect(w, r.WithContext(ctx), provider.GetURLPaymentSettings(), http.StatusSeeOther)
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
			logger.Errorw("stripe token", "error", err)
			http.Redirect(w, r.WithContext(ctx), provider.GetURLPaymentSettings(), http.StatusSeeOther)
			return
		}

		//create the url for the stripe oauth callback
		url, err := createStripeURL(URICallback)
		if err != nil {
			logger.Errorw("stripe callback url", "error", err)
			http.Redirect(w, r.WithContext(ctx), provider.GetURLPaymentSettings(), http.StatusSeeOther)
			return
		}

		//create url for the stripe oauth
		url, err = CreateOAuthURLStripe(ctx, token, url, provider.User.Email, provider.User.FirstName, provider.User.LastName, provider.Name, url)
		if err != nil {
			logger.Errorw("stripe oauth url", "error", err)
			http.Redirect(w, r.WithContext(ctx), provider.GetURLPaymentSettings(), http.StatusSeeOther)
			return
		}
		http.Redirect(w, r.WithContext(ctx), url, http.StatusSeeOther)
	}
}

//handle the stripe callback
func (s *Server) handleStripeOAuthCallback() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))

		//probe for a provider
		ctx, provider, ok := s.loadProvider(w, r.WithContext(ctx))
		if !ok {
			s.SetCookieErr(w, ErrOAuthStripe)
			s.redirectAbs(w, r.WithContext(ctx), provider.GetURLPaymentSettings())
			return
		}

		//check for an error
		oauthErr := r.FormValue(URLParams.Error)
		if oauthErr != "" {
			oauthErrDesc := r.FormValue(URLParams.ErrorDesc)
			logger.Errorw("stripe oauth", "error", oauthErr, "description", oauthErrDesc)
			s.SetCookieErr(w, ErrOAuthStripe)
			s.redirectAbs(w, r.WithContext(ctx), provider.GetURLPaymentSettings())
			return
		}

		//verify the token
		state := r.FormValue(URLParams.State)
		_, _, _, host, ok, err := ValidateOAuthToken(state)
		if err != nil {
			logger.Errorw("invalid stripe state", "error", err)
			s.SetCookieErr(w, ErrOAuthStripe)
			s.redirectAbs(w, r.WithContext(ctx), provider.GetURLPaymentSettings())
			return
		}
		if !ok {
			logger.Errorw("invalid stripe state token", "token", state)
			s.SetCookieErr(w, ErrOAuthStripe)
			s.redirectAbs(w, r.WithContext(ctx), provider.GetURLPaymentSettings())
			return
		}

		//store the host
		if host != "" {
			ctx = SetCtxCustomHost(ctx, host)
		}

		//extract data from stripe
		code := r.FormValue(URLParams.Code)
		token, err := RetrieveOAuthTokenStripe(ctx, code)
		if err != nil {
			logger.Errorw("stripe token", "error", err)
			s.SetCookieErr(w, ErrOAuthStripe)
			s.redirectAbs(w, r.WithContext(ctx), provider.GetURLPaymentSettings())
			return
		}
		provider.StripeToken = token

		//save the provider
		ctx, err = SaveProvider(ctx, s.getDB(), provider.Provider)
		if err != nil {
			logger.Errorw("save provider", "error", err, "provider", provider)
			s.SetCookieErr(w, ErrOAuthStripe)
			s.redirectAbs(w, r.WithContext(ctx), provider.GetURLPaymentSettings())
			return
		}

		//success
		s.redirectAbs(w, r.WithContext(ctx), provider.GetURLPaymentSettings(), URLParams.MsgKey, string(MsgStripeSuccess))
	}
}

//handle the stripe webhook callback
func (s *Server) handleStripeWebHookCallback() http.HandlerFunc {
	//callback types
	types := struct {
		TypeCheckout string
		TypeConnect  string
	}{
		TypeCheckout: "checkout",
		TypeConnect:  "connect",
	}
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))

		//determine the secret to use
		var secret string
		state := r.FormValue(URLParams.State)
		switch state {
		case types.TypeCheckout:
			secret = GetStripeWebHookSecretCheckout()
		case types.TypeConnect:
			secret = GetStripeWebHookSecretConnect()
		default:
			logger.Errorw("invalid state", "state", state)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		//verify a valid event
		event, body, err := VerifyWebHookSignatureStripe(w, r.WithContext(ctx), secret)
		if err != nil {
			logger.Errorw("verify signature stripe", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		//check if the live flag matches
		if event.Livemode == GetStripeLive() {
			//store the event
			ctx, err = SaveEventStripe(ctx, s.getDB(), event)
			if err != nil {
				logger.Errorw("save event stripe", "error", err, "event", event)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			//process the event
			now := GetTimeNow("")
			switch event.Type {
			case StripeEventTypeCheckoutSessionCompleted:
				session, err := ParseSessionStripe(event.Data.Raw)
				if err != nil {
					logger.Errorw("parse session stripe", "error", err)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				//store the response
				ctx, err = UpdatePaymentCapturedByExternalID(ctx, s.getDB(), &session.PaymentIntent.ID, &body, &now)
				if err != nil {
					logger.Errorw("update payment", "error", err, "body", body)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
			case StripeEventTypePaymentIntentSucceeded:
				intent, err := ParsePaymentIntentStripe(event.Data.Raw)
				if err != nil {
					logger.Errorw("parse payment intent stripe", "error", err)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				//store the response
				ctx, err = UpdatePaymentCapturedByExternalID(ctx, s.getDB(), &intent.ID, &body, &now)
				if err != nil {
					logger.Errorw("update payment", "error", err, "body", body)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
			}
		}
		w.WriteHeader(http.StatusOK)
	}
}
