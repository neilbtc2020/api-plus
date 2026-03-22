package service

import (
	"testing"

	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/stretchr/testify/require"
)

func TestScanChannelSecurityUsesConfiguredKeywordRule(t *testing.T) {
	previousEnabled := operation_setting.ChannelSecurityEnabled
	previousRules := operation_setting.ChannelSecurityRules
	previousRaw := operation_setting.ChannelSecurityRulesJSONString
	defer func() {
		operation_setting.ChannelSecurityEnabled = previousEnabled
		operation_setting.ChannelSecurityRules = previousRules
		operation_setting.ChannelSecurityRulesJSONString = previousRaw
	}()

	operation_setting.ChannelSecurityEnabled = true
	err := operation_setting.UpdateChannelSecurityRulesFromString(`[
	  {
	    "id": "custom_callback",
	    "name": "可疑回连",
	    "enabled": true,
	    "match_type": "keyword",
	    "pattern": "callback.example.com",
	    "risk_level": "high_risk",
	    "reason": "命中可疑回连地址"
	  }
	]`)
	require.NoError(t, err)

	result := ScanChannelSecurity([]byte(`please visit https://callback.example.com/report`))

	require.False(t, result.Safe)
	require.Equal(t, ChannelSecurityRiskLevelHighRisk, result.RiskLevel)
	require.Contains(t, result.RuleIDs, "custom_callback")
}

func TestScanChannelSecurityReturnsSafeWhenDisabled(t *testing.T) {
	previousEnabled := operation_setting.ChannelSecurityEnabled
	defer func() {
		operation_setting.ChannelSecurityEnabled = previousEnabled
	}()

	operation_setting.ChannelSecurityEnabled = false

	result := ScanChannelSecurity([]byte(`{"message":"<script>alert(1)</script>"}`))

	require.True(t, result.Safe)
	require.Equal(t, ChannelSecurityRiskLevelSafe, result.RiskLevel)
}
