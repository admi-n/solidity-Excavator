package main

import (
	"github.com/admi-n/solidity-Excavator/src/cmd"
)

func main() {
	cmd.Print()
	if err := cmd.Run(); err != nil {
		cmd.PrintFatal(err)
	}
}
