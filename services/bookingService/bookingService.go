package bookingService

import (
	"carHiringWebsite/data"
	"carHiringWebsite/db"
	"carHiringWebsite/session"
	"errors"
	"fmt"
	"strconv"
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

	if late == "true" {
		if !user.Repeat {
			return nil, errors.New("cannot make a late booking without repeat status")
		}
		lateValue = 1
	} else if late == "false" {
		lateValue = 0
	} else {
		return nil, errors.New("invalid late param")
	}

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

	daysValue, err := strconv.ParseFloat(days, 64)
	if err != nil {
		return nil, err
	}

	calculatedDays := (endTime.Sub(startTime).Hours() / 24) + 0.5
	if extension == "true" {
		extensionValue = 1
		calculatedDays += 0.5
	} else if extension == "false" {
		extensionValue = 0
	} else {
		return nil, errors.New("invalid extension param")
	}

	if calculatedDays != daysValue {
		return nil, errors.New("days param provided doesnt match date range given")
	}

	car, err := db.GetCar(carID)
	if err != nil {
		return nil, err
	}
	if car == nil {
		return nil, errors.New("problem retrieving car")
	}
	price := car.Cost * int(daysValue)

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
		lateValue, extensionValue)
	if err != nil || bookingID != 0 {
		return nil, err
	}

	fmt.Println(bookingID)

	return &data.Booking{ID: bookingID}, nil
}
