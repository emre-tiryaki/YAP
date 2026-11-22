package model

import "testing"

func TestTaskPriorityValid(t *testing.T) {
	tests := []struct {
		p    TaskPriority
		want bool
	}{
		{PriorityUrgent, true}, {PriorityHigh, true}, {PriorityMedium, true}, {PriorityLow, true}, {PriorityNone, true},
		{"", false}, {"URGENT", false}, {"critical", false},
	}
	for _, tt := range tests {
		if got := tt.p.Valid(); got != tt.want {
			t.Errorf("TaskPriority(%q).Valid() = %v, want %v", tt.p, got, tt.want)
		}
	}
}

func TestRelationKindValid(t *testing.T) {
	tests := []struct {
		k    RelationKind
		want bool
	}{
		{RelationBlocks, true}, {RelationRelates, true}, {RelationDuplicates, true},
		{"", false}, {"blocked_by", false}, {"Blocks", false},
	}
	for _, tt := range tests {
		if got := tt.k.Valid(); got != tt.want {
			t.Errorf("RelationKind(%q).Valid() = %v, want %v", tt.k, got, tt.want)
		}
	}
}
