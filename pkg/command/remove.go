package command

import (
	"fmt"

	"github.com/urfave/cli"
	"github.com/wangao1236/my-docker/pkg/container"
)

var RemoveCommand = cli.Command{
	Name:  "rm",
	Usage: "Remove the container",
	Action: func(ctx *cli.Context) error {
		if len(ctx.Args()) < 1 {
			return fmt.Errorf("missing container name")
		}
		return container.RemoveContainer(ctx.Args().Get(0))
	},
}
