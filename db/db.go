package db

import (
	"carHiringWebsite/data"
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
)

var(
	conn *sql.DB

)

func InitDB() error{

	db, err := sql.Open("mysql", "interaction:pass@tcp(localhost:3306)/carrental")
	if err != nil{
		return err
	}
	conn = db

	GetUser("test@gmail.com")

	return nil
}

func CloseDB() error{
	return conn.Close()
}


func GetUser(email string) (data.User, error){
	row := conn.QueryRow("SELECT * FROM USERS WHERE email = ?", email)

	user := data.User{}

	err := row.Scan(&user.ID,&user.FullName, &user.Email, &user.CreatedAt, &user.AuthHash, &user.AuthSalt, &user.Blacklisted, &user.DOB, &user.Verified, &user.Repeat)
	if err != nil{
		return data.User{}, err

	}

	return user, nil
}

