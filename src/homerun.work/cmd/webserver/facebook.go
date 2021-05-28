package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
)

//facebook constants
const (
	//parameters
	FacebookParamAccessToken  = "access_token"
	FacebookParamAdCategories = "special_ad_categories"
	FacebookParamAdFormat     = "ad_format"
	FacebookParamBidStrategy  = "bid_strategy"
	FacebookParamBillingEvent = "billing_event"
	FacebookParamCampaignID   = "campaign_id"
	FacebookParamDailyBudget  = "daily_budget"
	FacebookParamDatePreset   = "date_preset"
	FacebookParamFields       = "fields"
	FacebookParamHeight       = "height"
	FacebookParamName         = "name"
	FacebookParamObjective    = "objective"
	FacebookParamStatus       = "status"
	FacebookParamTargeting    = "targeting"
	FacebookParamTimeStart    = "start_time"
	FacebookParamTimeEnd      = "end_time"
	FacebookParamWidth        = "width"

	//general
	FacebookAdFields            = "id,name,effective_status"
	FacebookDatePresetsLifetime = "lifetime"
	FacebookInsightFields       = "clicks,cpc,ctr,impressions,reach,spend"
	FacebookPreviewHeight       = 690
	FacebookPreviewWidth        = 540
	FacebookRequestTimeOut      = 10 * time.Second
	FacebookTargetingUS         = "{\"geo_locations\":{\"countries\":[\"US\"]}}"

	//urls
	FacebookURLAccounts    = "https://graph.facebook.com/v7.0/me/accounts"
	FacebookURLAdAccounts  = "https://graph.facebook.com/v7.0/me/adaccounts"
	FacebookURLAdAccount   = "https://graph.facebook.com/v7.0/act_%s"
	FacebookURLAdPreviews  = "https://graph.facebook.com/v7.0/%s/previews"
	FacebookURLAd          = "https://graph.facebook.com/v7.0/%s"
	FacebookURLAdSetAds    = "https://graph.facebook.com/v7.0/%s/ads"
	FacebookURLAdSetCreate = "https://graph.facebook.com/v7.0/act_%s/adsets"
	FacebookURLAdSetUpdate = "https://graph.facebook.com/v7.0/%s"
	FacebookURLCampaign    = "https://graph.facebook.com/v7.0/act_%s/campaigns"
	FacebookURLInsights    = "https://graph.facebook.com/v7.0/%s/insights"
	FacebookURLUser        = "https://graph.facebook.com/v7.0/me?fields=id,email,first_name,last_name"
)

//FacebookAdCategory : Facebook ad category
type FacebookAdCategory string

//facebook ad categories
const (
	FacebookAdCategoryCredit     FacebookAdCategory = "CREDIT"
	FacebookAdCategoryEmployment FacebookAdCategory = "EMPLOYMENT"
	FacebookAdCategoryHousing    FacebookAdCategory = "HOUSING"
	FacebookAdCategoryNone       FacebookAdCategory = "NONE"
)

//FacebookAdFormat : Facebook ad format
type FacebookAdFormat string

//facebook ad formats
const (
	FacebookAdFormatDesktopFeedStandard FacebookAdFormat = "DESKTOP_FEED_STANDARD"
)

//FacebookAdObjective : Facebook ad objective
type FacebookAdObjective string

//facebook ad objectives
const (
	FacebookAdObjectiveAppInstalls         FacebookAdObjective = "APP_INSTALLS"
	FacebookAdObjectiveBrandAwareness      FacebookAdObjective = "BRAND_AWARENESS"
	FacebookAdObjectiveConversions         FacebookAdObjective = "CONVERSIONS"
	FacebookAdObjectiveEventResponses      FacebookAdObjective = "EVENT_RESPONSES"
	FacebookAdObjectiveLeadGeneration      FacebookAdObjective = "LEAD_GENERATION"
	FacebookAdObjectiveLinkClicks          FacebookAdObjective = "LINK_CLICKS"
	FacebookAdObjectiveLocalAwareness      FacebookAdObjective = "LOCAL_AWARENESS"
	FacebookAdObjectiveMessages            FacebookAdObjective = "MESSAGES"
	FacebookAdObjectiveOfferClaims         FacebookAdObjective = "OFFER_CLAIMS"
	FacebookAdObjectivePageLikes           FacebookAdObjective = "PAGE_LIKES"
	FacebookAdObjectivePostEngagement      FacebookAdObjective = "POST_ENGAGEMENT"
	FacebookAdObjectiveProductCatalogSales FacebookAdObjective = "PRODUCT_CATALOG_SALES"
	FacebookAdObjectiveReach               FacebookAdObjective = "REACH"
	FacebookAdObjectiveVideoViews          FacebookAdObjective = "VIDEO_VIEWS"
)

//FacebookBidStrategy : Facebook bid strategy
type FacebookBidStrategy string

//facebook ad categories
const (
	FacebookBidStrategyLowestCostWithoutCap FacebookBidStrategy = "LOWEST_COST_WITHOUT_CAP"
	FacebookBidStrategyLowestCostWithBidCap FacebookBidStrategy = "LOWEST_COST_WITH_BID_CAP"
	FacebookBidStrategyTargetCost           FacebookBidStrategy = "TARGET_COST"
	FacebookBidStrategyCostCap              FacebookBidStrategy = "COST_CAP"
)

//FacebookBillingEvent : Facebook billing event
type FacebookBillingEvent string

//facebook ad categories
const (
	FacebookBillingEventAppInstalls    FacebookBillingEvent = "APP_INSTALLS"
	FacebookBillingEventClicks         FacebookBillingEvent = "CLICKS"
	FacebookBillingEventImpressions    FacebookBillingEvent = "IMPRESSIONS"
	FacebookBillingEventLinkClicks     FacebookBillingEvent = "LINK_CLICKS"
	FacebookBillingEventNone           FacebookBillingEvent = "NONE"
	FacebookBillingEventOfferClaims    FacebookBillingEvent = "OFFER_CLAIMS"
	FacebookBillingEventPageLikes      FacebookBillingEvent = "PAGE_LIKES"
	FacebookBillingEventPostEngagement FacebookBillingEvent = "POST_ENGAGEMENT"
	FacebookBillingEventThruPlay       FacebookBillingEvent = "THRUPLAY"
)

//FacebookAdStatus : Facebook status
type FacebookAdStatus string

//facebook ad statuses
const (
	FacebookAdStatusActive             FacebookAdStatus = "ACTIVE"
	FacebookAdStatusAdSetPaused        FacebookAdStatus = "ADSET_PAUSED"
	FacebookAdStatusArchived           FacebookAdStatus = "ARCHIVED"
	FacebookAdStatusCampaignPaused     FacebookAdStatus = "CAMPAIGN_PAUSED"
	FacebookAdStatusDeleted            FacebookAdStatus = "DELETED"
	FacebookAdStatusDisapproved        FacebookAdStatus = "DISAPPROVED"
	FacebookAdStatusInProcess          FacebookAdStatus = "IN_PROCESS"
	FacebookAdStatusPaused             FacebookAdStatus = "PAUSED"
	FacebookAdStatusPendingReview      FacebookAdStatus = "PENDING_REVIEW"
	FacebookAdStatusPendingBillingInfo FacebookAdStatus = "PENDING_BILLING_INFO"
	FacebookAdStatusPreapproved        FacebookAdStatus = "PREAPPROVED"
	FacebookAdStatusWithIssues         FacebookAdStatus = "WITH_ISSUES"
)

//FacebookAdPreview : Facebook ad preview
type FacebookAdPreview struct {
	Body string `json:"body"`
}

//FacebookAdPreviewData : Facebook ad preview data
type FacebookAdPreviewData struct {
	Previews []*FacebookAdPreview `json:"data"`
}

//FacebookAd : Facebook ad
type FacebookAd struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	EffectiveStatus string `json:"effective_status"`
}

//IsActive : indicate if an ad is active
func (a *FacebookAd) IsActive() bool {
	return a.EffectiveStatus == string(FacebookAdStatusActive)
}

//FacebookAdSet : Facebook ad set
type FacebookAdSet struct {
	ID string `json:"id"`
}

//FacebookAdSetAd : Facebook ad set ad
type FacebookAdSetAd struct {
	ID string `json:"id"`
}

//FacebookAdSetAdData : Facebook ad set ad data
type FacebookAdSetAdData struct {
	Ads []*FacebookAdSetAd `json:"data"`
}

//FacebookCampaign : Facebook campaign
type FacebookCampaign struct {
	ID string `json:"id"`
}

//FacebookInsights : Facebook insight data
type FacebookInsights struct {
	Impressions   int     `json:"impressions,string"`
	Clicks        int     `json:"clicks,string"`
	ClickThruRate float32 `json:"ctr,string"`
	CostPerClick  float32 `json:"cpc,string"`
	Reach         int     `json:"reach,string"`
	Spend         float32 `json:"spend,string"`
}

//FormatClickThruRate : format the click-thru rate
func (i *FacebookInsights) FormatClickThruRate() string {
	return fmt.Sprintf("%.2f%%", i.ClickThruRate)
}

//FormatCostPerClick : format the cost per click
func (i *FacebookInsights) FormatCostPerClick() string {
	return fmt.Sprintf("$.2%f", i.CostPerClick)
}

//FormatSpend : format the spend
func (i *FacebookInsights) FormatSpend() string {
	return fmt.Sprintf("$%.2f", i.Spend)
}

//FacebookInsightsData : Facebook insights data
type FacebookInsightsData struct {
	Insights []*FacebookInsights `json:"data"`
}

//FacebookAccount : Facebook account
type FacebookAccount struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Category string `json:"category"`
}

//FacebookAccountData : Facebook account data
type FacebookAccountData struct {
	Accounts []*FacebookAccount `json:"data"`
}

//FacebookResponseError : Facebook response error
type FacebookResponseError struct {
	Message        string `json:"message"`
	Type           string `json:"type"`
	Code           int    `json:"code"`
	ErrorSubCode   int    `json:"error_subcode"`
	ErrorUserTitle string `json:"error_user_title"`
	ErrorUserMsg   string `json:"error_user_msg"`
	TraceID        string `json:"fbtrace_id"`
}

//FacebookResponse : Facebook response
type FacebookResponse struct {
	Success bool                   `json:"success"`
	Error   *FacebookResponseError `json:"error"`
}

//FacebookUser : Facebook user
type FacebookUser struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

//create a facebook http client
func createClientFacebook() *http.Client {
	client := &http.Client{
		Timeout: FacebookRequestTimeOut,
	}
	return client
}

//CreateCampaignFacebook : create a Facebook campaign
func CreateCampaignFacebook(ctx context.Context, accessToken string, adAccountID string, name string, adCategories []FacebookAdCategory, objective FacebookAdObjective) (context.Context, *FacebookCampaign, error) {
	ctx, logger := GetLogger(ctx)
	start := time.Now()
	defer func() {
		logger.Debugw("facebook campaign", "elapsedMS", FormatElapsedMS(start))
		AddCtxStatsAPI(ctx, ServerStatAPIFacebook, "facebook campaign", time.Since(start))
	}()

	//encode the categories
	categoriesData, err := json.Marshal(adCategories)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "facebook campaign encode json")
	}

	//create the request
	data := url.Values{}
	data.Set(FacebookParamAccessToken, accessToken)
	data.Set(FacebookParamAdCategories, string(categoriesData))
	data.Set(FacebookParamName, name)
	data.Set(FacebookParamObjective, string(objective))

	//create the url
	url := fmt.Sprintf(FacebookURLCampaign, adAccountID)

	//make the request
	request, err := http.NewRequest("POST", url, strings.NewReader(data.Encode()))
	if err != nil {
		return ctx, nil, errors.Wrap(err, "facebook campaign http request")
	}
	client := createClientFacebook()
	request.Header.Set(HeaderContentType, "application/x-www-form-urlencoded")
	response, err := client.Do(request)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "facebook campaign http request")
	}
	if response.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(response.Body)
		return ctx, nil, fmt.Errorf("status: %d: %s", response.StatusCode, body)
	}

	//process the response
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "facebook campaign read body")
	}
	var campaign FacebookCampaign
	err = json.Unmarshal(body, &campaign)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "facebook campaign unjson")
	}
	defer response.Body.Close()
	return ctx, &campaign, nil
}

//CreateAdSetFacebook : create a Facebook ad set
func CreateAdSetFacebook(ctx context.Context, accessToken string, adAccountID string, campaignID string, name string, budget float32, adStart time.Time, adEnd time.Time, bidStrategy FacebookBidStrategy, billingEvent FacebookBillingEvent) (context.Context, *FacebookAdSet, error) {
	ctx, logger := GetLogger(ctx)
	start := time.Now()
	defer func() {
		logger.Debugw("facebook ad set", "elapsedMS", FormatElapsedMS(start))
		AddCtxStatsAPI(ctx, ServerStatAPIFacebook, "facebook ad set create", time.Since(start))
	}()

	//create the request
	data := url.Values{}
	data.Set(FacebookParamAccessToken, accessToken)
	data.Set(FacebookParamBidStrategy, string(bidStrategy))
	data.Set(FacebookParamBillingEvent, string(billingEvent))
	data.Set(FacebookParamCampaignID, campaignID)
	data.Set(FacebookParamName, name)
	data.Set(FacebookParamStatus, string(FacebookAdStatusActive))
	data.Set(FacebookParamTargeting, FacebookTargetingUS)
	data.Set(FacebookParamTimeStart, strconv.FormatInt(adStart.UTC().Unix(), 10))
	data.Set(FacebookParamTimeEnd, strconv.FormatInt(adEnd.UTC().Unix(), 10))

	//use non-decimal budget
	budgetInt := int64(budget * 100)
	data.Set(FacebookParamDailyBudget, strconv.FormatInt(budgetInt, 10))

	//create the url
	url := fmt.Sprintf(FacebookURLAdSetCreate, adAccountID)

	//make the request
	request, err := http.NewRequest("POST", url, strings.NewReader(data.Encode()))
	if err != nil {
		return ctx, nil, errors.Wrap(err, "facebook ad set create http request")
	}
	request.Header.Set(HeaderContentType, "application/x-www-form-urlencoded")
	client := createClientFacebook()
	response, err := client.Do(request)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "facebook ad set create http request")
	}
	if response.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(response.Body)
		return ctx, nil, fmt.Errorf("status: %d: %s", response.StatusCode, body)
	}

	//process the response
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "facebook ad set create read body")
	}
	var adSet FacebookAdSet
	err = json.Unmarshal(body, &adSet)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "facebook ad set create unjson")
	}
	defer response.Body.Close()
	return ctx, &adSet, nil
}

//ChangeStatusAdSetFacebook : change the status of a Facebook ad set
func ChangeStatusAdSetFacebook(ctx context.Context, accessToken string, id string, status FacebookAdStatus) (context.Context, error) {
	ctx, logger := GetLogger(ctx)
	start := time.Now()
	defer func() {
		logger.Debugw("facebook ad set update", "elapsedMS", FormatElapsedMS(start))
		AddCtxStatsAPI(ctx, ServerStatAPIFacebook, "facebook ad set update", time.Since(start))
	}()

	//create the request
	data := url.Values{}
	data.Set(FacebookParamAccessToken, accessToken)
	data.Set(FacebookParamStatus, string(status))

	//create the url
	url := fmt.Sprintf(FacebookURLAdSetUpdate, id)

	//make the request
	request, err := http.NewRequest("POST", url, strings.NewReader(data.Encode()))
	if err != nil {
		return ctx, errors.Wrap(err, "facebook ad set update http request")
	}
	request.Header.Set(HeaderContentType, "application/x-www-form-urlencoded")
	client := createClientFacebook()
	response, err := client.Do(request)
	if err != nil {
		return ctx, errors.Wrap(err, "facebook ad set update http request")
	}
	if response.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(response.Body)
		return ctx, fmt.Errorf("status: %d: %s", response.StatusCode, body)
	}

	//process the response
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return ctx, errors.Wrap(err, "facebook ad set update read body")
	}
	var responseData FacebookResponse
	err = json.Unmarshal(body, &responseData)
	if err != nil {
		return ctx, errors.Wrap(err, "facebook ad set update unjson")
	}
	defer response.Body.Close()
	if !responseData.Success {
		return ctx, fmt.Errorf("facebook ad set update: %v", responseData)
	}
	return ctx, nil
}

//GetAdSetAdFacebook : get Facebook ad preview
func GetAdSetAdFacebook(ctx context.Context, accessToken string, id string) (context.Context, *FacebookAdSetAd, error) {
	ctx, logger := GetLogger(ctx)
	start := time.Now()
	defer func() {
		logger.Debugw("facebook ad set ad", "elapsedMS", FormatElapsedMS(start))
		AddCtxStatsAPI(ctx, ServerStatAPIFacebook, "facebook ad set ad", time.Since(start))
	}()

	//create the url
	url := fmt.Sprintf(FacebookURLAdSetAds, id)

	//create the request
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "facebook ad set ad http request")
	}

	//add the query string
	query := request.URL.Query()
	query.Add(FacebookParamAccessToken, accessToken)
	request.URL.RawQuery = query.Encode()

	//make the request
	client := createClientFacebook()
	response, err := client.Do(request)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "facebook ad set ad http request")
	}
	if response.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(response.Body)
		return ctx, nil, fmt.Errorf("status: %d: %s", response.StatusCode, body)
	}

	//process the response
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "facebook ad set ad read body")
	}
	var ads FacebookAdSetAdData
	err = json.Unmarshal(body, &ads)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "facebook ad set ad unjson")
	}
	defer response.Body.Close()

	//read the insight data
	if len(ads.Ads) > 0 {
		return ctx, ads.Ads[0], nil
	}
	return ctx, nil, nil
}

//GetInsightsFacebook : get Facebook insights
func GetInsightsFacebook(ctx context.Context, accessToken string, id string) (context.Context, *FacebookInsights, error) {
	ctx, logger := GetLogger(ctx)
	start := time.Now()
	defer func() {
		logger.Debugw("facebook insights", "elapsedMS", FormatElapsedMS(start))
		AddCtxStatsAPI(ctx, ServerStatAPIFacebook, "facebook insights", time.Since(start))
	}()
	if id == "" {
		return ctx, &FacebookInsights{}, nil
	}

	//create the url
	url := fmt.Sprintf(FacebookURLInsights, id)

	//create the request
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "facebook insights http request")
	}

	//add the query string
	query := request.URL.Query()
	query.Add(FacebookParamAccessToken, accessToken)
	query.Add(FacebookParamDatePreset, FacebookDatePresetsLifetime)
	query.Add(FacebookParamFields, FacebookInsightFields)
	request.URL.RawQuery = query.Encode()

	//make the request
	client := createClientFacebook()
	response, err := client.Do(request)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "facebook insights http request")
	}
	if response.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(response.Body)
		return ctx, nil, fmt.Errorf("status: %d: %s", response.StatusCode, body)
	}

	//process the response
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "facebook insights read body")
	}
	var insights FacebookInsightsData
	err = json.Unmarshal(body, &insights)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "facebook insights unjson")
	}
	defer response.Body.Close()

	//read the insight data
	if len(insights.Insights) > 0 {
		return ctx, insights.Insights[0], nil
	}
	return ctx, &FacebookInsights{}, nil
}

//GetAdFacebook : get Facebook ad
func GetAdFacebook(ctx context.Context, accessToken string, id string) (context.Context, *FacebookAd, error) {
	ctx, logger := GetLogger(ctx)
	start := time.Now()
	defer func() {
		logger.Debugw("facebook ad", "elapsedMS", FormatElapsedMS(start))
		AddCtxStatsAPI(ctx, ServerStatAPIFacebook, "facebook ad", time.Since(start))
	}()

	//create the url
	url := fmt.Sprintf(FacebookURLAd, id)

	//create the request
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "facebook ad http request")
	}

	//add the query string
	query := request.URL.Query()
	query.Add(FacebookParamAccessToken, accessToken)
	query.Add(FacebookParamFields, FacebookAdFields)
	request.URL.RawQuery = query.Encode()

	//make the request
	client := createClientFacebook()
	response, err := client.Do(request)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "facebook ad http request")
	}
	if response.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(response.Body)
		return ctx, nil, fmt.Errorf("status: %d: %s", response.StatusCode, body)
	}

	//process the response
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "facebook ad read body")
	}
	var ad FacebookAd
	err = json.Unmarshal(body, &ad)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "facebook ad unjson")
	}
	defer response.Body.Close()
	return ctx, &ad, nil
}

//GetAdPreviewFacebook : get Facebook ad preview
func GetAdPreviewFacebook(ctx context.Context, accessToken string, id string) (context.Context, *FacebookAdPreview, error) {
	ctx, logger := GetLogger(ctx)
	start := time.Now()
	defer func() {
		logger.Debugw("facebook ad preview", "elapsedMS", FormatElapsedMS(start))
		AddCtxStatsAPI(ctx, ServerStatAPIFacebook, "facebook ad preview", time.Since(start))
	}()

	//create the url
	url := fmt.Sprintf(FacebookURLAdPreviews, id)

	//create the request
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "facebook ad preview http request")
	}

	//add the query string
	query := request.URL.Query()
	query.Add(FacebookParamAccessToken, accessToken)
	query.Add(FacebookParamAdFormat, string(FacebookAdFormatDesktopFeedStandard))
	query.Add(FacebookParamHeight, strconv.FormatInt(FacebookPreviewHeight, 10))
	query.Add(FacebookParamWidth, strconv.FormatInt(FacebookPreviewWidth, 10))
	request.URL.RawQuery = query.Encode()

	//make the request
	client := createClientFacebook()
	response, err := client.Do(request)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "facebook ad preview http request")
	}
	if response.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(response.Body)
		return ctx, nil, fmt.Errorf("status: %d: %s", response.StatusCode, body)
	}

	//process the response
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "facebook ad preview read body")
	}
	var previews FacebookAdPreviewData
	err = json.Unmarshal(body, &previews)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "facebook ad preview unjson")
	}
	defer response.Body.Close()

	//read the insight data
	if len(previews.Previews) > 0 {
		return ctx, previews.Previews[0], nil
	}
	return ctx, nil, nil
}

//GetAccountsFacebook : get Facebook accounts
func GetAccountsFacebook(ctx context.Context, accessToken string) (context.Context, []*FacebookAccount, error) {
	ctx, logger := GetLogger(ctx)
	start := time.Now()
	defer func() {
		logger.Debugw("facebook accounts", "elapsedMS", FormatElapsedMS(start))
		AddCtxStatsAPI(ctx, ServerStatAPIFacebook, "facebook accounts", time.Since(start))
	}()

	//create the request
	request, err := http.NewRequest("GET", FacebookURLAccounts, nil)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "facebook accounts http request")
	}

	//add the query string
	query := request.URL.Query()
	query.Add(FacebookParamAccessToken, accessToken)
	request.URL.RawQuery = query.Encode()

	//make the request
	client := createClientFacebook()
	response, err := client.Do(request)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "facebook accounts http request")
	}
	if response.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(response.Body)
		return ctx, nil, fmt.Errorf("status: %d: %s", response.StatusCode, body)
	}

	//process the response
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "facebook accounts read body")
	}
	var accounts FacebookAccountData
	err = json.Unmarshal(body, &accounts)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "facebook accounts unjson")
	}
	defer response.Body.Close()

	//read the insight data
	if len(accounts.Accounts) > 0 {
		return ctx, accounts.Accounts, nil
	}
	return ctx, nil, nil
}

//GetUserFacebook : get Facebook user
func GetUserFacebook(ctx context.Context, accessToken string) (context.Context, *FacebookUser, error) {
	ctx, logger := GetLogger(ctx)
	start := time.Now()
	defer func() {
		logger.Debugw("facebook user", "elapsedMS", FormatElapsedMS(start))
		AddCtxStatsAPI(ctx, ServerStatAPIFacebook, "facebook user", time.Since(start))
	}()

	//create the request
	request, err := http.NewRequest("GET", FacebookURLUser, nil)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "facebook user http request")
	}

	//add the query string
	query := request.URL.Query()
	query.Add(FacebookParamAccessToken, accessToken)
	request.URL.RawQuery = query.Encode()

	//make the request
	client := createClientFacebook()
	response, err := client.Do(request)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "facebook user http request")
	}
	if response.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(response.Body)
		return ctx, nil, fmt.Errorf("status: %d: %s", response.StatusCode, body)
	}

	//process the response
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "facebook user read body")
	}
	var user FacebookUser
	err = json.Unmarshal(body, &user)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "facebook user unjson")
	}
	defer response.Body.Close()
	user.ID = FormatFacebookID(user.ID)
	return ctx, &user, nil
}

//FormatFacebookID : format the Facebook OAuth id
func FormatFacebookID(id string) string {
	return fmt.Sprintf("%s:%s", OAuthFacebook, id)
}
