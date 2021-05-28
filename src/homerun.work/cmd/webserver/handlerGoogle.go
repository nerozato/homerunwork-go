package main

import (
	"net/http"

	"github.com/pkg/errors"
)

//handle the google login
func (s *Server) handleGoogleLogin() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))

		//read the host
		host, err := s.GetCookieHost(r)
		if err != nil {
			logger.Warnw("get cookie host", "error", err)
			http.Redirect(w, r.WithContext(ctx), URIErr, http.StatusSeeOther)
			return
		}

		//generate a token used to validate the callback
		isSignUp := GetCtxIsSignUp(ctx)
		timeZone := GetCtxTimeZone(ctx)
		signUpType := GetCtxType(ctx)
		token, err := GenerateOAuthToken(isSignUp, timeZone, signUpType, host)
		if err != nil {
			panic(errors.Wrap(err, "google oauth token"))
		}
		url := GetURLOAuth(ctx, token, false)
		http.Redirect(w, r.WithContext(ctx), url, http.StatusSeeOther)
	}
}

//handle the google oauth callback
func (s *Server) handleGoogleOAuthCallback() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))

		//check for an error
		oauthErr := r.FormValue(URLParams.Error)
		if oauthErr != "" {
			logger.Errorw("google oauth", "error", oauthErr)
			s.SetCookieErr(w, ErrOAuthGoogle)
			s.redirectAbs(w, r.WithContext(ctx), URILogin)
			return
		}

		//verify the token
		state := r.FormValue(URLParams.State)
		isSignUp, timeZone, signUpType, host, ok, err := ValidateOAuthToken(state)
		if err != nil {
			logger.Errorw("invalid google state", "error", err)
			s.SetCookieErr(w, ErrOAuthGoogle)
			s.redirectAbs(w, r.WithContext(ctx), URILogin)
			return
		}
		if !ok {
			logger.Errorw("invalid google state token", "token", state)
			s.SetCookieErr(w, ErrOAuthGoogle)
			s.redirectAbs(w, r.WithContext(ctx), URILogin)
			return
		}

		//store the host
		if host != "" {
			ctx = SetCtxCustomHost(ctx, host)
		}

		//extract data from google
		code := r.FormValue(URLParams.Code)
		googleUser, err := GetGoogleUserData(ctx, code, false)
		if err != nil {
			logger.Errorw("google user data", "error", err)
			s.SetCookieErr(w, ErrOAuthGoogle)
			s.redirectAbs(w, r.WithContext(ctx), URILogin)
			return
		}

		//check for a user record, assuming the login is using the google id
		ctx, user, err := LoadUserByLogin(ctx, s.db, googleUser.ID)
		if err != nil {
			logger.Errorw("google user load", "error", err)
			s.SetCookieErr(w, ErrOAuthGoogle)
			s.redirectAbs(w, r.WithContext(ctx), URILogin)
			return
		}
		var token string
		if user != nil {
			//refresh the token and store in the cookie
			token, err = s.refreshToken(w, r.WithContext(ctx), user.ID)
			if err != nil {
				logger.Errorw("google refresh token", "error", err)
				s.SetCookieErr(w, ErrOAuthGoogle)
				s.redirectAbs(w, r.WithContext(ctx), URILogin)
				return
			}
		} else {
			//check if coming from the sign-up page
			if !isSignUp {
				//go back to the sign-up page to force explicitly agreeing to the sign-up
				s.SetCookieErr(w, ErrOAuthGoogleSignUp)
				s.redirectAbs(w, r.WithContext(ctx), URISignUp)
				return
			}

			//use the google id for the login
			user = &User{
				Login:      googleUser.ID,
				IsOAuth:    true,
				TimeZone:   timeZone,
				SignUpType: signUpType,
			}
		}

		//save the user information
		user.FirstName = googleUser.FirstName
		user.LastName = googleUser.LastName
		user.Email = googleUser.Email
		user.EmailVerified = googleUser.EmailVerified
		user.OAuthGoogleData = googleUser
		ctx, err = s.saveUser(w, r.WithContext(ctx), user, "")
		if err != nil {
			logger.Errorw("save user", "error", err)
			s.SetCookieErr(w, ErrOAuthGoogle)
			s.redirectAbs(w, r.WithContext(ctx), URILogin)
			return
		}
		s.SetCookieSignUpType(w, signUpType)

		//redirect after login
		ctx, err = s.redirectLogin(w, r.WithContext(ctx), user.ID, token)
		if err != nil {
			logger.Warnw("redirect login", "error", err)
			s.SetCookieErr(w, ErrOAuthGoogle)
			s.redirectAbs(w, r.WithContext(ctx), URILogin)
			return
		}
	}
}

//handle the google calendar-scope login
func (s *Server) handleGoogleCalLogin() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))

		//read the host
		host, err := s.GetCookieHost(r)
		if err != nil {
			logger.Warnw("get cookie host", "error", err)
			http.Redirect(w, r.WithContext(ctx), URIErr, http.StatusSeeOther)
			return
		}

		//generate a token used to validate the callback
		token, err := GenerateOAuthToken(false, "", "", host)
		if err != nil {
			panic(errors.Wrap(err, "google oauth token"))
		}
		url := GetURLOAuth(ctx, token, true)
		http.Redirect(w, r.WithContext(ctx), url, http.StatusSeeOther)
	}
}

//handle the google calendar oauth callback
func (s *Server) handleGoogleOAuthCalCallback() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))

		//check for an error
		oauthErr := r.FormValue(URLParams.Error)
		if oauthErr != "" {
			logger.Errorw("google oauth", "error", oauthErr)
			s.SetCookieErr(w, ErrOAuthGoogle)
			s.redirectAbs(w, r.WithContext(ctx), URILogin)
			return
		}

		//verify the token
		state := r.FormValue(URLParams.State)
		_, _, _, host, ok, err := ValidateOAuthToken(state)
		if err != nil {
			logger.Errorw("invalid google state", "error", err)
			s.SetCookieErr(w, ErrOAuthGoogle)
			s.redirectAbs(w, r.WithContext(ctx), URILogin)
			return
		}
		if !ok {
			logger.Errorw("invalid google state token", "token", state)
			s.SetCookieErr(w, ErrOAuthGoogle)
			s.redirectAbs(w, r.WithContext(ctx), URILogin)
			return
		}

		//store the host
		if host != "" {
			ctx = SetCtxCustomHost(ctx, host)
		}

		//extract data from google
		url := createDashboardURL(URICalendars)
		code := r.FormValue(URLParams.Code)
		googleToken, err := GetGoogleOAuthToken(ctx, code, true)
		if err != nil {
			logger.Errorw("google token", "error", err)
			s.SetCookieErr(w, ErrOAuthGoogle)
			s.redirectAbs(w, r.WithContext(ctx), url)
			return
		}

		//check for a user record
		userID := GetCtxUserID(ctx)
		if userID == nil {
			logger.Errorw("no user id", "error", err)
			s.SetCookieErr(w, ErrOAuthGoogle)
			s.redirectAbs(w, r.WithContext(ctx), url)
			return
		}
		ctx, user, err := LoadUserByID(ctx, s.db, userID)
		if err != nil {
			logger.Errorw("google user load", "error", err, "id", userID)
			s.SetCookieErr(w, ErrOAuthGoogle)
			s.redirectAbs(w, r.WithContext(ctx), url)
			return
		}

		//save the user information
		user.GoogleCalendarToken = googleToken
		ctx, err = s.saveUser(w, r.WithContext(ctx), user, "")
		if err != nil {
			logger.Errorw("save user", "error", err)
			s.SetCookieErr(w, ErrOAuthGoogle)
			return
		}
		s.redirectAbs(w, r.WithContext(ctx), url)
	}
}
