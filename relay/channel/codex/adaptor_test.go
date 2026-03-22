package codex

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestDoResponseTreatsEventStreamAsResponsesStream(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldStreamingTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 1
	t.Cleanup(func() {
		constant.StreamingTimeout = oldStreamingTimeout
	})

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header: http.Header{
			"Content-Type": []string{"text/event-stream"},
		},
		Body: io.NopCloser(bytes.NewBufferString("data: {\"type\":\"response.completed\",\"response\":{\"usage\":{\"input_tokens\":11,\"output_tokens\":7,\"total_tokens\":18}}}\n\ndata: [DONE]\n\n")),
	}

	info := &relaycommon.RelayInfo{
		RelayMode: relayconstant.RelayModeResponses,
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "gpt-5.4",
		},
	}

	usage, newAPIError := (&Adaptor{}).DoResponse(ctx, resp, info)
	require.Nil(t, newAPIError)

	usageDTO, ok := usage.(*dto.Usage)
	require.True(t, ok)
	require.Equal(t, 11, usageDTO.PromptTokens)
	require.Equal(t, 7, usageDTO.CompletionTokens)
	require.Equal(t, 18, usageDTO.TotalTokens)
	require.Contains(t, recorder.Body.String(), "response.completed")
}

func TestDoResponseConvertsResponsesSSEBackToCompactJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/responses/compact", nil)

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header: http.Header{
			"Content-Type": []string{"text/event-stream"},
		},
		Body: io.NopCloser(bytes.NewBufferString("event: response.completed\ndata: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_789\",\"object\":\"response\",\"created_at\":789,\"output\":[{\"type\":\"message\",\"id\":\"msg_1\",\"status\":\"completed\",\"role\":\"assistant\",\"content\":[{\"type\":\"output_text\",\"text\":\"hello\",\"annotations\":[]}]}],\"usage\":{\"input_tokens\":11,\"output_tokens\":7,\"total_tokens\":18}}}\n\ndata: [DONE]\n\n")),
	}

	info := &relaycommon.RelayInfo{
		RelayMode:   relayconstant.RelayModeResponses,
		RelayFormat: types.RelayFormatOpenAIResponsesCompaction,
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "gpt-5.4",
		},
	}

	usage, newAPIError := (&Adaptor{}).DoResponse(ctx, resp, info)
	require.Nil(t, newAPIError)

	usageDTO, ok := usage.(*dto.Usage)
	require.True(t, ok)
	require.Equal(t, 18, usageDTO.TotalTokens)
	require.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
	require.Contains(t, recorder.Body.String(), "\"output\":[")
	require.NotContains(t, recorder.Body.String(), "event: response.completed")
}
