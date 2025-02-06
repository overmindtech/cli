package discovery

import (
	"os"
	"runtime"
	"testing"

	"github.com/google/uuid"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// NB we do not call AddEngineFlags so we use command line flags, not environment variables
func TestEngineConfigFromViper(t *testing.T) {
	tests := []struct {
		name                          string
		setupViper                    func()
		engineType                    string
		version                       string
		expectedSourceName            string
		expectedSourceUUID            uuid.UUID
		expectedSourceAccessToken     string
		expectedSourceAccessTokenType string
		expectedManagedSource         sdp.SourceManaged
		expectedApp                   string
		expectedApiServerURL          string
		expectedApiKey                string
		expectedNATSUrl               string
		expectedMaxParallel           int
		expectUnauthenticated         bool
		expectError                   bool
	}{
		{
			name: "default values",
			setupViper: func() {
				viper.Set("app", "https://app.overmind.tech")
				viper.Set("api-key", "api-key")
			},
			engineType:                    "test-engine",
			version:                       "1.0",
			expectedSourceName:            "test-engine-" + getHostname(t),
			expectedSourceUUID:            uuid.Nil,
			expectedSourceAccessToken:     "",
			expectedSourceAccessTokenType: "",
			expectedManagedSource:         sdp.SourceManaged_LOCAL,
			expectedApp:                   "https://app.overmind.tech",
			expectedApiServerURL:          "https://api.app.overmind.tech",
			expectedNATSUrl:               "wss://messages.app.overmind.tech",
			expectedApiKey:                "api-key",
			expectedMaxParallel:           runtime.NumCPU(),
			expectError:                   false,
		},
		{
			name: "custom values",
			setupViper: func() {
				viper.Set("source-name", "custom-source")
				viper.Set("source-uuid", "123e4567-e89b-12d3-a456-426614174000")
				viper.Set("app", "https://df.overmind-demo.com/")
				viper.Set("api-key", "custom-api-key")
				viper.Set("max-parallel", 10)
			},
			engineType:                    "test-engine",
			version:                       "1.0",
			expectedSourceName:            "custom-source",
			expectedSourceUUID:            uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
			expectedSourceAccessToken:     "",
			expectedSourceAccessTokenType: "",
			expectedManagedSource:         sdp.SourceManaged_LOCAL,
			expectedApp:                   "https://df.overmind-demo.com/",
			expectedApiServerURL:          "https://api.df.overmind-demo.com",
			expectedNATSUrl:               "wss://messages.df.overmind-demo.com",
			expectedApiKey:                "custom-api-key",
			expectedMaxParallel:           10,
			expectError:                   false,
		},
		{
			name: "invalid UUID",
			setupViper: func() {
				viper.Set("source-uuid", "invalid-uuid")
			},
			engineType:  "test-engine",
			version:     "1.0",
			expectError: true,
		},
		{
			name: "managed source",
			setupViper: func() {
				viper.Set("source-name", "custom-source")
				viper.Set("source-uuid", "123e4567-e89b-12d3-a456-426614174000")
				viper.Set("source-access-token", "custom-access-token")
				viper.Set("source-access-token-type", "custom-token-type")
				viper.Set("overmind-managed-source", true)
				viper.Set("max-parallel", 10)

				viper.Set("api-server-service-host", "api.app.overmind.tech")
				viper.Set("api-server-service-port", "443")
				viper.Set("nats-service-host", "messages.app.overmind.tech")
				viper.Set("nats-service-port", "443")
			},
			engineType:                    "test-engine",
			version:                       "1.0",
			expectedSourceName:            "custom-source",
			expectedSourceUUID:            uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
			expectedSourceAccessToken:     "custom-access-token",
			expectedSourceAccessTokenType: "custom-token-type",
			expectedManagedSource:         sdp.SourceManaged_MANAGED,

			expectedApiServerURL: "https://api.app.overmind.tech:443",
			expectedNATSUrl:      "nats://messages.app.overmind.tech:443",
			expectedMaxParallel:  10,
			expectError:          false,
		},
		{
			name: "managed source local insecure",
			setupViper: func() {
				viper.Set("source-name", "custom-source")
				viper.Set("source-uuid", "123e4567-e89b-12d3-a456-426614174000")
				viper.Set("source-access-token", "custom-access-token")
				viper.Set("source-access-token-type", "custom-token-type")
				viper.Set("overmind-managed-source", true)
				viper.Set("max-parallel", 10)

				viper.Set("api-server-service-host", "localhost")
				viper.Set("api-server-service-port", "8080")
				viper.Set("nats-service-host", "localhost")
				viper.Set("nats-service-port", "4222")
			},
			engineType:                    "test-engine",
			version:                       "1.0",
			expectedSourceName:            "custom-source",
			expectedSourceUUID:            uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
			expectedSourceAccessToken:     "custom-access-token",
			expectedSourceAccessTokenType: "custom-token-type",
			expectedManagedSource:         sdp.SourceManaged_MANAGED,

			expectedApiServerURL: "http://localhost:8080",
			expectedNATSUrl:      "nats://localhost:4222",
			expectedMaxParallel:  10,
			expectError:          false,
		},
		{
			name:        "source access token and api key not set",
			setupViper:  func() {},
			engineType:  "test-engine",
			version:     "1.0",
			expectError: true,
		},
		{
			name: "fully unauthenticated",
			setupViper: func() {
				viper.Set("app", "https://app.overmind.tech")
				viper.Set("source-name", "custom-source")
				t.Setenv("ALLOW_UNAUTHENTICATED", "true")
			},
			engineType:            "test-engine",
			version:               "1.0",
			expectError:           false,
			expectedMaxParallel:   runtime.NumCPU(),
			expectedSourceName:    "custom-source",
			expectedApp:           "https://app.overmind.tech",
			expectedApiServerURL:  "https://api.app.overmind.tech",
			expectedNATSUrl:       "wss://messages.app.overmind.tech",
			expectUnauthenticated: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("ALLOW_UNAUTHENTICATED", "")
			viper.Reset()
			tt.setupViper()
			engineConfig, err := EngineConfigFromViper(tt.engineType, tt.version)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.engineType, engineConfig.EngineType)
				assert.Equal(t, tt.version, engineConfig.Version)
				assert.Equal(t, tt.expectedSourceName, engineConfig.SourceName)
				if tt.expectedSourceUUID == uuid.Nil {
					assert.NotEqual(t, uuid.Nil, engineConfig.SourceUUID)
				} else {
					assert.Equal(t, tt.expectedSourceUUID, engineConfig.SourceUUID)
				}
				assert.Equal(t, tt.expectedSourceAccessToken, engineConfig.SourceAccessToken)
				assert.Equal(t, tt.expectedSourceAccessTokenType, engineConfig.SourceAccessTokenType)
				assert.Equal(t, tt.expectedManagedSource, engineConfig.OvermindManagedSource)
				assert.Equal(t, tt.expectedApp, engineConfig.App)
				assert.Equal(t, tt.expectedApiServerURL, engineConfig.APIServerURL)
				assert.Equal(t, tt.expectedNATSUrl, engineConfig.NATSOptions.Servers[0])
				assert.Equal(t, tt.expectedApiKey, engineConfig.ApiKey)
				assert.Equal(t, tt.expectedMaxParallel, engineConfig.MaxParallelExecutions)
				assert.Equal(t, tt.expectUnauthenticated, engineConfig.Unauthenticated)
			}
		})
	}
}

func getHostname(t *testing.T) string {
	hostname, err := os.Hostname()
	if err != nil {
		t.Fatalf("error getting hostname: %v", err)
	}
	return hostname
}
