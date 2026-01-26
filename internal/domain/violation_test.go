package domain

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestCalculateViolationCounts(t *testing.T) {
	tests := []struct {
		name       string
		violations []Violation
		want       ViolationCounts
	}{
		{
			name:       "empty list",
			violations: []Violation{},
			want:       ViolationCounts{},
		},
		{
			name: "all pending",
			violations: []Violation{
				{ID: uuid.New(), Status: ViolationStatusPending},
				{ID: uuid.New(), Status: ViolationStatusPending},
			},
			want: ViolationCounts{Total: 2, Pending: 2},
		},
		{
			name: "all confirmed",
			violations: []Violation{
				{ID: uuid.New(), Status: ViolationStatusConfirmed},
				{ID: uuid.New(), Status: ViolationStatusConfirmed},
				{ID: uuid.New(), Status: ViolationStatusConfirmed},
			},
			want: ViolationCounts{Total: 3, Confirmed: 3},
		},
		{
			name: "all rejected",
			violations: []Violation{
				{ID: uuid.New(), Status: ViolationStatusRejected},
			},
			want: ViolationCounts{Total: 1, Rejected: 1},
		},
		{
			name: "mixed statuses",
			violations: []Violation{
				{ID: uuid.New(), Status: ViolationStatusPending},
				{ID: uuid.New(), Status: ViolationStatusConfirmed},
				{ID: uuid.New(), Status: ViolationStatusRejected},
				{ID: uuid.New(), Status: ViolationStatusPending},
				{ID: uuid.New(), Status: ViolationStatusConfirmed},
			},
			want: ViolationCounts{Total: 5, Pending: 2, Confirmed: 2, Rejected: 1},
		},
		{
			name: "unknown status is counted in total only",
			violations: []Violation{
				{ID: uuid.New(), Status: ViolationStatus("unknown")},
				{ID: uuid.New(), Status: ViolationStatusPending},
			},
			want: ViolationCounts{Total: 2, Pending: 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateViolationCounts(tt.violations)
			assert.Equal(t, tt.want, got)
		})
	}
}
