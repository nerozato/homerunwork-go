package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/pprof"

	"github.com/go-chi/chi"
)

//base client urls
const (
	BaseAPIURL               = "/api"
	BaseAuthURL              = "/auth"
	BaseClientServiceBookURL = "/b"
	BaseClientServiceURL     = "/s"
	BaseClientURL            = "/p"
	BaseDashboardURL         = "/dashboard"
	BasePaymentURL           = "/payment"
	BaseShortenedURL         = "/u"
	BaseWebHookURL           = "/webhook"
)

//create the router
func (s *Server) createRouter(ctx context.Context) *chi.Mux {
	serverRouter := chi.NewRouter()
	serverRouter.MethodNotAllowed(s.handleProviderErr())
	serverRouter.NotFound(s.handleProviderErr404())

	//set-up middleware
	serverRouter.Use(s.checkHostHdlr) //check the host domains
	serverRouter.Use(s.panicHdlr)     //use the panic handler as the 1st main handler
	serverRouter.Use(s.requestIDHdlr)
	serverRouter.Use(s.logHdlr)

	//standard routes
	serverRouter.Group(func(router chi.Router) {
		router.Use(s.preprocessHdlr)
		router.Use(s.userIDHdlr)
		router.Use(s.cookieTimeZoneHdlr)
		router.Use(s.timeoutHdlr)

		//routes
		router.Get(BaseClientURL, s.handleProviderBase())
		router.Get(URIAbout, s.handleProviderAbout())
		router.Get(URIDefault, s.handleProviderIndex())
		router.Get(URIEmailVerify, s.handleProviderEmailVerify())
		router.Get(URIErr, s.handleProviderErr())
		router.Get(URIFaq, s.handleProviderFaq())
		router.Get(URIHowItWorks, s.handleProviderHowItWorks())
		router.Get(URIHowTo, s.handleProviderHowTo())
		router.Get(URIIndex, s.handleProviderIndex())
		router.Get(URILogout, s.handleProviderLogout())
		router.Get(URIPolicy, s.handleProviderPolicy())
		router.Get(URIProviders, s.handleProviderProviders())
		router.Get(URIRobots, s.handleProviderRobots())
		router.Get(URIServiceAreas, s.handlerProviderServiceAreas())
		router.Get(URISignUpPricing, s.handleProviderSignUpPricing())
		router.Get(URISiteMap, s.handleProviderSiteMap())
		router.Get(URISupport, s.handleProviderSupport())
		router.Get(URITerms, s.handleProviderTerms())
		router.Get(URIZoomSupport, s.handleProviderZoomSupport())

		router.Get(URICampaignManage, s.handleProviderCampaignManage())
		router.Post(URICampaignManage, s.handleProviderCampaignManage())

		router.Get(URIForgotPwd, s.handleProviderForgotPwd())
		router.Post(URIForgotPwd, s.handleProviderForgotPwd())

		router.Get(URILogin, s.handleProviderLogin())
		router.Post(URILogin, s.handleProviderLogin())

		router.Get(URIPayment, s.handleProviderPayment())
		router.Post(URIPayment, s.handleProviderPayment())

		router.Get(URIPwdReset, s.handleProviderPwdReset())
		router.Post(URIPwdReset, s.handleProviderPwdReset())

		router.Get(URISignUp, s.handleProviderSignUp())
		router.Post(URISignUp, s.handleProviderSignUp())

		//landing pages
		//tutors
		router.Route(URITutors, func(r chi.Router) {
			r.Get(URIDefault, s.handleProviderLandingTutors())
			r.Get(URIIndex, s.handleProviderLandingTutors())
		})

		//protected provider routes
		router.Group(func(r chi.Router) {
			r.Use(s.authProviderHdlr)
			r.Get(URISignUpLink, s.handleProviderSignUpLink())
			r.Get(URISignUpSuccess, s.handleProviderSignUpSuccess())

			r.Get(URISignUpMain, s.handleProviderSignUpMain())
			r.Post(URISignUpMain, s.handleProviderSignUpMain())

			//dashboard
			r.Route(BaseDashboardURL, func(sr chi.Router) {
				sr.Get(URIBookingAddSuccess, s.handleDashboardBookingAddSuccess())
				sr.Get(URIBookingCancelSuccess, s.handleDashboardBookingCancelSuccess())
				sr.Get(URIBookingEditSuccess, s.handleDashboardBookingEditSuccess())
				sr.Get(URIBookings, s.handleDashboardBookings())
				sr.Get(URICalendars, s.handleDashboardCalendars())
				sr.Get(URICampaignAddStep3, s.handleDashboardCampaignAddStep3())
				sr.Get(URICampaigns, s.handleDashboardCampaigns())
				sr.Get(URICoupons, s.handleDashboardCoupons())
				sr.Get(URIDefault, s.handleDashboardIndex())
				sr.Get(URIIndex, s.handleDashboardIndex())
				sr.Get(URIPayments, s.handleDashboardPayments())
				sr.Get(URIUsers, s.handleDashboardUsers())

				sr.Get(URIAbout, s.handleDashboardAbout())
				sr.Post(URIAbout, s.handleDashboardAbout())

				sr.Get(URIAccount, s.handleDashboardAccount())
				sr.Post(URIAccount, s.handleDashboardAccount())

				sr.Get(URIAddOns, s.handleDashboardAddOns())
				sr.Post(URIAddOns, s.handleDashboardAddOns())

				sr.Get(URIBookingAdd, s.handleDashboardBookingAdd())
				sr.Post(URIBookingAdd, s.handleDashboardBookingAdd())

				sr.Get(URIBookingEdit, s.handleDashboardBookingEdit())
				sr.Post(URIBookingEdit, s.handleDashboardBookingEdit())

				sr.Get(URIBookingView, s.handleDashboardBookingView())
				sr.Post(URIBookingView, s.handleDashboardBookingView())

				sr.Get(URICalendars, s.handleDashboardCalendars())
				sr.Post(URICalendars, s.handleDashboardCalendars())

				sr.Get(URICampaignAddStep1, s.handleDashboardCampaignAddStep1())
				sr.Post(URICampaignAddStep1, s.handleDashboardCampaignAddStep1())

				sr.Get(URICampaignAddStep2, s.handleDashboardCampaignAddStep2())
				sr.Post(URICampaignAddStep2, s.handleDashboardCampaignAddStep2())

				sr.Get(URICampaignView, s.handleDashboardCampaignView())
				sr.Post(URICampaignView, s.handleDashboardCampaignView())

				sr.Get(URIClientAdd, s.handleDashboardClientAdd())
				sr.Post(URIClientAdd, s.handleDashboardClientAdd())

				sr.Get(URIClientEdit, s.handleDashboardClientEdit())
				sr.Post(URIClientEdit, s.handleDashboardClientEdit())

				sr.Get(URIClients, s.handleDashboardClients())
				sr.Post(URIClients, s.handleDashboardClients())

				sr.Get(URICouponAdd, s.handleDashboardCouponAdd())
				sr.Post(URICouponAdd, s.handleDashboardCouponAdd())

				sr.Get(URICouponEdit, s.handleDashboardCouponEdit())
				sr.Post(URICouponEdit, s.handleDashboardCouponEdit())

				sr.Get(URIFaqAdd, s.handleDashboardFaqAdd())
				sr.Post(URIFaqAdd, s.handleDashboardFaqAdd())

				sr.Get(URIFaqEdit, s.handleDashboardFaqEdit())
				sr.Post(URIFaqEdit, s.handleDashboardFaqEdit())

				sr.Get(URIFaqs, s.handleDashboardFaqs())
				sr.Post(URIFaqs, s.handleDashboardFaqs())

				sr.Get(URILinks, s.handleDashboardLinks())
				sr.Post(URILinks, s.handleDashboardLinks())

				sr.Get(URIHours, s.handleDashboardHours())
				sr.Post(URIHours, s.handleDashboardHours())

				sr.Get(URIPayment, s.handleDashboardPayment())
				sr.Post(URIPayment, s.handleDashboardPayment())

				sr.Get(URIPaymentSettings, s.handleDashboardPaymentSettings())
				sr.Post(URIPaymentSettings, s.handleDashboardPaymentSettings())

				sr.Get(URIPaymentView, s.handleDashboardPaymentView())
				sr.Post(URIPaymentView, s.handleDashboardPaymentView())

				sr.Get(URIProfile, s.handleDashboardProfile())
				sr.Post(URIProfile, s.handleDashboardProfile())

				sr.Get(URIProfileDomain, s.handleDashboardProfileDomain())
				sr.Post(URIProfileDomain, s.handleDashboardProfileDomain())

				sr.Get(URISvcAdd, s.handleDashboardServiceAdd())
				sr.Post(URISvcAdd, s.handleDashboardServiceAdd())

				sr.Get(URISvcEdit, s.handleDashboardServiceEdit())
				sr.Post(URISvcEdit, s.handleDashboardServiceEdit())

				sr.Get(URISvcUsers, s.handleDashboardServiceUsers())
				sr.Post(URISvcUsers, s.handleDashboardServiceUsers())

				sr.Get(URISvcs, s.handleDashboardServices())
				sr.Post(URISvcs, s.handleDashboardServices())

				sr.Get(URITestimonialAdd, s.handleDashboardTestimonialAdd())
				sr.Post(URITestimonialAdd, s.handleDashboardTestimonialAdd())

				sr.Get(URITestimonialEdit, s.handleDashboardTestimonialEdit())
				sr.Post(URITestimonialEdit, s.handleDashboardTestimonialEdit())

				sr.Get(URITestimonials, s.handleDashboardTestimonials())
				sr.Post(URITestimonials, s.handleDashboardTestimonials())

				sr.Get(URIUserAdd, s.handleDashboardUserAdd())
				sr.Post(URIUserAdd, s.handleDashboardUserAdd())

				sr.Get(URIUserEdit, s.handleDashboardUserEdit())
				sr.Post(URIUserEdit, s.handleDashboardUserEdit())

				//api
				sr.Route(BaseAPIURL, func(ssr chi.Router) {
					if GetDevModeEnable() {
						ssr.Get(URIEmail, s.handleAPIEmail())
					}
					ssr.Get(URIOrdersCalendar, s.handleAPIOrdersCalendar())
				})
			})
		})

		//client routes
		router.Route(fmt.Sprintf("%s/{%s}", BaseClientURL, URLParams.ProviderURLName), func(r chi.Router) {
			r.Use(s.providerURLNameHdlr)
			r.Get(URIAbout, s.handleClientAbout())
			r.Get(URIFaq, s.handleClientFaq())
			r.Get(URIDefault, s.handleClientIndex())
			r.Get(URIIndex, s.handleClientIndex())

			r.Get(URIContact, s.handleClientContact())
			r.Post(URIContact, s.handleClientContact())

			r.Get(URIPaymentDirect, s.handleClientPaymentDirect())
			r.Post(URIPaymentDirect, s.handleClientPaymentDirect())

			//provider payment routes
			r.Route(fmt.Sprintf("%s/{%s}", BasePaymentURL, URLParams.PaymentID), func(sr chi.Router) {
				sr.Use(s.paymentIDHdlr)

				sr.Get(URIPayment, s.handleClientPayment())
				sr.Post(URIPayment, s.handleClientPayment())
			})

			//provider service routes
			r.Route(fmt.Sprintf("%s/{%s}", BaseClientServiceURL, URLParams.SvcID), func(sr chi.Router) {
				sr.Use(s.serviceIDHdlr)
				sr.Get(URIDefault, s.handleClientServiceIndex())

				sr.Get(URIBooking, s.handleClientBooking())
				sr.Post(URIBooking, s.handleClientBooking())

				sr.Get(URIBookingSubmit, s.handleClientBookingSubmit())
				sr.Post(URIBookingSubmit, s.handleClientBookingSubmit())

				//provider booking routes
				sr.Route(fmt.Sprintf("%s/{%s}", BaseClientServiceBookURL, URLParams.BookID), func(ssr chi.Router) {
					ssr.Use(s.bookIDHdlr)
					ssr.Get(URIBookingConfirm, s.handleClientBookingConfirm())

					ssr.Get(URICancel, s.handleClientBookingCancel())
					ssr.Post(URICancel, s.handleClientBookingCancel())

					ssr.Get(URIDefault, s.handleClientBookingView())
					ssr.Post(URIDefault, s.handleClientBookingView())

					ssr.Get(URIPayment, s.handleClientPayment())
					ssr.Post(URIPayment, s.handleClientPayment())
				})
			})
		})

		//auth routes
		router.Route(BaseAuthURL, func(r chi.Router) {
			//google routes
			r.Route(URIGoogle, func(sr chi.Router) {
				sr.Get(URICallback, s.handleGoogleOAuthCallback())
				sr.Get(URICallbackCal, s.handleGoogleOAuthCalCallback())
				sr.Get(URIOAuthLogin, s.handleGoogleLogin())
			})

			//stripe routes
			r.Route(URIStripe, func(sr chi.Router) {
				sr.Get(URICallback, s.handleStripeOAuthCallback())
				sr.Get(URIOAuthLogin, s.handleStripeLogin())
			})

			//zoom routes
			r.Route(URIZoom, func(sr chi.Router) {
				sr.Get(URICallback, s.handleZoomOAuthCallback())
				sr.Get(URIOAuthLogin, s.handleZoomLogin())
			})
		})

		//payment routes
		router.Route(BasePaymentURL, func(r chi.Router) {
			//paypal routes
			r.Route(URIPayPal, func(sr chi.Router) {
				sr.Post(URICallback, s.handlePayPalWebHookCallback())
			})

			//stripe routes
			r.Route(URIStripe, func(sr chi.Router) {
				sr.Post(URICallback, s.handleStripeWebHookCallback())
			})
		})

		//webhook routes
		router.Route(BaseWebHookURL, func(r chi.Router) {
			//zoom routes
			r.Route(URIZoom, func(sr chi.Router) {
				sr.Post(URICallback, s.handleZoomWebHookCallback())
			})
		})

		//shortened-url routes
		router.Route(fmt.Sprintf("%s/{%s}", BaseShortenedURL, URLParams.URLShort), func(r chi.Router) {
			r.Use(s.urlShortHdlr)
			r.Get(URIDefault, s.handleAPIURLShort())
		})

		//api
		router.Route(BaseAPIURL, func(r chi.Router) {
			r.Use(s.apiTokenHdlr)
			r.Post(URIContent, s.handleAPIContent())
			r.Post(URIMaintenance, s.handleAPIMaintenance())
			r.Get(URIReport, s.handleAPIReport())
			r.Get(URIStats, s.handleAPIStats())

			//development api
			if GetDevModeEnable() {
				r.Delete(URIEmail, s.handleAPIEmailDelete())
			}
		})

		//configure the profiler
		if GetPProfEnable() {
			s.logger.Infow("initialize pprof routes")
			router.Get("/debug/pprof/*", http.HandlerFunc(pprof.Index))
			router.Get("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
			router.Get("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
			router.Get("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
			router.Get("/debug/pprof/trace", http.HandlerFunc(pprof.Trace))
		}
	})

	//serve static files
	fsAssets := fileOnlyFileSystem{http.Dir(BaseWebAssetPath)}
	s.setupStaticFiles(ctx, serverRouter, URIAssets, fsAssets)
	return serverRouter
}

//create the maintenance router
func (s *Server) createMaintenanceRouter() *chi.Mux {
	router := chi.NewRouter()
	router.Use(s.logHdlr)
	router.Get("/*", s.handleProviderErrMaintenance())
	router.Post("/*", s.handleProviderErrMaintenance())

	//api
	router.Route(BaseAPIURL, func(r chi.Router) {
		r.Use(s.apiTokenHdlr)
		r.Post(URIMaintenance, s.handleAPIMaintenance())
	})
	return router
}

//initialize the router used for redirecting http to https
func (s *Server) initHTTPSRedirectRouter() http.Handler {
	//force the redirect
	router := chi.NewRouter()
	router.Handle("/*", s.httpsRedirectHdlr(nil))
	return router
}
