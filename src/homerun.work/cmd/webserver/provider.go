package main

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

//provider constants
const (
	ProviderDefaultScheduleDuration = 480
	ProviderDefaultScheduleStart    = "9:00 AM"
	ProviderLengthRandomURLName     = 6
)

//ProviderSchedule : working schedule for a provider
type ProviderSchedule struct {
	DaySchedules map[time.Weekday]*DaySchedule `json:"DaySchedules"`
}

//IsUnavailable : check if a day of the week is unavailable
func (p *ProviderSchedule) IsUnavailable(dayOfWeek time.Weekday) bool {
	schedule, ok := p.DaySchedules[dayOfWeek]
	if !ok {
		return true
	}
	return len(schedule.TimePeriods) == 0
}

//process a day's schedule, returning any time period that overflows into the next day
func (p *ProviderSchedule) processDaySchedule(now time.Time, timeZone string, dayOfWeek time.Weekday, overflowPeriodIn *TimePeriod) (*TimePeriod, bool) {
	daySchedule := p.DaySchedules[dayOfWeek]

	//generate the time periods in the given day
	timePeriods := make([]*TimePeriod, 0, len(daySchedule.TimeDurations))

	//add the existing overflow period
	if overflowPeriodIn != nil {
		start, end := AdjTimes(now, overflowPeriodIn.Start, overflowPeriodIn.End)
		overflowPeriodIn = &TimePeriod{
			Start: start.UTC(),
			End:   end.UTC(),
		}
		timePeriods = append(timePeriods, overflowPeriodIn)
	}

	//process the periods
	var overflowPeriod *TimePeriod
	if !daySchedule.Unavailable {
		prevPeriod := overflowPeriodIn
		for _, timeDuration := range daySchedule.TimeDurations {
			start := timeDuration.Start
			end := timeDuration.GetEnd()
			start, end = AdjTimes(now, start, end)

			//sanity check overlaps
			if prevPeriod != nil && prevPeriod.IsOverlap(start, end) {
				return nil, false
			}

			//add the time period, splitting if the period crosses into the next day
			var timePeriod *TimePeriod
			if end.Weekday() != now.Weekday() {
				timePeriod = &TimePeriod{
					Start: start.UTC(),
					End:   GetEndOfDay(start).UTC(),
				}

				//sanity check that only one overflow exists
				if overflowPeriod != nil {
					return nil, false
				}
				overflowPeriod = &TimePeriod{
					Start: GetBeginningOfDay(end).UTC(),
					End:   end.UTC(),
				}

				//ignore any period that has no duration, which can happen if a period end on midnight
				if overflowPeriod.Start.Equal(overflowPeriod.End) {
					overflowPeriod = nil
				}
			} else {
				timePeriod = &TimePeriod{
					Start: start.UTC(),
					End:   end.UTC(),
				}
			}
			timePeriods = append(timePeriods, timePeriod)
			prevPeriod = timePeriod
		}
	}
	daySchedule.TimePeriods = timePeriods
	return overflowPeriod, true
}

//Process : process the schedule and generate the time periods across the various dates
func (p *ProviderSchedule) Process(now time.Time, timeZone string) []string {
	daysOfWeek := make([]string, 0, 7)
	overflowPeriod, ok := p.processDaySchedule(now, timeZone, time.Monday, nil)
	if !ok {
		daysOfWeek = append(daysOfWeek, time.Monday.String())
	}
	overflowPeriod, ok = p.processDaySchedule(now, timeZone, time.Tuesday, overflowPeriod)
	if !ok {
		daysOfWeek = append(daysOfWeek, time.Tuesday.String())
	}
	overflowPeriod, ok = p.processDaySchedule(now, timeZone, time.Wednesday, overflowPeriod)
	if !ok {
		daysOfWeek = append(daysOfWeek, time.Wednesday.String())
	}
	overflowPeriod, ok = p.processDaySchedule(now, timeZone, time.Thursday, overflowPeriod)
	if !ok {
		daysOfWeek = append(daysOfWeek, time.Thursday.String())
	}
	overflowPeriod, ok = p.processDaySchedule(now, timeZone, time.Friday, overflowPeriod)
	if !ok {
		daysOfWeek = append(daysOfWeek, time.Friday.String())
	}
	overflowPeriod, ok = p.processDaySchedule(now, timeZone, time.Saturday, overflowPeriod)
	if !ok {
		daysOfWeek = append(daysOfWeek, time.Saturday.String())
	}
	overflowPeriod, ok = p.processDaySchedule(now, timeZone, time.Sunday, overflowPeriod)
	if !ok {
		daysOfWeek = append(daysOfWeek, time.Sunday.String())
	}

	//add the overflow to monday if necessary
	if overflowPeriod != nil {
		_, ok := p.processDaySchedule(now, timeZone, time.Monday, overflowPeriod)
		if !ok {
			daysOfWeek = append(daysOfWeek, time.Monday.String())
		}
	}
	return daysOfWeek
}

//Adjust : adjust the schedule based on a change of timezones
func (p *ProviderSchedule) Adjust(now time.Time, timeZoneSrc string, timeZoneDst string) {
	//compute the adjustment for the times based on the timezones
	locSrc := GetLocation(timeZoneSrc)
	_, offsetSrc := now.In(locSrc).Zone()
	locDst := GetLocation(timeZoneDst)
	_, offsetDst := now.In(locDst).Zone()
	offset := time.Duration(offsetSrc-offsetDst) * time.Second

	//process the schedule
	for _, daySchedule := range p.DaySchedules {
		for _, timeDuration := range daySchedule.TimeDurations {
			timeDuration.Start = timeDuration.Start.Add(offset)
		}
		for _, timePeriod := range daySchedule.TimePeriods {
			timePeriod.Start = timePeriod.Start.Add(offset)
			timePeriod.End = timePeriod.End.Add(offset)
		}
	}
}

//Provider : provider definition
type Provider struct {
	ID              *uuid.UUID        `json:"-"`
	URLName         string            `json:"-"`
	URLNameFriendly string            `json:"-"`
	Domain          *string           `json:"-"`
	Created         time.Time         `json:"-"`
	About           string            `json:"About"`
	Name            string            `json:"Name"`
	Description     string            `json:"Description"`
	Education       string            `json:"Education"`
	Experience      string            `json:"Experience"`
	Location        string            `json:"Location"`
	ServiceArea     string            `json:"ServiceArea"`
	ServiceCreated  *time.Time        `json:"ServiceCreated"`
	URLFacebook     string            `json:"UrlFacebook"`
	URLInstagram    string            `json:"UrlInstagram"`
	URLLinkedIn     string            `json:"UrlLinkedIn"`
	URLTwitter      string            `json:"UrlTwitter"`
	URLWeb          string            `json:"URLWeb"`
	Schedule        *ProviderSchedule `json:"Schedule"`
	DomainPrevious  *string           `json:"DomainPrevious"`

	//images
	ImgBanner  *Img `json:"-"`
	ImgFavIcon *Img `json:"-"`
	ImgLogo    *Img `json:"-"`

	//user information
	User *User `json:"-"`

	//provider user information
	ProviderUser *ProviderUser `json:"-"`

	//payment methods
	PayPalEmail *string      `json:"PayPalEmail"`
	StripeToken *TokenStripe `json:"StripeToken"`
	ZelleID     *string      `json:"ZelleID"`

	//google
	GoogleTrackingID     *string         `json:"GoogleTrackingId"`
	GoogleCalendarID     *string         `json:"-"`
	GoogleCalendarUpdate bool            `json:"-"`
	GoogleCalendarData   *CalendarGoogle `json:"-"`
}

//IsAdmin : check if the user is an administrator
func (p *Provider) IsAdmin() bool {
	//ignore if the user added themselves
	return p.ProviderUser == nil || (p.ProviderUser.UserID != nil && p.User.ID.String() == p.ProviderUser.UserID.String())
}

//CheckAdmin : check if a user is an admin
func (p *Provider) CheckAdmin(userID *uuid.UUID) bool {
	if userID == nil {
		return false
	}
	userIDStr := userID.String()
	return userIDStr == p.User.ID.String()
}

//CheckUserAccess : check if a user has access to the provider as the owner or a user
func (p *Provider) CheckUserAccess(userID *uuid.UUID, providerUser *ProviderUser) bool {
	if userID == nil {
		return false
	}
	userIDStr := userID.String()
	if userIDStr == p.User.ID.String() {
		return true
	}
	if providerUser == nil {
		return false
	}
	if providerUser.User == nil {
		return false
	}
	return userIDStr == providerUser.User.ID.String()
}

//GetProviderUser : get the provider user
func (p *Provider) GetProviderUser() *User {
	if p.ProviderUser == nil {
		return nil
	}
	return p.ProviderUser.User
}

//GetUser : get the user
func (p *Provider) GetUser() *User {
	if p.IsAdmin() {
		return p.User
	}

	//use the provider user
	return p.ProviderUser.User
}

//GetZoomToken : get the Zoom token
func (p *Provider) GetZoomToken() *TokenZoom {
	if p.IsAdmin() {
		return p.User.ZoomToken
	}

	//use the provider user token
	if p.ProviderUser.User != nil {
		return p.ProviderUser.User.ZoomToken
	}
	return nil
}

//GetSchedule : get the provider schedule
func (p *Provider) GetSchedule() *ProviderSchedule {
	if p.IsAdmin() {
		return p.Schedule
	}

	//use the provider user schedule
	return p.ProviderUser.Schedule
}

//SetSchedule : set the provider schedule
func (p *Provider) SetSchedule(schedule *ProviderSchedule) {
	if p.IsAdmin() {
		p.Schedule = schedule
		return
	}

	//set the provider user schedule
	p.ProviderUser.Schedule = schedule
}

//SupportsPayment : check if payments are supported
func (p *Provider) SupportsPayment() bool {
	return p.StripeToken != nil || p.PayPalEmail != nil || p.ZelleID != nil
}

//IsMappable : check if the location is mappable
func (p *Provider) IsMappable() bool {
	if p.Location == "" {
		return false
	}
	return IsMappable(p.Location)
}

//IsUnavailable : check if a day of the week is unavailable
func (p *Provider) IsUnavailable(dayOfWeek time.Weekday) bool {
	schedule := p.GetSchedule()
	if schedule == nil {
		return true
	}
	return schedule.IsUnavailable(dayOfWeek)
}

//GetCalendarTitle : create the calendar title
func (p *Provider) GetCalendarTitle() string {
	return fmt.Sprintf("HomeRun: %s", p.Name)
}

//SetDomain : set the domain of the provider
func (p *Provider) SetDomain(domain *string) {
	p.DomainPrevious = p.Domain
	if domain == nil {
		p.Domain = domain
		return
	}
	domainLower := strings.ToLower(*domain)
	p.Domain = &domainLower
}

//SetName : set the name of the provider
func (p *Provider) SetName(name string) {
	if p.Name != name {
		p.GoogleCalendarUpdate = true
	}
	p.Name = name
}

//GetURLName : get the provider URL name, prioritizing the friendly name if set
func (p *Provider) GetURLName() string {
	if p.URLNameFriendly != "" {
		return p.URLNameFriendly
	}
	return p.URLName
}

//FormatCalendarGoogleURL : format the Google calendar URL
func (p *Provider) FormatCalendarGoogleURL() string {
	return FormatGoogleCalendarURL(p.GoogleCalendarID)
}

//FormatCalendarIcalURL : format the Ical calendar URL
func (p *Provider) FormatCalendarIcalURL() string {
	if p.GoogleCalendarID == nil {
		return ""
	}
	return FormatGoogleCalendarIcalURL(p.GoogleCalendarID)
}

//SetImgBanner : set the banner image
func (p *Provider) SetImgBanner(file string) {
	p.ImgBanner = &Img{
		Version: time.Now().Unix(),
	}
	p.ImgBanner.SetFile(file)
}

//DeleteImgBanner : delete the banner image
func (p *Provider) DeleteImgBanner() {
	p.ImgBanner = nil
}

//SetImgLogo : set the logo image
func (p *Provider) SetImgLogo(file string) {
	p.ImgLogo = &Img{
		Version: time.Now().Unix(),
	}
	p.ImgLogo.SetFile(file)
}

//DeleteImgLogo : delete the logo image
func (p *Provider) DeleteImgLogo() {
	p.ImgLogo = nil
}

//GetBoundaryTimes : the earliest and latest work times for a day of the week
func (p *Provider) GetBoundaryTimes(t time.Time) (time.Time, time.Time) {
	schedule := p.GetSchedule()
	if schedule == nil {
		return time.Time{}, time.Time{}
	}
	daySchedule := schedule.DaySchedules[t.Weekday()]

	//walk the durations and find the start and end times
	var startFirst time.Time
	var endLast time.Time
	for idx, timePeriod := range daySchedule.TimePeriods {
		start := timePeriod.Start
		end := timePeriod.End
		if idx == 0 {
			startFirst = start
		}
		endLast = end
	}

	//adjust the times to reflect the current time
	if !startFirst.IsZero() && !endLast.IsZero() {
		startFirst, endLast = AdjTimes(t, startFirst, endLast)
	}
	return startFirst, endLast
}

//IsValidWorkPeriod : check if the time period is valid giving the schedule
func (p *Provider) IsValidWorkPeriod(ref time.Time, period *TimePeriod) bool {
	schedule := p.GetSchedule()
	if schedule == nil {
		return true
	}
	daySchedule := schedule.DaySchedules[ref.Weekday()]

	//check if the period falls within a valid time duration
	for _, timeDuration := range daySchedule.TimeDurations {
		//adjust the times to reflect the current time
		scheduleStart, scheduleEnd := AdjTimes(ref, timeDuration.Start, timeDuration.GetEnd())
		if CheckTimeIn(period.Start, scheduleStart, scheduleEnd) && CheckTimeIn(period.End, scheduleStart, scheduleEnd) {
			return true
		}
	}

	//check if the period falls within a valid time period due to an overflow
	for _, timePeriod := range daySchedule.TimePeriods {
		//adjust the times to reflect the current time
		scheduleStart, scheduleEnd := AdjTimes(ref, timePeriod.Start, timePeriod.End)
		if CheckTimeIn(period.Start, scheduleStart, scheduleEnd) && CheckTimeIn(period.End, scheduleStart, scheduleEnd) {
			return true
		}
	}
	return false
}

//GetWorkingMinutes : get number of working minutes available on the given work day
func (p *Provider) GetWorkingMinutes(t time.Time) time.Duration {
	schedule := p.GetSchedule()
	if schedule == nil {
		return 0
	}
	daySchedule := schedule.DaySchedules[t.Weekday()]

	//check if the period falls within a valid time duration
	totalDuration := time.Duration(0)
	for _, timePeriod := range daySchedule.TimePeriods {
		totalDuration += timePeriod.End.Sub(timePeriod.Start)
	}
	return totalDuration
}

//CheckValidTime : check if the time is a valid service time
func (p *Provider) CheckValidTime(now time.Time, start time.Time, end time.Time) bool {
	//check if the time is in the future
	if start.Before(now) {
		return false
	}

	//check if the time is consistent with the working hours
	period := &TimePeriod{
		Start: start,
		End:   end,
	}
	return p.IsValidWorkPeriod(start, period)
}

//AdjToValidStart : adjust to the next valid start time
func (p *Provider) AdjToValidStart(d time.Time, interval time.Duration) time.Time {
	//round to the nearest interval
	minutes := math.Ceil(float64(d.Minute())/interval.Minutes()) * interval.Minutes()
	d = time.Date(d.Year(), d.Month(), d.Day(), d.Hour(), int(minutes), 0, 0, d.Location())

	//keep checking if a valid period and adjust to the next working block, checking up to a week
	intervals := int(math.Ceil(7 * 24 * 60 / interval.Minutes()))
	for i := 0; i < intervals; i++ {
		period := &TimePeriod{
			Start: d,
			End:   d,
		}
		if !p.IsValidWorkPeriod(d, period) {
			//probe the next interval
			d = d.Add(interval)
			continue
		}
		break
	}
	return d
}

//ListDaysOfWeekUnavailable : list the days of week that are unavailable, where 0=Sun through 6=Sat
func (p *Provider) ListDaysOfWeekUnavailable() []int {
	days := make([]int, 0, 7)
	if p.IsUnavailable(time.Sunday) {
		days = append(days, 0)
	}
	if p.IsUnavailable(time.Monday) {
		days = append(days, 1)
	}
	if p.IsUnavailable(time.Tuesday) {
		days = append(days, 2)
	}
	if p.IsUnavailable(time.Wednesday) {
		days = append(days, 3)
	}
	if p.IsUnavailable(time.Thursday) {
		days = append(days, 4)
	}
	if p.IsUnavailable(time.Friday) {
		days = append(days, 5)
	}
	if p.IsUnavailable(time.Saturday) {
		days = append(days, 6)
	}
	return days
}

//FormatUserID : get the user id string
func (p *Provider) FormatUserID() string {
	if p.User == nil || p.User.ID == nil {
		return ""
	}
	return p.User.ID.String()
}

//provider db tables
const (
	dbTableProvider     = "provider"
	dbTableProviderUser = "provider_user"
)

//URLNameFriendlyExists : check if a friendly URL name exists for a provider
func URLNameFriendlyExists(ctx context.Context, db *DB, name string) (context.Context, bool, error) {
	//use a safe form of the name
	urlName := GenURLString(name)

	//execute the query
	stmt := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE deleted=0 AND (url_name=? OR url_name_friendly=?)", dbTableProvider)
	ctx, row, err := db.QueryRow(ctx, stmt, urlName, urlName)
	if err != nil {
		return ctx, false, errors.Wrap(err, "query row provider url name")
	}

	//read the row
	var count int
	err = row.Scan(&count)
	if err != nil {
		if err == sql.ErrNoRows {
			return ctx, false, nil
		}
		return ctx, false, errors.Wrap(err, "select provider url name")
	}
	return ctx, count > 0, nil
}

//DomainExists : check if a domain name exists for a provider
func DomainExists(ctx context.Context, db *DB, domain string) (context.Context, bool, error) {
	//execute the query
	stmt := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE deleted=0 AND domain=?", dbTableProvider)
	ctx, row, err := db.QueryRow(ctx, stmt, domain)
	if err != nil {
		return ctx, false, errors.Wrap(err, "query row provider url domain")
	}

	//read the row
	var count int
	err = row.Scan(&count)
	if err != nil {
		if err == sql.ErrNoRows {
			return ctx, false, nil
		}
		return ctx, false, errors.Wrap(err, "select provider url domain")
	}
	return ctx, count > 0, nil
}

//load a provider using the given sql where clause
func loadProvider(ctx context.Context, db *DB, whereStmt string, args ...interface{}) (context.Context, *Provider, error) {
	//create the final query
	stmtLogoSelect := CreateImgSelect("imglogo")
	stmtLogo := CreateImgJoin("p", "provider_id", "imglogo", ImgTypeLogo, 0)
	stmtBannerSelect := CreateImgSelect("imgbanner")
	stmtBanner := CreateImgJoin("p", "provider_id", "imgbanner", ImgTypeBanner, 0)
	stmtFavIconSelect := CreateImgSelect("imgfavicon")
	stmtFavIcon := CreateImgJoin("p", "provider_id", "imgfavicon", ImgTypeFavIcon, 0)
	stmt := fmt.Sprintf("SELECT BIN_TO_UUID(u.id),u.email,u.email_verified,u.disable_emails,u.is_oauth,u.token_zoom_data,u.data,BIN_TO_UUID(p.id),p.url_name,p.url_name_friendly,p.domain,p.calendar_google_id,p.calendar_google_data,p.data,%s,%s,%s FROM %s p INNER JOIN %s u ON u.id=p.user_id %s %s %s WHERE %s", stmtLogoSelect, stmtBannerSelect, stmtFavIconSelect, dbTableProvider, dbTableUser, stmtLogo, stmtBanner, stmtFavIcon, whereStmt)

	//load the provider
	ctx, row, err := db.QueryRow(ctx, stmt, args...)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "query row provider")
	}

	//read the row
	var userIDStr string
	var email string
	var emailVerifiedBit string
	var disableEmailsBit string
	var isOAuthBit string
	var tokenZoomData sql.NullString
	var userDataStr string
	var idStr string
	var urlName string
	var urlNameFriendly sql.NullString
	var domain sql.NullString
	var calendarGoogleID sql.NullString
	var calendarGoogleData sql.NullString
	var dataStr string

	//logo image
	var logoIDStr sql.NullString
	var logoUserIDStr sql.NullString
	var logoProviderIDStr sql.NullString
	var logoSecondaryIDStr sql.NullString
	var logoImgType sql.NullInt32
	var logoFilePath sql.NullString
	var logoFileSrc sql.NullString
	var logoFileResized sql.NullString
	var logoIndex sql.NullInt32
	var logoDataStr sql.NullString

	//banner image
	var bannerIDStr sql.NullString
	var bannerUserIDStr sql.NullString
	var bannerProviderIDStr sql.NullString
	var bannerSecondaryIDStr sql.NullString
	var bannerImgType sql.NullInt32
	var bannerFilePath sql.NullString
	var bannerFileSrc sql.NullString
	var bannerFileResized sql.NullString
	var bannerIndex sql.NullInt32
	var bannerDataStr sql.NullString

	//favicon image
	var favIconIDStr sql.NullString
	var favIconUserIDStr sql.NullString
	var favIconProviderIDStr sql.NullString
	var favIconSecondaryIDStr sql.NullString
	var favIconImgType sql.NullInt32
	var favIconFilePath sql.NullString
	var favIconFileSrc sql.NullString
	var favIconFileResized sql.NullString
	var favIconIndex sql.NullInt32
	var favIconDataStr sql.NullString

	//read the data
	err = row.Scan(
		&userIDStr,
		&email,
		&emailVerifiedBit,
		&disableEmailsBit,
		&isOAuthBit,
		&tokenZoomData,
		&userDataStr,
		&idStr, &urlName,
		&urlNameFriendly,
		&domain,
		&calendarGoogleID,
		&calendarGoogleData,
		&dataStr,

		//logo
		&logoIDStr,
		&logoUserIDStr,
		&logoProviderIDStr,
		&logoSecondaryIDStr,
		&logoImgType,
		&logoFilePath,
		&logoFileSrc,
		&logoFileResized,
		&logoIndex,
		&logoDataStr,

		//banner
		&bannerIDStr,
		&bannerUserIDStr,
		&bannerProviderIDStr,
		&bannerSecondaryIDStr,
		&bannerImgType,
		&bannerFilePath,
		&bannerFileSrc,
		&bannerFileResized,
		&bannerIndex,
		&bannerDataStr,

		//favicon
		&favIconIDStr,
		&favIconUserIDStr,
		&favIconProviderIDStr,
		&favIconSecondaryIDStr,
		&favIconImgType,
		&favIconFilePath,
		&favIconFileSrc,
		&favIconFileResized,
		&favIconIndex,
		&favIconDataStr,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return ctx, nil, nil
		}
		return ctx, nil, errors.Wrap(err, "select provider")
	}

	//parse the uuid
	userID, err := uuid.FromString(userIDStr)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "parse uuid user")
	}
	providerID, err := uuid.FromString(idStr)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "parse uuid provider")
	}

	//unmarshal the user data
	var user User
	err = json.Unmarshal([]byte(userDataStr), &user)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "unjson user")
	}
	user.ID = &userID
	user.Email = email
	user.EmailVerified = emailVerifiedBit == "\x01"
	user.DisableEmails = disableEmailsBit == "\x01"
	user.IsOAuth = isOAuthBit == "\x01"

	//check for zoom token data
	if tokenZoomData.Valid {
		var token TokenZoom
		err = json.Unmarshal([]byte(tokenZoomData.String), &token)
		if err != nil {
			return ctx, nil, errors.Wrap(err, "unjson zoom token")
		}
		user.ZoomToken = &token
	}

	//unmarshal the data
	var provider *Provider
	err = json.Unmarshal([]byte(dataStr), &provider)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "unjson provider")
	}
	provider.User = &user
	provider.ID = &providerID
	provider.URLName = urlName

	//check for a valid friendly url name
	if urlNameFriendly.Valid {
		provider.URLNameFriendly = urlNameFriendly.String
	}

	//check for a valid domain
	if domain.Valid {
		provider.Domain = &domain.String
	}

	//check for google calendar data
	if calendarGoogleID.Valid {
		provider.GoogleCalendarID = &calendarGoogleID.String
	}
	if calendarGoogleData.Valid {
		var cal CalendarGoogle
		err = json.Unmarshal([]byte(calendarGoogleData.String), &cal)
		if err != nil {
			return ctx, nil, errors.Wrap(err, "unjson google calendar")
		}
		provider.GoogleCalendarData = &cal
	}

	//read the images
	img, err := CreateImg(logoIDStr, logoUserIDStr, logoSecondaryIDStr, logoProviderIDStr, logoImgType, logoFilePath, logoFileSrc, logoFileResized, logoIndex, logoDataStr)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "read image logo")
	}
	provider.ImgLogo = img
	img, err = CreateImg(bannerIDStr, bannerUserIDStr, bannerSecondaryIDStr, bannerProviderIDStr, bannerImgType, bannerFilePath, bannerFileSrc, bannerFileResized, bannerIndex, bannerDataStr)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "read image banner")
	}
	provider.ImgBanner = img
	img, err = CreateImg(favIconIDStr, favIconUserIDStr, favIconSecondaryIDStr, favIconProviderIDStr, favIconImgType, favIconFilePath, favIconFileSrc, favIconFileResized, favIconIndex, favIconDataStr)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "read image banner")
	}
	provider.ImgFavIcon = img
	return ctx, provider, nil
}

//LoadProviderByID : load a provider by id
func LoadProviderByID(ctx context.Context, db *DB, id *uuid.UUID) (context.Context, *Provider, error) {
	whereStmt := "u.deleted=0 AND p.deleted=0 AND p.id=UUID_TO_BIN(?)"
	ctx, provider, err := loadProvider(ctx, db, whereStmt, id)
	if err != nil {
		return ctx, nil, err
	}
	if provider == nil {
		return ctx, nil, fmt.Errorf("no provider: %s", id)
	}
	return ctx, provider, err
}

//LoadProviderByUserID : load a provider by user id
func LoadProviderByUserID(ctx context.Context, db *DB, userID *uuid.UUID) (context.Context, *Provider, error) {
	whereStmt := "u.deleted=0 AND p.deleted=0 AND p.user_id=UUID_TO_BIN(?)"
	return loadProvider(ctx, db, whereStmt, userID)
}

//LoadProviderByURLName : load a provider by URL name
func LoadProviderByURLName(ctx context.Context, db *DB, urlName string) (context.Context, *Provider, error) {
	whereStmt := "u.deleted=0 AND p.deleted=0 AND (p.url_name=? OR p.url_name_friendly=?)"
	ctx, provider, err := loadProvider(ctx, db, whereStmt, urlName, urlName)
	if err != nil {
		return ctx, nil, err
	}
	if provider == nil {
		return ctx, nil, fmt.Errorf("no provider url name: %s", urlName)
	}
	return ctx, provider, err
}

//LoadProviderByDomain : load a provider by domain
func LoadProviderByDomain(ctx context.Context, db *DB, domain string, domainRoot string) (context.Context, *Provider, error) {
	var err error
	var provider *Provider
	if domainRoot == "" {
		whereStmt := "u.deleted=0 AND p.deleted=0 AND domain=?"
		ctx, provider, err = loadProvider(ctx, db, whereStmt, domain)
	} else {
		whereStmt := "u.deleted=0 AND p.deleted=0 AND (domain=? OR domain=?)"
		ctx, provider, err = loadProvider(ctx, db, whereStmt, domain, domainRoot)
	}
	if err != nil {
		return ctx, nil, err
	}
	if provider == nil {
		return ctx, nil, fmt.Errorf("no provider domain: %s", domain)
	}
	return ctx, provider, err
}

//SaveProvider : save a provider
func SaveProvider(ctx context.Context, db *DB, provider *Provider) (context.Context, error) {
	ctx, err := db.ProcessTx(ctx, "save provider", func(ctx context.Context, db *DB) (context.Context, error) {
		//default the provider id if necessary
		if provider.ID == nil {
			uuid, err := uuid.NewV4()
			if err != nil {
				return ctx, errors.Wrap(err, "new uuid provider")
			}
			provider.ID = &uuid
		}

		//create the url path if necessary
		if provider.URLName == "" {
			urlName := GenURLStringRndm(ProviderLengthRandomURLName)

			//check if the name is taken
			stmt := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE deleted=0 AND url_name=?", dbTableProvider)
			ctx, row, err := db.QueryRow(ctx, stmt, urlName)
			if err != nil {
				return ctx, errors.Wrap(err, "query row provider count")
			}

			//read the row
			var count int
			err = row.Scan(&count)
			if err != nil {
				return ctx, errors.Wrap(err, "check provider url name")
			}

			//if already used, add the time to the name to make it unique
			if count > 0 {
				urlName = fmt.Sprintf("%s%d", urlName, time.Now().Unix())
			}
			provider.URLName = urlName
		}

		//ensure the friendly name is url-safe
		if provider.URLNameFriendly != "" {
			provider.URLNameFriendly = GenURLString(provider.URLNameFriendly)
		}

		//json encode the provider data
		providerJSON, err := json.Marshal(provider)
		if err != nil {
			return ctx, errors.Wrap(err, "json provider")
		}

		//json encode the calendar data
		var calJSON []byte
		if provider.GoogleCalendarData != nil {
			calJSON, err = json.Marshal(provider.GoogleCalendarData)
			if err != nil {
				return ctx, errors.Wrap(err, "json google calendar")
			}
		}

		//save to the db
		stmt := fmt.Sprintf("INSERT INTO %s(id,user_id,url_name,url_name_friendly,domain,calendar_google_id,calendar_google_update,calendar_google_data,data) VALUES (UUID_TO_BIN(?),UUID_TO_BIN(?),?,?,?,?,?,?,?) ON DUPLICATE KEY UPDATE url_name=VALUES(url_name),url_name_friendly=VALUES(url_name_friendly),domain=VALUES(domain),calendar_google_id=VALUES(calendar_google_id),calendar_google_update=VALUES(calendar_google_update),calendar_google_data=VALUES(calendar_google_data),data=VALUES(data)", dbTableProvider)
		ctx, result, err := db.Exec(ctx, stmt, provider.ID, provider.User.ID, provider.URLName, provider.URLNameFriendly, provider.Domain, provider.GoogleCalendarID, provider.GoogleCalendarUpdate, calJSON, providerJSON)
		if err != nil {
			return ctx, errors.Wrap(err, "insert provider")
		}
		count, err := result.RowsAffected()
		if err != nil {
			return ctx, errors.Wrap(err, "insert provider rows affected")
		}

		//0 indicated no update, 1 an insert, 2 an update
		if count < 0 || count > 2 {
			return ctx, fmt.Errorf("unable to insert provider: %s", provider.User.ID)
		}

		//process the images
		ctx, err = ProcessImgSingle(ctx, db, provider.User.ID, provider.ID, provider.ID, ImgTypeBanner, provider.ImgBanner)
		if err != nil {
			return ctx, errors.Wrap(err, "insert provider process image banner")
		}
		ctx, err = ProcessImgSingle(ctx, db, provider.User.ID, provider.ID, provider.ID, ImgTypeLogo, provider.ImgLogo)
		if err != nil {
			return ctx, errors.Wrap(err, "insert provider process image logo")
		}

		//use the logo for the favicon
		if provider.ImgLogo != nil {
			provider.ImgFavIcon = &Img{
				Version: time.Now().Unix(),
			}
			provider.ImgFavIcon.SetFile(provider.ImgLogo.GetFile())
		} else {
			provider.ImgFavIcon = nil
		}
		ctx, err = ProcessImgSingle(ctx, db, provider.User.ID, provider.ID, provider.ID, ImgTypeFavIcon, provider.ImgFavIcon)
		if err != nil {
			return ctx, errors.Wrap(err, "insert provider process image favicon")
		}

		//process the provider user
		if provider.ProviderUser != nil {
			ctx, err = SaveProviderUser(ctx, db, provider.ProviderUser, false)
			if err != nil {
				return ctx, errors.Wrap(err, "save provider user")
			}
		}
		return ctx, nil
	})
	if err != nil {
		return ctx, errors.Wrap(err, "save provider")
	}
	return ctx, nil
}

//ListProviderCalendarsToProcessForGoogle : list provider calendars to process for Google
func ListProviderCalendarsToProcessForGoogle(ctx context.Context, db *DB, limit int) (context.Context, []*Provider, error) {
	ctx, logger := GetLogger(ctx)
	var err error
	var providers []*Provider
	ctx, err = db.ProcessTx(ctx, "list provider calendars process", func(ctx context.Context, db *DB) (context.Context, error) {
		//list the providers
		stmt := fmt.Sprintf("SELECT BIN_TO_UUID(id),calendar_google_id,calendar_google_data,data FROM %s WHERE deleted=0 AND calendar_google_processing=0 AND (calendar_google_id IS NULL OR calendar_google_update=1) ORDER BY created LIMIT %d", dbTableProvider, limit)
		ctx, rows, err := db.Query(ctx, stmt)
		if err != nil {
			return ctx, errors.Wrap(err, "select providers")
		}
		defer func() {
			err := rows.Close()
			if err != nil {
				logger.Warnw("rows close", "error", err)
			}
		}()

		//read the rows
		providers = make([]*Provider, 0, 2)
		var idStr string
		var calendarGoogleID sql.NullString
		var calendarGoogleData sql.NullString
		var dataStr string
		for rows.Next() {
			err := rows.Scan(&idStr, &calendarGoogleID, &calendarGoogleData, &dataStr)
			if err != nil {
				return ctx, errors.Wrap(err, "rows scan providers")
			}

			//parse the uuid
			id, err := uuid.FromString(idStr)
			if err != nil {
				return ctx, errors.Wrap(err, "parse uuid id")
			}

			//unmarshal the data
			var provider Provider
			err = json.Unmarshal([]byte(dataStr), &provider)
			if err != nil {
				return ctx, errors.Wrap(err, "unjson provider")
			}
			provider.ID = &id

			//check for google calendar data
			if calendarGoogleID.Valid {
				provider.GoogleCalendarID = &calendarGoogleID.String
			}
			if calendarGoogleData.Valid {
				var cal CalendarGoogle
				err = json.Unmarshal([]byte(calendarGoogleData.String), &cal)
				if err != nil {
					return ctx, errors.Wrap(err, "unjson calendar")
				}
				provider.GoogleCalendarData = &cal
			}
			providers = append(providers, &provider)
		}
		if len(providers) == 0 {
			return ctx, nil
		}

		//mark the providers as being processed
		MarkProvidersProcessing(ctx, db, providers)
		return ctx, nil
	})
	if err != nil {
		return ctx, nil, errors.Wrap(err, "list provider calendars process")
	}
	return ctx, providers, nil
}

//ListProviderServiceAreas : list services areas with providers
func ListProviderServiceAreas(ctx context.Context, db *DB) (context.Context, []*string, error) {
	ctx, logger := GetLogger(ctx)
	stmt := fmt.Sprintf("SELECT distinct p.data->>'$.ServiceArea' FROM %s p INNER JOIN %s u ON u.id=p.user_id AND u.deleted=0 WHERE p.deleted=0 ORDER BY p.data->>'$.ServiceArea'", dbTableProvider, dbTableUser)
	ctx, rows, err := db.Query(ctx, stmt)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "select provider service areas")
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			logger.Warnw("rows close", "error", err)
		}
	}()

	//process the service areas
	serviceAreas := make([]*string, 0, 2)
	for rows.Next() {
		var serviceArea string
		err := rows.Scan(&serviceArea)
		if err != nil {
			return ctx, nil, errors.Wrap(err, "rows scan provider service areas")
		}
		serviceAreas = append(serviceAreas, &serviceArea)
	}
	return ctx, serviceAreas, nil
}

//ListProviders : list all providers
func ListProviders(ctx context.Context, db *DB, serviceArea string, prev string, next string, limit int) (context.Context, []*Provider, string, string, error) {
	ctx, logger := GetLogger(ctx)
	const delimiter = "-"
	var prevStr string
	var nextStr string
	var stmt string
	args := make([]interface{}, 0, 3)

	//incporate a service area filter if necessary
	whereServiceAreaStmt := ""
	if serviceArea != "" {
		whereServiceAreaStmt = "AND p.data->>'$.ServiceArea'=?"
	}

	//make the appropriate query based on if paginating
	if prev != "" {
		//sort descending to walk backwards, though the results should be reversed
		stmt = fmt.Sprintf("SELECT BIN_TO_UUID(p.id),p.url_name,p.url_name_friendly,p.domain,p.data FROM %s p INNER JOIN %s u ON u.id=p.user_id AND u.deleted=0 AND u.test=0 WHERE p.deleted=0 AND (p.data->>'$.Name'<? OR (p.data->>'$.Name'=? AND p.id<UUID_TO_BIN(?))) %s ORDER BY p.data->>'$.Name' DESC,p.id DESC LIMIT %d", dbTableProvider, dbTableUser, whereServiceAreaStmt, limit)
		tokens := strings.Split(prev, delimiter)
		nameData, err := base64.RawStdEncoding.DecodeString(tokens[0])
		if err != nil {
			return ctx, nil, "", "", errors.Wrap(err, "invalid name token")
		}
		name := string(nameData)
		id := DecodeUUIDBase64(tokens[1])
		args = []interface{}{name, name, id}
	} else if next != "" {
		stmt = fmt.Sprintf("SELECT BIN_TO_UUID(p.id),p.url_name,p.url_name_friendly,p.domain,p.data FROM %s p INNER JOIN %s u ON u.id=p.user_id AND u.deleted=0 WHERE p.deleted=0 AND (p.data->>'$.Name'>? OR (p.data->>'$.Name'=? AND p.id>UUID_TO_BIN(?))) %s ORDER BY p.data->>'$.Name',p.id LIMIT %d", dbTableProvider, dbTableUser, whereServiceAreaStmt, limit)
		tokens := strings.Split(next, delimiter)
		nameData, err := base64.RawStdEncoding.DecodeString(tokens[0])
		if err != nil {
			return ctx, nil, "", "", errors.Wrap(err, "invalid name token")
		}
		name := string(nameData)
		id := DecodeUUIDBase64(tokens[1])
		args = []interface{}{name, name, id}
	} else {
		stmt = fmt.Sprintf("SELECT BIN_TO_UUID(p.id),p.url_name,p.url_name_friendly,p.domain,p.data FROM %s p INNER JOIN %s u ON u.id=p.user_id AND u.deleted=0 WHERE p.deleted=0 %s ORDER BY p.data->>'$.Name',p.id LIMIT %d", dbTableProvider, dbTableUser, whereServiceAreaStmt, limit)
	}

	//add the service area filter parameter if necessary
	if whereServiceAreaStmt != "" {
		args = append(args, serviceArea)
	}
	ctx, rows, err := db.Query(ctx, stmt, args...)
	if err != nil {
		return ctx, nil, "", "", errors.Wrap(err, "select providers")
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			logger.Warnw("rows close", "error", err)
		}
	}()

	//bookkeeping for previous and next pages
	var idFirst *uuid.UUID
	var idLast *uuid.UUID
	var keyFirst string
	var keyLast string

	//read the rows
	providers := make([]*Provider, 0, 2)
	var idStr string
	var urlName string
	var urlNameFriendly sql.NullString
	var domain sql.NullString
	var dataStr string
	for rows.Next() {
		err := rows.Scan(&idStr, &urlName, &urlNameFriendly, &domain, &dataStr)
		if err != nil {
			return ctx, nil, "", "", errors.Wrap(err, "rows scan providers")
		}

		//parse the id
		id, err := uuid.FromString(idStr)
		if err != nil {
			return ctx, nil, "", "", errors.Wrap(err, "parse uuid")
		}

		//unmarshal the data
		var provider Provider
		err = json.Unmarshal([]byte(dataStr), &provider)
		if err != nil {
			return ctx, nil, "", "", errors.Wrap(err, "unjson provider")
		}
		provider.ID = &id
		provider.URLName = urlName
		if urlNameFriendly.Valid {
			provider.URLNameFriendly = urlNameFriendly.String
		}
		if domain.Valid {
			provider.Domain = &domain.String
		}
		providers = append(providers, &provider)
	}

	//reverse the list if going to a previous page, since the sort was descending
	lenProviders := len(providers)
	if prev != "" {
		for i, j := 0, lenProviders-1; i < j; i, j = i+1, j-1 {
			providers[i], providers[j] = providers[j], providers[i]
		}
	}

	//determine the first and last entries to determine the previous and next links
	if lenProviders > 0 {
		idFirst = providers[0].ID
		idLast = providers[lenProviders-1].ID
		keyFirst = providers[0].Name
		keyLast = providers[lenProviders-1].Name
	}

	//check if there's a previous page
	if idFirst != nil && keyFirst != "" {
		stmt = fmt.Sprintf("SELECT COUNT(*) FROM %s p INNER JOIN %s u ON u.id=p.user_id AND u.deleted=0 WHERE p.deleted=0 AND (p.data->>'$.Name'<? OR (p.data->>'$.Name'=? AND p.id<UUID_TO_BIN(?))) %s ORDER BY p.data->>'$.Name' DESC,p.id DESC LIMIT %d", dbTableProvider, dbTableUser, whereServiceAreaStmt, limit)
		args = []interface{}{keyFirst, keyFirst, idFirst}
		if whereServiceAreaStmt != "" {
			args = append(args, serviceArea)
		}
		ctx, row, err := db.QueryRow(ctx, stmt, args...)
		if err != nil {
			return ctx, nil, "", "", errors.Wrap(err, "query row providers count prev")
		}
		var count int
		err = row.Scan(&count)
		if err != nil && err != sql.ErrNoRows {
			return ctx, nil, "", "", errors.Wrap(err, "select providers count prev")
		}
		if count > 0 {
			prevStr = fmt.Sprintf("%s%s%s", base64.RawStdEncoding.EncodeToString([]byte(keyFirst)), delimiter, EncodeUUIDBase64(idFirst))
		}
	}

	//check if there's a next page
	if idLast != nil && keyLast != "" {
		stmt = fmt.Sprintf("SELECT COUNT(*) FROM %s p INNER JOIN %s u ON u.id=p.user_id AND u.deleted=0 WHERE p.deleted=0 AND (p.data->>'$.Name'>? OR (p.data->>'$.Name'=? AND p.id>UUID_TO_BIN(?))) %s ORDER BY p.data->>'$.Name',p.id LIMIT %d", dbTableProvider, dbTableUser, whereServiceAreaStmt, limit)
		args = []interface{}{keyLast, keyLast, idLast}
		if whereServiceAreaStmt != "" {
			args = append(args, serviceArea)
		}
		ctx, row, err := db.QueryRow(ctx, stmt, args...)
		if err != nil {
			return ctx, nil, "", "", errors.Wrap(err, "query row providers count next")
		}
		var count int
		err = row.Scan(&count)
		if err != nil && err != sql.ErrNoRows {
			return ctx, nil, "", "", errors.Wrap(err, "select providers count next")
		}
		if count > 0 {
			nextStr = fmt.Sprintf("%s%s%s", base64.RawStdEncoding.EncodeToString([]byte(keyLast)), delimiter, EncodeUUIDBase64(idLast))
		}
	}
	return ctx, providers, prevStr, nextStr, nil
}

//SaveProviderData : save the provider data
func SaveProviderData(ctx context.Context, db *DB, provider *Provider) (context.Context, error) {
	//json encode the provider data
	providerJSON, err := json.Marshal(provider)
	if err != nil {
		return ctx, errors.Wrap(err, "json provider")
	}

	//save to the db
	stmt := fmt.Sprintf("UPDATE %s SET data=? WHERE id=UUID_TO_BIN(?)", dbTableProvider)
	ctx, result, err := db.Exec(ctx, stmt, providerJSON, provider.ID)
	if err != nil {
		return ctx, errors.Wrap(err, "insert provider")
	}
	_, err = result.RowsAffected()
	if err != nil {
		return ctx, errors.Wrap(err, "insert provider rows affected")
	}
	return ctx, nil
}

//MarkProvidersProcessing : mark providers as processing
func MarkProvidersProcessing(ctx context.Context, db *DB, providers []*Provider) (context.Context, error) {
	lenProviders := len(providers)
	if lenProviders == 0 {
		return ctx, fmt.Errorf("no providers to mark")
	}

	//prepare the ids
	args := make([]interface{}, lenProviders)
	for i, img := range providers {
		args[i] = img.ID.String()
	}

	//generate the list of parameters to use in the query
	paramsStr := fmt.Sprintf("(UUID_TO_BIN(?)%s)", strings.Repeat(",UUID_TO_BIN(?)", lenProviders-1))

	//mark the providers
	stmt := fmt.Sprintf("UPDATE %s SET calendar_google_processing=1,calendar_google_processing_time=CURRENT_TIMESTAMP() WHERE calendar_google_processing=0 AND id in %s", dbTableProvider, paramsStr)
	ctx, result, err := db.Exec(ctx, stmt, args...)
	if err != nil {
		return ctx, errors.Wrap(err, "mark providers processing")
	}
	count, err := result.RowsAffected()
	if err != nil {
		return ctx, errors.Wrap(err, "mark providers processing rows affected")
	}
	if int(count) != lenProviders {
		return ctx, fmt.Errorf("unable to mark providers processing: %d: %d", count, lenProviders)
	}
	return ctx, nil
}

//UpdateProviderCalendar : update the provider calendar information
func UpdateProviderCalendar(ctx context.Context, db *DB, provider *Provider) (context.Context, error) {
	//json encode the calendar data
	var err error
	var calJSON []byte
	if provider.GoogleCalendarData != nil {
		calJSON, err = json.Marshal(provider.GoogleCalendarData)
		if err != nil {
			return ctx, errors.Wrap(err, "json calendar")
		}
	}

	//update
	stmt := fmt.Sprintf("UPDATE %s SET calendar_google_id=?,calendar_google_update=0,calendar_google_processing=0,calendar_google_data=? WHERE id=UUID_TO_BIN(?)", dbTableProvider)
	ctx, result, err := db.Exec(ctx, stmt, provider.GoogleCalendarID, calJSON, provider.ID)
	if err != nil {
		return ctx, errors.Wrap(err, "update provider")
	}
	count, err := result.RowsAffected()
	if err != nil {
		return ctx, errors.Wrap(err, "update provider rows affected")
	}
	if count != 1 {
		return ctx, fmt.Errorf("unable to update provider: %s", provider.ID)
	}
	return ctx, nil
}

//CountProviders : count providers
func CountProviders(ctx context.Context, db *DB) (context.Context, int, error) {
	stmt := fmt.Sprintf("SELECT COUNT(*) FROM %s p INNER JOIN %s u ON u.id=p.user_id AND u.deleted=0 AND u.test=0 WHERE p.deleted=0", dbTableProvider, dbTableUser)
	ctx, row, err := db.QueryRow(ctx, stmt)
	if err != nil {
		return ctx, 0, errors.Wrap(err, "query row provider count")
	}

	//read the row
	var count int
	err = row.Scan(&count)
	if err != nil {
		if err == sql.ErrNoRows {
			return ctx, 0, nil
		}
		return ctx, 0, errors.Wrap(err, "select provider count")
	}
	return ctx, count, nil
}

//FindLatestProvider : find the latest provider create time
func FindLatestProvider(ctx context.Context, db *DB) (context.Context, *Provider, *time.Time, error) {
	stmt := fmt.Sprintf("SELECT p.data,p.created FROM %s p INNER JOIN %s u ON u.id=p.user_id AND u.deleted=0 AND u.test=0 WHERE p.deleted=0 ORDER BY p.created DESC LIMIT 1", dbTableProvider, dbTableUser)
	ctx, row, err := db.QueryRow(ctx, stmt)
	if err != nil {
		return ctx, nil, nil, errors.Wrap(err, "query row provider time create")
	}

	//read the row
	var dataStr string
	var t time.Time
	err = row.Scan(&dataStr, &t)
	if err != nil {
		if err == sql.ErrNoRows {
			return ctx, nil, nil, nil
		}
		return ctx, nil, nil, errors.Wrap(err, "select provider time create")
	}

	//unmarshal the data
	var provider Provider
	err = json.Unmarshal([]byte(dataStr), &provider)
	if err != nil {
		return ctx, nil, nil, errors.Wrap(err, "unjson provider")
	}
	return ctx, &provider, &t, nil
}

//ProviderUser : definition of a provider user
type ProviderUser struct {
	ID         *uuid.UUID        `json:"-"`
	ProviderID *uuid.UUID        `json:"-"`
	Login      string            `json:"-"`
	UserID     *uuid.UUID        `json:"-"`
	User       *User             `json:"-"`
	Schedule   *ProviderSchedule `json:"Schedule"`
}

//ProviderLoginExists : check if a login exists for a provider user
func ProviderLoginExists(ctx context.Context, db *DB, providerID *uuid.UUID, login string) (context.Context, bool, *uuid.UUID, error) {
	stmt := fmt.Sprintf("SELECT BIN_TO_UUID(u.id),BIN_TO_UUID(pu.id) FROM %s u LEFT JOIN %s pu ON pu.user_id=u.id AND pu.deleted=0 AND pu.provider_id=UUID_TO_BIN(?) WHERE u.deleted=0 AND u.email=?", dbTableUser, dbTableProviderUser)
	ctx, row, err := db.QueryRow(ctx, stmt, providerID, login)
	if err != nil {
		return ctx, false, nil, errors.Wrap(err, "query row provider user login")
	}

	//read the row
	var userIDStr sql.NullString
	var providerUserIDStr sql.NullString
	err = row.Scan(&userIDStr, &providerUserIDStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return ctx, false, nil, nil
		}
		return ctx, false, nil, errors.Wrap(err, "select provider user login")
	}

	//parse the uuid
	if !userIDStr.Valid {
		return ctx, false, nil, nil
	}
	userID, err := uuid.FromString(userIDStr.String)
	if err != nil {
		return ctx, providerUserIDStr.Valid, nil, errors.Wrap(err, "parse uuid provider user")
	}
	return ctx, providerUserIDStr.Valid, &userID, nil
}

//SaveProviderUser : save a provider user
func SaveProviderUser(ctx context.Context, db *DB, user *ProviderUser, updateServices bool) (context.Context, error) {
	ctx, err := db.ProcessTx(ctx, "save provider user", func(ctx context.Context, db *DB) (context.Context, error) {
		//default the user id if necessary
		if user.ID == nil {
			uuid, err := uuid.NewV4()
			if err != nil {
				return ctx, errors.Wrap(err, "new uuid provider user")
			}
			user.ID = &uuid
		}

		//json encode the user data
		userJSON, err := json.Marshal(user)
		if err != nil {
			return ctx, errors.Wrap(err, "json user")
		}

		//save the user
		stmt := fmt.Sprintf("INSERT INTO %s(id,provider_id,login,user_id,data) VALUES (UUID_TO_BIN(?),UUID_TO_BIN(?),?,UUID_TO_BIN(?),?) ON DUPLICATE KEY UPDATE user_id=VALUES(user_id),data=VALUES(data)", dbTableProviderUser)
		ctx, result, err := db.Exec(ctx, stmt, user.ID, user.ProviderID, user.Login, user.UserID, userJSON)
		if err != nil {
			return ctx, errors.Wrap(err, "insert provider user")
		}
		count, err := result.RowsAffected()
		if err != nil {
			return ctx, errors.Wrap(err, "insert provider user rows affected")
		}
		//0 indicated no update, 1 an insert, 2 an update
		if count < 0 || count > 2 {
			return ctx, fmt.Errorf("unable to insert provider user: %s: %s", user.ProviderID, user.Login)
		}

		//assign to services if necessary
		if updateServices {
			stmt = fmt.Sprintf("INSERT INTO %s(id,provider_id,service_id,provider_user_id) SELECT UUID_TO_BIN(UUID()),provider_id,id,UUID_TO_BIN(?) FROM %s WHERE provider_id=UUID_TO_BIN(?)", dbTableServiceProviderUser, dbTableService)
			ctx, _, err := db.Exec(ctx, stmt, user.ID, user.ProviderID)
			if err != nil {
				return ctx, errors.Wrap(err, "insert service provider user")
			}
		}
		return ctx, nil
	})
	if err != nil {
		return ctx, errors.Wrap(err, "save provider user")
	}
	return ctx, nil
}

//create the statement to load a provider user
func providerUserQueryCreate(whereStmt string, orderStmt string, limit int) string {
	if orderStmt == "" {
		orderStmt = "pu.login"
	}
	stmt := fmt.Sprintf("SELECT BIN_TO_UUID(pu.id),BIN_TO_UUID(pu.provider_id),pu.login,pu.data,BIN_TO_UUID(u.id),u.login,u.email,u.email_verified,u.disable_emails,u.is_oauth,u.token_zoom_data,u.data FROM %s pu LEFT JOIN %s u ON u.id=pu.user_id AND u.deleted=0 WHERE pu.deleted=0 AND %s ORDER BY %s", dbTableProviderUser, dbTableUser, whereStmt, orderStmt)
	if limit > 0 {
		stmt = fmt.Sprintf("%s LIMIT %d", stmt, limit)
	}
	return stmt
}

//parse a provider user
func providerUserQueryParse(rowFn ScanFn) (*ProviderUser, error) {
	//read the row
	var idStr string
	var providerIDStr string
	var login string
	var dataStr string
	var userIDStr sql.NullString
	var userLogin sql.NullString
	var email sql.NullString
	var emailVerifiedBit sql.NullString
	var disableEmailsBit sql.NullString
	var isOAuthBit sql.NullString
	var tokenZoomData sql.NullString
	var userDataStr sql.NullString
	err := rowFn(&idStr, &providerIDStr, &login, &dataStr, &userIDStr, &userLogin, &email, &emailVerifiedBit, &disableEmailsBit, &isOAuthBit, &tokenZoomData, &userDataStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, errors.Wrap(err, "select provider user")
	}

	//parse the uuid
	id, err := uuid.FromString(idStr)
	if err != nil {
		return nil, errors.Wrap(err, "parse uuid provider user")
	}
	providerID, err := uuid.FromString(providerIDStr)
	if err != nil {
		return nil, errors.Wrap(err, "parse uuid provider user provider id")
	}

	//unmarshal the data
	var providerUser ProviderUser
	err = json.Unmarshal([]byte(dataStr), &providerUser)
	if err != nil {
		return nil, errors.Wrap(err, "unjson provider user")
	}
	providerUser.ID = &id
	providerUser.ProviderID = &providerID
	providerUser.Login = login
	if userDataStr.Valid {
		var user User
		err = json.Unmarshal([]byte(userDataStr.String), &user)
		if err != nil {
			return nil, errors.Wrap(err, "unjson user")
		}
		if userIDStr.Valid {
			userID, err := uuid.FromString(userIDStr.String)
			if err != nil {
				return nil, errors.Wrap(err, "parse uuid user")
			}
			user.ID = &userID
			providerUser.UserID = &userID
		}
		if userLogin.Valid {
			user.Login = userLogin.String
		}
		if email.Valid {
			user.Email = email.String
		}
		if emailVerifiedBit.Valid {
			user.EmailVerified = emailVerifiedBit.String == "\x01"
		}
		if disableEmailsBit.Valid {
			user.DisableEmails = disableEmailsBit.String == "\x01"
		}
		if isOAuthBit.Valid {
			user.IsOAuth = isOAuthBit.String == "\x01"
		}

		//check for zoom token data
		if tokenZoomData.Valid {
			var token TokenZoom
			err = json.Unmarshal([]byte(tokenZoomData.String), &token)
			if err != nil {
				return nil, errors.Wrap(err, "unjson zoom token")
			}
			user.ZoomToken = &token
		}
		providerUser.User = &user
	}
	return &providerUser, nil
}

//load a provider user
func loadProviderUser(ctx context.Context, db *DB, whereStmt string, orderStmt string, limit int, args ...interface{}) (context.Context, *ProviderUser, error) {
	stmt := providerUserQueryCreate(whereStmt, orderStmt, limit)
	ctx, row, err := db.QueryRow(ctx, stmt, args...)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "query row provider user")
	}
	providerUser, err := providerUserQueryParse(row.Scan)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "parse provider user")
	}
	return ctx, providerUser, nil
}

//LoadProviderUserByUserID : load a provider based on the user id
func LoadProviderUserByUserID(ctx context.Context, db *DB, userID *uuid.UUID) (context.Context, *ProviderUser, error) {
	//assume only 1 provider
	whereStmt := "pu.user_id=UUID_TO_BIN(?)"
	orderStmt := "pu.updated DESC"
	ctx, user, err := loadProviderUser(ctx, db, whereStmt, orderStmt, 1, userID)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "load provider user")
	}
	return ctx, user, nil
}

//LoadProviderUserByProviderIDAndID : load a provider user by provider id and id
func LoadProviderUserByProviderIDAndID(ctx context.Context, db *DB, providerID *uuid.UUID, id *uuid.UUID) (context.Context, *ProviderUser, error) {
	whereStmt := "pu.provider_id=UUID_TO_BIN(?) AND pu.id=UUID_TO_BIN(?)"
	ctx, user, err := loadProviderUser(ctx, db, whereStmt, "", 0, providerID, id)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "load provider user")
	}
	return ctx, user, nil
}

//LoadProviderUserByProviderIDAndUserID : load a provider user by provider id and user id
func LoadProviderUserByProviderIDAndUserID(ctx context.Context, db *DB, providerID *uuid.UUID, id *uuid.UUID) (context.Context, *ProviderUser, error) {
	whereStmt := "pu.provider_id=UUID_TO_BIN(?) AND pu.user_id=UUID_TO_BIN(?)"
	ctx, user, err := loadProviderUser(ctx, db, whereStmt, "", 0, providerID, id)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "load provider user")
	}
	return ctx, user, nil
}

//ListProviderUsersByProviderID : list provider users by provider id
func ListProviderUsersByProviderID(ctx context.Context, db *DB, providerID *uuid.UUID, excludeNotRegistered bool) (context.Context, []*ProviderUser, error) {
	ctx, logger := GetLogger(ctx)
	whereStmt := "pu.provider_id=UUID_TO_BIN(?)"
	if excludeNotRegistered {
		whereStmt = fmt.Sprintf("%s AND u.id IS NOT NULL", whereStmt)
	}
	stmt := providerUserQueryCreate(whereStmt, "", 0)
	ctx, rows, err := db.Query(ctx, stmt, providerID)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "query row provider users")
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			logger.Warnw("rows close", "error", err)
		}
	}()

	//read the rows
	providerUsers := make([]*ProviderUser, 0, 2)
	for rows.Next() {
		providerUser, err := providerUserQueryParse(rows.Scan)
		if err != nil {
			return ctx, nil, errors.Wrap(err, "parse provider user")
		}
		if providerUser != nil {
			providerUsers = append(providerUsers, providerUser)
		}
	}
	return ctx, providerUsers, nil
}

//DeleteUserProvider : delete a provider user
func DeleteUserProvider(ctx context.Context, db *DB, providerID *uuid.UUID, id *uuid.UUID) (context.Context, error) {
	stmt := fmt.Sprintf("UPDATE %s SET deleted=1 WHERE provider_id=UUID_TO_BIN(?) AND id=UUID_TO_BIN(?)", dbTableProviderUser)
	ctx, result, err := db.Exec(ctx, stmt, providerID, id)
	if err != nil {
		return ctx, errors.Wrap(err, "delete provider user")
	}
	count, err := result.RowsAffected()
	if err != nil {
		return ctx, errors.Wrap(err, "delete provider user rows affected")
	}
	if count == 0 {
		return ctx, fmt.Errorf("delete provider user error: %s", id)
	}
	return ctx, nil
}
