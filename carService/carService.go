package carService

import (
	"carHiringWebsite/data"
	"carHiringWebsite/db"
)

func GetCars() ([]*data.Car, error) {

	cars, err := db.GetCars()
	if err != nil {
		return nil, err
	}

	return cars, nil
}
