package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

//campaign db tables
const (
	dbTableCampaign = "campaign"
)

//campaign constants
const (
	AgeMin                       = 16
	AgeMax                       = 99
	CampaignBudgetDefault        = 5.0
	CampaignBudgetMin            = 2
	CampaignDurationDefault      = 5 * 24 * time.Hour
	CampaignFee                  = 10.0
	CampaignFeeFacebookAdAccount = 15.0
	CampaignFeeFacebookPage      = 25.0
	CampaignStartPadding         = 2 * 24 * time.Hour
)

//CampaignPlatform : campaign platform
type CampaignPlatform string

//campaign platforms
const (
	CampaignPlatformFacebook CampaignPlatform = "Facebook"
)

//Gender : gender definition
type Gender string

//genders
const (
	GenderAll   Gender = "All"
	GenderMen          = "Men"
	GenderWomen        = "Women"
)

//ParseGender : parse a gender
func ParseGender(in string) Gender {
	if in == "" {
		return ""
	}
	switch in {
	case string(GenderAll):
		return GenderAll
	case string(GenderMen):
		return GenderMen
	case string(GenderWomen):
		return GenderWomen
	}
	return ""
}

//CampaignStatus : campaign status
type CampaignStatus string

//campaign statuses
const (
	CampaignStatusInProgress = "In Progress"
	CampaignStatusPublished  = "Published"
	CampaignStatusStopped    = "Stopped"
	CampaignStatusSubmitted  = "Submitted"
)

//CampaignStatuses : campaign statuses
var CampaignStatuses []CampaignStatus = []CampaignStatus{
	CampaignStatusSubmitted,
	CampaignStatusInProgress,
	CampaignStatusPublished,
	CampaignStatusStopped,
}

//ParseCampaignStatus : parse a campaign status
func ParseCampaignStatus(in string) *CampaignStatus {
	if in == "" {
		return nil
	}
	for _, status := range CampaignStatuses {
		if in == string(status) {
			return &status
		}
	}
	return nil
}

//Campaign : definition of a provider campaign
type Campaign struct {
	ID                   *uuid.UUID       `json:"-"`
	UserID               *uuid.UUID       `json:"-"`
	ProviderID           *uuid.UUID       `json:"-"`
	ExternalID           *uuid.UUID       `json:"-"`
	ProviderName         string           `json:"ProviderName"`
	ServiceID            *uuid.UUID       `json:"ServiceId"`
	ServiceName          *string          `json:"ServiceName"`
	AgeMin               int              `json:"AgeMin"`
	AgeMax               int              `json:"AgeMax"`
	Budget               float32          `json:"Budget"`
	Gender               Gender           `json:"Gender"`
	Interests            string           `json:"Interests"`
	Locations            string           `json:"Locations"`
	Platform             CampaignPlatform `json:"Platform"`
	Text                 string           `json:"Text"`
	Start                time.Time        `json:"Start"`
	End                  time.Time        `json:"End"`
	HasFacebookAdAccount bool             `json:"HasFacebookAdAccount"`
	HasFacebookPage      bool             `json:"HasFacebookPage"`
	URLFacebook          string           `json:"FacebookUrl"`
	Status               CampaignStatus   `json:"Status"`
	Paid                 bool             `json:"Paid"`
	Deleted              bool             `json:"-"`
	Img                  *Img             `json:"-"`
}

//IsPublished : flag indicating if the campaign is published
func (c *Campaign) IsPublished() bool {
	switch c.Status {
	case CampaignStatusPublished:
		fallthrough
	case CampaignStatusStopped:
		return true
	}
	return false
}

//GetDurationDays : get the duration days
func (c *Campaign) GetDurationDays() int {
	duration := c.End.Sub(c.Start)
	days := int(math.Ceil(duration.Hours() / 24))
	return days
}

//GetBudgetTotal : get the total budget
func (c *Campaign) GetBudgetTotal() float32 {
	duration := c.GetDurationDays()
	return float32(duration) * c.Budget
}

//GetFee : get the fee for the campaign
func (c *Campaign) GetFee() float32 {
	fee := CampaignFee
	if !c.HasFacebookAdAccount {
		fee += CampaignFeeFacebookAdAccount
	}
	if !c.HasFacebookPage {
		fee += CampaignFeeFacebookPage
	}
	return float32(fee)
}

//FormatBudget : format the budget
func (c *Campaign) FormatBudget() string {
	return fmt.Sprintf("%s per day", FormatPrice(c.Budget))
}

//FormatName : format the name
func (c *Campaign) FormatName() string {
	return fmt.Sprintf("%s: %s", c.ProviderName, c.ID)
}

//FormatBudgetTotal : format the total budget
func (c *Campaign) FormatBudgetTotal() string {
	budgetTotal := c.GetBudgetTotal()
	return FormatPrice(budgetTotal)
}

//FormatFee : format the fee
func (c *Campaign) FormatFee() string {
	return FormatPrice(c.GetFee())
}

//FormatDuration : format the duration
func (c *Campaign) FormatDuration() string {
	days := c.GetDurationDays()
	if days == 1 {
		return "1 day"
	}
	return fmt.Sprintf("%d days", days)
}

//FormatService : format the service tied to the campaign
func (c *Campaign) FormatService() string {
	if c.ServiceName == nil {
		return "All Services"
	}
	return *c.ServiceName
}

//FormatStart : format the start date
func (c *Campaign) FormatStart(timeZone string) string {
	return FormatDateLocal(c.Start, timeZone)
}

//FormatEnd : format the end date
func (c *Campaign) FormatEnd(timeZone string) string {
	return FormatDateLocal(c.End, timeZone)
}

//FormatPaymentDescription : format the description for payment
func (c *Campaign) FormatPaymentDescription(timeZone string) string {
	return fmt.Sprintf("Campaign for %s, %s - %s", c.FormatService(), c.FormatStart(timeZone), c.FormatEnd(timeZone))
}

//SetService : set the service
func (c *Campaign) SetService(svc *Service) {
	if svc != nil {
		c.ServiceID = svc.ID
		c.ServiceName = &svc.Name
	}
}

//SetImg : set the image
func (c *Campaign) SetImg(file string) {
	c.Img = &Img{
		Version: time.Now().Unix(),
	}
	c.Img.SetFile(file)
}

//SaveCampaign : save a campaign
func SaveCampaign(ctx context.Context, db *DB, campaign *Campaign) (context.Context, error) {
	ctx, err := db.ProcessTx(ctx, "save campaign", func(ctx context.Context, db *DB) (context.Context, error) {
		//create the campaign id
		if campaign.ID == nil {
			id, err := uuid.NewV4()
			if err != nil {
				return ctx, errors.Wrap(err, "new uuid campaign")
			}
			campaign.ID = &id

		}

		//create the external id
		if campaign.ExternalID == nil {
			id, err := uuid.NewV4()
			if err != nil {
				return ctx, errors.Wrap(err, "new uuid campaign external")
			}
			campaign.ExternalID = &id
		}

		//json encode the campaign data
		campaignJSON, err := json.Marshal(campaign)
		if err != nil {
			return ctx, errors.Wrap(err, "json campaign")
		}

		//save to the db
		stmt := fmt.Sprintf("INSERT INTO %s(id,user_id,provider_id,external_id,data,deleted) VALUES (UUID_TO_BIN(?),UUID_TO_BIN(?),UUID_TO_BIN(?),UUID_TO_BIN(?),?,?) ON DUPLICATE KEY UPDATE data=VALUES(data),deleted=VALUES(deleted)", dbTableCampaign)
		ctx, result, err := db.Exec(ctx, stmt, campaign.ID, campaign.UserID, campaign.ProviderID, campaign.ExternalID, campaignJSON, campaign.Deleted)
		if err != nil {
			return ctx, errors.Wrap(err, "insert campaign")
		}
		count, err := result.RowsAffected()
		if err != nil {
			return ctx, errors.Wrap(err, "insert campaign rows affected")
		}

		//0 indicated no update, 1 an insert, 2 an update
		if count < 0 || count > 2 {
			return ctx, fmt.Errorf("unable to insert campaign: %s", campaign.ProviderID)
		}

		//process the image
		ctx, err = ProcessImgSingle(ctx, db, campaign.UserID, campaign.ProviderID, campaign.ID, ImgTypeAd, campaign.Img)
		if err != nil {
			return ctx, errors.Wrap(err, "insert campaign process image")
		}
		return ctx, nil
	})
	if err != nil {
		return ctx, errors.Wrap(err, "save campaign")
	}
	return ctx, nil
}

//load a campaign using the given where clause
func loadCampaign(ctx context.Context, db *DB, whereStmt string, args ...interface{}) (context.Context, *Campaign, error) {
	stmtImgSelect := CreateImgSelect("img")
	stmtImg := CreateImgJoin("c", "secondary_id", "img", ImgTypeAd, 0)
	stmt := fmt.Sprintf("SELECT BIN_TO_UUID(c.user_id),BIN_TO_UUID(c.provider_id),BIN_TO_UUID(c.id),BIN_TO_UUID(c.external_id),c.data,c.deleted,%s FROM %s c %s WHERE %s", stmtImgSelect, dbTableCampaign, stmtImg, whereStmt)
	ctx, row, err := db.QueryRow(ctx, stmt, args...)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "query row campaign")
	}

	//read the row
	var userIDStr string
	var providerIDStr string
	var idStr string
	var externalIDStr string
	var dataStr string
	var deletedBit string

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
	err = row.Scan(&userIDStr, &providerIDStr, &idStr, &externalIDStr, &dataStr, &deletedBit, &imgIDStr, &imgUserIDStr, &imgProviderIDStr, &imgSecondaryIDStr, &imgImgType, &imgFilePath, &imgFileSrc, &imgFileResized, &imgIndex, &imgDataStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return ctx, nil, nil
		}
		return ctx, nil, errors.Wrap(err, "select campaign")
	}

	//parse the uuid
	userID, err := uuid.FromString(userIDStr)
	if err != nil {
		return nil, nil, errors.Wrap(err, "parse uuid user id")
	}
	providerID, err := uuid.FromString(providerIDStr)
	if err != nil {
		return nil, nil, errors.Wrap(err, "parse uuid provider id")
	}
	id, err := uuid.FromString(idStr)
	if err != nil {
		return nil, nil, errors.Wrap(err, "parse uuid id")
	}
	externalID, err := uuid.FromString(externalIDStr)
	if err != nil {
		return nil, nil, errors.Wrap(err, "parse uuid external id")
	}

	//unmarshal the data
	var campaign Campaign
	err = json.Unmarshal([]byte(dataStr), &campaign)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "unjson campaign")
	}
	campaign.UserID = &userID
	campaign.ProviderID = &providerID
	campaign.ID = &id
	campaign.ExternalID = &externalID
	campaign.Deleted = deletedBit == "\x01"

	//read the image
	img, err := CreateImg(imgIDStr, imgUserIDStr, imgSecondaryIDStr, imgProviderIDStr, imgImgType, imgFilePath, imgFileSrc, imgFileResized, imgIndex, imgDataStr)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "read image")
	}
	campaign.Img = img
	return ctx, &campaign, nil
}

//LoadCampaignByID : load a campaign by id
func LoadCampaignByID(ctx context.Context, db *DB, id *uuid.UUID) (context.Context, *Campaign, error) {
	whereStmt := "c.id=UUID_TO_BIN(?)"
	return loadCampaign(ctx, db, whereStmt, id)
}

//LoadCampaignByExternalID : load a campaign by external id
func LoadCampaignByExternalID(ctx context.Context, db *DB, externalID *uuid.UUID) (context.Context, *Campaign, error) {
	whereStmt := "c.external_id=UUID_TO_BIN(?)"
	return loadCampaign(ctx, db, whereStmt, externalID)
}

//LoadCampaignByProviderIDAndID : load a campaign by provider id and id
func LoadCampaignByProviderIDAndID(ctx context.Context, db *DB, providerID *uuid.UUID, id *uuid.UUID, showDeleted bool) (context.Context, *Campaign, error) {
	whereStmt := "c.provider_id=UUID_TO_BIN(?) AND c.id=UUID_TO_BIN(?)"
	if !showDeleted {
		whereStmt = fmt.Sprintf("c.deleted=0 AND %s", whereStmt)
	}
	ctx, campaign, err := loadCampaign(ctx, db, whereStmt, providerID, id)
	if campaign == nil {
		return ctx, nil, fmt.Errorf("no campaign: %s: %s", providerID, id)
	}
	return ctx, campaign, err
}

//ListCampaignsByProviderID : list all campaigns for the provider
func ListCampaignsByProviderID(ctx context.Context, db *DB, provider *Provider) (context.Context, []*Campaign, error) {
	ctx, logger := GetLogger(ctx)
	stmtImgSelect := CreateImgSelect("img")
	stmtImg := CreateImgJoin("c", "secondary_id", "img", ImgTypeAd, 0)
	stmt := fmt.Sprintf("SELECT BIN_TO_UUID(c.user_id),BIN_TO_UUID(c.provider_id),BIN_TO_UUID(c.id),BIN_TO_UUID(c.external_id),c.data,%s FROM %s c %s WHERE c.deleted=0 AND c.provider_id=UUID_TO_BIN(?) ORDER BY c.created DESC", stmtImgSelect, dbTableCampaign, stmtImg)
	ctx, rows, err := db.Query(ctx, stmt, provider.ID)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "select campaigns")
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			logger.Warnw("rows close", "error", err)
		}
	}()

	//read the rows
	campaigns := make([]*Campaign, 0, 2)
	var userIDStr string
	var providerIDStr string
	var idStr string
	var externalIDStr string
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
		err := rows.Scan(&userIDStr, &providerIDStr, &idStr, &externalIDStr, &dataStr, &imgIDStr, &imgUserIDStr, &imgProviderIDStr, &imgSecondaryIDStr, &imgImgType, &imgFilePath, &imgFileSrc, &imgFileResized, &imgIndex, &imgDataStr)
		if err != nil {
			return ctx, nil, errors.Wrap(err, "rows scan campaigns")
		}

		//parse the uuid
		userID, err := uuid.FromString(userIDStr)
		if err != nil {
			return nil, nil, errors.Wrap(err, "parse uuid user id")
		}
		providerID, err := uuid.FromString(providerIDStr)
		if err != nil {
			return nil, nil, errors.Wrap(err, "parse uuid provider id")
		}
		id, err := uuid.FromString(idStr)
		if err != nil {
			return ctx, nil, errors.Wrap(err, "parse uuid")
		}
		externalID, err := uuid.FromString(externalIDStr)
		if err != nil {
			return nil, nil, errors.Wrap(err, "parse uuid external id")
		}

		//unmarshal the data
		var campaign Campaign
		err = json.Unmarshal([]byte(dataStr), &campaign)
		if err != nil {
			return ctx, nil, errors.Wrap(err, "unjson campaign")
		}
		campaign.UserID = &userID
		campaign.ProviderID = &providerID
		campaign.ID = &id
		campaign.ExternalID = &externalID

		//read the image
		img, err := CreateImg(imgIDStr, imgUserIDStr, imgSecondaryIDStr, imgProviderIDStr, imgImgType, imgFilePath, imgFileSrc, imgFileResized, imgIndex, imgDataStr)
		if err != nil {
			return ctx, nil, errors.Wrap(err, "read image")
		}
		campaign.Img = img
		campaigns = append(campaigns, &campaign)
	}
	return ctx, campaigns, nil
}

//DeleteCampaign : delete a campaign
func DeleteCampaign(ctx context.Context, db *DB, id *uuid.UUID) (context.Context, error) {
	stmt := fmt.Sprintf("UPDATE %s SET deleted=1 WHERE deleted=0 AND id=UUID_TO_BIN(?)", dbTableCampaign)
	ctx, _, err := db.Exec(ctx, stmt, id)
	if err != nil {
		return ctx, errors.Wrap(err, "delete campaign")
	}
	return ctx, nil
}
