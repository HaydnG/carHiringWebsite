package userService

import (
	"bytes"
	"carHiringWebsite/data"
	"carHiringWebsite/db"
	"carHiringWebsite/hash"
	"carHiringWebsite/session"
	"database/sql"
	"regexp"
	"time"
)

func Authenticate(email, password string) (*data.OutputUser, error) {

	authUser, err := db.SelectUserByEmail(email)
	if err != nil {
		return &data.OutputUser{}, err
	}

	hash, err := hash.Get(authUser.AuthSalt, password)

	if bytes.Compare(hash, authUser.AuthHash) != 0 {
		return &data.OutputUser{}, nil
	}

	outputUser := data.NewOutputUser(authUser)
	outputUser.SessionToken = session.New(authUser)

	return outputUser, nil
}

var emailRegex = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+\\/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")

func CreateUser(email, password, name string, dob time.Time) (bool, *data.OutputUser, error) {

	salt, hash, err := hash.New(password)
	if err != nil {
		return false, &data.OutputUser{}, err
	}

	_, err = db.SelectUserByEmail(email)
	if err != nil {
		if err != sql.ErrNoRows {
			return false, &data.OutputUser{}, err
		}
	} else {
		return false, &data.OutputUser{}, nil
	}

	userID, err := db.CreateUser(email, name, dob, salt, hash)
	if err != nil {
		return false, &data.OutputUser{}, err
	}

	newUser, err := db.SelectUserByID(userID)
	if err != nil {
		return false, &data.OutputUser{}, err
	}

	return true, data.NewOutputUser(newUser), nil
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
