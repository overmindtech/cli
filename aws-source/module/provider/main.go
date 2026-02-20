package main

import (
	"context"
	"fmt"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/overmindtech/cli/go/tracing"
	log "github.com/sirupsen/logrus"
	"github.com/uptrace/opentelemetry-go-extra/otellogrus"
)

const (
	version = "0.1.0"

	defaultHoneycombAPIKey = "hcaik_01j03qe0exnn2jxpj2vxkqb7yrqtr083kyk9rxxt2wzjamz8be94znqmwa" //nolint:gosec // public ingest key, same as CLI
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		//nolint:gocritic // os.Exit in main after deferred cleanup is the only option
		os.Exit(1)
	}
}

func run() error {
	formatter := new(log.TextFormatter)
	formatter.DisableTimestamp = true
	log.SetFormatter(formatter)
	log.SetOutput(os.Stderr)
	log.SetLevel(log.ErrorLevel)

	honeycombAPIKey := defaultHoneycombAPIKey
	if v, ok := os.LookupEnv("HONEYCOMB_API_KEY"); ok {
		honeycombAPIKey = v
	}
	if honeycombAPIKey != "" {
		if err := tracing.InitTracerWithUpstreams("overmind-terraform-provider", honeycombAPIKey, ""); err != nil {
			return fmt.Errorf("initialising tracing: %w", err)
		}
		defer tracing.ShutdownTracer(context.Background())

		log.AddHook(otellogrus.NewHook(otellogrus.WithLevels(
			log.AllLevels[:log.GetLevel()+1]...,
		)))
	}

	return providerserver.Serve(context.Background(), NewProvider(version), providerserver.ServeOpts{
		Address: "registry.terraform.io/overmindtech/overmind",
	})
}
