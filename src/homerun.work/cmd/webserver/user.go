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

//user db tables
const (
	dbTableUser             = "user"
	dbTablePwdResetToken    = "pwd_reset_token"
	dbTableEmailVerifyToken = "email_verify_token"
)

//User : user definition
type User struct {
	ID                  *uuid.UUID    `json:"-"`
	Login               string        `json:"-"`
	Email               string        `json:"-"`
	EmailVerified       bool          `json:"-"`
	DisableEmails       bool          `json:"-"`
	IsOAuth             bool          `json:"-"`
	DisablePhone        bool          `json:"DisablePhone"`
	FirstName           string        `json:"FirstName"`
	LastName            string        `json:"LastName"`
	Phone               string        `json:"Phone"`
	TimeZone            string        `json:"TimeZone"`
	SignUpType          string        `json:"SignUpType"`
	OAuthFacebookData   *FacebookUser `json:"FacebookUser"`
	OAuthGoogleData     *UserGoogle   `json:"GoogleUser"`
	GoogleCalendarToken *TokenGoogle  `json:"GoogleCalendarToken"`

	//zoom
	ZoomUser  *UserZoom  `json:"ZoomUser"`
	ZoomToken *TokenZoom `json:"-"`
}

//GetEmail : get the email for use
func (u *User) GetEmail() string {
	if !u.DisableEmails {
		return u.Email
	}
	return ""
}

//GetPhone : get the phone number used for SMS
func (u *User) GetPhone() string {
	if !u.DisablePhone {
		return u.Phone
	}
	return ""
}

//FormatName : format the name
func (u *User) FormatName() string {
	return fmt.Sprintf("%s %s", u.FirstName, u.LastName)
}

//LoginExists : check if a login exists for a user
func LoginExists(ctx context.Context, db *DB, login string) (context.Context, *User, error) {
	stmt := fmt.Sprintf("SELECT BIN_TO_UUID(id),login,email,is_oauth,data FROM %s WHERE deleted=0 AND email=?", dbTableUser)
	ctx, row, err := db.QueryRow(ctx, stmt, login)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "query row user login")
	}

	//read the row
	var idStr string
	var email string
	var oauthBit string
	var dataStr string
	err = row.Scan(&idStr, &login, &email, &oauthBit, &dataStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return ctx, nil, nil
		}
		return ctx, nil, errors.Wrap(err, "select user login")
	}

	//parse the uuid
	id, err := uuid.FromString(idStr)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "parse uuid user")
	}

	//unmarshal the data
	var user User
	err = json.Unmarshal([]byte(dataStr), &user)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "unjson user")
	}
	user.ID = &id
	user.Login = login
	user.Email = email
	user.IsOAuth = oauthBit == "\x01"
	return ctx, &user, nil
}

//load a user
func loadUser(ctx context.Context, db *DB, whereStmt string, args ...interface{}) (context.Context, *User, error) {
	//create the final query
	stmt := fmt.Sprintf("SELECT BIN_TO_UUID(id),login,email,email_verified,disable_emails,is_oauth,token_zoom_data,data FROM %s WHERE %s", dbTableUser, whereStmt)
	ctx, row, err := db.QueryRow(ctx, stmt, args...)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "query row user")
	}

	//read the row
	var idStr string
	var login string
	var email string
	var emailVerifiedBit string
	var disableEmailsBit string
	var isOAuthBit string
	var tokenZoomData sql.NullString
	var dataStr string
	err = row.Scan(&idStr, &login, &email, &emailVerifiedBit, &disableEmailsBit, &isOAuthBit, &tokenZoomData, &dataStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return ctx, nil, nil
		}
		return ctx, nil, errors.Wrap(err, "select user")
	}

	//parse the uuid
	id, err := uuid.FromString(idStr)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "parse uuid user")
	}

	//unmarshal the data
	var user User
	err = json.Unmarshal([]byte(dataStr), &user)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "unjson user")
	}
	user.ID = &id
	user.Login = login
	user.Email = email
	user.EmailVerified = emailVerifiedBit == "\x01"
	user.DisableEmails = disableEmailsBit == "\x01"
	user.IsOAuth = isOAuthBit == "\x01"

	//check for zoom token data
	if tokenZoomData.Valid {
		var token TokenZoom
		err = json.Unmarshal([]byte(tokenZoomData.String), &token)
		if err != nil {
			return ctx, nil, errors.Wrap(err, "unjson zoom token")
		}
		user.ZoomToken = &token
	}
	return ctx, &user, nil
}

//LoadUserByLogin : load a user by login
func LoadUserByLogin(ctx context.Context, db *DB, login string) (context.Context, *User, error) {
	whereStmt := "deleted=0 AND login=?"
	return loadUser(ctx, db, whereStmt, login)
}

//LoadUserByID : load a user by id
func LoadUserByID(ctx context.Context, db *DB, id *uuid.UUID) (context.Context, *User, error) {
	whereStmt := "deleted=0 AND id=UUID_TO_BIN(?)"
	ctx, user, err := loadUser(ctx, db, whereStmt, id)
	if err != nil {
		return ctx, nil, err
	}
	if user == nil {
		return ctx, nil, fmt.Errorf("no user: %s", id)
	}
	return ctx, user, err
}

//SaveUser : save a user
func SaveUser(ctx context.Context, db *DB, user *User, pwd Secret) (context.Context, error) {
	ctx, err := db.ProcessTx(ctx, "save user", func(ctx context.Context, db *DB) (context.Context, error) {
		//generate a user id if necessary
		if user.ID == nil {
			userID, err := uuid.NewV4()
			if err != nil {
				return ctx, errors.Wrap(err, "new uuid user")
			}
			user.ID = &userID
		}

		//hash the password if necessary
		var err error
		var hash []byte
		if pwd != "" {
			//hash the password
			hash, err = HashSaltPassword(pwd)
			if err != nil {
				return ctx, errors.Wrap(err, "hash and salt password")
			}
		}

		//json encode the token data
		var tokenJSON []byte
		if user.ZoomToken != nil {
			tokenJSON, err = json.Marshal(user.ZoomToken)
			if err != nil {
				return ctx, errors.Wrap(err, "json zoom token")
			}
		}

		//json encode the user data
		userJSON, err := json.Marshal(user)
		if err != nil {
			return ctx, errors.Wrap(err, "json user")
		}

		//save to the db
		stmt := fmt.Sprintf("INSERT INTO %s(id,login,email,email_verified,disable_emails,is_oauth,password,token_zoom_data,data) VALUES (UUID_TO_BIN(?),?,?,?,?,?,?,?,?) ON DUPLICATE KEY UPDATE email=VALUES(email),email_verified=VALUES(email_verified),disable_emails=VALUES(disable_emails),is_oauth=VALUES(is_oauth),password=VALUES(password),token_zoom_data=VALUES(token_zoom_data),data=VALUES(data)", dbTableUser)
		ctx, result, err := db.Exec(ctx, stmt, user.ID, user.Login, user.Email, user.EmailVerified, user.DisableEmails, user.IsOAuth, hash, tokenJSON, userJSON)
		if err != nil {
			return ctx, errors.Wrap(err, "insert user")
		}
		count, err := result.RowsAffected()
		if err != nil {
			return ctx, errors.Wrap(err, "insert user rows affected")
		}

		//0 indicated no update, 1 an insert, 2 an update
		if count < 0 || count > 2 {
			return ctx, fmt.Errorf("unable to insert user: %s", user.Login)
		}

		//check if clients should be bound
		ctx, err = BindClientsToUser(ctx, db, user.ID, user.Email)
		if err != nil {
			return ctx, errors.Wrap(err, fmt.Sprintf("bind clients to user: %s: %s", user.ID, user.Email))
		}

		//update provider users
		stmt = fmt.Sprintf("UPDATE %s SET user_id=UUID_TO_BIN(?) WHERE login=? AND deleted=0", dbTableProviderUser)
		ctx, _, err = db.Exec(ctx, stmt, user.ID, user.Email)
		if err != nil {
			return ctx, errors.Wrap(err, "update provider user")
		}
		return ctx, nil
	})
	if err != nil {
		return ctx, errors.Wrap(err, "save user")
	}
	return ctx, nil
}

//ResetPassword : save a password based on a token and mark the reset token as used
func ResetPassword(ctx context.Context, db *DB, userID *uuid.UUID, pwd Secret, token string) (context.Context, error) {
	ctx, err := db.ProcessTx(ctx, "reset password", func(ctx context.Context, db *DB) (context.Context, error) {
		var result sql.Result

		//hash the password
		hash, err := HashSaltPassword(pwd)
		if err != nil {
			return ctx, errors.Wrap(err, "hash and salt password")
		}

		//save the password
		stmt := fmt.Sprintf("UPDATE %s SET password=? WHERE id=UUID_TO_BIN(?)", dbTableUser)
		ctx, result, err = db.Exec(ctx, stmt, hash, userID)
		if err != nil {
			return ctx, errors.Wrap(err, "update user password")
		}
		count, err := result.RowsAffected()
		if err != nil {
			return ctx, errors.Wrap(err, "update user password rows affected")
		}
		if count == 0 {
			return ctx, fmt.Errorf("unable to update user password: %s", userID)
		}

		//mark the token as used
		if token != "" {
			stmt = fmt.Sprintf("UPDATE %s SET deleted=1 WHERE deleted=0 AND token=? AND user_id=UUID_TO_BIN(?)", dbTablePwdResetToken)
			ctx, result, err = db.Exec(ctx, stmt, token, userID)
			if err != nil {
				return ctx, errors.Wrap(err, "delete pwd reset token")
			}
			count, err = result.RowsAffected()
			if err != nil {
				return ctx, errors.Wrap(err, "delete pwd reset token rows affected")
			}
			if count == 0 {
				return ctx, fmt.Errorf("unable to delete pwd reset token: %s", token)
			}
		}
		return ctx, nil
	})
	if err != nil {
		return ctx, errors.Wrap(err, "save password")
	}
	return ctx, nil
}

//VerifyEmail : verify a user's email based on a token and mark the token as used
func VerifyEmail(ctx context.Context, db *DB, userID *uuid.UUID, token string) (context.Context, error) {
	ctx, err := db.ProcessTx(ctx, "verify email", func(ctx context.Context, db *DB) (context.Context, error) {
		var err error
		var result sql.Result

		//save the password
		stmt := fmt.Sprintf("UPDATE %s SET email_verified=1 WHERE id=UUID_TO_BIN(?)", dbTableUser)
		ctx, result, err = db.Exec(ctx, stmt, userID)
		if err != nil {
			return ctx, errors.Wrap(err, "update user email verified")
		}
		count, err := result.RowsAffected()
		if err != nil {
			return ctx, errors.Wrap(err, "update user email verified rows affected")
		}
		if count == 0 {
			return ctx, fmt.Errorf("unable to update user email verified: %s", userID)
		}

		//mark the token as used
		stmt = fmt.Sprintf("UPDATE %s SET deleted=1 WHERE deleted=0 AND user_id=UUID_TO_BIN(?) AND token=?", dbTableEmailVerifyToken)
		ctx, result, err = db.Exec(ctx, stmt, userID, token)
		if err != nil {
			return ctx, errors.Wrap(err, "delete email verification token")
		}
		count, err = result.RowsAffected()
		if err != nil {
			return ctx, errors.Wrap(err, "delete email verification token rows affected")
		}
		if count == 0 {
			return ctx, fmt.Errorf("delete to update email verification token: %s", token)
		}
		return ctx, nil
	})
	if err != nil {
		return ctx, errors.Wrap(err, "verify email")
	}
	return ctx, nil
}

//CheckPasswordUser : check the password for a user and return the user id
func CheckPasswordUser(ctx context.Context, db *DB, login string, pwd Secret) (context.Context, bool, *uuid.UUID, string, bool, error) {
	//load the password
	stmt := fmt.Sprintf("SELECT BIN_TO_UUID(id),password,login,is_oauth FROM %s WHERE deleted=0 AND email=?", dbTableUser)
	ctx, row, err := db.QueryRow(ctx, stmt, login)
	if err != nil {
		return ctx, false, nil, "", false, errors.Wrap(err, "query row password user")
	}

	//read the row
	var idStr string
	var hash sql.NullString
	var oauthBit string
	err = row.Scan(&idStr, &hash, &login, &oauthBit)
	if err != nil {
		if err == sql.ErrNoRows {
			return ctx, false, nil, "", false, nil
		}
		return ctx, false, nil, "", false, errors.Wrap(err, "select password user")
	}

	//check the password
	var ok bool
	if hash.Valid {
		ok, err = CheckPassword(hash.String, pwd)
		if err != nil {
			return ctx, false, nil, "", false, errors.Wrap(err, "check password")
		}
	}
	userID, err := uuid.FromString(idStr)
	if err != nil {
		return ctx, false, nil, "", false, errors.Wrap(err, "parse uuid")
	}
	return ctx, ok, &userID, login, oauthBit == "\x01", nil
}

//UpdateUserLastLogin : update the last login for a user
func UpdateUserLastLogin(ctx context.Context, db *DB, id *uuid.UUID) (context.Context, error) {
	stmt := fmt.Sprintf("UPDATE %s SET last_login=CURRENT_TIMESTAMP() WHERE id=UUID_TO_BIN(?)", dbTableUser)
	ctx, result, err := db.Exec(ctx, stmt, id)
	if err != nil {
		return ctx, errors.Wrap(err, "update user last login")
	}
	_, err = result.RowsAffected()
	if err != nil {
		return ctx, errors.Wrap(err, "update user last login rows affected")
	}
	return ctx, nil
}

//SavePwdResetToken : save a password reset token
func SavePwdResetToken(ctx context.Context, db *DB, userID *uuid.UUID, token string, expiration int64) (context.Context, error) {
	stmt := fmt.Sprintf("INSERT INTO %s(user_id,token,expiration) VALUES (UUID_TO_BIN(?),?,?)", dbTablePwdResetToken)
	ctx, result, err := db.Exec(ctx, stmt, userID, token, expiration)
	if err != nil {
		return ctx, errors.Wrap(err, "insert pwd reset token")
	}
	count, err := result.RowsAffected()
	if err != nil {
		return ctx, errors.Wrap(err, "insert pwd reset token rows affected")
	}
	if count == 0 {
		return ctx, fmt.Errorf("unable to insert pwd reset token: %s", userID)
	}
	return ctx, nil
}

//CheckPwdResetToken : check if a password reset token is valid
func CheckPwdResetToken(ctx context.Context, db *DB, token string, time int64) (context.Context, bool, *uuid.UUID, error) {
	stmt := fmt.Sprintf("SELECT BIN_TO_UUID(user_id) FROM %s WHERE deleted=0 AND token=? AND expiration>?", dbTablePwdResetToken)
	ctx, row, err := db.QueryRow(ctx, stmt, token, time)
	if err != nil {
		return ctx, false, nil, errors.Wrap(err, "query row pwd reset token")
	}

	//read the row
	var idStr string
	err = row.Scan(&idStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return ctx, false, nil, nil
		}
		return ctx, false, nil, errors.Wrap(err, "select pwd reset token")
	}
	userID, err := uuid.FromString(idStr)
	if err != nil {
		return ctx, false, nil, errors.Wrap(err, "parse uuid")
	}
	return ctx, true, &userID, nil
}

//SaveEmailVerifyToken : save an email verification token
func SaveEmailVerifyToken(ctx context.Context, db *DB, userID *uuid.UUID, token string, expiration int64) (context.Context, error) {
	stmt := fmt.Sprintf("INSERT INTO %s(user_id,token,expiration) VALUES (UUID_TO_BIN(?),?,?)", dbTableEmailVerifyToken)
	ctx, result, err := db.Exec(ctx, stmt, userID, token, expiration)
	if err != nil {
		return ctx, errors.Wrap(err, "insert pwd reset token")
	}
	count, err := result.RowsAffected()
	if err != nil {
		return ctx, errors.Wrap(err, "insert pwd reset token rows affected")
	}
	if count == 0 {
		return ctx, fmt.Errorf("unable to insert pwd reset token: %s", userID)
	}
	return ctx, nil
}

//CheckEmailVerifyToken : check if an email verification token is valid
func CheckEmailVerifyToken(ctx context.Context, db *DB, token string, time int64) (context.Context, bool, *uuid.UUID, error) {
	stmt := fmt.Sprintf("SELECT BIN_TO_UUID(user_id) FROM %s WHERE deleted=0 AND token=? AND expiration>?", dbTableEmailVerifyToken)
	ctx, row, err := db.QueryRow(ctx, stmt, token, time)
	if err != nil {
		return ctx, false, nil, errors.Wrap(err, "query row email verify token")
	}

	//read the row
	var idStr string
	err = row.Scan(&idStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return ctx, false, nil, nil
		}
		return ctx, false, nil, errors.Wrap(err, "select email verify token")
	}
	userID, err := uuid.FromString(idStr)
	if err != nil {
		return ctx, false, nil, errors.Wrap(err, "parse uuid")
	}
	return ctx, true, &userID, nil
}

//DeleteUserByEmail : delete a user by email
func DeleteUserByEmail(ctx context.Context, db *DB, email string) (context.Context, int64, error) {
	stmt := fmt.Sprintf("UPDATE %s SET deleted=1 WHERE email=?", dbTableUser)
	ctx, result, err := db.Exec(ctx, stmt, email)
	if err != nil {
		return ctx, 0, errors.Wrap(err, "delete user")
	}
	count, err := result.RowsAffected()
	if err != nil {
		return ctx, 0, errors.Wrap(err, "delete user rows affected")
	}
	return ctx, count, nil
}

//CountUsers : count users
func CountUsers(ctx context.Context, db *DB) (context.Context, int, error) {
	stmt := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE deleted=0 AND test=0", dbTableUser)
	ctx, row, err := db.QueryRow(ctx, stmt)
	if err != nil {
		return ctx, 0, errors.Wrap(err, "query row user count")
	}

	//read the row
	var count int
	err = row.Scan(&count)
	if err != nil {
		if err == sql.ErrNoRows {
			return ctx, 0, nil
		}
		return ctx, 0, errors.Wrap(err, "select user count")
	}
	return ctx, count, nil
}

//FindLatestUser : find the latest user create time
func FindLatestUser(ctx context.Context, db *DB) (context.Context, *User, *time.Time, error) {
	stmt := fmt.Sprintf("SELECT login,email,data,created FROM %s WHERE deleted=0 AND test=0 ORDER BY created DESC LIMIT 1", dbTableUser)
	ctx, row, err := db.QueryRow(ctx, stmt)
	if err != nil {
		return ctx, nil, nil, errors.Wrap(err, "query row user time create")
	}

	//read the row
	var login string
	var email string
	var dataStr string
	var t time.Time
	err = row.Scan(&login, &email, &dataStr, &t)
	if err != nil {
		if err == sql.ErrNoRows {
			return ctx, nil, nil, nil
		}
		return ctx, nil, nil, errors.Wrap(err, "select user time create")
	}

	//unmarshal the data
	var user User
	err = json.Unmarshal([]byte(dataStr), &user)
	if err != nil {
		return ctx, nil, nil, errors.Wrap(err, "unjson user")
	}
	user.Login = login
	user.Email = email
	return ctx, &user, &t, nil
}

//UpdateUserTokenZoom : update the user Zoom token
func UpdateUserTokenZoom(ctx context.Context, db *DB, user *User) (context.Context, error) {
	//json encode the zoom token data
	var err error
	var tokenJSON []byte
	if user.ZoomToken != nil {
		tokenJSON, err = json.Marshal(user.ZoomToken)
		if err != nil {
			return ctx, errors.Wrap(err, "json zoom token")
		}
	}

	//update
	stmt := fmt.Sprintf("UPDATE %s SET token_zoom_data=? WHERE id=UUID_TO_BIN(?)", dbTableUser)
	ctx, result, err := db.Exec(ctx, stmt, tokenJSON, user.ID)
	if err != nil {
		return ctx, errors.Wrap(err, "update user")
	}
	count, err := result.RowsAffected()
	if err != nil {
		return ctx, errors.Wrap(err, "update user rows affected")
	}
	if count != 1 {
		return ctx, fmt.Errorf("unable to update user: %s", user.ID)
	}
	return ctx, nil
}

//DeleteUserZoom : delete the Zoom support for a user
func DeleteUserZoom(ctx context.Context, db *DB, userID string) (context.Context, error) {
	stmt := fmt.Sprintf("UPDATE %s SET token_zoom_data=NULL,data=JSON_SET(data,'$.ZoomUser',NULL) WHERE data->>'$.ZoomUser.id'=?", dbTableUser)
	ctx, result, err := db.Exec(ctx, stmt, userID)
	if err != nil {
		return ctx, errors.Wrap(err, "update user zoom")
	}
	_, err = result.RowsAffected()
	if err != nil {
		return ctx, errors.Wrap(err, "update user zoom rows affected")
	}
	return ctx, nil
}
