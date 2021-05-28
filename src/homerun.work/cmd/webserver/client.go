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

//client db tables
const (
	dbTableClient = "client"
)

//Client : definition of a provider client
type Client struct {
	ID            *uuid.UUID `json:"-"`
	ProviderID    *uuid.UUID `json:"-"`
	UserID        *uuid.UUID `json:"-"`
	Email         string     `json:"-"`
	EmailPrevious string     `json:"-"`
	Invited       *time.Time `json:"-"`
	DisableEmails bool       `json:"-"`
	Name          string     `json:"Name"`
	Location      string     `json:"Location"`
	Phone         string     `json:"Phone"`
	TimeZone      string     `json:"TimeZone"`
}

//SetEmail : set the email
func (c *Client) SetEmail(email string) {
	if c.Email != email {
		c.EmailPrevious = email
	}
	c.Email = email
}

//GetEmail : get the email for use
func (c *Client) GetEmail() string {
	if !c.DisableEmails {
		return c.Email
	}
	return ""
}

//load a client
func loadClient(ctx context.Context, db *DB, whereStmt string, args ...interface{}) (context.Context, *Client, error) {
	//create the final query
	stmt := fmt.Sprintf("SELECT BIN_TO_UUID(id),BIN_TO_UUID(provider_id),email,invited,disable_emails,data FROM %s WHERE %s", dbTableClient, whereStmt)

	//load the client
	ctx, row, err := db.QueryRow(ctx, stmt, args...)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "query row client")
	}

	//read the row
	var idStr string
	var providerIDStr string
	var email string
	var invited sql.NullTime
	var disableEmailsBit string
	var dataStr string
	err = row.Scan(&idStr, &providerIDStr, &email, &invited, &disableEmailsBit, &dataStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return ctx, nil, nil
		}
		return ctx, nil, errors.Wrap(err, "select client")
	}

	//parse the uuid
	id, err := uuid.FromString(idStr)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "parse uuid client id")
	}
	providerID, err := uuid.FromString(providerIDStr)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "parse uuid provider id")
	}

	//unmarshal the data
	var client Client
	err = json.Unmarshal([]byte(dataStr), &client)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "unjson client")
	}
	client.ID = &id
	client.ProviderID = &providerID
	client.Email = email
	client.DisableEmails = disableEmailsBit == "\x01"
	if invited.Valid {
		client.Invited = &invited.Time
	}
	return ctx, &client, nil
}

//LoadClientByProviderIDAndEmail : load a client by provider id and email
func LoadClientByProviderIDAndEmail(ctx context.Context, db *DB, providerID *uuid.UUID, email string) (context.Context, *Client, error) {
	whereStmt := "deleted=0 AND provider_id=UUID_TO_BIN(?) AND email=?"
	return loadClient(ctx, db, whereStmt, providerID, email)
}

//LoadClientByProviderIDAndID : load a client by provider id and id
func LoadClientByProviderIDAndID(ctx context.Context, db *DB, providerID *uuid.UUID, id *uuid.UUID) (context.Context, *Client, error) {
	whereStmt := "deleted=0 AND provider_id=UUID_TO_BIN(?) AND id=UUID_TO_BIN(?)"
	return loadClient(ctx, db, whereStmt, providerID, id)
}

//LoadClientByID : load a client by id
func LoadClientByID(ctx context.Context, db *DB, id *uuid.UUID) (context.Context, *Client, error) {
	whereStmt := "deleted=0 AND id=UUID_TO_BIN(?)"
	return loadClient(ctx, db, whereStmt, id)
}

//SaveClient : save a client
func SaveClient(ctx context.Context, db *DB, client *Client) (context.Context, error) {
	ctx, err := db.ProcessTx(ctx, "save client", func(ctx context.Context, db *DB) (context.Context, error) {
		//attempt to find the appropriate client
		if client.ID == nil {
			//probe for an existing client
			ctx, loadedClient, err := LoadClientByProviderIDAndEmail(ctx, db, client.ProviderID, client.Email)
			if err != nil {
				return ctx, errors.Wrap(err, fmt.Sprintf("load client provider id and email: %s: %s", client.ProviderID, client.Email))
			}
			if loadedClient != nil {
				client.ID = loadedClient.ID
			} else {
				//generate an id
				clientID, err := uuid.NewV4()
				if err != nil {
					return ctx, errors.Wrap(err, "new uuid client")
				}
				client.ID = &clientID
			}
		} else if client.EmailPrevious != "" {
			//probe for an existing client
			ctx, loadedClient, err := LoadClientByProviderIDAndEmail(ctx, db, client.ProviderID, client.Email)
			if err != nil {
				return ctx, errors.Wrap(err, fmt.Sprintf("load client provider id and email: %s: %s", client.ProviderID, client.Email))
			}
			if loadedClient != nil {
				return ctx, fmt.Errorf("load client: %s: %s", client.ProviderID, client.Email)
			}
		}

		//json encode the client data
		clientJSON, err := json.Marshal(client)
		if err != nil {
			return ctx, errors.Wrap(err, "json client")
		}

		//save to the db
		stmt := fmt.Sprintf("INSERT INTO %s(id,provider_id,user_id,email,disable_emails,data) VALUES (UUID_TO_BIN(?),UUID_TO_BIN(?),UUID_TO_BIN(?),?,?,?) ON DUPLICATE KEY UPDATE email=VALUES(email),data=VALUES(data)", dbTableClient)
		ctx, result, err := db.Exec(ctx, stmt, client.ID, client.ProviderID, client.UserID, client.Email, client.DisableEmails, clientJSON)
		if err != nil {
			return ctx, errors.Wrap(err, "insert client")
		}
		count, err := result.RowsAffected()
		if err != nil {
			return ctx, errors.Wrap(err, "insert client rows affected")
		}

		//0 indicated no update, 1 an insert, 2 an update
		if count < 0 || count > 2 {
			return ctx, fmt.Errorf("unable to insert client: %s", client.Email)
		}
		return ctx, nil
	})
	if err != nil {
		return ctx, errors.Wrap(err, "save client")
	}
	return ctx, nil
}

//DeleteClient : delete a client, returning the number of bookings that may exists that would prevent the delete
func DeleteClient(ctx context.Context, db *DB, providerID *uuid.UUID, id *uuid.UUID) (context.Context, int, error) {
	var err error
	var booksCount int
	ctx, err = db.ProcessTx(ctx, "delete client", func(ctx context.Context, db *DB) (context.Context, error) {
		//check if there are any bookings, which will prevent the delete
		ctx, booksCount, err = CountBookingsForClient(ctx, db, id)
		if err != nil {
			return ctx, errors.Wrap(err, "delete client check booking")
		}
		if booksCount > 0 {
			return ctx, nil
		}

		//delete the client
		stmt := fmt.Sprintf("UPDATE %s SET deleted=1 WHERE provider_id=UUID_TO_BIN(?) AND id=UUID_TO_BIN(?)", dbTableClient)
		ctx, result, err := db.Exec(ctx, stmt, providerID, id)
		if err != nil {
			return ctx, errors.Wrap(err, "delete client")
		}
		count, err := result.RowsAffected()
		if err != nil {
			return ctx, errors.Wrap(err, "delete client rows affected")
		}
		if count == 0 {
			return ctx, fmt.Errorf("delete client error: %s", id)
		}
		return ctx, nil
	})
	if err != nil {
		return ctx, 0, errors.Wrap(err, "delete client")
	}
	return ctx, booksCount, nil
}

//ListClientsByProviderID : list all clients for the provider
func ListClientsByProviderID(ctx context.Context, db *DB, providerID *uuid.UUID) (context.Context, []*Client, error) {
	ctx, logger := GetLogger(ctx)
	stmt := fmt.Sprintf("SELECT BIN_TO_UUID(c.id),BIN_TO_UUID(c.provider_id),c.email,c.invited,c.disable_emails,c.data FROM %s c WHERE c.deleted=0 AND c.provider_id=UUID_TO_BIN(?) GROUP BY c.id ORDER BY c.email", dbTableClient)
	ctx, rows, err := db.Query(ctx, stmt, providerID)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "select clients")
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			logger.Warnw("rows close", "error", err)
		}
	}()

	//read the rows
	clients := make([]*Client, 0, 2)
	var idStr string
	var providerIDStr string
	var email string
	var disableEmailsBit string
	var dataStr string
	for rows.Next() {
		var invited sql.NullTime
		err := rows.Scan(&idStr, &providerIDStr, &email, &invited, &disableEmailsBit, &dataStr)
		if err != nil {
			return ctx, nil, errors.Wrap(err, "rows scan clients")
		}

		//parse the uuid
		id, err := uuid.FromString(idStr)
		if err != nil {
			return ctx, nil, errors.Wrap(err, "parse uuid client id")
		}
		providerID, err := uuid.FromString(providerIDStr)
		if err != nil {
			return ctx, nil, errors.Wrap(err, "parse uuid client provider id")
		}

		//unmarshal the data
		var client Client
		err = json.Unmarshal([]byte(dataStr), &client)
		if err != nil {
			return ctx, nil, errors.Wrap(err, "unjson client")
		}
		client.ID = &id
		client.ProviderID = &providerID
		client.Email = email
		client.DisableEmails = disableEmailsBit == "\x01"
		if invited.Valid {
			client.Invited = &invited.Time
		}
		clients = append(clients, &client)
	}
	return ctx, clients, nil
}

//UpdateClientInvited : update the client invited
func UpdateClientInvited(ctx context.Context, db *DB, id *uuid.UUID) (context.Context, error) {
	stmt := fmt.Sprintf("UPDATE %s SET invited=CURRENT_TIMESTAMP() WHERE id=UUID_TO_BIN(?)", dbTableClient)
	ctx, result, err := db.Exec(ctx, stmt, id)
	if err != nil {
		return ctx, errors.Wrap(err, "update client invited")
	}
	_, err = result.RowsAffected()
	if err != nil {
		return ctx, errors.Wrap(err, "update client invited rows affected")
	}
	return ctx, nil
}

//BindClientsToUser : bind clients to a user
func BindClientsToUser(ctx context.Context, db *DB, id *uuid.UUID, email string) (context.Context, error) {
	stmt := fmt.Sprintf("UPDATE %s SET user_id=UUID_TO_BIN(?) WHERE deleted=0 AND email=?", dbTableClient)
	ctx, result, err := db.Exec(ctx, stmt, id, email)
	if err != nil {
		return ctx, errors.Wrap(err, "update client email")
	}
	_, err = result.RowsAffected()
	if err != nil {
		return ctx, errors.Wrap(err, "update client email rows affected")
	}
	return ctx, nil
}
