package cmd

import (
	"strings"
	"testing"

	"github.com/overmindtech/cli/sdp-go"
	"github.com/spf13/viper"
)

func TestParseLabelsArgument(t *testing.T) {
	tests := []struct {
		name          string
		labels        []string
		want          []*sdp.Label
		errorContains string
	}{
		{
			name:   "empty labels",
			labels: []string{},
			want:   []*sdp.Label{},
		},
		{
			name:   "single label with hash",
			labels: []string{"label1=#FF0000"},
			want: []*sdp.Label{
				{
					Name:   "label1",
					Colour: "#FF0000",
					Type:   sdp.LabelType_LABEL_TYPE_USER,
				},
			},
		},
		{
			name:   "single label without hash",
			labels: []string{"label1=ff0000"},
			want: []*sdp.Label{
				{
					Name:   "label1",
					Colour: "#FF0000",
					Type:   sdp.LabelType_LABEL_TYPE_USER,
				},
			},
		},
		{
			name:   "single label with lowercase hex",
			labels: []string{"label1=abc123"},
			want: []*sdp.Label{
				{
					Name:   "label1",
					Colour: "#ABC123",
					Type:   sdp.LabelType_LABEL_TYPE_USER,
				},
			},
		},
		{
			name:   "multiple labels with hash",
			labels: []string{"label1=#FF0000", "label2=#00FF00", "label3=#0000FF"},
			want: []*sdp.Label{
				{
					Name:   "label1",
					Colour: "#FF0000",
					Type:   sdp.LabelType_LABEL_TYPE_USER,
				},
				{
					Name:   "label2",
					Colour: "#00FF00",
					Type:   sdp.LabelType_LABEL_TYPE_USER,
				},
				{
					Name:   "label3",
					Colour: "#0000FF",
					Type:   sdp.LabelType_LABEL_TYPE_USER,
				},
			},
		},
		{
			name:   "multiple labels mixed hash and no hash",
			labels: []string{"label1=#FF0000", "label2=00FF00", "label3=#0000FF"},
			want: []*sdp.Label{
				{
					Name:   "label1",
					Colour: "#FF0000",
					Type:   sdp.LabelType_LABEL_TYPE_USER,
				},
				{
					Name:   "label2",
					Colour: "#00FF00",
					Type:   sdp.LabelType_LABEL_TYPE_USER,
				},
				{
					Name:   "label3",
					Colour: "#0000FF",
					Type:   sdp.LabelType_LABEL_TYPE_USER,
				},
			},
		},
		{
			name:          "missing equals sign",
			labels:        []string{"label1FF0000"},
			errorContains: "invalid label format",
		},
		{
			name:          "empty label name",
			labels:        []string{"=#FF0000"},
			errorContains: "label name cannot be empty",
		},
		{
			name:          "empty colour",
			labels:        []string{"label1="},
			errorContains: "colour cannot be empty",
		},
		{
			name:          "colour too short",
			labels:        []string{"label1=#FF00"},
			errorContains: "must be 6 hex digits",
		},
		{
			name:          "colour too long",
			labels:        []string{"label1=#FF00000"},
			errorContains: "must be 6 hex digits",
		},
		{
			name:          "invalid hex characters",
			labels:        []string{"label1=#GGGGGG"},
			errorContains: "must be valid hex digits",
		},
		{
			name:          "colour without hash too short",
			labels:        []string{"label1=FF00"},
			errorContains: "must be 6 hex digits",
		},
		{
			name:          "colour without hash invalid characters",
			labels:        []string{"label1=ZZZZZZ"},
			errorContains: "must be valid hex digits",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up viper with test labels
			viper.Reset()
			viper.Set("labels", tt.labels)

			got, err := parseLabelsArgument()

			if tt.errorContains != "" {
				if err == nil {
					t.Errorf("parseLabelsArgument() expected error containing %q, got nil", tt.errorContains)
					return
				}
				if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("parseLabelsArgument() error = %v, want error containing %q", err, tt.errorContains)
				}
				return
			}

			if err != nil {
				t.Errorf("parseLabelsArgument() unexpected error: %v", err)
				return
			}

			if len(got) != len(tt.want) {
				t.Errorf("parseLabelsArgument() returned %d labels, want %d", len(got), len(tt.want))
				return
			}

			for i, wantLabel := range tt.want {
				if got[i].GetName() != wantLabel.GetName() {
					t.Errorf("parseLabelsArgument() label[%d].Name = %q, want %q", i, got[i].GetName(), wantLabel.GetName())
				}
				if got[i].GetColour() != wantLabel.GetColour() {
					t.Errorf("parseLabelsArgument() label[%d].Colour = %q, want %q", i, got[i].GetColour(), wantLabel.GetColour())
				}
				if got[i].GetType() != wantLabel.GetType() {
					t.Errorf("parseLabelsArgument() label[%d].Type = %v, want %v", i, got[i].GetType(), wantLabel.GetType())
				}
			}
		})
	}
}
