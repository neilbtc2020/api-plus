package model

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
)

type AvailabilityLogRow struct {
	Id        int    `json:"id"`
	Type      int    `json:"type"`
	ModelName string `json:"model_name"`
	CreatedAt int64  `json:"created_at"`
}

func GetRecentAvailabilityLogsByGroup(group string, limit int) ([]AvailabilityLogRow, error) {
	group = strings.TrimSpace(group)
	if group == "" || limit <= 0 {
		return []AvailabilityLogRow{}, nil
	}

	rows := make([]AvailabilityLogRow, 0, limit)
	err := LOG_DB.Model(&Log{}).
		Select("id, type, model_name, created_at").
		Where("logs."+logGroupCol+" = ? AND logs.type IN ?", group, []int{LogTypeConsume, LogTypeError}).
		Order("logs.id DESC").
		Limit(limit).
		Find(&rows).Error

	return rows, err
}

func GetEnabledAbilityGroups() ([]string, error) {
	groups := make([]string, 0)
	err := DB.Table("abilities").
		Where("enabled = ?", true).
		Distinct(commonGroupCol).
		Pluck("group", &groups).Error
	return groups, err
}

func GetEnabledChannelsByGroupModel(group string, modelName string) ([]*Channel, error) {
	group = strings.TrimSpace(group)
	modelName = strings.TrimSpace(modelName)
	if group == "" || modelName == "" {
		return []*Channel{}, nil
	}

	channels := make([]*Channel, 0)
	err := DB.Table("channels").
		Select("channels.*").
		Joins("JOIN abilities ON abilities.channel_id = channels.id").
		Where("abilities."+commonGroupCol+" = ? AND abilities.model = ? AND abilities.enabled = ? AND channels.status = ?", group, modelName, true, common.ChannelStatusEnabled).
		Order("channels.priority DESC, channels.id DESC").
		Find(&channels).Error
	return channels, err
}
