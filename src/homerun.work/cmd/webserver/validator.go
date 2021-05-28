package main

import (
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/nyaruka/phonenumbers"
	"github.com/pkg/errors"
)

//validation constants
const (
	budgetMax                  = 300
	durationCampaignDaysMin    = 1 * 24 * time.Hour
	durationScheduleMinutesMin = 10
	durationScheduleMinutesMax = 1380  //23 hours
	durationServiceMinutesMin  = 0     //0 for variable
	durationServiceMinutesMax  = 10080 //7 days
	paddingInitialServiceMin   = 0
	paddingInitialServiceMax   = 24
	paddingServiceMinutesMin   = 0
	paddingServiceMinutesMax   = 480 //8 hours
	priceMin                   = 0
	priceMax                   = 50000
	unixTimeMin                = 1546300800 // 01/01/2019 12am
)

//InitValidator : initialize the validator
func InitValidator() *Validator {
	vdtor := &Validator{
		Validator: validator.New(),
	}

	//custom validations
	vdtor.Validator.RegisterValidation("age", validateFieldCampaignAge)
	vdtor.Validator.RegisterValidation("ageGT", validateFieldCampaignAgeGT)
	vdtor.Validator.RegisterValidation("budget", validateFieldBudget)
	vdtor.Validator.RegisterValidation("campaignDateGT", validateFieldCampaignDateGT)
	vdtor.Validator.RegisterValidation("campaignStatus", validateFieldCampaignStatus)
	vdtor.Validator.RegisterValidation("couponType", validateFieldCouponType)
	vdtor.Validator.RegisterValidation("date", validateFieldDate)
	vdtor.Validator.RegisterValidation("dateGT", validateFieldDateGT)
	vdtor.Validator.RegisterValidation("domain", validateFieldDomain)
	vdtor.Validator.RegisterValidation("durationSchedule", validateFieldDurationSchedule)
	vdtor.Validator.RegisterValidation("durationScheduleStr", validateFieldDurationScheduleStr)
	vdtor.Validator.RegisterValidation("durationSvc", validateFieldDurationService)
	vdtor.Validator.RegisterValidation("durations", validateFieldDurations)
	vdtor.Validator.RegisterValidation("gender", validateFieldGender)
	vdtor.Validator.RegisterValidation("password", validateFieldPassword)
	vdtor.Validator.RegisterValidation("phone", validateFieldPhone)
	vdtor.Validator.RegisterValidation("price", validateFieldPrice)
	vdtor.Validator.RegisterValidation("priceType", validateFieldPriceType)
	vdtor.Validator.RegisterValidation("recFreq", validateFieldRecurrenceFreq)
	vdtor.Validator.RegisterValidation("svcInterval", validateFieldServiceInterval)
	vdtor.Validator.RegisterValidation("svcLoc", validateFieldServiceLocation)
	vdtor.Validator.RegisterValidation("svcLocType", validateFieldServiceLocationType)
	vdtor.Validator.RegisterValidation("svcPadding", validateFieldServicePadding)
	vdtor.Validator.RegisterValidation("svcPaddingInitial", validateFieldServicePaddingInitial)
	vdtor.Validator.RegisterValidation("svcPaddingUnit", validateFieldServicePaddingUnit)
	vdtor.Validator.RegisterValidation("time", validateFieldTime)
	vdtor.Validator.RegisterValidation("timeGT", validateFieldTimeGT)
	vdtor.Validator.RegisterValidation("timeUnix", validateFieldTimeUnix)
	vdtor.Validator.RegisterValidation("timeZone", validateFieldTimeZone)
	vdtor.Validator.RegisterValidation("urlVideo", validateFieldURLVideo)
	vdtor.Validator.RegisterValidation("weekDay", validateFieldWeekDay)
	return vdtor
}

//Validator : validator utility
type Validator struct {
	Validator *validator.Validate
}

//Validate : validate a struct
func (v *Validator) Validate(data interface{}) (bool, []string, error) {
	//validate and check for any errors, reading explicit validations errors and returning
	//a list of fields that failed or the error
	err := v.Validator.Struct(data)
	if err != nil {
		validationErrs, ok := err.(validator.ValidationErrors)
		if !ok {
			return false, nil, errors.Wrap(err, "validate")
		}
		fields := make([]string, 0)
		for _, validationErr := range validationErrs {
			fields = append(fields, validationErr.Field())
		}
		return false, fields, nil
	}
	return true, nil, nil
}

//validate a field as a budget value
func validateFieldBudget(fl validator.FieldLevel) bool {
	v, err := strconv.ParseFloat(fl.Field().String(), 32)
	if err != nil {
		return false
	}
	if v < CampaignBudgetMin || v > budgetMax {
		return false
	}
	return true
}

//validate a field as an age value
func validateFieldCampaignAge(fl validator.FieldLevel) bool {
	v, err := strconv.ParseInt(fl.Field().String(), 10, 32)
	if err != nil {
		return false
	}
	if v < AgeMin || v > AgeMax {
		return false
	}
	return true
}

//validate a field as an age that is greater than the parameter field
func validateFieldCampaignAgeGT(fl validator.FieldLevel) bool {
	fieldAge, err := strconv.ParseInt(fl.Field().String(), 10, 32)
	if err != nil {
		return false
	}

	//read the parameter field
	param, _, _, ok := fl.GetStructFieldOK2()
	if !ok {
		return false
	}
	paramAge, err := strconv.ParseInt(param.String(), 10, 32)
	if err != nil {
		return false
	}
	return fieldAge > paramAge
}

//validate a field as a campaign date that is greater than the parameter field
func validateFieldCampaignDateGT(fl validator.FieldLevel) bool {
	fieldTime := ParseDateUTC(fl.Field().String())
	if fieldTime.IsZero() {
		return false
	}

	//read the parameter field
	param, _, _, ok := fl.GetStructFieldOK2()
	if !ok {
		return false
	}
	paramTime := ParseDateUTC(param.String())
	if paramTime.IsZero() {
		return false
	}
	paramTime = paramTime.Add(durationCampaignDaysMin)
	return fieldTime.After(paramTime) || fieldTime.Equal(paramTime)
}

//validate a field as a campaign status
func validateFieldCampaignStatus(fl validator.FieldLevel) bool {
	v := ParseCampaignStatus(fl.Field().String())
	return v != nil
}

//validate a field as a coupon type
func validateFieldCouponType(fl validator.FieldLevel) bool {
	v := ParseCouponType(fl.Field().String())
	return v != nil
}

//validate a field as a date
func validateFieldDate(fl validator.FieldLevel) bool {
	v := ParseDateUTC(fl.Field().String())
	if v.IsZero() {
		return false
	}
	return true
}

//validate a field as a date that is greater than the parameter field
func validateFieldDateGT(fl validator.FieldLevel) bool {
	fieldTime := ParseDateUTC(fl.Field().String())
	if fieldTime.IsZero() {
		return false
	}

	//read the parameter field
	param, _, _, ok := fl.GetStructFieldOK2()
	if !ok {
		return false
	}
	paramTime := ParseDateUTC(param.String())
	if paramTime.IsZero() {
		return false
	}
	return fieldTime.After(paramTime)
}

//validate a field as a domain
func validateFieldDomain(fl validator.FieldLevel) bool {
	var o sync.Once
	var regex *regexp.Regexp
	o.Do(func() {
		regex = regexp.MustCompile("^([a-zA-Z0-9]+(-[a-zA-Z0-9]+)*\\.)+[a-zA-Z]{2,}$")
	})
	if !regex.MatchString(fl.Field().String()) {
		return false
	}
	return true
}

//validate a field as a schedule duration
func validateFieldDurationSchedule(fl validator.FieldLevel) bool {
	v := fl.Field().Int()
	if v < durationScheduleMinutesMin {
		return false
	}
	if v > durationScheduleMinutesMax {
		return false
	}
	return true
}

//validate a field as a schedule duration string
func validateFieldDurationScheduleStr(fl validator.FieldLevel) bool {
	v, err := strconv.ParseInt(fl.Field().String(), 10, 32)
	if err != nil {
		return false
	}
	if v < durationScheduleMinutesMin {
		return false
	}
	if v > durationScheduleMinutesMax {
		return false
	}
	return true
}

//validate a field as a service duration
func validateFieldDurationService(fl validator.FieldLevel) bool {
	v, err := strconv.ParseInt(fl.Field().String(), 10, 32)
	if err != nil {
		return false
	}
	if v < durationServiceMinutesMin {
		return false
	}
	if v > durationServiceMinutesMax {
		return false
	}
	return true
}

//validate a field and check if the flag is set and the specified other fields are set
func validateFieldDurations(fl validator.FieldLevel) bool {
	v := fl.Field().Bool()
	if v {
		//read the parameter and extract the other fields that were specified
		param := fl.Param()
		fields := strings.Fields(param)
		for _, field := range fields {
			//check if the field is set
			structField, _, _, ok := fl.GetStructFieldOKAdvanced2(fl.Parent(), field)
			if !ok || structField.IsZero() {
				return false
			}
		}
	}
	return true
}

//validate a field as a gender
func validateFieldGender(fl validator.FieldLevel) bool {
	v := ParseGender(fl.Field().String())
	return v != ""
}

//validate a field as a password
func validateFieldPassword(fl validator.FieldLevel) bool {
	return true
}

//validate a field as a phone number
func validateFieldPhone(fl validator.FieldLevel) bool {
	//validate the phone number
	phone, err := phonenumbers.Parse(fl.Field().String(), defaultCountry)
	if err != nil {
		return false
	}
	if !phonenumbers.IsValidNumber(phone) {
		return false
	}
	return true
}

//validate a field as a price
func validateFieldPrice(fl validator.FieldLevel) bool {
	v, err := strconv.ParseFloat(fl.Field().String(), 32)
	if err != nil {
		return false
	}
	if v < priceMin || v > priceMax {
		return false
	}
	return true
}

//validate a field as a price type
func validateFieldPriceType(fl validator.FieldLevel) bool {
	v := ParsePriceType(fl.Field().String())
	return v != ""
}

//validate a field as a recurrence frequency
func validateFieldRecurrenceFreq(fl validator.FieldLevel) bool {
	s := fl.Field().String()
	v := ParseRecurrenceFreq(&s)
	return v != nil
}

//validate a field as service interval
func validateFieldServiceInterval(fl validator.FieldLevel) bool {
	v := ParseServiceInterval(fl.Field().String())
	if v == nil {
		return false
	}
	return true
}

//validate a field as a service location based on the location type
func validateFieldServiceLocation(fl validator.FieldLevel) bool {
	//read the parameter field
	param, _, _, ok := fl.GetStructFieldOK2()
	if !ok {
		return false
	}

	//check if a location is required
	paramLoc := ParseServiceLocation(param.String())
	if paramLoc == nil {
		return false
	}
	if paramLoc.Type.IsLocationProvider() {
		s := fl.Field().String()
		len := len(s)
		if len < 2 || len > 100 { //LenLocation
			return false
		}
		return true
	}
	return true
}

//validate a field as a service location type
func validateFieldServiceLocationType(fl validator.FieldLevel) bool {
	v := ParseServiceLocation(fl.Field().String())
	return v != nil
}

//validate a field as service padding
func validateFieldServicePadding(fl validator.FieldLevel) bool {
	v, err := strconv.ParseInt(fl.Field().String(), 10, 32)
	if err != nil {
		return false
	}
	if v < paddingServiceMinutesMin {
		return false
	}
	if v > paddingServiceMinutesMax {
		return false
	}
	return true
}

//validate a field as initial service padding
func validateFieldServicePaddingInitial(fl validator.FieldLevel) bool {
	v, err := strconv.ParseInt(fl.Field().String(), 10, 32)
	if err != nil {
		return false
	}
	if v < paddingInitialServiceMin {
		return false
	}
	if v > paddingInitialServiceMax {
		return false
	}
	return true
}

//validate a field as a service padding unit
func validateFieldServicePaddingUnit(fl validator.FieldLevel) bool {
	v := ParsePaddingUnit(fl.Field().String())
	return v != ""
}

//validate a field as a time
func validateFieldTime(fl validator.FieldLevel) bool {
	v := ParseTimeUTC(fl.Field().String(), time.Now())
	if v.IsZero() {
		return false
	}
	return true
}

//validate a field as a time that is greater than the parameter field
func validateFieldTimeGT(fl validator.FieldLevel) bool {
	fieldTime := ParseTimeUTC(fl.Field().String(), time.Now())
	if fieldTime.IsZero() {
		return false
	}

	//read the parameter field
	param, _, _, ok := fl.GetStructFieldOK2()
	if !ok {
		return false
	}
	paramTime := ParseTimeUTC(param.String(), time.Now())
	if paramTime.IsZero() {
		return false
	}
	return fieldTime.After(paramTime)
}

//validate a field as a unix time
func validateFieldTimeUnix(fl validator.FieldLevel) bool {
	v := ParseTimeUnixUTC(fl.Field().String())
	if v.IsZero() {
		return false
	}
	return v.Unix() > unixTimeMin
}

//validate a field as a time zone
func validateFieldTimeZone(fl validator.FieldLevel) bool {
	_, err := time.LoadLocation(fl.Field().String())
	if err != nil {
		return false
	}
	return true
}

//validate a field as a video url
func validateFieldURLVideo(fl validator.FieldLevel) bool {
	_, ok := ExtractURLYouTubeVideoID(fl.Field().String())
	return ok
}

//validate a field as a week day
func validateFieldWeekDay(fl validator.FieldLevel) bool {
	_, ok := ParseWeekDay(fl.Field().String())
	if !ok {
		return false
	}
	return true
}
