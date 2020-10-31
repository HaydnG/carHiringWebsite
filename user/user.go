package user

import (
	"os/user"
	"regexp"
	"time"
)

type User struct {
	ID 			int
	FullName 	string
	Email 		string
	CreatedAt 	time.Time
	Password 	string
	AuthHash 	[]byte
	AuthSalt 	[]byte
	Blacklisted bool
	DOB 		time.Time
	Verified 	bool
	Repeat 		bool
}

var emailRegex = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+\\/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")




func CreateUser(email, password, name string, dob time.Time) user.User{


}

func ValidateCredentials(email, password string) bool{

	if !isEmailValid(email){
		return false
	}
	if !isPasswordValid(password){
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
	if len(password) < 8{
		return false
	}
	return true
}