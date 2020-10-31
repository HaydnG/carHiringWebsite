package session

import(

	"carHiringWebsite/db"
	"carHiringWebsite/hash"
	"carHiringWebsite/user"
)

func AuthenticateUser(email, password string) (string, error){


	user, err := db.GetUser(email)
	if err != nil{
		return "",err
	}

	hash, err := hash.Generate(user.AuthSalt, password)





	//sessionToken := uuid.New().String()


}




