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

	db, err := sql.Open("mysql", "interaction:root@tcp(localhost:3306)/test")
	if err != nil{
		return err
	}
	conn = db

	rows, err := conn.Query("SELECT * FROM USERS WHERE email = '$1'", "test@gmail.com")
	if err != nil{
		return err
	}

	fmt.Println(rows)

	return nil
}

func CloseDB() error{
	return conn.Close()
}
