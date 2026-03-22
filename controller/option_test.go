package controller

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestUpdateOptionRejectsInvalidChannelSecurityRulesBeforePersist(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	body, err := common.Marshal(OptionUpdateRequest{
		Key:   "ChannelSecurityRules",
		Value: `{"bad":true}`,
	})
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/option/", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	UpdateOption(c)

	require.Equal(t, http.StatusOK, recorder.Code)

	var response struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	err = common.Unmarshal(recorder.Body.Bytes(), &response)
	require.NoError(t, err)
	require.False(t, response.Success)
	require.NotEmpty(t, response.Message)
}
