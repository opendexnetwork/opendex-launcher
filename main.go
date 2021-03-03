package main

import (
	"fmt"
	"github.com/opendexnetwork/opendex-launcher/core"
	"os"
)

func main() {
	launcher := core.NewLauncher()
	err := launcher.Start()
	if err != nil {
		if core.Debug {
			fmt.Println(err)
		}
		os.Exit(1)
	}
}
