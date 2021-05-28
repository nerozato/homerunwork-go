package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
)

//content tables
const (
	dbTableContent = "content"
)

//ContentType : content type
type ContentType int

//image types
const (
	ContentTypeAlert ContentType = iota + 1
	ContentTypeTips
)

//Content : definition of content
type Content struct {
	Type ContentType `json:"Type"`
}

//ContentAlert : definition of alert content
type ContentAlert struct {
	Content
	Title     string `json:"Title"`
	LinkURL   string `json:"LinkUrl"`
	LinkTitle string `json:"LinkTitle"`
}

//ContentTip : definition of tip content
type ContentTip struct {
	Title     string `json:"Title"`
	LinkURL   string `json:"LinkUrl"`
	LinkTitle string `json:"LinkTitle"`
	ImgURL    string `json:"ImgUrl"`
}

//ContentTips : definition of alert content
type ContentTips struct {
	Content
	Tips []ContentTip `json:"Tips"`
}

//save content
func saveContent(ctx context.Context, db *DB, contentType ContentType, contentData interface{}) (context.Context, error) {
	ctx, err := db.ProcessTx(ctx, "save content", func(ctx context.Context, db *DB) (context.Context, error) {
		//json encode the content data
		contentJSON, err := json.Marshal(contentData)
		if err != nil {
			return ctx, errors.Wrap(err, "json content")
		}

		//delete previous content
		stmt := fmt.Sprintf("UPDATE %s SET deleted=1 WHERE type=? AND deleted=0", dbTableContent)
		ctx, result, err := db.Exec(ctx, stmt, contentType)
		if err != nil {
			return ctx, errors.Wrap(err, "delete content")
		}

		//save to the db
		stmt = fmt.Sprintf("INSERT INTO %s(id,type,data) VALUES (UUID_TO_BIN(UUID()),?,?)", dbTableContent)
		ctx, result, err = db.Exec(ctx, stmt, contentType, contentJSON)
		if err != nil {
			return ctx, errors.Wrap(err, "insert content")
		}
		count, err := result.RowsAffected()
		if err != nil {
			return ctx, errors.Wrap(err, "insert content rows affected")
		}
		if count != 1 {
			return ctx, fmt.Errorf("unable to insert content")
		}
		return ctx, nil
	})
	if err != nil {
		return ctx, errors.Wrap(err, "save content")
	}
	return ctx, nil
}

//SaveContentAlert : save the alert content
func SaveContentAlert(ctx context.Context, db *DB, content *ContentAlert) (context.Context, error) {
	return saveContent(ctx, db, content.Type, *content)
}

//SaveContentTips : save the tips content
func SaveContentTips(ctx context.Context, db *DB, content *ContentTips) (context.Context, error) {
	return saveContent(ctx, db, content.Type, *content)
}

//load content by type
func loadContentByType(ctx context.Context, db *DB, contentType ContentType) (context.Context, *string, error) {
	//list the content
	stmt := fmt.Sprintf("SELECT data FROM %s WHERE type=? AND deleted=0 ORDER BY updated DESC", dbTableContent)
	ctx, row, err := db.QueryRow(ctx, stmt, contentType)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "query row content")
	}
	var dataStr string
	err = row.Scan(&dataStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return ctx, nil, nil
		}
		return ctx, nil, errors.Wrap(err, "row scan content")
	}
	return ctx, &dataStr, nil
}

//LoadContentAlert : list the alert content
func LoadContentAlert(ctx context.Context, db *DB) (context.Context, *ContentAlert, error) {
	//load the content
	ctx, dataStr, err := loadContentByType(ctx, db, ContentTypeAlert)
	if err != nil {
		return ctx, nil, errors.Wrap(err, fmt.Sprintf("load content: %d", ContentTypeAlert))
	}
	if dataStr == nil {
		return ctx, nil, nil
	}

	//unmarshal the data
	var content ContentAlert
	err = json.Unmarshal([]byte(*dataStr), &content)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "unjson content")
	}
	content.Type = ContentTypeAlert
	return ctx, &content, nil
}

//LoadContentTips : load the tips content
func LoadContentTips(ctx context.Context, db *DB) (context.Context, *ContentTips, error) {
	//load the content
	ctx, dataStr, err := loadContentByType(ctx, db, ContentTypeTips)
	if err != nil {
		return ctx, nil, errors.Wrap(err, fmt.Sprintf("list content: %d", ContentTypeTips))
	}
	if dataStr == nil {
		return ctx, nil, nil
	}

	//unmarshal the data
	var content ContentTips
	err = json.Unmarshal([]byte(*dataStr), &content)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "unjson content")
	}
	content.Type = ContentTypeTips
	return ctx, &content, nil
}
