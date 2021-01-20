package userService

import (
	"carHiringWebsite/data"
	"carHiringWebsite/db"
	"carHiringWebsite/hash"
	"carHiringWebsite/session"
	"database/sql"
	"errors"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func Logout(token string) error {
	err := session.ValidateToken(token)
	if err != nil {
		return err
	}

	bag, err := session.GetByToken(token)
	if err != nil {
		return err
	}

	if !session.Delete(bag) {
		return errors.New("failed to delete session")
	}

	return nil
}

func Get(token string) (*data.OutputUser, error) {
	err := session.ValidateToken(token)
	if err != nil {
		return nil, err
	}

	bag, err := session.GetByToken(token)
	if err != nil {
		return nil, err
	}

	user := bag.GetUser()
	newUser, err := db.SelectUserByID(user.ID)
	if err != nil {
		newUser = user
	} else {
		newUser.SessionToken = user.SessionToken
	}

	bag.UpdateUser(newUser)

	outputUser := data.NewOutputUser(newUser)

	return outputUser, nil
}

func ValidateSession(token string) (*data.OutputUser, error) {
	err := session.ValidateToken(token)
	if err != nil {
		return nil, err
	}

	bag, err := session.GetByToken(token)
	if err != nil {
		return nil, err
	}

	user := bag.GetUser()
	newUser, err := db.SelectUserByID(user.ID)
	if err != nil {
		newUser = user
	} else {
		newUser.SessionToken = user.SessionToken
	}

	bag.UpdateUser(newUser)

	outputUser := data.NewOutputUser(newUser)

	return outputUser, nil
}

func Authenticate(email, password string) (*data.OutputUser, bool, error) {
	var authUser *data.User
	var err error
	newSession := false

	email = strings.TrimSpace(email)

	if !ValidateCredentials(email, password) {
		return &data.OutputUser{}, false, nil
	}

	bag, err := session.GetByEmail(email)

	if err == nil {
		authUser = bag.GetUser()
	} else if err == session.InactiveSession {
		authUser, err = db.SelectUserByEmail(email)
		if err != nil {
			return &data.OutputUser{}, false, err
		}
		newSession = true
	}

	if authUser.Disabled {
		return &data.OutputUser{}, false, nil
	}

	hash, err := hash.Get(authUser.AuthSalt, password)

	if strings.Compare(hash, authUser.AuthHash) != 0 {
		return &data.OutputUser{}, false, nil
	}

	outputUser := data.NewOutputUser(authUser)

	if newSession {
		outputUser.SessionToken = session.New(authUser)
	}

	return outputUser, true, nil
}

var emailRegex = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+\\/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")

func CreateUser(email, password, firstname, names, dobString string) (bool, *data.OutputUser, error) {

	dobUnix, err := strconv.ParseInt(dobString, 10, 64)
	if err != nil {
		return false, nil, err
	}

	dob := time.Unix(dobUnix, 0)

	if CalculateAge(dobUnix) < 18 {
		return false, nil, errors.New("age validation error")
	}

	if !ValidateCredentials(email, password) {
		return false, nil, errors.New("userService failed validation")
	}

	_, err = db.SelectUserByEmail(email)
	if err != nil {
		if err != sql.ErrNoRows {
			return false, &data.OutputUser{}, err
		}
	} else {
		return false, &data.OutputUser{}, nil
	}

	salt, hash, err := hash.New(password)
	if err != nil {
		return false, &data.OutputUser{}, err
	}

	email = strings.TrimSpace(email)

	userID, err := db.CreateUser(email, firstname, names, dob, salt, hash)
	if err != nil {
		return false, &data.OutputUser{}, err
	}

	newUser, err := db.SelectUserByID(userID)
	if err != nil {
		return false, &data.OutputUser{}, err
	}

	return true, data.NewOutputUser(newUser), nil
}

func CalculateAge(dobUnix int64) int {
	ageDifMs := time.Now().Unix() - dobUnix
	ageDate := time.Unix(ageDifMs, 0)
	age := math.Abs(float64(ageDate.Year() - 1970))

	return int(age)
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

var lowerCaseLetters = regexp.MustCompile("[a-z]")
var upperCaseLetters = regexp.MustCompile("[A-Z]")
var numbers = regexp.MustCompile("[0-9]")

// password validation rules
func isPasswordValid(password string) bool {
	if len(password) < 8 {
		return false
	}
	if !lowerCaseLetters.MatchString(password) {
		return false
	}
	if !upperCaseLetters.MatchString(password) {
		return false
	}
	if !numbers.MatchString(password) {
		return false
	}
	return true
}

func GetUserFromSession(token string) (*data.User, error) {
	err := session.ValidateToken(token)
	if err != nil {
		return nil, err
	}

	bag, err := session.GetByToken(token)
	if err != nil {
		return nil, err
	}

	return bag.GetUser(), nil
}
