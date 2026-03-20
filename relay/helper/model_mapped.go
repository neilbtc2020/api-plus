package helper

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	appcommon "github.com/QuantumNous/new-api/common"
	appconstant "github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
)

type modelMappingRule struct {
	TargetModel  string                   `json:"target_model"`
	EndpointType appconstant.EndpointType `json:"endpoint_type,omitempty"`
}

func ModelMappedHelper(c *gin.Context, info *common.RelayInfo, request dto.Request) error {
	if info.ChannelMeta == nil {
		info.ChannelMeta = &common.ChannelMeta{}
	}

	originalRelayMode := info.RelayMode
	isResponsesCompact := originalRelayMode == relayconstant.RelayModeResponsesCompact
	originModelName := info.OriginModelName
	defaultUpstreamModelName := originModelName
	if isResponsesCompact && strings.HasSuffix(defaultUpstreamModelName, ratio_setting.CompactModelSuffix) {
		defaultUpstreamModelName = strings.TrimSuffix(defaultUpstreamModelName, ratio_setting.CompactModelSuffix)
	}

	// map model name
	modelMapping := c.GetString("model_mapping")
	var endpointOverride appconstant.EndpointType
	if modelMapping != "" && modelMapping != "{}" {
		modelMap, err := parseModelMappingRules(modelMapping)
		if err != nil {
			return fmt.Errorf("unmarshal_model_mapping_failed")
		}

		// 支持链式模型重定向，最终使用链尾的模型
		currentModel := originModelName
		visitedModels := map[string]bool{
			currentModel: true,
		}
		for {
			rule, exists := findModelMappingRule(modelMap, currentModel, isResponsesCompact)
			if exists && rule.TargetModel != "" {
				mappedModel := rule.TargetModel
				// 模型重定向循环检测，避免无限循环
				if visitedModels[mappedModel] {
					if mappedModel == currentModel {
						if currentModel == info.OriginModelName {
							info.IsModelMapped = false
							if rule.EndpointType != "" {
								endpointOverride = rule.EndpointType
							}
							break
						} else {
							info.IsModelMapped = true
							break
						}
					}
					return errors.New("model_mapping_contains_cycle")
				}
				visitedModels[mappedModel] = true
				currentModel = mappedModel
				info.IsModelMapped = true
				if rule.EndpointType != "" {
					endpointOverride = rule.EndpointType
				}
			} else {
				break
			}
		}
		if info.IsModelMapped {
			info.UpstreamModelName = currentModel
		}
	}

	finalUpstreamModelName := defaultUpstreamModelName
	if info.IsModelMapped && info.UpstreamModelName != "" {
		finalUpstreamModelName = info.UpstreamModelName
	}

	if endpointOverride != "" {
		if err := applyEndpointOverride(info, request, endpointOverride); err != nil {
			return err
		}
	}

	if isResponsesCompact && info.RelayMode == relayconstant.RelayModeResponsesCompact {
		finalUpstreamModelName := defaultUpstreamModelName
		if info.IsModelMapped && info.UpstreamModelName != "" {
			finalUpstreamModelName = info.UpstreamModelName
		}
		info.UpstreamModelName = finalUpstreamModelName
		info.OriginModelName = ratio_setting.WithCompactModelSuffix(finalUpstreamModelName)
	} else {
		info.UpstreamModelName = finalUpstreamModelName
	}
	if request != nil {
		request.SetModelName(info.UpstreamModelName)
	}
	return nil
}

func parseModelMappingRules(raw string) (map[string]modelMappingRule, error) {
	rawRules := make(map[string]json.RawMessage)
	if err := json.Unmarshal([]byte(raw), &rawRules); err != nil {
		return nil, err
	}

	rules := make(map[string]modelMappingRule, len(rawRules))
	for sourceModel, rawRule := range rawRules {
		var targetModel string
		if err := json.Unmarshal(rawRule, &targetModel); err == nil {
			rules[sourceModel] = modelMappingRule{TargetModel: targetModel}
			continue
		}

		var rule modelMappingRule
		if err := json.Unmarshal(rawRule, &rule); err != nil {
			return nil, err
		}
		if rule.TargetModel == "" {
			return nil, fmt.Errorf("target_model is required for %q", sourceModel)
		}
		rules[sourceModel] = rule
	}
	return rules, nil
}

func findModelMappingRule(modelMap map[string]modelMappingRule, currentModel string, isResponsesCompact bool) (modelMappingRule, bool) {
	if rule, exists := modelMap[currentModel]; exists {
		return rule, true
	}
	if isResponsesCompact && strings.HasSuffix(currentModel, ratio_setting.CompactModelSuffix) {
		trimmedModel := strings.TrimSuffix(currentModel, ratio_setting.CompactModelSuffix)
		if rule, exists := modelMap[trimmedModel]; exists {
			return rule, true
		}
	}
	return modelMappingRule{}, false
}

func applyEndpointOverride(info *common.RelayInfo, request dto.Request, endpointType appconstant.EndpointType) error {
	if !isRequestCompatibleWithEndpoint(request, endpointType) {
		return fmt.Errorf("endpoint override %q is incompatible with request type %T", endpointType, request)
	}

	endpointInfo, ok := appcommon.GetDefaultEndpointInfo(endpointType)
	if !ok {
		return fmt.Errorf("unsupported endpoint override %q", endpointType)
	}

	info.RequestURLPath = endpointInfo.Path
	info.RelayMode = relayconstant.Path2RelayMode(endpointInfo.Path)
	if relayFormat, ok := relayFormatForEndpointType(endpointType); ok {
		info.FinalRequestRelayFormat = relayFormat
	}
	return nil
}

func isRequestCompatibleWithEndpoint(request dto.Request, endpointType appconstant.EndpointType) bool {
	switch request.(type) {
	case *dto.GeneralOpenAIRequest:
		return endpointType == appconstant.EndpointTypeOpenAI ||
			endpointType == appconstant.EndpointTypeOpenAIResponse
	case *dto.OpenAIResponsesRequest:
		return endpointType == appconstant.EndpointTypeOpenAIResponse
	case *dto.OpenAIResponsesCompactionRequest:
		return endpointType == appconstant.EndpointTypeOpenAIResponse || endpointType == appconstant.EndpointTypeOpenAIResponseCompact
	case *dto.ClaudeRequest:
		return endpointType == appconstant.EndpointTypeAnthropic
	case *dto.GeminiChatRequest, *dto.GeminiEmbeddingRequest, *dto.GeminiBatchEmbeddingRequest:
		return endpointType == appconstant.EndpointTypeGemini
	case *dto.EmbeddingRequest:
		return endpointType == appconstant.EndpointTypeEmbeddings
	case *dto.RerankRequest:
		return endpointType == appconstant.EndpointTypeJinaRerank
	case *dto.ImageRequest:
		return endpointType == appconstant.EndpointTypeImageGeneration
	default:
		return false
	}
}

func relayFormatForEndpointType(endpointType appconstant.EndpointType) (types.RelayFormat, bool) {
	switch endpointType {
	case appconstant.EndpointTypeOpenAI:
		return types.RelayFormatOpenAI, true
	case appconstant.EndpointTypeOpenAIResponse:
		return types.RelayFormatOpenAIResponses, true
	case appconstant.EndpointTypeOpenAIResponseCompact:
		return types.RelayFormatOpenAIResponsesCompaction, true
	case appconstant.EndpointTypeAnthropic:
		return types.RelayFormatClaude, true
	case appconstant.EndpointTypeGemini:
		return types.RelayFormatGemini, true
	case appconstant.EndpointTypeEmbeddings:
		return types.RelayFormatEmbedding, true
	case appconstant.EndpointTypeImageGeneration:
		return types.RelayFormatOpenAIImage, true
	case appconstant.EndpointTypeJinaRerank:
		return types.RelayFormatRerank, true
	default:
		return "", false
	}
}
