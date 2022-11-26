package command

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"github.com/wangao1236/my-docker/pkg/container"
)

var ExecCommand = cli.Command{
	Name:  "exec",
	Usage: "Exec command to enter into a container",
	Action: func(ctx *cli.Context) error {
		if os.Getenv(container.EnvMyDockerPID) != "" {
			logrus.Infof("pid callback pid: %v", os.Getgid())
			return nil
		}
		if len(ctx.Args()) < 2 {
			return fmt.Errorf("missing container name or command")
		}
		containerName := ctx.Args().Get(0)
		args := append([]string{}, ctx.Args().Tail()...)
		return container.ExecContainer(containerName, args)
	},
}
