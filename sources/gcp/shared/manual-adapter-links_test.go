package shared

import (
	"reflect"
	"testing"

	"github.com/overmindtech/cli/sdp-go"
)

func TestAWSLinkByARN(t *testing.T) {
	type args struct {
		awsItem          string
		blastPropagation *sdp.BlastPropagation
	}

	tests := []struct {
		name string
		arn  string
		args args
		want *sdp.LinkedItemQuery
	}{
		{
			name: "Link by ARN for AWS IAM Role - global scope",
			arn:  "arn:aws:iam::123456789012:role/MyRole",
			args: args{
				awsItem: "iam-role",
				blastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: false,
				},
			},
			want: &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "iam-role",
					Method: sdp.QueryMethod_SEARCH,
					Query:  "arn:aws:iam::123456789012:role/MyRole",
					Scope:  "123456789012",
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: false,
				},
			},
		},
		{
			name: "Link by ARN for AWS KMS Key - region scope",
			arn:  "arn:aws:kms:us-west-2:123456789012:key/abcd1234-56ef-78gh-90ij-klmnopqrstuv",
			args: args{
				awsItem: "kms-key",
				blastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: false,
				},
			},
			want: &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "kms-key",
					Method: sdp.QueryMethod_SEARCH,
					Query:  "arn:aws:kms:us-west-2:123456789012:key/abcd1234-56ef-78gh-90ij-klmnopqrstuv",
					Scope:  "123456789012.us-west-2", // Region scope
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: false,
				},
			},
		},
		{
			name: "Malformed ARN",
			arn:  "invalid-arn",
			args: args{
				awsItem:          "iam-role",
				blastPropagation: &sdp.BlastPropagation{},
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotFunc := AWSLinkByARN(tt.args.awsItem)
			gotLIQ := gotFunc("", "", tt.arn, tt.args.blastPropagation)
			if !reflect.DeepEqual(gotLIQ, tt.want) {
				t.Errorf("AWSLinkByARN() = %v, want %v", gotLIQ, tt.want)
			}
		})
	}
}
