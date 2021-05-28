package main

//form constants
const (
	LenCampaignInterests = 100
	LenCampaignLocations = 100
	LenCodeCoupon        = 10
	LenDescBook          = 200
	LenDescCoupon        = 200
	LenDescPayment       = 200
	LenDescProvider      = 1000
	LenDescProviderNote  = 1000
	LenDescSvc           = 200
	LenEducation         = 500
	LenExperience        = 500
	LenNoteSvc           = 200
	LenEmail             = 50
	LenLocation          = 100
	LenName              = 50
	LenTextContact       = 500
	LenTextCampaign      = 150
	LenTextFaq           = 500
	LenTextLong          = 1000000
	LenTextTestimonal    = 500
	LenURL               = 100
)

//EmailForm : form for an email
type EmailForm struct {
	Email string `validate:"required,email,max=50"` //LenEmail
}

//DateForm : form for a date
type DateForm struct {
	Date string `validate:"required,date"`
}

//TimeUnixForm : form for a UNIX time
type TimeUnixForm struct {
	Time string `validate:"required,timeUnix"`
}

//TimeZoneForm : form for a timezone
type TimeZoneForm struct {
	TimeZone string `validate:"required,timeZone"`
}

//NameForm : form for a name
type NameForm struct {
	Name string `validate:"required,min=2,max=50"` //LenName
}

//TimeDurationForm : form for a time duration
type TimeDurationForm struct {
	Start    string `json:"from" validate:"required,time"`
	Duration int    `json:"duration" validate:"required,durationSchedule"`
}

//DayScheduleForm : form for creating a day schedule
type DayScheduleForm struct {
	DayOfWeek     string              `json:"day" validate:"required,weekDay"`
	TimeDurations []*TimeDurationForm `json:"working_hours" validate:"omitempty,min=1,max=3,dive"`
	Available     bool                `json:"availability" validate:"durations=TimeDurations"`
}

//CampaignForm : form for adding a campaign
type CampaignForm struct {
	AgeMin    string `validate:"required,min=1,max=2,numeric,age"`
	AgeMax    string `validate:"required,min=1,max=2,numeric,age,ageGT=AgeMin"`
	Budget    string `validate:"required,min=1,max=5,numeric,budget"`
	Gender    string `validate:"required,min=1,max=5,gender"`
	Interests string `validate:"omitempty,min=2,max=100"` //LenCampaignInterests
	Locations string `validate:"omitempty,min=2,max=100"` //LenCampaignLocations
	Start     string `validate:"required,date"`
	End       string `validate:"required,date,campaignDateGT=Start"`
	ServiceID string `validate:"omitempty,uuid_rfc4122"`
	Text      string `validate:"omitempty,min=3,max=150"` //LenTextCampaign
}

//CampaignFacebookForm : form for adding Facebook information to a campaign
type CampaignFacebookForm struct {
	HasFacebookAdAccount bool
	HasFacebookPage      bool
	URLFacebook          string `validate:"required_with=HasFacebookPage,omitempty,min=6,max=100,url"` //LenURL
}

//CampaignStatusForm : form for the campaign status
type CampaignStatusForm struct {
	Status string `validate:"required,campaignStatus"`
}

//CampaignPaymentForm : form for a campaign payment
type CampaignPaymentForm struct {
	Price       string `validate:"required,min=1,max=5,numeric,price"`
	Description string `validate:"omitempty,min=3,max=200"` //LenDescPayment
}

//ClientDataForm : form for client data
type ClientDataForm struct {
	EmailForm
	NameForm
	Location string `validate:"omitempty,min=2,max=100"` //LenLocation
	Phone    string `validate:"omitempty,phone"`
}

//ClientForm : form for creating a client
type ClientForm struct {
	ClientDataForm
	TimeZoneForm
}

//ClientBookingDateTimeForm : form for a client booking
type ClientBookingDateTimeForm struct {
	TimeUnixForm
	TimeZoneForm
}

//ClientBookingForm : form for adding a client booking
type ClientBookingForm struct {
	ServiceID       string `validate:"required,uuid_rfc4122"`
	ClientID        string `validate:"required_without_all=Email Name Phone,omitempty,uuid_rfc4122"`
	Email           string `validate:"required_without=ClientID,omitempty,email,max=50"` //LenEmail
	Name            string `validate:"required_without=ClientID,omitempty,min=2,max=50"` //LenName
	Phone           string `validate:"required_with=EnablePhone,omitempty,phone"`
	ProviderNote    string `validate:"omitempty,max=10000"` //LenDescProviderNote
	ProviderNoteSet bool
	Freq            string `validate:"omitempty,recFreq"`
	FreqSet         bool
	Description     string `validate:"omitempty,max=200"` //LenDescBook
	DescriptionSet  bool
	Confirmed       bool
	ClientCreated   bool
	EnablePhone     bool
	Location        string `validate:"omitempty,min=2,max=100"` //LenLocation
	Code            string `validate:"omitempty,min=1,max=10"`  //LenCodeCoupon
	ClientBookingDateTimeForm
}

//ClientConfirmForm : form for a confirming a client
type ClientConfirmForm struct {
	Text string `validate:"omitempty,max=10000"`    //LenDescProviderNote
	Code string `validate:"omitempty,min=1,max=10"` //LenCodeCoupon
}

//ContactForm : form for a contact request
type ContactForm struct {
	ClientForm
	Text string `validate:"required,min=3,max=500"` //LenTextContact
}

//CouponForm : form for adding a coupon
type CouponForm struct {
	Type        string `validate:"required,couponType"`
	Code        string `validate:"required,min=1,max=10"` //LenCodeCoupon
	Value       string `validate:"required,min=1,max=5,numeric,price"`
	Start       string `validate:"required,date"`
	End         string `validate:"required,date,dateGT=Start"`
	Description string `validate:"omitempty,min=2,max=200"` //LenDescCoupon
	ServiceID   string `validate:"omitempty,uuid_rfc4122"`
	NewClients  bool
}

//CredentialsForm : form for credentials
type CredentialsForm struct {
	EmailForm
	PasswordForm
}

//FaqForm : form for creating a faq
type FaqForm struct {
	Question string `validate:"required,min=3,max=500"` //LenTextFaq
	Answer   string `validate:"required,min=3,max=500"` //LenTextFaq
}

//GoogleTrackingIDForm : form for a Google tracking id
type GoogleTrackingIDForm struct {
	ID string `validate:"required,min=6,max=16"`
}

//PaymentForm : form for a payment
type PaymentForm struct {
	EmailForm
	NameForm
	Phone           string `validate:"omitempty,phone"`
	Price           string `validate:"required,min=1,max=5,numeric,price"`
	Description     string `validate:"omitempty,min=3,max=200"` //LenDescPayment
	ClientInitiated bool
	DirectCapture   bool
}

//PasswordForm : form for a password
type PasswordForm struct {
	Password Secret `validate:"required,min=8,password"`
}

//ProviderForm : form for creating a provider
type ProviderForm struct {
	NameForm
	Description string `validate:"required,min=1,max=1000"` //LenDescProvider
	Education   string `validat:"omitempty,max=500"`        //LenEducation
	Experience  string `validat:"omitempty,max=500"`        //LenExperience
	ServiceArea string `validate:"required,min=2,max=50"`
	Location    string `validate:"omitempty,min=2,max=100"` //LenLocation
	URLName     string `validate:"omitempty,min=3,max=50"`  //LenName
}

//ProviderDomainForm : form for setting the domain for a provider
type ProviderDomainForm struct {
	Domain string `validate:"required,min=3,max=50,domain"` //LenName
}

//ProviderScheduleForm : form for creating a provider schedule
type ProviderScheduleForm struct {
	DaySchedules []*DayScheduleForm `json:"schedules" validate:"required,len=7"`
}

//ProviderLinksForm : form for provider links
type ProviderLinksForm struct {
	URLFacebook  string `validate:"omitempty,min=6,max=100,url"` //LenURL
	URLInstagram string `validate:"omitempty,min=6,max=100,url"` //LenURL
	URLLinkedIn  string `validate:"omitempty,min=6,max=100,url"` //LenURL
	URLTwitter   string `validate:"omitempty,min=6,max=100,url"` //LenURL
	URLWeb       string `validate:"omitempty,min=6,max=100,url"` //LenURL
}

//TextLongForm : form for long text
type TextLongForm struct {
	Text string `validate:"omitempty,max=1000000"` //LenTextLong
}

//ServiceForm : form for creating a service
type ServiceForm struct {
	ApptOnly bool
	NameForm
	Description        string `validate:"required,min=3,max=200"` //LenDescSvc
	Duration           string `validate:"required,min=1,max=5,numeric,durationSvc"`
	EnableZoom         bool
	Interval           string `validate:"required,min=1,max=2,numeric,svcInterval"`
	Location           string `validate:"svcLoc=LocationType,omitempty,min=2,max=100"` //LenLocation
	LocationType       string `validate:"required,svcLocType"`
	Note               string `validate:"omitempty,min=3,max=200"` //LenNoteSvc
	Padding            string `validate:"required,min=1,max=3,numeric,svcPadding"`
	PaddingInitial     string `validate:"required,min=1,max=2,numeric,svcPaddingInitial"`
	PaddingInitialUnit string `validate:"required,svcPaddingUnit"`
	Price              string `validate:"required,min=1,max=5,numeric,price"`
	PriceType          string `validate:"required,priceType"`
	URLVideo           string `validate:"omitempty,min=6,max=100,url,urlVideo"` //LenURL
}

//ScheduleForm : form for defining a schedule
type ScheduleForm struct {
	Time     string `validate:"required,min=1,time"`
	Duration string `validate:"required,min=1,max=4,numeric,durationScheduleStr"`
}

//TestimonialForm : form for creating a testimonial
type TestimonialForm struct {
	NameForm
	City string `validate:"omitempty,min=2,max=50"` //LenName
	Text string `validate:"required,min=3,max=500"` //LenTextTestimonal
}

//TutorForm : form for creating a tutor
type TutorForm struct {
	ProviderName     string `validate:"required,min=2,max=50"`   //LenName
	Biography        string `validate:"required,min=1,max=1000"` //LenDescProvider
	Education        string `validat:"omitempty,max=500"`        //LenEducation
	Experience       string `validat:"omitempty,max=500"`        //LenExperience
	ServiceArea      string `validate:"required,min=2,max=50"`
	Service          ServiceForm
	Time             string `validate:"required,min=1,time"`
	ScheduleDuration string `validate:"required,min=1,max=4,numeric,durationScheduleStr"`
}

//UserForm : form for specifying a user
type UserForm struct {
	FirstName string `validate:"required,min=2,max=50"` //LenName
	LastName  string `validate:"required,min=2,max=50"` //LenName
}

//UserUpdateForm : form for updating a user
type UserUpdateForm struct {
	DisablePhone bool
	Password     Secret `validate:"omitempty,min=8,password"`
	Phone        string `validate:"omitempty,phone"`
	TimeZoneForm
	UserForm
}

//UserSignUpForm : form for creating a user
type UserSignUpForm struct {
	EmailForm
	PasswordForm
	TimeZoneForm
	UserForm
}

//ZelleIDForm : form for a Zelle id
type ZelleIDForm struct {
	ZelleID string `validate:"required,phone|email,max=50"` //LenEmail
}
