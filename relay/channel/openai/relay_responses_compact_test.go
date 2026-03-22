package openai

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/dto"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestResponsesCompactBodyPreviewTruncates(t *testing.T) {
	t.Parallel()

	body := bytes.Repeat([]byte("a"), responsesCompactBodyPreviewLimit+32)
	preview := responsesCompactBodyPreview(body)

	require.Len(t, preview, responsesCompactBodyPreviewLimit+len("...(truncated)"))
	require.Contains(t, preview, "...(truncated)")
}

func TestOaiResponsesCompactionHandlerLogsParseDiagnostics(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/responses/compact", nil)

	oldWriter := gin.DefaultErrorWriter
	var logBuf bytes.Buffer
	gin.DefaultErrorWriter = &logBuf
	t.Cleanup(func() {
		gin.DefaultErrorWriter = oldWriter
	})

	body := []byte("event: response.completed\ndata: {\"type\":\"response.completed\"}\n\n")
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header: http.Header{
			"Content-Type": []string{"text/event-stream"},
		},
		Body: io.NopCloser(bytes.NewReader(body)),
	}

	usage, newAPIError := OaiResponsesCompactionHandler(ctx, resp)
	require.Nil(t, usage)
	require.NotNil(t, newAPIError)
	require.Contains(t, newAPIError.Error(), "invalid character 'e'")
	require.Contains(t, newAPIError.UpstreamErrorBody, "event:")
	require.Contains(t, newAPIError.UpstreamErrorBody, ".completed")

	logOutput := logBuf.String()
	require.Contains(t, logOutput, "responses compact parse failed")
	require.Contains(t, logOutput, `content_type="text/event-stream"`)
	require.Contains(t, logOutput, `body_preview="event:`)
	require.Contains(t, logOutput, `.completed`)
}

func TestOaiResponsesToCompactHandlerConvertsJSONResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/responses/compact", nil)

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header: http.Header{
			"Content-Type": []string{"application/json"},
		},
		Body: io.NopCloser(bytes.NewBufferString(`{
			"id":"resp_123",
			"object":"response",
			"created_at":123,
			"output":[{"type":"message","id":"msg_1","status":"completed","role":"assistant","content":[{"type":"output_text","text":"hello","annotations":[]}]}],
			"usage":{"input_tokens":11,"output_tokens":7,"total_tokens":18}
		}`)),
	}

	usage, newAPIError := OaiResponsesToCompactHandler(ctx, resp)
	require.Nil(t, newAPIError)
	require.NotNil(t, usage)
	require.Equal(t, "application/json", recorder.Header().Get("Content-Type"))

	var compactResp dto.OpenAIResponsesCompactionResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &compactResp))
	require.Equal(t, "resp_123", compactResp.ID)
	require.Equal(t, 123, compactResp.CreatedAt)
	require.Contains(t, string(compactResp.Output), `"type":"message"`)
	require.Contains(t, string(compactResp.Output), `"text":"hello"`)
	require.NotNil(t, compactResp.Usage)
	require.Equal(t, 11, compactResp.Usage.InputTokens)
}

func TestOaiResponsesToCompactHandlerConvertsCompletedSSEToCompactJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/responses/compact", nil)

	body := "event: response.output_text.delta\n" +
		"data: {\"type\":\"response.output_text.delta\",\"delta\":\"hel\"}\n\n" +
		"event: response.completed\n" +
		"data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_456\",\"object\":\"response\",\"created_at\":456,\"output\":[{\"type\":\"message\",\"id\":\"msg_1\",\"status\":\"completed\",\"role\":\"assistant\",\"content\":[{\"type\":\"output_text\",\"text\":\"hello\",\"annotations\":[]}]}],\"usage\":{\"input_tokens\":21,\"output_tokens\":9,\"total_tokens\":30}}}\n\n" +
		"data: [DONE]\n\n"
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header: http.Header{
			"Content-Type": []string{"text/event-stream"},
		},
		Body: io.NopCloser(bytes.NewBufferString(body)),
	}

	usage, newAPIError := OaiResponsesToCompactHandler(ctx, resp)
	require.Nil(t, newAPIError)
	require.NotNil(t, usage)
	require.Equal(t, 21, usage.PromptTokens)
	require.Equal(t, 9, usage.CompletionTokens)
	require.Equal(t, 30, usage.TotalTokens)

	var compactResp dto.OpenAIResponsesCompactionResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &compactResp))
	require.Equal(t, "resp_456", compactResp.ID)
	require.Contains(t, string(compactResp.Output), `"type":"message"`)
	require.Contains(t, string(compactResp.Output), `"text":"hello"`)
}
