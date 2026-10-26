package cmd

import (
	"errors"
	"strings"
	"testing"

	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/spf13/viper"
)

func TestParseTagsArgument(t *testing.T) {
	tests := []struct {
		name          string
		tags          []string
		want          map[string]string
		errorContains string
		flagError     bool
	}{
		{
			name: "empty tags",
			tags: []string{},
			want: map[string]string{},
		},
		{
			name: "single tag",
			tags: []string{"env=prod"},
			want: map[string]string{"env": "prod"},
		},
		{
			name: "multiple tags",
			tags: []string{"env=prod", "team=platform"},
			want: map[string]string{"env": "prod", "team": "platform"},
		},
		{
			name:          "missing equals sign",
			tags:          []string{"bad"},
			errorContains: "invalid --tags value 'bad', expected format 'key=value'",
			flagError:     true,
		},
		{
			name: "empty value is allowed",
			tags: []string{"key="},
			want: map[string]string{"key": ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			viper.Reset()
			viper.Set("tags", tt.tags)

			got, err := parseTagsArgument()
			if tt.errorContains != "" {
				if err == nil {
					t.Fatalf("parseTagsArgument() expected error containing %q, got nil", tt.errorContains)
				}
				if !strings.Contains(err.Error(), tt.errorContains) {
					t.Fatalf("parseTagsArgument() error = %v, want containing %q", err, tt.errorContains)
				}
				if tt.flagError {
					if _, ok := errors.AsType[flagError](err); !ok {
						t.Fatalf("parseTagsArgument() expected flagError, got %T", err)
					}
				}
				return
			}
			if err != nil {
				t.Fatalf("parseTagsArgument() unexpected error: %v", err)
			}
			if len(got.GetTagValue()) != len(tt.want) {
				t.Fatalf("parseTagsArgument() returned %d tags, want %d", len(got.GetTagValue()), len(tt.want))
			}
			for key, value := range tt.want {
				tag, ok := got.GetTagValue()[key]
				if !ok {
					t.Fatalf("parseTagsArgument() missing tag %q", key)
				}
				if tag.GetUserTagValue().GetValue() != value {
					t.Fatalf("parseTagsArgument() tag %q = %q, want %q", key, tag.GetUserTagValue().GetValue(), value)
				}
			}
		})
	}
}

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
