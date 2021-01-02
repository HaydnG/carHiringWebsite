package adminService

import (
	"carHiringWebsite/data"
	"carHiringWebsite/db"
	"carHiringWebsite/services/bookingService"
	"carHiringWebsite/services/userService"
	"carHiringWebsite/session"
	"errors"
	"strconv"
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

	user, err = db.SelectUserByID(adminBooking.Booking.UserID)
	if err != nil {
		return nil, err
	}
	adminBooking.User = data.NewOutputUser(user)

	return adminBooking, nil
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

func ProgressBooking(token, bookingID string) error {

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

	if booking.ProcessID < bookingService.AwaitingConfirmation {
		return errors.New("booking not ready")
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

	nextID := 0
	if booking.ProcessID == bookingService.AwaitingConfirmation {
		nextID = bookingService.BookingConfirmed
	} else if booking.ProcessID == bookingService.BookingConfirmed {
		nextID = bookingService.CollectedBooking
	} else if booking.ProcessID == bookingService.CollectedBooking {
		nextID = bookingService.ReturnedBooking
	} else if booking.ProcessID == bookingService.ReturnedBooking {
		nextID = bookingService.CompletedBooking
	}

	if nextID == 0 {
		return errors.New("booking not in correct state")
	}

	_, err = db.InsertBookingStatus(bookID, nextID, user.ID, 1, "admin progressed booking")
	if err != nil {
		return err
	}

	return nil
}
