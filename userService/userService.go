package userService

import (
	"carHiringWebsite/data"
	"carHiringWebsite/db"
	"carHiringWebsite/hash"
	"regexp"
	"time"
)

var emailRegex = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+\\/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")

func CreateUser(email, password, name string, dob time.Time) (*data.User, error) {

	salt, hash, err := hash.New(password)
	if err != nil {
		return &data.User{}, err
	}

	userID, err := db.CreateUser(email, name, dob, salt, hash)
	if err != nil {
		return &data.User{}, err
	}

	newUser, err := db.SelectUserByID(userID)
	if err != nil {
		return &data.User{}, err
	}

	return newUser, nil
}

func ValidateCredentials(email, password string) bool {

	if !isEmailValid(email) {
		return false
	}
	if !isPasswordValid(password) {
		return false
	}

	return true
}

// email validation rules
func isEmailValid(e string) bool {
	if len(e) < 3 && len(e) > 254 {
		return false
	}
	return emailRegex.MatchString(e)
}

// password validation rules
func isPasswordValid(password string) bool {
	if len(password) < 8 {
		return false
	}
	return true
}
