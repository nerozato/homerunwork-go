package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

//testimonial db tables
const (
	dbTableTestimonial = "testimonial"
)

//Testimonial : definition of a provider testimonial
type Testimonial struct {
	ID         *uuid.UUID `json:"-"`
	ProviderID *uuid.UUID `json:"-"`
	UserID     *uuid.UUID `json:"-"`
	Name       string     `json:"Name"`
	City       string     `json:"City"`
	Text       string     `json:"Text"`
	Img        *Img       `json:"-"`
}

//SetImg : set the image
func (t *Testimonial) SetImg(file string) {
	t.Img = &Img{
		Version: time.Now().Unix(),
	}
	t.Img.SetFile(file)
}

//DeleteImg : delete the image
func (t *Testimonial) DeleteImg() {
	t.Img = nil
}

//LoadTestimonialByProviderIDAndID : load a testimonial by provider id and email
func LoadTestimonialByProviderIDAndID(ctx context.Context, db *DB, provider *Provider, id *uuid.UUID) (context.Context, *Testimonial, error) {
	stmtImgSelect := CreateImgSelect("img")
	stmtImg := CreateImgJoin("t", "secondary_id", "img", ImgTypeTestimonial, 0)
	stmt := fmt.Sprintf("SELECT t.data,%s FROM %s t %s WHERE t.deleted=0 AND t.provider_id=UUID_TO_BIN(?) AND t.id=UUID_TO_BIN(?)", stmtImgSelect, dbTableTestimonial, stmtImg)
	ctx, row, err := db.QueryRow(ctx, stmt, provider.ID, id)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "query row testimonial")
	}

	//read the row
	var dataStr string

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
	err = row.Scan(&dataStr, &imgIDStr, &imgUserIDStr, &imgProviderIDStr, &imgSecondaryIDStr, &imgImgType, &imgFilePath, &imgFileSrc, &imgFileResized, &imgIndex, &imgDataStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return ctx, nil, fmt.Errorf("no testimonial: %s: %s", provider.ID, id)
		}
		return ctx, nil, errors.Wrap(err, "select testimonial")
	}

	//unmarshal the data
	var testimonial *Testimonial
	err = json.Unmarshal([]byte(dataStr), &testimonial)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "unjson testimonial")
	}
	testimonial.ID = id
	testimonial.ProviderID = provider.ID
	testimonial.UserID = provider.User.ID

	//read the image
	img, err := CreateImg(imgIDStr, imgUserIDStr, imgSecondaryIDStr, imgProviderIDStr, imgImgType, imgFilePath, imgFileSrc, imgFileResized, imgIndex, imgDataStr)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "read image")
	}
	testimonial.Img = img
	return ctx, testimonial, nil
}

//SaveTestimonial : save a testimonial
func SaveTestimonial(ctx context.Context, db *DB, testimonial *Testimonial) (context.Context, error) {
	ctx, err := db.ProcessTx(ctx, "save testimonial", func(ctx context.Context, db *DB) (context.Context, error) {
		//create the testimonial id
		if testimonial.ID == nil {
			id, err := uuid.NewV4()
			if err != nil {
				return ctx, errors.Wrap(err, "new uuid testimonial")
			}
			testimonial.ID = &id
		}

		//json encode the testimonial data
		testimonialJSON, err := json.Marshal(testimonial)
		if err != nil {
			return ctx, errors.Wrap(err, "json testimonial")
		}

		//save to the db
		stmt := fmt.Sprintf("INSERT INTO %s(id,provider_id,data) VALUES (UUID_TO_BIN(?),UUID_TO_BIN(?),?) ON DUPLICATE KEY UPDATE data=VALUES(data)", dbTableTestimonial)
		ctx, result, err := db.Exec(ctx, stmt, testimonial.ID, testimonial.ProviderID, testimonialJSON)
		if err != nil {
			return ctx, errors.Wrap(err, "insert testimonial")
		}
		count, err := result.RowsAffected()
		if err != nil {
			return ctx, errors.Wrap(err, "insert testimonial rows affected")
		}

		//0 indicated no update, 1 an insert, 2 an update
		if count < 0 || count > 2 {
			return ctx, fmt.Errorf("unable to insert testimonial: %s", testimonial.ProviderID)
		}

		//process the image
		ctx, err = ProcessImgSingle(ctx, db, testimonial.UserID, testimonial.ProviderID, testimonial.ID, ImgTypeTestimonial, testimonial.Img)
		if err != nil {
			return ctx, errors.Wrap(err, "insert testimonial process image")
		}
		return ctx, nil
	})
	if err != nil {
		return ctx, errors.Wrap(err, "save testimonial")
	}
	return ctx, nil
}

//DeleteTestimonial : delete a testimonial
func DeleteTestimonial(ctx context.Context, db *DB, providerID *uuid.UUID, id *uuid.UUID) (context.Context, error) {
	stmt := fmt.Sprintf("UPDATE %s SET deleted=1 WHERE provider_id=UUID_TO_BIN(?) AND id=UUID_TO_BIN(?)", dbTableTestimonial)
	ctx, result, err := db.Exec(ctx, stmt, providerID, id)
	if err != nil {
		return ctx, errors.Wrap(err, "delete testimonial")
	}
	count, err := result.RowsAffected()
	if err != nil {
		return ctx, errors.Wrap(err, "delete testimonial rows affected")
	}
	if count == 0 {
		return ctx, fmt.Errorf("delete testimonial error: %s", id)
	}
	return ctx, nil
}

//ListTestimonialsByProviderID : list all testimonials for the provider
func ListTestimonialsByProviderID(ctx context.Context, db *DB, provider *Provider) (context.Context, []*Testimonial, error) {
	ctx, logger := GetLogger(ctx)
	stmtImgSelect := CreateImgSelect("img")
	stmtImg := CreateImgJoin("t", "secondary_id", "img", ImgTypeTestimonial, 0)
	stmt := fmt.Sprintf("SELECT BIN_TO_UUID(t.id),t.data,%s FROM %s t %s WHERE t.deleted=0 AND t.provider_id=UUID_TO_BIN(?) ORDER BY t.created", stmtImgSelect, dbTableTestimonial, stmtImg)
	ctx, rows, err := db.Query(ctx, stmt, provider.ID)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "select testimonials")
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			logger.Warnw("rows close", "error", err)
		}
	}()

	//read the rows
	testimonials := make([]*Testimonial, 0, 2)
	var idStr string
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
		err := rows.Scan(&idStr, &dataStr, &imgIDStr, &imgUserIDStr, &imgProviderIDStr, &imgSecondaryIDStr, &imgImgType, &imgFilePath, &imgFileSrc, &imgFileResized, &imgIndex, &imgDataStr)
		if err != nil {
			return ctx, nil, errors.Wrap(err, "rows scan testimonials")
		}

		//parse the uuid
		id, err := uuid.FromString(idStr)
		if err != nil {
			return ctx, nil, errors.Wrap(err, "parse uuid")
		}

		//unmarshal the data
		var testimonial Testimonial
		err = json.Unmarshal([]byte(dataStr), &testimonial)
		if err != nil {
			return ctx, nil, errors.Wrap(err, "unjson testimonial")
		}
		testimonial.ID = &id
		testimonial.ProviderID = provider.ID
		testimonial.UserID = provider.User.ID
		testimonials = append(testimonials, &testimonial)

		//read the image
		img, err := CreateImg(imgIDStr, imgUserIDStr, imgSecondaryIDStr, imgProviderIDStr, imgImgType, imgFilePath, imgFileSrc, imgFileResized, imgIndex, imgDataStr)
		if err != nil {
			return ctx, nil, errors.Wrap(err, "read image")
		}
		testimonial.Img = img
	}
	return ctx, testimonials, nil
}
