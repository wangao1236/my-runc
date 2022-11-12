package command

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"github.com/wangao1236/my-docker/pkg/container"
)

var RunCommand = cli.Command{
	Name:  "run",
	Usage: `Create a container with namespace and cgroups limit mydocker run -ti [command]`,
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "it",
			Usage: "Enable tty",
		},
	},

	// 这里是 run 命令执行的真正函数：
	// 1. 判断参数是否包含 command；
	// 2. 获取用户指定的 command；
	// 3. 调用 Run function 去准备启动容器。
	Action: func(ctx *cli.Context) error {
		if len(ctx.Args()) < 1 {
			return fmt.Errorf("missing container command")
		}
		logrus.Infof("run args: %+v", ctx.Args())
		tty := ctx.Bool("it")
		Run(tty, ctx.Args())
		return nil
	},
}

// Run fork 出当前进程，执行 init 命令。
// 它首先会 clone 出来一批 namespace 隔离的进程，然后在子进程中，调用 /proc/self/exe，也就是自己调用自己。
// 发送 init 参数，调用我们写的 init 方法，去初始化容器的一些资源
func Run(tty bool, args []string) {
	parent := container.NewParentProcess(tty, args)
	logrus.Infof("parent process command: %v", parent.String())
	if err := parent.Start(); err != nil {
		logrus.Fatalf("parent process failed to start: %v", err)
	}
	logrus.Info("parent process started successfully")
	if err := parent.Wait(); err != nil {
		logrus.Errorf("failed to wait parent process stopping: %v", err)
	}
	logrus.Info("parent process stopped")
	os.Exit(-1)
}
