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

//coupon db tables
const (
	dbTableCoupon = "coupon"
)

//CouponType : type of coupon
type CouponType string

//coupon types
const (
	CouponTypePercentage CouponType = "%"
	CouponTypeUSD                   = "USD"
)

//CouponTypes : coupon types
var CouponTypes []CouponType = []CouponType{
	CouponTypePercentage,
	CouponTypeUSD,
}

//ParseCouponType : parse a coupon type by label
func ParseCouponType(couponTypeStr string) *CouponType {
	if couponTypeStr == "" {
		return nil
	}
	for _, couponType := range CouponTypes {
		if couponTypeStr == string(couponType) {
			return &couponType
		}
	}
	return nil
}

//Coupon : definition of a provider coupon
type Coupon struct {
	ID          *uuid.UUID `json:"-"`
	ProviderID  *uuid.UUID `json:"-"`
	Type        CouponType `json:"Type"`
	Code        string     `json:"Code"`
	Value       float32    `json:"Value"`
	Start       time.Time  `json:"-"`
	End         time.Time  `json:"-"`
	Description string     `json:"Description"`
	ServiceID   *uuid.UUID `json:"ServiceID"`
	ServiceName string     `json:"ServiceName"`
	NewClients  bool       `json:"NewClients"`
}

//FormatValue : format the value
func (c *Coupon) FormatValue() string {
	//determine if a currency symbol is necessary
	var valueStr string
	switch c.Type {
	case CouponTypePercentage:
		valueStr = fmt.Sprintf("%s%%", FormatFloat(c.Value))
	case CouponTypeUSD:
		valueStr = FormatPrice(c.Value)
	}
	return fmt.Sprintf("%s OFF", valueStr)
}

//FormatService : format the service
func (c *Coupon) FormatService() string {
	if c.ServiceID != nil {
		return c.ServiceName
	}
	return "All Services"
}

//FormatTarget : format the target
func (c *Coupon) FormatTarget() string {
	if c.NewClients {
		return "New Clients"
	}
	return "All Clients"
}

//FormatStart : format the start date
func (c *Coupon) FormatStart(timeZone string) string {
	return FormatDateLocal(c.Start, timeZone)
}

//FormatEnd : format the end date
func (c *Coupon) FormatEnd(timeZone string) string {
	return FormatDateLocal(c.End, timeZone)
}

//SetService : set the service information
func (c *Coupon) SetService(svc *Service) {
	if svc != nil {
		c.ServiceID = svc.ID
		c.ServiceName = svc.Name
	}
	return
}

//AdjustPrice : adjust the price based on the coupon
func (c *Coupon) AdjustPrice(price float32, svcID *uuid.UUID, isNewClient bool, now time.Time) float32 {
	//check if the coupon is still valid
	if c.Start.After(now) || c.End.Before(now) {
		return price
	}

	//check if the coupon only applies to a service
	if c.ServiceID != nil && svcID != nil && c.ServiceID.String() != svcID.String() {
		return price
	}

	//check if the coupon applies to new clients
	if c.NewClients && !isNewClient {
		return price
	}

	//make the appropriate adjustment of the price
	switch c.Type {
	case CouponTypePercentage:
		price = price * (1 - (c.Value / 100))
	case CouponTypeUSD:
		price = price - c.Value
	}

	//prevent a negative price
	price = float32(math.Max(float64(price), 0))
	return price
}

//LoadCouponByProviderIDAndID : load a coupon by provider id and email
func LoadCouponByProviderIDAndID(ctx context.Context, db *DB, provider *Provider, id *uuid.UUID) (context.Context, *Coupon, error) {
	stmt := fmt.Sprintf("SELECT code,start,end,data FROM %s WHERE deleted=0 AND provider_id=UUID_TO_BIN(?) AND id=UUID_TO_BIN(?)", dbTableCoupon)
	ctx, row, err := db.QueryRow(ctx, stmt, provider.ID, id)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "query row coupon")
	}

	//read the row
	var code string
	var start time.Time
	var end time.Time
	var dataStr string
	err = row.Scan(&code, &start, &end, &dataStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return ctx, nil, fmt.Errorf("no coupon: %s: %s", provider.ID, id)
		}
		return ctx, nil, errors.Wrap(err, "select coupon")
	}

	//unmarshal the data
	var coupon Coupon
	err = json.Unmarshal([]byte(dataStr), &coupon)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "unjson coupon")
	}
	coupon.ID = id
	coupon.ProviderID = provider.ID
	coupon.Code = code
	coupon.Start = start
	coupon.End = end
	return ctx, &coupon, nil
}

//LoadCouponByProviderIDAndCode : load a coupon by provider id and code
func LoadCouponByProviderIDAndCode(ctx context.Context, db *DB, providerID *uuid.UUID, code string, now *time.Time) (context.Context, *Coupon, error) {
	//build the query, checking if a valid time check is required
	var stmt string
	var args []interface{}
	if now == nil {
		stmt = fmt.Sprintf("SELECT BIN_TO_UUID(id),start,end,data FROM %s WHERE deleted=0 AND provider_id=UUID_TO_BIN(?) AND code=UPPER(?)", dbTableCoupon)
		args = []interface{}{providerID, code}
	} else {
		stmt = fmt.Sprintf("SELECT BIN_TO_UUID(id),start,end,data FROM %s WHERE deleted=0 AND provider_id=UUID_TO_BIN(?) AND code=UPPER(?) AND start<=? AND end>?", dbTableCoupon)
		args = []interface{}{providerID, code, now, now}
	}
	ctx, row, err := db.QueryRow(ctx, stmt, args...)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "select coupon id")
	}

	//read the row
	var idStr string
	var start time.Time
	var end time.Time
	var dataStr string
	err = row.Scan(&idStr, &start, &end, &dataStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return ctx, nil, nil
		}
		return ctx, nil, errors.Wrap(err, "query row coupon")
	}

	//parse the uuid
	id, err := uuid.FromString(idStr)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "parse uuid")
	}

	//unmarshal the data
	var coupon Coupon
	err = json.Unmarshal([]byte(dataStr), &coupon)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "unjson coupon")
	}
	coupon.ID = &id
	coupon.ProviderID = providerID
	coupon.Code = code
	coupon.Start = start
	coupon.End = end
	return ctx, &coupon, nil
}

//SaveCoupon : save a coupon
func SaveCoupon(ctx context.Context, db *DB, coupon *Coupon) (context.Context, error) {
	ctx, err := db.ProcessTx(ctx, "save coupon", func(ctx context.Context, db *DB) (context.Context, error) {
		//check for an existing id based on the code
		if coupon.ID == nil {
			var err error
			var id *uuid.UUID
			ctx, existingCoupon, err := LoadCouponByProviderIDAndCode(ctx, db, coupon.ProviderID, coupon.Code, nil)
			if err != nil {
				return ctx, errors.Wrap(err, "select coupon code")
			}
			if existingCoupon == nil {
				tempID, err := uuid.NewV4()
				if err != nil {
					return ctx, errors.Wrap(err, "new uuid coupon")
				}
				id = &tempID
			} else {
				id = existingCoupon.ID
			}
			coupon.ID = id
		}

		//json encode the coupon data
		dataJSON, err := json.Marshal(coupon)
		if err != nil {
			return ctx, errors.Wrap(err, "json coupon")
		}

		//save to the db
		stmt := fmt.Sprintf("INSERT INTO %s(id,provider_id,code,start,end,data) VALUES (UUID_TO_BIN(?),UUID_TO_BIN(?),?,?,?,?) ON DUPLICATE KEY UPDATE code=VALUES(code),data=VALUES(data)", dbTableCoupon)
		ctx, result, err := db.Exec(ctx, stmt, coupon.ID, coupon.ProviderID, coupon.Code, coupon.Start, coupon.End, dataJSON)
		if err != nil {
			return ctx, errors.Wrap(err, "insert coupon")
		}
		count, err := result.RowsAffected()
		if err != nil {
			return ctx, errors.Wrap(err, "insert coupon rows affected")
		}

		//0 indicated no update, 1 an insert, 2 an update
		if count < 0 || count > 2 {
			return ctx, fmt.Errorf("unable to insert coupon: %s", coupon.ProviderID)
		}
		return ctx, nil
	})
	if err != nil {
		return ctx, errors.Wrap(err, "save coupon")
	}
	return ctx, nil
}

//DeleteCoupon : delete a coupon
func DeleteCoupon(ctx context.Context, db *DB, providerID *uuid.UUID, id *uuid.UUID) (context.Context, error) {
	stmt := fmt.Sprintf("UPDATE %s SET deleted=1 WHERE provider_id=UUID_TO_BIN(?) AND id=UUID_TO_BIN(?)", dbTableCoupon)
	ctx, result, err := db.Exec(ctx, stmt, providerID, id)
	if err != nil {
		return ctx, errors.Wrap(err, "delete coupon")
	}
	count, err := result.RowsAffected()
	if err != nil {
		return ctx, errors.Wrap(err, "delete coupon rows affected")
	}
	if count == 0 {
		return ctx, fmt.Errorf("delete coupon error: %s", id)
	}
	return ctx, nil
}

//ListCouponsByProviderID : list all coupons for the provider
func ListCouponsByProviderID(ctx context.Context, db *DB, provider *Provider) (context.Context, []*Coupon, error) {
	ctx, logger := GetLogger(ctx)
	stmt := fmt.Sprintf("SELECT BIN_TO_UUID(id),code,start,end,data FROM %s WHERE deleted=0 AND provider_id=UUID_TO_BIN(?) ORDER BY created", dbTableCoupon)
	ctx, rows, err := db.Query(ctx, stmt, provider.ID)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "select coupons")
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			logger.Warnw("rows close", "error", err)
		}
	}()

	//read the rows
	coupons := make([]*Coupon, 0, 2)
	var idStr string
	var code string
	var start time.Time
	var end time.Time
	var dataStr string
	for rows.Next() {
		err := rows.Scan(&idStr, &code, &start, &end, &dataStr)
		if err != nil {
			return ctx, nil, errors.Wrap(err, "rows scan coupons")
		}

		//parse the uuid
		id, err := uuid.FromString(idStr)
		if err != nil {
			return ctx, nil, errors.Wrap(err, "parse uuid")
		}

		//unmarshal the data
		var coupon Coupon
		err = json.Unmarshal([]byte(dataStr), &coupon)
		if err != nil {
			return ctx, nil, errors.Wrap(err, "unjson coupon")
		}
		coupon.ID = &id
		coupon.ProviderID = provider.ID
		coupon.Code = code
		coupon.Start = start
		coupon.End = end
		coupons = append(coupons, &coupon)
	}
	return ctx, coupons, nil
}
