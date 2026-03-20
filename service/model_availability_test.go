package service

import (
	"testing"
)

func TestBuildRecentResultsWindow_FillsMissingWithDefaultGreen(t *testing.T) {
	logs := []availabilityLogEntry{
		{ModelName: "gpt-4o", Success: false},
		{ModelName: "gpt-4o", Success: true},
	}

	window := buildRecentResultsWindow(logs, 5)

	if len(window) != 5 {
		t.Fatalf("expected 5 results, got %d", len(window))
	}
	if window[0].Status != availabilityResultFail {
		t.Fatalf("expected first status %q, got %q", availabilityResultFail, window[0].Status)
	}
	if window[0].Defaulted {
		t.Fatalf("expected first result to be real")
	}
	if window[1].Status != availabilityResultSuccess {
		t.Fatalf("expected second status %q, got %q", availabilityResultSuccess, window[1].Status)
	}
	if window[1].Defaulted {
		t.Fatalf("expected second result to be real")
	}
	if window[2].Status != availabilityResultSuccess {
		t.Fatalf("expected third status %q, got %q", availabilityResultSuccess, window[2].Status)
	}
	if !window[2].Defaulted {
		t.Fatalf("expected third result to be defaulted")
	}
}

func TestFilterAvailabilityItems_OnlyFailedAndKeyword(t *testing.T) {
	items := []ModelAvailabilityItem{
		{
			ModelName:     "gpt-4o",
			FailCount:     2,
			RecentResults: []RecentResult{{Status: availabilityResultFail}},
		},
		{
			ModelName:     "claude-3-7-sonnet",
			FailCount:     0,
			RecentResults: []RecentResult{{Status: availabilityResultSuccess, Defaulted: true}},
		},
	}

	filtered := filterAvailabilityItems(items, "gpt", true, false)
	if len(filtered) != 1 {
		t.Fatalf("expected 1 filtered item, got %d", len(filtered))
	}
	if filtered[0].ModelName != "gpt-4o" {
		t.Fatalf("expected filtered model gpt-4o, got %q", filtered[0].ModelName)
	}
}

func TestResolveSelectedGroup_FallsBackToFirstVisible(t *testing.T) {
	groups := []VisibleGroup{
		{Name: "vip"},
		{Name: "default"},
	}

	selected := resolveSelectedGroup(groups, "not-exists")
	if selected != "vip" {
		t.Fatalf("expected selected group vip, got %q", selected)
	}
}
