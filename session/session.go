package session

import (
	"carHiringWebsite/data"
	"strings"
	"sync"

	"github.com/google/uuid"
)

var (
	sessions         sessionStore
	sessionFormation []int = []int{8, 4, 4, 4, 12}
)

func init() {
	sessions = sessionStore{
		RWMutex:      sync.RWMutex{},
		storeByToken: make(map[string]*sessionBag),
		storeByEmail: make(map[string]*sessionBag),
	}
}

func New(user *data.User) string {
	userCopy := *user

	newBag := &sessionBag{
		lock:  sync.RWMutex{},
		token: uuid.New().String(),
		user:  userCopy,
		bag:   make(map[string]interface{}),
	}

	sessions.Add(newBag)

	return newBag.token
}

func GetByToken(token string) *sessionBag {
	return sessions.GetByToken(token)
}

func ValidateToken(token string) bool {

	if len(token) != 36 {
		return false
	}
	if strings.Count(token, "-") != 4 {
		return false
	}

	parts := strings.Split(token, "-")

	for i, v := range sessionFormation {
		if len(parts[i]) != v {
			return false
		}

	}

	return true
}
