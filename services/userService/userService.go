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

var (
	InvalidPassword       = errors.New("invalid password")
	UsernameAlreadyExists = errors.New("username already exists")
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
	if err == session.InactiveSession {
		newSession = true
	} else if err != nil {
		return &data.OutputUser{}, false, err
	}

	authUser, err = db.SelectUserByEmail(email)
	if err != nil {
		return &data.OutputUser{}, false, err
	}

	if bag != nil && !newSession {
		authUser.SessionToken = bag.GetToken()
		bag.UpdateUser(authUser)
	}

	if authUser.Disabled {
		return &data.OutputUser{}, false, nil
	}

	hash, err := hash.Get(authUser.AuthSalt, password)
	if err != nil {
		return &data.OutputUser{}, false, err
	}

	if strings.Compare(hash, authUser.AuthHash) != 0 {
		return &data.OutputUser{}, false, nil
	}

	outputUser := data.NewOutputUser(authUser)

	if newSession {
		outputUser.SessionToken = session.New(authUser)
	}

	return outputUser, true, nil
}

func EditUser(token, userID, email, oldPassword, password, firstname, names, dobString string) (*data.OutputUser, error) {

	id, err := strconv.Atoi(userID)
	if err != nil {
		return nil, err
	}

	user, err := GetUserFromSession(token)
	if err != nil {
		return nil, err
	}

	if oldPassword == "" && !user.Admin {
		return nil, errors.New("old password not provided")
	}

	if id != user.ID && !user.Admin {
		return nil, errors.New("user must be admin to do this")
	}

	authUser, err := db.SelectUserByID(id)
	if err != nil {
		return nil, err
	}

	if len(firstname) >= 50 || firstname == "" {
		firstname = authUser.FirstName
	}

	if len(names) >= 100 || names == "" {
		names = authUser.Names
	}

	if authUser.Disabled {
		return nil, nil
	}

	if !user.Admin {
		if oldPassword == "" {
			return nil, errors.New("old password not provided")
		}

		hash, err := hash.Get(authUser.AuthSalt, oldPassword)
		if err != nil {
			return nil, err
		}

		if strings.Compare(hash, authUser.AuthHash) != 0 {
			return nil, InvalidPassword
		}
	}

	var dob time.Time
	if dobString != "" {
		dobUnix, err := strconv.ParseInt(dobString, 10, 64)
		if err != nil {
			return nil, err
		}

		dob = time.Unix(dobUnix, 0)

		if !dob.Equal(authUser.DOB) && CalculateAge(dobUnix) < 18 {
			return nil, errors.New("age validation error")
		}
	} else {
		dob = authUser.DOB
	}

	if strings.Compare(email, authUser.Email) != 0 && email != "" {
		if !isEmailValid(email) {
			return nil, errors.New("email validation error")
		}

		_, err = db.SelectUserByEmail(email)
		if err != nil {
			if err != sql.ErrNoRows {
				return &data.OutputUser{}, err
			}
		} else {
			return &data.OutputUser{}, UsernameAlreadyExists
		}
	} else {
		email = authUser.Email
	}
	email = strings.TrimSpace(email)

	var salt string
	var hashstring string
	if password != "" {
		if !isPasswordValid(password) {
			return nil, errors.New("password validation error")
		}

		salt, hashstring, err = hash.New(password)
		if err != nil {
			return &data.OutputUser{}, err
		}
	} else {
		salt = authUser.AuthSalt
		hashstring = authUser.AuthHash
	}

	err = db.UpdateUser(id, email, firstname, names, dob, salt, hashstring)
	if err != nil {
		return nil, err
	}

	newUser, err := db.SelectUserByID(id)
	if err != nil {
		return &data.OutputUser{}, err
	}

	bag, err := session.GetByEmail(newUser.Email)
	if err != nil {
		if err != session.InactiveSession {
			return nil, err
		}
	}
	if bag != nil {
		newUser.SessionToken = bag.GetToken()
		bag.UpdateUser(newUser)
	}

	return data.NewOutputUser(newUser), nil
}

var emailRegex = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+\\/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")

func CreateUser(email, password, firstname, names, dobString string) (bool, *data.OutputUser, error) {

	if len(firstname) >= 50 || len(names) >= 100 {
		return false, &data.OutputUser{}, errors.New("validation error")
	}

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
