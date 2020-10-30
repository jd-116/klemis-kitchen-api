package auth

import (
	"sync"
	"time"
)

// FlowContinuationMap is a wrapper around a standard
// sync.Map that also periodically evicts old entries.
// This is useful because it ensures the size of the map doesn't
// get too large after abandoned authentication flows
type FlowContinuationMap struct {
	internal sync.Map
}

// item is the value type for the internal map
type item struct {
	value        string
	creationTime int64
}

// NewFlowContinuationMap creates a new instance of FlowContinuationMap
// and starts the goroutine that evicts old entries.
func NewFlowContinuationMap(interval time.Duration, maxTTL int64) *FlowContinuationMap {
	m := FlowContinuationMap{
		internal: sync.Map{},
	}

	go m.evict(interval, maxTTL)
	return &m
}

// Blocking function that periodically evicts old entries
func (m *FlowContinuationMap) evict(interval time.Duration, maxTTL int64) {
	for now := range time.Tick(interval) {
		// Deletes all values that are too old
		m.internal.Range(func(key interface{}, value interface{}) bool {
			if now.Unix()-value.(item).creationTime > int64(maxTTL) {
				m.internal.Delete(key)
			}
			return true
		})
	}
}

// Gets a value in the map, or returns with false as the second value
func (m *FlowContinuationMap) Get(key string) (string, bool) {
	result, ok := m.internal.LoadAndDelete(key)
	if ok {
		return result.(item).value, true
	}

	return "", false
}

// Stores a value in the map
func (m *FlowContinuationMap) Put(key string, value string) {
	m.internal.Store(key, item{value, time.Now().Unix()})
}
