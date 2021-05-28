package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"
)

//jwt configuration
const (
	JWTExpirationSec      = 4 * 60 * 60 //4 hours
	JWTOAuthExpirationSec = 300
	JWTSigningAlgorithm   = "HS256"
)

//token configuration
const (
	resetPwdTokenByteLength    = 16
	resetPwdTokenExpiration    = 20 * time.Minute
	verifyEmailTokenByteLength = 16
	verifyEmailTokenExpiration = 24 * time.Hour
)

//oauth types
const (
	OAuthFacebook = "facebook"
	OAuthGoogle   = "google"
)

//EncryptString : encrypt a string
func EncryptString(in string) (string, error) {
	cphr, err := aes.NewCipher(GetTokenKey())
	if err != nil {
		return "", errors.Wrap(err, "new cipher")
	}
	gcm, err := cipher.NewGCM(cphr)
	if err != nil {
		return "", errors.Wrap(err, "new gcm")
	}
	nonce := make([]byte, gcm.NonceSize())
	_, err = io.ReadFull(rand.Reader, nonce)
	if err != nil {
		return "", errors.Wrap(err, "new nonce")
	}
	data := gcm.Seal(nonce, nonce, []byte(in), nil)

	//base64 encode
	out := base64.StdEncoding.EncodeToString(data)
	return out, nil
}

//DecryptString : decrypt a string
func DecryptString(in string) (string, error) {
	//base64 decode
	data, err := base64.StdEncoding.DecodeString(in)
	if err != nil {
		return "", errors.Wrap(err, "new nonce")
	}

	//decrypt the data
	cphr, err := aes.NewCipher(GetTokenKey())
	if err != nil {
		return "", errors.Wrap(err, "new cipher")
	}
	gcm, err := cipher.NewGCM(cphr)
	if err != nil {
		return "", errors.Wrap(err, "new gcm")
	}
	nonceSize := gcm.NonceSize()
	nonce := make([]byte, nonceSize)
	if len(data) < nonceSize {
		return "", fmt.Errorf("invalid size: %d", nonceSize)
	}
	nonce, dataEncrypted := data[:nonceSize], data[nonceSize:]
	out, err := gcm.Open(nil, nonce, dataEncrypted, nil)
	if err != nil {
		return "", errors.Wrap(err, "gcm open")
	}
	return string(out), nil
}

//HashSaltPassword : hash and salt the password
func HashSaltPassword(pwd Secret) ([]byte, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.Wrap(err, "hash password")
	}
	return hash, nil
}

//CheckPassword : check the password
func CheckPassword(storedPwd string, plainPwd Secret) (bool, error) {
	err := bcrypt.CompareHashAndPassword([]byte(storedPwd), []byte(plainPwd))
	if err != nil {
		return false, errors.Wrap(err, "compare password")
	}
	return true, nil
}

//AuthClaims : claims to include in the authentication JWT
type AuthClaims struct {
	jwt.StandardClaims
	UserID string `json:"user_id"`
}

//GenerateAuthToken : generate an authentication JWT using the given claims
func GenerateAuthToken(userID *uuid.UUID) (string, error) {
	//compute the expiration
	expiration := time.Now().Unix() + JWTExpirationSec

	//create the claims
	claims := &AuthClaims{
		UserID: userID.String(),
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expiration,
		},
	}

	//create the token
	algorithm := jwt.GetSigningMethod(JWTSigningAlgorithm)
	token := jwt.NewWithClaims(algorithm, claims)

	//create the signed string
	tokenStr, err := token.SignedString([]byte(GetJWTKey()))
	if err != nil {
		return "", errors.Wrap(err, "sign auth token")
	}
	return tokenStr, nil
}

//ValidateAuthToken : validate an authentication JWT
func ValidateAuthToken(tokenStr string) (bool, *uuid.UUID, error) {
	//initialize the claims
	claims := &AuthClaims{}

	//parse the JWT and load the claims
	token, err := jwt.ParseWithClaims(tokenStr, claims, getTokenKey)
	if err != nil {
		if err == jwt.ErrSignatureInvalid {
			return false, nil, nil
		}
		return false, nil, err
	}

	//verify the signing algorithm
	if token.Method.Alg() != JWTSigningAlgorithm {
		return false, nil, fmt.Errorf("invalid signing algorthm: %s", token.Method.Alg())
	}

	//check if the token is valid
	if !token.Valid {
		return false, nil, nil
	}

	//extract the user id
	userIDStr := claims.UserID
	userID := uuid.FromStringOrNil(userIDStr)
	if userID == uuid.Nil {
		return false, nil, nil
	}
	return true, &userID, nil
}

//EmailID : definition of an email id
type EmailID struct {
	ID    *uuid.UUID `json:"-"`
	IDStr string     `json:"id"`
}

//FormatIDs : format the ids
func (e *EmailID) FormatIDs() {
	e.IDStr = EncodeUUIDBase64(e.ID)
}

//ParseIDs : parse the ids
func (e *EmailID) ParseIDs() {
	e.ID = DecodeUUIDBase64(e.IDStr)
}

//EmailClaims : claims to include in the email JWT
type EmailClaims struct {
	jwt.StandardClaims
	EmailID `json:"email_id"`
}

//GenerateEmailToken : generate an email JWT using the given claims
func GenerateEmailToken(id *uuid.UUID) (string, error) {
	//create the claims
	claims := &EmailClaims{
		EmailID: EmailID{
			ID: id,
		},
	}
	claims.FormatIDs()

	//create the token
	algorithm := jwt.GetSigningMethod(JWTSigningAlgorithm)
	token := jwt.NewWithClaims(algorithm, claims)

	//create the signed string
	tokenStr, err := token.SignedString([]byte(GetJWTKey()))
	if err != nil {
		return "", errors.Wrap(err, "sign email token")
	}
	return tokenStr, nil
}

//ValidateEmailToken : validate an email JWT
func ValidateEmailToken(tokenStr string) (bool, *EmailID, error) {
	//initialize the claims
	claims := &EmailClaims{}

	//parse the JWT and load the claims
	token, err := jwt.ParseWithClaims(tokenStr, claims, getTokenKey)
	if err != nil {
		if err == jwt.ErrSignatureInvalid {
			return false, nil, nil
		}
		return false, nil, err
	}

	//verify the signing algorithm
	if token.Method.Alg() != JWTSigningAlgorithm {
		return false, nil, fmt.Errorf("invalid signing algorthm: %s", token.Method.Alg())
	}

	//check if the token is valid
	if !token.Valid {
		return false, nil, nil
	}

	//extract the ids
	claims.ParseIDs()
	return true, &claims.EmailID, nil
}

//get the key used to sign the JWT
func getTokenKey(token *jwt.Token) (interface{}, error) {
	return []byte(GetJWTKey()), nil
}

//CreateRandomBytes : create a byte array filled with random data
func CreateRandomBytes(length int) ([]byte, error) {
	if length <= 0 {
		return nil, fmt.Errorf("invalid length: %d", length)
	}

	//create a byte array and populate with random data
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		return nil, errors.Wrap(err, "random bytes")
	}
	return bytes, nil
}

//SHA256HashBytes : use SHA256 to hash the byte array
func SHA256HashBytes(bytes []byte) ([]byte, error) {
	hash := sha256.New()
	_, err := hash.Write(bytes)
	if err != nil {
		return nil, errors.Wrap(err, "hash bytes")
	}
	hashedBytes := hash.Sum(nil)
	return hashedBytes, nil
}

//CreatePwdResetToken : create a token used for password reset
func CreatePwdResetToken() (string, error) {
	//create a random array of bytes for the token
	bytes, err := CreateRandomBytes(resetPwdTokenByteLength)
	if err != nil {
		return "", errors.Wrap(err, "create random bytes")
	}

	//hash the token
	hashedBytes, err := SHA256HashBytes(bytes)
	if err != nil {
		return "", errors.Wrap(err, "hash random bytes")
	}

	//create a base64 url-safe string
	hashedStr := base64.URLEncoding.EncodeToString(hashedBytes)
	return hashedStr, nil
}

//CreateEmailVerifyToken : create a token used for email verification
func CreateEmailVerifyToken() (string, error) {
	//create a random array of bytes for the token
	bytes, err := CreateRandomBytes(verifyEmailTokenByteLength)
	if err != nil {
		return "", errors.Wrap(err, "create random bytes")
	}

	//hash the token
	hashedBytes, err := SHA256HashBytes(bytes)
	if err != nil {
		return "", errors.Wrap(err, "hash random bytes")
	}

	//create a base64 url-safe string
	hashedStr := base64.URLEncoding.EncodeToString(hashedBytes)
	return hashedStr, nil
}

//OAuthClaims : claims to include in the OAuth JWT
type OAuthClaims struct {
	jwt.StandardClaims
	IsSignUp bool   `json:"is_signup"`
	TimeZone string `json:"timezone"`
	Type     string `json:"type"`
	Host     string `json:"host"`
}

//GenerateOAuthToken : generate an OAuth authentication JWT
func GenerateOAuthToken(isSignUp bool, timeZone string, typeVal string, host string) (string, error) {
	//compute the expiration
	expiration := time.Now().Unix() + JWTOAuthExpirationSec

	//create the claims
	claims := &OAuthClaims{
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expiration,
		},
		IsSignUp: isSignUp,
		TimeZone: timeZone,
		Type:     typeVal,
		Host:     host,
	}

	//create the token
	algorithm := jwt.GetSigningMethod(JWTSigningAlgorithm)
	token := jwt.NewWithClaims(algorithm, claims)

	//create the signed string
	tokenStr, err := token.SignedString([]byte(GetJWTKey()))
	if err != nil {
		return "", errors.Wrap(err, "sign oauth token")
	}
	return tokenStr, nil
}

//ValidateOAuthToken : validate an OAuth authentication JWT
func ValidateOAuthToken(tokenStr string) (bool, string, string, string, bool, error) {
	//initialize the claims
	claims := &OAuthClaims{}

	//parse the JWT and load the claims
	token, err := jwt.ParseWithClaims(tokenStr, claims, getTokenKey)
	if err != nil {
		if err == jwt.ErrSignatureInvalid {
			return false, "", "", "", false, nil
		}
		return false, "", "", "", false, err
	}

	//verify the signing algorithm
	if token.Method.Alg() != JWTSigningAlgorithm {
		return false, "", "", "", false, fmt.Errorf("invalid signing algorthm: %s", token.Method.Alg())
	}

	//check if the token is valid
	if !token.Valid {
		return false, "", "", "", false, nil
	}
	return claims.IsSignUp, claims.TimeZone, claims.Type, claims.Host, true, nil
}
