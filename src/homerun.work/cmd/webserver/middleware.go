package main

import (
	"context"
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

//wrapper for the response writer that captures data for logging
type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

//capture the status code
func (w *loggingResponseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

//create an instance of the logging response writer
func (s *Server) createLoggingResponseWriter(w http.ResponseWriter) *loggingResponseWriter {
	return &loggingResponseWriter{w, http.StatusOK}
}

//read the auth token
func (s *Server) readToken(r *http.Request) (bool, *uuid.UUID, error) {
	//extract the JWT from the cookie
	token, err := s.GetCookieToken(r)
	if err != nil {
		return false, nil, errors.Wrap(err, "get cookie token")
	}
	if token == "" {
		return false, nil, nil
	}

	//verify the token
	ok, userID, err := ValidateAuthToken(token)
	if err != nil {
		return false, nil, errors.Wrap(err, "validate token")
	}
	if !ok {
		return false, nil, nil
	}
	if userID == nil {
		return false, nil, errors.Wrap(err, "validate user id")
	}
	return true, userID, nil
}

//refresh the auth token
func (s *Server) refreshToken(w http.ResponseWriter, r *http.Request, userID *uuid.UUID) (string, error) {
	//create a JWT
	token, err := GenerateAuthToken(userID)
	if err != nil {
		return "", errors.Wrap(err, "generate token")
	}

	//store in a cookie
	s.SetCookieToken(w, token)
	return token, nil
}

//check the api token
func (s *Server) apiTokenHdlr(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, _ := GetLogger(s.getCtx(r))

		//verify the api token
		token := r.Header.Get(HeaderAPIToken)
		if token != GetDevAPIToken() {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

//check provider authentication by verifying the JWT
func (s *Server) authProviderHdlr(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, _ := GetLogger(s.getCtx(r))

		//force a login if no valid user id
		userID := GetCtxUserID(ctx)
		if userID == nil {
			//store the request uri in a cookie
			s.SetCookieRequestURI(w, r.RequestURI)

			//display the login page
			http.Redirect(w, r.WithContext(ctx), URILogin, http.StatusSeeOther)
			return
		}

		//track the user id in the logger
		ctx = SetLoggerUserID(ctx, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

//extract a book id from the url
func (s *Server) bookIDHdlr(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, _ := GetLogger(s.getCtx(r))

		//read the book id from the path
		bookID := chi.URLParam(r.WithContext(ctx), URLParams.BookID)
		if bookID == "" {
			panic("read book id")
		}

		//store the book id and proceed
		ctx = SetCtxBookID(ctx, bookID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

//check the host to redirect
func (s *Server) checkHostHdlr(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, _ := GetLogger(r.Context())

		//check if not the standard domain
		tokens := strings.Split(r.Host, ":")
		port := ""
		if len(tokens) > 1 {
			port = tokens[1]
		}
		ok := s.redirectHost(w, r.WithContext(ctx), tokens[0], port)
		if !ok {
			return
		}

		//check for a forwarded host
		host := r.Header.Get(HeaderForwardedHost)
		tokens = strings.Split(host, ":")
		port = ""
		if len(tokens) > 1 {
			port = tokens[1]
		}
		if host != "" {
			ok = s.redirectHost(w, r.WithContext(ctx), tokens[0], port)
			if !ok {
				return
			}
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

//probe for a timezone in a cookie
func (s *Server) cookieTimeZoneHdlr(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))

		//read the timezone from the cookie
		timeZone, err := s.GetCookieTimeZone(r)
		if err != nil {
			logger.Warnw("read cookie timezone", "error", err)
		} else if timeZone != "" {
			//store the timezone
			ctx = SetCtxTimeZone(ctx, timeZone)
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

//force a redirect to https
func (s *Server) httpsRedirectHdlr(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, _ := GetLogger(s.getCtx(r))

		//avoid modifying the original url
		url := *r.URL

		//store the request id in a cookie
		requestID := GetCtxRequestID(s.getCtx(r))
		if requestID != "" {
			s.SetCookieRequestID(w, requestID)
		}

		//explicitly set the host and set the port and scheme
		tokens := strings.Split(r.Host, ":")
		url.Host = fmt.Sprintf("%s%s", tokens[0], GetServerAddressPublic())
		url.Scheme = GetServerSchemePublic()
		http.Redirect(w, r.WithContext(ctx), url.String(), http.StatusMovedPermanently)
	})
}

//log all requests
func (s *Server) logHdlr(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//prepare the logger
		ctx, logger := GetLogger(s.getCtx(r), "method", r.Method, "host", r.Host, "https", r.TLS != nil, "protocol", r.Proto, "url", r.URL, "remoteAddress", r.RemoteAddr)

		//use the logging writer to capture additional data
		lw := s.createLoggingResponseWriter(w)

		//capture the duration of the request
		start := time.Now()
		defer func() {
			//compute the duration as a decimal number
			logger.Infow("request", "statusCode", lw.statusCode, "durationMS", FormatElapsedMS(start))
			AddCtxStatsAPI(ctx, ServerStatWeb, "request", time.Since(start))
		}()
		next.ServeHTTP(lw, r.WithContext(ctx))
	})
}

//recover from panics
func (s *Server) panicHdlr(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if !GetPanicHandlerDisable() {
				data := recover()
				if data != nil {
					_, ok := data.(timeoutError)
					if !ok {
						err, ok := data.(error)
						if ok {
							s.logger.Warnw("panic", "error", err, "stack", string(debug.Stack()))
						} else {
							s.logger.Warnw("panic", "data", data)
						}
					}
					AddCtxStatsCount(s.getCtx(r), ServerStatLogPanics, 1)
					s.invokeHdlrGet(s.handleProviderErr(), w, r)
					return
				}
			}
		}()
		next.ServeHTTP(w, r)
	})
}

//extract a payment id from the url
func (s *Server) paymentIDHdlr(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, _ := GetLogger(s.getCtx(r))

		//read the payment id from the path
		paymentID := chi.URLParam(r.WithContext(ctx), URLParams.PaymentID)
		if paymentID == "" {
			panic("read payment id")
		}

		//store the payment id and proceed
		ctx = SetCtxPaymentID(ctx, paymentID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

//extract a provider url name from the url
func (s *Server) providerURLNameHdlr(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, _ := GetLogger(s.getCtx(r))

		//read the provider url name from the path
		providerURLName := chi.URLParam(r.WithContext(ctx), URLParams.ProviderURLName)
		if providerURLName == "" {
			panic("read provider url name")
		}

		//store the provider url name and proceed
		ctx = SetCtxProviderURLName(ctx, providerURLName)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

//preprocessing handler
func (s *Server) preprocessHdlr(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))

		//check for certain parameters in the query string
		authToken := r.FormValue(URLParams.AuthToken)
		msgKey := r.FormValue(URLParams.MsgKey)

		//store in cookies
		if authToken != "" {
			s.SetCookieToken(w, authToken)
		} else if msgKey != "" {
			ctx = SetCtxMsg(ctx, GetMsgText(MsgKey(msgKey)))
		}

		//look for a host override
		host, err := s.GetCookieHost(r)
		if err != nil {
			logger.Warnw("get cookie host", "error", err)
		}
		if host != "" {
			ctx = SetCtxCustomHost(ctx, host)
		}

		//proceed
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

//extract a request id, generating one if necessary
func (s *Server) requestIDHdlr(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))

		//probe the header for the request id
		requestID := r.Header.Get(HeaderRequestID)
		if requestID == "" {
			//probe for a request id cookie
			requestIDCookie, err := s.GetCookieRequestID(r.WithContext(ctx))
			if err != nil {
				panic(errors.Wrap(err, "get cookie request id"))
			}
			requestID = requestIDCookie
			s.DeleteCookieRequestID(w)
		}

		//create a new request id if necessary
		if requestID == "" {
			uuid, err := uuid.NewV4()
			if err != nil {
				logger.Warnw("new uuid", "error", err)
				requestID = ""
			} else {
				requestID = uuid.String()
			}
		}
		ctx = SetCtxRequestID(ctx, requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

//extract a service id from the url
func (s *Server) serviceIDHdlr(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, _ := GetLogger(s.getCtx(r))

		//read the service id from the path
		svcID := chi.URLParam(r.WithContext(ctx), URLParams.SvcID)
		if svcID == "" {
			panic("read service id")
		}

		//store the service id and proceed
		ctx = SetCtxServiceID(ctx, svcID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

//timeout error
type timeoutError struct {
	data interface{}
}

//introduce an explicit time-out
func (s *Server) timeoutHdlr(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))

		//set-up the timeout
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(GetServerTimeOutHandlerSec())*time.Second)
		defer cancel()

		//use a routine for the rest of the call and use channels for signalling
		channelDone := make(chan struct{})
		channelPanic := make(chan interface{}, 1)
		go func(logger *Logger) {
			defer func(logger *Logger) {
				if !GetPanicHandlerDisable() {
					data := recover()
					if data != nil {
						err, ok := data.(error)
						if ok {
							s.logger.Warnw("panic timeout handler", "error", err, "stack", string(debug.Stack()))
						} else {
							s.logger.Warnw("panic timeout handler", "data", data)
						}
						timeoutErr := timeoutError{
							data: err,
						}
						channelPanic <- timeoutErr
					}
				}
			}(logger)
			next.ServeHTTP(w, r.WithContext(ctx))
			close(channelDone)
		}(logger)
		select {
		case err := <-channelPanic:
			panic(err)
		case <-channelDone:
		case <-ctx.Done():
			logger.Warnw("timeout")
			w.WriteHeader(http.StatusGatewayTimeout)
		}
	})
}

//extract a shortened-url from the url
func (s *Server) urlShortHdlr(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, _ := GetLogger(s.getCtx(r))

		//read the shortened-url from the path
		urlShort := chi.URLParam(r.WithContext(ctx), URLParams.URLShort)
		if urlShort == "" {
			panic("read url short")
		}

		//store the shortened url and proceed
		ctx = SetCtxURLShort(ctx, urlShort)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

//extract a user id
func (s *Server) userIDHdlr(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))

		//probe for a valid token
		ok, userID, err := s.readToken(r.WithContext(ctx))
		if err != nil {
			//delete the token
			s.DeleteCookieToken(w)

			//display the login page
			logger.Warnw("read token", "error", err)
			http.Redirect(w, r.WithContext(ctx), URILogin, http.StatusSeeOther)
			return
		}

		//refresh the token
		if ok && userID != nil {
			_, err = s.refreshToken(w, r.WithContext(ctx), userID)
			if err != nil {
				//delete the token
				s.DeleteCookieToken(w)

				//display the login page
				logger.Warnw("refresh token", "error", err)
				http.Redirect(w, r.WithContext(ctx), URILogin, http.StatusSeeOther)
				return
			}
		}

		//store the user id and proceed
		ctx = SetCtxUserID(ctx, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
