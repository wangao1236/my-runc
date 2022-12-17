package command

import (
	"fmt"

	"github.com/urfave/cli"
	"github.com/wangao1236/my-runc/pkg/network"
)

var NetworkCommand = cli.Command{
	Name:  "network",
	Usage: "Subcommand of container network",
	Subcommands: []cli.Command{
		{
			Name:  "create",
			Usage: "Create a container network",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "driver",
					Usage: "network driver name",
					Value: network.DriverBridge,
				},
				cli.StringFlag{
					Name:  "subnet",
					Usage: "subnet cidr",
				},
			},
			Action: func(ctx *cli.Context) error {
				if len(ctx.Args()) < 1 {
					return fmt.Errorf("missing network name")
				}

				driver := ctx.String("driver")
				subnet := ctx.String("subnet")
				name := ctx.Args()[0]
				return network.CreateNetwork(driver, subnet, name)
			},
		},
		{
			Name:  "list",
			Usage: "List the existing container networks",
			Action: func(ctx *cli.Context) error {
				return network.ListNetworks()
			},
		},
		{
			Name:  "delete",
			Usage: "Delete the container network",
			Action: func(ctx *cli.Context) error {
				if len(ctx.Args()) < 1 {
					return fmt.Errorf("missing network name")
				}

				name := ctx.Args()[0]
				return network.DeleteNetwork(name)
			},
		},
	},
}
