package session

import (
	"carHiringWebsite/data"
	"sync"

	"github.com/google/uuid"
)

type sessionStore struct {
	sync.RWMutex
	storeByID    map[string]*sessionBag
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
	token string
	user  data.User
	bag   map[string]interface{}
}

var (
	sessions sessionStore
)

func init() {
	sessions = sessionStore{
		RWMutex:      sync.RWMutex{},
		storeByID:    make(map[string]*sessionBag),
		storeByEmail: make(map[string]*sessionBag),
	}
}

func New(user *data.User) string {
	userCopy := *user

	newBag := &sessionBag{
		RWMutex: sync.RWMutex{},
		token:   uuid.New().String(),
		user:    userCopy,
		bag:     make(map[string]interface{}),
	}

	sessions.Add(newBag)

	return newBag.token

}
