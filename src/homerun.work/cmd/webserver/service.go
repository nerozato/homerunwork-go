package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

//service db tables
const (
	dbTableService             = "service"
	dbTableServiceProviderUser = "service_provider_user"
)

//service constants
const (
	ServiceDurationDefault = 60 //minutes
	ServiceIntervalDefault = 15 //minutes
)

//order service duration
type serviceDuration struct {
	Label    string
	Value    int
	ValueStr string
}

//IsDurationVariable : check if a duration is variable
func (s *serviceDuration) IsDurationVariable(duration int) bool {
	return duration == ServiceDurationVariable.Value
}

//ServiceDurationVariable : definition of a variable duration
var ServiceDurationVariable serviceDuration = serviceDuration{
	Label:    "Variable",
	Value:    0,
	ValueStr: "0",
}

//ServiceDurationsBooking : booking service durations
var ServiceDurationsBooking []serviceDuration = []serviceDuration{
	{
		Label:    "30 mins",
		Value:    30,
		ValueStr: "30",
	},
	{
		Label:    "45 mins",
		Value:    45,
		ValueStr: "45",
	},
	{
		Label:    "60 mins",
		Value:    60,
		ValueStr: "60",
	},
	{
		Label:    "90 mins",
		Value:    90,
		ValueStr: "90",
	},
	{
		Label:    "120 mins",
		Value:    120,
		ValueStr: "120",
	},
	{
		Label:    "3 hrs.",
		Value:    180,
		ValueStr: "180",
	},
	{
		Label:    "4 hrs.",
		Value:    240,
		ValueStr: "240",
	},
	{
		Label:    "5 hrs.",
		Value:    300,
		ValueStr: "300",
	},
	{
		Label:    "6 hrs.",
		Value:    360,
		ValueStr: "360",
	},
	{
		Label:    "8 hrs.",
		Value:    480,
		ValueStr: "480",
	},
}

//ServiceDurationsOrder : order service durations
var ServiceDurationsOrder []serviceDuration = append(ServiceDurationsBooking, []serviceDuration{
	{
		Label:    "2 days", //2 work days
		Value:    960,
		ValueStr: "960",
	},
	{
		Label:    "3 days", //3 work days
		Value:    1440,
		ValueStr: "1440",
	},
	{
		Label:    "4 days", //4 work days
		Value:    1920,
		ValueStr: "1920",
	},
	{
		Label:    "5 days", //5 work days
		Value:    2400,
		ValueStr: "2400",
	},
	{
		Label:    "6 days", //6 work days
		Value:    2880,
		ValueStr: "2880",
	},
	{
		Label:    "7 days", //7 work days
		Value:    3360,
		ValueStr: "3360",
	},
	ServiceDurationVariable,
}...)

//ServiceType : type of service
type ServiceType int

//service types
const (
	ServiceTypeAppt ServiceType = iota + 1
	ServiceTypeOnDemand
)

//IsApptOnly : check if the service type is appointment only
func IsApptOnly(serviceType ServiceType) bool {
	return serviceType == ServiceTypeAppt
}

//ServiceInterval : definition of a service interval
type ServiceInterval struct {
	Label    string
	Value    int
	ValueStr string
}

//ServiceIntervals : service intervals
var ServiceIntervals []ServiceInterval = []ServiceInterval{
	{
		Label:    "15 mins",
		Value:    15,
		ValueStr: "15",
	},
	{
		Label:    "30 mins",
		Value:    30,
		ValueStr: "30",
	},
	{
		Label:    "60 mins",
		Value:    60,
		ValueStr: "60",
	},
}

//ParseServiceInterval : parse a service interval by label
func ParseServiceInterval(intervalStr string) *ServiceInterval {
	if intervalStr == "" {
		return nil
	}
	for _, serviceInterval := range ServiceIntervals {
		if intervalStr == serviceInterval.ValueStr {
			return &serviceInterval
		}
	}
	return nil
}

//ServiceLocationType : type of service location
type ServiceLocationType string

//IsLocationProvider : check if a provider location
func (s *ServiceLocationType) IsLocationProvider() bool {
	return *s == ServiceLocationTypeProvider
}

//IsLocationClient : check if a client location
func (s *ServiceLocationType) IsLocationClient() bool {
	return *s == ServiceLocationTypeClient
}

//HasLocation : check if a location is present
func (s *ServiceLocationType) HasLocation() bool {
	return s.IsLocationProvider() || s.IsLocationClient()
}

//recurrence intervals
const (
	ServiceLocationTypeFlexible ServiceLocationType = "flexible"
	ServiceLocationTypeClient                       = "client"
	ServiceLocationTypeProvider                     = "provider"
	ServiceLocationTypeRemote                       = "remote"
)

//ServiceLocation : definition of a service location
type ServiceLocation struct {
	Label string
	Type  ServiceLocationType
}

//service locations
var (
	ServiceLocationFlexible = ServiceLocation{
		Label: "Flexible",
		Type:  ServiceLocationTypeFlexible,
	}
	ServiceLocationClient = ServiceLocation{
		Label: "Client's Location",
		Type:  ServiceLocationTypeClient,
	}
	ServiceLocationProvider = ServiceLocation{
		Label: "My Location",
		Type:  ServiceLocationTypeProvider,
	}
	ServiceLocationRemote = ServiceLocation{
		Label: "Remote",
		Type:  ServiceLocationTypeRemote,
	}
)

//ServiceLocations : service locations
var ServiceLocations []ServiceLocation = []ServiceLocation{
	ServiceLocationRemote,
	ServiceLocationProvider,
	ServiceLocationClient,
}

//ParseServiceLocation : parse a service location by label
func ParseServiceLocation(locationTypeStr string) *ServiceLocation {
	if locationTypeStr == "" {
		return nil
	}
	for _, serviceLocation := range ServiceLocations {
		if locationTypeStr == string(serviceLocation.Type) {
			return &serviceLocation
		}
	}
	return nil
}

//PaddingUnit : unit for padding
type PaddingUnit string

//padding units
const (
	PaddingUnitHours PaddingUnit = "Hours"
	PaddingUnitDays              = "Days"
)

//PaddingUnits : padding units
var PaddingUnits []PaddingUnit = []PaddingUnit{
	PaddingUnitHours,
	PaddingUnitDays,
}

//ParsePaddingUnit : parse a padding unit
func ParsePaddingUnit(in string) PaddingUnit {
	if in == "" {
		return ""
	}
	switch in {
	case string(PaddingUnitHours):
		return PaddingUnitHours
	case string(PaddingUnitDays):
		return PaddingUnitDays
	}
	return ""
}

//PriceType : type of pricing
type PriceType string

//Format : format a price
func (p *PriceType) Format(price float32) string {
	var priceStr string
	if price == 0 {
		priceStr = "FREE"
		return priceStr
	}
	priceStr = FormatPrice(price)
	if *p == PriceTypeHourly {
		priceStr = fmt.Sprintf("%s/hour", priceStr)
	}
	return priceStr
}

//Compute : compute a price
func (p *PriceType) Compute(price float32, durationMin int) float32 {
	if *p == PriceTypeFixed {
		return price
	}

	//compute hours
	hours := float32(durationMin) / 60
	if hours == 0 {
		hours = 1
	}
	return price * hours
}

//price type
const (
	PriceTypeFixed  PriceType = "Fixed"
	PriceTypeHourly           = "Hourly"
)

//PriceTypes : prices types
var PriceTypes []PriceType = []PriceType{
	PriceTypeFixed,
	PriceTypeHourly,
}

//ParsePriceType : parse a price type
func ParsePriceType(in string) PriceType {
	if in == "" {
		return ""
	}
	switch in {
	case string(PriceTypeFixed):
		return PriceTypeFixed
	case string(PriceTypeHourly):
		return PriceTypeHourly
	}
	return ""
}

//Service : service definition
type Service struct {
	ID                 *uuid.UUID          `json:"-"`
	UserID             *uuid.UUID          `json:"-"`
	Type               ServiceType         `json:"-"`
	ImgMain            *Img                `json:"-"`
	Imgs               []*Img              `json:"-"`
	Provider           *Provider           `json:"-"`
	Description        string              `json:"Description"`
	Duration           int                 `json:"Duration"` //minutes
	EnableZoom         bool                `json:"EnableZoom"`
	Interval           int                 `json:"Interval"`
	Location           string              `json:"Location"`
	LocationType       ServiceLocationType `json:"LocationType"`
	Name               string              `json:"Name"`
	Note               string              `json:"Note"`
	Padding            int                 `json:"Padding"` //minutes
	PaddingChanged     bool                `json:"-"`
	PaddingInitial     int                 `json:"PaddingInitial"`
	PaddingInitialUnit PaddingUnit         `json:"PaddingInitialUnit"`
	Price              float32             `json:"Price"` //dollars
	PriceType          PriceType           `json:"PriceType"`
	URLVideo           string              `json:"UrlVideo"`
	HTMLVideoPlayer    string              `json:"HtmlVideoPlayer"`
}

//ComputePrice : compute the price of the service
func (s *Service) ComputePrice() float32 {
	return s.PriceType.Compute(s.Price, s.Duration)
}

//SetFields : set service values
func (s *Service) SetFields(apptOnly bool, name string, desc string, note string, priceStr string, priceTypeStr string, durationStr string, locTypeStr string, loc string, paddingStr string, paddingInitialStr string, paddingInitialUnitStr string, intervalStr string, enableZoom bool, urlVideo string) {
	s.SetApptOnly(apptOnly)
	s.Description = desc
	s.EnableZoom = enableZoom && apptOnly
	s.Location = loc
	s.Name = name
	s.Note = note

	//parse the duration
	duration, _ := strconv.ParseInt(durationStr, 10, 32)
	s.Duration = int(duration)

	//parse the location
	locationType := ParseServiceLocation(locTypeStr)
	s.LocationType = locationType.Type

	//zoom is not supported for location types that support an actual location
	if locationType.Type.HasLocation() {
		s.EnableZoom = false
	}

	//parse the price
	price, _ := strconv.ParseFloat(priceStr, 32)
	s.Price = float32(price)
	priceType := ParsePriceType(priceTypeStr)
	s.PriceType = priceType

	//parse the padding
	padding, _ := strconv.ParseInt(paddingStr, 10, 32)
	s.SetPadding(int(padding))
	paddingInitial, _ := strconv.ParseInt(paddingInitialStr, 10, 32)
	s.PaddingInitial = int(paddingInitial)
	paddingInitialUnit := ParsePaddingUnit(paddingInitialUnitStr)
	s.PaddingInitialUnit = paddingInitialUnit

	//parse the interval
	interval := ParseServiceInterval(intervalStr)
	s.Interval = interval.Value

	//handle the video url
	s.SetURLVideo(urlVideo)
}

//SetPadding : set the padding
func (s *Service) SetPadding(padding int) {
	if s.Padding != padding {
		s.PaddingChanged = true
	}
	s.Padding = padding
}

//IsApptOnly : check if appointment-only
func (s *Service) IsApptOnly() bool {
	return IsApptOnly(s.Type)
}

//SetApptOnly : set the service type based on the appointment-only flag
func (s *Service) SetApptOnly(apptOnly bool) {
	if apptOnly {
		s.Type = ServiceTypeAppt
	} else {
		s.Type = ServiceTypeOnDemand
	}
}

//SetURLVideo : set the video URL
func (s *Service) SetURLVideo(url string) {
	s.URLVideo = url
	s.HTMLVideoPlayer = GenerateYouTubePlayerHTML(url)
}

//FormatPrice : format the price
func (s *Service) FormatPrice() string {
	return s.PriceType.Format(s.Price)
}

//IsDurationVariable : check if the duration is variable
func (s *Service) IsDurationVariable() bool {
	return ServiceDurationVariable.IsDurationVariable(s.Duration)
}

//FormatDuration : format the duration
func (s *Service) FormatDuration() string {
	if s.IsApptOnly() {
		for _, svcDuration := range ServiceDurationsBooking {
			if svcDuration.Value == s.Duration {
				return svcDuration.Label
			}
		}
	} else {
		for _, svcDuration := range ServiceDurationsOrder {
			if svcDuration.Value == s.Duration {
				return svcDuration.Label
			}
		}
	}
	return "n/a"
}

//compute the initial padding in minutes
func (s *Service) computePaddingInitialMinutes() int {
	switch s.PaddingInitialUnit {
	case PaddingUnitHours:
		return s.PaddingInitial * 60
	case PaddingUnitDays:
		return s.PaddingInitial * 24 * 60
	}
	return 0
}

//GetInterval : get the service interval
func (s *Service) GetInterval() time.Duration {
	if s.Interval == 0 {
		s.Interval = ServiceIntervalDefault
	}
	return time.Duration(s.Interval) * time.Minute
}

//ComputeStartTime : compute the minimum start time
func (s *Service) ComputeStartTime(now time.Time) time.Time {
	paddingMin := time.Duration(s.computePaddingInitialMinutes()) * time.Minute
	start := now.Add(paddingMin)
	start = s.Provider.AdjToValidStart(start, s.GetInterval())
	return start
}

//CheckValidTime : check if the time is a valid service time
func (s *Service) CheckValidTime(provider *Provider, now time.Time, start time.Time, end time.Time) bool {
	//check if the time is in the future
	if start.Before(now) {
		return false
	}

	//check if the time is after the minimum start
	svcStart := s.ComputeStartTime(now)
	if start.Before(svcStart) {
		return false
	}

	//check against the provider schedule
	check := provider.CheckValidTime(now, start, end)
	if !check {
		return false
	}
	return true
}

//ComputeTimeTo : compute the to-time based on the from-time
func (s *Service) ComputeTimeTo(from time.Time) time.Time {
	if s.IsApptOnly() {
		return from.Add(time.Duration(s.Duration) * time.Minute)
	}

	//use the same time for on-demand services
	return from
}

//SetImgs : set the service images
func (s *Service) SetImgs(files []*FileUpload) {
	imgs := make([]*Img, 0, len(files))
	for _, file := range files {
		img := &Img{
			Version: time.Now().Unix(),
		}
		img.SetFile(file.GetFile())
		imgs = append(imgs, img)
	}
	s.Imgs = imgs
}

//AddImgs : add the service images
func (s *Service) AddImgs(files []*FileUpload) {
	//set the images if there are no previous images
	if len(s.Imgs) == 0 {
		s.SetImgs(files)
		return
	}

	//add the new images
	for _, file := range files {
		img := &Img{
			Version: time.Now().Unix(),
		}
		img.SetFile(file.GetFile())
		s.Imgs = append(s.Imgs, img)
	}
}

//ProcessImgIndices : process the re-ordering and deleting of images based on a list of image indices
func (s *Service) ProcessImgIndices(idxStrs []string) error {
	lenImgs := len(s.Imgs)
	if lenImgs > 0 {
		imgs := make([]*Img, 0, len(idxStrs))
		for _, idxStr := range idxStrs {
			idx, err := strconv.Atoi(idxStr)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("invalid index: %s", idxStr))
			}
			if idx < 0 || idx > lenImgs {
				return fmt.Errorf("invalid index: %d", idx)
			}
			imgs = append(imgs, s.Imgs[idx])
		}
		s.Imgs = imgs
	}
	return nil
}

//FormatTime : format the service time
func (s *Service) FormatTime(timeFrom time.Time, timeZone string) string {
	return FormatDateTimeLocal(timeFrom, timeZone)
}

//ProcessBookingLocationInput : process the booking location input
func (s *Service) ProcessBookingLocationInput(location string) string {
	//use the service location
	if s.LocationType.IsLocationProvider() {
		return s.Location
	}

	//use the input location
	if s.LocationType.IsLocationClient() {
		return location
	}
	return ""
}

//load a service using the given statement
func loadService(ctx context.Context, db *DB, whereStmt string, args ...interface{}) (context.Context, *Service, error) {
	stmtImgSelect := CreateImgSelect("img")
	stmtImg := CreateImgJoin("s", "secondary_id", "img", ImgTypeSvc, ImgTypeSvcMain)
	stmtImgOrder := CreateImgOrder("img")
	stmt := fmt.Sprintf("SELECT BIN_TO_UUID(p.id),p.url_name,p.url_name_friendly,BIN_TO_UUID(p.user_id),p.data,BIN_TO_UUID(s.id),s.type,s.data,%s FROM %s s INNER JOIN %s p ON p.id=s.provider_id %s WHERE %s ORDER BY %s", stmtImgSelect, dbTableService, dbTableProvider, stmtImg, whereStmt, stmtImgOrder)

	//load the service
	ctx, rows, err := db.Query(ctx, stmt, args...)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "query row service")
	}

	//initialize the service
	var svc Service

	//read the row
	count := 0
	var providerIDStr string
	var providerURLName string
	var providerURLNameFriendly string
	var userIDStr string
	var providerDataStr string
	var idStr string
	var svcType int
	var dataStr string
	for rows.Next() {
		//image
		var imgIDStr sql.NullString
		var imgUserIDStr sql.NullString
		var imgProviderIDStr sql.NullString
		var imgSecondaryIDStr sql.NullString
		var imgImgType sql.NullInt32
		var imgFilePath sql.NullString
		var imgFileSrc sql.NullString
		var imgFileResized sql.NullString
		var imgIndex sql.NullInt32
		var imgDataStr sql.NullString
		err = rows.Scan(&providerIDStr, &providerURLName, &providerURLNameFriendly, &userIDStr, &providerDataStr, &idStr, &svcType, &dataStr, &imgIDStr, &imgUserIDStr, &imgProviderIDStr, &imgSecondaryIDStr, &imgImgType, &imgFilePath, &imgFileSrc, &imgFileResized, &imgIndex, &imgDataStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return ctx, nil, fmt.Errorf("no service: %v", args)
			}
			return ctx, nil, errors.Wrap(err, "select service")
		}

		//only use the first entry for the actual service
		if count == 0 {
			//parse the uuid
			providerID, err := uuid.FromString(providerIDStr)
			if err != nil {
				return nil, nil, errors.Wrap(err, "parse uuid provider id")
			}
			userID, err := uuid.FromString(userIDStr)
			if err != nil {
				return nil, nil, errors.Wrap(err, "parse uuid user id")
			}
			id, err := uuid.FromString(idStr)
			if err != nil {
				return nil, nil, errors.Wrap(err, "parse uuid service id")
			}

			//unmarshal provider data
			var provider Provider
			err = json.Unmarshal([]byte(providerDataStr), &provider)
			if err != nil {
				return ctx, nil, errors.Wrap(err, "unjson provider")
			}
			provider.ID = &providerID
			provider.URLName = providerURLName
			provider.URLNameFriendly = providerURLNameFriendly

			//unmarshal the data
			err = json.Unmarshal([]byte(dataStr), &svc)
			if err != nil {
				return ctx, nil, errors.Wrap(err, "unjson service")
			}
			svc.Provider = &provider
			svc.ID = &id
			svc.UserID = &userID
			svc.Type = ServiceType(svcType)
			svc.Imgs = make([]*Img, 0, 1)
		}
		count++

		//read the image
		img, err := CreateImg(imgIDStr, imgUserIDStr, imgSecondaryIDStr, imgProviderIDStr, imgImgType, imgFilePath, imgFileSrc, imgFileResized, imgIndex, imgDataStr)
		if err != nil {
			return ctx, nil, errors.Wrap(err, "read image")
		}
		if img != nil {
			switch img.Type {
			case ImgTypeSvc:
				svc.Imgs = append(svc.Imgs, img)
			case ImgTypeSvcMain:
				svc.ImgMain = img
			}
		}
	}
	return ctx, &svc, nil
}

//LoadServiceByID : load a service by id
func LoadServiceByID(ctx context.Context, db *DB, id *uuid.UUID) (context.Context, *Service, error) {
	whereStmt := "s.deleted=0 AND s.id=UUID_TO_BIN(?)"
	return loadService(ctx, db, whereStmt, id)
}

//LoadServiceByProviderIDAndID : load a service by the provider id and service id
func LoadServiceByProviderIDAndID(ctx context.Context, db *DB, providerID *uuid.UUID, id *uuid.UUID) (context.Context, *Service, error) {
	whereStmt := "s.deleted=0 AND p.id=UUID_TO_BIN(?) and s.id=UUID_TO_BIN(?)"
	return loadService(ctx, db, whereStmt, providerID, id)
}

//SaveService : save a service
func SaveService(ctx context.Context, db *DB, provider *Provider, svc *Service, now time.Time) (context.Context, error) {
	ctx, err := db.ProcessTx(ctx, "save service", func(ctx context.Context, db *DB) (context.Context, error) {
		//default the service id if necessary
		if svc.ID == nil {
			uuid, err := uuid.NewV4()
			if err != nil {
				return ctx, errors.Wrap(err, "new uuid service")
			}
			svc.ID = &uuid
		}
		svc.UserID = provider.User.ID
		svc.Provider = provider

		//json encode the service data
		svcJSON, err := json.Marshal(svc)
		if err != nil {
			return ctx, errors.Wrap(err, "json service")
		}

		//save to the db
		stmt := fmt.Sprintf("INSERT INTO %s(id,provider_id,type,data) VALUES (UUID_TO_BIN(?),UUID_TO_BIN(?),?,?) ON DUPLICATE KEY UPDATE type=VALUES(type),data=VALUES(data)", dbTableService)
		ctx, result, err := db.Exec(ctx, stmt, svc.ID, provider.ID, svc.Type, svcJSON)
		if err != nil {
			return ctx, errors.Wrap(err, "insert service")
		}
		count, err := result.RowsAffected()
		if err != nil {
			return ctx, errors.Wrap(err, "insert service rows affected")
		}

		//0 indicated no update, 1 an insert, 2 an update
		if count < 0 || count > 2 {
			return ctx, fmt.Errorf("unable to insert service: %s", provider.ID)
		}

		//save the first image as the main image
		if len(svc.Imgs) > 0 {
			svc.ImgMain = &Img{
				Version: time.Now().Unix(),
			}
			svc.ImgMain.SetFile(svc.Imgs[0].GetFile())
		} else {
			svc.ImgMain = nil
		}
		ctx, err = ProcessImgSingle(ctx, db, svc.UserID, svc.Provider.ID, svc.ID, ImgTypeSvcMain, svc.ImgMain)
		if err != nil {
			return ctx, errors.Wrap(err, "insert provider process image banner")
		}

		//process the images
		ctx, err = ProcessImgs(ctx, db, svc.UserID, svc.Provider.ID, svc.ID, ImgTypeSvc, svc.Imgs)
		if err != nil {
			return ctx, errors.Wrap(err, "insert service process images")
		}

		//adjust the padding for any upcoming bookings if necessary
		if svc.PaddingChanged {
			UpdateBookingPaddingForService(ctx, db, svc.ID, svc.Padding, now)
		}
		return ctx, nil
	})
	if err != nil {
		return ctx, errors.Wrap(err, "save service")
	}
	return ctx, nil
}

//DeleteService : delete a service, returning the number of bookings that may exists
//that would prevent the delete
func DeleteService(ctx context.Context, db *DB, providerID *uuid.UUID, svcID *uuid.UUID) (context.Context, int, error) {
	var err error
	var booksCount int
	ctx, err = db.ProcessTx(ctx, "delete service", func(ctx context.Context, db *DB) (context.Context, error) {
		//check for any bookings that would prevent the delete
		ctx, booksCount, err = CountBookingsForService(ctx, db, svcID)
		if err != nil {
			return ctx, errors.Wrap(err, "count bookings")
		}
		if booksCount > 0 {
			return ctx, nil
		}

		//delete the service
		stmt := fmt.Sprintf("UPDATE %s SET deleted=1 WHERE provider_id=UUID_TO_BIN(?) AND id=UUID_TO_BIN(?)", dbTableService)
		ctx, result, err := db.Exec(ctx, stmt, providerID, svcID)
		if err != nil {
			return ctx, errors.Wrap(err, "delete service")
		}
		count, err := result.RowsAffected()
		if err != nil {
			return ctx, errors.Wrap(err, "delete service rows affected")
		}
		if count == 0 {
			return ctx, fmt.Errorf("unable to delete service: %s", providerID)
		}
		return ctx, nil
	})
	if err != nil {
		return ctx, booksCount, errors.Wrap(err, "delete service")
	}
	return ctx, booksCount, nil
}

//listServices : list all services using the given statement
func listServices(ctx context.Context, db *DB, whereStmt string, args ...interface{}) (context.Context, []*Service, error) {
	ctx, logger := GetLogger(ctx)
	stmtImgSelect := CreateImgSelect("img")
	stmtImg := CreateImgJoin("s", "secondary_id", "img", ImgTypeSvc, ImgTypeSvcMain)
	stmt := fmt.Sprintf("SELECT BIN_TO_UUID(p.id),p.url_name,p.url_name_friendly,BIN_TO_UUID(p.user_id),p.data,BIN_TO_UUID(s.id),s.type,s.data,%s FROM %s s INNER JOIN %s p ON p.id=s.provider_id %s WHERE %s ORDER BY s.idx,s.created,s.id", stmtImgSelect, dbTableService, dbTableProvider, stmtImg, whereStmt)

	//list the services
	ctx, rows, err := db.Query(ctx, stmt, args...)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "select services")
	}

	//read the rows
	svcs := make([]*Service, 0, 2)
	var svc *Service
	var svcID string
	var providerIDStr string
	var providerURLName string
	var providerURLNameFriendly string
	var userIDStr string
	var providerDataStr string
	var idStr string
	var svcType int
	var dataStr string
	for rows.Next() {
		//image
		var imgIDStr sql.NullString
		var imgUserIDStr sql.NullString
		var imgProviderIDStr sql.NullString
		var imgSecondaryIDStr sql.NullString
		var imgImgType sql.NullInt32
		var imgFilePath sql.NullString
		var imgFileSrc sql.NullString
		var imgFileResized sql.NullString
		var imgIndex sql.NullInt32
		var imgDataStr sql.NullString
		err := rows.Scan(&providerIDStr, &providerURLName, &providerURLNameFriendly, &userIDStr, &providerDataStr, &idStr, &svcType, &dataStr, &imgIDStr, &imgUserIDStr, &imgProviderIDStr, &imgSecondaryIDStr, &imgImgType, &imgFilePath, &imgFileSrc, &imgFileResized, &imgIndex, &imgDataStr)
		if err != nil {
			return ctx, nil, errors.Wrap(err, "rows scan services")
		}

		//create a new service based on when the id changes
		if svcID != idStr {
			svc = &Service{}

			//parse the uuid
			providerID, err := uuid.FromString(providerIDStr)
			if err != nil {
				return ctx, nil, errors.Wrap(err, "parse uuid provider id")
			}
			userID, err := uuid.FromString(userIDStr)
			if err != nil {
				return ctx, nil, errors.Wrap(err, "parse uuid user id")
			}
			id, err := uuid.FromString(idStr)
			if err != nil {
				return ctx, nil, errors.Wrap(err, "parse uuid service id")
			}

			//unmarshal provider data
			var provider Provider
			err = json.Unmarshal([]byte(providerDataStr), &provider)
			if err != nil {
				return ctx, nil, errors.Wrap(err, "unjson provider")
			}
			provider.ID = &providerID
			provider.URLName = providerURLName
			provider.URLNameFriendly = providerURLNameFriendly

			//unmarshal the data
			err = json.Unmarshal([]byte(dataStr), &svc)
			if err != nil {
				return ctx, nil, errors.Wrap(err, "unjson service")
			}
			svc.Provider = &provider
			svc.ID = &id
			svc.UserID = &userID
			svc.Type = ServiceType(svcType)
			svc.Imgs = make([]*Img, 0, 1)
			svcs = append(svcs, svc)

			//track the id, to be used to detect a new service
			svcID = idStr
		}

		//continue to load images
		img, err := CreateImg(imgIDStr, imgUserIDStr, imgSecondaryIDStr, imgProviderIDStr, imgImgType, imgFilePath, imgFileSrc, imgFileResized, imgIndex, imgDataStr)
		if err != nil {
			return ctx, nil, errors.Wrap(err, "read image")
		}
		if img != nil {
			switch img.Type {
			case ImgTypeSvc:
				svc.Imgs = append(svc.Imgs, img)
			case ImgTypeSvcMain:
				svc.ImgMain = img
			}
		}
	}
	err = rows.Close()
	if err != nil {
		logger.Warnw("rows close", "error", err)
	}
	return ctx, svcs, nil
}

//ListServices : list all services for a provider
func ListServices(ctx context.Context, db *DB, providerID *uuid.UUID) (context.Context, []*Service, error) {
	whereStmt := "s.deleted=0 AND p.id=UUID_TO_BIN(?)"
	return listServices(ctx, db, whereStmt, providerID)
}

//ListServicesExcludeID : list all services for a provider excluding the specified id
func ListServicesExcludeID(ctx context.Context, db *DB, providerURLName string, id *uuid.UUID) (context.Context, []*Service, error) {
	whereStmt := "s.deleted=0 AND p.url_name=? AND s.id!=UUID_TO_BIN(?)"
	return listServices(ctx, db, whereStmt, providerURLName, id)
}

//UpdateServiceIndices : update service indices
func UpdateServiceIndices(ctx context.Context, db *DB, ids []*uuid.UUID) (context.Context, error) {
	ctx, err := db.ProcessTx(ctx, "update service indices", func(ctx context.Context, db *DB) (context.Context, error) {
		for i, id := range ids {
			stmt := fmt.Sprintf("UPDATE %s SET idx=? WHERE id=UUID_TO_BIN(?)", dbTableService)
			ctx, result, err := db.Exec(ctx, stmt, i, id)
			if err != nil {
				return ctx, errors.Wrap(err, fmt.Sprintf("update service index: %d: %s", i, id))
			}
			_, err = result.RowsAffected()
			if err != nil {
				return ctx, errors.Wrap(err, fmt.Sprintf("update service index rows affected: %d: %s", i, id))
			}
		}
		return ctx, nil
	})
	if err != nil {
		return ctx, errors.Wrap(err, "update service indices")
	}
	return ctx, nil
}

//ServiceProviderUser : definition of a service provider user
type ServiceProviderUser struct {
	ID   *uuid.UUID    `json:"-"`
	User *ProviderUser `json:"-"`
}

//AddProviderUserToService : add a provider user to a service
func AddProviderUserToService(ctx context.Context, db *DB, providerID *uuid.UUID, serviceID *uuid.UUID, userID *uuid.UUID) (context.Context, error) {
	stmt := fmt.Sprintf("INSERT INTO %s(id,provider_id,service_id,provider_user_id) SELECT UUID_TO_BIN(UUID()),provider_id,id,UUID_TO_BIN(?) FROM %s WHERE deleted=0 AND provider_id=UUID_TO_BIN(?) AND id=UUID_TO_BIN(?)", dbTableServiceProviderUser, dbTableService)
	ctx, result, err := db.Exec(ctx, stmt, userID, providerID, serviceID)
	if err != nil {
		return ctx, errors.Wrap(err, "insert service provider user")
	}
	count, err := result.RowsAffected()
	if err != nil {
		return ctx, errors.Wrap(err, "insert service provider user rows affected")
	}
	if count != 1 {
		return ctx, fmt.Errorf("unable to insert service provider user: %s: %s: %s", providerID, serviceID, userID)
	}
	return ctx, nil
}

//DeleteProviderUserFromService : delete a provider user from a service
func DeleteProviderUserFromService(ctx context.Context, db *DB, providerID *uuid.UUID, serviceID *uuid.UUID, userID *uuid.UUID) (context.Context, error) {
	stmt := fmt.Sprintf("UPDATE %s SET deleted=1 WHERE provider_id=UUID_TO_BIN(?) AND service_id=UUID_TO_BIN(?) AND id=UUID_TO_BIN(?)", dbTableServiceProviderUser)
	ctx, result, err := db.Exec(ctx, stmt, providerID, serviceID, userID)
	if err != nil {
		return ctx, errors.Wrap(err, "delete service provider user")
	}
	_, err = result.RowsAffected()
	if err != nil {
		return ctx, errors.Wrap(err, "delete service provider user rows affected")
	}
	return ctx, nil
}

//LoadProviderUserForServiceByProviderIDAndServiceIDAndUserID : load a a provider user for a service by provider id, service id, and user id
func LoadProviderUserForServiceByProviderIDAndServiceIDAndUserID(ctx context.Context, db *DB, providerID *uuid.UUID, serviceID *uuid.UUID, providerUserID *uuid.UUID) (context.Context, *ProviderUser, error) {
	stmt := fmt.Sprintf("SELECT pu.login,pu.data,BIN_TO_UUID(u.id),u.email,u.data FROM %s spu INNER JOIN %s pu ON pu.id=spu.provider_user_id AND pu.deleted=0 INNER JOIN %s u ON u.id=pu.user_id and u.deleted=0 WHERE spu.deleted=0 AND spu.provider_id=UUID_TO_BIN(?) AND spu.service_id=UUID_TO_BIN(?) AND spu.provider_user_id=UUID_TO_BIN(?)", dbTableServiceProviderUser, dbTableProviderUser, dbTableUser)
	ctx, row, err := db.QueryRow(ctx, stmt, providerID, serviceID, providerUserID)

	//read the row
	var providerLogin string
	var providerDataStr string
	var userIDStr string
	var userEmail string
	var userDataStr string
	err = row.Scan(&providerLogin, &providerDataStr, &userIDStr, &userEmail, &userDataStr)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "select service provider user")
	}

	//unmarshal the data
	var providerUser ProviderUser
	err = json.Unmarshal([]byte(providerDataStr), &providerUser)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "unjson provider user")
	}
	providerUser.ID = providerUserID
	providerUser.ProviderID = providerID
	providerUser.Login = providerLogin

	//unmarshal the user
	var user User
	err = json.Unmarshal([]byte(userDataStr), &user)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "unjson user")
	}
	userID, err := uuid.FromString(userIDStr)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "parse uuid user id")
	}
	user.ID = &userID
	user.Email = userEmail
	providerUser.User = &user
	return ctx, &providerUser, nil
}

//ListProviderUsersForService : list provider users for a service
func ListProviderUsersForService(ctx context.Context, db *DB, providerID *uuid.UUID, serviceID *uuid.UUID) (context.Context, map[uuid.UUID]*ServiceProviderUser, error) {
	ctx, logger := GetLogger(ctx)
	stmt := fmt.Sprintf("SELECT BIN_TO_UUID(spu.id),BIN_TO_UUID(pu.id),pu.login,pu.data,BIN_TO_UUID(u.id),u.email,u.data FROM %s spu INNER JOIN %s pu ON pu.id=spu.provider_user_id AND pu.deleted=0 INNER JOIN %s u ON u.id=pu.user_id AND u.deleted=0 WHERE spu.deleted=0 AND spu.provider_id=UUID_TO_BIN(?) AND spu.service_id=UUID_TO_BIN(?) ORDER BY u.email", dbTableServiceProviderUser, dbTableProviderUser, dbTableUser)
	ctx, rows, err := db.Query(ctx, stmt, providerID, serviceID)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "select service provider users")
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			logger.Warnw("rows close", "error", err)
		}
	}()

	//read the rows
	serviceProviderUsers := make(map[uuid.UUID]*ServiceProviderUser, 2)
	var idStr string
	var providerUserID string
	var providerUserLogin string
	var providerUserData string
	var userIDStr string
	var email string
	var userData string
	for rows.Next() {
		err := rows.Scan(&idStr, &providerUserID, &providerUserLogin, &providerUserData, &userIDStr, &email, &userData)
		if err != nil {
			return ctx, nil, errors.Wrap(err, "rows scan service provider users")
		}

		//parse the uuid
		id, err := uuid.FromString(idStr)
		if err != nil {
			return ctx, nil, errors.Wrap(err, "parse uuid id")
		}
		providerID, err := uuid.FromString(providerUserID)
		if err != nil {
			return ctx, nil, errors.Wrap(err, "parse uuid provider user id")
		}
		userID, err := uuid.FromString(userIDStr)
		if err != nil {
			return ctx, nil, errors.Wrap(err, "parse uuid user id")
		}

		//create the service provider user
		serviceProviderUser := &ServiceProviderUser{
			ID: &id,
		}

		//unmarshal the provider user data
		var providerUser ProviderUser
		err = json.Unmarshal([]byte(providerUserData), &providerUser)
		if err != nil {
			return ctx, nil, errors.Wrap(err, "unjson provider user")
		}
		providerUser.ID = &providerID
		providerUser.Login = providerUserLogin
		serviceProviderUser.User = &providerUser

		//unmarshal the user data
		var user User
		err = json.Unmarshal([]byte(userData), &user)
		if err != nil {
			return ctx, nil, errors.Wrap(err, "unjson user")
		}
		user.ID = &userID
		user.Email = email
		providerUser.User = &user
		serviceProviderUsers[*providerUser.ID] = serviceProviderUser
	}
	return ctx, serviceProviderUsers, nil
}
