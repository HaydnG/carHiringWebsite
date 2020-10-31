package data

import "time"

type User struct {
	ID          int
	FullName    string    `json:"FullName"`
	Email       string    `json:"Email"`
	CreatedAt   time.Time `json:"CreatedAt"`
	Password    string
	AuthHash    []byte
	AuthSalt    []byte
	Blacklisted bool      `json:"Blacklisted"`
	DOB         time.Time `json:"DOB"`
	Verified    bool      `json:"Verified"`
	Repeat      bool      `json:"Repeat"`
}
