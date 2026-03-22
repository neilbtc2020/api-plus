package operation_setting

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUpdateChannelSecurityRulesFromStringUsesDefaultWhenEmpty(t *testing.T) {
	previousRules := ChannelSecurityRules
	previousRaw := ChannelSecurityRulesJSONString
	defer func() {
		ChannelSecurityRules = previousRules
		ChannelSecurityRulesJSONString = previousRaw
	}()

	err := UpdateChannelSecurityRulesFromString("")

	require.NoError(t, err)
	require.NotEmpty(t, ChannelSecurityRules)
	require.NotEmpty(t, ChannelSecurityRulesJSONString)
}

func TestUpdateChannelSecurityRulesFromStringRejectsInvalidJSON(t *testing.T) {
	previousRules := ChannelSecurityRules
	previousRaw := ChannelSecurityRulesJSONString
	defer func() {
		ChannelSecurityRules = previousRules
		ChannelSecurityRulesJSONString = previousRaw
	}()

	err := UpdateChannelSecurityRulesFromString(`{"bad":true}`)

	require.Error(t, err)
}
