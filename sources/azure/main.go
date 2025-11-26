package main

import (
	_ "go.uber.org/automaxprocs"

	"github.com/overmindtech/cli/sources/azure/cmd"
)

func main() {
	cmd.Execute()
}
