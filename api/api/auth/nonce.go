package auth

import (
	"log"
	"sync"
	"time"

	"github.com/segmentio/ksuid"
)

// NonceMap is a wrapper around a standard
// sync.Map that also periodically evicts old entries.
// This is useful because it ensures the size of the map doesn't
// get too large after abandoned authentication flows
type NonceMap struct {
	internal sync.Map
	maxTTL   int64
}

// item is the value type for the internal map
type item struct {
	value        interface{}
	creationTime int64
}

// NewNonceMap creates a new instance of NonceMap
// and starts the goroutine that evicts old entries.
func NewNonceMap(interval time.Duration, maxTTL int64) *NonceMap {
	m := NonceMap{
		internal: sync.Map{},
		maxTTL:   maxTTL,
	}

	go m.evict(interval)
	return &m
}

// Blocking function that periodically evicts old entries
func (m *NonceMap) evict(interval time.Duration) {
	for now := range time.Tick(interval) {
		// Deletes all values that are too old
		m.internal.Range(func(key interface{}, value interface{}) bool {
			if now.Unix()-value.(item).creationTime > int64(m.maxTTL) {
				m.internal.Delete(key)
			}
			return true
		})
	}
}

// Gets a value in the map, or returns with false as the second value.
// Deletes the item if it exists
func (m *NonceMap) Use(nonce string) (interface{}, bool) {
	log.Println("nonce map dump:")
	m.internal.Range(func(key interface{}, value interface{}) bool {
		remaining := time.Now().Unix() - value.(item).creationTime
		log.Printf("'%s' -> '%s' (%d)\n", key.(string), value.(item).value, remaining)
		return true
	})

	value, ok := m.internal.LoadAndDelete(nonce)
	if ok {
		// Make sure the item isn't invalid
		if time.Now().Unix()-value.(item).creationTime > int64(m.maxTTL) {
			return "", false
		}

		return value.(item).value, true
	}

	return "", false
}

// Provisions a new unique nonce and stores it in the map
func (m *NonceMap) Provision(value interface{}) (string, error) {
	newItem := item{value, time.Now().Unix()}
	for {
		nonce, err := ksuid.NewRandom()
		if err != nil {
			return "", err
		}

		nonceStr := nonce.String()
		_, exists := m.internal.LoadOrStore(nonceStr, newItem)
		if !exists {
			return nonceStr, nil
		}
	}
}
