package main

import (
	"os"

	"github.com/mggarofalo/plane-cli/cmd"
)

func main() {
	os.Exit(cmd.Execute())
}
