package openai

import (
	"bufio"
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

const (
	responsesStreamScannerInitialBuffer = 64 << 10
	responsesStreamScannerMaxBuffer     = 64 << 20
)

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

func OaiResponsesToCompactHandler(c *gin.Context, resp *http.Response) (*dto.Usage, *types.NewAPIError) {
	if resp == nil || resp.Body == nil {
		return nil, types.NewError(fmt.Errorf("invalid response"), types.ErrorCodeBadResponse)
	}

	contentType := resp.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "text/event-stream") {
		return oaiResponsesStreamToCompactHandler(c, resp)
	}
	return oaiResponsesJSONToCompactHandler(c, resp)
}

func oaiResponsesJSONToCompactHandler(c *gin.Context, resp *http.Response) (*dto.Usage, *types.NewAPIError) {
	defer service.CloseResponseBodyGracefully(resp)

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeReadResponseBodyFailed, http.StatusInternalServerError)
	}

	var responsesResp dto.OpenAIResponsesResponse
	if err := common.Unmarshal(responseBody, &responsesResp); err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}
	if oaiError := responsesResp.GetOpenAIError(); oaiError != nil && oaiError.Type != "" {
		return nil, types.WithOpenAIError(*oaiError, resp.StatusCode)
	}

	return writeCompactResponse(c, &responsesResp)
}

func oaiResponsesStreamToCompactHandler(c *gin.Context, resp *http.Response) (*dto.Usage, *types.NewAPIError) {
	defer service.CloseResponseBodyGracefully(resp)

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, responsesStreamScannerInitialBuffer), responsesStreamScannerMaxBuffer)

	var completedResponse *dto.OpenAIResponsesResponse
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.HasPrefix(line, "data:") {
			continue
		}

		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" || data == "[DONE]" {
			continue
		}

		var streamResp dto.ResponsesStreamResponse
		if err := common.UnmarshalJsonStr(data, &streamResp); err != nil {
			logger.LogError(c, fmt.Sprintf("responses stream->compact parse failed: data=%q err=%v", responsesCompactBodyPreview([]byte(data)), err))
			continue
		}

		if streamResp.Type == "response.completed" && streamResp.Response != nil {
			completedResponse = streamResp.Response
		}
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		return nil, types.NewOpenAIError(err, types.ErrorCodeReadResponseBodyFailed, http.StatusInternalServerError)
	}
	if completedResponse == nil {
		return nil, types.NewOpenAIError(fmt.Errorf("stream disconnected before completion"), types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}
	if oaiError := completedResponse.GetOpenAIError(); oaiError != nil && oaiError.Type != "" {
		return nil, types.WithOpenAIError(*oaiError, resp.StatusCode)
	}

	return writeCompactResponse(c, completedResponse)
}

func writeCompactResponse(c *gin.Context, responsesResp *dto.OpenAIResponsesResponse) (*dto.Usage, *types.NewAPIError) {
	compactResp, err := buildResponsesCompactionResponse(responsesResp)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}

	responseBody, err := common.Marshal(compactResp)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}

	c.Writer.Header().Set("Content-Type", "application/json")
	service.IOCopyBytesGracefully(c, nil, responseBody)

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

func buildResponsesCompactionResponse(responsesResp *dto.OpenAIResponsesResponse) (*dto.OpenAIResponsesCompactionResponse, error) {
	if responsesResp == nil {
		return nil, fmt.Errorf("responses response is nil")
	}

	output, err := common.Marshal(responsesResp.Output)
	if err != nil {
		return nil, err
	}

	return &dto.OpenAIResponsesCompactionResponse{
		ID:        responsesResp.ID,
		Object:    responsesResp.Object,
		CreatedAt: responsesResp.CreatedAt,
		Output:    output,
		Usage:     responsesResp.Usage,
		Error:     responsesResp.Error,
	}, nil
}

func responsesCompactBodyPreview(body []byte) string {
	preview := strings.TrimSpace(common.MaskSensitiveInfo(string(body)))
	if len(preview) <= responsesCompactBodyPreviewLimit {
		return preview
	}
	return preview[:responsesCompactBodyPreviewLimit] + "...(truncated)"
}
