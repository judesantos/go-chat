package auth

import (
	"yt/chat/lib/utils/log"

	"golang.org/x/crypto/bcrypt"
)

func HashString(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	return string(bytes), err
}

func Validate(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		log.GetLogger().Error("Validate failed: " + err.Error())
		return false
	}
	return true
}
