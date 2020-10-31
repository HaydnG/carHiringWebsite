package session

import (
	"carHiringWebsite/db"
	"carHiringWebsite/hash"
)

func AuthenticateUser(email, password string) (string, error) {

	user, err := db.SelectUser(email)
	if err != nil {
		return "", err
	}

	hash, err := hash.Generate(user.AuthSalt, password)

	//sessionToken := uuid.New().String()

}
