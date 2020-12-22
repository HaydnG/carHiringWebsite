package bookingService

import (
	"carHiringWebsite/data"
	"carHiringWebsite/db"
	"carHiringWebsite/session"
	"errors"
	"strconv"
	"strings"
	"time"
)

const (
	lateReturnIncrease = 0.6
	extensionIncrease  = 0.5
	PaymentNeeded      = iota
)

func Create(token, start, end, carID, late, extension, accessories, days string) (*data.Booking, error) {
	var finishString string

	err := session.ValidateToken(token)
	if err != nil {
		return nil, err
	}

	bag, err := session.GetByToken(token)
	if err != nil {
		return nil, err
	}
	user := bag.GetUser()

	startNum, err := strconv.ParseInt(start, 10, 64)
	if err != nil {
		return nil, err
	}
	endNum, err := strconv.ParseInt(end, 10, 64)
	if err != nil {
		return nil, err
	}

	if startNum > endNum {
		return nil, errors.New("start date bigger than end date")
	}

	startTime := time.Unix(startNum, 0)
	endTime := time.Unix(endNum, 0)
	finishTime := time.Unix(endNum, 0)

	calculatedDays := (endTime.Sub(startTime).Hours() / 24) + 0.5

	lateValue, err := strconv.ParseBool(late)
	if err != nil {
		return nil, err
	}
	extensionValue, err := strconv.ParseBool(extension)
	if err != nil {
		return nil, err
	}

	if lateValue {
		if !user.Repeat {
			return nil, errors.New("cannot make a late booking without repeat status")
		}
		extensionValue = false
	}

	if lateValue {
		calculatedDays += lateReturnIncrease
	} else if extensionValue {
		calculatedDays += extensionIncrease
	}

	if calculatedDays < 0.5 || (calculatedDays > 14 && lateValue) || (calculatedDays > 14.1 && lateValue) {
		return nil, errors.New("booking duration out of bounds")
	}

	daysValue, err := strconv.ParseFloat(days, 64)
	if err != nil {
		return nil, err
	}

	if calculatedDays != daysValue {
		return nil, errors.New("days param provided doesnt match date range given")
	}

	// Check if extension or lateBooking is allowed
	dayAfterBooking := endTime.Add(time.Hour * 24).Format("2006-01-02")
	nextDayBooked, err := db.BookingHasOverlap(dayAfterBooking, dayAfterBooking, carID)
	if err != nil {
		return nil, err
	}
	if nextDayBooked && (lateValue || extensionValue) {
		return nil, errors.New("no extension allowed on this booking")
	}

	car, err := db.GetCar(carID)
	if err != nil {
		return nil, err
	}
	if car == nil {
		return nil, errors.New("problem retrieving car")
	}
	price := car.Cost * daysValue

	startString := startTime.Format("2006-01-02")
	endString := endTime.Format("2006-01-02")

	if lateValue || extensionValue {
		finishTime := finishTime.Add(time.Hour * 24)
		finishString = finishTime.Format("2006-01-02")
	} else {
		finishString = finishTime.Format("2006-01-02")
	}

	overlap, err := db.BookingHasOverlap(startString, endString, carID)
	if err != nil {
		return nil, err
	}
	if overlap {
		return nil, errors.New("booking has overlap")
	}

	bookingID, err := db.CreateBooking(car.ID,
		user.ID,
		startString,
		endString,
		finishString,
		price,
		lateValue, extensionValue, calculatedDays)
	if err != nil {
		return nil, err
	}

	_, err = db.InsertBookingStatus(bookingID, 1, 0, "")
	if err != nil {
		return nil, err
	}

	if len(accessories) != 0 {
		accessory := strings.Split(accessories, ",")
		if len(accessory) != 0 {
			err := db.AddBookingEquipment(bookingID, accessory)
			if err != nil {
				return nil, err
			}
		}
	}

	booking, err := db.GetSingleBooking(bookingID)
	if err != nil {
		return nil, err
	}

	bookingAccesories, err := db.GetBookingAccessories(bookingID)
	if err != nil {
		return nil, err
	}

	booking.CarData = car
	booking.Accessories = bookingAccesories

	return booking, nil
}

func MakePayment(token, bookingID string) error {
	err := session.ValidateToken(token)
	if err != nil {
		return err
	}

	bag, err := session.GetByToken(token)
	if err != nil {
		return err
	}
	user := bag.GetUser()

	bookingIDValid, err := strconv.Atoi(bookingID)
	if err != nil {
		return err
	}

	booking, err := db.GetSingleBooking(bookingIDValid)
	if err != nil {
		return err
	}

	if booking.ProcessID != 1 {
		return errors.New("booking not awaiting payment")
	}

	amountDue := booking.TotalCost - booking.AmountPaid
	if amountDue <= 0 {
		return errors.New("no payment needed")
	}

	err = db.UpdateBookingPayment(booking.ID, user.ID, amountDue)
	if err != nil {
		return err
	}

	_, err = db.InsertBookingStatus(booking.ID, 2, 0, "Made payment of £"+strconv.FormatFloat(amountDue, 'f', 2, 64))
	if err != nil {
		return err
	}

	_, err = db.InsertBookingStatus(booking.ID, 3, 0, "")
	if err != nil {
		return err
	}

	return nil
}

func GetUsersBookings(token string) (map[int][]*data.Booking, error) {
	err := session.ValidateToken(token)
	if err != nil {
		return nil, err
	}

	bag, err := session.GetByToken(token)
	if err != nil {
		return nil, err
	}
	user := bag.GetUser()

	bookings, err := db.GetUsersBookings(user.ID)
	if err != nil {
		return nil, err
	}
	if len(bookings) <= 0 {
		return map[int][]*data.Booking{}, nil
	}

	return organiseBookings(bookings), nil
}

func organiseBookings(bookings []*data.Booking) map[int][]*data.Booking {
	organisedBookings := make(map[int][]*data.Booking)

	for _, value := range bookings {
		if _, exists := organisedBookings[value.ProcessID]; !exists {
			organisedBookings[value.ProcessID] = make([]*data.Booking, 1)
			organisedBookings[value.ProcessID][0] = value
		} else {
			organisedBookings[value.ProcessID] = append(organisedBookings[value.ProcessID], value)
		}
	}

	return organisedBookings
}

func CancelBooking(token, bookingID string) error {
	err := session.ValidateToken(token)
	if err != nil {
		return err
	}

	bag, err := session.GetByToken(token)
	if err != nil {
		return err
	}
	user := bag.GetUser()

	bookingIDValid, err := strconv.Atoi(bookingID)
	if err != nil {
		return err
	}

	booking, err := db.GetSingleBooking(bookingIDValid)
	if err != nil {
		return err
	}

	if user.ID != booking.UserID {
		return errors.New("this booking does not belong to this user")
	}

	if booking.ProcessID == 10 {
		return errors.New("booking already canceled")
	}

	_, err = db.InsertBookingStatus(booking.ID, 10, 0, "user canceled booking")
	if err != nil {
		return err
	}

	return nil
}

func EditBooking(token, bookingID, remove, add, lateReturn, extension string) (int, error) {
	err := session.ValidateToken(token)
	if err != nil {
		return -1, err
	}

	bag, err := session.GetByToken(token)
	if err != nil {
		return -1, err
	}
	user := bag.GetUser()

	bookingIDValid, err := strconv.Atoi(bookingID)
	if err != nil {
		return -1, err
	}

	booking, err := db.GetSingleBooking(bookingIDValid)
	if err != nil {
		return -1, err
	}

	if user.ID != booking.UserID {
		return -1, errors.New("this booking does not belong to this user")
	}

	if booking.ProcessID == 10 {
		return -1, errors.New("booking already canceled")
	}

	lateReturnValue, err := strconv.ParseBool(lateReturn)
	if err != nil {
		return -1, err
	}
	extensionValue, err := strconv.ParseBool(extension)
	if err != nil {
		return -1, err
	}

	if lateReturnValue {
		extensionValue = false
	}

	if lateReturnValue != booking.LateReturn {

	} else if extensionValue != booking.Extension {

	}
	// WIP
	return 0, nil
}
