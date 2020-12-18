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

func Create(token, start, end, carID, late, accessories, days string) (*data.Booking, error) {
	var lateValue int

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

	if ((endTime.Sub(startTime).Hours() / 24) + 0.5) != daysValue {
		return nil, errors.New("days param provided doesnt match date range given")
	}

	car, err := db.GetCar(carID)
	if err != nil && car != nil {
		return nil, err
	}

	price := car.Cost * int(daysValue)

	bookingID, err := db.CreateBooking(car.ID,
		user.ID,
		startTime.Format("2006-01-02"),
		endTime.Format("2006-01-02"),
		price,
		lateValue)
	if err != nil || bookingID != 0 {
		return nil, err
	}

	fmt.Println(bookingID)

	return &data.Booking{ID: bookingID}, nil
}
