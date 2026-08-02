package main

import (
	"github.com/dashimaki/mentle/cmd/cli"
	"github.com/dashimaki/mentle/cmd/server"
)

func main() {
	root := cli.NewCommand()
	root.AddCommand(server.NewCommand())
	root.Execute()
}
