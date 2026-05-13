// Copyright 2026 OpenProse contributors. Licensed under MIT. See LICENSE.

package main

import (
	"fmt"
	"os"

	"github.com/openprose/grant-finder/cli/grant-finder/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(cli.ExitCode(err))
	}
}
