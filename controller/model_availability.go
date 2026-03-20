package controller

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
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
