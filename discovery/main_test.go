package discovery

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/overmindtech/cli/tracing"
)

func TestMain(m *testing.M) {
	exitCode := func() int {
		defer tracing.ShutdownTracer(context.Background())

		if err := tracing.InitTracerWithUpstreams("discovery-tests", os.Getenv("HONEYCOMB_API_KEY"), ""); err != nil {
			log.Fatal(err)
		}

		return m.Run()
	}()

	os.Exit(exitCode)
}
