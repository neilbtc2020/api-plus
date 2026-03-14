package controller

import (
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

type channelHealthTestAttempt struct {
	endpointType string
	stream       bool
}

type ChannelHealthItem struct {
	Id               int      `json:"id"`
	Name             string   `json:"name"`
	Type             int      `json:"type"`
	TypeName         string   `json:"type_name"`
	Status           int      `json:"status"`
	Group            string   `json:"group"`
	Tag              string   `json:"tag"`
	TestModel        string   `json:"test_model,omitempty"`
	IsMultiKey       bool     `json:"is_multi_key"`
	Available        bool     `json:"available"`
	AvailableMessage string   `json:"available_message"`
	ResponseTime     float64  `json:"response_time"`
	ResponseTimeMs   int64    `json:"response_time_ms"`
	BalanceSupported bool     `json:"balance_supported"`
	Balance          *float64 `json:"balance,omitempty"`
	BalanceMessage   string   `json:"balance_message"`
	CheckedAt        int64    `json:"checked_at"`
}

type ChannelHealthSummary struct {
	Total                 int `json:"total"`
	AvailableCount        int `json:"available_count"`
	UnavailableCount      int `json:"unavailable_count"`
	BalanceSupportedCount int `json:"balance_supported_count"`
	BalanceSuccessCount   int `json:"balance_success_count"`
	BalanceFailedCount    int `json:"balance_failed_count"`
	MultiKeyCount         int `json:"multi_key_count"`
}

func isBalanceCheckSupported(channel *model.Channel) bool {
	if channel == nil || channel.ChannelInfo.IsMultiKey {
		return false
	}
	switch channel.Type {
	case constant.ChannelTypeOpenAI,
		constant.ChannelTypeCustom,
		constant.ChannelTypeAIProxy,
		constant.ChannelTypeAPI2GPT,
		constant.ChannelTypeAIGC2D,
		constant.ChannelTypeSiliconFlow,
		constant.ChannelTypeDeepSeek,
		constant.ChannelTypeOpenRouter,
		constant.ChannelTypeMoonshot:
		return true
	default:
		return false
	}
}

func getChannelHealthTestModel(channel *model.Channel) string {
	if channel == nil {
		return ""
	}
	if channel.TestModel != nil && strings.TrimSpace(*channel.TestModel) != "" {
		return strings.TrimSpace(*channel.TestModel)
	}
	models := channel.GetModels()
	if len(models) > 0 && strings.TrimSpace(models[0]) != "" {
		return strings.TrimSpace(models[0])
	}
	return ""
}

func buildChannelHealthAttempts(channel *model.Channel, testModel string) []channelHealthTestAttempt {
	attempts := make([]channelHealthTestAttempt, 0, 6)
	seen := make(map[string]struct{})
	add := func(endpointType string, stream bool) {
		key := endpointType + "|" + strconv.FormatBool(stream)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		attempts = append(attempts, channelHealthTestAttempt{
			endpointType: endpointType,
			stream:       stream,
		})
	}

	if cachedEndpoint := model.GetChannelEndpoint(channel.Id, testModel); cachedEndpoint != "" {
		add(cachedEndpoint, false)
		add(cachedEndpoint, true)
	}

	add("", false)

	switch channel.Type {
	case constant.ChannelTypeOpenAI, constant.ChannelTypeCustom, constant.ChannelTypeCodex:
		add(string(constant.EndpointTypeOpenAIResponse), false)
		add(string(constant.EndpointTypeOpenAIResponse), true)
		add(string(constant.EndpointTypeOpenAI), false)
	}

	return attempts
}

func checkChannelHealth(channel *model.Channel) ChannelHealthItem {
	item := ChannelHealthItem{
		Id:               channel.Id,
		Name:             channel.Name,
		Type:             channel.Type,
		TypeName:         constant.GetChannelTypeName(channel.Type),
		Status:           channel.Status,
		Group:            channel.Group,
		Tag:              channel.GetTag(),
		IsMultiKey:       channel.ChannelInfo.IsMultiKey,
		Available:        false,
		BalanceSupported: false,
		CheckedAt:        time.Now().Unix(),
	}
	if channel.TestModel != nil {
		item.TestModel = *channel.TestModel
	}
	testModel := getChannelHealthTestModel(channel)
	if item.TestModel == "" {
		item.TestModel = testModel
	}

	var lastErrMsg string
	for _, attempt := range buildChannelHealthAttempts(channel, testModel) {
		tik := time.Now()
		testResult := testChannel(channel, testModel, attempt.endpointType, attempt.stream)
		item.ResponseTimeMs = time.Since(tik).Milliseconds()
		item.ResponseTime = float64(item.ResponseTimeMs) / 1000.0

		if testResult.localErr != nil {
			lastErrMsg = testResult.localErr.Error()
			continue
		}
		if testResult.newAPIError != nil {
			lastErrMsg = testResult.newAPIError.Error()
			continue
		}

		channel.UpdateResponseTime(item.ResponseTimeMs)
		item.Available = true
		if attempt.endpointType != "" {
			item.AvailableMessage = "检测成功（endpoint: " + attempt.endpointType + "）"
			model.SetChannelEndpoint(channel.Id, testResult.testModel, attempt.endpointType)
		} else {
			item.AvailableMessage = "检测成功"
		}
		break
	}
	if !item.Available {
		item.AvailableMessage = lastErrMsg
	}

	if !isBalanceCheckSupported(channel) {
		if channel.ChannelInfo.IsMultiKey {
			item.BalanceMessage = "多密钥渠道暂不支持额度查询"
		} else {
			item.BalanceMessage = "当前渠道类型暂不支持额度查询"
		}
		return item
	}

	item.BalanceSupported = true
	balance, err := updateChannelBalance(channel)
	if err != nil {
		item.BalanceMessage = err.Error()
		return item
	}
	item.Balance = &balance
	return item
}

func GetChannelHealth(c *gin.Context) {
	statusFilter := parseStatusFilter(c.Query("status"))
	typeFilter := -1
	if typeStr := c.Query("type"); typeStr != "" {
		if parsed, err := strconv.Atoi(typeStr); err == nil {
			typeFilter = parsed
		}
	}

	channels, err := model.GetAllChannels(0, 0, true, false)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	filteredChannels := make([]*model.Channel, 0, len(channels))
	for _, channel := range channels {
		if statusFilter == common.ChannelStatusEnabled && channel.Status != common.ChannelStatusEnabled {
			continue
		}
		if statusFilter == 0 && channel.Status == common.ChannelStatusEnabled {
			continue
		}
		if typeFilter >= 0 && channel.Type != typeFilter {
			continue
		}
		filteredChannels = append(filteredChannels, channel)
	}

	items := make([]ChannelHealthItem, len(filteredChannels))
	concurrency := 5
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	for idx, channel := range filteredChannels {
		wg.Add(1)
		go func(i int, ch *model.Channel) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() {
				<-sem
			}()
			items[i] = checkChannelHealth(ch)
		}(idx, channel)
	}
	wg.Wait()

	summary := ChannelHealthSummary{
		Total: len(items),
	}
	for _, item := range items {
		if item.IsMultiKey {
			summary.MultiKeyCount++
		}
		if item.Available {
			summary.AvailableCount++
		} else {
			summary.UnavailableCount++
		}
		if item.BalanceSupported {
			summary.BalanceSupportedCount++
			if item.Balance != nil {
				summary.BalanceSuccessCount++
			} else {
				summary.BalanceFailedCount++
			}
		}
	}

	common.ApiSuccess(c, gin.H{
		"items":   items,
		"summary": summary,
	})
}
