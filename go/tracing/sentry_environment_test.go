package tracing

import (
	"testing"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func TestSentryEnvironment_ExplicitHoneycombOverridesRunMode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                 string
		runMode              string
		honeycombEnvironment string
		want                 string
	}{
		{
			name:                 "dogfood passes through even when run-mode is release",
			runMode:              "release",
			honeycombEnvironment: "dogfood",
			want:                 "dogfood",
		},
		{
			name:                 "prod passes through unchanged",
			runMode:              "release",
			honeycombEnvironment: "prod",
			want:                 "prod",
		},
		{
			name:                 "explicit honeycomb value trims whitespace",
			runMode:              "debug",
			honeycombEnvironment: "  dogfood  ",
			want:                 "dogfood",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := sentryEnvironment(tc.runMode, tc.honeycombEnvironment)
			if got != tc.want {
				t.Errorf("sentryEnvironment() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestSentryEnvironment_AbsentHoneycombUsesRunMode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		runMode string
		want    string
	}{
		{name: "release maps to prod", runMode: "release", want: "prod"},
		{name: "test maps to dogfood", runMode: "test", want: "dogfood"},
		{name: "debug maps to local", runMode: "debug", want: "local"},
		{name: "unknown maps to dev", runMode: "something-else", want: "dev"},
		{name: "empty maps to dev", runMode: "", want: "dev"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// Empty honeycombEnvironment models Brent's empty cobra default
			// (or a service that never registers the key).
			got := sentryEnvironment(tc.runMode, "")
			if got != tc.want {
				t.Errorf("sentryEnvironment(%q, \"\") = %q, want %q", tc.runMode, got, tc.want)
			}
		})
	}
}

// TestSentryEnvironmentFromViper_BrentConfigurationPath exercises the real
// Brent wiring after BRENT-651: empty cobra default, BindEnv for
// UNTIL_BACKEND_HONEYCOMB_ENVIRONMENT, and BindFlagsToViper's rule of skipping
// empty unbound defaults. AWS overlays set the env var; local runs do not.
func TestSentryEnvironmentFromViper_BrentConfigurationPath(t *testing.T) {
	tests := []struct {
		name        string
		runMode     string
		envValue    string // empty means unset
		cliOverride string // empty means flag not Changed
		want        string
	}{
		{
			name:     "dogfood overlay with release run-mode labels dogfood",
			runMode:  "release",
			envValue: "dogfood",
			want:     "dogfood",
		},
		{
			name:     "prod overlay with release run-mode labels prod",
			runMode:  "release",
			envValue: "prod",
			want:     "prod",
		},
		{
			name:    "local debug without honeycomb env keeps local",
			runMode: "debug",
			want:    "local",
		},
		{
			name:    "test run-mode without honeycomb env keeps dogfood",
			runMode: "test",
			want:    "dogfood",
		},
		{
			name:    "unconfigured release without honeycomb env keeps prod",
			runMode: "release",
			want:    "prod",
		},
		{
			name:        "explicit CLI honeycomb-environment overrides run-mode",
			runMode:     "debug",
			cliOverride: "dogfood",
			want:        "dogfood",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			viper.Reset()
			t.Cleanup(viper.Reset)

			if tc.envValue != "" {
				t.Setenv("UNTIL_BACKEND_HONEYCOMB_ENVIRONMENT", tc.envValue)
			}

			viper.Set("run-mode", tc.runMode)
			if err := viper.BindEnv("honeycomb-environment", "UNTIL_BACKEND_HONEYCOMB_ENVIRONMENT"); err != nil {
				t.Fatalf("BindEnv: %v", err)
			}

			fs := pflag.NewFlagSet("brent-backend", pflag.ContinueOnError)
			// Match services/brent-backend/cmd/root.go: empty default so an
			// unbound flag is not treated as an explicit honeycomb env.
			fs.String("honeycomb-environment", "", "Honeycomb environment slug")
			args := []string{}
			if tc.cliOverride != "" {
				args = append(args, "--honeycomb-environment="+tc.cliOverride)
			}
			if err := fs.Parse(args); err != nil {
				t.Fatalf("Parse: %v", err)
			}

			// Mirror startup.BindFlagsToViper: bind only when DefValue is
			// non-empty or the flag was explicitly changed.
			f := fs.Lookup("honeycomb-environment")
			if f.DefValue != "" || f.Changed {
				if err := viper.BindPFlag("honeycomb-environment", f); err != nil {
					t.Fatalf("BindPFlag: %v", err)
				}
			}

			got := sentryEnvironmentFromViper()
			if got != tc.want {
				t.Errorf("sentryEnvironmentFromViper() = %q, want %q (IsSet=%v Get=%q)",
					got, tc.want, viper.IsSet("honeycomb-environment"), viper.GetString("honeycomb-environment"))
			}
		})
	}
}
