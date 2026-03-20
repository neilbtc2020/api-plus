package service

import (
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/pkg/cachex"

	"github.com/samber/hot"
)

var ErrModelAvailabilityGroupNotAccessible = errors.New("group not found or not accessible")

const (
	modelAvailabilitySnapshotFreshTTL     = 60 * time.Second
	modelAvailabilitySnapshotCacheTTL     = 10 * time.Minute
	modelAvailabilityProbeCacheTTL        = 5 * time.Minute
	modelAvailabilitySnapshotCacheNS      = "model_availability_snapshot:v1"
	modelAvailabilityProbeCacheNS         = "model_availability_probe:v1"
	modelAvailabilitySnapshotCacheCapcity = 256
	modelAvailabilityProbeCacheCapacity   = 4096
)

var (
	modelAvailabilitySnapshotCacheOnce sync.Once
	modelAvailabilitySnapshotCache     *cachex.HybridCache[groupAvailabilityCacheValue]

	modelAvailabilityProbeCacheOnce sync.Once
	modelAvailabilityProbeCache     *cachex.HybridCache[ProbeStatus]
)

func getModelAvailabilitySnapshotCache() *cachex.HybridCache[groupAvailabilityCacheValue] {
	modelAvailabilitySnapshotCacheOnce.Do(func() {
		modelAvailabilitySnapshotCache = cachex.NewHybridCache[groupAvailabilityCacheValue](cachex.HybridCacheConfig[groupAvailabilityCacheValue]{
			Namespace: cachex.Namespace(modelAvailabilitySnapshotCacheNS),
			Redis:     common.RDB,
			RedisEnabled: func() bool {
				return common.RedisEnabled && common.RDB != nil
			},
			RedisCodec: cachex.JSONCodec[groupAvailabilityCacheValue]{},
			Memory: func() *hot.HotCache[string, groupAvailabilityCacheValue] {
				return hot.NewHotCache[string, groupAvailabilityCacheValue](hot.LRU, modelAvailabilitySnapshotCacheCapcity).
					WithTTL(modelAvailabilitySnapshotCacheTTL).
					WithJanitor().
					Build()
			},
		})
	})
	return modelAvailabilitySnapshotCache
}

func getModelAvailabilityProbeCache() *cachex.HybridCache[ProbeStatus] {
	modelAvailabilityProbeCacheOnce.Do(func() {
		modelAvailabilityProbeCache = cachex.NewHybridCache[ProbeStatus](cachex.HybridCacheConfig[ProbeStatus]{
			Namespace: cachex.Namespace(modelAvailabilityProbeCacheNS),
			Redis:     common.RDB,
			RedisEnabled: func() bool {
				return common.RedisEnabled && common.RDB != nil
			},
			RedisCodec: cachex.JSONCodec[ProbeStatus]{},
			Memory: func() *hot.HotCache[string, ProbeStatus] {
				return hot.NewHotCache[string, ProbeStatus](hot.LRU, modelAvailabilityProbeCacheCapacity).
					WithTTL(modelAvailabilityProbeCacheTTL).
					WithJanitor().
					Build()
			},
		})
	})
	return modelAvailabilityProbeCache
}

func isGroupAvailabilitySnapshotFresh(refreshedAt int64) bool {
	if refreshedAt <= 0 {
		return false
	}
	return time.Since(time.Unix(refreshedAt, 0)) < modelAvailabilitySnapshotFreshTTL
}

func loadGroupAvailabilityCache(group string) (groupAvailabilityCacheValue, bool, error) {
	return getModelAvailabilitySnapshotCache().Get(strings.TrimSpace(group))
}

func saveGroupAvailabilityCache(value groupAvailabilityCacheValue) error {
	value.Group = strings.TrimSpace(value.Group)
	if value.Group == "" {
		return nil
	}
	return getModelAvailabilitySnapshotCache().SetWithTTL(value.Group, value, modelAvailabilitySnapshotCacheTTL)
}

func LoadModelAvailabilityProbe(group string, modelName string) (*ProbeStatus, bool, error) {
	value, found, err := getModelAvailabilityProbeCache().Get(modelAvailabilityProbeCacheKey(group, modelName))
	if err != nil || !found {
		return nil, found, err
	}
	probe := value
	return &probe, true, nil
}

func SaveModelAvailabilityProbe(group string, modelName string, probe ProbeStatus) error {
	return getModelAvailabilityProbeCache().SetWithTTL(modelAvailabilityProbeCacheKey(group, modelName), probe, modelAvailabilityProbeCacheTTL)
}

func modelAvailabilityProbeCacheKey(group string, modelName string) string {
	group = strings.TrimSpace(group)
	modelName = strings.TrimSpace(modelName)
	if group == "" || modelName == "" {
		return ""
	}
	return group + ":" + modelName
}
