package main

import (
	"net/http"
)

//handle the facebook login
func (s *Server) handleFacebookLogin() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))

		//read state
		isSignUp := GetCtxIsSignUp(ctx)
		timeZone := GetCtxTimeZone(ctx)
		signUpType := GetCtxType(ctx)

		//extract data from facebook
		token := r.FormValue(URLParams.Token)
		ctx, fbUser, err := GetUserFacebook(ctx, token)
		if err != nil {
			logger.Errorw("facebook user data", "error", err)
			s.SetCookieErr(w, ErrOAuthFacebook)
			s.redirectAbs(w, r.WithContext(ctx), URILogin)
			return
		}

		//check for a user record, assuming the login is using the facebook id
		ctx, user, err := LoadUserByLogin(ctx, s.db, fbUser.ID)
		if err != nil {
			logger.Errorw("facebook user load", "error", err)
			s.SetCookieErr(w, ErrOAuthFacebook)
			s.redirectAbs(w, r.WithContext(ctx), URILogin)
			return
		}
		if user != nil {
			//refresh the token and store in the cookie
			token, err = s.refreshToken(w, r.WithContext(ctx), user.ID)
			if err != nil {
				logger.Errorw("facebook refresh token", "error", err)
				s.SetCookieErr(w, ErrOAuthFacebook)
				s.redirectAbs(w, r.WithContext(ctx), URILogin)
				return
			}
		} else {
			//check if coming from the sign-up page
			if !isSignUp {
				//go back to the sign-up page to force explicitly agreeing to the sign-up
				s.SetCookieErr(w, ErrOAuthFacebookSignUp)
				s.redirectAbs(w, r.WithContext(ctx), URISignUp)
				return
			}

			//use the facebook id for the login
			user = &User{
				Login:      fbUser.ID,
				IsOAuth:    true,
				TimeZone:   timeZone,
				SignUpType: signUpType,
			}
		}

		//save the user information
		user.FirstName = fbUser.FirstName
		user.LastName = fbUser.LastName
		user.Email = fbUser.Email
		user.EmailVerified = true
		user.OAuthFacebookData = fbUser
		ctx, err = s.saveUser(w, r.WithContext(ctx), user, "")
		if err != nil {
			logger.Errorw("save user", "error", err)
			s.SetCookieErr(w, ErrOAuthFacebook)
			s.redirectAbs(w, r.WithContext(ctx), URILogin)
			return
		}
		s.SetCookieSignUpType(w, signUpType)

		//redirect after login
		ctx, err = s.redirectLogin(w, r.WithContext(ctx), user.ID, token)
		if err != nil {
			logger.Warnw("redirect login", "error", err)
			s.SetCookieErr(w, ErrOAuthFacebook)
			s.redirectAbs(w, r.WithContext(ctx), URILogin)
			return
		}
	}
}
