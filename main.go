package main

import (
	"fmt"
	"os"

	"github.com/errordeveloper/imagine/cmd"
)

func main() {
	root := &cmd.Command{}
	cmd.Root(root)
	if err := root.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
