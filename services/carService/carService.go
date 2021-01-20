package carService

import (
	"carHiringWebsite/data"
	"carHiringWebsite/db"
	"errors"
	"strconv"
	"time"
)

func GetAllCars(fuelTypes, gearTypes, carTypes, carSizes, colours, search string) ([]*data.Car, error) {

	cars, err := db.GetAllCars(fuelTypes, gearTypes, carTypes, carSizes, colours, search)
	if err != nil {
		return nil, err
	}

	return cars, nil
}

func GetCar(id string) (*data.Car, error) {

	car, err := db.GetCar(id)
	if err != nil {
		return nil, err
	}
	if car.Disabled {
		return nil, errors.New("car disabled")
	}

	return car, nil
}

func GetCarAttributes() (map[string][]*data.CarAttribute, error) {

	attributes, err := db.GetCarAttributes()
	if err != nil {
		return nil, err
	}
	return attributes, nil
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

	carIDValid, err := strconv.Atoi(carID)
	if err != nil {
		return nil, err
	}

	timeRanges, err := db.GetCarBookings(startTime.Format("2006-01-02"),
		endTime.Format("2006-01-02"), carIDValid)
	if err != nil {
		return nil, err
	}

	outputTimeRanges := data.ConvertTimeRangeSlice(timeRanges)

	return outputTimeRanges, nil
}
