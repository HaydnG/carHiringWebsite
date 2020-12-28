package bookingService

import (
	"carHiringWebsite/data"
	"carHiringWebsite/db"
	"carHiringWebsite/session"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

const (
	lateReturnIncrease = 0.6
	extensionIncrease  = 0.5
)
const (
	awaitingPayment = iota + 1
	paymentAccepted
	awaitingConfirmation
	bookingConfirmed
	bookingEdited
	editAwaitingPayment
	editPaymentAccepted
	queryingRefund
	refundRejected
	refundIssued
	canceledBooking
)

func Create(token, start, end, carID, late, extension, accessories, days string) (*data.Booking, error) {
	var finishString string

	err := session.ValidateToken(token)
	if err != nil {
		return nil, err
	}

	bag, err := session.GetByToken(token)
	if err != nil {
		return nil, err
	}
	user := bag.GetUser()

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
	extensionValue, err := strconv.ParseBool(extension)
	if err != nil {
		return nil, err
	}

	if lateValue {
		if !user.Repeat {
			return nil, errors.New("cannot make a late booking without repeat status")
		}
		extensionValue = false
	}

	if lateValue {
		calculatedDays += lateReturnIncrease
	} else if extensionValue {
		calculatedDays += extensionIncrease
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

	// Check if extension or lateBooking is allowed
	dayAfterBooking := endTime.Add(time.Hour * 24).Format("2006-01-02")
	nextDayBooked, err := db.BookingHasOverlap(dayAfterBooking, dayAfterBooking, carID)
	if err != nil {
		return nil, err
	}
	if nextDayBooked && (lateValue || extensionValue) {
		return nil, errors.New("no extension allowed on this booking")
	}

	car, err := db.GetCar(carID)
	if err != nil {
		return nil, err
	}
	if car == nil {
		return nil, errors.New("problem retrieving car")
	}
	price := car.Cost * daysValue

	startString := startTime.Format("2006-01-02")
	endString := endTime.Format("2006-01-02")

	if lateValue || extensionValue {
		finishTime := finishTime.Add(time.Hour * 24)
		finishString = finishTime.Format("2006-01-02")
	} else {
		finishString = finishTime.Format("2006-01-02")
	}

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
		finishString,
		price,
		lateValue, extensionValue, calculatedDays)
	if err != nil {
		return nil, err
	}

	_, err = db.InsertBookingStatus(bookingID, awaitingPayment, 0, 1, "")
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
	err := session.ValidateToken(token)
	if err != nil {
		return err
	}

	bag, err := session.GetByToken(token)
	if err != nil {
		return err
	}
	user := bag.GetUser()

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

	if booking.ProcessID != awaitingPayment {
		return errors.New("booking not awaiting payment")
	}

	amountDue := booking.TotalCost - booking.AmountPaid
	if amountDue <= 0 {
		return errors.New("no payment needed")
	}

	status, err := db.GetBookingProcessStatus(booking.ID, awaitingPayment)
	if err != nil {
		return err
	} else if status == nil || !status.Active {
		return errors.New("booking not awaiting payment")
	}

	err = db.UpdateBookingPayment(booking.ID, user.ID, amountDue)
	if err != nil {
		return err
	}

	_, err = db.InsertBookingStatus(booking.ID, paymentAccepted, 0, 0, "Made payment of £"+strconv.FormatFloat(amountDue, 'f', 2, 64))
	if err != nil {
		return err
	}

	// disable awaiting payment status
	err = db.SetBookingStatus(status.ID, false)
	if err != nil {
		return err
	}

	_, err = db.InsertBookingStatus(booking.ID, awaitingConfirmation, 0, 1, "")
	if err != nil {
		return err
	}

	return nil
}

func GetUsersBookings(token string) (map[int][]*data.Booking, error) {
	err := session.ValidateToken(token)
	if err != nil {
		return nil, err
	}

	bag, err := session.GetByToken(token)
	if err != nil {
		return nil, err
	}
	user := bag.GetUser()

	bookings, err := db.GetUsersBookings(user.ID)
	if err != nil {
		return nil, err
	}
	if len(bookings) <= 0 {
		return map[int][]*data.Booking{}, nil
	}

	return organiseBookings(bookings), nil
}

func organiseBookings(bookings []*data.Booking) map[int][]*data.Booking {
	organisedBookings := make(map[int][]*data.Booking)

	for _, value := range bookings {
		if _, exists := organisedBookings[value.ProcessID]; !exists {
			organisedBookings[value.ProcessID] = make([]*data.Booking, 1)
			organisedBookings[value.ProcessID][0] = value
		} else {
			organisedBookings[value.ProcessID] = append(organisedBookings[value.ProcessID], value)
		}
	}

	return organisedBookings
}

func CancelBooking(token, bookingID string) error {
	err := session.ValidateToken(token)
	if err != nil {
		return err
	}

	bag, err := session.GetByToken(token)
	if err != nil {
		return err
	}
	user := bag.GetUser()

	bookingIDValid, err := strconv.Atoi(bookingID)
	if err != nil {
		return err
	}

	booking, err := db.GetSingleBooking(bookingIDValid)
	if err != nil {
		return err
	}

	if user.ID != booking.UserID {
		return errors.New("this booking does not belong to this user")
	}

	if booking.ProcessID == canceledBooking {
		return errors.New("booking already canceled")
	}

	err = db.DeactivateBookingStatuses(booking.ID)
	if err != nil {
		return err
	}

	_, err = db.InsertBookingStatus(booking.ID, queryingRefund, 0, 1, "Automatic refund query requested")
	if err != nil {
		return err
	}

	_, err = db.InsertBookingStatus(booking.ID, canceledBooking, 0, 1, "User canceled booking")
	if err != nil {
		return err
	}

	return nil
}

func GetHistory(token, bookingID string) ([]*data.BookingStatus, error) {
	err := session.ValidateToken(token)
	if err != nil {
		return nil, err
	}

	bag, err := session.GetByToken(token)
	if err != nil {
		return nil, err
	}
	user := bag.GetUser()

	bookingIDValid, err := strconv.Atoi(bookingID)
	if err != nil {
		return nil, err
	}

	booking, err := db.GetSingleBooking(bookingIDValid)
	if err != nil {
		return nil, err
	}

	if user.ID != booking.UserID {
		return nil, errors.New("this booking does not belong to this user")
	}

	history, err := db.GetBookingHistory(booking.ID)

	return history, err
}

func EditBooking(token, bookingID, remove, add, lateReturn, extension string) error {
	edited := false
	var description string
	var AddAccessory, RemoveAccessory []string
	var amountDue float64

	err := session.ValidateToken(token)
	if err != nil {
		return err
	}

	bag, err := session.GetByToken(token)
	if err != nil {
		return err
	}
	user := bag.GetUser()

	bookingIDValid, err := strconv.Atoi(bookingID)
	if err != nil {
		return err
	}

	booking, err := db.GetSingleBooking(bookingIDValid)
	if err != nil {
		return err
	}

	if user.ID != booking.UserID {
		return errors.New("this booking does not belong to this user")
	}

	if booking.ProcessID == canceledBooking {
		return errors.New("booking already canceled")
	}

	lateReturnValue, err := strconv.ParseBool(lateReturn)
	if err != nil {
		return err
	}
	extensionValue, err := strconv.ParseBool(extension)
	if err != nil {
		return err
	}

	if lateReturnValue {
		extensionValue = false
	}

	days := booking.BookingLength
	newCost := booking.TotalCost
	if lateReturnValue != booking.LateReturn || extensionValue != booking.Extension {
		dailyCost := booking.TotalCost / booking.BookingLength

		if booking.LateReturn {
			days = days - lateReturnIncrease
		} else if booking.Extension {
			days = days - extensionIncrease
		}

		if lateReturnValue {
			days = days + lateReturnIncrease
		} else if extensionValue {
			days = days + extensionIncrease
		}

		newCost = dailyCost * days

		description += fmt.Sprintf("£%.2f -> £%.2f | ", booking.TotalCost, newCost)

		if lateReturnValue != booking.LateReturn {
			description += fmt.Sprintf("LateReturn: %t | ", lateReturnValue)
		} else if extensionValue != booking.Extension {
			description += fmt.Sprintf("Extension: %t | ", extensionValue)
		}

		err := db.UpdateBooking(booking.ID, user.ID, newCost, days, lateReturnValue, extensionValue)
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
		_, err = db.InsertBookingStatus(booking.ID, bookingEdited, 0, 0, description)
		if err != nil {
			return err
		}

		if booking.ProcessID != awaitingPayment {
			status, err := db.GetBookingProcessStatus(booking.ID, editAwaitingPayment)
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

				_, err = db.InsertBookingStatus(booking.ID, editAwaitingPayment, 0, 1, paymentDesc)
				if err != nil {
					return err
				}
			}

		}
	}

	return nil
}

func GetAccessoryNames(acces []*data.Accessory, ids []string) ([]string, error) {

	if len(ids) <= 0 || len(acces) <= 0 {
		return nil, errors.New("GetAccessoryNames list is empty")
	}

	names := make([]string, len(ids))
	count := 0

	for _, v := range ids {
		id, err := strconv.Atoi(v)
		if err != nil {
			return nil, err
		}

		for _, v := range acces {
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
