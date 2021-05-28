package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/url"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

//provider bucketing
const (
	providerBucket1Size = 3
	providerBucket2Size = 7 - providerBucket1Size
)

//BookingFilter : booking filter
type BookingFilter string

//types of filters for bookings
var (
	BookingFilterAll      BookingFilter = "all"
	BookingFilterInvoiced BookingFilter = "invoiced"
	BookingFilterNew      BookingFilter = "new"
	BookingFilterPaid     BookingFilter = "paid"
	BookingFilterPast     BookingFilter = "past"
	BookingFilterUnPaid   BookingFilter = "unpaid"
	BookingFilterUpcoming BookingFilter = "upcoming"
)

//ParseBookingFilter : parse a string as a booking filter
func ParseBookingFilter(in string) (BookingFilter, error) {
	switch in {
	case string(BookingFilterAll):
		return BookingFilterAll, nil
	case string(BookingFilterInvoiced):
		return BookingFilterInvoiced, nil
	case string(BookingFilterNew):
		return BookingFilterNew, nil
	case string(BookingFilterPaid):
		return BookingFilterPaid, nil
	case string(BookingFilterPast):
		return BookingFilterPast, nil
	case string(BookingFilterUnPaid):
		return BookingFilterUnPaid, nil
	case string(BookingFilterUpcoming):
		return BookingFilterUpcoming, nil
	default:
		return "", fmt.Errorf("invalid filter: %s", in)
	}
}

//PaymentFilter : payment filter
type PaymentFilter string

//types of filters for payments
var (
	PaymentFilterAll    PaymentFilter = "all"
	PaymentFilterUnPaid PaymentFilter = "unpaid"
)

//ParsePaymentFilter : parse a string as a payment filter
func ParsePaymentFilter(in string) (PaymentFilter, error) {
	switch in {
	case string(PaymentFilterAll):
		return PaymentFilterAll, nil
	case string(PaymentFilterUnPaid):
		return PaymentFilterUnPaid, nil
	default:
		return "", fmt.Errorf("invalid filter: %s", in)
	}
}

//ImgViewTypes : image view types
var ImgViewTypes = struct {
	TypeLogo   string
	TypeBanner string
	TypeImg    string
}{
	TypeLogo:   "logo",
	TypeBanner: "banner",
	TypeImg:    "img",
}

//PaymentTypes : payment types
var PaymentTypes = struct {
	TypePayPal string
	TypeStripe string
	TypeZelle  string
}{
	TypePayPal: "paypal",
	TypeStripe: "stripe",
	TypeZelle:  "zelle",
}

//provider schedule for a day of week
type providerSchedule struct {
	DayOfWeek string
	Times     []*TimePeriod
}

//provider wrapper used for the ui
type providerUI struct {
	*Provider
}

//create a schedule
func (p *providerUI) createSchedule(bucket1 []*providerSchedule, bucket2 []*providerSchedule, schedule *DaySchedule) ([]*providerSchedule, []*providerSchedule) {
	if schedule.Unavailable {
		return bucket1, bucket2
	}
	bucketItem := &providerSchedule{
		DayOfWeek: schedule.DayOfWeek.String()[0:3],
		Times:     make([]*TimePeriod, len(schedule.TimeDurations)),
	}
	for idx, timeDuration := range schedule.TimeDurations {
		bucketItem.Times[idx] = &TimePeriod{
			Start: timeDuration.Start,
			End:   timeDuration.GetEnd(),
		}
	}
	if len(bucket1) < providerBucket1Size {
		bucket1 = append(bucket1, bucketItem)
	} else {
		bucket2 = append(bucket2, bucketItem)
	}
	return bucket1, bucket2
}

//GetScheduleBuckets : get the schedule in two buckets
func (p *providerUI) GetScheduleBuckets() ([]*providerSchedule, []*providerSchedule) {
	bucket1 := make([]*providerSchedule, 0, providerBucket1Size)
	bucket2 := make([]*providerSchedule, 0, providerBucket2Size)
	schedule := p.GetSchedule()
	if schedule == nil {
		return bucket1, bucket2
	}
	bucket1, bucket2 = p.createSchedule(bucket1, bucket2, schedule.DaySchedules[time.Monday])
	bucket1, bucket2 = p.createSchedule(bucket1, bucket2, schedule.DaySchedules[time.Tuesday])
	bucket1, bucket2 = p.createSchedule(bucket1, bucket2, schedule.DaySchedules[time.Wednesday])
	bucket1, bucket2 = p.createSchedule(bucket1, bucket2, schedule.DaySchedules[time.Thursday])
	bucket1, bucket2 = p.createSchedule(bucket1, bucket2, schedule.DaySchedules[time.Friday])
	bucket1, bucket2 = p.createSchedule(bucket1, bucket2, schedule.DaySchedules[time.Saturday])
	bucket1, bucket2 = p.createSchedule(bucket1, bucket2, schedule.DaySchedules[time.Sunday])
	return bucket1, bucket2
}

//GetURLAbout : get the URL for the provider about page
func (p *providerUI) GetURLAbout() string {
	return createDashboardURL(URIAbout)
}

//GetURLAboutClient : get the URL for the provider about page
func (p *providerUI) GetURLAboutClient() string {
	return createProviderURL(p.GetURLName(), URIAbout)
}

//GetURLAccount : get the URL for the provider account page
func (p *providerUI) GetURLAccount() string {
	return createDashboardURL(URIAccount)
}

//GetURLAccountAnchor : get the URL for the provider account page with an anchor tag
func (p *providerUI) GetURLAccountAnchor(anchor string) string {
	url := p.GetURLAccount()
	if anchor == "" {
		return url
	}
	url = fmt.Sprintf("%s#%s", url, anchor)
	return url
}

//GetURLAddOns : get the URL for the provider other services page
func (p *providerUI) GetURLAddOns() string {
	return createDashboardURL(URIAddOns)
}

//GetURLBookingAdd : get the URL for the provider add booking page
func (p *providerUI) GetURLBookingAdd() string {
	return createDashboardURL(URIBookingAdd)
}

//GetURLBookingAdd : get the URL for the provider add booking success page
func (p *providerUI) GetURLBookingAddSuccess(id *uuid.UUID) string {
	url := createDashboardURL(URIBookingAddSuccess)
	if id == nil {
		return url
	}
	url, err := CreateURLRelParams(url, URLParams.BookID, id)
	if err != nil {
		_, logger := GetLogger(nil)
		logger.Errorf("create url", "url", url)
		return ""
	}
	return url
}

//GetURLBookingCancelSuccess : get the URL for the provider cancel booking success page
func (p *providerUI) GetURLBookingCancelSuccess(id *uuid.UUID) string {
	url := createDashboardURL(URIBookingCancelSuccess)
	if id == nil {
		return url
	}
	url, err := CreateURLRelParams(url, URLParams.BookID, id)
	if err != nil {
		_, logger := GetLogger(nil)
		logger.Errorf("create url", "url", url)
		return ""
	}
	return url
}

//GetURLBookingEdit : get the URL for the provider edit booking page
func (p *providerUI) GetURLBookingEdit() string {
	return createDashboardURL(URIBookingEdit)
}

//GetURLBookingEditSuccess : get the URL for the provider edit booking success page
func (p *providerUI) GetURLBookingEditSuccess(id *uuid.UUID) string {
	url := createDashboardURL(URIBookingEditSuccess)
	if id == nil {
		return url
	}
	url, err := CreateURLRelParams(url, URLParams.BookID, id)
	if err != nil {
		_, logger := GetLogger(nil)
		logger.Errorf("create url", "url", url)
		return ""
	}
	return url
}

//GetURLBookingView : get the URL for the provider view booking page
func (p *providerUI) GetURLBookingView() string {
	return createDashboardURL(URIBookingView)
}

//GetURLBookings : get the URL for the provider bookings page
func (p *providerUI) GetURLBookings() string {
	return createDashboardURL(URIBookings)
}

//GetURLCalendar : get the URL for the provider calendar page
func (p *providerUI) GetURLCalendar() string {
	return createDashboardURL(URICalendars)
}

//GetURLCampaignAddStep1 : get the URL for the provider add campaign step 1 page
func (p *providerUI) GetURLCampaignAddStep1(id *uuid.UUID) string {
	url := createDashboardURL(URICampaignAddStep1)
	if id == nil {
		return url
	}
	url, err := CreateURLRelParams(url, URLParams.ID, id)
	if err != nil {
		_, logger := GetLogger(nil)
		logger.Errorf("create url", "url", url)
		return ""
	}
	return url
}

//GetURLCampaignAddStep2 : get the URL for the provider add campaign step 2 page
func (p *providerUI) GetURLCampaignAddStep2(id *uuid.UUID) string {
	url := createDashboardURL(URICampaignAddStep2)
	if id == nil {
		return url
	}
	url, err := CreateURLRelParams(url, URLParams.ID, id)
	if err != nil {
		_, logger := GetLogger(nil)
		logger.Errorf("create url", "url", url)
		return ""
	}
	return url
}

//GetURLCampaignAddStep3 : get the URL for the provider add campaign step 3 page
func (p *providerUI) GetURLCampaignAddStep3() string {
	return createDashboardURL(URICampaignAddStep3)
}

//GetURLCampaignView : get the URL for the provider view campaign page
func (p *providerUI) GetURLCampaignView(id *uuid.UUID) string {
	url := createDashboardURL(URICampaignView)
	if id == nil {
		return url
	}
	url, err := CreateURLRelParams(url, URLParams.ID, id)
	if err != nil {
		_, logger := GetLogger(nil)
		logger.Errorf("create url", "url", url)
		return ""
	}
	return url
}

//GetURLCampaigns : get the URL for the provider campaigns
func (p *providerUI) GetURLCampaigns() string {
	return createDashboardURL(URICampaigns)
}

//GetURLClientAdd : get the URL for the provider add client page
func (p *providerUI) GetURLClientAdd() string {
	return createDashboardURL(URIClientAdd)
}

//GetURLClientEdit : get the URL for the provider edit client page
func (p *providerUI) GetURLClientEdit() string {
	return createDashboardURL(URIClientEdit)
}

//GetURLClients : get the URL for the provider clients page
func (p *providerUI) GetURLClients() string {
	return createDashboardURL(URIClients)
}

//GetURLContactClient : get the URL for the provider contact page seen by the client
func (p *providerUI) GetURLContactClient() string {
	return createProviderURL(p.GetURLName(), URIContact)
}

//GetURLCouponAdd : get the URL for the provider add coupon page
func (p *providerUI) GetURLCouponAdd() string {
	return createDashboardURL(URICouponAdd)
}

//GetURLCouponEdit : get the URL for the provider edit coupon page
func (p *providerUI) GetURLCouponEdit(id *uuid.UUID) string {
	url := createDashboardURL(URICouponEdit)
	if id == nil {
		return url
	}
	url, err := CreateURLRelParams(url, URLParams.ID, id)
	if err != nil {
		_, logger := GetLogger(nil)
		logger.Errorf("create url", "url", url)
		return ""
	}
	return url
}

//GetURLCoupons : get the URL for the provider coupons page
func (p *providerUI) GetURLCoupons() string {
	return createDashboardURL(URICoupons)
}

//GetURLDashboard : get the URL for the provider dashboard
func (p *providerUI) GetURLDashboard() string {
	return createDashboardURL(URIIndex)
}

//GetURLFaqAdd : get the URL for the provider add faq page
func (p *providerUI) GetURLFaqAdd() string {
	return createDashboardURL(URIFaqAdd)
}

//GetURLFaqClient : get the URL for the provider faq page
func (p *providerUI) GetURLFaqClient() string {
	return createProviderURL(p.GetURLName(), URIFaq)
}

//GetURLFaqEdit : get the URL for the provider edit faq page
func (p *providerUI) GetURLFaqEdit(id *uuid.UUID) string {
	url := createDashboardURL(URIFaqEdit)
	if id == nil {
		return url
	}
	url, err := CreateURLRelParams(url, URLParams.ID, id)
	if err != nil {
		_, logger := GetLogger(nil)
		logger.Errorf("create url", "url", url)
		return ""
	}
	return url
}

//GetURLFaqs : get the URL for the provider faqs
func (p *providerUI) GetURLFaqs() string {
	return createDashboardURL(URIFaqs)
}

//GetURLHours : get the URL for the provider hours page
func (p *providerUI) GetURLHours() string {
	return createDashboardURL(URIHours)
}

//GetURLLinks : get the URL for the provider links page
func (p *providerUI) GetURLLinks() string {
	return createDashboardURL(URILinks)
}

//GetURLMap : get the URL for the provider location map
func (p *providerUI) GetURLMap() string {
	if p.Location != "" {
		return fmt.Sprintf(GetGoogleURLMap(), url.QueryEscape(p.Location))
	}
	return ""
}

//GetURLPaymentDirect : get the URL for the provider direct payment page seen by the client
func (p *providerUI) GetURLPaymentDirectClient() string {
	return createProviderURL(p.GetURLName(), URIPaymentDirect)
}

//GetURLPaymentSettings : get the URL for the provider payment settings page
func (p *providerUI) GetURLPaymentSettings() string {
	return createDashboardURL(URIPaymentSettings)
}

//GetURLPayments : get the URL for the provider payments page
func (p *providerUI) GetURLPayments() string {
	return createDashboardURL(URIPayments)
}

//GetURLProfile : get the URL for the provider profile page
func (p *providerUI) GetURLProfile() string {
	return createDashboardURL(URIProfile)
}

//GetURLProfileDomain : get the URL for the provider profile domain page
func (p *providerUI) GetURLProfileDomain() string {
	return createDashboardURL(URIProfileDomain)
}

//GetURLProfileBanner : get the URL for the provider profile banner page
func (p *providerUI) GetURLProfileBanner() string {
	url, err := CreateURLRelParams(createDashboardURL(URIProfile), URLParams.Type, ImgViewTypes.TypeBanner)
	if err != nil {
		_, logger := GetLogger(nil)
		logger.Errorf("create url", "url", createDashboardURL(URIProfile))
		return ""
	}
	return url
}

//GetURLProfileLogo : get the URL for the provider profile logo page
func (p *providerUI) GetURLProfileLogo() string {
	url, err := CreateURLRelParams(createDashboardURL(URIProfile), URLParams.Type, ImgViewTypes.TypeLogo)
	if err != nil {
		_, logger := GetLogger(nil)
		logger.Errorf("create url", "url", createDashboardURL(URIProfile))
		return ""
	}
	return url
}

//GetURLProfileType : get the URL for the provider profile page based on the type
func (p *providerUI) GetURLProfileType(profileType string) string {
	switch profileType {
	case ImgViewTypes.TypeLogo:
		return p.GetURLProfileLogo()
	case ImgViewTypes.TypeBanner:
		return p.GetURLProfileBanner()
	}
	return p.GetURLProfile()
}

//GetURLProviderPermanent : get the permanent URL for the provider
func (p *providerUI) GetURLProviderPermanent() string {
	return createProviderURL(p.URLName, URIDefault)
}

//GetURLProvider : get the URL for the provider
func (p *providerUI) GetURLProvider() string {
	return createProviderURL(p.GetURLName(), URIDefault)
}

//GetURLServiceAdd : get the URL for the provider add service page
func (p *providerUI) GetURLServiceAdd() string {
	return createDashboardURL(URISvcAdd)
}

//GetURLServiceEdit : get the URL for the provider edit service page
func (p *providerUI) GetURLServiceEdit(id *uuid.UUID) string {
	url := createDashboardURL(URISvcEdit)
	if id == nil {
		return url
	}
	url, err := CreateURLRelParams(url, URLParams.SvcID, id)
	if err != nil {
		_, logger := GetLogger(nil)
		logger.Errorf("create url", "url", url)
		return ""
	}
	return url
}

//GetURLServiceEditImgs : get the URL for the provider edit service images page
func (p *providerUI) GetURLServiceEditImgs(id *uuid.UUID) string {
	url := p.GetURLServiceEdit(id)
	if id == nil {
		return url
	}
	url, err := CreateURLRelParams(url, URLParams.Type, ImgViewTypes.TypeImg)
	if err != nil {
		_, logger := GetLogger(nil)
		logger.Errorf("create url", "url", url)
		return ""
	}
	return url
}

//GetURLServiceUsers : get the URL for the provider service users page
func (p *providerUI) GetURLServiceUsers(id *uuid.UUID) string {
	url := createDashboardURL(URISvcUsers)
	if id == nil {
		return url
	}
	url, err := CreateURLRelParams(url, URLParams.SvcID, id)
	if err != nil {
		_, logger := GetLogger(nil)
		logger.Errorf("create url", "url", url)
		return ""
	}
	return url
}

//GetURLServices : get the URL for the provider services
func (p *providerUI) GetURLServices() string {
	return createDashboardURL(URISvcs)
}

//GetURLTestimonialAdd : get the URL for the provider add testimonial page
func (p *providerUI) GetURLTestimonialAdd() string {
	return createDashboardURL(URITestimonialAdd)
}

//GetURLTestimonialEdit : get the URL for the provider edit testimonial page
func (p *providerUI) GetURLTestimonialEdit() string {
	return createDashboardURL(URITestimonialEdit)
}

//GetURLTestimonials : get the URL for the provider testimonials
func (p *providerUI) GetURLTestimonials() string {
	return createDashboardURL(URITestimonials)
}

//GetURLUserAdd : get the URL for the provider add user page
func (p *providerUI) GetURLUserAdd() string {
	return createDashboardURL(URIUserAdd)
}

//GetURLUserEdit : get the URL for the provider edit user page
func (p *providerUI) GetURLUserEdit() string {
	return createDashboardURL(URIUserEdit)
}

//GetURLUsers : get the URL for the provider users page
func (p *providerUI) GetURLUsers() string {
	return createDashboardURL(URIUsers)
}

//GetURLImgBanner : get the URL for the banner image
func (p *providerUI) GetURLImgBanner() string {
	img := p.ImgBanner
	if img == nil {
		return URLJoin(GetURLAssets(), ImgDefaultProviderBanner)
	}
	return URLJoin(GetURLUploads(), URLAssetUpload, img.GetFile())
}

//GetURLImgBannerSet : get the URL for the set banner image
func (p *providerUI) GetURLImgBannerSet() string {
	img := p.ImgBanner
	if img == nil {
		return ""
	}
	return URLJoin(GetURLUploads(), URLAssetUpload, img.GetFile())
}

//GetURLImgFavIcon : get the URL for the favicon image
func (p *providerUI) GetURLImgFavIcon() string {
	img := p.ImgFavIcon
	if img == nil {
		return URLJoin(GetURLAssets(), ImgDefaultProviderFavIcon)
	}

	//use the uploaded image
	url := URLJoin(GetURLUploads(), URLAssetUpload, img.GetFile())
	if img.Version != 0 {
		var err error
		url, err = CreateURLRelParams(url, URLParams.Version, img.Version)
		if err != nil {
			_, logger := GetLogger(nil)
			logger.Errorf("create url", "url", url)
		}
		return url
	}
	return url
}

//GetURLImgLogo : get the URL for the logo image
func (p *providerUI) GetURLImgLogo() string {
	img := p.ImgLogo
	if img == nil {
		return URLJoin(GetURLAssets(), ImgDefaultProviderLogo)
	}
	return URLJoin(GetURLUploads(), URLAssetUpload, img.GetFile())
}

//GetURLImgLogoSet : get the URL for the set logo image
func (p *providerUI) GetURLImgLogoSet() string {
	img := p.ImgLogo
	if img == nil {
		return ""
	}
	return URLJoin(GetURLUploads(), URLAssetUpload, img.GetFile())
}

//MarkURLClient : mark a url as coming from a client page
func (p *providerUI) MarkURLClient(url string) string {
	url, err := CreateURLRelParams(url, URLParams.Client, true)
	if err != nil {
		_, logger := GetLogger(nil)
		logger.Errorf("create url", "url", url)
		return ""
	}
	return url
}

//FormatAbout : format the about string
func (p *providerUI) FormatAbout() template.HTML {
	return template.HTML(p.About)
}

//FormatEducation : format the education string
func (p *providerUI) FormatEducation() template.HTML {
	//convert newlines to breaks
	txt := ConvertNewLinesToBreaks(p.Education)
	return template.HTML(txt)
}

//FormatExperience : format the experience string
func (p *providerUI) FormatExperience() template.HTML {
	//convert newlines to breaks
	txt := ConvertNewLinesToBreaks(p.Experience)
	return template.HTML(txt)
}

//FormatLocation : format the location string
func (p *providerUI) FormatLocation() template.HTML {
	//convert newlines to breaks
	txt := ConvertNewLinesToBreaks(p.Location)
	return template.HTML(txt)
}

//service wrapper used for the ui
type serviceUI struct {
	*Service
}

//FormatNote : format the note string
func (s *serviceUI) FormatNote() template.HTML {
	//convert newlines to breaks
	txt := ConvertNewLinesToBreaks(s.Note)
	return template.HTML(txt)
}

//FormatVideoPlayerHTML : format the video player HTML
func (s *serviceUI) FormatVideoPlayerHTML() template.HTML {
	return template.HTML(s.HTMLVideoPlayer)
}

//GetURLBooking : return the URL to book the service
func (s *serviceUI) GetURLBooking() string {
	return createProviderServiceURL(s.Provider.URLName, s.ID, URIBooking)
}

//GetURLBookingSubmit : return the URL to submit the booking of the service
func (s *serviceUI) GetURLBookingSubmit() string {
	return createProviderServiceURL(s.Provider.URLName, s.ID, URIBookingSubmit)
}

//GetURLService : return the URL for the service
func (s *serviceUI) GetURLService() string {
	return createProviderServiceURL(s.Provider.URLName, s.ID, URIDefault)
}

//GetURLImgMain : get the URLs for the service main image
func (s *serviceUI) GetURLImgMain() string {
	img := s.ImgMain
	if img == nil {
		return URLJoin(GetURLAssets(), ImgDefaultService)
	}
	return URLJoin(GetURLUploads(), URLAssetUpload, s.ImgMain.GetFile())
}

//GetURLImgs : get the URLs for the set service images
func (s *serviceUI) GetURLImgs() []string {
	imgs := s.Imgs
	lenImgs := len(imgs)
	if lenImgs == 0 {
		return nil
	}
	fullURLs := make([]string, lenImgs)
	for idx := range imgs {
		fullURLs[idx] = URLJoin(GetURLUploads(), URLAssetUpload, imgs[idx].GetFile())
	}
	return fullURLs
}

//GetURLImgsJSON : get the URLS for the set service images as JSON
func (s *serviceUI) GetURLImgsJSON() string {
	imgs := s.Imgs
	lenImgs := len(imgs)
	if lenImgs == 0 {
		return ""
	}
	fullURLs := make([]string, lenImgs)
	for idx := range imgs {
		fullURLs[idx] = URLJoin(GetURLUploads(), URLAssetUpload, imgs[idx].GetFile())
	}
	data, err := json.Marshal(fullURLs)
	if err != nil {
		_, logger := GetLogger(nil)
		logger.Errorf("create imgs json", "urls", fullURLs)
		return ""
	}
	return string(data)
}

//booking wrapper used for the ui
type bookingUI struct {
	*Booking
}

//FormatDescription : format the description string
func (b *bookingUI) FormatDescription() template.HTML {
	//convert newlines to breaks
	txt := ConvertNewLinesToBreaks(b.Description)
	return template.HTML(txt)
}

//FormatProviderNote : format the provider note string
func (b *bookingUI) FormatProviderNote() template.HTML {
	//convert newlines to breaks
	txt := ConvertNewLinesToBreaks(b.ProviderNote)
	return template.HTML(txt)
}

//GetEventTitle : create the event title
func (b *bookingUI) GetEventTitle() string {
	str := fmt.Sprintf("%s - %s", b.ServiceName, b.Client.Name)
	if !b.Confirmed {
		str = fmt.Sprintf("%s (unconfirmed)", str)
	}
	return str
}

//GetEventDescription : create the event description
func (b *bookingUI) GetEventDescription(ctx context.Context, useAnchor bool, addZoomURL bool) (context.Context, string, error) {
	var str strings.Builder

	//add the client
	str.WriteString("Client:\n")
	str.WriteString(b.Client.Name)
	str.WriteString("\n")
	str.WriteString(b.Client.Email)
	if b.Client.Phone != "" {
		str.WriteString("\n")
		str.WriteString(b.Client.Phone)
	}

	//add the special request
	if b.Description != "" {
		str.WriteString("\n")
		str.WriteString("\n")
		str.WriteString("Special Request:\n")
		str.WriteString(b.Description)
	}

	//add the note
	if b.ProviderNote != "" {
		str.WriteString("\n")
		str.WriteString("\n")
		str.WriteString("Message to Client:\n")
		str.WriteString(b.ProviderNote)
	}

	//add the order url
	str.WriteString("\n")
	str.WriteString("\n")
	url, err := CreateURLAbs(ctx, b.GetURLView(), nil)
	if err != nil {
		return ctx, "", errors.Wrap(err, "create url")
	}
	if useAnchor {
		str.WriteString(fmt.Sprintf("<a href=\"%s\">View Order</a>", url))
	} else {
		str.WriteString(fmt.Sprintf("View Order:\n%s", url))
	}

	//add zoom
	if b.MeetingZoomData != nil && addZoomURL {
		str.WriteString("\n")
		str.WriteString("\n")
		if useAnchor {
			str.WriteString(fmt.Sprintf("<a href=\"%s\">Start Zoom Meeting</a>", b.MeetingZoomData.URLStart))
		} else {
			str.WriteString(fmt.Sprintf("Start Zoom Meeting:\n%s", b.MeetingZoomData.URLStart))
		}
	}
	str.WriteString("\n")
	return ctx, str.String(), nil
}

//GetURLEdit : get the URL for the provider edit booking page
func (b *bookingUI) GetURLEdit() string {
	url, err := CreateURLRelParams(createDashboardURL(URIBookingEdit), URLParams.BookID, b.ID)
	if err != nil {
		_, logger := GetLogger(nil)
		logger.Errorf("create url", "url", createDashboardURL(URIBookingEdit))
		return ""
	}
	return url
}

//GetURLPayment : get the URL for the provider booking payment page
func (b *bookingUI) GetURLPayment() string {
	url, err := CreateURLRelParams(createDashboardURL(URIPayment), URLParams.BookID, b.ID)
	if err != nil {
		_, logger := GetLogger(nil)
		logger.Errorf("create url", "url", createDashboardURL(URIPayment))
		return ""
	}
	return url
}

//GetURLPaymentClient : get the URL for the booking payment client page
func (b *bookingUI) GetURLPaymentClient() string {
	return createProviderServiceBookURL(b.Provider.URLName, b.Service.ID, b.ID, URIPayment)
}

//GetURLPaymentView : get the URL for the provider view booking payment page
func (b *bookingUI) GetURLPaymentView() string {
	url, err := CreateURLRelParams(createDashboardURL(URIPaymentView), URLParams.BookID, b.ID)
	if err != nil {
		_, logger := GetLogger(nil)
		logger.Errorf("create url", "url", createDashboardURL(URIPaymentView))
		return ""
	}
	return url
}

//GetURLView : get the URL for the provider view booking page
func (b *bookingUI) GetURLView() string {
	url, err := CreateURLRelParams(createDashboardURL(URIBookingView), URLParams.BookID, b.ID)
	if err != nil {
		_, logger := GetLogger(nil)
		logger.Errorf("create url", "url", createDashboardURL(URIBookingView))
		return ""
	}
	return url
}

//GetURLConfirmClient : return the URL for the confirmation of a booking by a client
func (b *bookingUI) GetURLConfirmClient() string {
	return createProviderServiceBookURL(b.Provider.URLName, b.Service.ID, b.ID, URIBookingConfirm)
}

//GetURLViewClient : return the URL to view the booking by a client
func (b *bookingUI) GetURLViewClient() string {
	return createProviderServiceBookURL(b.Provider.URLName, b.Service.ID, b.ID, URIDefault)
}

//GetURLCancelClient : return the URL to cancel the booking by a client
func (b *bookingUI) GetURLCancelClient() string {
	return createProviderServiceBookURL(b.Provider.URLName, b.Service.ID, b.ID, URIBookingCancel)
}

//GetURLocationMap : return the URL to map the location
func (b *bookingUI) GetURLocationMap() string {
	if b.Location != "" {
		return fmt.Sprintf(GetGoogleURLMap(), url.QueryEscape(b.Location))
	}
	return ""
}

//bookings for a date
type bookingByDate struct {
	Date     time.Time
	Bookings []*bookingUI
}

//AddBooking : add a booking
func (b *bookingByDate) AddBooking(book *bookingUI) {
	b.Bookings = append(b.Bookings, book)
}

//FormatWeekDay : return the day of the week
func (b *bookingByDate) FormatWeekDay() string {
	return b.Date.Weekday().String()
}

//FormatDateLong : format the as a long date
func (b *bookingByDate) FormatDateLong(timeZone string) string {
	return FormatDateLongLocal(b.Date, timeZone)
}

//bookings organized by the date
type bookingsByDate struct {
	Items []*bookingByDate
}

//AddBooking : add a booking
func (b *bookingsByDate) AddBooking(book *bookingUI, timeZone string) {
	//initialize if necessary
	if b.Items == nil {
		b.Items = make([]*bookingByDate, 0, 1)
	}

	//find the bookings for the date
	t := GetTimeLocal(book.TimeFrom, timeZone)
	bod := GetBeginningOfDay(t)
	for _, b := range b.Items {
		if b.Date.Equal(bod) {
			b.AddBooking(book)
			return
		}
	}

	//add a new entry
	bbw := &bookingByDate{
		Date:     bod,
		Bookings: make([]*bookingUI, 0, 1),
	}
	b.Items = append(b.Items, bbw)
	bbw.AddBooking(book)
}

//faq wrapper used for the ui
type faqUI struct {
	*Faq
}

//payment wrapper used for the ui
type paymentUI struct {
	*Payment
}

//FormatNote : format the description string
func (p *paymentUI) FormatNote() template.HTML {
	//convert newlines to breaks
	txt := ConvertNewLinesToBreaks(p.Note)
	return template.HTML(txt)
}

//GetURLView : get the URL for the payment page
func (p *paymentUI) GetURLView() string {
	url, err := CreateURLRelParams(createDashboardURL(URIPaymentView), URLParams.PaymentID, p.ID)
	if err != nil {
		_, logger := GetLogger(nil)
		logger.Errorf("create url", "url", createDashboardURL(URIPaymentView))
		return ""
	}
	return url
}

//testimonial wrapper used for the ui
type testimonialUI struct {
	*Testimonial
}

//GetURLImg : get the URL for the testimonial image
func (t *testimonialUI) GetURLImg() string {
	img := t.Img
	if img == nil {
		return ""
	}
	return URLJoin(GetURLUploads(), URLAssetUpload, img.GetFile())
}

//campaign wrapper used for the ui
type campaignUI struct {
	*Campaign
}

//GetURLImg : get the URL for the ad image
func (c *campaignUI) GetURLImg() string {
	img := c.Img
	if img == nil {
		return ""
	}
	return URLJoin(GetURLUploads(), URLAssetUpload, img.GetFile())
}

//GetURLViewExternal : get the URL for the manage campaign page
func (c *campaignUI) GetURLViewExternal() string {
	url := URICampaignManage
	if c.ExternalID == nil {
		return url
	}
	url, err := CreateURLRelParams(url, URLParams.ID, c.ExternalID)
	if err != nil {
		_, logger := GetLogger(nil)
		logger.Errorf("create url", "url", url)
		return ""
	}
	return url
}

//GetURLPayment : get the URL for payment
func (c *campaignUI) GetURLPayment(id *uuid.UUID) string {
	url := URIPayment
	if id == nil {
		return url
	}
	url, err := CreateURLRelParams(url, URLParams.ID, id)
	if err != nil {
		_, logger := GetLogger(nil)
		logger.Errorf("create url", "url", url)
		return ""
	}
	return url
}
