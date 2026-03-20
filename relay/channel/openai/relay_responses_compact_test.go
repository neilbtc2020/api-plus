package openai

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

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
