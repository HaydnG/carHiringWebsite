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
	var err error

	conn, err = sql.Open("mysql", "interaction:pass@tcp(localhost:3306)/carrental?parseTime=true&timeout=3s")
	if err != nil {
		return err
	}
	conn.SetMaxOpenConns(8)
	conn.SetMaxIdleConns(8)
	conn.SetConnMaxLifetime(5 * time.Minute)

	return nil
}

func CloseDB() error {
	return conn.Close()
}

// Database User Logic
//
//

func CreateUser(email, firstname, names string, dob time.Time, salt, hash []byte) (int, error) {

	//Prepared statements
	createUser, err := conn.Prepare(`INSERT INTO USERS
								(firstname, names,email,createdAt,authHash,authSalt,DOB)
								VALUES(?,?,?,?,?,?,?)`)
	defer createUser.Close()

	res, err := createUser.Exec(firstname, names, email, time.Now(), hash, salt, dob)
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

	err := row.Scan(&newUser.ID, &newUser.FirstName, &newUser.Names, &newUser.Email, &newUser.CreatedAt, &newUser.AuthHash, &newUser.AuthSalt, &newUser.Blacklisted, &newUser.DOB, &newUser.Verified, &newUser.Repeat)
	if err != nil {
		return &newUser, err
	}

	return &newUser, nil
}

// Database User Logic
//
//

func GetAllCars() ([]*data.Car, error) {
	rows, err := conn.Query(`SELECT cars.*, fuelType.description, gearType.description, carType.description, size.description, colour.description
									FROM carrental.cars
									INNER JOIN fueltype ON cars.fuelType = fuelType.id
									INNER JOIN gearType ON cars.gearType = gearType.id
									INNER JOIN carType ON cars.carType = carType.id
									INNER JOIN size ON cars.size = size.id
									INNER JOIN colour ON cars.colour = colour.id
									LIMIT 48`)
	if err != nil {
		return nil, err
	}

	cars := make([]*data.Car, 48)
	count := 0
	for rows.Next() {

		car := data.NewCar()
		cars[count] = car

		err := rows.Scan(&car.ID, &car.FuelType.ID, &car.GearType.ID, &car.CarType.ID, &car.Size.ID, &car.Colour.ID, &car.Cost, &car.Description, &car.Image, &car.Seats,
			&car.FuelType.Description, &car.GearType.Description, &car.CarType.Description, &car.Size.Description, &car.Colour.Description)
		if err != nil {
			return nil, err
		}

		count++
	}

	cars = cars[:count]

	return cars, nil
}

func GetCar(id string) (*data.Car, error) {
	row := conn.QueryRow(`SELECT cars.*, fuelType.description, gearType.description, carType.description, size.description, colour.description
									FROM carrental.cars
									INNER JOIN fueltype ON cars.fuelType = fuelType.id
									INNER JOIN gearType ON cars.gearType = gearType.id
									INNER JOIN carType ON cars.carType = carType.id
									INNER JOIN size ON cars.size = size.id
									INNER JOIN colour ON cars.colour = colour.id
									WHERE cars.id = ?`, id)

	car := data.NewCar()

	err := row.Scan(&car.ID, &car.FuelType.ID, &car.GearType.ID, &car.CarType.ID, &car.Size.ID, &car.Colour.ID, &car.Cost, &car.Description, &car.Image, &car.Seats,
		&car.FuelType.Description, &car.GearType.Description, &car.CarType.Description, &car.Size.Description, &car.Colour.Description)
	if err != nil {
		return car, err
	}
	return car, nil
}
