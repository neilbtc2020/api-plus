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
