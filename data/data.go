package data

import (
	"database/sql"
	"strconv"
	"time"
)

type BookingStat struct {
	ProcessID     int    `json:"ProcessID"`
	Description   string `json:"Description"`
	Count         int    `json:"Count"`
	AdminRequired bool   `json:"AdminRequired"`
}

type UserStat struct {
	AdminCount       int `json:"AdminCount"`
	BlackListedCount int `json:"BlackListedCount"`
	RepeatUsersCount int `json:"RepeatUsersCount"`
	UserCount        int `json:"UserCount"`
	ActiveUsers      int `json:"ActiveUsers"`
	DisabledCount    int `json:"DisabledCount"`
}

type CarStat struct {
	CarCount       int `json:"CarCount"`
	DisabledCount  int `json:"DisabledCount"`
	AvailableCount int `json:"AvailableCount"`
}

type AccessoryStat struct {
	ID          int    `json:"ID"`
	Description string `json:"Description"`
	Stock       int    `json:"Stock"`
}

type BookingStatusType struct {
	ID            int    `json:"ID"`
	Description   string `json:"Description"`
	AdminRequired bool   `json:"AdminRequired"`
	Order         int    `json:"Order"`
	BookingPage   bool   `json:"BookingPage"`
}

type AdminBooking struct {
	Booking *Booking    `json:"booking"`
	User    *OutputUser `json:"user"`
}

type ExtensionResponse struct {
	Days int `json:"days"`
}

type Booking struct {
	ID                   int                  `json:ID`
	CarID                int                  `json:"carID"`
	UserID               int                  `json:"userID"`
	Start                timestamp            `json:"start"`
	End                  timestamp            `json:"end"`
	Finish               timestamp            `json:"finish"`
	TotalCost            float64              `json:"totalCost"`
	AmountPaid           float64              `json:"amountPaid"`
	LateReturn           bool                 `json:"lateReturn"`
	FullDay              bool                 `json:"fullDay"`
	Created              timestamp            `json:"created"`
	BookingLength        float64              `json:"bookingLength"`
	ProcessID            int                  `json:"processID"`
	ProcessName          string               `json:"processName"`
	AdminRequired        bool                 `json:"adminRequired"`
	CarData              *Car                 `json:"carData"`
	Accessories          []*Accessory         `json:"accessories"`
	AwaitingExtraPayment bool                 `json:"awaitingExtraPayment"`
	IsRefund             bool                 `json:"isRefund"`
	ActiveStatuses       []*BookingStatusType `json:"activeStatuses"`
	PerDay               float64              `json:"perDay"`
	DriverID             sql.NullInt32
}
type ImageBundle struct {
	License   string `json:"license"`
	Document1 string `json:"document1"`
	Document2 string `json:"document2"`
}

type Driver struct {
	ID            int
	LastName      string
	Names         string
	LicenseNumber string
	Address       string
	PostCode      string
	BlackListed   bool
	DOB           timestamp
}

type InsurerColumn struct {
	ID          int
	LastName    string
	FisrtName   string
	DOB         timestamp
	Address     string
	DOC         timestamp
	InsurerCode string
}

type BookingColumn struct {
	ID             int       `json:ID`
	CarID          int       `json:"carID"`
	UserID         int       `json:"userID"`
	Start          timestamp `json:"start"`
	End            timestamp `json:"end"`
	Finish         timestamp `json:"finish"`
	TotalCost      float64   `json:"totalCost"`
	AmountPaid     float64   `json:"amountPaid"`
	LateReturn     bool      `json:"lateReturn"`
	FullDay        bool      `json:"fullDay"`
	Created        timestamp `json:"created"`
	BookingLength  float64   `json:"bookingLength"`
	CarDescription string    `json:"CarDescription"`
	UserFirstName  string    `json:"UserFirstName"`
	UserOtherName  string    `json:"UserOtherName"`
	Process        string    `json:"process"`
	ProcessID      int       `json:"processID"`
	PerDay         float64   `json:"perDay"`
	DriverID       sql.NullInt32
}

type CarAttribute struct {
	ID          int    `json:"ID"`
	Description string `json:"Description"`
}

type TimeRange struct {
	Start time.Time `json:"Start"`
	End   time.Time `json:"End"`
}
type OutputTimeRange struct {
	Start timestamp `json:"Start"`
	End   timestamp `json:"End"`
}

func ConvertTimeRangeSlice(ranges []*TimeRange) []*OutputTimeRange {

	outputSlice := make([]*OutputTimeRange, len(ranges))

	for i, timeRange := range ranges {
		outputSlice[i] = &OutputTimeRange{
			Start: timestamp{timeRange.Start},
			End:   timestamp{timeRange.End},
		}
	}

	return outputSlice
}

type Accessory struct {
	ID          int    `json:"ID"`
	Description string `json:"Description"`
	Checked     bool   `json:"Checked"`
}

type Attribute struct {
	ID          int    `json:"ID"`
	Description string `json:"Description"`
}

type UserBundle struct {
	User     *OutputUser      `json:"user"`
	Bookings []*BookingColumn `json:"bookings"`
}

type Car struct {
	ID           int        `json:"ID"`
	FuelType     *Attribute `json:"FuelType"`
	GearType     *Attribute `json:"GearType"`
	CarType      *Attribute `json:"CarType"`
	Size         *Attribute `json:"Size"`
	Colour       *Attribute `json:"Colour"`
	Cost         float64    `json:"Cost"`
	Description  string     `json:"Description"`
	Image        string     `json:"Image"`
	Seats        int        `json:"Seats"`
	Disabled     bool
	BookingCount int  `json:"BookingCount"`
	Over25       bool `json:"Over25"`
}

func NewCar() *Car {
	return &Car{
		ID: 0,
		FuelType: &Attribute{
			ID:          0,
			Description: "",
		},
		GearType: &Attribute{
			ID:          0,
			Description: "",
		},
		CarType: &Attribute{
			ID:          0,
			Description: "",
		},
		Size: &Attribute{
			ID:          0,
			Description: "",
		},
		Colour: &Attribute{
			ID:          0,
			Description: "",
		},
		Cost:        0,
		Description: "",
		Image:       "",
	}
}

type User struct {
	ID           int
	FirstName    string
	Names        string
	Email        string
	CreatedAt    time.Time
	Password     string
	AuthHash     string
	AuthSalt     string
	Blacklisted  bool
	DOB          time.Time
	Verified     bool
	Repeat       bool
	SessionToken string
	Admin        bool
	BookingCount int
	Disabled     bool
}

type timestamp struct {
	time.Time
}

func ConvertDate(d time.Time) *timestamp {
	return &timestamp{d}
}

//OutputUser used for serialisation
type OutputUser struct {
	ID           int       `json:"ID,omitempty"`
	FirstName    string    `json:"FirstName"`
	Names        string    `json:"Names"`
	Email        string    `json:"Email"`
	CreatedAt    timestamp `json:"CreatedAt"`
	Blacklisted  bool      `json:"Blacklisted"`
	DOB          timestamp `json:"DOB"`
	Verified     bool      `json:"Verified"`
	Repeat       bool      `json:"Repeat"`
	SessionToken string    `json:"SessionToken"`
	Admin        bool      `json:"Admin"`
	BookingCount int       `json:"BookingCount"`
	Disabled     bool      `json:"Disabled"`
}

type BookingStatus struct {
	ID                 int       `json:"ID"`
	BookingID          int       `json:"BookingID"`
	Completed          timestamp `json:"Completed"`
	Active             bool      `json:"Active"`
	AdminID            int       `json:"AdminID"`
	Description        string    `json:"Description"`
	ProcessID          int       `json:"ProcessID"`
	ProcessDescription string    `json:"ProcessDescription"`
	AdminRequired      bool      `json:"AdminRequired"`
	Order              int       `json:"Order"`
	BookingPage        bool      `json:"BookingPage"`
	Extra              float64   `json:"Extra"`
}

type Response struct {
	ID string `json:"ID"`
}

func (t timestamp) MarshalJSON() ([]byte, error) {
	tim := time.Time(t.Time).Unix()
	if tim < 0 {
		tim = 0
	}
	return []byte(strconv.FormatInt(tim, 10)), nil
}

func NewOutputUser(u *User) *OutputUser {
	return &OutputUser{
		FirstName:    u.FirstName,
		Names:        u.Names,
		Email:        u.Email,
		CreatedAt:    timestamp{u.CreatedAt},
		Blacklisted:  u.Blacklisted,
		DOB:          timestamp{u.DOB},
		Verified:     u.Verified,
		Repeat:       u.Repeat,
		SessionToken: u.SessionToken,
		Admin:        u.Admin,
		BookingCount: u.BookingCount,
		Disabled:     u.Disabled,
		ID:           u.ID,
	}
}
