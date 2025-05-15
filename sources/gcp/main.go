package main

import (
	_ "go.uber.org/automaxprocs"

	"github.com/overmindtech/cli/sources/gcp/cmd"
)

func main() {
	cmd.Execute()
}
