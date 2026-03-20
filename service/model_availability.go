package service

import (
	"sort"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

const (
	availabilityResultSuccess = "success"
	availabilityResultFail    = "fail"

	modelAvailabilityRecentWindow = 20
	modelAvailabilityLogScanLimit = 2000
)

type availabilityLogEntry struct {
	ModelName string
	Success   bool
}

type RecentResult struct {
	Status    string `json:"status"`
	Defaulted bool   `json:"defaulted"`
}

type ProbeStatus struct {
	Status         string `json:"status"`
	CheckedAt      int64  `json:"checked_at"`
	Message        string `json:"message"`
	ResponseTimeMs int64  `json:"response_time_ms"`
}

type ModelAvailabilityItem struct {
	ModelName       string         `json:"model_name"`
	ConfigAvailable bool           `json:"config_available"`
	RecentResults   []RecentResult `json:"recent_results"`
	SuccessCount    int            `json:"success_count"`
	FailCount       int            `json:"fail_count"`
	HasRealLogs     bool           `json:"has_real_logs"`
	Probe           *ProbeStatus   `json:"probe,omitempty"`
}

type VisibleGroup struct {
	Name     string `json:"name"`
	Selected bool   `json:"selected"`
}

type GroupAvailabilitySnapshot struct {
	Groups        []VisibleGroup          `json:"groups"`
	SelectedGroup string                  `json:"selected_group"`
	RefreshedAt   int64                   `json:"refreshed_at"`
	Items         []ModelAvailabilityItem `json:"items"`
}

type groupAvailabilityCacheValue struct {
	Group       string                  `json:"group"`
	RefreshedAt int64                   `json:"refreshed_at"`
	Items       []ModelAvailabilityItem `json:"items"`
}

func buildRecentResultsWindow(entries []availabilityLogEntry, windowSize int) []RecentResult {
	if windowSize <= 0 {
		return []RecentResult{}
	}

	results := make([]RecentResult, 0, windowSize)
	for _, entry := range entries {
		if len(results) >= windowSize {
			break
		}
		status := availabilityResultSuccess
		if !entry.Success {
			status = availabilityResultFail
		}
		results = append(results, RecentResult{
			Status: status,
		})
	}

	for len(results) < windowSize {
		results = append(results, RecentResult{
			Status:    availabilityResultSuccess,
			Defaulted: true,
		})
	}

	return results
}

func filterAvailabilityItems(items []ModelAvailabilityItem, keyword string, onlyFailed bool, onlyWithLogs bool) []ModelAvailabilityItem {
	normalizedKeyword := strings.ToLower(strings.TrimSpace(keyword))
	filtered := make([]ModelAvailabilityItem, 0, len(items))

	for _, item := range items {
		if normalizedKeyword != "" && !strings.Contains(strings.ToLower(item.ModelName), normalizedKeyword) {
			continue
		}
		if onlyFailed && item.FailCount == 0 {
			continue
		}
		if onlyWithLogs && !item.HasRealLogs {
			continue
		}
		filtered = append(filtered, item)
	}

	return filtered
}

func resolveSelectedGroup(groups []VisibleGroup, requestedGroup string) string {
	if len(groups) == 0 {
		return ""
	}

	requestedGroup = strings.TrimSpace(requestedGroup)
	if requestedGroup != "" {
		for _, group := range groups {
			if group.Name == requestedGroup {
				return requestedGroup
			}
		}
	}

	return groups[0].Name
}

func buildModelAvailabilityItem(modelName string, entries []availabilityLogEntry, probe *ProbeStatus) ModelAvailabilityItem {
	item := ModelAvailabilityItem{
		ModelName:       modelName,
		ConfigAvailable: true,
		RecentResults:   buildRecentResultsWindow(entries, modelAvailabilityRecentWindow),
		HasRealLogs:     len(entries) > 0,
		Probe:           probe,
	}

	for _, entry := range entries {
		if entry.Success {
			item.SuccessCount++
			continue
		}
		item.FailCount++
	}

	return item
}

func LoadGroupAvailabilitySnapshot(userID int, role int, requestedGroup string, keyword string, onlyFailed bool, onlyWithLogs bool) (GroupAvailabilitySnapshot, bool, error) {
	groups, err := listVisibleAvailabilityGroups(userID, role)
	if err != nil {
		return GroupAvailabilitySnapshot{}, false, err
	}

	selectedGroup := resolveSelectedGroup(groups, requestedGroup)
	if selectedGroup == "" {
		return buildGroupAvailabilitySnapshot(groups, "", groupAvailabilityCacheValue{}, keyword, onlyFailed, onlyWithLogs), false, nil
	}

	cachedValue, found, cacheErr := loadGroupAvailabilityCache(selectedGroup)
	if cacheErr == nil && found && isGroupAvailabilitySnapshotFresh(cachedValue.RefreshedAt) {
		return buildGroupAvailabilitySnapshot(groups, selectedGroup, cachedValue, keyword, onlyFailed, onlyWithLogs), false, nil
	}

	rebuiltValue, rebuildErr := rebuildGroupAvailabilityCacheValue(selectedGroup)
	if rebuildErr == nil {
		if saveErr := saveGroupAvailabilityCache(rebuiltValue); saveErr != nil {
			common.SysError("save model availability snapshot cache failed: " + saveErr.Error())
		}
		return buildGroupAvailabilitySnapshot(groups, selectedGroup, rebuiltValue, keyword, onlyFailed, onlyWithLogs), false, nil
	}

	if cacheErr == nil && found {
		return buildGroupAvailabilitySnapshot(groups, selectedGroup, cachedValue, keyword, onlyFailed, onlyWithLogs), true, nil
	}

	if cacheErr != nil {
		return GroupAvailabilitySnapshot{}, false, cacheErr
	}
	return GroupAvailabilitySnapshot{}, false, rebuildErr
}

func RefreshGroupAvailabilitySnapshot(userID int, role int, requestedGroup string) (GroupAvailabilitySnapshot, error) {
	groups, err := listVisibleAvailabilityGroups(userID, role)
	if err != nil {
		return GroupAvailabilitySnapshot{}, err
	}

	selectedGroup := strings.TrimSpace(requestedGroup)
	if selectedGroup == "" {
		return GroupAvailabilitySnapshot{}, nil
	}
	if !hasVisibleGroup(groups, selectedGroup) {
		return GroupAvailabilitySnapshot{}, ErrModelAvailabilityGroupNotAccessible
	}

	rebuiltValue, err := rebuildGroupAvailabilityCacheValue(selectedGroup)
	if err != nil {
		return GroupAvailabilitySnapshot{}, err
	}
	if err := saveGroupAvailabilityCache(rebuiltValue); err != nil {
		return GroupAvailabilitySnapshot{}, err
	}

	return buildGroupAvailabilitySnapshot(groups, selectedGroup, rebuiltValue, "", false, false), nil
}

func buildGroupAvailabilitySnapshot(groups []VisibleGroup, selectedGroup string, cacheValue groupAvailabilityCacheValue, keyword string, onlyFailed bool, onlyWithLogs bool) GroupAvailabilitySnapshot {
	markedGroups := make([]VisibleGroup, 0, len(groups))
	for _, group := range groups {
		group.Selected = group.Name == selectedGroup
		markedGroups = append(markedGroups, group)
	}

	items := filterAvailabilityItems(cacheValue.Items, keyword, onlyFailed, onlyWithLogs)

	return GroupAvailabilitySnapshot{
		Groups:        markedGroups,
		SelectedGroup: selectedGroup,
		RefreshedAt:   cacheValue.RefreshedAt,
		Items:         items,
	}
}

func listVisibleAvailabilityGroups(userID int, role int) ([]VisibleGroup, error) {
	groupNames, err := model.GetEnabledAbilityGroups()
	if err != nil {
		return nil, err
	}

	groupNames = normalizeAndSortNames(groupNames)
	if role < common.RoleAdminUser {
		userGroup, err := model.GetUserGroup(userID, false)
		if err != nil {
			return nil, err
		}
		allowedGroups := GetUserUsableGroups(userGroup)
		filteredNames := make([]string, 0, len(groupNames))
		for _, groupName := range groupNames {
			if _, ok := allowedGroups[groupName]; ok {
				filteredNames = append(filteredNames, groupName)
			}
		}
		groupNames = filteredNames
	}

	groups := make([]VisibleGroup, 0, len(groupNames))
	for _, groupName := range groupNames {
		groups = append(groups, VisibleGroup{Name: groupName})
	}
	return groups, nil
}

func rebuildGroupAvailabilityCacheValue(group string) (groupAvailabilityCacheValue, error) {
	group = strings.TrimSpace(group)
	if group == "" {
		return groupAvailabilityCacheValue{}, nil
	}

	modelNames := normalizeAndSortNames(model.GetGroupEnabledModels(group))
	logRows, err := model.GetRecentAvailabilityLogsByGroup(group, modelAvailabilityLogScanLimit)
	if err != nil {
		return groupAvailabilityCacheValue{}, err
	}

	logsByModel := buildAvailabilityLogsByModel(logRows)
	items := make([]ModelAvailabilityItem, 0, len(modelNames))
	for _, modelName := range modelNames {
		probe, _, err := LoadModelAvailabilityProbe(group, modelName)
		if err != nil {
			common.SysError("load model availability probe cache failed: " + err.Error())
			probe = nil
		}
		items = append(items, buildModelAvailabilityItem(modelName, logsByModel[modelName], probe))
	}

	return groupAvailabilityCacheValue{
		Group:       group,
		RefreshedAt: time.Now().Unix(),
		Items:       items,
	}, nil
}

func buildAvailabilityLogsByModel(rows []model.AvailabilityLogRow) map[string][]availabilityLogEntry {
	logsByModel := make(map[string][]availabilityLogEntry)
	for _, row := range rows {
		modelName := strings.TrimSpace(row.ModelName)
		if modelName == "" {
			continue
		}
		if len(logsByModel[modelName]) >= modelAvailabilityRecentWindow {
			continue
		}
		logsByModel[modelName] = append(logsByModel[modelName], availabilityLogEntry{
			ModelName: modelName,
			Success:   row.Type == model.LogTypeConsume,
		})
	}
	return logsByModel
}

func hasVisibleGroup(groups []VisibleGroup, selectedGroup string) bool {
	selectedGroup = strings.TrimSpace(selectedGroup)
	if selectedGroup == "" {
		return false
	}
	for _, group := range groups {
		if group.Name == selectedGroup {
			return true
		}
	}
	return false
}

func normalizeAndSortNames(names []string) []string {
	if len(names) == 0 {
		return []string{}
	}

	seen := make(map[string]struct{}, len(names))
	normalized := make([]string, 0, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		normalized = append(normalized, name)
	}
	sort.Strings(normalized)
	return normalized
}
