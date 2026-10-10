package main

import (
	"github.com/ProjectViVy/laputa/mentle/cmd/cli"
	"github.com/ProjectViVy/laputa/mentle/cmd/server"
)

func main() {
	root := cli.NewCommand()
	root.AddCommand(server.NewCommand())
	root.Execute()
}
