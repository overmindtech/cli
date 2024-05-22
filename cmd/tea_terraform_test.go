package cmd

import (
	"reflect"
	"testing"
)

func TestPlanArgsFromApplyArgs(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want []string
	}{
		{
			name: "No apply-specific arguments",
			args: []string{"-var-file=vars.tfvars", "-out=tfplan"},
			want: []string{"-var-file=vars.tfvars", "-out=tfplan"},
		},
		{
			name: "Single apply-specific argument",
			args: []string{"-var-file=vars.tfvars", "-out=tfplan", "--auto-approve"},
			want: []string{"-var-file=vars.tfvars", "-out=tfplan"},
		},
		{
			name: "Single apply-specific argument, one dash",
			args: []string{"-var-file=vars.tfvars", "-out=tfplan", "-auto-approve"},
			want: []string{"-var-file=vars.tfvars", "-out=tfplan"},
		},
		{
			name: "Multiple apply-specific arguments",
			args: []string{"-var-file=vars.tfvars", "-out=tfplan", "--auto-approve"},
			want: []string{"-var-file=vars.tfvars", "-out=tfplan"},
		},
		{
			name: "Arguments with boolean values",
			args: []string{"-var-file=vars.tfvars", "-out=tfplan", "--auto-approve=FALSE"},
			want: []string{"-var-file=vars.tfvars", "-out=tfplan"},
		},
		{
			name: "No arguments",
			args: []string{},
			want: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := planArgsFromApplyArgs(tt.args)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("planArgsFromApplyArgs() = %v, want %v", got, tt.want)
			}
		})
	}
}
