package db

import (
	"carHiringWebsite/data"
	"database/sql"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

var (
	conn *sql.DB
)

func InitDB() error {

	db, err := sql.Open("mysql", "interaction:pass@tcp(localhost:3306)/carrental?parseTime=true")
	if err != nil {
		return err
	}
	conn = db
	return nil
}

func CloseDB() error {
	return conn.Close()
}

// Database User Logic
//
//

func CreateUser(email, name string, dob time.Time, salt, hash []byte) (int, error) {

	stmt, err := conn.Prepare(`INSERT INTO USERS
								(fullName,email,createdAt,authHash,authSalt,DOB)
								VALUES(?,?,?,?,?,?)`)
	if err != nil {
		return 0, err
	}

	res, err := stmt.Exec(name, email, time.Now(), hash, salt, dob)
	if err != nil {
		return 0, err
	}

	userID, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	return int(userID), nil
}

func SelectUserByEmail(email string) (*data.User, error) {
	row := conn.QueryRow("SELECT * FROM USERS WHERE email = ?", email)

	return readUserRow(row)
}

func SelectUserByID(id int) (*data.User, error) {
	row := conn.QueryRow("SELECT * FROM USERS WHERE id = ?", id)

	return readUserRow(row)
}

func readUserRow(row *sql.Row) (*data.User, error) {
	newUser := data.User{}

	err := row.Scan(&newUser.ID, &newUser.FullName, &newUser.Email, &newUser.CreatedAt, &newUser.AuthHash, &newUser.AuthSalt, &newUser.Blacklisted, &newUser.DOB, &newUser.Verified, &newUser.Repeat)
	if err != nil {
		return &data.User{}, err
	}

	return &newUser, nil
}
