package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"math"
	"net"
	"net/http"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi"
	"github.com/gofrs/uuid"
	"github.com/jhillyerd/enmime"
	"github.com/pkg/errors"
)

//MaxMemoryUploadMB : maximum memory to use on file uploads
const MaxMemoryUploadMB = 1 << 20 // 1 MB

//MaxImgSvcCount : maximum number of images for a service
const MaxImgSvcCount = 10

//create the url to a provider page
func createProviderURL(providerURLName string, url string) string {
	return path.Join(BaseClientURL, providerURLName, url)
}

//create the url to the provider payment page
func createProviderPaymentURL(providerURLName string, id *uuid.UUID) string {
	return path.Join(BaseClientURL, providerURLName, BasePaymentURL, id.String(), URIPayment)
}

//create the url to a provider service page
func createProviderServiceURL(providerURLName string, svcID *uuid.UUID, url string) string {
	return path.Join(BaseClientURL, providerURLName, BaseClientServiceURL, svcID.String(), url)
}

//create the url to a provider booking page
func createProviderServiceBookURL(providerURLName string, svcID *uuid.UUID, bookID *uuid.UUID, url string) string {
	serviceURL := createProviderServiceURL(providerURLName, svcID, BaseClientServiceBookURL)
	return path.Join(serviceURL, bookID.String(), url)
}

//create the url to a dashboard page
func createDashboardURL(url string) string {
	return path.Join(BaseDashboardURL, url)
}

//create the URL for the provider dashboard
func createDashboardURLDashboard() string {
	return createDashboardURL(URIBookings)
}

//create the url to a dashboard api
func createDashboardAPIURL(url string) string {
	return path.Join(BaseDashboardURL, BaseAPIURL, url)
}

//create the absolute url to a google oauth endpoint
func createGoogleURL(url string) (string, error) {
	url, err := CreateURLAbs(nil, path.Join(BaseAuthURL, URIGoogle, url), nil)
	if err != nil {
		return "", errors.Wrap(err, fmt.Sprintf("invalid google url: %s", url))
	}
	return url, nil
}

//create the short url
func createShortURL(url string) string {
	return path.Join(BaseShortenedURL, url)
}

//create the absolute url to a stripe oauth endpoint
func createStripeURL(url string) (string, error) {
	url, err := CreateURLAbs(nil, path.Join(BaseAuthURL, URIStripe, url), nil)
	if err != nil {
		return "", errors.Wrap(err, fmt.Sprintf("invalid stripe url: %s", url))
	}
	return url, nil
}

//create the absolute url to a zoom oauth endpoint
func createZoomURL(url string) (string, error) {
	url, err := CreateURLAbs(nil, path.Join(BaseAuthURL, URIZoom, url), nil)
	if err != nil {
		return "", errors.Wrap(err, fmt.Sprintf("invalid zoom url: %s", url))
	}
	return url, nil
}

//ServerStatKey : keys for server statistics
type ServerStatKey string

//server statistics
const (
	ServerStatAPIAWS                    ServerStatKey = "AWS"
	ServerStatAPIBitly                  ServerStatKey = "Bitly"
	ServerStatAPIFacebook               ServerStatKey = "Facebook"
	ServerStatAPIGoogle                 ServerStatKey = "Google"
	ServerStatAPIPayPal                 ServerStatKey = "PayPal"
	ServerStatAPIPlaid                  ServerStatKey = "Plaid"
	ServerStatAPIStripe                 ServerStatKey = "Stripe"
	ServerStatAPIZoom                   ServerStatKey = "Zoom"
	ServerStatCurrentTime               ServerStatKey = "CurrentTime"
	ServerStatDB                        ServerStatKey = "DB"
	ServerStatLogPanics                 ServerStatKey = "Panics"
	ServerStatLogWarnings               ServerStatKey = "Warnings"
	ServerStatLogErrors                 ServerStatKey = "Errors"
	ServerStatMaintenanceEnabled        ServerStatKey = "MaintenanceEnabled"
	ServerStatProcessGoogle             ServerStatKey = "ProcessGoogle"
	ServerStatProcessImgs               ServerStatKey = "ProcessImgs"
	ServerStatProcessMsgs               ServerStatKey = "ProcessMsgs"
	ServerStatProcessNotifications      ServerStatKey = "ProcessNotifications"
	ServerStatProcessIncomingEmail      ServerStatKey = "ProcessIncomingEmail"
	ServerStatProcessIncomingEmailCount ServerStatKey = "ProcessIncomingEmailCount"
	ServerStatRecurringOrders           ServerStatKey = "ProcessRecurringOrders"
	ServerStatWeb                       ServerStatKey = "Web"
	ServerStatZoom                      ServerStatKey = "ProcessZoom"
)

//APIStatistic : statistic for an API call
type APIStatistic struct {
	Count        int64
	AvgElapsedMS float64
}

//add statistics data
func (s *APIStatistic) addStat(duration time.Duration) {
	s.AvgElapsedMS = ((float64(s.Count) * s.AvgElapsedMS) + (duration.Seconds() * 1000)) / float64(s.Count+1)
	s.Count = s.Count + 1
}

//APIStatistics : statistics for an API call
type APIStatistics struct {
	mu    *sync.RWMutex
	Stats map[string]*APIStatistic
}

//add statistics data
func (s *APIStatistics) addStat(label string, duration time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	//save the data
	stat, ok := s.Stats[label]
	if !ok {
		stat = &APIStatistic{}
		s.Stats[label] = stat
	}
	stat.addStat(duration)
}

//ServerStatistics : server statistics
type ServerStatistics struct {
	mu       *sync.RWMutex
	Counts   map[ServerStatKey]int64
	Data     map[ServerStatKey]interface{}
	Times    map[ServerStatKey]time.Time
	APIStats map[ServerStatKey]*APIStatistics
}

//AddData : add statistic data
func (s *ServerStatistics) AddData(key ServerStatKey, data interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()

	//save the data
	s.Data[key] = data
}

//AddTime : add time statistic data
func (s *ServerStatistics) AddTime(key ServerStatKey, time time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	//save the data
	s.Times[key] = time
}

//AddAPIStat : add statistic data for an API call
func (s *ServerStatistics) AddAPIStat(key ServerStatKey, label string, duration time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	//initialize if necessary and save the data
	apiStat, ok := s.APIStats[key]
	if !ok {
		apiStat = &APIStatistics{
			mu:    &sync.RWMutex{},
			Stats: make(map[string]*APIStatistic, 10),
		}
		s.APIStats[key] = apiStat
	}
	apiStat.addStat(label, duration)
}

//Count : accumulate a count
func (s *ServerStatistics) Count(key ServerStatKey, c int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	count := s.Counts[key]
	count = count + int64(c)
	s.Counts[key] = count
}

//DisplayTimes : display times as local times based on the timezone
func (s *ServerStatistics) DisplayTimes(timeZone string) {
	loc := GetLocation(timeZone)
	now := time.Now().In(loc)
	s.AddData(ServerStatCurrentTime, now.Format(time.RFC3339))
	if timeZone != "" {
		s.mu.RLock()
		defer s.mu.RUnlock()
		for k, v := range s.Times {
			vtz := v.In(loc)
			s.mu.RUnlock()
			s.AddData(k, vtz.Format(time.RFC3339))
			s.mu.RLock()
		}
	}
}

//CreateJSON : marshal the data as JSON
func (s *ServerStatistics) CreateJSON() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	data, err := json.Marshal(s)
	return data, err
}

//CreateServer : create a server instance
func CreateServer() *Server {
	s := &Server{
		maintenanceEnable: false,
		stats: &ServerStatistics{
			mu:       &sync.RWMutex{},
			Counts:   make(map[ServerStatKey]int64, 10),
			Data:     make(map[ServerStatKey]interface{}, 10),
			Times:    make(map[ServerStatKey]time.Time, 10),
			APIStats: make(map[ServerStatKey]*APIStatistics, 10),
		},
	}
	s.init()
	return s
}

//Server : server definition
type Server struct {
	wg                sync.WaitGroup
	awsSession        *AWSSession
	ctx               context.Context
	db                *DB
	label             string
	logger            *Logger
	maintenanceEnable bool
	router            http.Handler
	scheduler         *Scheduler
	serverHTTP        *http.Server
	serverHTTPS       *http.Server
	stats             *ServerStatistics
	validator         *Validator
}

//get the db connection
func (s *Server) getDB() *DB {
	//copy the database object
	db := &DB{
		db: s.db.db, //db connection can be shared
		tx: nil,     //transaction cannot be shared
	}
	return db
}

//get the context
func (s *Server) getCtx(r *http.Request) context.Context {
	ctx := r.Context()
	ctx = SetCtxStats(ctx, s.stats)
	return ctx
}

//set a message in the context
func (s *Server) setCtxMsg(ctx context.Context, key MsgKey, args ...interface{}) context.Context {
	msg := GetMsgText(key, args...)
	ctx = SetCtxMsg(ctx, msg)

	//find the title to use
	title := GetMsgTitle(key)
	ctx = SetCtxTitleAlert(ctx, title)
	return ctx
}

//set an error message in the context
func (s *Server) setCtxErr(ctx context.Context, key ErrKey, args ...interface{}) context.Context {
	msg := GetErrText(key, args...)
	ctx = SetCtxErr(ctx, msg)

	//find the title to use
	title := GetErrTitle(key)
	ctx = SetCtxTitleAlert(ctx, title)
	return ctx
}

//initialize the server
func (s *Server) init() {
	ctx := context.Background()
	ctx = SetCtxStats(ctx, s.stats)
	s.ctx = ctx

	//initialize the logger
	logger, err := InitLogger(ctx, "")
	if err != nil {
		panic(fmt.Sprintf("log failure: %v", err))
	}
	s.logger = logger

	//initialize aws
	awsSession, err := InitAWS(ctx)
	if err != nil {
		panic(errors.Wrap(err, "init aws"))
	}
	s.awsSession = awsSession

	//initialize the validator
	validator := InitValidator()
	s.validator = validator

	//initialize stripe
	InitStripe(s.logger)
}

//create an http server
func (s *Server) createHTTPServer(address string, router http.Handler) *http.Server {
	return &http.Server{
		Addr:         address,
		ReadTimeout:  time.Duration(GetServerTimeOutReadSec()) * time.Second,
		WriteTimeout: time.Duration(GetServerTimeOutWriteSec()) * time.Second,
		Handler:      router,
	}
}

//Start : start the server
func (s *Server) Start() {
	//check if maintenance mode should be used
	if s.maintenanceEnable {
		s.StartMaintenance()
		return
	}
	s.label = "main"
	s.wg.Wait()
	s.logger.Infow("start server", "label", s.label)
	s.wg.Add(1)
	defer s.wg.Done()

	//initialize the db
	db, err := OpenDB(s.ctx, GetDBAddress(), GetDBUser(), GetDBPwd(), GetDBName())
	if err != nil {
		panic(errors.Wrap(err, "init db"))
	}
	s.db = db

	//start the sqs processing
	s.awsSession.ProcessSQSEmail(s.processSQSEmailMsg)

	//start the scheduler
	scheduler := InitScheduler(s.ctx, s)
	s.scheduler = scheduler
	s.scheduler.Start()

	//configure the router
	s.router = s.createRouter(s.ctx)

	//configure the http server
	s.serverHTTP = s.createHTTPServer(GetServerAddressHTTP(), s.router)

	//start the https server
	if GetServerUseHTTPS() {
		s.serverHTTPS = s.createHTTPServer(GetServerAddressHTTPS(), s.router)
		go func(logger *Logger) {
			logger.Infow("https starting", "address", s.serverHTTPS.Addr, "label", s.label)
			err := s.serverHTTPS.ListenAndServeTLS(GetServerTLSCert(), GetServerTLSKey())
			if err != http.ErrServerClosed {
				logger.Errorw("https unhandled", "error", err, "label", s.label)
				s.stopHTTPServer(s.ctx, s.serverHTTPS)
			}
		}(s.logger)

		//change the http server to always redirect to https
		s.serverHTTP.Handler = s.initHTTPSRedirectRouter()
	}

	//start the http server
	go func(logger *Logger) {
		logger.Infow("http starting", "address", s.serverHTTP.Addr, "label", s.label)
		err := s.serverHTTP.ListenAndServe()
		if err != http.ErrServerClosed {
			logger.Errorw("http unhandled", "error", err, "label", s.label)
			s.stopHTTPServer(s.ctx, s.serverHTTP)
		}
	}(s.logger)
}

//StartMaintenance : start the server in maintenance mode
func (s *Server) StartMaintenance() {
	s.label = "maintenance"
	s.wg.Wait()
	s.logger.Infow("start maintenance server")
	s.wg.Add(1)
	defer s.wg.Done()

	//configure maintenance router
	s.router = s.createMaintenanceRouter()

	//start the https server
	if GetServerUseHTTPS() {
		s.serverHTTPS = s.createHTTPServer(GetServerAddressHTTPS(), s.router)
		go func(logger *Logger) {
			logger.Infow("https starting", "address", s.serverHTTPS.Addr, "label", s.label)
			err := s.serverHTTPS.ListenAndServeTLS(GetServerTLSCert(), GetServerTLSKey())
			if err != http.ErrServerClosed {
				logger.Errorw("https unhandled", "error", err, "label", s.label)
				s.stopHTTPServer(s.ctx, s.serverHTTPS)
			}
		}(s.logger)
	}

	//start the http server
	s.serverHTTP = s.createHTTPServer(GetServerAddressHTTP(), s.router)
	go func(logger *Logger) {
		logger.Infow("http starting", "address", s.serverHTTP.Addr, "label", s.label)
		err := s.serverHTTP.ListenAndServe()
		if err != http.ErrServerClosed {
			logger.Errorw("http unhandled", "error", err, "label", s.label)
			s.stopHTTPServer(s.ctx, s.serverHTTP)
		}
	}(s.logger)
}

//Stop : stop the server
func (s *Server) Stop(ctx context.Context) {
	s.stopHTTPServer(ctx, s.serverHTTPS)
	s.stopHTTPServer(ctx, s.serverHTTP)

	//stop aws
	s.awsSession.Stop()

	//stop the scheduler
	s.scheduler.Stop()

	//close the db
	err := s.db.Close()
	if err != nil {
		s.logger.Warnw("db close", "error", err, "label", s.label)
	}
	s.logger.Infow("db close", "label", s.label)
	s.logger.Infow("stop server", "label", s.label)
}

//StopMaintenance : stop the maintenance server
func (s *Server) StopMaintenance(ctx context.Context) {
	s.stopHTTPServer(ctx, s.serverHTTPS)
	s.stopHTTPServer(ctx, s.serverHTTP)
	s.logger.Infow("stop server", "label", s.label)
}

//stop an http server
func (s *Server) stopHTTPServer(ctx context.Context, server *http.Server) {
	if server != nil {
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		s.logger.Infow("http shutdown", "address", server.Addr, "label", s.label)
		err := server.Shutdown(ctx)
		if err != nil {
			s.logger.Warnw("http shutdown", "error", err, "label", s.label)
		}
	}
}

//create the function map for use in the template
func (s *Server) createTemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"forceURLAbs":           ForceURLAbs,
		"addURLEmailProviderID": AddURLEmailProviderID,
		"addURLStep":            AddURLStep,
		"addURLType":            AddURLType,
	}
}

//process an sqs email message
func (s *Server) processSQSEmailMsg(ctx context.Context, inMsg *AWSSQSMsg) error {
	ctx, logger := GetLogger(ctx)

	//read the sqs message
	sqsData := make(map[string]string)
	err := json.Unmarshal([]byte(*inMsg.Body), &sqsData)
	if err != nil {
		return errors.Wrap(err, "unmarshal json sqs")
	}

	//read the email data
	msgData, ok := sqsData["Message"]
	if !ok {
		return fmt.Errorf("no message data")
	}
	var data AWSSQSMsgData
	err = json.Unmarshal([]byte(msgData), &data)
	if err != nil {
		return errors.Wrap(err, "unmarshal json sqs data")
	}

	//parse the email
	inEmailContent, err := base64.StdEncoding.DecodeString(data.Content)
	if err != nil {
		return errors.Wrap(err, "decode email")
	}
	inEmail, err := enmime.ReadEnvelope(strings.NewReader(string(inEmailContent)))
	if err != nil {
		return errors.Wrap(err, "read email envelope")
	}

	//extract the ids from the email address
	tos, err := inEmail.AddressList("To")
	if err != nil {
		return errors.Wrap(err, "read email tos")
	}
	if len(tos) == 0 {
		return errors.Wrap(err, "no email tos")
	}
	msgID, err := ParseReplyToAddress(tos[0].Address)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("parse reply-to: %s", tos[0].Address))
	}

	//load the message that was the source
	ctx, msg, err := LoadMsgByID(ctx, s.getDB(), msgID.ID)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("load message: %s", msgID.ID))
	}

	//prepare the message to send
	outMsg := &Message{
		SecondaryID:  msg.SecondaryID,
		FromUserID:   msg.ToUserID,
		FromClientID: msg.ToClientID,
		Type:         MsgTypeMessage,
		Subject:      inEmail.GetHeader("Subject"),
		BodyText:     StripEmail(inEmail.Text),
	}
	if msg.FromUserID != nil {
		//load the user that sent the email
		_, user, err := LoadUserByID(ctx, s.getDB(), msg.FromUserID)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("load user: %s", msg.FromUserID))
		}

		//set-up the email receiver
		outMsg.ToUserID = msg.FromUserID
		outMsg.ToEmail = user.Email
	} else if msg.FromClientID != nil {
		//load the client that sent the email
		_, client, err := LoadClientByID(ctx, s.getDB(), msg.FromClientID)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("load client: %s", msg.FromClientID))
		}
		if client == nil {
			return fmt.Errorf("no client: %s", msg.FromClientID)
		}

		//set-up the email receiver
		outMsg.ToClientID = msg.FromClientID
		outMsg.ToEmail = client.Email
	} else {
		logger.Debugw("no from-user or from-client to send email")
		return nil
	}

	//queue the email
	ctx, err = SaveMsg(ctx, s.getDB(), outMsg)
	if err != nil {
		return errors.Wrap(err, "save message")
	}
	return nil
}

//render a web template
func (s *Server) renderWebTemplate(w http.ResponseWriter, r *http.Request, tpl *template.Template, data templateData) {
	ctx, logger := GetLogger(s.getCtx(r))

	//probe the cookie for a flag
	v, err := s.GetCookieFlag(r)
	if err != nil {
		logger.Warnw("invalid flag cookie", "error", err)
	}
	if v != "" {
		data[TplParamCookieFlag] = v
	}
	s.DeleteCookieFlag(w)

	//probe the cookie for a message
	msg, err := s.GetCookieMsg(r)
	if err != nil {
		logger.Warnw("invalid message cookie", "error", err)
	}
	if msg != "" {
		data[TplParamMsg] = msg
	}
	s.DeleteCookieMsg(w)

	//probe the context for a message
	msg = GetCtxMsg(ctx)
	if msg != "" {
		data[TplParamMsg] = msg
	}

	//probe the cookie for an error
	errMsg, err := s.GetCookieErr(r)
	if err != nil {
		logger.Warnw("invalid error cookie", "error", err)
	}
	if errMsg != "" {
		data[TplParamErr] = errMsg
	}
	s.DeleteCookieErr(w)

	//probe the context for an errors
	errMsg = GetCtxErr(ctx)
	if errMsg != "" {
		data[TplParamErr] = errMsg
	}

	//probe the cookie for an alert title
	title, err := s.GetCookieTitleAlert(r)
	if err != nil {
		logger.Warnw("invalid title cookie", "error", err)
	}
	if title != "" {
		data[TplParamTitleAlert] = title
	}
	s.DeleteCookieTitleAlert(w)

	//probe the content for an alert title
	title = GetCtxTitleAlert(ctx)
	if title != "" {
		data[TplParamTitleAlert] = title
	}

	//check for a marquee to be displayed for a provider
	provider, ok := data[TplParamProvider].(*providerUI)
	if ok && provider != nil {
		//load the new bookings count
		now := data[TplParamCurrentTime].(time.Time)
		user := provider.GetProviderUser()
		_, count, err := CountBookingsByProviderIDAndFilter(ctx, s.getDB(), provider.ID, user, BookingFilterNew, "", now)
		if err != nil {
			logger.Warnw("count bookings new", "error", err, "id", provider.ID)
		}
		data[TplParamMarquee] = GenerateMarqueeProvider(provider, count)
	}

	//pass the context
	data[TplParamContext] = ctx

	//force the template data to be keyed by string for use in the template
	templateMap := data.CreateMap()
	err = tpl.Execute(w, templateMap)
	if err != nil {
		panic(errors.Wrap(err, "template execute"))
	}
}

//invoke a handler as an http get request
func (s *Server) invokeHdlrGet(fn http.HandlerFunc, w http.ResponseWriter, r *http.Request) {
	r.Method = http.MethodGet
	fn(w, r)
}

//prepare to service static files on the specified path
func (s *Server) setupStaticFiles(ctx context.Context, router chi.Router, path string, fs http.FileSystem) {
	//exclude path parameters
	if strings.ContainsAny(path, "{}*") {
		panic(fmt.Sprintf("invalid static file path: %s", path))
	}

	//prepare to serve the file
	hdlr := http.StripPrefix(path, http.FileServer(fs))
	if path != "/" && path[len(path)-1] != '/' {
		router.Get(path, http.RedirectHandler(path+"/", 301).ServeHTTP)
		path += "/"
	}
	path += "*"

	//attach the handler
	router.Get(path, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hdlr.ServeHTTP(w, r)
	}))
}

//load a provider based on the user id in the context
func (s *Server) loadProvider(w http.ResponseWriter, r *http.Request) (context.Context, *providerUI, bool) {
	ctx, logger := GetLogger(s.getCtx(r))

	//probe for the user id
	userID := GetCtxUserID(ctx)
	if userID == nil {
		logger.Errorw("no user id")
		http.Redirect(w, r.WithContext(ctx), URILogin, http.StatusSeeOther)
		return ctx, nil, false
	}

	//probe for a provider explicitly linked to a user
	var provider *Provider
	ctx, providerUser, err := LoadProviderUserByUserID(ctx, s.getDB(), userID)
	if err != nil {
		logger.Warnw("load provider user", "error", err, "id", userID)
	}
	if providerUser != nil {
		ctx, provider, err = LoadProviderByID(ctx, s.getDB(), providerUser.ProviderID)
		if err != nil {
			logger.Errorw("load provider", "error", err, "id", providerUser.ID)
			http.Redirect(w, r.WithContext(ctx), URILogin, http.StatusSeeOther)
			return ctx, nil, false
		}
		provider.ProviderUser = providerUser
	} else {
		ctx, provider, err = LoadProviderByUserID(ctx, s.getDB(), userID)
		if err != nil {
			logger.Errorw("load provider", "error", err, "id", userID)
			http.Redirect(w, r.WithContext(ctx), URIErr, http.StatusSeeOther)
			return ctx, nil, false
		}
	}

	//check if a provider has been loaded
	if provider == nil {
		return ctx, nil, true
	}

	//store the provider's timezone in the context
	ctx = SetCtxTimeZone(ctx, provider.User.TimeZone)
	providerUI := s.createProviderUI(provider)
	return ctx, providerUI, true
}

//load a provider based on the url name in the path
func (s *Server) loadProviderByURLName(w http.ResponseWriter, r *http.Request, tpl *template.Template, data templateData) (context.Context, *providerUI, bool) {
	ctx, logger := GetLogger(s.getCtx(r))

	//check for a provider url name
	providerURLName := GetCtxProviderURLName(ctx)
	if providerURLName == "" {
		logger.Warnw("no provider name")
		s.redirectError(w, r.WithContext(ctx), Err)
		return ctx, nil, false
	}

	//load the provider
	ctx, provider, err := LoadProviderByURLName(ctx, s.getDB(), providerURLName)
	if err != nil {
		logger.Errorw("load provider", "error", err, "name", providerURLName)
		s.invokeHdlrGet(s.handleProviderErr404(), w, r.WithContext(ctx))
		return ctx, nil, false
	}
	providerUI := s.createProviderUI(provider)
	return ctx, providerUI, true
}

//load the provider hours
func (s *Server) loadTemplateProviderSchedule(w http.ResponseWriter, r *http.Request, tpl *template.Template, data templateData, provider *providerUI) bool {
	ctx, logger := GetLogger(s.getCtx(r))
	schedule, err := s.getSchedule(provider)
	if err != nil {
		logger.Errorw("get provider schedule", "error", err)
		data[TplParamErr] = GetErrText(Err)
		s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
		return false
	}
	data[TplParamSchedule] = schedule
	return true
}

//load the services for a provider
func (s *Server) loadTemplateServices(w http.ResponseWriter, r *http.Request, tpl *template.Template, data templateData, provider *providerUI) (context.Context, []*serviceUI, bool) {
	ctx, logger := GetLogger(s.getCtx(r))
	ctx, svcs, err := ListServices(ctx, s.getDB(), provider.ID)
	if err != nil {
		logger.Errorw("list services", "error", err, "id", provider.ID)
		data[TplParamErr] = GetErrText(Err)
		s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
		return ctx, nil, false
	}
	svcUIs := s.createServiceUIs(provider, svcs)
	data[TplParamSvcs] = svcUIs
	return ctx, svcUIs, true
}

//load a service
func (s *Server) loadTemplateService(w http.ResponseWriter, r *http.Request, tpl *template.Template, data templateData, provider *providerUI, svcID *uuid.UUID, now time.Time) (context.Context, *serviceUI, bool) {
	ctx, logger := GetLogger(s.getCtx(r))

	//load the service
	ctx, svc, err := LoadServiceByProviderIDAndID(ctx, s.getDB(), provider.ID, svcID)
	if err != nil {
		logger.Errorw("load service", "error", err, "providerId", provider.ID, "id", svcID)
		data[TplParamErr] = GetErrText(Err)
		s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
		return ctx, nil, false
	}
	svcUI := s.createServiceUI(provider, svc)
	data[TplParamSvc] = svcUI
	return ctx, svcUI, true
}

//load the campaign
func (s *Server) loadTemplateCampaign(w http.ResponseWriter, r *http.Request, tpl *template.Template, data templateData, provider *providerUI, showDeleted bool) (context.Context, *campaignUI, bool) {
	ctx, logger := GetLogger(s.getCtx(r))

	//validate the id
	idStr := r.FormValue(URLParams.ID)
	campaignID := uuid.FromStringOrNil(idStr)
	if campaignID == uuid.Nil {
		logger.Warnw("invalid uuid", "id", idStr)
		s.SetCookieErr(w, Err)
		http.Redirect(w, r.WithContext(ctx), provider.GetURLCampaigns(), http.StatusSeeOther)
		return ctx, nil, false
	}

	//load the campaign
	ctx, campaign, err := LoadCampaignByProviderIDAndID(ctx, s.getDB(), provider.Provider.ID, &campaignID, showDeleted)
	if err != nil {
		logger.Errorw("load campaign", "error", err, "id", campaignID)
		s.SetCookieErr(w, Err)
		http.Redirect(w, r.WithContext(ctx), provider.GetURLCampaigns(), http.StatusSeeOther)
		return ctx, nil, false
	}
	campaignUI := s.createCampaignUI(campaign)
	data[TplParamCampaign] = campaignUI
	return ctx, campaignUI, true
}

//load the clients for a provider
func (s *Server) loadTemplateClients(w http.ResponseWriter, r *http.Request, tpl *template.Template, data templateData, providerID *uuid.UUID) (context.Context, []*Client, bool) {
	ctx, logger := GetLogger(s.getCtx(r))
	ctx, clients, err := ListClientsByProviderID(ctx, s.getDB(), providerID)
	if err != nil {
		logger.Errorw("list clients", "error", err, "id", providerID)
		data[TplParamErr] = GetErrText(Err)
		s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
		return ctx, nil, false
	}
	data[TplParamClients] = clients
	return ctx, clients, true
}

//load the client
func (s *Server) loadTemplateClient(w http.ResponseWriter, r *http.Request, tpl *template.Template, data templateData, errs map[string]string, provider *providerUI) (context.Context, *Client, bool) {
	ctx, logger := GetLogger(s.getCtx(r))

	//validate the id
	idStr := r.FormValue(URLParams.ClientID)
	clientID := uuid.FromStringOrNil(idStr)
	if clientID == uuid.Nil {
		logger.Warnw("invalid uuid", "id", idStr)
		s.SetCookieErr(w, Err)
		http.Redirect(w, r.WithContext(ctx), provider.GetURLClients(), http.StatusSeeOther)
		return ctx, nil, false
	}

	//load the client
	ctx, client, err := LoadClientByProviderIDAndID(ctx, s.getDB(), provider.ID, &clientID)
	if err != nil {
		logger.Errorw("load client", "error", err, "id", clientID)
		data[TplParamErr] = GetErrText(Err)
		s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
		return ctx, nil, false
	}
	if client == nil {
		logger.Errorw("no client", "id", clientID)
		data[TplParamErr] = GetErrText(Err)
		s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
		return ctx, nil, false
	}
	data[TplParamClient] = client
	return ctx, client, true
}

//create a service
func (s *Server) createService(provider *providerUI, form *ServiceForm) *Service {
	svc := &Service{
		Type: ServiceTypeAppt,
	}
	svc.Provider = provider.Provider
	svc.SetFields(form.ApptOnly, form.Name, form.Description, form.Note, form.Price, form.PriceType, form.Duration, form.LocationType, form.Location, form.Padding, form.PaddingInitial, form.PaddingInitialUnit, form.Interval, form.EnableZoom, form.URLVideo)
	return svc
}

//CreateTimePeriods : create the time periods given a from and to time
func (s *Server) createTimePeriods(now time.Time, minStart time.Time, date time.Time, provider *providerUI, svc *serviceUI, existingBooks []*Booking, busyTimes []*TimePeriod, isClient bool) (time.Time, []*TimePeriod) {
	from, to := provider.GetBoundaryTimes(date)
	if from.IsZero() || to.IsZero() {
		return time.Time{}, nil
	}

	//compute the service time periods every service interval and the from and to times
	startInterval := svc.GetInterval()
	var serviceDuration time.Duration
	if svc.IsApptOnly() {
		serviceDuration = time.Duration(svc.Duration) * time.Minute
	}
	totalDuration := to.Sub(from)
	count := int(math.Ceil(totalDuration.Minutes() / startInterval.Minutes()))

	//create the time periods
	var firstAvailableTime time.Time
	timePeriods := make([]*TimePeriod, count)
	for i := 0; i < count; i++ {
		start := from.Add(time.Duration(i * int(startInterval)))
		timePeriod := &TimePeriod{
			Start: start,
			End:   start.Add(serviceDuration),
		}

		//check for a valid period
		if isClient && start.Before(minStart) {
			//check if before the given date
			timePeriod.Unavailable = true
		} else if !provider.IsValidWorkPeriod(from, timePeriod) {
			//check if the time falls in a valid period
			timePeriod.Hidden = true
		} else {
			//check for conflicts with existing bookings
			for _, existingBook := range existingBooks {
				ok := timePeriod.IsOverlap(existingBook.TimeFromPadded, existingBook.TimeToPadded)
				timePeriod.Unavailable = ok
				if timePeriod.Unavailable {
					break
				}
			}

			//check for conflicts with the busy times
			if !timePeriod.Unavailable && busyTimes != nil {
				for _, busyTime := range busyTimes {
					ok := timePeriod.IsOverlap(busyTime.Start, busyTime.End)
					timePeriod.Unavailable = ok
					if timePeriod.Unavailable {
						break
					}
				}
			}

			//check if the slot is in the past
			if !timePeriod.Unavailable {
				timePeriod.Unavailable = timePeriod.Start.Before(now)
			}
		}

		//pick the first available time
		if firstAvailableTime.IsZero() && !timePeriod.Unavailable && !timePeriod.Hidden {
			firstAvailableTime = timePeriod.Start
		}
		timePeriods[i] = timePeriod
	}
	return firstAvailableTime, timePeriods
}

//load the service and time slots
func (s *Server) generateServiceTimes(ctx context.Context, provider *providerUI, svc *serviceUI, svcStartDate time.Time, date time.Time, isClient bool) (context.Context, time.Time, time.Time, []*TimePeriod, error) {
	var err error
	var books []*Booking
	var googleCalBusyTimes []*TimePeriod

	//check for conflicts
	if svc.IsApptOnly() {
		from, to := provider.GetBoundaryTimes(date)
		if !from.IsZero() && !to.IsZero() {
			//load the existing bookings, filtering explicitly by the user if from the client
			user := provider.GetProviderUser()
			if isClient && user == nil {
				user = provider.User
			}
			ctx, books, err = ListBookingsByProviderIDAndTypeAndTime(ctx, s.getDB(), provider.ID, user, ServiceTypeAppt, from, to)
			if err != nil {
				return ctx, time.Time{}, time.Time{}, nil, errors.Wrap(err, fmt.Sprintf("load bookings: %s", provider.ID))
			}
			//load any busy times based on the calendar
			if provider.User.GoogleCalendarToken != nil {
				var googleToken *TokenGoogle
				googleToken, googleCalBusyTimes, err = ListBusyCalendarGoogle(ctx, provider.User.GoogleCalendarToken, from, to)

				//save token if refreshed
				if googleToken != nil {
					provider.User.GoogleCalendarToken = googleToken
					ctx, err := SaveUser(ctx, s.getDB(), provider.User, "")
					if err != nil {
						return ctx, time.Time{}, time.Time{}, nil, errors.Wrap(err, fmt.Sprintf("save provider google token: %s", provider.ID))
					}
				}
				if err != nil {
					return ctx, time.Time{}, time.Time{}, nil, errors.Wrap(err, fmt.Sprintf("load google busy times: %s", provider.ID))
				}
			}
		}
	}

	//generate the available times
	firstAvailableTime, timePeriods := s.createTimePeriods(date, svcStartDate, date, provider, svc, books, googleCalBusyTimes, isClient)
	return ctx, svcStartDate, firstAvailableTime, timePeriods, nil
}

//generate the time periods and find the date and first available service time given the date
func (s *Server) generateTimes(ctx context.Context, provider *providerUI, svc *serviceUI, svcStartDate time.Time, date time.Time, isClient bool) (context.Context, time.Time, time.Time, []*TimePeriod, error) {
	//check if any times are available and try the next date if necessary
	var err error
	var firstAvailableTime time.Time
	var timePeriods []*TimePeriod
	for i := 0; i < 7; i++ {
		ctx, svcStartDate, firstAvailableTime, timePeriods, err = s.generateServiceTimes(ctx, provider, svc, svcStartDate, date, isClient)
		if err != nil {
			return ctx, time.Time{}, time.Time{}, nil, errors.Wrap(err, "generate times")
		}
		if !firstAvailableTime.IsZero() {
			break
		}

		//try the next date
		date = date.AddDate(0, 0, 1)
		date = GetBeginningOfDay(date)
	}
	return ctx, svcStartDate, firstAvailableTime, timePeriods, nil
}

//load the service and time slots
func (s *Server) loadTemplateServiceTimes(w http.ResponseWriter, r *http.Request, tpl *template.Template, data templateData, errs map[string]string, provider *providerUI, svc *serviceUI, dateStr string, now time.Time, isClient bool) (context.Context, time.Time, time.Time, []*TimePeriod, bool) {
	ctx, logger := GetLogger(s.getCtx(r))

	//determine the days of week that should be disabled
	days := provider.ListDaysOfWeekUnavailable()
	data[TplParamDaysOfWeek] = strings.Trim(strings.Replace(fmt.Sprint(days), " ", ",", -1), "[]")

	//sanity check the date and load the information for that date if set
	svcStartDate := svc.ComputeStartTime(now)
	var err error
	var firstAvailableTime time.Time
	var timePeriods []*TimePeriod

	//compute the relevant times for the first possible service date
	ctx, svcStartDate, firstAvailableTime, timePeriods, err = s.generateTimes(ctx, provider, svc, svcStartDate, svcStartDate, isClient)
	if err != nil {
		logger.Errorw("generate times", "error", err, "id", provider.ID, "date", svcStartDate)
		data[TplParamErr] = GetErrText(Err)
		s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
		return ctx, time.Time{}, time.Time{}, nil, false
	}

	//compute the relevant times for the specified date
	if dateStr != "" {
		form := DateForm{
			Date: dateStr,
		}
		ok := s.validateForm(w, r.WithContext(ctx), tpl, data, errs, form, true)
		if !ok {
			return ctx, time.Time{}, time.Time{}, nil, false
		}
		timeZone := GetCtxTimeZone(ctx)
		date := ParseDateLocal(form.Date, timeZone)

		//check if the date is valid
		if isClient {
			testDate := GetBeginningOfDay(svcStartDate)
			if date.Before(testDate) {
				logger.Debugw("invalid service start date", "date", date, "test", testDate)
				errs[string(FieldErrDate)] = GetFieldErrText(string(FieldErrDate))
				s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
				return ctx, time.Time{}, time.Time{}, nil, false
			}
		}

		//generate the relevant times for the date
		ctx, _, firstAvailableTime, timePeriods, err = s.generateTimes(ctx, provider, svc, svcStartDate, date, isClient)
		if err != nil {
			logger.Errorw("generate times", "error", err, "id", provider.ID, "date", date)
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return ctx, time.Time{}, time.Time{}, nil, false
		}
	}
	return ctx, svcStartDate, firstAvailableTime, timePeriods, true
}

//create the provider schedule
func (s *Server) createSchedule(provider *providerUI, now time.Time, unavailMon bool, unavailTue bool, unavailWed bool, unavailThu bool, unavailFri bool, unavailSat bool, unavailSun bool, startStr string, duration int) error {
	//sanity check the start
	start := ParseTimeLocalAsUTC(startStr, now, provider.User.TimeZone)
	if start == nil {
		return fmt.Errorf("invalid start: %s", start)
	}

	//create a schedule for a day of the week
	createDaySchedule := func(dayOfWeek time.Weekday, unavailable bool, start time.Time, duration int) *DaySchedule {
		schedule := &DaySchedule{
			DayOfWeek:   dayOfWeek,
			Unavailable: unavailable,
		}
		if !schedule.Unavailable {
			schedule.TimeDurations = []*TimeDuration{
				{
					Start:    start,
					Duration: duration,
				},
			}
		}
		return schedule
	}

	//create the schedule
	providerSchedules := make(map[time.Weekday]*DaySchedule, 7)
	providerSchedules[time.Monday] = createDaySchedule(time.Monday, unavailMon, *start, duration)
	providerSchedules[time.Tuesday] = createDaySchedule(time.Tuesday, unavailTue, *start, duration)
	providerSchedules[time.Wednesday] = createDaySchedule(time.Wednesday, unavailWed, *start, duration)
	providerSchedules[time.Thursday] = createDaySchedule(time.Thursday, unavailThu, *start, duration)
	providerSchedules[time.Friday] = createDaySchedule(time.Friday, unavailFri, *start, duration)
	providerSchedules[time.Saturday] = createDaySchedule(time.Saturday, unavailSat, *start, duration)
	providerSchedules[time.Sunday] = createDaySchedule(time.Sunday, unavailSun, *start, duration)
	providerSchedule := &ProviderSchedule{
		DaySchedules: providerSchedules,
	}
	errDays := providerSchedule.Process(now, provider.User.TimeZone)
	if len(errDays) > 0 {
		return fmt.Errorf("schedule: %v", errDays)
	}
	provider.SetSchedule(providerSchedule)
	return nil
}

//set the schedule from the form JSON, returning the days of the week that have a problem
func (s *Server) setSchedule(ctx context.Context, provider *providerUI, inputJSON string, now time.Time, timeZone string) ([]int, error) {
	_, logger := GetLogger(ctx)

	//parse as json
	var formSchedule []*DayScheduleForm
	err := json.Unmarshal([]byte(inputJSON), &formSchedule)
	if err != nil {
		return nil, errors.Wrap(err, "unjson form")
	}

	//list the days that have issues
	daysOfWeek := make([]int, 0, 7)

	//validate the form
	var providerSchedules map[time.Weekday]*DaySchedule
	form := ProviderScheduleForm{
		DaySchedules: formSchedule,
	}
	ok, _, err := s.validator.Validate(form)
	if err != nil {
		return nil, errors.Wrap(err, "validate form")
	} else if ok {
		//validate the schedule and prepare to update the provider
		providerSchedules = make(map[time.Weekday]*DaySchedule, len(form.DaySchedules))
		for scheduleIdx, schedule := range form.DaySchedules {
			ok, fieldErrs, err := s.validator.Validate(schedule)
			if err != nil {
				return nil, errors.Wrap(err, fmt.Sprintf("validate schedule: %s", schedule.DayOfWeek))
			}
			if ok {
				//parse the day of the week
				dayOfWeek, ok := ParseWeekDay(schedule.DayOfWeek)
				if ok {
					providerSchedule := &DaySchedule{
						DayOfWeek:   dayOfWeek,
						Unavailable: !schedule.Available,
					}
					providerSchedules[dayOfWeek] = providerSchedule

					//process the durations for the day of the week
					if !providerSchedule.Unavailable {
						providerTimeDurations := make([]*TimeDuration, len(schedule.TimeDurations))
						for idx, timeDuration := range schedule.TimeDurations {
							start := ParseTimeLocalAsUTC(timeDuration.Start, now, timeZone)
							if start == nil {
								return nil, fmt.Errorf("invalid start: %s: %s", schedule.DayOfWeek, timeDuration.Start)
							}
							providerTimeDurations[idx] = &TimeDuration{
								Start:    *start,
								Duration: timeDuration.Duration,
							}
						}

						//sanity check the times, making sure the time durations are sorted
						sort.SliceStable(providerTimeDurations, func(i int, j int) bool {
							return providerTimeDurations[i].Start.Before(providerTimeDurations[j].Start)
						})
						invalid := false
						end := time.Time{}
						for _, timeDuration := range providerTimeDurations {
							//given the sort, just check if the next start overlaps the previous end
							if !end.IsZero() && timeDuration.Start.Before(end) {
								invalid = true
								break
							}
							end = timeDuration.GetEnd()
						}
						if invalid {
							logger.Warnw("invalid durations", "weekday", schedule.DayOfWeek, "durations", providerTimeDurations)
							daysOfWeek = append(daysOfWeek, scheduleIdx)
						} else {
							providerSchedule.TimeDurations = providerTimeDurations
						}
					}
				} else {
					logger.Warnw("parse weekday", "weekday", schedule.DayOfWeek)
					daysOfWeek = append(daysOfWeek, scheduleIdx)
				}
			} else {
				logger.Warnw("schedule validation errors", "weekday", schedule.DayOfWeek, "fields", fieldErrs)
				daysOfWeek = append(daysOfWeek, scheduleIdx)
			}
		}
	}
	if len(daysOfWeek) > 0 {
		return daysOfWeek, nil
	}

	//process the schedule to properly bucket by the day of the week
	schedule := &ProviderSchedule{
		DaySchedules: providerSchedules,
	}
	days := schedule.Process(now, timeZone)
	if len(days) > 0 {
		//convert the days of the week to indices based on the incoming schedule
		for _, day := range days {
			for scheduleIdx, schedule := range form.DaySchedules {
				if day == schedule.DayOfWeek {
					daysOfWeek = append(daysOfWeek, scheduleIdx)
					break
				}
			}
		}
		return daysOfWeek, nil
	}
	provider.SetSchedule(schedule)
	return nil, nil
}

//get the schedule form json
func (s *Server) getSchedule(provider *providerUI) (string, error) {
	schedule := provider.GetSchedule()
	if schedule == nil {
		return "", fmt.Errorf("no schedule")
	}

	//sort by the days of the week, starting on monday
	providerSchedules := make([]*DaySchedule, len(schedule.DaySchedules))
	providerSchedules[0] = schedule.DaySchedules[time.Monday]
	providerSchedules[1] = schedule.DaySchedules[time.Tuesday]
	providerSchedules[2] = schedule.DaySchedules[time.Wednesday]
	providerSchedules[3] = schedule.DaySchedules[time.Thursday]
	providerSchedules[4] = schedule.DaySchedules[time.Friday]
	providerSchedules[5] = schedule.DaySchedules[time.Saturday]
	providerSchedules[6] = schedule.DaySchedules[time.Sunday]

	//create the schedules
	formSchedules := make([]*DayScheduleForm, len(providerSchedules))
	for idx, schedule := range providerSchedules {
		formSchedule := &DayScheduleForm{
			DayOfWeek: schedule.DayOfWeek.String(),
			Available: !schedule.Unavailable,
		}
		formSchedules[idx] = formSchedule
		if formSchedule.Available {
			formTimeDurations := make([]*TimeDurationForm, len(schedule.TimeDurations))
			formSchedule.TimeDurations = formTimeDurations
			for idx, timeDuration := range schedule.TimeDurations {
				formTimeDurations[idx] = &TimeDurationForm{
					Start:    FormatTimeLocal(timeDuration.Start, provider.User.TimeZone),
					Duration: timeDuration.Duration,
				}
			}
		} else {
			formTimeDurations := make([]*TimeDurationForm, 0)
			formSchedule.TimeDurations = formTimeDurations
		}
	}

	//encode as json
	formJSON, err := json.Marshal(formSchedules)
	if err != nil {
		return "", errors.Wrap(err, "json form")
	}
	return string(formJSON), nil
}

//load the booking
func (s *Server) loadTemplateBook(w http.ResponseWriter, r *http.Request, tpl *template.Template, data templateData, errs map[string]string, idStr string, updateViewed bool, includeDeleted bool) (context.Context, *bookingUI, bool) {
	ctx, logger := GetLogger(s.getCtx(r))

	//validate the id
	bookID := uuid.FromStringOrNil(idStr)
	if bookID == uuid.Nil {
		logger.Warnw("invalid uuid", "id", idStr)
		return ctx, nil, false
	}

	//load the booking
	ctx, book, err := LoadBookingByID(ctx, s.getDB(), &bookID, updateViewed, includeDeleted)
	if err != nil {
		logger.Errorw("load booking", "error", err, "id", bookID)
		return ctx, nil, false
	}
	bookUI := s.createBookingUI(book)
	data[TplParamBook] = bookUI
	return ctx, bookUI, true
}

//load the faqs for a provider
func (s *Server) loadTemplateFaqs(w http.ResponseWriter, r *http.Request, tpl *template.Template, data templateData, provider *providerUI) (context.Context, []*Faq, bool) {
	ctx, logger := GetLogger(s.getCtx(r))
	ctx, faqs, err := ListFaqsByProviderID(ctx, s.getDB(), provider.Provider)
	if err != nil {
		logger.Errorw("list faqs", "error", err, "id", provider.ID)
		data[TplParamErr] = GetErrText(Err)
		s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
		return ctx, nil, false
	}
	data[TplParamFaqs] = faqs
	return ctx, faqs, true
}

//load the testimonials for a provider
func (s *Server) loadTemplateTestimonials(w http.ResponseWriter, r *http.Request, tpl *template.Template, data templateData, provider *providerUI) (context.Context, []*testimonialUI, bool) {
	ctx, logger := GetLogger(s.getCtx(r))
	ctx, testimonials, err := ListTestimonialsByProviderID(ctx, s.getDB(), provider.Provider)
	if err != nil {
		logger.Errorw("list testimonials", "error", err, "id", provider.ID)
		data[TplParamErr] = GetErrText(Err)
		s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
		return ctx, nil, false
	}
	testimonialUIs := s.createTestimonialUIs(testimonials)
	data[TplParamTestimonials] = testimonialUIs
	return ctx, testimonialUIs, true
}

//create the template data
func (s *Server) createTemplateData(r *http.Request) templateData {
	data := make(templateData)

	//set the timezone based on the context and find the current time
	var timeZone string
	if r != nil {
		timeZone = GetCtxTimeZone(s.getCtx(r))
		data[TplParamTimeZone] = timeZone
	}
	now := GetTimeNow(timeZone)

	//create basic template data
	data[TplParamActiveNav] = ""
	data[TplParamCouponTypes] = CouponTypes
	data[TplParamCurrentTime] = now
	data[TplParamCurrentYear] = now.Year()
	data[TplParamDevModeEnable] = GetDevModeEnable()
	data[TplParamDisableAuth] = false
	data[TplParamDisableNav] = false
	data[TplParamDomainPublic] = GetDomain()
	data[TplParamDurationsBooking] = ServiceDurationsBooking
	data[TplParamDurationsOrder] = ServiceDurationsOrder
	data[TplParamFileCSS] = GetFileCSS()
	data[TplParamFileJS] = GetFileJS()
	data[TplParamInputs] = URLParams
	data[TplParamIPPublic] = GetServerAddressPublicIP()
	data[TplParamMetaDesc] = GetMsgText(MsgMetaDesc)
	data[TplParamMetaKeywords] = GetMsgText(MsgMetaKeywords)
	data[TplParamPaddingUnits] = PaddingUnits
	data[TplParamPageTitle] = GetMsgText(MsgPageTitle)
	data[TplParamPayPalClientID] = GetPayPalClientID()
	data[TplParamPriceTypes] = PriceTypes
	data[TplParamRecurrenceFreqs] = RecurrenceFreqs
	data[TplParamServiceIntervals] = ServiceIntervals
	data[TplParamServiceLocations] = ServiceLocations
	data[TplParamStripePublicKey] = GetStripePublicKey()
	data[TplParamTimeZones] = TimeZoneList
	data[TplParamTypeSignUp] = ""

	//common urls
	data[TplParamURLAbout] = URIAbout
	data[TplParamURLAssets] = GetURLAssets()
	data[TplParamURLDashboard] = createDashboardURLDashboard()
	data[TplParamURLDefault] = URIDefault
	data[TplParamURLEmailVerify] = URIEmailVerify
	data[TplParamURLFacebook] = GetURLFacebook()
	data[TplParamURLFaq] = URIFaq
	data[TplParamURLForgotPwd] = URIForgotPwd
	data[TplParamURLHowItWorks] = URIHowItWorks
	data[TplParamURLHowTo] = URIHowTo
	data[TplParamURLInstagram] = GetURLInstagram()
	data[TplParamURLLinkedIn] = GetURLLinkedIn()
	data[TplParamURLLogin] = URILogin
	data[TplParamURLLogout] = URILogout
	data[TplParamURLPolicy] = URIPolicy
	data[TplParamURLProviders] = URIProviders
	data[TplParamURLSignUp] = URISignUp
	data[TplParamURLSignUpPricing] = URISignUpPricing
	data[TplParamURLSignUpSuccess] = URISignUpSuccess
	data[TplParamURLSupport] = URISupport
	data[TplParamURLTerms] = URITerms
	data[TplParamURLTutors] = URITutors
	data[TplParamURLTwitter] = GetURLTwitter()
	data[TplParamURLUploads] = GetURLUploads()
	data[TplParamURLYouTube] = GetURLYouTube()

	//add the constants
	constants := make(map[string]interface{})
	constants["ageMin"] = AgeMin
	constants["ageMax"] = AgeMax
	constants["bookingFilterAll"] = BookingFilterAll
	constants["bookingFilterInvoiced"] = BookingFilterInvoiced
	constants["bookingFilterNew"] = BookingFilterNew
	constants["bookingFilterPaid"] = BookingFilterPaid
	constants["bookingFilterPast"] = BookingFilterPast
	constants["bookingFilterUnPaid"] = BookingFilterUnPaid
	constants["bookingFilterUpcoming"] = BookingFilterUpcoming
	constants["campaignBudgetMin"] = CampaignBudgetMin
	constants["campaignFee"] = FormatPrice(CampaignFee)
	constants["campaignFeeFacebookAdAccount"] = FormatPrice(CampaignFeeFacebookAdAccount)
	constants["campaignFeeFacebookPage"] = FormatPrice(CampaignFeeFacebookPage)
	constants["campaignStatuses"] = CampaignStatuses
	constants["cookieErr"] = CookieErr
	constants["cookieMsg"] = CookieMsg
	constants["cookieTimeZone"] = CookieTimeZone
	constants["genderAll"] = GenderAll
	constants["genderMen"] = GenderMen
	constants["genderWomen"] = GenderWomen
	constants["imgAboutHeight"] = ImgAboutHeight
	constants["imgAboutWidth"] = ImgAboutWidth
	constants["imgAdHeight"] = ImgAdHeight
	constants["imgAdWidth"] = ImgAdWidth
	constants["imgBannerHeight"] = ImgBannerHeight
	constants["imgBannerWidth"] = ImgBannerWidth
	constants["imgLogoHeight"] = ImgLogoHeight
	constants["imgLogoWidth"] = ImgLogoWidth
	constants["imgSvcHeight"] = ImgSvcHeight
	constants["imgSvcWidth"] = ImgSvcWidth
	constants["imgTestimonialHeight"] = ImgTestimonialHeight
	constants["imgTestimonialWidth"] = ImgTestimonialWidth
	constants["lenCampaignInterests"] = LenCampaignInterests
	constants["lenCampaignLocations"] = LenCampaignLocations
	constants["lenCodeCoupon"] = LenCodeCoupon
	constants["lenDescBook"] = LenDescBook
	constants["lenDescCoupon"] = LenDescCoupon
	constants["lenDescPayment"] = LenDescPayment
	constants["lenDescProvider"] = LenDescProvider
	constants["lenDescProviderNote"] = LenDescProviderNote
	constants["lenDescSvc"] = LenDescSvc
	constants["lenEducation"] = LenEducation
	constants["lenExperience"] = LenExperience
	constants["lenEmail"] = LenEmail
	constants["lenLocation"] = LenLocation
	constants["lenName"] = LenName
	constants["lenNoteSvc"] = LenNoteSvc
	constants["lenTextCampaign"] = LenTextCampaign
	constants["lenTextContact"] = LenTextContact
	constants["lenTextFaq"] = LenTextFaq
	constants["lenTextLong"] = LenTextLong
	constants["lenTextTestimonal"] = LenTextTestimonal
	constants["lenUrl"] = LenURL
	constants["oauthFacebook"] = OAuthFacebook
	constants["oauthGoogle"] = OAuthGoogle
	constants["paymentFilterAll"] = PaymentFilterAll
	constants["paymentFilterUnPaid"] = PaymentFilterUnPaid
	constants["serviceAreaEducationAndTraining"] = ServiceAreaEducationAndTraining
	data[TplParamConstants] = constants

	//facebook
	data[TplParamFacebookAPIVersion] = GetFacebookAPIVersion()
	data[TplParamFacebookAppID] = GetFacebookAppID()
	data[TplParamFacebookConversionCost] = GetFacebookConversionCost()
	data[TplParamFacebookTrackingID] = GetFacebookTrackingID()

	//google
	data[TplParamGoogleRecaptchaSiteKey] = GetGoogleRecaptchaSiteKey()
	data[TplParamGoogleTagManagerID] = GetGoogleTagManagerID()
	data[TplParamGoogleTrackingID] = GetGoogleTrackingID()

	//probe for a user id
	data[TplParamUserID] = ""
	if r != nil {
		ctx, _ := GetLogger(s.getCtx(r))
		userID := GetCtxUserID(ctx)
		if userID != nil {
			data[TplParamUserID] = userID.String()
		}
	}
	return data
}

//create the campaign data used in the template
func (s *Server) createCampaignUI(campaign *Campaign) *campaignUI {
	return &campaignUI{
		Campaign: campaign,
	}
}

//create the faq data used in the template
func (s *Server) createFaqUI(faq *Faq) *faqUI {
	return &faqUI{
		Faq: faq,
	}
}

//create the array of faq data used in the template
func (s *Server) createFaqUIs(faqs []*Faq) []*faqUI {
	faqsLen := len(faqs)
	datas := make([]*faqUI, faqsLen)
	for i := 0; i < faqsLen; i++ {
		datas[i] = s.createFaqUI(faqs[i])
	}
	return datas
}

//create the payment data used in the template
func (s *Server) createPaymentUI(payment *Payment) *paymentUI {
	return &paymentUI{
		Payment: payment,
	}
}

//create the array of payment data used in the template
func (s *Server) createPaymentUIs(payments []*Payment) []*paymentUI {
	paymentsLen := len(payments)
	datas := make([]*paymentUI, paymentsLen)
	for i := 0; i < paymentsLen; i++ {
		datas[i] = s.createPaymentUI(payments[i])
	}
	return datas
}

//create the provider data used in the template
func (s *Server) createProviderUI(provider *Provider) *providerUI {
	return &providerUI{
		Provider: provider,
	}
}

//create the array of provider data used in the template
func (s *Server) createProviderUIs(providers []*Provider) []*providerUI {
	providersLen := len(providers)
	datas := make([]*providerUI, providersLen)
	for i := 0; i < providersLen; i++ {
		datas[i] = s.createProviderUI(providers[i])
	}
	return datas
}

//create the service data used in the template
func (s *Server) createServiceUI(provider *providerUI, service *Service) *serviceUI {
	svc := &serviceUI{
		Service: service,
	}
	svc.Provider = provider.Provider
	return svc
}

//create the array of service data used in the template
func (s *Server) createServiceUIs(provider *providerUI, svcs []*Service) []*serviceUI {
	svcsLen := len(svcs)
	datas := make([]*serviceUI, svcsLen)
	for i := 0; i < svcsLen; i++ {
		datas[i] = s.createServiceUI(provider, svcs[i])
	}
	return datas
}

//create the booking data used in the template
func (s *Server) createBookingUI(book *Booking) *bookingUI {
	return &bookingUI{
		Booking: book,
	}
}

//create the array of booking data used in the template
func (s *Server) createBookingUIs(books []*Booking) []*bookingUI {
	booksLen := len(books)
	datas := make([]*bookingUI, booksLen)
	for i := 0; i < booksLen; i++ {
		datas[i] = s.createBookingUI(books[i])
	}
	return datas
}

//create the testimonial data used in the template
func (s *Server) createTestimonialUI(testimonial *Testimonial) *testimonialUI {
	return &testimonialUI{
		Testimonial: testimonial,
	}
}

//create the array of testimonial data used in the template
func (s *Server) createTestimonialUIs(testimonials []*Testimonial) []*testimonialUI {
	testiominalsLen := len(testimonials)
	datas := make([]*testimonialUI, testiominalsLen)
	for i := 0; i < testiominalsLen; i++ {
		datas[i] = s.createTestimonialUI(testimonials[i])
	}
	return datas
}

//validate a form
func (s *Server) validateForm(w http.ResponseWriter, r *http.Request, tpl *template.Template, data templateData, errs map[string]string, form interface{}, renderTpl bool) bool {
	ctx, logger := GetLogger(s.getCtx(r))
	ok, fieldErrs, err := s.validator.Validate(form)
	if err != nil {
		logger.Errorw("validation", "error", err, "form", form)
		data[TplParamErr] = GetErrText(Err)
		if renderTpl {
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
		}
		return false
	} else if !ok {
		//process the validation errors
		if GetDevModeEnable() {
			logger.Warnw("validation", "fields", fieldErrs)
		} else {
			logger.Debugw("validation", "fields", fieldErrs)
		}
		for _, fieldErr := range fieldErrs {
			errs[fieldErr] = GetFieldErrText(fieldErr)
		}
		if renderTpl {
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
		}
		return false
	}
	return true
}

//process the file uploads, also storing the file in S3
func (s *Server) processFileUploads(r *http.Request, fileName string, outPath string) (context.Context, []*FileUpload, bool, error) {
	ctx, logger := GetLogger(s.getCtx(r))

	//process the uploaded files
	ctx, uploads, ok, err := ProcessFileUploads(r.WithContext(ctx), fileName, outPath)
	if err != nil {
		return ctx, nil, false, errors.Wrap(err, "process file uploads")
	}
	if !ok {
		return ctx, nil, false, nil
	}

	//upload to aws
	if GetAWSS3Enable() {
		for _, upload := range uploads {
			ctx, err = s.uploadS3File(ctx, upload)
			if err != nil {
				return ctx, nil, false, errors.Wrap(err, "s3 upload")
			}
			err = os.Remove(upload.FullPath)
			if err != nil {
				logger.Warnw("file remove", "error", err, "file", upload)
			}
		}
	}
	return ctx, uploads, true, nil
}

//process the file upload, also storing the file in S3
func (s *Server) processFileUpload(r *http.Request, fileName string, outPath string) (context.Context, *FileUpload, error) {
	ctx, logger := GetLogger(s.getCtx(r))

	//process the uploaded file
	ctx, upload, ok, err := ProcessFileUpload(r.WithContext(ctx), fileName, outPath)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "process file upload")
	}
	if !ok {
		return ctx, nil, nil
	}

	//upload to aws
	if GetAWSS3Enable() {
		ctx, err = s.uploadS3File(ctx, upload)
		if err != nil {
			return ctx, nil, errors.Wrap(err, "s3 upload")
		}
		err = os.Remove(upload.FullPath)
		if err != nil {
			logger.Warnw("file remove", "error", err, "file", upload)
		}
	}
	return ctx, upload, nil
}

//process an image that was uploaded as base64
func (s *Server) processFileUploadBase64(r *http.Request, fileName string, outPath string) (context.Context, *FileUpload, error) {
	ctx, logger := GetLogger(s.getCtx(r))

	//process the uploaded file
	ctx, upload, err := ProcessFileUploadBase64(r.WithContext(ctx), fileName, outPath)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "process file upload")
	}
	if upload == nil {
		return ctx, nil, nil
	}

	//upload to aws
	if GetAWSS3Enable() {
		ctx, err = s.uploadS3File(ctx, upload)
		if err != nil {
			return ctx, nil, errors.Wrap(err, "s3 upload")
		}
		err = os.Remove(upload.FullPath)
		if err != nil {
			logger.Warnw("file remove", "error", err, "file", upload)
		}
	}
	return ctx, upload, nil
}

//upload a file to s3
func (s *Server) uploadS3File(ctx context.Context, file *FileUpload) (context.Context, error) {
	uploadDir := path.Join(URLAssetUpload, file.Path)
	ctx, err := s.awsSession.UploadS3File(ctx, uploadDir, file.FullPath, file.ContentType)
	if err != nil {
		return ctx, errors.Wrap(err, "upload")
	}
	err = os.Remove(file.FullPath)
	if err != nil {
		return ctx, errors.Wrap(err, "remove file")
	}
	return ctx, nil
}

//queue booking cancel emails
func (s *Server) queueEmailsBookingCancel(ctx context.Context, provider *providerUI, svc *serviceUI, book *bookingUI, isClient bool) (context.Context, error) {
	//queue the provider email
	msg := &Message{
		SecondaryID:  book.ID,
		FromClientID: book.Client.ID,
		ToUserID:     provider.User.ID,
		ToEmail:      provider.User.Email,
		ToPhone:      provider.User.GetPhone(),
		Type:         MsgTypeBookingCancelProvider,
	}
	ctx, err := SaveMsg(ctx, s.getDB(), msg)
	if err != nil {
		return ctx, errors.Wrap(err, "save email cancel provider")
	}
	if book.ProviderUser != nil && book.ProviderUser.User != nil {
		msg = &Message{
			SecondaryID:  book.ID,
			FromClientID: book.Client.ID,
			ToUserID:     book.ProviderUser.User.ID,
			ToEmail:      book.ProviderUser.User.Email,
			ToPhone:      book.ProviderUser.User.GetPhone(),
			Type:         MsgTypeBookingCancelProvider,
		}
		ctx, err = SaveMsg(ctx, s.getDB(), msg)
		if err != nil {
			return ctx, errors.Wrap(err, "save email cancel provider")
		}
	}

	//queue the client email
	msg = &Message{
		SecondaryID: book.ID,
		FromUserID:  provider.User.ID,
		ToClientID:  book.Client.ID,
		ToEmail:     book.Client.Email,
		ToPhone:     book.GetClientPhoneSMS(),
		Type:        MsgTypeBookingCancelClient,
		SenderName:  provider.Name,
	}
	ctx, err = SaveMsg(ctx, s.getDB(), msg)
	if err != nil {
		return ctx, errors.Wrap(err, "save email cancel client")
	}
	return ctx, nil
}

//queue booking confirm emails
func (s *Server) queueEmailsBookingConfirm(ctx context.Context, book *bookingUI) (context.Context, error) {
	msg := &Message{
		SecondaryID: book.ID,
		FromUserID:  book.Provider.User.ID,
		ToClientID:  book.Client.ID,
		ToEmail:     book.Client.Email,
		ToPhone:     book.GetClientPhoneSMS(),
		Type:        MsgTypeBookingConfirmClient,
		SenderName:  book.Provider.Name,
	}
	ctx, err := SaveMsg(ctx, s.getDB(), msg)
	if err != nil {
		return ctx, errors.Wrap(err, "save email confirm client")
	}
	return ctx, nil
}

//send booking edit emails
func (s *Server) queueEmailsBookingEdit(ctx context.Context, provider *providerUI, svc *serviceUI, book *bookingUI, isClient bool) (context.Context, error) {
	msg := &Message{
		SecondaryID: book.ID,
		FromUserID:  provider.User.ID,
		ToClientID:  book.Client.ID,
		ToEmail:     book.Client.Email,
		ToPhone:     book.GetClientPhoneSMS(),
		Type:        MsgTypeBookingEditClient,
		SenderName:  provider.Name,
	}
	ctx, err := SaveMsg(ctx, s.getDB(), msg)
	if err != nil {
		return ctx, errors.Wrap(err, "save email booking edit provider")
	}
	return ctx, nil
}

//queue booking new emails
func (s *Server) queueEmailsBookingNew(ctx context.Context, provider *providerUI, svc *serviceUI, book *bookingUI, isClient bool) (context.Context, error) {
	//queue the provider email
	msg := &Message{
		SecondaryID:  book.ID,
		FromClientID: book.Client.ID,
		ToUserID:     provider.User.ID,
		ToEmail:      provider.User.Email,
		ToPhone:      provider.User.GetPhone(),
		Type:         MsgTypeBookingNewProvider,
		IsClient:     isClient,
	}
	ctx, err := SaveMsg(ctx, s.getDB(), msg)
	if err != nil {
		return ctx, errors.Wrap(err, "save email booking new provider")
	}
	if book.ProviderUser != nil && book.ProviderUser.User != nil {
		msg = &Message{
			SecondaryID:  book.ID,
			FromClientID: book.Client.ID,
			ToUserID:     book.ProviderUser.User.ID,
			ToEmail:      book.ProviderUser.User.Email,
			ToPhone:      book.ProviderUser.User.GetPhone(),
			Type:         MsgTypeBookingNewProvider,
			IsClient:     isClient,
		}
		ctx, err = SaveMsg(ctx, s.getDB(), msg)
		if err != nil {
			return ctx, errors.Wrap(err, "save email booking new provider")
		}
	}

	//queue the client email
	msg = &Message{
		SecondaryID: book.ID,
		FromUserID:  provider.User.ID,
		ToClientID:  book.Client.ID,
		ToEmail:     book.Client.Email,
		ToPhone:     book.GetClientPhoneSMS(),
		Type:        MsgTypeBookingNewClient,
		SenderName:  provider.Name,
		IsClient:    isClient,
	}
	ctx, err = SaveMsg(ctx, s.getDB(), msg)
	if err != nil {
		return ctx, errors.Wrap(err, "save email booking new client")
	}
	return ctx, nil
}

//queue booking reminder emails
func (s *Server) queueEmailsBookingReminder(ctx context.Context, book *bookingUI) (context.Context, error) {
	//queue the provider email
	msg := &Message{
		SecondaryID:  book.ID,
		FromClientID: book.Client.ID,
		ToUserID:     book.Provider.User.ID,
		ToEmail:      book.Provider.User.GetEmail(),
		ToPhone:      book.Provider.User.GetPhone(),
		Type:         MsgTypeBookingReminderProvider,
	}
	ctx, err := SaveMsg(ctx, s.getDB(), msg)
	if err != nil {
		return ctx, errors.Wrap(err, "save booking email reminder provider")
	}
	if book.ProviderUser != nil && book.ProviderUser.User != nil {
		msg = &Message{
			SecondaryID:  book.ID,
			FromClientID: book.Client.ID,
			ToUserID:     book.ProviderUser.User.ID,
			ToEmail:      book.ProviderUser.User.GetEmail(),
			ToPhone:      book.ProviderUser.User.GetPhone(),
			Type:         MsgTypeBookingReminderProvider,
		}
		ctx, err = SaveMsg(ctx, s.getDB(), msg)
		if err != nil {
			return ctx, errors.Wrap(err, "save booking email reminder provider")
		}
	}

	//queue the client email
	msg = &Message{
		SecondaryID: book.ID,
		FromUserID:  book.Provider.User.ID,
		ToClientID:  book.Client.ID,
		ToEmail:     book.Client.GetEmail(),
		ToPhone:     book.GetClientPhoneSMS(),
		Type:        MsgTypeBookingReminderClient,
		SenderName:  book.Provider.Name,
	}
	ctx, err = SaveMsg(ctx, s.getDB(), msg)
	if err != nil {
		return ctx, errors.Wrap(err, "save booking email reminder client")
	}
	return ctx, nil
}

//queue a client invite email
func (s *Server) queueEmailClientInvite(ctx context.Context, provider *providerUI, client *Client) (context.Context, error) {
	msg := &Message{
		SecondaryID: client.ID,
		FromUserID:  provider.User.ID,
		ToClientID:  client.ID,
		ToEmail:     client.GetEmail(),
		Type:        MsgTypeClientInvite,
		SenderName:  provider.Name,
	}
	ctx, err := SaveMsg(ctx, s.getDB(), msg)
	if err != nil {
		return ctx, errors.Wrap(err, "save email invite client")
	}
	return ctx, nil
}

//queue a contact email
func (s *Server) queueEmailContact(ctx context.Context, provider *providerUI, client *Client, text string) (context.Context, error) {
	msg := &Message{
		SecondaryID:  client.ID,
		FromClientID: client.ID,
		ToUserID:     provider.User.ID,
		ToEmail:      provider.User.GetEmail(),
		ToPhone:      provider.User.GetPhone(),
		Type:         MsgTypeContact,
		SenderName:   client.Name,
		Text:         text,
	}
	ctx, err := SaveMsg(ctx, s.getDB(), msg)
	if err != nil {
		return ctx, errors.Wrap(err, "save email contact")
	}
	return ctx, nil
}

//queue an invoice email
func (s *Server) queueEmailInvoice(ctx context.Context, sender string, payment *paymentUI) (context.Context, error) {
	msg := &Message{
		SecondaryID: payment.ID,
		ToEmail:     payment.Email,
		ToPhone:     payment.Phone,
		Type:        MsgTypeInvoice,
		SenderName:  sender,
	}
	ctx, err := SaveMsg(ctx, s.getDB(), msg)
	if err != nil {
		return ctx, errors.Wrap(err, "save email invoice")
	}
	return ctx, nil
}

//queue an internal invoice email
func (s *Server) queueEmailInvoiceInternal(ctx context.Context, sender string, payment *paymentUI) (context.Context, error) {
	msg := &Message{
		SecondaryID: payment.ID,
		ToEmail:     payment.Email,
		ToPhone:     payment.Phone,
		Type:        MsgTypeInvoiceInternal,
		SenderName:  sender,
	}
	ctx, err := SaveMsg(ctx, s.getDB(), msg)
	if err != nil {
		return ctx, errors.Wrap(err, "save email invoice internal")
	}
	return ctx, nil
}

//queue payment emails
func (s *Server) queueEmailsPayment(ctx context.Context, provider *providerUI, payment *paymentUI) (context.Context, error) {
	//queue an email to the provider
	msg := &Message{
		SecondaryID: payment.ID,
		ToUserID:    provider.User.ID,
		ToEmail:     provider.User.GetEmail(),
		ToPhone:     provider.User.GetPhone(),
		Type:        MsgTypePaymentProvider,
	}
	ctx, err := SaveMsg(ctx, s.getDB(), msg)
	if err != nil {
		return ctx, errors.Wrap(err, "save email payment provider")
	}

	//queue an email to the client
	msg = &Message{
		SecondaryID: payment.ID,
		FromUserID:  provider.User.ID,
		ToEmail:     payment.Email,
		Type:        MsgTypePaymentClient,
		SenderName:  provider.Name,
	}
	ctx, err = SaveMsg(ctx, s.getDB(), msg)
	if err != nil {
		return ctx, errors.Wrap(err, "save email payment client")
	}
	return ctx, nil
}

//queue campaign add emails
func (s *Server) queueEmailsCampaignAdd(ctx context.Context, provider *providerUI, campaign *campaignUI) (context.Context, error) {
	//queue an email to the provider
	msg := &Message{
		SecondaryID: campaign.ID,
		ToUserID:    provider.User.ID,
		ToEmail:     provider.User.GetEmail(),
		ToPhone:     provider.User.GetPhone(),
		Type:        MsgTypeCampaignAddProvider,
	}
	ctx, err := SaveMsg(ctx, s.getDB(), msg)
	if err != nil {
		return ctx, errors.Wrap(err, "save email campaign add provider")
	}

	//queue an email notification
	emails := GetNotificationEmails()
	tokens := strings.Split(emails, ",")
	for _, token := range tokens {
		msg = &Message{
			SecondaryID: campaign.ID,
			FromUserID:  provider.User.ID,
			ToEmail:     token,
			Type:        MsgTypeCampaignAddNotification,
			SenderName:  provider.Name,
		}
		ctx, err = SaveMsg(ctx, s.getDB(), msg)
		if err != nil {
			return ctx, errors.Wrap(err, fmt.Sprintf("save email campaign add notification: %s", token))
		}
	}
	return ctx, nil
}

//queue campaign payment emails
func (s *Server) queueEmailsCampaignPayment(ctx context.Context, provider *providerUI, campaign *campaignUI) (context.Context, error) {
	//queue an email notification
	emails := GetNotificationEmails()
	tokens := strings.Split(emails, ",")
	for _, token := range tokens {
		msg := &Message{
			SecondaryID: campaign.ID,
			FromUserID:  provider.User.ID,
			ToEmail:     token,
			Type:        MsgTypeCampaignPaymentNotification,
			SenderName:  provider.Name,
		}
		ctx, err := SaveMsg(ctx, s.getDB(), msg)
		if err != nil {
			return ctx, errors.Wrap(err, fmt.Sprintf("save email campaign payment notification: %s", token))
		}
	}
	return ctx, nil
}

//queue campaign status emails
func (s *Server) queueEmailsCampaignStatus(ctx context.Context, campaign *campaignUI) (context.Context, error) {
	//load the provider
	ctx, provider, err := LoadProviderByID(ctx, s.getDB(), campaign.ProviderID)
	if err != nil {
		return ctx, errors.Wrap(err, fmt.Sprintf("load provider: %s", campaign.ProviderID))
	}

	//queue an email to the provider
	msg := &Message{
		SecondaryID: campaign.ID,
		ToUserID:    provider.User.ID,
		ToEmail:     provider.User.GetEmail(),
		ToPhone:     provider.User.GetPhone(),
		Type:        MsgTypeCampaignStatusProvider,
	}
	ctx, err = SaveMsg(ctx, s.getDB(), msg)
	if err != nil {
		return ctx, errors.Wrap(err, "save email campaign status provider")
	}
	return ctx, nil
}

//queue domain email
func (s *Server) queueEmailsDomain(ctx context.Context, provider *providerUI) (context.Context, error) {
	emails := GetNotificationEmails()
	tokens := strings.Split(emails, ",")
	for _, token := range tokens {
		msg := &Message{
			SecondaryID: provider.ID,
			FromUserID:  provider.User.ID,
			ToEmail:     token,
			Type:        MsgTypeDomainNotification,
			SenderName:  provider.Name,
		}
		ctx, err := SaveMsg(ctx, s.getDB(), msg)
		if err != nil {
			return ctx, errors.Wrap(err, fmt.Sprintf("save email domain notification: %s", token))
		}
	}
	return ctx, nil
}

//queue a password reset email
func (s *Server) queueEmailPwdReset(ctx context.Context, user *User, token string) (context.Context, error) {
	//create the reset url
	tokenURL, err := CreateURLRelParams(URIPwdReset, URLParams.Token, token)
	if err != nil {
		return ctx, errors.Wrap(err, "token url")
	}

	//queue the email
	msg := &Message{
		SecondaryID: user.ID,
		ToUserID:    user.ID,
		ToEmail:     user.GetEmail(),
		Type:        MsgTypePwdReset,
		TokenURL:    tokenURL,
	}
	ctx, err = SaveMsg(ctx, s.getDB(), msg)
	if err != nil {
		return ctx, errors.Wrap(err, "save email password reset")
	}
	return ctx, nil
}

//queue a provider user invite
func (s *Server) queueEmailProviderUserInvite(ctx context.Context, provider *providerUI, user *ProviderUser) (context.Context, error) {
	msg := &Message{
		SecondaryID: provider.ID,
		ToEmail:     user.Login,
		Type:        MsgTypeProviderUserInvite,
	}
	ctx, err := SaveMsg(ctx, s.getDB(), msg)
	if err != nil {
		return ctx, errors.Wrap(err, "save email provider user invite")
	}
	return ctx, nil
}

//queue an email request to verify the email
func (s *Server) queueEmailVerify(ctx context.Context, user *User) (context.Context, error) {
	//create a verify token
	token, err := CreateEmailVerifyToken()
	if err != nil {
		return ctx, errors.Wrap(err, "email verify token")
	}

	//compute the expiration from now and store the token for verification
	expiration := time.Now().Unix() + int64(verifyEmailTokenExpiration.Seconds())
	ctx, err = SaveEmailVerifyToken(ctx, s.getDB(), user.ID, token, expiration)

	//create the verify token
	tokenURL, err := CreateURLRelParams(URIEmailVerify, URLParams.Token, token)
	if err != nil {
		return ctx, errors.Wrap(err, "token url")
	}

	//queue the email
	msg := &Message{
		SecondaryID: user.ID,
		ToUserID:    user.ID,
		ToEmail:     user.GetEmail(),
		Type:        MsgTypeEmailVerify,
		TokenURL:    tokenURL,
	}
	ctx, err = SaveMsg(ctx, s.getDB(), msg)
	if err != nil {
		return ctx, errors.Wrap(err, "save email verify")
	}
	return ctx, nil
}

//queue a welcome email
func (s *Server) queueEmailWelcome(ctx context.Context, provider *providerUI) (context.Context, error) {
	msg := &Message{
		SecondaryID: provider.ID,
		ToUserID:    provider.User.ID,
		ToEmail:     provider.User.GetEmail(),
		Type:        MsgTypeWelcome,
	}
	ctx, err := SaveMsg(ctx, s.getDB(), msg)
	if err != nil {
		return ctx, errors.Wrap(err, "save email welcome")
	}
	return ctx, nil
}

//save a user and refresh the authentication token
func (s *Server) saveUser(w http.ResponseWriter, r *http.Request, user *User, pwd Secret) (context.Context, error) {
	ctx, _ := GetLogger(s.getCtx(r))
	ctx, err := SaveUser(ctx, s.getDB(), user, pwd)
	if err != nil {
		return ctx, errors.Wrap(err, "save user")
	}

	//refresh the token and store in the cookie
	_, err = s.refreshToken(w, r.WithContext(ctx), user.ID)
	if err != nil {
		return ctx, errors.Wrap(err, "refresh token")
	}

	//update the last login
	ctx, err = UpdateUserLastLogin(ctx, s.getDB(), user.ID)
	if err != nil {
		return ctx, errors.Wrap(err, fmt.Sprintf("update last login: %s", user.ID))
	}
	return ctx, nil
}

//save a booking
func (s *Server) saveBooking(w http.ResponseWriter, r *http.Request, tpl *template.Template, data templateData, errs map[string]string, provider *providerUI, providerUser *ProviderUser, svc *serviceUI, bookUI *bookingUI, now time.Time, changeAllFollowing bool, form *ClientBookingForm, isClient bool) (*bookingUI, bool) {
	ctx, logger := GetLogger(s.getCtx(r))
	timeFrom := ParseTimeUnixLocal(form.Time, form.TimeZone)
	if timeFrom.IsZero() {
		errs[string(FieldErrTime)] = GetFieldErrText(string(FieldErrTime))
		s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
		return nil, false
	}
	timeTo := svc.ComputeTimeTo(timeFrom)

	//check that the time honors the minimum start time
	if isClient {
		check := svc.CheckValidTime(provider.Provider, now, timeFrom, timeTo)
		if !check {
			errs[string(FieldErrTime)] = GetFieldErrText(string(FieldErrTime))
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return nil, false
		}
	}

	//probe for a client id
	var clientID *uuid.UUID
	if form.Email == "" && form.Name == "" && form.ClientID != "" {
		id := uuid.FromStringOrNil(form.ClientID)
		if id == uuid.Nil {
			logger.Warnw("invalid uuid", "id", form.ClientID)
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return nil, false
		}
		clientID = &id
	}

	//create the booking if necessary
	createBook := false
	var book *Booking
	if bookUI == nil {
		createBook = true
		book = &Booking{
			Provider:  provider.Provider,
			Confirmed: form.Confirmed,
			Client: &Client{
				ID:       clientID,
				Email:    form.Email,
				Location: form.Location,
				Name:     form.Name,
				Phone:    form.Phone,
				TimeZone: form.TimeZone,
			},
			EnableClientPhone: form.EnablePhone,
		}

		//assign an id
		id, err := uuid.NewV4()
		if err != nil {
			logger.Errorw("create booking id", "error", err)
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return nil, false
		}
		book.ID = &id
		bookUI = s.createBookingUI(book)
	} else {
		book = bookUI.Booking
	}
	book.SetCouponCode(form.Code)
	book.SetProvider(provider.Provider)
	book.SetService(svc.Service)
	book.SetLocation(form.Location)
	book.SetTimeFrom(timeFrom)
	if form.DescriptionSet {
		book.SetDescription(form.Description)
	}
	if form.ProviderNoteSet {
		book.SetProviderNote(form.ProviderNote)
	}

	//check if the user should be overridden
	if provider.IsAdmin() || isClient {
		book.SetProviderUser(providerUser)
	} else {
		book.SetProviderUser(provider.ProviderUser)
	}
	user := book.GetUser()

	//sanity check the date
	if book.IsApptOnly() {
		if book.TimeChange {
			//check if the time is available
			ctx, count, err := CountBookingsForProviderAndTime(ctx, s.getDB(), provider.ID, user, book.TimeFrom, book.TimeTo, ServiceTypeAppt)
			if err != nil {
				logger.Errorw("count bookings", "error", err, "id", provider.ID, "from", book.TimeFrom, "to", book.TimeTo)
				data[TplParamErr] = GetErrText(Err)
				s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
				return nil, false
			}
			if count > 0 {
				data[TplParamErr] = GetErrText(ErrBookingTime)
				s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
				return nil, false
			}
		}
	} else {
		book.SetTimeTo(book.TimeFrom)
	}

	//process the recurrence frequency
	if form.FreqSet {
		recurrenceFreq := ParseRecurrenceFreq(&form.Freq)
		err := book.SetRecurrenceFreq(recurrenceFreq, false)
		if err != nil {
			logger.Errorw("set recurrence freq", "error", err, "freq", recurrenceFreq)
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return nil, false
		}
	}

	//create a zoom meeting
	if createBook && user.ZoomToken != nil && svc.EnableZoom && svc.IsApptOnly() {
		ctx, token, meeting, err := s.createMeetingZoom(ctx, bookUI)
		if err != nil {
			logger.Errorw("create zoom meeting", "error", err)
			data[TplParamErr] = GetErrText(Err)
			s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
			return nil, false
		}
		meetingID := strconv.Itoa(meeting.ID)
		book.MeetingZoomID = &meetingID
		book.MeetingZoomData = meeting
		book.MeetingZoomUpdate = true

		//update the token if necessary
		if token != nil {
			user.ZoomToken = token
			ctx, err = UpdateUserTokenZoom(ctx, s.getDB(), user)
			if err != nil {
				logger.Errorw("update zoom token", "error", err)
				data[TplParamErr] = GetErrText(Err)
				s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
				return nil, false
			}
		}
	}

	//update the booking, forcing a change-all if the recurrence frequency has changed
	changeAllFollowing = changeAllFollowing || book.RecurrenceFreqChange
	bookUI, err := s.updateServiceBooking(ctx, provider, svc, book, now, changeAllFollowing, form.Confirmed, form.ClientCreated, false)
	if err != nil {
		logger.Errorw("update service booking", "error", err)
		data[TplParamErr] = GetErrText(Err)
		s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
		return nil, false
	}
	data[TplParamBook] = bookUI
	data[TplParamSvcTime] = bookUI.FormatDateTime(form.TimeZone)

	//queue emails, checking if creating a new booking
	if bookUI.TimeFrom.After(now) {
		if createBook {
			ctx, err := s.queueEmailsBookingNew(ctx, provider, svc, bookUI, form.ClientCreated)
			if err != nil {
				logger.Errorw("queue email booking new", "error", err)
				data[TplParamErr] = GetErrText(Err)
				s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
				return nil, false
			}
		} else {
			ctx, err := s.queueEmailsBookingEdit(ctx, provider, svc, bookUI, form.ClientCreated)
			if err != nil {
				logger.Errorw("queue email booking edit", "error", err)
				data[TplParamErr] = GetErrText(Err)
				s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
				return nil, false
			}
		}
	}
	return bookUI, true
}

//cancel a booking
func (s *Server) cancelServiceBooking(w http.ResponseWriter, r *http.Request, tpl *template.Template, data templateData, errs map[string]string, provider *providerUI, svc *serviceUI, bookUI *bookingUI, now time.Time, changeAllFollowing bool) bool {
	ctx, logger := GetLogger(s.getCtx(r))

	//update the booking
	bookUI, err := s.updateServiceBooking(ctx, provider, svc, bookUI.Booking, now, changeAllFollowing, bookUI.Confirmed, bookUI.ClientCreated, true)
	if err != nil {
		logger.Errorw("update service booking", "error", err)
		data[TplParamErr] = GetErrText(Err)
		s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
		return false
	}

	//queue the emails
	ctx, err = s.queueEmailsBookingCancel(ctx, provider, svc, bookUI, bookUI.ClientCreated)
	if err != nil {
		logger.Errorw("queue email booking cancel", "error", err)
		data[TplParamErr] = GetErrText(Err)
		s.renderWebTemplate(w, r.WithContext(ctx), tpl, data)
		return false
	}
	return true
}

//save a booking
func (s *Server) updateServiceBooking(ctx context.Context, provider *providerUI, svc *serviceUI, book *Booking, now time.Time, changeAllFollowing bool, confirmed bool, isClient bool, cancel bool) (*bookingUI, error) {
	//check for a coupon
	if !cancel && book.CouponCodeChange {
		_, coupon, err := LoadCouponByProviderIDAndCode(ctx, s.getDB(), provider.ID, book.CouponCode, &now)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("load coupon: %s", book.CouponCode))
		}
		book.Coupon = coupon
	}

	//save the booking
	ctx, err := SaveBooking(ctx, s.getDB(), provider.Provider, svc.Service, book, now, changeAllFollowing, confirmed, isClient, cancel)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("save booking: %s", book.ID))
	}
	bookUI := s.createBookingUI(book)
	return bookUI, nil
}

//redirect based on the host as necessary
func (s *Server) redirectHost(w http.ResponseWriter, r *http.Request, host string, port string) bool {
	ctx, logger := GetLogger(r.Context())
	host = strings.ToLower(host)
	if host != GetDomain() {
		//explicitly redirect from the root domain to the standard domain
		if host == GetDomainRoot() {
			url := *r.URL
			host := GetDomain()
			if port != "" {
				host = fmt.Sprintf("%s:%s", host, port)
			}
			url.Host = host
			http.Redirect(w, r, url.String(), http.StatusMovedPermanently)
			return false
		}

		//ignore ip addresses
		addr := net.ParseIP(host)
		if addr == nil {
			//redirect to the provider if the base url
			if r.Method == http.MethodGet && r.URL.Path == "/" && r.URL.RawQuery == "" {
				//special case for dealing w/ www and a root domain
				hostRoot := ""
				if strings.HasPrefix(host, "www.") && strings.Count(host, ".") == 2 {
					hostRoot = strings.TrimPrefix(host, "www.")
				}
				ctx, provider, err := LoadProviderByDomain(ctx, s.getDB(), host, hostRoot)
				if err != nil {
					logger.Errorw("load provider", "error", err, "domain", host)
					s.invokeHdlrGet(s.handleProviderErr404(), w, r.WithContext(ctx))
					return false
				}
				providerUI := s.createProviderUI(provider)
				http.Redirect(w, r, providerUI.GetURLProvider(), http.StatusTemporaryRedirect)
				return false
			}
			//store the host
			s.SetCookieHost(w, r.Host)
		}
	} else {
		s.DeleteCookieHost(w)
	}
	return true
}

//determine the redirect, using a custom host if set
func (s *Server) redirectAbs(w http.ResponseWriter, r *http.Request, uri string, params ...string) {
	ctx, logger := GetLogger(r.Context())

	//force an absolute url and allow a host override
	uri, err := CreateURLAbsParams(ctx, uri, params...)
	if err != nil {
		logger.Warnw("create url host", "error", err, "url", uri)
		http.Redirect(w, r.WithContext(ctx), URIErr, http.StatusSeeOther)
		return
	}
	http.Redirect(w, r.WithContext(ctx), uri, http.StatusSeeOther)
}

//determine the redirect after login
func (s *Server) redirectLogin(w http.ResponseWriter, r *http.Request, id *uuid.UUID, token string) (context.Context, error) {
	ctx, _ := GetLogger(r.Context())

	//update the last login
	ctx, err := UpdateUserLastLogin(ctx, s.getDB(), id)
	if err != nil {
		return ctx, errors.Wrap(err, fmt.Sprintf("update last login: %s", id))
	}

	//check for a request uri in the cookie to use for the redirect
	provider := &providerUI{}
	uri := provider.GetURLBookings()
	requestURI, err := s.GetCookieRequestURI(r.WithContext(ctx))
	if err != nil {
		return ctx, errors.Wrap(err, "cookie request uri")
	}
	if requestURI != "" {
		uri = requestURI
	}
	s.DeleteCookieRequestURI(w)

	//probe for a host override and pass the token
	var params []string
	host := GetCtxCustomHost(ctx)
	if host != "" && token != "" {
		params = []string{URLParams.AuthToken, token}
	}
	s.redirectAbs(w, r.WithContext(ctx), uri, params...)
	return ctx, nil
}

//redirect when an error occurs
func (s *Server) redirectError(w http.ResponseWriter, r *http.Request, key ErrKey, args ...interface{}) {
	s.SetCookieErr(w, key, args...)
	http.Redirect(w, r, URIErr, http.StatusSeeOther)
	return
}

//create a provider google calendar
func (s *Server) createProviderCalendarGoogle(ctx context.Context, provider *Provider) (*CalendarGoogle, error) {
	title := provider.GetCalendarTitle()
	data, err := CreateCalendarGoogle(ctx, title)
	if err != nil {
		return nil, errors.Wrap(err, "create google calendar")
	}
	return data, nil
}

//update a provider google calendar
func (s *Server) updateProviderCalendarGoogle(ctx context.Context, provider *Provider) (*CalendarGoogle, error) {
	data, err := UpdateCalendarGoogle(ctx, provider.GoogleCalendarID, provider.Name)
	if err != nil {
		return nil, errors.Wrap(err, "update google calendar")
	}
	return data, nil
}

//create a booking google calendar event
func (s *Server) createEventGoogle(ctx context.Context, book *bookingUI) (context.Context, *EventGoogle, error) {
	title := book.GetEventTitle()
	ctx, desc, err := book.GetEventDescription(ctx, true, true)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "google calendar event description")
	}
	data, err := CreateEventGoogle(ctx, book.Provider.GoogleCalendarID, book.ID.String(), book.TimeFrom, book.TimeTo, title, desc, book.Location, book.RecurrenceRules)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "create google calendar event")
	}
	return ctx, data, nil
}

//update a booking google calendar event
func (s *Server) updateEventGoogle(ctx context.Context, book *bookingUI) (context.Context, *EventGoogle, error) {
	title := book.GetEventTitle()
	ctx, desc, err := book.GetEventDescription(ctx, true, true)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "update calendar event description")
	}
	data, err := UpdateEventGoogle(ctx, book.Provider.GoogleCalendarID, book.EventGoogleID, book.ID.String(), book.TimeFrom, book.TimeTo, title, desc, book.Location, book.RecurrenceRules)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "update google calendar event")
	}
	return ctx, data, nil
}

//cancel a booking google calendar event
func (s *Server) cancelEventGoogle(ctx context.Context, book *bookingUI) (*EventGoogle, error) {
	data, err := CancelEventGoogle(ctx, book.Provider.GoogleCalendarID, book.EventGoogleID, book.TimeFrom, book.TimeTo)
	if err != nil {
		return nil, errors.Wrap(err, "cancel google calendar event")
	}
	return data, nil
}

//create a booking zoom meeting
func (s *Server) createMeetingZoom(ctx context.Context, book *bookingUI) (context.Context, *TokenZoom, *MeetingZoom, error) {
	title := book.GetEventTitle()
	ctx, desc, err := book.GetEventDescription(ctx, false, false)
	if err != nil {
		return ctx, nil, nil, errors.Wrap(err, "create zoom meeting description")
	}
	user := book.GetUser()
	token, data, err := CreateMeetingZoom(ctx, user.ZoomToken, book.ID.String(), title, desc, book.TimeFrom, user.TimeZone, book.ServiceDuration, RecurrenceIntervalOnce)
	if err != nil {
		return ctx, nil, nil, errors.Wrap(err, "create zoom meeting")
	}
	return ctx, token, data, nil
}

//update a booking zoom meeting
func (s *Server) updateMeetingZoom(ctx context.Context, book *bookingUI) (context.Context, *TokenZoom, *MeetingZoom, error) {
	title := book.GetEventTitle()
	ctx, desc, err := book.GetEventDescription(ctx, false, false)
	if err != nil {
		return ctx, nil, nil, errors.Wrap(err, "update zoom meeting description")
	}
	user := book.GetUser()
	token, data, err := UpdateMeetingZoom(ctx, user.ZoomToken, book.MeetingZoomID, book.ID.String(), title, desc, book.TimeFrom, user.TimeZone, book.ServiceDuration, RecurrenceIntervalOnce)
	if err != nil {
		return ctx, nil, nil, errors.Wrap(err, "update zoom meeting")
	}
	return ctx, token, data, nil
}

//cancel a booking zoom meeting
func (s *Server) cancelBookingMeetingZoom(ctx context.Context, book *bookingUI) (*TokenZoom, error) {
	user := book.GetUser()
	data, err := DeleteMeetingZoom(ctx, user.ZoomToken, book.MeetingZoomID)
	if err != nil {
		return nil, errors.Wrap(err, "delete zoom meeting")
	}
	return data, nil
}

//add the client view flag if appropriate
func (s *Server) checkClientView(data templateData, provider *providerUI, url string) string {
	view, ok := data[TplParamClientView].(string)
	if ok && len(view) > 0 {
		url = provider.MarkURLClient(url)
	}
	return url
}

//create a shortened URL to use in SMS
func (s *Server) createShortURL(ctx context.Context, url string) (string, error) {
	var err error
	url, err = CreateURLAbs(ctx, url, nil)
	if err != nil {
		return "", errors.Wrap(err, fmt.Sprintf("create url"))
	}
	ctx, shortURL, err := SaveURL(ctx, s.getDB(), url)
	if err != nil {
		return "", errors.Wrap(err, "save url")
	}

	//create the full url
	shortURL = createShortURL(shortURL)
	shortURL, err = CreateURLAbs(ctx, shortURL, nil)
	if err != nil {
		return "", errors.Wrap(err, fmt.Sprintf("create url"))
	}
	return shortURL, nil
}

//create and save a booking payment
func (s *Server) savePaymentBooking(ctx context.Context, provider *providerUI, book *bookingUI, form *PaymentForm, now time.Time) (context.Context, *Payment, error) {
	payment := &Payment{
		Description:     book.FormatServicePaymentDescription(provider.User.TimeZone),
		Email:           form.Email,
		Name:            form.Name,
		Note:            form.Description,
		Phone:           form.Phone,
		ProviderID:      provider.ID,
		ProviderName:    provider.Name,
		SecondaryID:     book.ID,
		Type:            PaymentTypeBooking,
		URL:             book.GetURLPaymentClient(),
		ClientInitiated: form.ClientInitiated,
		DirectCapture:   form.DirectCapture,
		Invoiced:        &now,
	}
	price, _ := strconv.ParseFloat(form.Price, 32)
	payment.SetAmount(float32(price))

	//mark paid if necessary
	if form.DirectCapture {
		payment.Paid = &now
		payment.Captured = &now
	}

	//save the payment
	ctx, err := SavePayment(ctx, s.getDB(), payment, payment.DirectCapture, true)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "save payment")
	}
	return ctx, payment, nil
}

//create and save a campaign payment
func (s *Server) savePaymentCampaign(ctx context.Context, campaign *campaignUI, form *CampaignPaymentForm, now *time.Time) (context.Context, *Payment, error) {
	//load the provider
	ctx, provider, err := LoadProviderByID(ctx, s.getDB(), campaign.ProviderID)
	if err != nil {
		return ctx, nil, errors.Wrap(err, fmt.Sprintf("load provider: %s", campaign.ProviderID))
	}

	//create the payment id
	paymentID, err := uuid.NewV4()
	if err != nil {
		return ctx, nil, errors.Wrap(err, "new uuid")
	}

	//set-up the payment
	payment := &Payment{
		ID:           &paymentID,
		Description:  campaign.FormatPaymentDescription(provider.User.TimeZone),
		Email:        provider.User.Email,
		Name:         provider.User.FormatName(),
		Note:         form.Description,
		ProviderID:   provider.ID,
		ProviderName: provider.Name,
		SecondaryID:  campaign.ID,
		Type:         PaymentTypeCampaign,
		URL:          campaign.GetURLPayment(&paymentID),
		Internal:     true,
		Invoiced:     now,
	}
	price, _ := strconv.ParseFloat(form.Price, 32)
	payment.SetAmount(float32(price))

	//save the payment
	ctx, err = SavePayment(ctx, s.getDB(), payment, false, false)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "save payment")
	}
	return ctx, payment, nil
}

//create and save a direct payment
func (s *Server) savePaymentDirect(ctx context.Context, provider *providerUI, svc *Service, form *PaymentForm, now time.Time, timeZone string) (context.Context, *Payment, error) {
	//save the client
	client := &Client{
		ProviderID: provider.ID,
		Email:      form.Email,
		Name:       form.Name,
		Phone:      form.Phone,
		TimeZone:   timeZone,
	}

	//save the client
	ctx, err := SaveClient(ctx, s.getDB(), client)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "save client")
	}

	//generate the payment id
	id, err := uuid.NewV4()
	if err != nil {
		return ctx, nil, errors.Wrap(err, "new uuid payment")
	}

	//save the payment
	payment := &Payment{
		ID:              &id,
		Description:     "direct payment",
		Email:           form.Email,
		Name:            form.Name,
		Note:            form.Description,
		Phone:           form.Phone,
		ProviderID:      provider.ID,
		ProviderName:    provider.Name,
		SecondaryID:     client.ID,
		Type:            PaymentTypeDirect,
		URL:             createProviderPaymentURL(provider.GetURLName(), &id),
		ClientInitiated: form.ClientInitiated,
		DirectCapture:   form.DirectCapture,
		Invoiced:        &now,
	}
	price, _ := strconv.ParseFloat(form.Price, 32)
	payment.SetAmount(float32(price))

	//apply service information
	if svc != nil {
		payment.Description = fmt.Sprintf("%s for %s", payment.Description, svc.Name)
		payment.ServiceID = svc.ID.String()
	}

	//mark paid if necessary
	if form.DirectCapture {
		payment.Paid = &now
		payment.Captured = &now
	}

	//save the payment
	ctx, err = SavePayment(ctx, s.getDB(), payment, payment.DirectCapture, false)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "save payment")
	}
	return ctx, payment, nil
}

//create the payment data for paypal
func (s *Server) createPaymentPayPal(ctx context.Context, payeeEmail *string, payment *Payment) error {
	//create a paypal order
	order, err := CreateOrderPayPal(ctx, payeeEmail, payment.ProviderName, payment.Description, payment.ID.String(), payment.SecondaryID.String(), payment.GetAmount())
	if err != nil {
		return errors.Wrap(err, "create paypal order")
	}

	//save the payment data
	orderData, err := json.Marshal(order)
	if err != nil {
		return errors.Wrap(err, "json paypal order")
	}
	orderJSON := string(orderData)
	payment.PayPalID = &order.ID
	ctx, err = UpdatePaymentPayPalID(ctx, s.getDB(), payment.ID, payment.PayPalID, &orderJSON)
	if err != nil {
		return errors.Wrap(err, "save paypal order")
	}
	return nil
}

//create the payment data for stripe
func (s *Server) createPaymentStripe(ctx context.Context, token *TokenStripe, payment *Payment) error {
	//create a stripe session
	session, err := CreateSessionStripe(ctx, token, payment.ProviderName, payment.Description, payment.ID.String(), payment.SecondaryID.String(), payment.Amount, payment.URL)
	if err != nil {
		return errors.Wrap(err, "create stripe session")
	}
	sessionID := session.ID

	//retrieve the user id for forwarded payments
	var stripeAccountID string
	if token != nil {
		stripeAccountID, err = token.GetStripeUserID()
		if err != nil {
			return errors.Wrap(err, "get stripe account id")
		}
	}

	//save the payment data
	sessionData, err := json.Marshal(session)
	if err != nil {
		return errors.Wrap(err, "json stripe session")
	}
	sessionJSON := string(sessionData)
	payment.StripeAccountID = &stripeAccountID
	payment.StripeSessionID = &sessionID
	payment.StripeID = &session.PaymentIntent.ID
	ctx, err = UpdatePaymentStripeID(ctx, s.getDB(), payment.ID, payment.StripeID, payment.StripeSessionID, payment.StripeAccountID, &sessionJSON)
	if err != nil {
		return errors.Wrap(err, "save stripe session")
	}
	return nil
}

//create the payment for stripe ach
func (s *Server) createPaymentStripeACH(ctx context.Context, token *TokenStripe, payment *Payment, plaidData string) error {
	//create the payment
	linkData, err := ParsePlaidLinkData(plaidData)
	if err != nil {
		return errors.Wrap(err, "plaid link data")
	}
	exchangeData, err := ExchangePlaidToken(ctx, linkData.PublicToken)
	if err != nil {
		return errors.Wrap(err, "plaid exchange token")
	}
	stripeData, err := CreatePlaidStripeToken(ctx, exchangeData.AccessToken, linkData.AccountID)
	if err != nil {
		return errors.Wrap(err, "plaid stripe token")
	}
	chargeData, err := CreateStripeCharge(ctx, token, payment.ProviderName, payment.Description, payment.ID.String(), payment.Amount, stripeData.StripeBankAccountToken)
	if err != nil {
		return errors.Wrap(err, "stripe charge")
	}

	//store the data
	data := map[string]interface{}{
		"Link":     linkData,
		"Exchange": exchangeData,
		"Stripe":   stripeData,
		"Charge":   chargeData,
	}
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return errors.Wrap(err, "json stripe session")
	}
	dataStr := string(dataJSON)
	payment.StripeAccountID = &linkData.AccountID
	payment.StripeSessionID = &stripeData.StripeBankAccountToken
	payment.StripeID = &chargeData.ID
	ctx, err = UpdatePaymentStripeID(ctx, s.getDB(), payment.ID, payment.StripeID, payment.StripeSessionID, payment.StripeAccountID, &dataStr)
	if err != nil {
		return errors.Wrap(err, "save stripe session")
	}
	return nil
}

//check the permissions
func (s *Server) checkPermission(w http.ResponseWriter, r *http.Request, provider *providerUI, requiresAdmin bool) bool {
	if requiresAdmin {
		if !provider.IsAdmin() {
			w.WriteHeader(http.StatusUnauthorized)
			return false
		}
	}
	return true
}

//check the owner of an order
func (s *Server) checkOwner(w http.ResponseWriter, r *http.Request, provider *providerUI, book *bookingUI) bool {
	if provider.IsAdmin() {
		return true
	}
	if book.ProviderUser == nil {
		return true
	}
	return book.ProviderUserID.String() == provider.ProviderUser.ID.String()
}
