package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInspection_DetermineAnalysisAction(t *testing.T) {
	tests := []struct {
		name           string
		status         InspectionStatus
		pendingImages  int64
		totalImages    int64
		jobInProgress  bool
		wantCanAnalyze bool
		wantMessage    string
	}{
		// Draft status
		{
			name:        "draft with no images",
			status:      InspectionStatusDraft,
			totalImages: 0,
			wantMessage: "Upload photos to begin analysis",
		},
		{
			name:           "draft with 1 pending image",
			status:         InspectionStatusDraft,
			pendingImages:  1,
			totalImages:    1,
			wantCanAnalyze: true,
			wantMessage:    "Ready to analyze 1 image",
		},
		{
			name:           "draft with multiple pending images",
			status:         InspectionStatusDraft,
			pendingImages:  5,
			totalImages:    5,
			wantCanAnalyze: true,
			wantMessage:    "Ready to analyze 5 images",
		},
		{
			name:          "draft with job in progress",
			status:        InspectionStatusDraft,
			pendingImages: 3,
			totalImages:   5,
			jobInProgress: true,
			wantMessage:   "Analyzing images...",
		},
		{
			name:          "draft with all images analyzed",
			status:        InspectionStatusDraft,
			pendingImages: 0,
			totalImages:   5,
			wantMessage:   "All images have been analyzed",
		},

		// Analyzing status
		{
			name:        "analyzing status",
			status:      InspectionStatusAnalyzing,
			totalImages: 5,
			wantMessage: "Analyzing images...",
		},

		// Review status
		{
			name:           "review with 1 new pending image",
			status:         InspectionStatusReview,
			pendingImages:  1,
			totalImages:    6,
			wantCanAnalyze: true,
			wantMessage:    "Ready to analyze 1 new image",
		},
		{
			name:           "review with multiple new pending images",
			status:         InspectionStatusReview,
			pendingImages:  3,
			totalImages:    8,
			wantCanAnalyze: true,
			wantMessage:    "Ready to analyze 3 new images",
		},
		{
			name:          "review with job in progress",
			status:        InspectionStatusReview,
			pendingImages: 2,
			totalImages:   5,
			jobInProgress: true,
			wantMessage:   "Analyzing new images...",
		},
		{
			name:        "review with analysis complete",
			status:      InspectionStatusReview,
			totalImages: 5,
			wantMessage: "Analysis complete",
		},

		// Completed status
		{
			name:        "completed",
			status:      InspectionStatusCompleted,
			totalImages: 5,
			wantMessage: "Inspection finalized",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inspection := &Inspection{Status: tt.status}
			canAnalyze, message := inspection.DetermineAnalysisAction(tt.pendingImages, tt.totalImages, tt.jobInProgress)
			assert.Equal(t, tt.wantCanAnalyze, canAnalyze, "canAnalyze mismatch")
			assert.Equal(t, tt.wantMessage, message, "message mismatch")
		})
	}
}

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
