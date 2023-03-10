package command

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"github.com/wangao1236/my-runc/pkg/container"
)

var CommitCommand = cli.Command{
	Name:  "commit",
	Usage: "Commit a container into image",
	Action: func(ctx *cli.Context) error {
		if len(ctx.Args()) < 1 {
			logrus.Errorf("missing container name")
			return fmt.Errorf("missing container name")
		}
		imageName := ctx.Args().Get(0)
		return container.CommitContainer(imageName)
	},
}
