package controller

import "testing"

func TestMarkSelectedGroup(t *testing.T) {
	groups := []availabilityGroupView{
		{Name: "vip"},
		{Name: "default"},
	}

	marked := markSelectedGroup(groups, "default")
	if marked[0].Selected {
		t.Fatalf("expected vip to remain unselected")
	}
	if !marked[1].Selected {
		t.Fatalf("expected default to be selected")
	}
}

func TestWithSnapshotWarning_UsesWarningWhenDataIsStale(t *testing.T) {
	response := withSnapshotWarning(modelAvailabilityResponseData{
		SelectedGroup: "vip",
	}, true)

	if response.Warning == "" {
		t.Fatalf("expected stale response to include warning")
	}
}

func TestSummarizeProbeAttempts_AnySuccessWins(t *testing.T) {
	attempts := []probeAttemptSummary{
		{ChannelID: 1, Success: false, Message: "timeout"},
		{ChannelID: 2, Success: true, Message: "ok", ResponseTimeMs: 850},
	}

	result := summarizeProbeAttempts(attempts)
	if result.Status != "success" {
		t.Fatalf("expected success status, got %q", result.Status)
	}
	if result.ResponseTimeMs != 850 {
		t.Fatalf("expected response time 850ms, got %d", result.ResponseTimeMs)
	}
}

func TestSummarizeProbeAttempts_AllFailKeepsLastMessage(t *testing.T) {
	attempts := []probeAttemptSummary{
		{ChannelID: 1, Success: false, Message: "timeout"},
		{ChannelID: 2, Success: false, Message: "upstream 500"},
	}

	result := summarizeProbeAttempts(attempts)
	if result.Status != "fail" {
		t.Fatalf("expected fail status, got %q", result.Status)
	}
	if result.Message != "upstream 500" {
		t.Fatalf("expected last failure message to be kept, got %q", result.Message)
	}
}
