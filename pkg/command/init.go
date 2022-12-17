package command

import (
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"github.com/wangao1236/my-runc/pkg/container"
)

var InitCommand = cli.Command{
	Name:  "init",
	Usage: "Init container process run user's process in container. Do not call it outside",

	// 1. 获取传递过来的 command 参数；
	// 2. 执行容器初始化操作。
	Action: func(ctx *cli.Context) error {
		logrus.Infof("init args: %+v", ctx.Args())
		return container.RunContainerInitProcess()
	},
}
