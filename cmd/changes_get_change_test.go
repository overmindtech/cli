package cmd

import (
	"testing"

	"github.com/overmindtech/cli/sdp-go"
	"github.com/stretchr/testify/assert"
)

func TestGetTagsLine(t *testing.T) {
	tests := []struct {
		name string
		tags map[string]*sdp.TagValue
		want string
	}{
		{
			name: "empty tags",
			tags: map[string]*sdp.TagValue{},
			want: "",
		},
		{
			name: "key only",
			tags: map[string]*sdp.TagValue{
				"key": {},
			},
			want: "`key` ",
		},
		{
			name: "auto tag without value",
			tags: map[string]*sdp.TagValue{
				"autoTag": {
					Value: &sdp.TagValue_AutoTagValue{
						AutoTagValue: &sdp.AutoTagValue{},
					},
				},
			},
			want: "`✨autoTag` ",
		},
		{
			name: "auto tag with value",
			tags: map[string]*sdp.TagValue{
				"autoTag": {
					Value: &sdp.TagValue_AutoTagValue{
						AutoTagValue: &sdp.AutoTagValue{
							Value: "value",
						},
					},
				},
			},
			want: "`✨autoTag|value` ",
		},
		{
			name: "user tag without value",
			tags: map[string]*sdp.TagValue{
				"userTag": {
					Value: &sdp.TagValue_UserTagValue{
						UserTagValue: &sdp.UserTagValue{},
					},
				},
			},
			want: "`userTag` ",
		},
		{
			name: "user tag with value",
			tags: map[string]*sdp.TagValue{
				"userTag": {
					Value: &sdp.TagValue_UserTagValue{
						UserTagValue: &sdp.UserTagValue{
							Value: "value",
						},
					},
				},
			},
			want: "`userTag|value` ",
		},
		{
			name: "mixed tags",
			tags: map[string]*sdp.TagValue{
				"autoTag": {
					Value: &sdp.TagValue_AutoTagValue{
						AutoTagValue: &sdp.AutoTagValue{
							Value: "value",
						},
					},
				},
				"userTag": {
					Value: &sdp.TagValue_UserTagValue{
						UserTagValue: &sdp.UserTagValue{
							Value: "value",
						},
					},
				},
			},
			want: "`✨autoTag|value` `userTag|value` ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getTagsLine(tt.tags)
			assert.Equal(t, tt.want, got)
		})
	}
}
