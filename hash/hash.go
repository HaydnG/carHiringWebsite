package hash

import (
	"crypto/sha256"
	"encoding/base64"
	"math/rand"
	"time"
)

const saltSize = 16

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

// Get generates a hash with the salt and password
func Get(salt string, password string) (string, error) {

	hash := sha256.New()

	salted := saltPassword(salt, password)

	_, err := hash.Write(salted)
	if err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(hash.Sum(nil)), nil
}

// New generates a hash and salt for the given password
func New(password string) (string, string, error) {

	salt, err := generateSalt()
	if err != nil {
		return "", "", err
	}
	authHash, err := Get(salt, password)

	return salt, authHash, err
}

func generateSalt() (string, error) {
	salt := make([]byte, saltSize)

	_, err := rand.Read(salt[:])
	if err != nil {
		panic(err)
	}
	return base64.URLEncoding.EncodeToString(salt), nil
}

func saltPassword(salt string, password string) []byte {
	return append([]byte(salt), []byte(password)...)
}
