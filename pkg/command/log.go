package command

import (
	"fmt"

	"github.com/urfave/cli"
	"github.com/wangao1236/my-docker/pkg/container"
)

var LogCommand = cli.Command{
	Name:  "logs",
	Usage: "Print logs of containers",
	Action: func(ctx *cli.Context) error {
		if len(ctx.Args()) < 1 {
			return fmt.Errorf("missing container name")
		}
		return container.LogContainer(ctx.Args().Get(0))
	},
}
