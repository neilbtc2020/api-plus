package service

import "strings"

const (
	availabilityResultSuccess = "success"
	availabilityResultFail    = "fail"
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
