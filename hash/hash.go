package hash

import (
	"crypto/sha256"
	"math/rand"
	"time"
)

const saltSize = 16

func Init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

// Get generates a hash with the salt and password
func Get(salt []byte, password string) ([]byte, error) {

	hash := sha256.New()

	salted := saltPassword(salt, password)

	_, err := hash.Write(salted)
	if err != nil {
		return nil, err
	}

	return hash.Sum(nil), nil
}

// New generates a hash and salt for the given password
func New(password string) ([]byte, []byte, error) {

	salt, err := generateSalt()
	if err != nil {
		return nil, nil, err
	}
	authHash, err := Get(salt, password)

	return salt, authHash, err
}

func generateSalt() ([]byte, error) {
	salt := make([]byte, saltSize)

	_, err := rand.Read(salt[:])
	if err != nil {
		panic(err)
	}
	return salt, nil
}

func saltPassword(salt []byte, password string) []byte {
	return append(salt, []byte(password)...)
}
