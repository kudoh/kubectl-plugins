package main

import (
	"fmt"
	"http/cmd"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"os"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
