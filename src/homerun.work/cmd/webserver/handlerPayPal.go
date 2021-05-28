package main

import (
	"bytes"
	"io/ioutil"
	"net/http"
)

//handle the paypal webhook callback
func (s *Server) handlePayPalWebHookCallback() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, logger := GetLogger(s.getCtx(r))

		//peek the body
		var err error
		var body []byte
		var bodyStr string
		if r.Body != nil {
			body, err = ioutil.ReadAll(r.Body)
			if err != nil {
				logger.Errorw("body read all", "error", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			bodyStr = string(body)
		} else {
			logger.Errorw("no body")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		r.Body = ioutil.NopCloser(bytes.NewBuffer(body))

		//verify a valid event
		ok, err := VerifyWebHookSignaturePayPal(r.WithContext(ctx), GetPayPalWebHookID())
		if err != nil {
			logger.Errorw("verify signature paypal", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if !ok {
			logger.Errorw("invalid signature paypal")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		//parse the data and validate
		event, err := ParseEventPayPal(body)
		if err != nil {
			logger.Errorw("parse event paypal", "error", err, "body", bodyStr)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		//store the event
		ctx, err = SaveEventPayPal(ctx, s.getDB(), event)
		if err != nil {
			logger.Errorw("save event paypal", "error", err, "body", bodyStr)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		//sanity check the event
		if event.EventVersion != PayPalEventVersion {
			logger.Errorw("invalid event version", "error", err, "body", bodyStr)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if event.ResourceVersion != PayPalResourceVersion {
			logger.Errorw("invalid resource version", "error", err, "body", bodyStr)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		//process the event
		if event.ResourceType == PayPalResourceTypeCapture {
			if event.EventType == PayPalEventTypePaymentCaptureCompleted {
				resource, err := ParseResourcePaymentPayPal([]byte(event.Resource))
				if err != nil {
					logger.Errorw("parse payment resource paypal", "error", err, "body", event.Resource)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				//store the response
				now := GetTimeNow("")
				ctx, err = UpdatePaymentCaptured(ctx, s.getDB(), &resource.InvoiceID, &bodyStr, &now)
				if err != nil {
					logger.Errorw("update payment", "error", err, "body", event.Resource)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
			}
		}
		w.WriteHeader(http.StatusOK)
	}
}
