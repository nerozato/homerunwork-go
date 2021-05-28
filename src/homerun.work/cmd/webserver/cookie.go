package main

import (
	"net/http"
	"time"

	"github.com/pkg/errors"
)

//cookie configuration
const (
	CookieExpirationAlert = 1
	CookieExpirationSec   = 300
	CookieErr             = "err"
	CookieFlag            = "flag"
	CookieHost            = "host"
	CookieMsg             = "msg"
	CookieRequestID       = "requestId"
	CookieRequestURI      = "requestUri"
	CookieSignUp          = "signUp"
	CookieSignUpType      = "signUpType"
	CookieTimeZone        = "timeZone"
	CookieTitleAlert      = "msgTitleAlert"
	CookieToken           = "token"
)

//create a basic cookie
func createBaseCookie() *http.Cookie {
	return &http.Cookie{
		HttpOnly: true,
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
	}
}

//SetCookieErr : store the error in a cookie
func (s *Server) SetCookieErr(w http.ResponseWriter, key ErrKey, args ...interface{}) {
	cookie := createBaseCookie()
	cookie.Name = CookieErr
	cookie.Value = GetErrText(key, args...)
	cookie.MaxAge = CookieExpirationAlert
	http.SetCookie(w, cookie)
}

//GetCookieErr : retrieve the error from the cookie
func (s *Server) GetCookieErr(r *http.Request) (string, error) {
	cookie, err := r.Cookie(CookieErr)
	if err != nil {
		if err == http.ErrNoCookie {
			return "", nil
		}
		return "", errors.Wrap(err, "get cookie")
	}
	return cookie.Value, nil
}

//DeleteCookieErr : delete the error cookie
func (s *Server) DeleteCookieErr(w http.ResponseWriter) {
	cookie := &http.Cookie{
		Name:   CookieErr,
		MaxAge: -1,
	}
	http.SetCookie(w, cookie)
}

//SetCookieFlag : store the flag in a cookie
func (s *Server) SetCookieFlag(w http.ResponseWriter, v string) {
	cookie := createBaseCookie()
	cookie.Name = CookieFlag
	cookie.Value = v
	cookie.MaxAge = CookieExpirationAlert
	http.SetCookie(w, cookie)
}

//GetCookieFlag : retrieve the flag from the cookie
func (s *Server) GetCookieFlag(r *http.Request) (string, error) {
	cookie, err := r.Cookie(CookieFlag)
	if err != nil {
		if err == http.ErrNoCookie {
			return "", nil
		}
		return "", errors.Wrap(err, "get cookie")
	}
	return cookie.Value, nil
}

//DeleteCookieFlag : delete the flag cookie
func (s *Server) DeleteCookieFlag(w http.ResponseWriter) {
	cookie := &http.Cookie{
		Name:   CookieFlag,
		MaxAge: -1,
	}
	http.SetCookie(w, cookie)
}

//SetCookieHost : store the host in a cookie
func (s *Server) SetCookieHost(w http.ResponseWriter, val string) {
	cookie := createBaseCookie()
	cookie.Name = CookieHost
	cookie.Value = val
	cookie.Expires = MaxTime
	http.SetCookie(w, cookie)
}

//GetCookieHost : retrieve the host from the cookie
func (s *Server) GetCookieHost(r *http.Request) (string, error) {
	cookie, err := r.Cookie(CookieHost)
	if err != nil {
		if err == http.ErrNoCookie {
			return "", nil
		}
		return "", errors.Wrap(err, "get cookie")
	}
	return cookie.Value, nil
}

//DeleteCookieHost : delete the host cookie
func (s *Server) DeleteCookieHost(w http.ResponseWriter) {
	cookie := &http.Cookie{
		Name:   CookieHost,
		MaxAge: -1,
	}
	http.SetCookie(w, cookie)
}

//SetCookieMsg : store the message in a cookie
func (s *Server) SetCookieMsg(w http.ResponseWriter, key MsgKey, args ...interface{}) {
	cookie := createBaseCookie()
	cookie.Name = CookieMsg
	cookie.Value = GetMsgText(key, args...)
	cookie.MaxAge = CookieExpirationAlert
	http.SetCookie(w, cookie)

	//find the title to use
	s.SetCookieTitleAlert(w, key)
}

//GetCookieMsg : retrieve the message from the cookie
func (s *Server) GetCookieMsg(r *http.Request) (string, error) {
	cookie, err := r.Cookie(CookieMsg)
	if err != nil {
		if err == http.ErrNoCookie {
			return "", nil
		}
		return "", errors.Wrap(err, "get cookie")
	}
	return cookie.Value, nil
}

//DeleteCookieMsg : delete the message cookie
func (s *Server) DeleteCookieMsg(w http.ResponseWriter) {
	cookie := &http.Cookie{
		Name:   CookieMsg,
		MaxAge: -1,
	}
	http.SetCookie(w, cookie)
}

//SetCookieRequestID : store the request ID in a cookie
func (s *Server) SetCookieRequestID(w http.ResponseWriter, requestID string) {
	cookie := createBaseCookie()
	cookie.Name = CookieRequestID
	cookie.Value = requestID
	cookie.MaxAge = CookieExpirationSec
	http.SetCookie(w, cookie)
}

//GetCookieRequestID : retrieve the request ID from the cookie
func (s *Server) GetCookieRequestID(r *http.Request) (string, error) {
	cookie, err := r.Cookie(CookieRequestID)
	if err != nil {
		if err == http.ErrNoCookie {
			return "", nil
		}
		return "", errors.Wrap(err, "get cookie")
	}
	return cookie.Value, nil
}

//DeleteCookieRequestID : delete the request ID cookie
func (s *Server) DeleteCookieRequestID(w http.ResponseWriter) {
	cookie := &http.Cookie{
		Name:   CookieRequestID,
		MaxAge: -1,
	}
	http.SetCookie(w, cookie)
}

//SetCookieRequestURI : store the request URI in a cookie
func (s *Server) SetCookieRequestURI(w http.ResponseWriter, requestURI string) {
	cookie := createBaseCookie()
	cookie.Name = CookieRequestURI
	cookie.Value = requestURI
	cookie.MaxAge = CookieExpirationSec
	http.SetCookie(w, cookie)
}

//GetCookieRequestURI : retrieve the request URI from the cookie
func (s *Server) GetCookieRequestURI(r *http.Request) (string, error) {
	cookie, err := r.Cookie(CookieRequestURI)
	if err != nil {
		if err == http.ErrNoCookie {
			return "", nil
		}
		return "", errors.Wrap(err, "get cookie")
	}
	return cookie.Value, nil
}

//DeleteCookieRequestURI : delete the request URI cookie
func (s *Server) DeleteCookieRequestURI(w http.ResponseWriter) {
	cookie := &http.Cookie{
		Name:   CookieRequestURI,
		MaxAge: -1,
	}
	http.SetCookie(w, cookie)
}

//SetCookieSignUp : store the sign-up date in a cookie
func (s *Server) SetCookieSignUp(w http.ResponseWriter) {
	cookie := createBaseCookie()
	cookie.Name = CookieSignUp
	cookie.Value = time.Now().UTC().Format(time.RFC3339)
	cookie.Expires = MaxTime
	http.SetCookie(w, cookie)
}

//SetCookieSignUpType : store the signup type in a cookie
func (s *Server) SetCookieSignUpType(w http.ResponseWriter, val string) {
	cookie := createBaseCookie()
	cookie.Name = CookieSignUpType
	cookie.Value = val
	cookie.Expires = MaxTime
	http.SetCookie(w, cookie)
}

//GetCookieSignUpType : retrieve the error from the cookie
func (s *Server) GetCookieSignUpType(r *http.Request) (string, error) {
	cookie, err := r.Cookie(CookieSignUpType)
	if err != nil {
		if err == http.ErrNoCookie {
			return "", nil
		}
		return "", errors.Wrap(err, "get cookie")
	}
	return cookie.Value, nil
}

//GetCookieTimeZone : retrieve the timezone from the cookie
func (s *Server) GetCookieTimeZone(r *http.Request) (string, error) {
	cookie, err := r.Cookie(CookieTimeZone)
	if err != nil {
		if err == http.ErrNoCookie {
			return "", nil
		}
		return "", errors.Wrap(err, "get cookie")
	}
	return cookie.Value, nil
}

//SetCookieTitleAlert : store the alert title in a cookie
func (s *Server) SetCookieTitleAlert(w http.ResponseWriter, key MsgKey) {
	cookie := createBaseCookie()
	cookie.Name = CookieTitleAlert
	cookie.Value = GetMsgTitle(key)
	cookie.MaxAge = CookieExpirationAlert
	http.SetCookie(w, cookie)
}

//GetCookieTitleAlert : retrieve the alert title from the cookie
func (s *Server) GetCookieTitleAlert(r *http.Request) (string, error) {
	cookie, err := r.Cookie(CookieTitleAlert)
	if err != nil {
		if err == http.ErrNoCookie {
			return "", nil
		}
		return "", errors.Wrap(err, "get cookie")
	}
	return cookie.Value, nil
}

//DeleteCookieTitleAlert : delete the alert title cookie
func (s *Server) DeleteCookieTitleAlert(w http.ResponseWriter) {
	cookie := &http.Cookie{
		Name:   CookieTitleAlert,
		MaxAge: -1,
	}
	http.SetCookie(w, cookie)
}

//SetCookieToken : store the token in a cookie
func (s *Server) SetCookieToken(w http.ResponseWriter, token string) {
	cookie := createBaseCookie()
	cookie.Name = CookieToken
	cookie.Value = token
	cookie.MaxAge = JWTExpirationSec
	http.SetCookie(w, cookie)
}

//GetCookieToken : retrieve the token from the cookie
func (s *Server) GetCookieToken(r *http.Request) (string, error) {
	cookie, err := r.Cookie(CookieToken)
	if err != nil {
		if err == http.ErrNoCookie {
			return "", nil
		}
		return "", errors.Wrap(err, "get cookie")
	}
	return cookie.Value, nil
}

//DeleteCookieToken : delete the token cookie
func (s *Server) DeleteCookieToken(w http.ResponseWriter) {
	cookie := &http.Cookie{
		Name:   CookieToken,
		MaxAge: -1,
	}
	http.SetCookie(w, cookie)
}
