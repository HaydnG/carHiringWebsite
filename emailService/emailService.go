package emailService

import (
	"carHiringWebsite/data"
	"fmt"
	"os"
	"strconv"
	"time"
)

func SendEmail(driver *data.Driver) error {
	var err error

	file, err := os.Create("emails/" + driver.LicenseNumber + "_" + strconv.FormatInt(time.Now().Unix(), 10) + ".txt")
	if err != nil {
		return err
	}
	defer file.Close()

	string := fmt.Sprintf("DVLA Offense Alert\n\n"+
		"Company: Banger\n"+
		"LicenseNumber: %s\n"+
		"Name: %s %s\n"+
		"DateTime of Occurence: %s", driver.LicenseNumber, driver.LastName, driver.Names, time.Now().Format("2006-01-02 15:04:05"))

	_, err = file.WriteString(string)
	if err != nil {
		return err
	}

	return nil
}
