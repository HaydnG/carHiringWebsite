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

func Create(token, start, end, carID, late, extension, accessories, days string) (*data.Booking, error) {
	lateValue := 0
	extensionValue := 0

	if !session.ValidateToken(token) {
		return nil, errors.New("invalid token")
	}

	bag, activeSession := session.GetByToken(token)
	if bag == nil || !activeSession {
		return nil, errors.New("inactive session")
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

	calculatedDays := (endTime.Sub(startTime).Hours() / 24) + 0.5

	if extension == "true" {
		extensionValue = 1
	} else if extension == "false" {
		extensionValue = 0
	} else {
		return nil, errors.New("invalid extension param")
	}

	if late == "true" {
		if !user.Repeat {
			return nil, errors.New("cannot make a late booking without repeat status")
		}
		lateValue = 1
		extensionValue = 0
	} else if late == "false" {
		lateValue = 0
	} else {
		return nil, errors.New("invalid late param")
	}

	if lateValue == 1 {
		calculatedDays += 0.6
	} else if extensionValue == 1 {
		calculatedDays += 0.5
	}

	if calculatedDays < 0.5 || (calculatedDays > 14 && lateValue == 0) || (calculatedDays > 14.1 && lateValue == 1) {
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
	if nextDayBooked && (lateValue == 1 || extensionValue == 1) {
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
	if !session.ValidateToken(token) {
		return errors.New("invalid token")
	}

	bag, activeSession := session.GetByToken(token)
	if bag == nil || !activeSession {
		return errors.New("inactive session")
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

	amountDue := booking.TotalCost - booking.AmountPaid
	if amountDue <= 0 {
		return errors.New("no payment needed")
	}

	err = db.UpdateBookingPayment(booking.ID, user.ID, amountDue)
	if err != nil {
		return err
	}

	_, err = db.InsertBookingStatus(booking.ID, 2, 0, "")
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
	if !session.ValidateToken(token) {
		return nil, errors.New("invalid token")
	}

	bag, activeSession := session.GetByToken(token)
	if bag == nil || !activeSession {
		return nil, errors.New("inactive session")
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
