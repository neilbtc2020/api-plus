package openai

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

const responsesCompactBodyPreviewLimit = 256

func OaiResponsesCompactionHandler(c *gin.Context, resp *http.Response) (*dto.Usage, *types.NewAPIError) {
	defer service.CloseResponseBodyGracefully(resp)

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeReadResponseBodyFailed, http.StatusInternalServerError)
	}

	var compactResp dto.OpenAIResponsesCompactionResponse
	if err := common.Unmarshal(responseBody, &compactResp); err != nil {
		contentType := ""
		if resp != nil {
			contentType = resp.Header.Get("Content-Type")
		}
		logger.LogError(c, fmt.Sprintf(
			"responses compact parse failed: content_type=%q body_preview=%q err=%v",
			contentType,
			responsesCompactBodyPreview(responseBody),
			err,
		))
		newAPIError := types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
		newAPIError.SetUpstreamErrorBody(responseBody)
		return nil, newAPIError
	}
	if oaiError := compactResp.GetOpenAIError(); oaiError != nil && oaiError.Type != "" {
		return nil, types.WithOpenAIError(*oaiError, resp.StatusCode)
	}

	service.IOCopyBytesGracefully(c, resp, responseBody)

	usage := dto.Usage{}
	if compactResp.Usage != nil {
		usage.PromptTokens = compactResp.Usage.InputTokens
		usage.CompletionTokens = compactResp.Usage.OutputTokens
		usage.TotalTokens = compactResp.Usage.TotalTokens
		if compactResp.Usage.InputTokensDetails != nil {
			usage.PromptTokensDetails.CachedTokens = compactResp.Usage.InputTokensDetails.CachedTokens
		}
	}

	return &usage, nil
}

func responsesCompactBodyPreview(body []byte) string {
	preview := strings.TrimSpace(common.MaskSensitiveInfo(string(body)))
	if len(preview) <= responsesCompactBodyPreviewLimit {
		return preview
	}
	return preview[:responsesCompactBodyPreviewLimit] + "...(truncated)"
}
