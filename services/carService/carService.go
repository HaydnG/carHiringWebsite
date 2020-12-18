package carService

import (
	"carHiringWebsite/data"
	"carHiringWebsite/db"
	"errors"
	"strconv"
	"time"
)

func GetAllCars() ([]*data.Car, error) {

	cars, err := db.GetAllCars()
	if err != nil {
		return nil, err
	}

	return cars, nil
}

func GetCar(id string) (*data.Car, error) {

	cars, err := db.GetCar(id)
	if err != nil {
		return nil, err
	}

	return cars, nil
}

func GetCarAccessories(start, end string) ([]*data.Accessory, error) {

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

	accessories, err := db.GetCarAccessories(startTime.Format("2006-01-02"),
		endTime.Format("2006-01-02"))
	if err != nil {
		return nil, err
	}

	return accessories, nil
}

func GetCarBookings(start, end, carID string) ([]*data.OutputTimeRange, error) {
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

	timeRanges, err := db.GetCarBookings(startTime.Format("2006-01-02"),
		endTime.Format("2006-01-02"), carID)
	if err != nil {
		return nil, err
	}

	outputTimeRanges := data.ConvertTimeRangeSlice(timeRanges)

	return outputTimeRanges, nil
}
