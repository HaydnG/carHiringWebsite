package db

import (
	"carHiringWebsite/data"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

var (
	conn    *sql.DB
	space   = regexp.MustCompile(`\s+`)
	User    *string
	Pass    *string
	Address *string
	Schema  *string
)

func InitDB() error {
	var err error

	conn, err = sql.Open("mysql", *User+":"+*Pass+"@tcp("+*Address+")/"+*Schema+"?parseTime=true&timeout=3s")
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

func CreateCar(fuelType, gearType, carType, size, colour, seats, price int, disabled, over25 bool, fileName, description string) (int, error) {

	//Prepared statements
	createCar, err := conn.Prepare(`INSERT INTO cars
							(fuelType, gearType, carType, size, colour, cost, description, image, seats, disabled, over25)
							VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);`)
	if err != nil {
		return 0, err
	}
	defer createCar.Close()

	res, err := createCar.Exec(fuelType, gearType, carType, size, colour, price, description, fileName, seats, disabled, over25)
	if err != nil {
		return 0, err
	}

	carID, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	if carID == 0 {
		return 0, errors.New("no car inserted")
	}

	return int(carID), nil
}

func UpdateCar(fuelType, gearType, carType, size, colour, seats, price int, disabled, over25 bool, fileName, description string, id int) (bool, error) {

	//Prepared statements
	result, err := conn.Exec(`UPDATE cars SET fuelType = ?, gearType = ?, carType = ?,
 									size = ?, colour = ?, cost = ?, description = ?,
									image = ?, seats = ?, disabled = ?, over25 = ? WHERE (id = ?);`,
		fuelType, gearType, carType, size, colour, price, description, fileName, seats, disabled, over25, id)
	if err != nil {
		return false, err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return false, err
	}

	return rows > 0, nil
}

func CreateUser(email, firstname, names string, dob time.Time, salt, hash string) (int, error) {

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

func UpdateUser(id int, email, firstname, names string, dob time.Time, salt, hash string) error {
	result, err := conn.Exec("UPDATE users SET firstname = ?, names = ?, email = ?, authHash = ?, authSalt = ?, DOB = ? WHERE (id = ?)",
		firstname, names, email, hash, salt, dob, id)
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

func SelectUserByEmail(email string) (*data.User, error) {
	row := conn.QueryRow("SELECT u.*, (select count(*) from bookings as b where b.userID = u.id) as bookingCount FROM USERS as u WHERE u.email = ?", email)

	return readUserRow(row)
}

func SelectUserByID(id int) (*data.User, error) {
	row := conn.QueryRow("SELECT u.*, (select count(*) from bookings as b where b.userID = u.id) as bookingCount FROM USERS as u WHERE u.id = ?", id)

	return readUserRow(row)
}

func GetUsers(userSearch string) ([]*data.OutputUser, error) {

	userSearch = space.ReplaceAllString(userSearch, " ")
	userSearch = strings.TrimSpace(userSearch)
	userSearch = fmt.Sprintf("%%%s%%", userSearch)

	var (
		createdAt time.Time
		dob       time.Time
	)

	rows, err := conn.Query(`SELECT u.id, u.firstname, u.names, u.email, u.createdAt, u.blackListed, u.DOB, u.repeat, u.admin, u.disabled, 
										(select count(*) from bookings as b where b.userID = u.id) as bookingCount
										FROM USERS as u 
										WHERE u.firstname like ? OR u.names like ? OR u.email like ? LIMIT 32`,
		userSearch, userSearch, userSearch)
	if err != nil {
		return nil, err
	}

	users := make([]*data.OutputUser, 32)
	count := 0
	for rows.Next() {

		newUser := &data.OutputUser{}
		users[count] = newUser

		err := rows.Scan(&newUser.ID, &newUser.FirstName, &newUser.Names, &newUser.Email, &createdAt, &newUser.Blacklisted, &dob, &newUser.Repeat, &newUser.Admin, &newUser.Disabled, &newUser.BookingCount)
		if err != nil {
			return nil, err
		}

		newUser.CreatedAt = *data.ConvertDate(createdAt)
		newUser.DOB = *data.ConvertDate(dob)

		count++
	}

	users = users[:count]

	return users, nil

}

func SetRepeatUser(userID int) error {
	result, err := conn.Exec("UPDATE users SET `repeat` = 1 WHERE (id = ?);", userID)
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

func SetDisableUser(userID int, value bool) error {
	result, err := conn.Exec("UPDATE users SET `disabled` = ? WHERE (id = ?);", value, userID)
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
func SetAdminUser(userID int, value bool) error {
	result, err := conn.Exec("UPDATE users SET `admin` = ? WHERE (id = ?);", value, userID)
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
func SetBlackListUser(userID int, value bool) error {
	result, err := conn.Exec("UPDATE users SET `blackListed` = ? WHERE (id = ?);", value, userID)
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

func readUserRow(row *sql.Row) (*data.User, error) {
	newUser := data.User{}

	err := row.Scan(&newUser.ID, &newUser.FirstName, &newUser.Names, &newUser.Email, &newUser.CreatedAt, &newUser.AuthHash, &newUser.AuthSalt, &newUser.Blacklisted, &newUser.DOB, &newUser.Verified, &newUser.Repeat, &newUser.Admin, &newUser.Disabled, &newUser.BookingCount)
	if err != nil {
		return &newUser, err
	}

	return &newUser, nil
}

// Database User Logic
//
//
//`SELECT b.*,cars.description, users.firstname, users.names, P.pid, P.description FROM bookings as b
//INNER JOIN (SELECT bookings.id,processtype.id as pid,processtype.description FROM bookingstatus
//								INNER JOIN bookings ON bookingstatus.bookingID = bookings.id
//								INNER JOIN processtype ON bookingstatus.processID = processtype.id
//								WHERE bookingstatus.active = 1 AND
//                                processtype.bookingPage = 1
//								ORDER BY processtype.order DESC) P on p.id = b.id
//INNER JOIN users on b.userID = users.id
//INNER JOIN cars on cars.id = b.carID
//WHERE (users.id like ?
//OR users.firstname like ?
//OR users.names like ?
//OR users.email like ?
//OR CONCAT(users.firstname, ' ', users.names) like ?) AND
//b.id like ? AND
//P.pid NOT in (?`+strings.Repeat(",?", len(filters)-1)+`)
//ORDER BY b.created DESC LIMIT 10;`

func GetAllCars(fuelTypes, gearTypes, carTypes, carSizes, colourTypes, search string) ([]*data.Car, error) {

	search = space.ReplaceAllString(search, " ")
	search = strings.TrimSpace(search)
	search = fmt.Sprintf("%%%s%%", search)

	args := []interface{}{search, search, search, search, search, search, search}

	sql := `SELECT cars.*, fuelType.description, gearType.description, carType.description, size.description, colour.description
	FROM carrental.cars
	INNER JOIN fueltype ON cars.fuelType = fuelType.id
	INNER JOIN gearType ON cars.gearType = gearType.id
	INNER JOIN carType ON cars.carType = carType.id
	INNER JOIN size ON cars.size = size.id
	INNER JOIN colour ON cars.colour = colour.id
	WHERE cars.disabled = 0 AND
	(cars.Description like ? or cars.seats like ? or fuelType.description like ? or gearType.description like ? or
		carType.description like ? or size.description like ? or colour.description like ?)`

	fuels := strings.Split(fuelTypes, ",")
	if len(fuels) > 0 && fuels[0] != "" {
		for _, x := range fuels {
			args = append(args, x)
		}
		sql += `AND cars.fuelType in (?` + strings.Repeat(",?", len(fuels)-1) + `)`
	}

	gears := strings.Split(gearTypes, ",")
	if len(gears) > 0 && gears[0] != "" {
		for _, x := range gears {
			args = append(args, x)
		}
		sql += `AND cars.gearType in (?` + strings.Repeat(",?", len(gears)-1) + `)`
	}

	types := strings.Split(carTypes, ",")
	if len(types) > 0 && types[0] != "" {
		for _, x := range types {
			args = append(args, x)
		}
		sql += `AND cars.carType in (?` + strings.Repeat(",?", len(types)-1) + `)`
	}

	sizes := strings.Split(carSizes, ",")
	if len(sizes) > 0 && sizes[0] != "" {
		for _, x := range sizes {
			args = append(args, x)
		}
		sql += `AND cars.size in (?` + strings.Repeat(",?", len(sizes)-1) + `)`
	}

	colours := strings.Split(colourTypes, ",")
	if len(colours) > 0 && colours[0] != "" {
		for _, x := range colours {
			args = append(args, x)
		}
		sql += `AND cars.colour in (?` + strings.Repeat(",?", len(colours)-1) + `)`
	}
	sql += ` LIMIT 48`

	rows, err := conn.Query(sql, args...)
	if err != nil {
		return nil, err
	}

	cars := make([]*data.Car, 48)
	count := 0
	for rows.Next() {

		car := data.NewCar()
		cars[count] = car

		err := rows.Scan(&car.ID, &car.FuelType.ID, &car.GearType.ID, &car.CarType.ID, &car.Size.ID, &car.Colour.ID, &car.Cost, &car.Description, &car.Image, &car.Seats, &car.Disabled, &car.Over25,
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

	err := row.Scan(&car.ID, &car.FuelType.ID, &car.GearType.ID, &car.CarType.ID, &car.Size.ID, &car.Colour.ID, &car.Cost, &car.Description, &car.Image, &car.Seats, &car.Disabled, &car.Over25,
		&car.FuelType.Description, &car.GearType.Description, &car.CarType.Description, &car.Size.Description, &car.Colour.Description)
	if err != nil {
		return car, err
	}
	return car, nil
}

func AdminGetCars(fuelTypes, gearTypes, carTypes, carSizes, colourTypes, search string) ([]*data.Car, error) {

	search = space.ReplaceAllString(search, " ")
	search = strings.TrimSpace(search)
	search = fmt.Sprintf("%%%s%%", search)

	args := []interface{}{search, search, search, search, search, search, search}

	sql := `SELECT c.*, fuelType.description, gearType.description, carType.description, size.description, colour.description, COALESCE(b.bookingCount, 0) as bookingCount
	FROM carrental.cars c
	INNER JOIN fueltype ON c.fuelType = fuelType.id
	INNER JOIN gearType ON c.gearType = gearType.id
	INNER JOIN carType ON c.carType = carType.id
	INNER JOIN size ON c.size = size.id
	INNER JOIN colour ON c.colour = colour.id
	LEFT JOIN (select carID,count(*) as bookingCount from bookings group by carID) as b on b.carID = c.id
	WHERE (c.Description like ? or c.seats like ? or fuelType.description like ? or gearType.description like ? or
		carType.description like ? or size.description like ? or colour.description like ?)`

	fuels := strings.Split(fuelTypes, ",")
	if len(fuels) > 0 && fuels[0] != "" {
		for _, x := range fuels {
			args = append(args, x)
		}
		sql += `AND c.fuelType in (?` + strings.Repeat(",?", len(fuels)-1) + `)`
	}

	gears := strings.Split(gearTypes, ",")
	if len(gears) > 0 && gears[0] != "" {
		for _, x := range gears {
			args = append(args, x)
		}
		sql += `AND c.gearType in (?` + strings.Repeat(",?", len(gears)-1) + `)`
	}

	types := strings.Split(carTypes, ",")
	if len(types) > 0 && types[0] != "" {
		for _, x := range types {
			args = append(args, x)
		}
		sql += `AND c.carType in (?` + strings.Repeat(",?", len(types)-1) + `)`
	}

	sizes := strings.Split(carSizes, ",")
	if len(sizes) > 0 && sizes[0] != "" {
		for _, x := range sizes {
			args = append(args, x)
		}
		sql += `AND c.size in (?` + strings.Repeat(",?", len(sizes)-1) + `)`
	}

	colours := strings.Split(colourTypes, ",")
	if len(colours) > 0 && colours[0] != "" {
		for _, x := range colours {
			args = append(args, x)
		}
		sql += `AND c.colour in (?` + strings.Repeat(",?", len(colours)-1) + `)`
	}
	sql += ` LIMIT 48`

	rows, err := conn.Query(sql, args...)
	if err != nil {
		return nil, err
	}

	cars := make([]*data.Car, 32)
	count := 0
	for rows.Next() {
		car := data.NewCar()
		cars[count] = car

		err := rows.Scan(&car.ID, &car.FuelType.ID, &car.GearType.ID, &car.CarType.ID, &car.Size.ID, &car.Colour.ID, &car.Cost, &car.Description, &car.Image, &car.Seats, &car.Disabled, &car.Over25,
			&car.FuelType.Description, &car.GearType.Description, &car.CarType.Description, &car.Size.Description, &car.Colour.Description, &car.BookingCount)
		if err != nil {
			return nil, err
		}
		count++
	}

	cars = cars[:count]

	return cars, nil

}

//Car bookins count
//select * , (SELECT COUNT(*) FROM bookings AS b
//WHERE ((now() <= b.end ) AND (now() >= b.start))
//AND (SELECT processID FROM bookingstatus
//INNER JOIN bookings ON bookingstatus.bookingID = b.id
//INNER JOIN processtype ON bookingstatus.processID = processtype.id
//WHERE bookingstatus.active = 1
//ORDER BY processtype.order DESC
//LIMIT 1) != 11
//AND b.carID = c.id) as bookings from cars as c

//CAR SEARCH
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
bookingstatus.processID, bookingstatus.extra, processtype.description, processtype.adminRequired, processtype.order, processtype.bookingPage
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
			&status.ProcessID, &status.Extra, &status.ProcessDescription, &status.AdminRequired, &status.Order, &status.BookingPage)
		if err != nil {
			return nil, err
		}
		status.Completed = *data.ConvertDate(completed)

		count++
	}

	statuses = statuses[:count]

	return statuses, nil
}

func CountExtensionDays(start, end string, carID, bookingID int) (*data.ExtensionResponse, error) {

	row := conn.QueryRow(`SELECT DATEDIFF(b.start, ?) as extensionDays FROM bookings as b
						WHERE ((? <= b.finish ) and (? >= b.start))
						AND b.carID = ? 
						AND b.ID != ?
						AND (SELECT processID FROM bookingstatus 
								INNER JOIN bookings ON bookingstatus.bookingID = b.id 
								INNER JOIN processtype ON bookingstatus.processID = processtype.id
								WHERE bookingstatus.active = 1
								ORDER BY processtype.order DESC
								LIMIT 1) != 11
						ORDER BY b.start ASC
						LIMIT 1 `, start, start, end, carID, bookingID)

	response := &data.ExtensionResponse{Days: 0}

	err := row.Scan(&response.Days)
	if err != nil {
		if err == sql.ErrNoRows {
			response.Days = 14
		} else {
			return nil, err
		}
	}

	return response, nil
}

func GetCarBookings(start, end string, carID int) ([]*data.TimeRange, error) {
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

func GetUpcomingBookings(processID string, limit int) ([]*data.BookingColumn, error) {

	var (
		start   time.Time
		end     time.Time
		finish  time.Time
		created time.Time
	)

	rows, err := conn.Query(`SELECT b.*,cars.description, users.firstname, users.names, P.pid, P.description FROM bookings as b
INNER JOIN (SELECT bookings.id,processtype.id as pid,processtype.description FROM bookingstatus 
								INNER JOIN bookings ON bookingstatus.bookingID = bookings.id 
								INNER JOIN processtype ON bookingstatus.processID = processtype.id
								WHERE bookingstatus.active = 1 AND
                                processtype.bookingPage = 1
								ORDER BY processtype.order DESC) P on p.id = b.id
INNER JOIN users on b.userID = users.id
INNER JOIN cars on cars.id = b.carID
WHERE P.pid = ?
ORDER BY b.start ASC LIMIT ?;`, processID, limit)

	if err != nil {
		return nil, err
	}
	columns := make([]*data.BookingColumn, limit)

	count := 0
	for rows.Next() {

		column := &data.BookingColumn{}
		columns[count] = column

		err := rows.Scan(&column.ID, &column.CarID, &column.UserID, &start, &end, &finish, &column.TotalCost, &column.AmountPaid, &column.LateReturn, &column.FullDay, &created, &column.BookingLength, &column.PerDay, &column.DriverID, &column.CarDescription, &column.UserFirstName, &column.UserOtherName, &column.ProcessID, &column.Process)
		if err != nil {
			return nil, err
		}
		column.Start = *data.ConvertDate(start)
		column.End = *data.ConvertDate(end)
		column.Finish = *data.ConvertDate(finish)
		column.Created = *data.ConvertDate(created)

		count++
	}

	columns = columns[:count]

	return columns, nil
}

func GetQueryingRefundBookings() ([]*data.BookingColumn, error) {

	var (
		start   time.Time
		end     time.Time
		finish  time.Time
		created time.Time
	)

	rows, err := conn.Query(`SELECT b.*,cars.description, users.firstname, users.names, RS.pid, RS.description FROM bookings as b
INNER JOIN (SELECT bookings.id,processtype.id as pid,processtype.description FROM bookingstatus 
								INNER JOIN bookings ON bookingstatus.bookingID = bookings.id 
								INNER JOIN processtype ON bookingstatus.processID = processtype.id
								WHERE bookingstatus.active = 1 AND
                                processtype.id = 8
								ORDER BY processtype.order DESC) RS on RS.id = b.id
INNER JOIN users on b.userID = users.id
INNER JOIN cars on cars.id = b.carID
WHERE RS.pid = 8
ORDER BY b.start ASC LIMIT 10;`)

	if err != nil {
		return nil, err
	}
	columns := make([]*data.BookingColumn, 10)

	count := 0
	for rows.Next() {

		column := &data.BookingColumn{}
		columns[count] = column

		err := rows.Scan(&column.ID, &column.CarID, &column.UserID, &start, &end, &finish, &column.TotalCost, &column.AmountPaid, &column.LateReturn, &column.FullDay, &created, &column.BookingLength, &column.PerDay, &column.DriverID, &column.CarDescription, &column.UserFirstName, &column.UserOtherName, &column.ProcessID, &column.Process)
		if err != nil {
			return nil, err
		}
		column.Start = *data.ConvertDate(start)
		column.End = *data.ConvertDate(end)
		column.Finish = *data.ConvertDate(finish)
		column.Created = *data.ConvertDate(created)

		count++
	}

	columns = columns[:count]

	return columns, nil
}

func GetAdminUsersBookings(userID int) ([]*data.BookingColumn, error) {

	var (
		start   time.Time
		end     time.Time
		finish  time.Time
		created time.Time
	)

	rows, err := conn.Query(`SELECT b.*,cars.description, users.firstname, users.names, RS.pid, RS.description FROM bookings as b
INNER JOIN users on b.userID = users.id
INNER JOIN cars on cars.id = b.carID
INNER JOIN (SELECT bookings.id,processtype.id as pid,processtype.description FROM bookingstatus 
								INNER JOIN bookings ON bookingstatus.bookingID = bookings.id 
								INNER JOIN processtype ON bookingstatus.processID = processtype.id
								WHERE bookingstatus.active = 1
								AND processtype.bookingPage = 1
								ORDER BY processtype.order DESC) RS on RS.id = b.id
WHERE users.id = ?
ORDER BY b.created ASC LIMIT 20;`, userID)

	if err != nil {
		return nil, err
	}
	columns := make([]*data.BookingColumn, 20)

	count := 0
	for rows.Next() {

		column := &data.BookingColumn{}
		columns[count] = column

		err := rows.Scan(&column.ID, &column.CarID, &column.UserID, &start, &end, &finish, &column.TotalCost, &column.AmountPaid, &column.LateReturn, &column.FullDay, &created, &column.BookingLength, &column.PerDay, &column.DriverID, &column.CarDescription, &column.UserFirstName, &column.UserOtherName, &column.ProcessID, &column.Process)
		if err != nil {
			return nil, err
		}
		column.Start = *data.ConvertDate(start)
		column.End = *data.ConvertDate(end)
		column.Finish = *data.ConvertDate(finish)
		column.Created = *data.ConvertDate(created)

		count++
	}

	columns = columns[:count]

	return columns, nil
}

func GetAwaitingConfirmationBookings() ([]*data.BookingColumn, error) {

	var (
		start   time.Time
		end     time.Time
		finish  time.Time
		created time.Time
	)

	rows, err := conn.Query(`SELECT b.*,cars.description, users.firstname, users.names FROM bookings as b
INNER JOIN users on b.userID = users.id
INNER JOIN cars on cars.id = b.carID
WHERE (SELECT processID FROM bookingstatus 
								INNER JOIN bookings ON bookingstatus.bookingID = b.id 
								INNER JOIN processtype ON bookingstatus.processID = processtype.id
								WHERE bookingstatus.active = 1 AND
                                processtype.bookingPage = 1
								ORDER BY processtype.order DESC
								LIMIT 1) = 4
ORDER BY b.start ASC LIMIT 5;`)

	if err != nil {
		return nil, err
	}
	columns := make([]*data.BookingColumn, 5)

	count := 0
	for rows.Next() {

		column := &data.BookingColumn{}
		columns[count] = column

		err := rows.Scan(&column.ID, &column.CarID, &column.UserID, &start, &end, &finish, &column.TotalCost, &column.AmountPaid, &column.LateReturn, &column.FullDay, &created, &column.BookingLength, &column.PerDay, &column.DriverID, &column.CarDescription, &column.UserFirstName, &column.UserOtherName)
		if err != nil {
			return nil, err
		}
		column.Start = *data.ConvertDate(start)
		column.End = *data.ConvertDate(end)
		column.Finish = *data.ConvertDate(finish)
		column.Created = *data.ConvertDate(created)

		count++
	}

	columns = columns[:count]

	return columns, nil
}

func GetSearchedBookings(userSearch, bookingSearch, statusFilter string) ([]*data.BookingColumn, error) {
	var (
		start   time.Time
		end     time.Time
		finish  time.Time
		created time.Time
	)

	userSearch = space.ReplaceAllString(userSearch, " ")
	bookingSearch = space.ReplaceAllString(bookingSearch, " ")

	userSearch = strings.TrimSpace(userSearch)
	bookingSearch = strings.TrimSpace(bookingSearch)

	userSearch = fmt.Sprintf("%%%s%%", userSearch)
	bookingSearch = fmt.Sprintf("%%%s%%", bookingSearch)

	filters := strings.Split(statusFilter, ",")
	args := []interface{}{userSearch, userSearch, userSearch, userSearch, userSearch, bookingSearch}
	for _, x := range filters {
		args = append(args, x)
	}

	rows, err := conn.Query(`SELECT b.*,cars.description, users.firstname, users.names, P.pid, P.description FROM bookings as b
INNER JOIN (SELECT bookings.id,processtype.id as pid,processtype.description FROM bookingstatus 
								INNER JOIN bookings ON bookingstatus.bookingID = bookings.id 
								INNER JOIN processtype ON bookingstatus.processID = processtype.id
								WHERE bookingstatus.active = 1 AND
                                processtype.bookingPage = 1
								ORDER BY processtype.order DESC) P on p.id = b.id
INNER JOIN users on b.userID = users.id
INNER JOIN cars on cars.id = b.carID
WHERE (users.id like ?
OR users.firstname like ?
OR users.names like ?
OR users.email like ?
OR CONCAT(users.firstname, ' ', users.names) like ?) AND
b.id like ? AND 
P.pid NOT in (?`+strings.Repeat(",?", len(filters)-1)+`)
ORDER BY b.created DESC LIMIT 10;`, args...)

	if err != nil {
		return nil, err
	}
	columns := make([]*data.BookingColumn, 10)

	count := 0
	for rows.Next() {

		column := &data.BookingColumn{}
		columns[count] = column

		err := rows.Scan(&column.ID, &column.CarID, &column.UserID, &start, &end, &finish, &column.TotalCost, &column.AmountPaid, &column.LateReturn, &column.FullDay, &created, &column.BookingLength, &column.PerDay, &column.DriverID, &column.CarDescription, &column.UserFirstName, &column.UserOtherName, &column.ProcessID, &column.Process)
		if err != nil {
			return nil, err
		}
		column.Start = *data.ConvertDate(start)
		column.End = *data.ConvertDate(end)
		column.Finish = *data.ConvertDate(finish)
		column.Created = *data.ConvertDate(created)

		count++
	}

	columns = columns[:count]

	return columns, nil
}

func BookingHasOverlap(start, end string, carID int) (bool, error) {

	row := conn.QueryRow(`SELECT COUNT(*) AS overlaps FROM bookings AS b
INNER JOIN (SELECT bookings.id,processtype.id as pid,processtype.description FROM bookingstatus 
								INNER JOIN bookings ON bookingstatus.bookingID = bookings.id 
								INNER JOIN processtype ON bookingstatus.processID = processtype.id
								WHERE bookingstatus.active = 1 AND
                                processtype.bookingPage = 1
								ORDER BY processtype.order DESC) P on p.id = b.id
								WHERE ((? <= b.end ) AND (? >= b.start))
								AND P.pid != 11
								AND b.carID = ?`, start, end, carID)
	overlaps := 0

	err := row.Scan(&overlaps)
	if err != nil {
		return false, err
	}

	return overlaps > 0, nil
}

func GetDriverByName(lastName, names string) (*data.Driver, error) {

	var dob time.Time

	row := conn.QueryRow(`SELECT * from drivers where lastName = ? and names = ?`, lastName, names)

	driver := &data.Driver{}

	err := row.Scan(&driver.ID, &driver.LastName, &driver.Names, &driver.LicenseNumber, &driver.Address, &driver.PostCode, &driver.BlackListed, &dob)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	driver.DOB = *data.ConvertDate(dob)

	return driver, nil
}

func GetDriverByID(ID int) (*data.Driver, error) {

	var dob time.Time

	row := conn.QueryRow(`SELECT * from drivers where id = ?`, ID)

	driver := &data.Driver{}

	err := row.Scan(&driver.ID, &driver.LastName, &driver.Names, &driver.LicenseNumber, &driver.Address, &driver.PostCode, &driver.BlackListed, &dob)
	if err != nil {
		return nil, err
	}

	driver.DOB = *data.ConvertDate(dob)

	return driver, nil
}

func CreateDriver(lastName, names, license, address, postcode string, blackListed bool, dob time.Time) (int, error) {

	//Prepared statements
	createDriver, err := conn.Prepare(`INSERT INTO drivers(lastName, names, licenseNumber, address, postcode, blackListed, dob)
												VALUES(?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return 0, err
	}
	defer createDriver.Close()

	res, err := createDriver.Exec(lastName, names, license, address, postcode, blackListed, dob)
	if err != nil {
		return 0, err
	}

	driverID, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	if driverID == 0 {
		return 0, errors.New("no driver inserted")
	}

	return int(driverID), nil
}

func AddBookingDriver(bookingID, driverID int) error {
	result, err := conn.Exec("update bookings set driverID = ? where id = ?",
		driverID, bookingID)
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

func BlackListedDriver(id int) error {
	result, err := conn.Exec("update driver set blackListed = ? where id = ?",
		true, id)
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

func UpdateDriver(id int, licenseNumber, address, postcode string, blackListed bool, dob time.Time) error {

	result, err := conn.Exec(`update drivers set licenseNumber = ?, address = ?, postcode = ?, blackListed = ?, dob = ? WHERE id  = ?`,
		licenseNumber, address, postcode, blackListed, dob, id)
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

func CreateBooking(carID, userID int, start, end, finish string, price float64, lateReturn, fullDay bool, bookingLength, cost float64) (int, error) {

	//Prepared statements
	createBooking, err := conn.Prepare(`INSERT INTO bookings(carID, userID, start, end, finish,totalCost, amountPaid, lateReturn, fullDay, created, bookingLength, perDay, driverID)
												VALUES(?, ?, ?, ?, ?, ?, '0', ?, ?, ?, ?, ?, NULL)`)
	if err != nil {
		return 0, err
	}
	defer createBooking.Close()

	res, err := createBooking.Exec(carID, userID, start, end, finish, price, lateReturn, fullDay, time.Now(), bookingLength, cost)
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

func InsertBookingStatus(bookingID, processID, adminID, active int, extra float64, description string) (int, error) {

	//Prepared statements
	insertBookingStatus, err := conn.Prepare(`INSERT INTO bookingstatus(bookingID, processID, completed, active, adminID, description, extra)
												VALUES(?, ?, ?, ?, ?, ?,?)`)
	if err != nil {
		return 0, err
	}
	defer insertBookingStatus.Close()

	res, err := insertBookingStatus.Exec(bookingID, processID, time.Now(), active, adminID, description, extra)
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

	result, err := conn.Exec(`UPDATE bookingstatus SET active = 0 WHERE (bookingID = ?)`, bookingID)
	if err != nil {
		return err
	}

	_, err = result.RowsAffected()
	if err != nil {
		return err
	}

	return nil
}

func SetBookingStatus(statusID int, active bool) error {

	result, err := conn.Exec(`UPDATE bookingstatus SET active = ? WHERE (id = ?)`, active, statusID)
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

//GetBookingProcessStatus returns the most recent process with processID specified
func GetBookingProcessStatus(bookingID, processID int) (*data.BookingStatus, error) {
	bookingStatus := &data.BookingStatus{}
	var completed time.Time

	result := conn.QueryRow(`SELECT * FROM carrental.bookingstatus
								WHERE bookingID = ? AND processID = ?
								ORDER  BY completed DESC LIMIT 1`, bookingID, processID)
	err := result.Scan(&bookingStatus.ID, &bookingStatus.BookingID, &bookingStatus.ProcessID, &completed, &bookingStatus.Active, &bookingStatus.AdminID, &bookingStatus.Description, &bookingStatus.Extra)
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

func GetCarAttributes() (map[string][]*data.CarAttribute, error) {
	rows, err := conn.Query(`SELECT '0' as typeIndex, cartype.description, cartype.id from cartype
UNION
SELECT '1' as typeIndex, colour.description, colour.id from colour
UNION
SELECT '2' as typeIndex, fueltype.description, fueltype.id from fueltype
UNION
SELECT '3' as typeIndex, geartype.description, geartype.id from geartype
UNION
SELECT '4' as typeIndex, size.description, size.id from size
`)
	if err != nil {
		return nil, err
	}
	attributes := make(map[string][]*data.CarAttribute)

	lastCount := 0
	lastIndex := ""
	typeIndex := ""
	count := 0
	for rows.Next() {
		attr := &data.CarAttribute{}

		err := rows.Scan(&typeIndex, &attr.Description, &attr.ID)
		if err != nil {
			return nil, err
		}

		if _, ok := attributes[typeIndex]; ok {
			if count >= 5 {
				attributes[typeIndex] = append(attributes[typeIndex], attr)
			} else {
				attributes[typeIndex][count] = attr
			}
			count++
		} else {
			lastCount = count
			count = 0
			attributes[typeIndex] = make([]*data.CarAttribute, 5)
			attributes[typeIndex][count] = attr
			count++
		}

		if count == 1 {
			attributes[lastIndex] = attributes[lastIndex][:lastCount]
		}

		lastIndex = typeIndex
	}
	attributes[lastIndex] = attributes[lastIndex][:count]

	return attributes, nil
}

func GetBookingStats() ([]*data.BookingStat, error) {

	rows, err := conn.Query(`SELECT processID, processtype.description , count(*) as count, processtype.adminRequired  FROM bookings as b 
								INNER JOIN bookingstatus ON bookingstatus.bookingID = b.id 
								INNER JOIN processtype ON bookingstatus.processID = processtype.id
								WHERE bookingstatus.active = 1
                                group by processID
								ORDER BY processtype.order ASC LIMIT 15`)
	if err != nil {
		return nil, err
	}
	stats := make([]*data.BookingStat, 15)

	count := 0
	for rows.Next() {

		stat := &data.BookingStat{}
		stats[count] = stat

		err := rows.Scan(&stat.ProcessID, &stat.Description, &stat.Count, &stat.AdminRequired)
		if err != nil {
			return nil, err
		}

		count++
	}

	stats = stats[:count]

	return stats, nil
}

func GetActiveBookingStatuses(bookingID int) ([]*data.BookingStatusType, error) {

	rows, err := conn.Query(`SELECT pt.* FROM processtype pt
								Inner join bookingstatus bs on bs.processID = pt.id
								WHERE bs.bookingID = ?
								AND bs.active = 1
								LIMIT 17`, bookingID)
	if err != nil {
		return nil, err
	}
	statuses := make([]*data.BookingStatusType, 17)

	count := 0
	for rows.Next() {

		status := &data.BookingStatusType{}
		statuses[count] = status

		err := rows.Scan(&status.ID, &status.Description, &status.AdminRequired, &status.Order, &status.BookingPage)
		if err != nil {
			return nil, err
		}

		count++
	}

	statuses = statuses[:count]

	return statuses, nil
}

func GetBookingStatuses() ([]*data.BookingStatusType, error) {

	rows, err := conn.Query(`SELECT * FROM carrental.processtype WHERE processtype.bookingPage = 1 LIMIT 17`)
	if err != nil {
		return nil, err
	}
	statuses := make([]*data.BookingStatusType, 13)

	count := 0
	for rows.Next() {

		status := &data.BookingStatusType{}
		statuses[count] = status

		err := rows.Scan(&status.ID, &status.Description, &status.AdminRequired, &status.Order, &status.BookingPage)
		if err != nil {
			return nil, err
		}

		count++
	}

	statuses = statuses[:count]

	return statuses, nil
}

func GetUserStats() (*data.UserStat, error) {

	row := conn.QueryRow(`SELECT 
sum(case users.admin when 1 then 1 else 0 end) as adminCount,
sum(case users.blackListed when 1 then 1 else 0 end) as blackListedCount,
sum(case users.repeat when 1 then 1 else 0 end) as repeatCount,
count(*) as userCount,
sum(case users.disabled when 1 then 1 else 0 end) as disabledCount
FROM users;`)

	stat := &data.UserStat{}

	err := row.Scan(&stat.AdminCount, &stat.BlackListedCount, &stat.RepeatUsersCount, &stat.UserCount, &stat.DisabledCount)
	if err != nil {
		return nil, err
	}

	return stat, nil
}

func GetAccessoryStats() ([]*data.AccessoryStat, error) {

	rows, err := conn.Query(`select a1.id, a1.description, (a1.stock -
(select COUNT(*) from equipmentbooking
inner join bookings on bookings.id = equipmentbooking.bookingID
inner join equipment on equipmentbooking.equipmentID = equipment.id
where
((now() <= bookings.end ) and (now() >= bookings.start))
And equipment.id = a1.id))
as stock
from equipment as a1
LIMIT 15`)
	if err != nil {
		return nil, err
	}
	stats := make([]*data.AccessoryStat, 15)

	count := 0
	for rows.Next() {

		stat := &data.AccessoryStat{}
		stats[count] = stat

		err := rows.Scan(&stat.ID, &stat.Description, &stat.Stock)
		if err != nil {
			return nil, err
		}

		count++
	}

	stats = stats[:count]

	return stats, nil
}

func GetCarStats() (*data.CarStat, error) {

	row := conn.QueryRow(`select
count(*) as cars,
coalesce(sum(case disabled when 1 then 1 else 0 end), 0) as disabled,
coalesce((select COUNT(*) from cars) - (SELECT COUNT(*) FROM bookings AS b
			INNER Join cars ca on ca.id = b.carID
			WHERE (((now() <= b.end ) AND (now() >= b.start)) AND (SELECT processID FROM bookingstatus 
			INNER JOIN bookings ON bookingstatus.bookingID = b.id 
			INNER JOIN processtype ON bookingstatus.processID = processtype.id
			WHERE bookingstatus.active = 1
            AND bookings.carID = ca.id
			ORDER BY processtype.order DESC
			LIMIT 1) != 11)
			or ca.disabled = 1), 0) as available
from cars as c`)

	stat := &data.CarStat{}

	err := row.Scan(&stat.CarCount, &stat.DisabledCount, &stat.AvailableCount)
	if err != nil {
		return nil, err
	}

	return stat, nil
}

func GetSingleBooking(bookingID int) (*data.Booking, error) {
	var (
		start   time.Time
		end     time.Time
		finish  time.Time
		created time.Time
	)

	row := conn.QueryRow(`SELECT b.*, P.pid, P.description, P.adminRequired, 
CASE
   When (select count(*) from bookingstatus
		where bookingstatus.processID in (6,8)
		AND bookingstatus.active = 1
        AND b.id = bookingstatus.bookingID) > 0 Then 1
   ELSE 0
END as awaitingExtraPayment,
CASE
   When b.totalCost < b.amountPaid || P.pid = 11 Then 1
   ELSE 0
END as isRefund
FROM bookings as b
INNER JOIN 
	(SELECT bookings.id,processtype.id as pid,processtype.description, processtype.adminRequired 
    FROM bookingstatus
    INNER JOIN bookings ON bookingstatus.bookingID = bookings.id
    INNER JOIN processtype ON bookingstatus.processID = processtype.id
    WHERE bookingstatus.active = 1 AND
    processtype.bookingPage = 1
    ORDER BY processtype.order DESC) P on p.id = b.id
    WHERE b.id = ?`, bookingID)

	booking := &data.Booking{}

	err := row.Scan(&booking.ID, &booking.CarID, &booking.UserID, &start, &end, &finish, &booking.TotalCost,
		&booking.AmountPaid, &booking.LateReturn, &booking.FullDay, &created, &booking.BookingLength, &booking.PerDay, &booking.DriverID, &booking.ProcessID, &booking.ProcessName, &booking.AdminRequired, &booking.AwaitingExtraPayment, &booking.IsRefund)
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

	rows, err := conn.Query(`SELECT b.id,b.start, b.end, b.finish, b.totalCost, b.amountPaid, b.lateReturn, b.fullDay, b.created, b.bookingLength, b.perDay,b.driverID ,P.pid,
								cars.id as carID, cars.cost, cars.description, cars.image, cars.seats, fuelType.description, gearType.description, carType.description, size.description, colour.description
								FROM bookings AS b
								INNER JOIN (SELECT bookings.id,processtype.id as pid,processtype.description, processtype.adminRequired, processtype.order  FROM bookingstatus 
								INNER JOIN bookings ON bookingstatus.bookingID = bookings.id 
								INNER JOIN processtype ON bookingstatus.processID = processtype.id
								WHERE bookingstatus.active = 1 AND
                                processtype.bookingPage = 1
								ORDER BY processtype.order DESC) P on p.id = b.id
								INNER JOIN cars ON b.carID = cars.id
								INNER JOIN fueltype ON cars.fuelType = fuelType.id
								INNER JOIN gearType ON cars.gearType = gearType.id
								INNER JOIN carType ON cars.carType = carType.id
								INNER JOIN size ON cars.size = size.id
								INNER JOIN colour ON cars.colour = colour.id
								WHERE b.userID = ?
								ORDER BY P.order ASC, b.created DESC LIMIT 16`, userID)
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
			&booking.LateReturn, &booking.FullDay, &created, &booking.BookingLength, &booking.PerDay, &booking.DriverID, &booking.ProcessID,
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

func UpdateBooking(bookingID int, amount, bookingLength float64, lateReturn, fullDay bool, end, finish string) error {
	result, err := conn.Exec("UPDATE bookings SET totalCost = ?, lateReturn = ?, fullDay = ?, bookingLength = ?, `end` = ?, `finish` = ? WHERE (id = ?)",
		amount, lateReturn, fullDay, bookingLength, end, finish, bookingID)
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
