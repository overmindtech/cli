package internal

import "fmt"

// GatewayURL returns the URL for the gateway for a given pase URL. For example
// if the base URL is https://api.prod.overmind.tech, the gateway URL will be
// https://api.prod.overmind.tech/api/gateway
func GatewayURL(base string) string {
	return fmt.Sprintf("%v/api/gateway", base)
}
