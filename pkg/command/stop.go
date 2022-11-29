package command

import (
	"fmt"

	"github.com/urfave/cli"
	"github.com/wangao1236/my-docker/pkg/container"
)

var StopCommand = cli.Command{
	Name:  "stop",
	Usage: "Stop the container",
	Action: func(ctx *cli.Context) error {
		if len(ctx.Args()) < 1 {
			return fmt.Errorf("missing container name")
		}
		return container.StopContainer(ctx.Args().Get(0))
	},
}
