package adminService

import (
	"carHiringWebsite/data"
	"carHiringWebsite/db"
	"carHiringWebsite/services/bookingService"
	"carHiringWebsite/services/userService"
	"carHiringWebsite/session"
	"encoding/base64"
	"errors"
	"image/jpeg"
	"io"
	"os"
	"strconv"
	"strings"
)

func GetBookingStatuses(token string) ([]*data.BookingStatusType, error) {

	user, err := userService.GetUserFromSession(token)
	if err != nil {
		return nil, err
	}

	if !user.Admin {
		return nil, errors.New("user is not admin")
	}

	statuses, err := db.GetBookingStatuses()
	if err != nil {
		return nil, err
	}

	return statuses, nil
}

func GetBookingStats(token string) ([]*data.BookingStat, error) {

	user, err := userService.GetUserFromSession(token)
	if err != nil {
		return nil, err
	}

	if !user.Admin {
		return nil, errors.New("user is not admin")
	}

	stats, err := db.GetBookingStats()
	if err != nil {
		return nil, err
	}

	return stats, nil
}

func GetUserStats(token string) (*data.UserStat, error) {

	user, err := userService.GetUserFromSession(token)
	if err != nil {
		return nil, err
	}

	if !user.Admin {
		return nil, errors.New("user is not admin")
	}

	stats, err := db.GetUserStats()
	if err != nil {
		return nil, err
	}

	stats.ActiveUsers = session.CountSesssions()

	return stats, nil
}

func GetUsers(token string, userSearch string) ([]*data.OutputUser, error) {

	user, err := userService.GetUserFromSession(token)
	if err != nil {
		return nil, err
	}

	if !user.Admin {
		return nil, errors.New("user is not admin")
	}

	users, err := db.GetUsers(userSearch)
	if err != nil {
		return nil, err
	}

	return users, nil
}

func GetCarStats(token string) (*data.CarStat, error) {

	user, err := userService.GetUserFromSession(token)
	if err != nil {
		return nil, err
	}

	if !user.Admin {
		return nil, errors.New("user is not admin")
	}

	stats, err := db.GetCarStats()
	if err != nil {
		return nil, err
	}

	return stats, nil
}

func GetBooking(token, bookingID string) (*data.AdminBooking, error) {

	user, err := userService.GetUserFromSession(token)
	if err != nil {
		return nil, err
	}

	if !user.Admin {
		return nil, errors.New("user is not admin")
	}

	bookingIDValid, err := strconv.Atoi(bookingID)
	if err != nil {
		return nil, err
	}

	adminBooking := &data.AdminBooking{}

	adminBooking.Booking, err = db.GetSingleBooking(bookingIDValid)
	if err != nil {
		return nil, err
	}

	adminBooking.Booking.CarData, err = db.GetCar(strconv.Itoa(adminBooking.Booking.CarID))
	if err != nil {
		return nil, err
	}

	adminBooking.Booking.Accessories, err = db.GetBookingAccessories(bookingIDValid)
	if err != nil {
		return nil, err
	}

	adminBooking.Booking.ActiveStatuses, err = db.GetActiveBookingStatuses(bookingIDValid)
	if err != nil {
		return nil, err
	}

	user, err = db.SelectUserByID(adminBooking.Booking.UserID)
	if err != nil {
		return nil, err
	}
	adminBooking.User = data.NewOutputUser(user)

	return adminBooking, nil
}

func GetUser(token, userID string) (*data.UserBundle, error) {

	user, err := userService.GetUserFromSession(token)
	if err != nil {
		return nil, err
	}

	if !user.Admin {
		return nil, errors.New("user is not admin")
	}

	userIDValid, err := strconv.Atoi(userID)
	if err != nil {
		return nil, err
	}

	userBundle := &data.UserBundle{}

	user, err = db.SelectUserByID(userIDValid)
	if err != nil {
		return nil, err
	}

	userBundle.User = data.NewOutputUser(user)

	userBundle.Bookings, err = db.GetAdminUsersBookings(userIDValid)
	if err != nil {
		return nil, err
	}

	return userBundle, nil
}

func GetAccessoryStats(token string) ([]*data.AccessoryStat, error) {

	user, err := userService.GetUserFromSession(token)
	if err != nil {
		return nil, err
	}

	if !user.Admin {
		return nil, errors.New("user is not admin")
	}

	stats, err := db.GetAccessoryStats()
	if err != nil {
		return nil, err
	}

	return stats, nil
}

func SetUser(token, userID, mode, value string) error {

	user, err := userService.GetUserFromSession(token)
	if err != nil {
		return err
	}

	if !user.Admin {
		return errors.New("user is not admin")
	}

	valueBool, err := strconv.ParseBool(value)
	if err != nil {
		return err
	}
	modeValue, err := strconv.Atoi(mode)
	if err != nil {
		return err
	}
	userIDValue, err := strconv.Atoi(userID)
	if err != nil {
		return err
	}

	if userIDValue == user.ID {
		return errors.New("admin cannot demote themself")
	}

	switch modeValue {
	case 0:
		err = db.SetDisableUser(userIDValue, valueBool)
		if err != nil {
			return err
		}
		break
	case 1:
		err = db.SetBlackListUser(userIDValue, valueBool)
		if err != nil {
			return err
		}
		break
	case 2:
		err = db.SetAdminUser(userIDValue, valueBool)
		if err != nil {
			return err
		}
		break
	}

	return nil
}

func CreateCar(token, fuelType, gearType, carType, size, colour, seats, price, disabled, over25, description string, body io.Reader) error {

	user, err := userService.GetUserFromSession(token)
	if err != nil {
		return err
	}

	if !user.Admin {
		return errors.New("user is not admin")
	}

	disabledBool, err := strconv.ParseBool(disabled)
	if err != nil {
		return err
	}

	over25Bool, err := strconv.ParseBool(over25)
	if err != nil {
		return err
	}

	fuelTypeID, err := strconv.Atoi(fuelType)
	if err != nil {
		return err
	}
	gearTypeID, err := strconv.Atoi(gearType)
	if err != nil {
		return err
	}
	carTypeID, err := strconv.Atoi(carType)
	if err != nil {
		return err
	}
	sizeID, err := strconv.Atoi(size)
	if err != nil {
		return err
	}
	colourID, err := strconv.Atoi(colour)
	if err != nil {
		return err
	}
	seatsNumber, err := strconv.Atoi(seats)
	if err != nil {
		return err
	}
	priceNumber, err := strconv.Atoi(price)
	if err != nil {
		return err
	}

	fileName := strings.ReplaceAll(description, " ", "_")
	fileName += "_" + colour

	file, err := os.Create("cars/" + fileName + ".jpg")
	if err != nil {
		return err
	}
	defer func() {
		if file != nil {
			file.Close()
			if err != nil {
				os.Remove("cars/" + fileName + ".jpg")
			}
		}
	}()

	decoder := base64.NewDecoder(base64.StdEncoding.WithPadding(base64.StdPadding), body)

	image, err := jpeg.Decode(decoder)
	if err != nil {
		return err
	}

	err = jpeg.Encode(file, image, nil)
	if err != nil {
		return err
	}

	_, err = db.CreateCar(fuelTypeID, gearTypeID, carTypeID, sizeID, colourID, seatsNumber, priceNumber, disabledBool, over25Bool, fileName, description)

	return nil
}

func UpdateCar(token, carID, fuelType, gearType, carType, size, colour, seats, price, disabled, description, over25 string, body io.Reader) error {

	user, err := userService.GetUserFromSession(token)
	if err != nil {
		return err
	}

	if !user.Admin {
		return errors.New("user is not admin")
	}

	disabledBool, err := strconv.ParseBool(disabled)
	if err != nil {
		return err
	}
	over25Bool, err := strconv.ParseBool(over25)
	if err != nil {
		return err
	}

	fuelTypeID, err := strconv.Atoi(fuelType)
	if err != nil {
		return err
	}
	gearTypeID, err := strconv.Atoi(gearType)
	if err != nil {
		return err
	}
	carTypeID, err := strconv.Atoi(carType)
	if err != nil {
		return err
	}
	sizeID, err := strconv.Atoi(size)
	if err != nil {
		return err
	}
	colourID, err := strconv.Atoi(colour)
	if err != nil {
		return err
	}
	seatsNumber, err := strconv.Atoi(seats)
	if err != nil {
		return err
	}
	priceNumber, err := strconv.Atoi(price)
	if err != nil {
		return err
	}

	car, err := db.GetCar(carID)
	if err != nil {
		return err
	}

	fileName := ""
	if body != nil {
		fileName = strings.ReplaceAll(description, " ", "_")
		fileName += "_" + colour

		file, err := os.Create("cars/" + fileName + ".jpg")
		if err != nil {
			return err
		}
		defer func() {
			if file != nil {
				file.Close()
				if err != nil {
					os.Remove("cars/" + fileName + ".jpg")
				}
			}
		}()

		decoder := base64.NewDecoder(base64.StdEncoding.WithPadding(base64.StdPadding), body)

		image, err := jpeg.Decode(decoder)
		if err != nil {
			return err
		}

		err = jpeg.Encode(file, image, nil)
		if err != nil {
			return err
		}
	} else {
		fileName = car.Image
	}

	_, err = db.UpdateCar(fuelTypeID, gearTypeID, carTypeID, sizeID, colourID, seatsNumber, priceNumber, disabledBool, over25Bool, fileName, description, car.ID)

	return nil
}

func GetQueryingRefundBookings(token string) ([]*data.BookingColumn, error) {
	user, err := userService.GetUserFromSession(token)
	if err != nil {
		return nil, err
	}

	if !user.Admin {
		return nil, errors.New("user is not admin")
	}

	bookings, err := db.GetQueryingRefundBookings()
	if err != nil {
		return nil, err
	}

	return bookings, nil
}

func GetAwaitingBookings(token, status, limit string) ([]*data.BookingColumn, error) {

	user, err := userService.GetUserFromSession(token)
	if err != nil {
		return nil, err
	}

	if !user.Admin {
		return nil, errors.New("user is not admin")
	}

	limitNum, err := strconv.Atoi(limit)
	if err != nil {
		return nil, err
	}
	if limitNum < 0 || limitNum > 20 {
		return nil, errors.New("limit out of bound")
	}

	bookings, err := db.GetUpcomingBookings(status, limitNum)
	if err != nil {
		return nil, err
	}

	return bookings, nil
}

func GetSearchedBookings(token, userSearch, bookingSearch, statusFilter string) ([]*data.BookingColumn, error) {

	user, err := userService.GetUserFromSession(token)
	if err != nil {
		return nil, err
	}

	if !user.Admin {
		return nil, errors.New("user is not admin")
	}

	bookings, err := db.GetSearchedBookings(userSearch, bookingSearch, statusFilter)
	if err != nil {
		return nil, err
	}

	return bookings, nil
}

func GetCars(token, fuelTypes, gearTypes, carTypes, carSizes, colours, search string) ([]*data.Car, error) {

	user, err := userService.GetUserFromSession(token)
	if err != nil {
		return nil, err
	}

	if !user.Admin {
		return nil, errors.New("user is not admin")
	}

	cars, err := db.AdminGetCars(fuelTypes, gearTypes, carTypes, carSizes, colours, search)
	if err != nil {
		return nil, err
	}

	return cars, nil
}

func ProgressBooking(token, bookingID, failed string) error {

	user, err := userService.GetUserFromSession(token)
	if err != nil {
		return err
	}

	if !user.Admin {
		return errors.New("user is not admin")
	}

	failedValue, err := strconv.ParseBool(failed)
	if err != nil {
		return err
	}

	bookID, err := strconv.Atoi(bookingID)
	if err != nil {
		return err
	}

	booking, err := db.GetSingleBooking(bookID)
	if err != nil {
		return err
	}

	if booking.ProcessID < bookingService.AwaitingConfirmation {
		return errors.New("booking not ready")
	}

	failedMsg := ""
	blackListUser := false
	nextID := 0
	if booking.ProcessID == bookingService.AwaitingConfirmation {
		_, err := db.InsertBookingStatus(bookID, bookingService.ABICheck, 1, 1, 0.0, "Awaiting ABI Check")
		if err != nil {
			return err
		}
		_, err = db.InsertBookingStatus(bookID, bookingService.DVLACheck, 1, 1, 0.0, "Awaiting DVLA Check")
		if err != nil {
			return err
		}
		nextID = bookingService.BookingConfirmed
	} else if booking.ProcessID == bookingService.BookingConfirmed {
		status, err := db.GetBookingProcessStatus(bookID, bookingService.DVLACheck)
		if err != nil {
			return err
		}
		if status != nil && status.Active {
			return errors.New("booking not ready")
		}

		status, err = db.GetBookingProcessStatus(bookID, bookingService.ABICheck)
		if err != nil {
			return err
		}
		if status != nil && status.Active {
			return errors.New("booking not ready")
		}

		nextID = bookingService.CollectedBooking
		if failedValue {
			blackListUser = true
			failedMsg = "user failed to collect booking"
		}
	} else if booking.ProcessID == bookingService.CollectedBooking {
		nextID = bookingService.ReturnedBooking
		if failedValue {
			blackListUser = true
			failedMsg = "user failed to return booking"
		}
	} else if booking.ProcessID == bookingService.ReturnedBooking {
		nextID = bookingService.CompletedBooking

		err := db.SetRepeatUser(booking.UserID)
		if err != nil {
			return err
		}
	}

	status, err := db.GetBookingProcessStatus(bookID, booking.ProcessID)
	if err != nil {
		return err
	}
	if status != nil && status.Active {
		err := db.SetBookingStatus(status.ID, false)
		if err != nil {
			return err
		}
	}

	if nextID == 0 {
		return errors.New("booking not in correct state")
	}

	if !blackListUser && !failedValue {
		_, err = db.InsertBookingStatus(bookID, nextID, user.ID, 1, 0.0, "admin progressed booking")
		if err != nil {
			return err
		}

	} else {
		failedMsg += " - User will be blackListed"

		err = db.SetBlackListUser(booking.UserID, true)
		if err != nil {
			return err
		}

		err = db.DeactivateBookingStatuses(booking.ID)
		if err != nil {
			return err
		}

		_, err = db.InsertBookingStatus(booking.ID, bookingService.CanceledBooking, user.ID, 1, 0.0, failedMsg)
		if err != nil {
			return err
		}

	}

	return nil
}

func ProcessRefundHandler(token, bookingID, accept, reason string) error {

	acceptBool, err := strconv.ParseBool(accept)
	if err != nil {
		return err
	}

	user, err := userService.GetUserFromSession(token)
	if err != nil {
		return err
	}

	if !user.Admin {
		return errors.New("user is not admin")
	}

	bookID, err := strconv.Atoi(bookingID)
	if err != nil {
		return err
	}

	booking, err := db.GetSingleBooking(bookID)
	if err != nil {
		return err
	}

	if booking.ProcessID != bookingService.CanceledBooking || !booking.AwaitingExtraPayment {
		return errors.New("booking not ready")
	}

	status, err := db.GetBookingProcessStatus(booking.ID, bookingService.QueryingRefund)
	if err != nil {
		return err
	}
	if status != nil && status.Active {
		err := db.SetBookingStatus(status.ID, false)
		if err != nil {
			return err
		}
	}

	if acceptBool {
		err = db.UpdateBookingPayment(booking.ID, booking.UserID, 0)
		if err != nil {
			return err
		}
		message := "Refund of £" + strconv.FormatFloat(booking.AmountPaid, 'f', 2, 64) + " Given"
		if reason != "" {
			message += " - " + reason
		}

		_, err = db.InsertBookingStatus(booking.ID, bookingService.RefundIssued, user.ID, 0, booking.AmountPaid, message)

		if err != nil {
			return err
		}
	} else {
		message := "Refund Rejected"
		if reason != "" {
			message += " - " + reason
		}
		_, err = db.InsertBookingStatus(booking.ID, bookingService.RefundRejected, user.ID, 0, booking.AmountPaid, message)
		if err != nil {
			return err
		}
	}

	return nil
}

func CreateUser(token, email, password, firstname, names, dobString string) (bool, *data.OutputUser, error) {
	user, err := userService.GetUserFromSession(token)
	if err != nil {
		return false, nil, err
	}

	if !user.Admin {
		return false, nil, errors.New("user is not admin")
	}

	return userService.CreateUser(email, password, firstname, names, dobString)
}

func ProcessExtraPayment(token, bookingID string) error {

	user, err := userService.GetUserFromSession(token)
	if err != nil {
		return err
	}

	if !user.Admin {
		return errors.New("user is not admin")
	}

	bookID, err := strconv.Atoi(bookingID)
	if err != nil {
		return err
	}

	booking, err := db.GetSingleBooking(bookID)
	if err != nil {
		return err
	}

	if booking.ProcessID == bookingService.CanceledBooking || !booking.AwaitingExtraPayment || booking.ProcessID < bookingService.BookingConfirmed {
		return errors.New("booking not ready")
	}
	amount := .0
	message := "User "
	if booking.IsRefund {
		amount = booking.AmountPaid - booking.TotalCost
		message += "Refunded £" + strconv.FormatFloat(amount, 'f', 2, 64)
	} else {
		amount = booking.TotalCost - booking.AmountPaid
		message += "Payed £" + strconv.FormatFloat(amount, 'f', 2, 64)
	}
	if booking.ProcessID <= bookingService.BookingConfirmed {
		message += " on Collection"
	} else {
		message += " on Return"
	}

	status, err := db.GetBookingProcessStatus(booking.ID, bookingService.EditAwaitingPayment)
	if err != nil {
		return err
	}
	if status != nil && status.Active {
		err := db.SetBookingStatus(status.ID, false)
		if err != nil {
			return err
		}
	}

	err = db.UpdateBookingPayment(booking.ID, booking.UserID, booking.TotalCost)
	if err != nil {
		return err
	}

	_, err = db.InsertBookingStatus(booking.ID, bookingService.EditPaymentAccepted, user.ID, 0, booking.TotalCost-booking.AmountPaid, message)
	if err != nil {
		return err
	}

	return nil
}
