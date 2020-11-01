package session

import (
	"carHiringWebsite/data"
	"sync"

	"github.com/google/uuid"
)

type sessionStore struct {
	sync.RWMutex
	storeByID    map[uuid.UUID]*sessionBag
	storeByEmail map[string]*sessionBag
}

func (ss *sessionStore) Add(bag *sessionBag) {
	ss.Lock()
	ss.storeByID[bag.token] = bag
	ss.storeByEmail[bag.user.Email] = bag
	ss.Unlock()
}

type sessionBag struct {
	sync.RWMutex
	token uuid.UUID
	user  data.User
	bag   map[string]interface{}
}

var (
	Sessions sessionStore
)

func init() {
	Sessions = sessionStore{
		RWMutex:      sync.RWMutex{},
		storeByID:    make(map[uuid.UUID]*sessionBag),
		storeByEmail: make(map[string]*sessionBag),
	}
}

func NewSessionBag(user *data.User) {
	userCopy := *user

	newBag := &sessionBag{
		RWMutex: sync.RWMutex{},
		token:   uuid.New(),
		user:    userCopy,
		bag:     make(map[string]interface{}),
	}

	Sessions.Add(newBag)

}
