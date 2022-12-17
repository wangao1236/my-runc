package command

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"github.com/wangao1236/my-runc/pkg/container"
	"github.com/wangao1236/my-runc/pkg/network"
)

var RemoveCommand = cli.Command{
	Name:  "rm",
	Usage: "Remove the container",
	Action: func(ctx *cli.Context) error {
		if len(ctx.Args()) < 1 {
			return fmt.Errorf("missing container name")
		}
		containerName := ctx.Args().Get(0)
		metadata, err := container.ReadMetadata(containerName)
		if err != nil {
			logrus.Errorf("failed to read metadata of %v: %v", containerName, err)
			return err
		}
		if err = network.Disconnect(metadata); err != nil {
			logrus.Errorf("failed to disconnect network for container (%v): %v", metadata, err)
			return err
		}
		return container.RemoveContainer(metadata)
	},
}
