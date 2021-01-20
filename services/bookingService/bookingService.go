package bookingService

import (
	"carHiringWebsite/data"
	"carHiringWebsite/db"
	"carHiringWebsite/services/userService"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

const (
	lateReturnIncrease = 0.6
	fullDayIncrease    = 0.5
)
const (
	AwaitingPayment = iota + 1
	PaymentAccepted
	AwaitingConfirmation
	BookingConfirmed
	BookingEdited
	EditAwaitingPayment
	EditPaymentAccepted
	QueryingRefund
	RefundRejected
	RefundIssued
	CanceledBooking
	CollectedBooking
	ReturnedBooking
	CompletedBooking
	ExtendedBooking
	ExtensionAwaitingPayment
	ExtensionPaymentAccepted
)

func Create(token, start, end, carID, late, fullDay, accessories, days string) (*data.Booking, error) {
	var finishString string

	user, err := userService.GetUserFromSession(token)
	if err != nil {
		return nil, err
	}

	dbUser, err := db.SelectUserByID(user.ID)
	if err != nil {
		return nil, err
	}

	if dbUser.Blacklisted {
		return nil, errors.New("user is blackListed")
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
	finishTime := time.Unix(endNum, 0)

	calculatedDays := (endTime.Sub(startTime).Hours() / 24) + 0.5

	lateValue, err := strconv.ParseBool(late)
	if err != nil {
		return nil, err
	}
	fullDayValue, err := strconv.ParseBool(fullDay)
	if err != nil {
		return nil, err
	}

	if lateValue {
		if !user.Repeat {
			return nil, errors.New("cannot make a late booking without repeat status")
		}
		fullDayValue = false
	}

	if lateValue {
		calculatedDays += lateReturnIncrease
	} else if fullDayValue {
		calculatedDays += fullDayIncrease
	}

	if calculatedDays < 0.5 || (calculatedDays > 14 && !lateValue) || (calculatedDays > 14.1 && lateValue) {
		return nil, errors.New("booking duration out of bounds")
	}

	daysValue, err := strconv.ParseFloat(days, 64)
	if err != nil {
		return nil, err
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
	if car.Disabled {
		return nil, errors.New("car disabled")
	}

	if car.Over25 && userService.CalculateAge(user.DOB.Unix()) < 25 {
		return nil, errors.New("user does not meet age requirements")
	}

	price := car.Cost * daysValue

	startString := startTime.Format("2006-01-02")
	endString := endTime.Format("2006-01-02")

	if lateValue || fullDayValue {
		finishTime = finishTime.Add(time.Hour * 24)
	}
	finishString = finishTime.Format("2006-01-02")

	// Check if extension or lateBooking is allowed
	nextDayBooked, err := db.BookingHasOverlap(finishString, finishString, car.ID)
	if err != nil {
		return nil, err
	}
	if nextDayBooked && (lateValue || fullDayValue) {
		return nil, errors.New("no extension allowed on this booking")
	}

	overlap, err := db.BookingHasOverlap(startString, endString, car.ID)
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
		finishString,
		price,
		lateValue, fullDayValue, calculatedDays)
	if err != nil {
		return nil, err
	}

	_, err = db.InsertBookingStatus(bookingID, AwaitingPayment, 0, 1, 0.0, "")
	if err != nil {
		return nil, err
	}

	if len(accessories) != 0 {
		accessory := strings.Split(accessories, ",")
		if len(accessory) != 0 && validateEquipmentList(accessory, nil) {
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
	user, err := userService.GetUserFromSession(token)
	if err != nil {
		return err
	}

	bookingIDValid, err := strconv.Atoi(bookingID)
	if err != nil {
		return err
	}

	booking, err := db.GetSingleBooking(bookingIDValid)
	if err != nil {
		return err
	}

	if booking.UserID != user.ID {
		return errors.New("this booking does not belong to this user")
	}

	if booking.ProcessID != AwaitingPayment {
		return errors.New("booking not awaiting payment")
	}

	amountDue := booking.TotalCost - booking.AmountPaid
	if amountDue <= 0 {
		return errors.New("no payment needed")
	}

	status, err := db.GetBookingProcessStatus(booking.ID, AwaitingPayment)
	if err != nil {
		return err
	} else if status == nil || !status.Active {
		return errors.New("booking not awaiting payment")
	}

	err = db.UpdateBookingPayment(booking.ID, user.ID, amountDue)
	if err != nil {
		return err
	}

	_, err = db.InsertBookingStatus(booking.ID, PaymentAccepted, 0, 0, amountDue, "Made payment of £"+strconv.FormatFloat(amountDue, 'f', 2, 64))
	if err != nil {
		return err
	}

	// disable awaiting payment status
	err = db.SetBookingStatus(status.ID, false)
	if err != nil {
		return err
	}

	_, err = db.InsertBookingStatus(booking.ID, AwaitingConfirmation, 0, 1, 0.0, "")
	if err != nil {
		return err
	}

	return nil
}

func MakeExtensionPayment(token, bookingID string) error {
	user, err := userService.GetUserFromSession(token)
	if err != nil {
		return err
	}

	bookingIDValid, err := strconv.Atoi(bookingID)
	if err != nil {
		return err
	}

	booking, err := db.GetSingleBooking(bookingIDValid)
	if err != nil {
		return err
	}

	if booking.UserID != user.ID {
		return errors.New("this booking does not belong to this user")
	}

	status, err := db.GetBookingProcessStatus(booking.ID, ExtensionAwaitingPayment)
	if err != nil {
		return err
	}
	if status == nil {
		return errors.New("booking not awaiting payment")
	}
	if status != nil && !status.Active {
		return errors.New("booking not awaiting payment")
	}

	amountDue := booking.TotalCost - booking.AmountPaid
	if amountDue <= 0 {
		return errors.New("no payment needed")
	}

	err = db.UpdateBookingPayment(booking.ID, user.ID, booking.TotalCost)
	if err != nil {
		return err
	}

	_, err = db.InsertBookingStatus(booking.ID, ExtensionPaymentAccepted, 0, 0, amountDue, "Made payment of £"+strconv.FormatFloat(amountDue, 'f', 2, 64))
	if err != nil {
		return err
	}

	// disable awaiting payment status
	err = db.SetBookingStatus(status.ID, false)
	if err != nil {
		return err
	}

	return nil
}

func GetUsersBookings(token string) (map[int][]*data.Booking, error) {
	user, err := userService.GetUserFromSession(token)
	if err != nil {
		return nil, err
	}

	bookings, err := db.GetUsersBookings(user.ID)
	if err != nil {
		return nil, err
	}
	if len(bookings) <= 0 {
		return map[int][]*data.Booking{}, nil
	}

	return organiseBookings(bookings)
}

func CountExtensionDays(token string, bookingID string) (*data.ExtensionResponse, error) {
	user, err := userService.GetUserFromSession(token)
	if err != nil {
		return nil, err
	}

	bookingIDValid, err := strconv.Atoi(bookingID)
	if err != nil {
		return nil, err
	}

	booking, err := db.GetSingleBooking(bookingIDValid)
	if err != nil {
		return nil, err
	}

	if booking.UserID != user.ID && !user.Admin {
		return nil, errors.New("booking does not belong to user")
	}

	response, err := db.CountExtensionDays(booking.End.Add(time.Hour*24).Format("2006-01-02"),
		booking.End.Add((time.Hour*24)*14).Format("2006-01-02"),
		booking.CarID, bookingIDValid)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func organiseBookings(bookings []*data.Booking) (map[int][]*data.Booking, error) {
	var err error
	organisedBookings := make(map[int][]*data.Booking)

	for _, value := range bookings {
		value.ActiveStatuses, err = db.GetActiveBookingStatuses(value.ID)
		if err != nil {
			return nil, err
		}
		if _, exists := organisedBookings[value.ProcessID]; !exists {
			organisedBookings[value.ProcessID] = make([]*data.Booking, 1)
			organisedBookings[value.ProcessID][0] = value
		} else {
			organisedBookings[value.ProcessID] = append(organisedBookings[value.ProcessID], value)
		}
	}

	return organisedBookings, nil
}

func CancelBooking(token, bookingID string) error {
	adminID := 0
	cancelMsg := "User canceled booking"
	user, err := userService.GetUserFromSession(token)
	if err != nil {
		return err
	}

	bookingIDValid, err := strconv.Atoi(bookingID)
	if err != nil {
		return err
	}

	booking, err := db.GetSingleBooking(bookingIDValid)
	if err != nil {
		return err
	}

	if user.ID != booking.UserID && !user.Admin {
		return errors.New("this booking does not belong to this user")
	} else if user.Admin {
		adminID = user.ID
		cancelMsg = "Admin canceled booking"
	}

	if booking.ProcessID == CanceledBooking {
		return errors.New("booking already canceled")
	}
	if booking.ProcessID > BookingConfirmed && !user.Admin {
		return errors.New("booking can only be canceled by an admin after collection")
	}

	err = db.DeactivateBookingStatuses(booking.ID)
	if err != nil {
		return err
	}

	if booking.AmountPaid > 0 {
		_, err = db.InsertBookingStatus(booking.ID, QueryingRefund, adminID, 1, 0, "Automatic refund query requested")
		if err != nil {
			return err
		}

	}

	_, err = db.InsertBookingStatus(booking.ID, CanceledBooking, adminID, 1, 0, cancelMsg)
	if err != nil {
		return err
	}

	return nil
}

func GetHistory(token, bookingID string) ([]*data.BookingStatus, error) {
	user, err := userService.GetUserFromSession(token)
	if err != nil {
		return nil, err
	}

	bookingIDValid, err := strconv.Atoi(bookingID)
	if err != nil {
		return nil, err
	}

	booking, err := db.GetSingleBooking(bookingIDValid)
	if err != nil {
		return nil, err
	}

	if user.ID != booking.UserID && !user.Admin {
		return nil, errors.New("this booking does not belong to this user")
	}

	history, err := db.GetBookingHistory(booking.ID)

	return history, err
}

func ExtendBooking(token, bookingID, lateReturn, fullDay, days string) error {
	adminID := 0

	user, err := userService.GetUserFromSession(token)
	if err != nil {
		return err
	}

	bookingIDValid, err := strconv.Atoi(bookingID)
	if err != nil {
		return err
	}

	booking, err := db.GetSingleBooking(bookingIDValid)
	if err != nil {
		return err
	}

	if user.ID != booking.UserID && !user.Admin {
		return errors.New("this booking does not belong to this user")
	} else if user.Admin {
		adminID = user.ID
	}

	if booking.ProcessID != CollectedBooking {
		return errors.New("booking in an incorrect state")
	}

	status, err := db.GetBookingProcessStatus(booking.ID, ExtensionAwaitingPayment)
	if err != nil {
		return err
	}
	if status != nil && status.Active {
		return errors.New("current extension awaiting payment")
	}

	lateReturnValue, err := strconv.ParseBool(lateReturn)
	if err != nil {
		return err
	}
	fullDayValue, err := strconv.ParseBool(fullDay)
	if err != nil {
		return err
	}

	daysValid, err := strconv.ParseFloat(days, 64)
	if err != nil {
		return err
	}
	if daysValid < 1 || daysValid > 14 {
		return errors.New("days value out of bounds")
	}

	response, err := db.CountExtensionDays(booking.End.Add(time.Hour*24).Format("2006-01-02"),
		booking.End.Add((time.Hour*24)*14).Format("2006-01-02"),
		booking.CarID, bookingIDValid)
	if err != nil {
		return err
	}

	if int(daysValid) > response.Days {
		return errors.New("extension of this amount not allowed")
	}

	needEarlyReturn := false
	if daysValid < 14.0 && int(daysValid) == response.Days {
		needEarlyReturn = true
	} else if daysValid == 14.0 {
		needEarlyReturn, err = db.BookingHasOverlap(booking.End.Add((time.Hour*24)*15).Format("2006-01-02"),
			booking.End.Add((time.Hour*24)*15).Format("2006-01-02"), booking.CarID)
		if err != nil {
			return err
		}

	}

	if needEarlyReturn && (lateReturnValue || fullDayValue) {
		lateReturnValue = false
		fullDayValue = false
	}
	if lateReturnValue {
		fullDayValue = false
	}

	newDaysValue := booking.BookingLength

	if booking.LateReturn {
		newDaysValue += -lateReturnIncrease
	} else if booking.FullDay {
		newDaysValue += -fullDayIncrease
	}

	if lateReturnValue {
		newDaysValue += lateReturnIncrease
	} else if fullDayValue {
		newDaysValue += fullDayIncrease
	}

	newEndDate := booking.End.Add((time.Hour * 24) * time.Duration(daysValid))
	newFinishDateString := newEndDate.Format("2006-01-02")
	if lateReturnValue || fullDayValue {
		newFinishDateString = newEndDate.Add(time.Hour * 24).Format("2006-01-02")
	}
	newDaysValue += daysValid
	CarDailyCost := booking.TotalCost / booking.BookingLength

	newCost := newDaysValue * CarDailyCost
	amountToPay := newCost - booking.AmountPaid

	paymentDesc := fmt.Sprintf("Need to pay £%.2f", amountToPay)
	_, err = db.InsertBookingStatus(booking.ID, ExtensionAwaitingPayment, 0, 1, amountToPay, paymentDesc)
	if err != nil {
		return err
	}

	description := fmt.Sprintf("£%.2f -> £%.2f | Days %.1f -> %.1f | ", booking.TotalCost, newCost, booking.BookingLength, newDaysValue)
	if lateReturnValue != booking.LateReturn {
		description += fmt.Sprintf("LateReturn: %t", lateReturnValue)
	} else if fullDayValue != booking.FullDay {
		description += fmt.Sprintf("Full Day: %t", fullDayValue)
	}

	_, err = db.InsertBookingStatus(booking.ID, ExtendedBooking, adminID, 0, newDaysValue, description)
	if err != nil {
		return err
	}

	err = db.UpdateBooking(booking.ID, newCost, newDaysValue, lateReturnValue, fullDayValue, newEndDate.Format("2006-01-02"), newFinishDateString)
	if err != nil {
		return err
	}

	return err
}

func EditBooking(token, bookingID, remove, add, lateReturn, fullDay string) error {
	adminID := 0
	edited := false
	var description string
	var AddAccessory, RemoveAccessory []string
	var amountDue float64
	var finishString string

	user, err := userService.GetUserFromSession(token)

	bookingIDValid, err := strconv.Atoi(bookingID)
	if err != nil {
		return err
	}

	booking, err := db.GetSingleBooking(bookingIDValid)
	if err != nil {
		return err
	}

	if user.ID != booking.UserID && !user.Admin {
		return errors.New("this booking does not belong to this user")
	} else if user.Admin {
		adminID = user.ID
	}

	if booking.ProcessID > BookingConfirmed {
		return errors.New("booking not in an editable state")
	}

	lateReturnValue, err := strconv.ParseBool(lateReturn)
	if err != nil {
		return err
	}
	fullDayValue, err := strconv.ParseBool(fullDay)
	if err != nil {
		return err
	}

	if lateReturnValue {
		fullDayValue = false
	}

	days := booking.BookingLength
	newCost := booking.TotalCost
	if lateReturnValue != booking.LateReturn || fullDayValue != booking.FullDay {

		if lateReturnValue || fullDayValue {
			newfinishTime := booking.Finish.Add(time.Hour * 24)

			finishString = newfinishTime.Format("2006-01-02")

			// Check if fullday or lateBooking is allowed
			nextDayBooked, err := db.BookingHasOverlap(finishString, finishString, booking.CarID)
			if err != nil {
				return err
			}
			if nextDayBooked {
				return errors.New("no extension allowed on this booking")
			}
		} else if !lateReturnValue && !fullDayValue {
			newfinishTime := booking.Finish.Add(-(time.Hour * 24))
			finishString = newfinishTime.Format("2006-01-02")
		}

		dailyCost := booking.TotalCost / booking.BookingLength

		if booking.LateReturn {
			days = days - lateReturnIncrease
		} else if booking.FullDay {
			days = days - fullDayIncrease
		}

		if lateReturnValue {
			days = days + lateReturnIncrease
		} else if fullDayValue {
			days = days + fullDayIncrease
		}

		newCost = dailyCost * days

		description += fmt.Sprintf("£%.2f -> £%.2f | ", booking.TotalCost, newCost)

		if lateReturnValue != booking.LateReturn {
			description += fmt.Sprintf("LateReturn: %t | ", lateReturnValue)
		} else if fullDayValue != booking.FullDay {
			description += fmt.Sprintf("Full Day: %t | ", fullDayValue)
		}

		err := db.UpdateBooking(booking.ID, newCost, days, lateReturnValue, fullDayValue, booking.End.Format("2006-01-02"), finishString)
		if err != nil {
			return err
		}

		amountDue = newCost - booking.AmountPaid
		edited = true
	}

	if len(add) != 0 {
		AddAccessory = strings.Split(add, ",")
	}
	if len(remove) != 0 {
		RemoveAccessory = strings.Split(remove, ",")
	}

	if validateEquipmentList(AddAccessory, RemoveAccessory) {
		accessories, err := db.GetCarAccessories("0", "0")
		if err != nil {
			return err
		}

		if len(AddAccessory) != 0 {
			names, err := GetAccessoryNames(accessories, AddAccessory)
			if err != nil {
				return err
			}

			err = db.AddBookingEquipment(booking.ID, AddAccessory)
			if err != nil {
				return err
			}
			description += fmt.Sprint("ADD: ")
			for i, v := range names {
				description += fmt.Sprintf("%s", v)
				if i != len(names)-1 {
					description += fmt.Sprint(", ")
				} else {
					description += fmt.Sprint(" | ")
				}
			}
		}
		if len(RemoveAccessory) != 0 {
			names, err := GetAccessoryNames(accessories, RemoveAccessory)
			if err != nil {
				return err
			}

			err = db.RemoveBookingEquipment(booking.ID, RemoveAccessory)
			if err != nil {
				return err
			}
			description += fmt.Sprint("REMOVE: ")
			for i, v := range names {
				description += fmt.Sprintf("%s", v)
				if i != len(names)-1 {
					description += fmt.Sprint(", ")
				} else {
					description += fmt.Sprint(" | ")
				}
			}

		}
		edited = true
	}

	if edited {
		_, err = db.InsertBookingStatus(booking.ID, BookingEdited, adminID, 0, 0.0, description)
		if err != nil {
			return err
		}

		if booking.ProcessID != AwaitingPayment {
			status, err := db.GetBookingProcessStatus(booking.ID, EditAwaitingPayment)
			if err != nil {
				return err
			}

			if status != nil && status.Active {
				err := db.SetBookingStatus(status.ID, false)
				if err != nil {
					return err
				}
			}
			if amountDue != 0 {
				paymentDesc := ""
				if amountDue > 0 {
					paymentDesc = fmt.Sprintf("Need to pay £%.2f on Collection", math.Abs(amountDue))
				} else {
					paymentDesc = fmt.Sprintf("Refund of £%.2f on Collection", math.Abs(amountDue))
				}

				_, err = db.InsertBookingStatus(booking.ID, EditAwaitingPayment, 0, 1, amountDue, paymentDesc)
				if err != nil {
					return err
				}
			}

		}
	}

	return nil
}

func GetAccessoryNames(access []*data.Accessory, ids []string) ([]string, error) {

	if len(ids) <= 0 || len(access) <= 0 {
		return nil, errors.New("GetAccessoryNames list is empty")
	}

	names := make([]string, len(ids))
	count := 0

	for _, v := range ids {
		id, err := strconv.Atoi(v)
		if err != nil {
			return nil, err
		}

		for _, v := range access {
			if v.ID == id {
				names[count] = v.Description
				count++
				break
			}
		}
	}

	return names, nil

}

func validateEquipmentList(add, remove []string) bool {
	var list []string

	if (add == nil || len(add) > 10) && (remove == nil || len(remove) > 10) {
		return false
	}

	if remove != nil && len(remove) > 0 {
		list = append(add, remove...)
	} else {
		list = add
	}

	for i := 0; i < len(list)-1; i++ {
		for j := i + 1; j < len(list); j++ {
			if list[i] == "" || list[j] == "" {
				return false
			}
			if strings.Compare(list[i], list[j]) == 0 {
				return false
			}
		}
	}

	return true
}
