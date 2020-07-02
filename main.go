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

/*
func main() {
	wd, err := os.Getwd()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	g, err := git.New(wd)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	commitHash, err := g.CommitHashForHead(true)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Printf("commitHash = %s\n", commitHash)

	treeHash, err := g.TreeHashForHead("pkg")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Printf("treeHash = %s\n", treeHash)

	semverTag, err := g.SemVerTagForHead(false)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Printf("semverTag = %s\n", semverTag)
}
*/
