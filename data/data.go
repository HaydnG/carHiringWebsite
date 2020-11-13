package data

import (
	"strconv"
	"time"
)

type User struct {
	ID           int
	FirstName    string
	Names        string
	Email        string
	CreatedAt    time.Time
	Password     string
	AuthHash     []byte
	AuthSalt     []byte
	Blacklisted  bool
	DOB          time.Time
	Verified     bool
	Repeat       bool
	SessionToken string
}

type timestamp struct {
	time.Time
}

//OutputUser used for serialisation
type OutputUser struct {
	FirstName    string    `json:"FirstName"`
	Names        string    `json:"Names"`
	Email        string    `json:"Email"`
	CreatedAt    timestamp `json:"CreatedAt"`
	Blacklisted  bool      `json:"Blacklisted"`
	DOB          timestamp `json:"DOB"`
	Verified     bool      `json:"Verified"`
	Repeat       bool      `json:"Repeat"`
	SessionToken string    `json:"SessionToken"`
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
	}
}
