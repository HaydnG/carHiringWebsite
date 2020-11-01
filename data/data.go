package data

import (
	"strconv"
	"time"
)

type User struct {
	ID          int
	FullName    string
	Email       string
	CreatedAt   time.Time
	Password    string
	AuthHash    []byte
	AuthSalt    []byte
	Blacklisted bool
	DOB         time.Time
	Verified    bool
	Repeat      bool
}

type timestamp struct {
	time.Time
}

//OutputUser used for serialisation
type OutputUser struct {
	FullName     string    `json:"FullName"`
	Email        string    `json:"Email"`
	CreatedAt    timestamp `json:"CreatedAt"`
	Blacklisted  bool      `json:"Blacklisted"`
	DOB          timestamp `json:"DOB"`
	Verified     bool      `json:"Verified"`
	Repeat       bool      `json:"Repeat"`
	SessionToken string    `json:"SessionToken"`
}

func (t timestamp) MarshalJSON() ([]byte, error) {
	return []byte(strconv.FormatInt(time.Time(t.Time).Unix(), 10)), nil
}

func NewOutputUser(u *User) *OutputUser {
	return &OutputUser{
		FullName:    u.FullName,
		Email:       u.Email,
		CreatedAt:   timestamp{u.CreatedAt},
		Blacklisted: u.Blacklisted,
		DOB:         timestamp{u.DOB},
		Verified:    u.Verified,
		Repeat:      u.Repeat,
	}
}
