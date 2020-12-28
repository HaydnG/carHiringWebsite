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

//SELECT cars.*, fuelType.description, gearType.description, carType.description, size.description, colour.description
//FROM carrental.cars
//INNER JOIN fueltype ON cars.fuelType = fuelType.id
//INNER JOIN gearType ON cars.gearType = gearType.id
//INNER JOIN carType ON cars.carType = carType.id
//INNER JOIN size ON cars.size = size.id
//INNER JOIN colour ON cars.colour = colour.id
//WHERE cars.fuelType = coalesce(1, cars.fuelType) AND
//cars.gearType = coalesce(NULL, cars.gearType) AND
//cars.carType = coalesce(NULL, cars.carType) AND
//cars.size = coalesce(NULL, cars.size) AND
//cars.colour = coalesce(NULL, cars.colour)

func GetCarAccessories(start, end string) ([]*data.Accessory, error) {
	rows, err := conn.Query(`select a1.id, a1.description
							from equipment as a1
							Where (a1.stock - 
							(select COUNT(*) from equipmentbooking 
							inner join bookings on bookings.id = equipmentbooking.bookingID
							inner join equipment on equipmentbooking.equipmentID = equipment.id
							where 
							((? <= bookings.end ) and (? >= bookings.finish))
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

func GetBookingHistory(bookingID int) ([]*data.BookingStatus, error) {
	var (
		completed time.Time
		size      = 32
	)

	rows, err := conn.Query(`SELECT bookingstatus.id, bookingstatus.bookingID, bookingstatus.completed, bookingstatus.active, bookingstatus.adminID, bookingstatus.description,
bookingstatus.processID, processtype.description, processtype.adminRequired, processtype.order, processtype.bookingPage
FROM bookingstatus
inner join processtype on processtype.id = bookingstatus.processID
where bookingstatus.bookingID = ?
order by bookingstatus.completed asc;`, bookingID)
	if err != nil {
		return nil, err
	}
	statuses := make([]*data.BookingStatus, size)

	count := 0
	for rows.Next() {

		status := &data.BookingStatus{}
		if count > size {
			statuses = append(statuses, status)
		} else {
			statuses[count] = status
		}

		err := rows.Scan(&status.ID, &status.BookingID, &completed, &status.Active, &status.AdminID, &status.Description,
			&status.ProcessID, &status.ProcessDescription, &status.AdminRequired, &status.Order, &status.BookingPage)
		if err != nil {
			return nil, err
		}
		status.Completed = *data.ConvertDate(completed)

		count++
	}

	statuses = statuses[:count]

	return statuses, nil
}

func GetCarBookings(start, end, carID string) ([]*data.TimeRange, error) {
	rows, err := conn.Query(`SELECT b.start, b.finish FROM bookings as b
						WHERE ((? <= b.finish ) and (? >= b.start))
						AND b.carID = ? 
						AND (SELECT processID FROM bookingstatus 
								INNER JOIN bookings ON bookingstatus.bookingID = b.id 
								INNER JOIN processtype ON bookingstatus.processID = processtype.id
								WHERE bookingstatus.active = 1
								ORDER BY processtype.order DESC
								LIMIT 1) != 11
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

	row := conn.QueryRow(`SELECT COUNT(*) AS overlaps FROM bookings AS b
								WHERE ((? <= b.end ) AND (? >= b.start))
								AND (SELECT processID FROM bookingstatus 
								INNER JOIN bookings ON bookingstatus.bookingID = b.id 
								INNER JOIN processtype ON bookingstatus.processID = processtype.id
								WHERE bookingstatus.active = 1
								ORDER BY processtype.order DESC
								LIMIT 1) != 11
								AND b.carID = ?`, start, end, carID)
	overlaps := 0

	err := row.Scan(&overlaps)
	if err != nil {
		return false, err
	}

	return overlaps > 0, nil
}

func CreateBooking(carID, userID int, start, end, finish string, cost float64, lateReturn, extension bool, bookingLength float64) (int, error) {

	//Prepared statements
	createBooking, err := conn.Prepare(`INSERT INTO bookings(carID, userID, start, end, finish,totalCost, amountPaid, lateReturn, extension, created, bookingLength)
												VALUES(?, ?, ?, ?, ?, ?, '0', ?, ?, ?, ?)`)
	if err != nil {
		return 0, err
	}
	defer createBooking.Close()

	res, err := createBooking.Exec(carID, userID, start, end, finish, cost, lateReturn, extension, time.Now(), bookingLength)
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

func InsertBookingStatus(bookingID, processID, adminID, active int, description string) (int, error) {

	//Prepared statements
	insertBookingStatus, err := conn.Prepare(`INSERT INTO bookingstatus(bookingID, processID, completed, active, adminID, description)
												VALUES(?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return 0, err
	}
	defer insertBookingStatus.Close()

	res, err := insertBookingStatus.Exec(bookingID, processID, time.Now(), active, adminID, description)
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

func DeactivateBookingStatuses(bookingID int) error {

	conn, err := conn.Exec(`UPDATE bookingstatus SET active = 0 WHERE (bookingID = ?)`, bookingID)
	if err != nil {
		return err
	}

	count, err := conn.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return errors.New("no rows affected")
	}

	return nil
}

func SetBookingStatus(statusID int, active bool) error {

	conn, err := conn.Exec(`UPDATE bookingstatus SET active = ? WHERE (id = ?)`, active, statusID)
	if err != nil {
		return err
	}

	count, err := conn.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return errors.New("no rows affected")
	}

	return nil
}

//GetBookingProcessStatus returns the most recent process with processID specified
func GetBookingProcessStatus(bookingID, processID int) (*data.BookingStatus, error) {
	bookingStatus := &data.BookingStatus{}
	var completed time.Time

	conn := conn.QueryRow(`SELECT * FROM carrental.bookingstatus
								WHERE bookingID = ? AND processID = ?
								ORDER  BY completed DESC LIMIT 1`, bookingID, processID)
	err := conn.Scan(&bookingStatus.ID, &bookingStatus.BookingID, &bookingStatus.ProcessID, &completed, &bookingStatus.Active, &bookingStatus.AdminID, &bookingStatus.Description)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	bookingStatus.Completed = *data.ConvertDate(completed)

	return bookingStatus, nil
}

func AddBookingEquipment(bookingID int, equipment []string) error {

	//Prepared statements
	insertEquipment, err := conn.Prepare(`INSERT INTO equipmentbooking(bookingID, equipmentID)
												VALUES(?, ?)`)
	if err != nil {
		return err
	}
	defer insertEquipment.Close()

	for _, v := range equipment {
		if v == "" {
			return errors.New("invalid equipment param")
		}

		_, err := insertEquipment.Exec(bookingID, v)
		if err != nil {
			return err
		}
	}

	return nil
}

func RemoveBookingEquipment(bookingID int, equipment []string) error {

	removeEquipment, err := conn.Prepare(`DELETE FROM equipmentbooking WHERE (bookingID = ?) and (equipmentID = ?);
`)
	if err != nil {
		return err
	}
	defer removeEquipment.Close()

	for _, v := range equipment {
		if v == "" {
			return errors.New("invalid equipment param")
		}

		_, err := removeEquipment.Exec(bookingID, v)
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
		finish  time.Time
		created time.Time
	)

	row := conn.QueryRow(`SELECT b.*, (SELECT processID FROM bookingstatus 
								INNER JOIN bookings ON bookingstatus.bookingID = b.id 
								INNER JOIN processtype ON bookingstatus.processID = processtype.id
								WHERE bookingstatus.active = 1
								AND processtype.bookingPage = 1
								ORDER BY processtype.order DESC
								LIMIT 1) as processID FROM bookings as b 
								WHERE b.id = ?;`, bookingID)

	booking := &data.Booking{}

	err := row.Scan(&booking.ID, &booking.CarID, &booking.UserID, &start, &end, &finish, &booking.TotalCost,
		&booking.AmountPaid, &booking.LateReturn, &booking.Extension, &created, &booking.BookingLength, &booking.ProcessID)
	if err != nil {
		return nil, err
	}
	booking.Start = *data.ConvertDate(start)
	booking.End = *data.ConvertDate(end)
	booking.Finish = *data.ConvertDate(finish)
	booking.Created = *data.ConvertDate(created)

	return booking, nil
}

func GetUsersBookings(userID int) ([]*data.Booking, error) {
	var (
		start   time.Time
		end     time.Time
		finish  time.Time
		created time.Time
	)

	rows, err := conn.Query(`SELECT b.id,b.start, b.end, b.finish, b.totalCost, b.amountPaid, b.lateReturn, b.extension, b.created, b.bookingLength,
								(SELECT processID FROM bookingstatus 
								INNER JOIN bookings ON bookingstatus.bookingID = b.id 
								INNER JOIN processtype ON bookingstatus.processID = processtype.id
								WHERE bookingstatus.active = 1
								AND processtype.bookingPage = 1
								ORDER BY processtype.order DESC
								LIMIT 1) as processID,
								cars.id as carID, cars.cost, cars.description, cars.image, cars.seats, fuelType.description, gearType.description, carType.description, size.description, colour.description
								FROM bookings AS b
								INNER JOIN cars ON b.carID = cars.id
								INNER JOIN fueltype ON cars.fuelType = fuelType.id
								INNER JOIN gearType ON cars.gearType = gearType.id
								INNER JOIN carType ON cars.carType = carType.id
								INNER JOIN size ON cars.size = size.id
								INNER JOIN colour ON cars.colour = colour.id
								WHERE b.userID = ?
								ORDER BY b.created DESC LIMIT 16`, userID)
	if err != nil {
		return nil, err
	}

	bookings := make([]*data.Booking, 16)

	count := 0
	for rows.Next() {

		booking := &data.Booking{}
		booking.CarData = data.NewCar()
		bookings[count] = booking

		err := rows.Scan(&booking.ID, &start, &end, &finish, &booking.TotalCost, &booking.AmountPaid,
			&booking.LateReturn, &booking.Extension, &created, &booking.BookingLength, &booking.ProcessID,
			&booking.CarData.ID, &booking.CarData.Cost, &booking.CarData.Description, &booking.CarData.Image, &booking.CarData.Seats,
			&booking.CarData.FuelType.Description, &booking.CarData.GearType.Description, &booking.CarData.CarType.Description, &booking.CarData.Size.Description, &booking.CarData.Colour.Description)
		if err != nil {
			return nil, err
		}
		booking.Start = *data.ConvertDate(start)
		booking.End = *data.ConvertDate(end)
		booking.Finish = *data.ConvertDate(finish)
		booking.Created = *data.ConvertDate(created)

		booking.Accessories, err = GetBookingAccessories(booking.ID)
		if err != nil {
			return nil, err
		}

		count++
	}
	bookings = bookings[:count]

	return bookings, nil
}

func UpdateBookingPayment(bookingID, userID int, amount float64) error {
	result, err := conn.Exec(`UPDATE bookings SET amountPaid = ? WHERE (id = ? AND userID = ?)`, amount, bookingID, userID)
	if err != nil {
		return err
	}

	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return errors.New("no rows affected")
	}

	return nil
}

func UpdateBooking(bookingID, userID int, amount, bookingLength float64, lateReturn, extension bool) error {
	result, err := conn.Exec(`UPDATE bookings SET totalCost = ?, lateReturn = ?, extension = ?, bookingLength = ?
								WHERE (id = ? AND userID = ?)`, amount, lateReturn, extension, bookingLength, bookingID, userID)
	if err != nil {
		return err
	}

	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return errors.New("no rows affected")
	}

	return nil
}
