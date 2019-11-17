package main

import (
	"fmt"
	"http/cmd"
	"os"
)

func main() {

	if err := cmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
