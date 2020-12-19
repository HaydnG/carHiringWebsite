package db

import (
	"carHiringWebsite/data"
	"database/sql"
	"errors"
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
	if err != nil {
		return 0, err
	}
	defer createUser.Close()

	res, err := createUser.Exec(firstname, names, email, time.Now(), hash, salt, dob)
	if err != nil {
		return 0, err
	}

	userID, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	if userID == 0 {
		return 0, errors.New("no user inserted")
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

func GetCarAccessories(start, end string) ([]*data.Accessory, error) {
	rows, err := conn.Query(`select a1.id, a1.description
							from equipment as a1
							Where (a1.stock - 
							(select COUNT(*) from equipmentbooking 
							inner join bookings on bookings.id = equipmentbooking.bookingID
							inner join equipment on equipmentbooking.equipmentID = equipment.id
							where 
							((? <= bookings.end ) and (? >= bookings.start))
							And equipment.id = a1.id)) > 0
							LIMIT 16`, start, end)
	if err != nil {
		return nil, err
	}
	accessories := make([]*data.Accessory, 16)

	count := 0
	for rows.Next() {

		accessory := &data.Accessory{}
		accessories[count] = accessory

		err := rows.Scan(&accessory.ID, &accessory.Description)
		if err != nil {
			return nil, err
		}

		count++
	}

	accessories = accessories[:count]

	return accessories, nil
}

func GetCarBookings(start, end, carID string) ([]*data.TimeRange, error) {
	rows, err := conn.Query(`SELECT bookings.start, bookings.end FROM bookings
						WHERE ((? <= bookings.end ) and (? >= bookings.start))
						AND bookings.carID = ?
						LIMIT 30`, start, end, carID)
	if err != nil {
		return nil, err
	}
	timeRanges := make([]*data.TimeRange, 30)

	count := 0
	for rows.Next() {

		timeRange := &data.TimeRange{}
		timeRanges[count] = timeRange

		err := rows.Scan(&timeRange.Start, &timeRange.End)
		if err != nil {
			return nil, err
		}

		count++
	}

	timeRanges = timeRanges[:count]

	return timeRanges, nil
}

func BookingHasOverlap(start, end, carID string) (bool, error) {
	row := conn.QueryRow(`SELECT COUNT(*) AS overlaps FROM bookings
								WHERE ((? <= bookings.end ) and (? >= bookings.start))
								AND bookings.carID = ?`, start, end, carID)
	overlaps := 0

	err := row.Scan(&overlaps)
	if err != nil {
		return false, err
	}

	return overlaps > 0, nil
}

func CreateBooking(carID, userID int, start, end string, cost float64, lateReturn, extension int) (int, error) {

	//Prepared statements
	createBooking, err := conn.Prepare(`INSERT INTO bookings(carID, userID, start, end, totalCost, amountPaid, lateReturn, extension, created)
												VALUES(?, ?, ?, ?, ?, '0', ?, ?, ?)`)
	if err != nil {
		return 0, err
	}
	defer createBooking.Close()

	res, err := createBooking.Exec(carID, userID, start, end, cost, lateReturn, extension, time.Now())
	if err != nil {
		return 0, err
	}

	bookingID, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	if bookingID == 0 {
		return 0, errors.New("no booking inserted")
	}

	return int(bookingID), nil
}

func InsertBookingStatus(bookingID, processID, adminID int, description string) (int, error) {

	//Prepared statements
	insertBookingStatus, err := conn.Prepare(`INSERT INTO bookingstatus(bookingID, processID, completed, adminID, description)
												VALUES(?, ?, ?, ?,?)`)
	if err != nil {
		return 0, err
	}
	defer insertBookingStatus.Close()

	res, err := insertBookingStatus.Exec(bookingID, processID, time.Now(), adminID, description)
	if err != nil {
		return 0, err
	}

	statusID, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	if statusID == 0 {
		return 0, errors.New("no status inserted")
	}

	return int(statusID), nil
}

func AddBookingEquipment(bookingID int, equipment []string) error {

	//Prepared statements
	insertBookingStatus, err := conn.Prepare(`INSERT INTO equipmentbooking(bookingID, equipmentID)
												VALUES(?, ?)`)
	if err != nil {
		return err
	}
	defer insertBookingStatus.Close()

	for _, v := range equipment {
		if v == "" {
			return errors.New("invalid equipment param")
		}

		_, err := insertBookingStatus.Exec(bookingID, v)
		if err != nil {
			return err
		}
	}

	return nil
}

func GetBookingAccessories(bookingID int) ([]*data.Accessory, error) {

	rows, err := conn.Query(`SELECT equipment.id, equipment.description FROM equipment
inner JOIN equipmentbooking ON equipmentbooking.equipmentID = equipment.id 
WHERE  equipmentbooking.bookingID = ? LIMIT 10`, bookingID)
	if err != nil {
		return nil, err
	}
	accessories := make([]*data.Accessory, 10)

	count := 0
	for rows.Next() {

		accessory := &data.Accessory{}
		accessories[count] = accessory

		err := rows.Scan(&accessory.ID, &accessory.Description)
		if err != nil {
			return nil, err
		}

		count++
	}

	accessories = accessories[:count]

	return accessories, nil
}

func GetSingleBooking(bookingID int) (*data.Booking, error) {
	var (
		start   time.Time
		end     time.Time
		created time.Time
	)

	row := conn.QueryRow(`SELECT bookings.*, bookingstatus.processID FROM carrental.bookings, carrental.bookingstatus 
WHERE bookings.id = ? 
AND bookingstatus.bookingID = bookings.id
ORDER BY bookingstatus.completed DESC LIMIT 1`, bookingID)

	booking := &data.Booking{}

	err := row.Scan(&booking.ID, &booking.CarID, &booking.UserID, &start, &end, &booking.TotalCost,
		&booking.AmountPaid, &booking.LateReturn, &booking.Extension, &created, &booking.ProcessID)
	if err != nil {
		return nil, err
	}
	booking.Start = *data.ConvertDate(start)
	booking.End = *data.ConvertDate(end)
	booking.Created = *data.ConvertDate(created)

	return booking, nil
}
