package session

import (
	"carHiringWebsite/data"
	"sync"
)

type sessionStore struct {
	sync.RWMutex
	storeByToken map[string]*sessionBag
	storeByEmail map[string]*sessionBag
}

func (ss *sessionStore) Add(bag *sessionBag) {
	ss.Lock()
	ss.storeByToken[bag.token] = bag
	ss.storeByEmail[bag.user.Email] = bag
	ss.Unlock()
}

func (ss *sessionStore) GetByToken(token string) *sessionBag {
	ss.RLock()
	bag, ok := ss.storeByToken[token]
	ss.RUnlock()

	if !ok {
		return nil
	}

	return bag
}

type sessionBag struct {
	lock  sync.RWMutex
	token string
	user  data.User
	bag   map[string]interface{}
}

//GetUser gives back a copy of the user object stored in the session
func (sb *sessionBag) GetUser() *data.User {
	sb.lock.RLock()
	userCopy := sb.user
	sb.lock.RUnlock()
	return &userCopy
}
