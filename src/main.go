package main

import (
	"github.com/admi-n/solidity-Excavator/src/cmd"
)

func main() {
	if err := cmd.Run(); err != nil {
		cmd.PrintFatal(err)
	}
}
