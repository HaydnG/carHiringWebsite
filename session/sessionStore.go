package session

import (
	"carHiringWebsite/data"
	"sync"
	"time"
)

const sessionExpiry = time.Hour

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

func (ss *sessionStore) GetByToken(token string) (*sessionBag, bool) {
	ss.RLock()
	bag, ok := ss.storeByToken[token]
	ss.RUnlock()

	if ok && !bag.isActive() {
		ss.Delete(bag)
		return nil, false
	}

	return bag, ok
}

func (ss *sessionStore) GetByEmail(email string) (*sessionBag, bool) {
	ss.RLock()
	bag, ok := ss.storeByEmail[email]
	ss.RUnlock()

	if ok && !bag.isActive() {
		ss.Delete(bag)
		return nil, false
	}

	return bag, ok
}

func (ss *sessionStore) Delete(bag *sessionBag) {
	ss.Lock()
	defer ss.Unlock()

	if _, ok := ss.storeByEmail[bag.email]; ok {
		delete(ss.storeByEmail, bag.email)
	}
	if _, ok := ss.storeByToken[bag.token]; ok {
		delete(ss.storeByToken, bag.token)
	}
}

type sessionBag struct {
	lock       sync.RWMutex
	token      string
	email      string
	user       data.User
	bag        map[string]interface{}
	lastActive time.Time
}

//GetUser gives back a copy of the user object stored in the session
func (sb *sessionBag) GetUser() *data.User {
	sb.lock.RLock()
	userCopy := sb.user
	sb.lock.RUnlock()
	return &userCopy
}

func (sb *sessionBag) GetToken() string {
	sb.lock.RLock()
	tokenCopy := sb.token
	sb.lock.RUnlock()
	return tokenCopy
}

func (sb *sessionBag) isActive() bool {
	sessionDuration := time.Now().Sub(sb.lastActive)
	if sessionDuration > sessionExpiry {
		return false
	}

	sb.lastActive = time.Now()
	return true
}