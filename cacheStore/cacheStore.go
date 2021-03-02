package cacheStore

import (
	"fmt"
	"sync"
	"time"
)

func NewStore(name string, duration time.Duration) store {

	s := store{
		name:            name,
		data:            make(map[string]cacheItem),
		duration:        duration,
		cleanUpInterval: 60,
		cleanUpActive:   true,
	}
	s.cleanUpJob()

	return s
}

type cacheItem struct {
	key     string
	created time.Time
	data    interface{}
}

type store struct {
	lock            sync.RWMutex
	name            string
	data            map[string]cacheItem
	duration        time.Duration
	cleanUpInterval int
	cleanUpActive   bool
}

func (s *store) cleanUpJob() {
	duration := s.duration * time.Second
	sleepInterval := time.Duration(s.cleanUpInterval) * time.Second

	go func() {
		for s.cleanUpActive {
			time.Sleep(sleepInterval)
			if !s.cleanUpActive {
				return
			}
			if len(s.data) < 1 {
				continue
			}
			fmt.Println("Cleaning cache.....")
			s.lock.Lock()
			now := time.Now()

			for key, item := range s.data {
				if now.Sub(item.created) >= duration {
					delete(s.data, key)
				}
			}
			s.lock.Unlock()
		}
	}()
}

func (s *store) GetData(key string, dataFunction func(key string) (interface{}, error)) (interface{}, error) {
	var err error

	s.lock.RLock()
	item, ok := s.data[key]
	if !ok {
		s.lock.RUnlock()
		s.lock.RLock()
		item, ok = s.data[key]
		if !ok {
			s.lock.RUnlock()
			item, err = s.addData(key, dataFunction)
			if err != nil {
				return nil, err
			}
		}
	}

	if ok {
		s.lock.RUnlock()
		if time.Now().Sub(item.created) >= (time.Second * s.duration) {
			item, err = s.addData(key, dataFunction)
			if err != nil {
				return nil, err
			}
		}
	}

	return item.data, nil
}

func (s *store) invalidateData(key string) {
	s.lock.Lock()
	delete(s.data, key)
	s.lock.Unlock()
}

func (s *store) addData(key string, dataFunction func(key string) (interface{}, error)) (cacheItem, error) {

	s.lock.Lock()
	data, err := dataFunction(key)
	if err != nil {
		return cacheItem{}, err
	}

	item := cacheItem{
		key:     key,
		created: time.Now(),
		data:    data,
	}
	s.data[key] = item

	s.lock.Unlock()

	return item, nil
}
