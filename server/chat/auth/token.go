package auth

import (
	"fmt"
	"time"
	"yt/chat/server/chat/datasource"

	"github.com/dgrijalva/jwt-go"
)

const HMAC_SECRET = ")sd*fIske2^se(f_@E&qw=_-"
const EXPIRE_TIME_SECS = 3600 // seconds. 1 hour

type TokenMeta struct {
	AccessToken string // The signed access-token
	TTL         int    // Time in seconds from creation
	ExpiresAt   int64  // Timestamp in seconds of expiration
}

type TokenClaim struct {
	jwt.StandardClaims
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (m *TokenClaim) GetId() string {
	return m.ID
}

func (m *TokenClaim) GetName() string {
	return m.Name
}

// Create fresh token for a specified subscriber
func NewToken(user *datasource.Subscriber) (*TokenMeta, error) {

	expiresAt := time.Now().Unix() + EXPIRE_TIME_SECS
	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		jwt.MapClaims{
			"Id":        user.GetId(),
			"Name":      user.GetName(),
			"ExpiresAt": expiresAt,
		},
	)

	signed, err := token.SignedString([]byte(HMAC_SECRET))
	if err != nil {
		return nil, err
	}

	tokenMeta := &TokenMeta{
		AccessToken: signed,
		TTL:         EXPIRE_TIME_SECS,
		ExpiresAt:   expiresAt,
	}

	return tokenMeta, nil

}

func validateKey(token *jwt.Token) (interface{}, error) {

	_, ok := token.Method.(*jwt.SigningMethodHMAC)
	if !ok {
		return nil, fmt.Errorf("error signing: %v", token.Header["alg"])
	}

	return []byte(HMAC_SECRET), nil
}

func ValidateToken(signed string) (*TokenClaim, error) {

	parsed, err := jwt.ParseWithClaims(signed, &TokenClaim{}, validateKey)
	subs, ok := parsed.Claims.(*TokenClaim)
	if ok && parsed.Valid {
		return subs, nil
	}

	return nil, err
}
