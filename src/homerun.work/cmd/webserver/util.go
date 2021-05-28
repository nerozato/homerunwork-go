package main

import (
	"encoding/base64"
	"fmt"
	"math"
	"math/rand"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gofrs/uuid"
	"github.com/nyaruka/phonenumbers"
)

//layouts for formatting and parsing time
const (
	layoutDate      = "01/02/2006"
	layoutDateLong  = "January 02, 2006"
	layoutDateTime  = "01/02/2006 3:04 PM"
	layoutMonthLong = "January 2006"
	layoutTime      = "3:04 PM"
)

//default country code
const defaultCountry = "US"

//MaxTime : maximum time
var MaxTime = time.Date(9999, 1, 1, 0, 0, 0, 0, time.UTC)

//TimeDuration : definition of a time duration
type TimeDuration struct {
	Start    time.Time
	Duration int //minutes
}

//GetEnd : get the end date
func (t *TimeDuration) GetEnd() time.Time {
	end := t.Start.Add(time.Duration(t.Duration) * time.Minute)
	return end
}

//FormatTimePeriod : format the time duration as a period
func (t *TimeDuration) FormatTimePeriod(timeZone string) string {
	return fmt.Sprintf("%s-%s", FormatTimeLocal(t.Start, timeZone), FormatTimeLocal(t.GetEnd(), timeZone))
}

//DaySchedule : definition of time durations for a specific day
type DaySchedule struct {
	DayOfWeek     time.Weekday    `json:"DayOfWeek"`
	TimeDurations []*TimeDuration `json:"TimeDurations"`
	Unavailable   bool            `json:"Unavailable"`
	TimePeriods   []*TimePeriod   `json:"TimePeriods"`
}

//TimePeriod : definition of a time period
type TimePeriod struct {
	Start       time.Time
	End         time.Time
	Unavailable bool
	Hidden      bool
}

//IsOverlap : check if the incoming time period overlaps
func (t *TimePeriod) IsOverlap(start time.Time, end time.Time) bool {
	return t.Start.Before(end) && t.End.After(start)
}

//FormatTimesLocal : format the start and end time
func (t *TimePeriod) FormatTimesLocal(timeZone string) string {
	return fmt.Sprintf("%s-%s", t.FormatStartLocal(timeZone), t.FormatEndLocal(timeZone))
}

//FormatPeriodLocal : format the time period
func (t *TimePeriod) FormatPeriodLocal(isAppt bool, timeZone string) string {
	if t.Unavailable {
		return fmt.Sprintf("%s (%s)", t.FormatStartLocal(timeZone), GetMsgText(MsgUnavailable))
	}
	if isAppt {
		return fmt.Sprintf("%s - %s", t.FormatStartLocal(timeZone), t.FormatEndLocal(timeZone))
	}
	return t.FormatStartLocal(timeZone)
}

//FormatStartLocal : format the start time
func (t *TimePeriod) FormatStartLocal(timeZone string) string {
	return FormatTimeLocal(t.Start, timeZone)
}

//FormatEndLocal : format the end time
func (t *TimePeriod) FormatEndLocal(timeZone string) string {
	return FormatTimeLocal(t.End, timeZone)
}

//FormatStartUnix : format the start time as a Unix time
func (t *TimePeriod) FormatStartUnix() string {
	return strconv.FormatInt(t.Start.Unix(), 10)
}

//location cache key
type locationKey string

//location cache
var locationCache map[locationKey]*time.Location = make(map[locationKey]*time.Location, 2)

//GetLocation : get a location, caching used locations
func GetLocation(timeZone string) *time.Location {
	_, logger := GetLogger(nil)
	if timeZone == "" {
		return time.UTC
	}
	key := locationKey(timeZone)

	//probe the cache
	loc, ok := locationCache[key]
	if ok {
		return loc
	}

	//load the location
	loc, err := time.LoadLocation(timeZone)
	if err != nil {
		logger.Warnw("invalid location", "location", timeZone)
		return time.UTC
	}

	//cache the location
	locationCache[key] = loc
	return loc
}

//ParseDateLocal : parse a string as a local date
func ParseDateLocal(in string, timeZone string) time.Time {
	loc := GetLocation(timeZone)
	v, err := time.ParseInLocation(layoutDate, in, loc)
	if err != nil {
		return time.Time{}
	}
	return v
}

//ParseDateUTC : parse a string as a UTC date
func ParseDateUTC(in string) time.Time {
	v, err := time.Parse(layoutDate, in)
	if err != nil {
		return time.Time{}
	}
	return v
}

//ParseTimeLocal : parse a string as a local time
func ParseTimeLocal(in string, ref time.Time, timeZone string) time.Time {
	//parse the time, incuding the year to avoid year 0 issues
	loc := GetLocation(timeZone)
	v, err := time.ParseInLocation(layoutDateTime, fmt.Sprintf("%s %s", FormatDateLocal(ref, timeZone), strings.ToUpper(in)), loc)
	if err != nil {
		return time.Time{}
	}
	return v
}

//ParseTimeUTC : parse a string as a UTC time
func ParseTimeUTC(in string, ref time.Time) time.Time {
	//parse the time, incuding the year to avoid year 0 issues
	v, err := time.Parse(layoutDateTime, fmt.Sprintf("%s %s", FormatDateUTC(ref), strings.ToUpper(in)))
	if err != nil {
		return time.Time{}
	}
	return v
}

//ParseTimeLocalAsUTC : parse a string as a local time and return the UTC time
func ParseTimeLocalAsUTC(in string, ref time.Time, timeZone string) *time.Time {
	//parse the time, incuding the year to avoid year 0 issues
	loc := GetLocation(timeZone)
	v, err := time.ParseInLocation(layoutDateTime, fmt.Sprintf("%s %s", FormatDateLocal(ref, timeZone), strings.ToUpper(in)), loc)
	if err != nil {
		return nil
	}
	utc := v.UTC()
	return &utc
}

//ParseDateTimeRFC3339 : parse a string as an ISO8601 date/time
func ParseDateTimeRFC3339(in string) time.Time {
	v, err := time.Parse(time.RFC3339, strings.ToUpper(in))
	if err != nil {
		return time.Time{}
	}
	return v
}

//ParseDateTimeUTC : parse a string as a UTC date/time
func ParseDateTimeUTC(in string) time.Time {
	v, err := time.Parse(layoutDateTime, strings.ToUpper(in))
	if err != nil {
		return time.Time{}
	}
	return v
}

//FormatDateLocal : format a date in local time as a string
func FormatDateLocal(t time.Time, timeZone string) string {
	loc := GetLocation(timeZone)
	localDate := t.In(loc)
	return localDate.Format(layoutDate)
}

//FormatDateUTC : format a date in UTC as a string
func FormatDateUTC(t time.Time) string {
	return t.Format(layoutDate)
}

//FormatDateLongLocal : format a date in local time using the long format as a string
func FormatDateLongLocal(t time.Time, timeZone string) string {
	loc := GetLocation(timeZone)
	localDate := t.In(loc)
	return localDate.Format(layoutDateLong)
}

//FormatDateTimeLocal : format a date/time in local time as a string
func FormatDateTimeLocal(t time.Time, timeZone string) string {
	loc := GetLocation(timeZone)
	localTime := t.In(loc)
	return localTime.Format(layoutDateTime)
}

//FormatMonthLongLocal : format a month in local time as a string
func FormatMonthLongLocal(t time.Time, timeZone string) string {
	loc := GetLocation(timeZone)
	localDate := t.In(loc)
	return localDate.Format(layoutMonthLong)
}

//FormatTimeLocal : format a time in local time as a string
func FormatTimeLocal(t time.Time, timeZone string) string {
	loc := GetLocation(timeZone)
	localTime := t.In(loc)
	return localTime.Format(layoutTime)
}

//ParseTimeUnixUTC : parse a Unix time as UTC
func ParseTimeUnixUTC(in string) time.Time {
	i, err := strconv.ParseInt(in, 10, 64)
	if err != nil {
		return time.Time{}
	}
	v := time.Unix(i, 0)
	return v
}

//ParseTimeUnixLocal : parse a Unix time as a local time
func ParseTimeUnixLocal(in string, timeZone string) time.Time {
	t := ParseTimeUnixUTC(in)
	return GetTimeLocal(t, timeZone)
}

//GetTimeLocal : return as a local time
func GetTimeLocal(t time.Time, timeZone string) time.Time {
	loc := GetLocation(timeZone)
	t = t.In(loc)
	return t
}

//AdjTimes : adjust a time span forward to reflect the reference date
func AdjTimes(ref time.Time, from time.Time, to time.Time) (time.Time, time.Time) {
	//use the timezone
	from = from.In(ref.Location())
	to = to.In(ref.Location())

	//find the time difference between the from-time's date and the reference date by
	//adjusting each to be the beginning-of-day
	refBOD := GetBeginningOfDay(ref)
	fromBOD := GetBeginningOfDay(from)
	diff := refBOD.Sub(fromBOD)
	fromAdj := from.Add(diff)
	toAdj := to.Add(diff)
	return fromAdj, toAdj
}

//CheckTimeIn : check if a time is withing a span, inclusive of the from and to time
func CheckTimeIn(t time.Time, from time.Time, to time.Time) bool {
	return (t.Equal(from) || t.After(from)) && (t.Equal(to) || t.Before(to))
}

//GetBeginningOfDay : return the time as beginning-of-day
func GetBeginningOfDay(t time.Time) time.Time {
	y, m, d := t.Date()
	timeBOD := time.Date(y, m, d, 0, 0, 0, 0, t.Location())
	return timeBOD
}

//GetEndOfDay : return the time as end-of-day
func GetEndOfDay(t time.Time) time.Time {
	y, m, d := t.Date()
	timeBOD := time.Date(y, m, d, 23, 59, 59, 999999999, t.Location())
	return timeBOD
}

//GetStartOfMonth : get the start of the month
func GetStartOfMonth(t time.Time) time.Time {
	timeBOM := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
	return timeBOM
}

//GetStartOfMonthNext : get the start of the next month
func GetStartOfMonthNext(t time.Time) time.Time {
	//find the start of the specified month
	timeBOM := GetStartOfMonth(t)

	//adjust to next month
	nextBOM := timeBOM.AddDate(0, 1, 0)
	return nextBOM
}

//GetStartOfMonthPrev : get the start of the previous month
func GetStartOfMonthPrev(t time.Time) time.Time {
	//find the start of the specified month
	timeBOM := GetStartOfMonth(t)

	//adjust to previous month
	prevBOM := timeBOM.AddDate(0, -1, 0)
	return prevBOM
}

//GetStartOfWeek : get the start of the week, assuming Mon. is the start
func GetStartOfWeek(t time.Time) time.Time {
	//find the day of week, making Sun. the last day
	weekDay := time.Duration(t.Weekday())
	if weekDay == 0 {
		weekDay = 7
	}

	//adjust for beginning-of-day
	timeBOD := GetBeginningOfDay(t)

	//find the beginning-of-week
	timeBOW := timeBOD.Add(-1 * (weekDay - 1) * 24 * time.Hour)
	return timeBOW
}

//GetStartOfWeekNext : get the start of the next week, assuming Mon. is the start
func GetStartOfWeekNext(t time.Time) time.Time {
	//find the start of the specified week
	timeBOW := GetStartOfWeek(t)

	//adjust to next week
	nextBOW := timeBOW.Add(7 * 24 * time.Hour)
	return nextBOW
}

//GetStartOfWeekPrev : get the start of the previous week, assuming Mon. is the start
func GetStartOfWeekPrev(t time.Time) time.Time {
	//find the start of the specified week
	timeBOW := GetStartOfWeek(t)

	//adjust to previous week
	nextBOW := timeBOW.Add(-7 * 24 * time.Hour)
	return nextBOW
}

//DaysOfWeek : list of days of the week
var DaysOfWeek map[string]time.Weekday = map[string]time.Weekday{
	time.Sunday.String():    time.Sunday,
	time.Monday.String():    time.Monday,
	time.Tuesday.String():   time.Tuesday,
	time.Wednesday.String(): time.Wednesday,
	time.Thursday.String():  time.Thursday,
	time.Friday.String():    time.Friday,
	time.Saturday.String():  time.Saturday,
}

//ParseWeekDay : parse the day of the week
func ParseWeekDay(in string) (time.Weekday, bool) {
	dayOfWeek, ok := DaysOfWeek[in]
	if !ok {
		return 0, false
	}
	return dayOfWeek, true
}

//GenURLString : generate a string using the input that is URL path-safe
func GenURLString(in string) string {
	//force lower-case
	in = strings.ToLower(in)

	//convert any non-alphanumeric characters to be a "-"
	var o sync.Once
	var rex *regexp.Regexp
	o.Do(func() {
		rex = regexp.MustCompile(`[^A-Za-z0-9-]`)
	})
	s := rex.ReplaceAllLiteralString(in, "-")

	//return a url path-safe string
	return url.PathEscape(s)
}

//characters to use for the random string
const rndmLetters = "abcdefghijklmnopqrstuvwxyz0123456789"

//GenURLStringRndm : generate a random string that is URL path-safe
func GenURLStringRndm(length int) string {
	//seed the random number generator
	var o sync.Once
	o.Do(func() {
		rand.Seed(time.Now().UnixNano())
	})

	//generate the random string
	b := make([]byte, length)
	for i := range b {
		b[i] = rndmLetters[rand.Intn(len(rndmLetters))]
	}
	return string(b)
}

//Min : find the minimum of two integers
func Min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

//Max : find the maximimum of two integers
func Max(x, y int) int {
	if x > y {
		return x
	}
	return y
}

//FormatPhone : format a phone number
func FormatPhone(phoneStr string) string {
	if phoneStr == "" {
		return ""
	}

	//validate the phone number
	phone, err := phonenumbers.Parse(phoneStr, defaultCountry)
	if err != nil {
		return phoneStr
	}

	//ensure the phone number is formatted to e.164
	phoneFormatted := phonenumbers.Format(phone, phonenumbers.E164)
	return phoneFormatted
}

//FormatElapsedMS : format the elapsed time since the start as milliseconds
func FormatElapsedMS(start time.Time) string {
	durationMS := time.Since(start).Seconds() * 1000
	return fmt.Sprintf("%.6f", durationMS)
}

//FormatFloat : format the float
func FormatFloat(val float32) string {
	var str string
	if val == float32(math.Trunc(float64(val))) {
		str = fmt.Sprintf("%.0f", val)
	} else {
		str = fmt.Sprintf("%.2f", val)
	}
	return str
}

//FormatPrice : format the price
func FormatPrice(price float32) string {
	return fmt.Sprintf("$%s", FormatFloat(price))
}

//FileNoExt : filename with no extension
func FileNoExt(file string) string {
	ext := path.Ext(file)
	return strings.TrimSuffix(file, ext)
}

//EncodeUUIDBase64 : encode a UUID in base64
func EncodeUUIDBase64(id *uuid.UUID) string {
	if id == nil {
		return ""
	}
	return base64.RawStdEncoding.EncodeToString(id.Bytes())
}

//DecodeUUIDBase64 : parse a base64 string to extract a UUID
func DecodeUUIDBase64(in string) *uuid.UUID {
	if in == "" {
		return nil
	}
	str, err := base64.RawStdEncoding.DecodeString(in)
	if err != nil {
		return nil
	}
	id := uuid.FromBytesOrNil([]byte(str))
	if id != uuid.Nil {
		return &id
	}
	return nil
}

//ConvertNewLinesToBreaks : convert newlines to breaks
func ConvertNewLinesToBreaks(in string) string {
	return strings.Replace(in, "\n", "<br>", -1)
}

//IsMappable : check if a string is mappable
func IsMappable(in string) bool {
	return false
}
