package command

import (
	"github.com/urfave/cli"
	"github.com/wangao1236/my-runc/pkg/container"
)

var ListCommand = cli.Command{
	Name:  "ps",
	Usage: "List all containers",
	Action: func(ctx *cli.Context) error {
		return container.ListContainers()
	},
}
