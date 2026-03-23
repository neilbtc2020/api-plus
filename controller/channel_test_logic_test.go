package controller

import (
	"errors"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"
	"github.com/stretchr/testify/require"
)

func TestEvaluateChannelAutoTestDetectsSecurityRisk(t *testing.T) {
	t.Parallel()

	channel := &model.Channel{Status: common.ChannelStatusEnabled}
	result := testResult{responseBody: []byte(`{"message":"<script>alert(1)</script>"}`)}

	decision := evaluateChannelAutoTest(channel, result, 800, 5000)

	require.True(t, decision.shouldSecurityDisable)
	require.False(t, decision.shouldDisable)
	require.False(t, decision.shouldEnable)
	require.NotNil(t, decision.securityScan)
	require.Equal(t, service.ChannelSecurityRiskLevelHighRisk, decision.securityScan.RiskLevel)
}

func TestEvaluateChannelAutoTestKeepsSecurityDisabledChannelDisabled(t *testing.T) {
	t.Parallel()

	channel := &model.Channel{Status: common.ChannelStatusSecurityDisabled}
	result := testResult{}

	decision := evaluateChannelAutoTest(channel, result, 800, 5000)

	require.False(t, decision.shouldEnable)
	require.False(t, decision.shouldSecurityDisable)
}

func TestEvaluateChannelAutoTestKeepsAutoDisabledRecovery(t *testing.T) {
	previous := common.AutomaticEnableChannelEnabled
	common.AutomaticEnableChannelEnabled = true
	defer func() {
		common.AutomaticEnableChannelEnabled = previous
	}()

	channel := &model.Channel{Status: common.ChannelStatusAutoDisabled}
	result := testResult{}

	decision := evaluateChannelAutoTest(channel, result, 800, 5000)

	require.True(t, decision.shouldEnable)
}

func TestEvaluateChannelAutoTestKeepsOrdinaryAutoDisable(t *testing.T) {
	previous := common.AutomaticDisableChannelEnabled
	common.AutomaticDisableChannelEnabled = true
	defer func() {
		common.AutomaticDisableChannelEnabled = previous
	}()

	channel := &model.Channel{
		Status: common.ChannelStatusEnabled,
		Type:   constant.ChannelTypeGemini,
	}
	result := testResult{
		newAPIError: types.NewOpenAIError(errors.New("forbidden"), types.ErrorCodeChannelInvalidKey, 403),
	}

	decision := evaluateChannelAutoTest(channel, result, 800, 5000)

	require.True(t, decision.shouldDisable)
	require.False(t, decision.shouldSecurityDisable)
}

func TestShouldSkipChannelForTestRunMode(t *testing.T) {
	t.Parallel()

	channel := &model.Channel{SkipAutoTest: true}

	require.True(t, shouldSkipChannelForTestRun(channel, channelTestRunModeAuto))
	require.False(t, shouldSkipChannelForTestRun(channel, channelTestRunModeManual))
	require.False(t, shouldSkipChannelForTestRun(&model.Channel{SkipAutoTest: false}, channelTestRunModeAuto))
	require.True(t, shouldSkipChannelForTestRun(&model.Channel{Status: common.ChannelStatusManuallyDisabled, SkipAutoTest: true}, channelTestRunModeManual))
}
