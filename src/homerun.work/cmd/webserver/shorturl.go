package main

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pkg/errors"
)

//short url db tables
const (
	dbTableURLShort = "url_short"
)

//URLShortLength : length of the shortened url
const URLShortLength = 10

//LoadURL : load a shortened url
func LoadURL(ctx context.Context, db *DB, urlShort string) (context.Context, string, error) {
	stmt := fmt.Sprintf("SELECT url FROM %s WHERE url_short=?", dbTableURLShort)
	ctx, row, err := db.QueryRow(ctx, stmt, urlShort)
	if err != nil {
		return ctx, "", errors.Wrap(err, "query row url short")
	}

	//read the row
	var url string
	err = row.Scan(&url)
	if err != nil {
		if err == sql.ErrNoRows {
			return ctx, "", nil
		}
		return ctx, "", errors.Wrap(err, "select url short")
	}
	return ctx, url, nil
}

//SaveURL : save a url
func SaveURL(ctx context.Context, db *DB, url string) (context.Context, string, error) {
	//create the short url
	urlShort := GenURLStringRndm(URLShortLength)

	//save
	stmt := fmt.Sprintf("INSERT INTO %s(id,url_short,url) VALUES (UUID_TO_BIN(UUID()),?,?)", dbTableURLShort)
	ctx, result, err := db.Exec(ctx, stmt, urlShort, url)
	if err != nil {
		return ctx, urlShort, errors.Wrap(err, "insert url short")
	}
	count, err := result.RowsAffected()
	if err != nil {
		return ctx, urlShort, errors.Wrap(err, "insert url short rows affected")
	}
	if count != 1 {
		return ctx, urlShort, fmt.Errorf("unable to insert url short: %s", url)
	}
	return ctx, urlShort, nil
}
