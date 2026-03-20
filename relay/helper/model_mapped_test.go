package helper

import (
	"net/http"
	"net/http/httptest"
	"testing"

	appconstant "github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestModelMappedHelperLegacyStringMappingKeepsCurrentBehavior(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	ctx.Set("model_mapping", `{"gpt-4o-mini":"gpt-4.1-mini"}`)

	request := &dto.GeneralOpenAIRequest{Model: "gpt-4o-mini"}
	info := &relaycommon.RelayInfo{
		RelayMode:       relayconstant.RelayModeChatCompletions,
		RequestURLPath:  "/v1/chat/completions",
		OriginModelName: "gpt-4o-mini",
		RelayFormat:     "openai",
		ChannelMeta: &relaycommon.ChannelMeta{
			ApiType: appconstant.APITypeOpenAI,
		},
	}

	err := ModelMappedHelper(ctx, info, request)
	require.NoError(t, err)
	require.True(t, info.IsModelMapped)
	require.Equal(t, "gpt-4.1-mini", info.UpstreamModelName)
	require.Equal(t, "gpt-4o-mini", info.OriginModelName)
	require.Equal(t, "/v1/chat/completions", info.RequestURLPath)
	require.Equal(t, relayconstant.RelayModeChatCompletions, info.RelayMode)
	require.Equal(t, "gpt-4.1-mini", request.Model)
}

func TestModelMappedHelperSupportsCompactAliasEndpointOverride(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/responses/compact", nil)
	ctx.Set("model_mapping", `{
		"gpt-5.4-openai-compact":{
			"target_model":"gpt-5.4",
			"endpoint_type":"openai-response"
		}
	}`)

	request := &dto.OpenAIResponsesCompactionRequest{Model: "gpt-5.4-openai-compact"}
	info := &relaycommon.RelayInfo{
		RelayMode:       relayconstant.RelayModeResponsesCompact,
		RequestURLPath:  "/v1/responses/compact",
		OriginModelName: "gpt-5.4-openai-compact",
		RelayFormat:     "openai_responses_compaction",
		ChannelMeta: &relaycommon.ChannelMeta{
			ApiType: appconstant.APITypeOpenAI,
		},
	}

	err := ModelMappedHelper(ctx, info, request)
	require.NoError(t, err)
	require.True(t, info.IsModelMapped)
	require.Equal(t, "gpt-5.4", info.UpstreamModelName)
	require.Equal(t, "gpt-5.4-openai-compact", info.OriginModelName)
	require.Equal(t, "/v1/responses", info.RequestURLPath)
	require.Equal(t, relayconstant.RelayModeResponses, info.RelayMode)
	require.Equal(t, "gpt-5.4", request.Model)
}

func TestModelMappedHelperCompactFallsBackToBaseModelMapping(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/responses/compact", nil)
	ctx.Set("model_mapping", `{"gpt-5.4":"gpt-5.4-mini"}`)

	request := &dto.OpenAIResponsesCompactionRequest{Model: "gpt-5.4-openai-compact"}
	info := &relaycommon.RelayInfo{
		RelayMode:       relayconstant.RelayModeResponsesCompact,
		RequestURLPath:  "/v1/responses/compact",
		OriginModelName: "gpt-5.4-openai-compact",
		RelayFormat:     "openai_responses_compaction",
		ChannelMeta: &relaycommon.ChannelMeta{
			ApiType: appconstant.APITypeOpenAI,
		},
	}

	err := ModelMappedHelper(ctx, info, request)
	require.NoError(t, err)
	require.True(t, info.IsModelMapped)
	require.Equal(t, "gpt-5.4-mini", info.UpstreamModelName)
	require.Equal(t, ratio_setting.WithCompactModelSuffix("gpt-5.4-mini"), info.OriginModelName)
	require.Equal(t, "gpt-5.4-mini", request.Model)
}

func TestModelMappedHelperRejectsIncompatibleEndpointOverride(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/responses/compact", nil)
	ctx.Set("model_mapping", `{
		"gpt-5.4-openai-compact":{
			"target_model":"claude-sonnet-4-5",
			"endpoint_type":"anthropic"
		}
	}`)

	request := &dto.OpenAIResponsesCompactionRequest{Model: "gpt-5.4-openai-compact"}
	info := &relaycommon.RelayInfo{
		RelayMode:       relayconstant.RelayModeResponsesCompact,
		RequestURLPath:  "/v1/responses/compact",
		OriginModelName: "gpt-5.4-openai-compact",
		RelayFormat:     "openai_responses_compaction",
		ChannelMeta: &relaycommon.ChannelMeta{
			ApiType: appconstant.APITypeOpenAI,
		},
	}

	err := ModelMappedHelper(ctx, info, request)
	require.Error(t, err)
	require.Contains(t, err.Error(), "endpoint override")
	require.Contains(t, err.Error(), "anthropic")
}

func TestModelMappedHelperAllowsSameModelEndpointOverride(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	ctx.Set("model_mapping", `{
		"gpt-5.4":{
			"target_model":"gpt-5.4",
			"endpoint_type":"openai-response"
		}
	}`)

	request := &dto.GeneralOpenAIRequest{Model: "gpt-5.4"}
	info := &relaycommon.RelayInfo{
		RelayMode:       relayconstant.RelayModeChatCompletions,
		RequestURLPath:  "/v1/chat/completions",
		OriginModelName: "gpt-5.4",
		RelayFormat:     "openai",
		ChannelMeta: &relaycommon.ChannelMeta{
			ApiType: appconstant.APITypeOpenAI,
		},
	}

	err := ModelMappedHelper(ctx, info, request)
	require.NoError(t, err)
	require.False(t, info.IsModelMapped)
	require.Equal(t, "gpt-5.4", info.UpstreamModelName)
	require.Equal(t, "/v1/responses", info.RequestURLPath)
	require.Equal(t, relayconstant.RelayModeResponses, info.RelayMode)
	require.Equal(t, "gpt-5.4", request.Model)
}
