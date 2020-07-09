package cmd

import (
	"github.com/spf13/cobra"

	"github.com/errordeveloper/imagine/cmd/build"
	"github.com/errordeveloper/imagine/cmd/generate"
	"github.com/errordeveloper/imagine/cmd/image"
)

type Command = cobra.Command

func Root(root *Command) {
	root.Use = "imagine"
	// root.Args = cobra.NoArgs()
	root.AddCommand(generate.GenerateCmd())
	root.AddCommand(build.BuildCmd())
	root.AddCommand(image.ImageCmd())
}
