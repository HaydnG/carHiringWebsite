package carService

import (
	"carHiringWebsite/data"
	"carHiringWebsite/db"
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
