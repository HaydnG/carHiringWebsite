package session

import (
	"carHiringWebsite/data"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

var (
	InvalidToken     error = errors.New("invalid token")
	InactiveSession  error = errors.New("inactive session")
	sessions         sessionStore
	sessionFormation []int = []int{8, 4, 4, 4, 12}
)

func init() {
	sessions = sessionStore{
		RWMutex:      sync.RWMutex{},
		storeByToken: make(map[string]*sessionBag),
		storeByEmail: make(map[string]*sessionBag),
		count:        0,
	}
}

func CountSesssions() int {
	return sessions.CountSessions()
}

func New(user *data.User) string {
	userCopy := *user

	userCopy.SessionToken = uuid.New().String()

	newBag := &sessionBag{
		lock:       sync.RWMutex{},
		token:      userCopy.SessionToken,
		email:      userCopy.Email,
		user:       userCopy,
		bag:        make(map[string]interface{}),
		lastActive: time.Now(),
	}

	sessions.Add(newBag)

	return newBag.token
}

func GetByEmail(email string) (*sessionBag, error) {
	return sessions.GetByEmail(email)
}

func GetByToken(token string) (*sessionBag, error) {
	return sessions.GetByToken(token)
}

func ValidateToken(token string) error {

	if len(token) != 36 {
		return InvalidToken
	}
	if strings.Count(token, "-") != 4 {
		return InvalidToken
	}

	parts := strings.Split(token, "-")

	for i, v := range sessionFormation {
		if len(parts[i]) != v {
			return InvalidToken
		}

	}

	return nil
}

func Delete(bag *sessionBag) bool {
	sessions.Delete(bag)

	return true
}
