package controller

import (
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

type availabilityGroupView struct {
	Name     string `json:"name"`
	Selected bool   `json:"selected"`
}

type modelAvailabilityResponseData struct {
	Groups        []availabilityGroupView         `json:"groups"`
	SelectedGroup string                          `json:"selected_group"`
	RefreshedAt   int64                           `json:"refreshed_at"`
	Items         []service.ModelAvailabilityItem `json:"items"`
	Warning       string                          `json:"warning,omitempty"`
}

type refreshModelAvailabilityRequest struct {
	Group string `json:"group"`
}

type probeModelAvailabilityRequest struct {
	Group     string `json:"group"`
	ModelName string `json:"model_name"`
}

type probeAttemptSummary struct {
	ChannelID      int
	Success        bool
	Message        string
	ResponseTimeMs int64
}

func GetModelAvailability(c *gin.Context) {
	snapshot, stale, err := service.LoadGroupAvailabilitySnapshot(
		c.GetInt("id"),
		c.GetInt("role"),
		c.Query("group"),
		c.Query("keyword"),
		c.Query("only_failed") == "true",
		c.Query("only_with_logs") == "true",
	)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	common.ApiSuccess(c, withSnapshotWarning(buildModelAvailabilityResponse(snapshot), stale))
}

func RefreshModelAvailability(c *gin.Context) {
	var req refreshModelAvailabilityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	if strings.TrimSpace(req.Group) == "" {
		common.ApiErrorMsg(c, "group is required")
		return
	}

	snapshot, err := service.RefreshGroupAvailabilitySnapshot(c.GetInt("id"), c.GetInt("role"), req.Group)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	common.ApiSuccess(c, buildModelAvailabilityResponse(snapshot))
}

func ProbeModelAvailability(c *gin.Context) {
	var req probeModelAvailabilityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	req.Group = strings.TrimSpace(req.Group)
	req.ModelName = strings.TrimSpace(req.ModelName)
	if req.Group == "" {
		common.ApiErrorMsg(c, "group is required")
		return
	}
	if req.ModelName == "" {
		common.ApiErrorMsg(c, "model_name is required")
		return
	}

	snapshot, _, err := service.LoadGroupAvailabilitySnapshot(c.GetInt("id"), c.GetInt("role"), req.Group, "", false, false)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if snapshot.SelectedGroup != req.Group {
		common.ApiError(c, service.ErrModelAvailabilityGroupNotAccessible)
		return
	}

	channels, err := model.GetEnabledChannelsByGroupModel(req.Group, req.ModelName)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	attempts := make([]probeAttemptSummary, 0, len(channels))
	for _, channel := range channels {
		tik := time.Now()
		result := testChannel(channel, req.ModelName, "", false)
		attempt := probeAttemptSummary{
			ChannelID:      channel.Id,
			ResponseTimeMs: time.Since(tik).Milliseconds(),
		}

		if result.localErr != nil {
			attempt.Message = result.localErr.Error()
			attempts = append(attempts, attempt)
			continue
		}
		if result.newAPIError != nil {
			attempt.Message = result.newAPIError.Error()
			attempts = append(attempts, attempt)
			continue
		}

		attempt.Success = true
		attempt.Message = "检测成功"
		attempts = append(attempts, attempt)
		channel.UpdateResponseTime(attempt.ResponseTimeMs)
		if result.endpointType != "" && result.testModel != "" {
			model.SetChannelEndpoint(channel.Id, result.testModel, result.endpointType)
		}
		break
	}

	probeStatus := summarizeProbeAttempts(attempts)
	if err := service.SaveModelAvailabilityProbe(req.Group, req.ModelName, probeStatus); err != nil {
		common.ApiError(c, err)
		return
	}
	if _, err := service.RefreshGroupAvailabilitySnapshot(c.GetInt("id"), c.GetInt("role"), req.Group); err != nil {
		common.ApiError(c, err)
		return
	}

	common.ApiSuccess(c, probeStatus)
}

func buildModelAvailabilityResponse(snapshot service.GroupAvailabilitySnapshot) modelAvailabilityResponseData {
	groups := make([]availabilityGroupView, 0, len(snapshot.Groups))
	for _, group := range snapshot.Groups {
		groups = append(groups, availabilityGroupView{
			Name:     group.Name,
			Selected: group.Selected,
		})
	}

	return modelAvailabilityResponseData{
		Groups:        markSelectedGroup(groups, snapshot.SelectedGroup),
		SelectedGroup: snapshot.SelectedGroup,
		RefreshedAt:   snapshot.RefreshedAt,
		Items:         snapshot.Items,
	}
}

func markSelectedGroup(groups []availabilityGroupView, selectedGroup string) []availabilityGroupView {
	selectedGroup = strings.TrimSpace(selectedGroup)
	marked := make([]availabilityGroupView, 0, len(groups))
	for _, group := range groups {
		group.Selected = group.Name == selectedGroup
		marked = append(marked, group)
	}
	return marked
}

func withSnapshotWarning(data modelAvailabilityResponseData, stale bool) modelAvailabilityResponseData {
	if stale {
		data.Warning = "数据可能已过期，请稍后手动刷新"
	}
	return data
}

func summarizeProbeAttempts(attempts []probeAttemptSummary) service.ProbeStatus {
	result := service.ProbeStatus{
		Status:    "fail",
		CheckedAt: time.Now().Unix(),
		Message:   "未找到可用渠道",
	}

	for _, attempt := range attempts {
		if attempt.Success {
			message := strings.TrimSpace(attempt.Message)
			if message == "" {
				message = "检测成功"
			}
			return service.ProbeStatus{
				Status:         "success",
				CheckedAt:      result.CheckedAt,
				Message:        message,
				ResponseTimeMs: attempt.ResponseTimeMs,
			}
		}
		if strings.TrimSpace(attempt.Message) != "" {
			result.Message = strings.TrimSpace(attempt.Message)
		}
	}

	return result
}
