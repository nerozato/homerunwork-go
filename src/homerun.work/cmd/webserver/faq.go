package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

//faq db tables
const (
	dbTableFaq = "faq"
)

//Faq : definition of a provider faq
type Faq struct {
	ID         *uuid.UUID `json:"-"`
	ProviderID *uuid.UUID `json:"-"`
	Question   string     `json:"Question"`
	Answer     string     `json:"Answer"`
}

//LoadFaqByProviderIDAndID : load a faq by provider id and email
func LoadFaqByProviderIDAndID(ctx context.Context, db *DB, provider *Provider, id *uuid.UUID) (context.Context, *Faq, error) {
	stmt := fmt.Sprintf("SELECT data FROM %s WHERE deleted=0 AND provider_id=UUID_TO_BIN(?) AND id=UUID_TO_BIN(?)", dbTableFaq)
	ctx, row, err := db.QueryRow(ctx, stmt, provider.ID, id)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "query row faq")
	}

	//read the row
	var dataStr string
	err = row.Scan(&dataStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return ctx, nil, fmt.Errorf("no faq: %s: %s", provider.ID, id)
		}
		return ctx, nil, errors.Wrap(err, "select faq")
	}

	//unmarshal the data
	var faq *Faq
	err = json.Unmarshal([]byte(dataStr), &faq)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "unjson faq")
	}
	faq.ID = id
	faq.ProviderID = provider.ID
	return ctx, faq, nil
}

//SaveFaq : save a faq
func SaveFaq(ctx context.Context, db *DB, faq *Faq) (context.Context, error) {
	//create the faq id
	if faq.ID == nil {
		id, err := uuid.NewV4()
		if err != nil {
			return ctx, errors.Wrap(err, "new uuid faq")
		}
		faq.ID = &id
	}

	//json encode the faq data
	dataJSON, err := json.Marshal(faq)
	if err != nil {
		return ctx, errors.Wrap(err, "json faq")
	}

	//save to the db
	stmt := fmt.Sprintf("INSERT INTO %s(id,provider_id,data) VALUES (UUID_TO_BIN(?),UUID_TO_BIN(?),?) ON DUPLICATE KEY UPDATE data=VALUES(data)", dbTableFaq)
	ctx, result, err := db.Exec(ctx, stmt, faq.ID, faq.ProviderID, dataJSON)
	if err != nil {
		return ctx, errors.Wrap(err, "insert faq")
	}
	count, err := result.RowsAffected()
	if err != nil {
		return ctx, errors.Wrap(err, "insert faq rows affected")
	}

	//0 indicated no update, 1 an insert, 2 an update
	if count < 0 || count > 2 {
		return ctx, fmt.Errorf("unable to insert faq: %s", faq.ProviderID)
	}
	return ctx, nil
}

//DeleteFaq : delete a faq
func DeleteFaq(ctx context.Context, db *DB, providerID *uuid.UUID, id *uuid.UUID) (context.Context, error) {
	stmt := fmt.Sprintf("UPDATE %s SET deleted=1 WHERE provider_id=UUID_TO_BIN(?) AND id=UUID_TO_BIN(?)", dbTableFaq)
	ctx, result, err := db.Exec(ctx, stmt, providerID, id)
	if err != nil {
		return ctx, errors.Wrap(err, "delete faq")
	}
	count, err := result.RowsAffected()
	if err != nil {
		return ctx, errors.Wrap(err, "delete faq rows affected")
	}
	if count == 0 {
		return ctx, fmt.Errorf("delete faq error: %s", id)
	}
	return ctx, nil
}

//ListFaqsByProviderID : list all faqs for the provider
func ListFaqsByProviderID(ctx context.Context, db *DB, provider *Provider) (context.Context, []*Faq, error) {
	ctx, logger := GetLogger(ctx)
	stmt := fmt.Sprintf("SELECT BIN_TO_UUID(id),data FROM %s WHERE deleted=0 AND provider_id=UUID_TO_BIN(?) ORDER BY idx,created", dbTableFaq)
	ctx, rows, err := db.Query(ctx, stmt, provider.ID)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "select faqs")
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			logger.Warnw("rows close", "error", err)
		}
	}()

	//read the rows
	faqs := make([]*Faq, 0, 2)
	var idStr string
	var dataStr string
	for rows.Next() {
		err := rows.Scan(&idStr, &dataStr)
		if err != nil {
			return ctx, nil, errors.Wrap(err, "rows scan faqs")
		}

		//parse the uuid
		id, err := uuid.FromString(idStr)
		if err != nil {
			return ctx, nil, errors.Wrap(err, "parse uuid")
		}

		//unmarshal the data
		var faq Faq
		err = json.Unmarshal([]byte(dataStr), &faq)
		if err != nil {
			return ctx, nil, errors.Wrap(err, "unjson faq")
		}
		faq.ID = &id
		faq.ProviderID = provider.ID
		faqs = append(faqs, &faq)
	}
	return ctx, faqs, nil
}

//CountFaqsByProviderID : list all faqs for the provider
func CountFaqsByProviderID(ctx context.Context, db *DB, provider *Provider) (context.Context, int, error) {
	stmt := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE deleted=0 AND provider_id=UUID_TO_BIN(?) ORDER BY idx,created", dbTableFaq)
	ctx, row, err := db.QueryRow(ctx, stmt, provider.ID)
	if err != nil {
		return ctx, 0, errors.Wrap(err, "count faqs")
	}

	//read the rows
	var count int
	err = row.Scan(&count)
	if err != nil {
		return ctx, 0, errors.Wrap(err, "row scan count faqs")
	}
	return ctx, count, nil
}

//UpdateFaqIndices : update faq indices
func UpdateFaqIndices(ctx context.Context, db *DB, ids []*uuid.UUID) (context.Context, error) {
	ctx, err := db.ProcessTx(ctx, "update faq indices", func(ctx context.Context, db *DB) (context.Context, error) {
		for i, id := range ids {
			stmt := fmt.Sprintf("UPDATE %s SET idx=? WHERE id=UUID_TO_BIN(?)", dbTableFaq)
			ctx, result, err := db.Exec(ctx, stmt, i, id)
			if err != nil {
				return ctx, errors.Wrap(err, fmt.Sprintf("update faq index: %d: %s", i, id))
			}
			_, err = result.RowsAffected()
			if err != nil {
				return ctx, errors.Wrap(err, fmt.Sprintf("update faq index rows affected: %d: %s", i, id))
			}
		}
		return ctx, nil
	})
	if err != nil {
		return ctx, errors.Wrap(err, "update faq indices")
	}
	return ctx, nil
}
