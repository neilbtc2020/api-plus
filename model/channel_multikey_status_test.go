package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

func TestHandlerMultiKeyUpdateReEnablesAutoDisabledChannelWhenAKeyRecovers(t *testing.T) {
	t.Parallel()

	channel := &Channel{
		Status: common.ChannelStatusAutoDisabled,
		Key:    "key-1\nkey-2",
		ChannelInfo: ChannelInfo{
			IsMultiKey:             true,
			MultiKeySize:           2,
			MultiKeyStatusList:     map[int]int{0: common.ChannelStatusAutoDisabled, 1: common.ChannelStatusAutoDisabled},
			MultiKeyDisabledReason: map[int]string{0: "quota", 1: "quota"},
			MultiKeyDisabledTime:   map[int]int64{0: 111, 1: 222},
		},
	}
	channel.SetOtherInfo(map[string]interface{}{
		"status_reason": "All keys are disabled",
		"status_time":   int64(123456),
	})

	handlerMultiKeyUpdate(channel, "key-1", common.ChannelStatusEnabled, "")

	require.Equal(t, common.ChannelStatusEnabled, channel.Status)
	require.Equal(t, map[int]int{1: common.ChannelStatusAutoDisabled}, channel.ChannelInfo.MultiKeyStatusList)
	require.Equal(t, map[int]string{1: "quota"}, channel.ChannelInfo.MultiKeyDisabledReason)
	require.Equal(t, map[int]int64{1: 222}, channel.ChannelInfo.MultiKeyDisabledTime)
	otherInfo := channel.GetOtherInfo()
	_, hasReason := otherInfo["status_reason"]
	_, hasTime := otherInfo["status_time"]
	require.False(t, hasReason)
	require.False(t, hasTime)
}
