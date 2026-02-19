package main

import (
	_ "go.uber.org/automaxprocs"

	"github.com/overmindtech/cli/sources/snapshot/cmd"
)

func main() {
	cmd.Execute()
}
