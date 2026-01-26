package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInspection_TransitionTo(t *testing.T) {
	tests := []struct {
		name      string
		from      InspectionStatus
		to        InspectionStatus
		wantErr   bool
		wantState InspectionStatus
	}{
		// Valid forward transitions
		{"draft to analyzing", InspectionStatusDraft, InspectionStatusAnalyzing, false, InspectionStatusAnalyzing},
		{"analyzing to review", InspectionStatusAnalyzing, InspectionStatusReview, false, InspectionStatusReview},
		{"review to completed", InspectionStatusReview, InspectionStatusCompleted, false, InspectionStatusCompleted},

		// Valid backward transitions
		{"completed to review", InspectionStatusCompleted, InspectionStatusReview, false, InspectionStatusReview},
		{"analyzing to draft", InspectionStatusAnalyzing, InspectionStatusDraft, false, InspectionStatusDraft},
		{"review to draft", InspectionStatusReview, InspectionStatusDraft, false, InspectionStatusDraft},
		{"completed to draft", InspectionStatusCompleted, InspectionStatusDraft, false, InspectionStatusDraft},

		// Invalid transitions
		{"draft to review", InspectionStatusDraft, InspectionStatusReview, true, InspectionStatusDraft},
		{"draft to completed", InspectionStatusDraft, InspectionStatusCompleted, true, InspectionStatusDraft},
		{"analyzing to completed", InspectionStatusAnalyzing, InspectionStatusCompleted, true, InspectionStatusAnalyzing},
		{"review to analyzing", InspectionStatusReview, InspectionStatusAnalyzing, true, InspectionStatusReview},
		{"completed to analyzing", InspectionStatusCompleted, InspectionStatusAnalyzing, true, InspectionStatusCompleted},

		// Same status (draft → draft is allowed since any → draft is valid)
		{"draft to draft", InspectionStatusDraft, InspectionStatusDraft, false, InspectionStatusDraft},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inspection := &Inspection{Status: tt.from}
			err := inspection.TransitionTo(tt.to)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "cannot transition")
				// Status should not change on error
				assert.Equal(t, tt.from, inspection.Status)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantState, inspection.Status)
			}
		})
	}
}
