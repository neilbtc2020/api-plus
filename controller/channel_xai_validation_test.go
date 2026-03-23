package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/require"
)

func TestValidateChannelXAIAccountTokenAcceptsMultilineTokens(t *testing.T) {
	t.Parallel()

	channel := &model.Channel{
		Type: constant.ChannelTypeXai,
		Key:  "\n  token-1  \n\n   \n token-2 \n",
	}
	channel.SetOtherSettings(dto.ChannelOtherSettings{XAIAuthMode: dto.XAIAuthModeAccountToken})

	err := validateChannel(channel, true)

	require.NoError(t, err)
}

func TestValidateChannelXAIAccountTokenRejectsEffectivelyEmptyKey(t *testing.T) {
	t.Parallel()

	channel := &model.Channel{
		Type: constant.ChannelTypeXai,
		Key:  " \n \t \n  ",
	}
	channel.SetOtherSettings(dto.ChannelOtherSettings{XAIAuthMode: dto.XAIAuthModeAccountToken})

	err := validateChannel(channel, true)

	require.ErrorContains(t, err, "xAI account token")
}

func TestValidateChannelXAIAccountTokenAllowsSparseUpdateWithoutKeyReplacement(t *testing.T) {
	t.Parallel()

	channel := &model.Channel{
		Type: constant.ChannelTypeXai,
		Key:  "",
	}
	channel.SetOtherSettings(dto.ChannelOtherSettings{XAIAuthMode: dto.XAIAuthModeAccountToken})

	err := validateChannel(channel, false)

	require.NoError(t, err)
}

func TestValidateChannelXAIAPIKeyKeepsExistingValidationBehavior(t *testing.T) {
	t.Parallel()

	channel := &model.Channel{
		Type: constant.ChannelTypeXai,
		Key:  " \n \t \n  ",
	}
	channel.SetOtherSettings(dto.ChannelOtherSettings{XAIAuthMode: dto.XAIAuthModeAPIKey})

	err := validateChannel(channel, true)

	require.NoError(t, err)
}

func TestNormalizeXAIAuthMode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		settings dto.ChannelOtherSettings
		expected dto.XAIAuthMode
	}{
		{
			name:     "defaults empty to api key",
			settings: dto.ChannelOtherSettings{},
			expected: dto.XAIAuthModeAPIKey,
		},
		{
			name:     "defaults unknown to api key",
			settings: dto.ChannelOtherSettings{XAIAuthMode: dto.XAIAuthMode("something_else")},
			expected: dto.XAIAuthModeAPIKey,
		},
		{
			name:     "preserves account token",
			settings: dto.ChannelOtherSettings{XAIAuthMode: dto.XAIAuthModeAccountToken},
			expected: dto.XAIAuthModeAccountToken,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.expected, tt.settings.NormalizeXAIAuthMode())
		})
	}
}
