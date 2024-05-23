package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

type PasswordMeta struct {
	time    uint32
	memory  uint32
	threads uint8
	keyLen  uint32
}

func CreatePassword(password string) (string, error) {

	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	meta := &PasswordMeta{
		time:    1,
		memory:  64 * 1024,
		threads: 4,
		keyLen:  32,
	}

	hash := argon2.IDKey(
		[]byte(password),
		salt,
		meta.time,
		meta.memory,
		meta.threads,
		meta.keyLen,
	)

	return fmt.Sprintf(
			"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
			argon2.Version,
			meta.memory,
			meta.time,
			meta.threads,
			base64.RawStdEncoding.EncodeToString(salt),
			base64.RawStdEncoding.EncodeToString(hash),
		),
		nil
}

func Validate(password, hash string) (bool, error) {

	parts := strings.Split(hash, "$")

	c := &PasswordMeta{}
	_, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &c.memory, &c.time, &c.threads)
	if err != nil {
		return false, err
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, err
	}

	decodedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, err
	}
	c.keyLen = uint32(len(decodedHash))

	comparisonHash := argon2.IDKey([]byte(password), salt, c.time, c.memory, c.threads, c.keyLen)

	return (subtle.ConstantTimeCompare(decodedHash, comparisonHash) == 1), nil
}
