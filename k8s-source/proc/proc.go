package proc

import (
	"fmt"

	"github.com/spf13/viper"
)

// K8sConfig holds configuration for the k8s source read from viper.
type K8sConfig struct {
	Kubeconfig      string
	ClusterName     string
	RateLimitQPS    float64
	RateLimitBurst  int
	HealthCheckPort int
}

// ConfigFromViper reads and validates k8s source configuration from viper.
// Kubeconfig may be empty (in-cluster config). Returns an error if rate limits
// or health-check-port are invalid.
func ConfigFromViper() (*K8sConfig, error) {
	rateLimitQPS := viper.GetFloat64("rate-limit-qps")
	rateLimitBurst := viper.GetInt("rate-limit-burst")
	healthCheckPort := viper.GetInt("health-check-port")

	if rateLimitQPS <= 0 {
		return nil, fmt.Errorf("rate-limit-qps must be positive, got %v", rateLimitQPS)
	}
	if rateLimitBurst <= 0 {
		return nil, fmt.Errorf("rate-limit-burst must be positive, got %v", rateLimitBurst)
	}
	if healthCheckPort < 1 || healthCheckPort > 65535 {
		return nil, fmt.Errorf("health-check-port must be between 1 and 65535, got %v", healthCheckPort)
	}

	return &K8sConfig{
		Kubeconfig:      viper.GetString("kubeconfig"),
		ClusterName:     viper.GetString("cluster-name"),
		RateLimitQPS:    rateLimitQPS,
		RateLimitBurst:  rateLimitBurst,
		HealthCheckPort: healthCheckPort,
	}, nil
}
