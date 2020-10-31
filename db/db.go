package db

import (
	"database/sql"
	"fmt"
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

	rows, err := conn.Query("SELECT * FROM USERS WHERE email = ?", "test@gmail.com")
	if err != nil{
		return err
	}

	fmt.Println(rows)

	return nil
}

func CloseDB() error{
	return conn.Close()
}
