package model

import (
	"fmt"
	"sync"
	"time"
)

// ChannelEndpointEntry stores a tested endpoint type for a (channelId, model) pair.
type ChannelEndpointEntry struct {
	EndpointType string
	TestTime     int64
}

var (
	// channelEndpointCache: channelId -> model -> ChannelEndpointEntry
	channelEndpointCache     = make(map[int]map[string]ChannelEndpointEntry)
	channelEndpointCacheLock sync.RWMutex
)

func channelEndpointKey(channelId int, model string) string {
	return fmt.Sprintf("%d:%s", channelId, model)
}

// SetChannelEndpoint records a successfully tested endpoint type for a channel+model.
func SetChannelEndpoint(channelId int, model string, endpointType string) {
	if model == "" || endpointType == "" {
		return
	}
	channelEndpointCacheLock.Lock()
	defer channelEndpointCacheLock.Unlock()
	if _, ok := channelEndpointCache[channelId]; !ok {
		channelEndpointCache[channelId] = make(map[string]ChannelEndpointEntry)
	}
	channelEndpointCache[channelId][model] = ChannelEndpointEntry{
		EndpointType: endpointType,
		TestTime:     time.Now().Unix(),
	}
}

// GetChannelEndpoint returns the tested endpoint type for a channel+model, or "" if none.
func GetChannelEndpoint(channelId int, model string) string {
	channelEndpointCacheLock.RLock()
	defer channelEndpointCacheLock.RUnlock()
	if models, ok := channelEndpointCache[channelId]; ok {
		if entry, ok := models[model]; ok {
			return entry.EndpointType
		}
	}
	return ""
}

// ClearChannelEndpoints removes all cached endpoints for a channel.
func ClearChannelEndpoints(channelId int) {
	channelEndpointCacheLock.Lock()
	defer channelEndpointCacheLock.Unlock()
	delete(channelEndpointCache, channelId)
}

// GetAllChannelEndpoints returns a snapshot of the full cache (for observability).
func GetAllChannelEndpoints() map[int]map[string]ChannelEndpointEntry {
	channelEndpointCacheLock.RLock()
	defer channelEndpointCacheLock.RUnlock()
	result := make(map[int]map[string]ChannelEndpointEntry, len(channelEndpointCache))
	for chId, models := range channelEndpointCache {
		m := make(map[string]ChannelEndpointEntry, len(models))
		for model, entry := range models {
			m[model] = entry
		}
		result[chId] = m
	}
	return result
}
