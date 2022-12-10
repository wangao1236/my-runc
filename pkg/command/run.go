package command

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"github.com/wangao1236/my-docker/pkg/cgroup"
	"github.com/wangao1236/my-docker/pkg/container"
	"github.com/wangao1236/my-docker/pkg/layer"
)

var RunCommand = cli.Command{
	Name:  "run",
	Usage: `Create a container with namespace and cgroups limit my-docker run -ti [command]`,
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "it",
			Usage: "Enable tty",
		},
		// 增加内存等限制参数
		cli.StringFlag{
			Name:  "mem",
			Usage: "Memory limit",
		},
		cli.StringFlag{
			Name:  "cpu-set",
			Usage: "CPU set limit",
		},
		cli.StringFlag{
			Name:  "cpu-share",
			Usage: "CPU share limit",
		},
		cli.StringFlag{
			Name:  "image-tar",
			Value: "busybox.tar",
			Usage: "Image tar file name",
		},
		cli.StringFlag{
			Name:  "v",
			Usage: "Volume",
		},
		cli.BoolFlag{
			Name:  "d",
			Usage: "Detach container",
		},
		cli.StringFlag{
			Name:  "name",
			Usage: "Container name",
		},
		cli.StringSliceFlag{
			Name:  "e",
			Usage: "Environment variables in containers",
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
		tty := ctx.Bool("it")
		detach := ctx.Bool("d")
		containerName := ctx.String("name")
		imageTar := ctx.String("image-tar")
		volume := ctx.String("v")
		envs := ctx.StringSlice("e")
		logrus.Infof("run args: %+v, container name: %v, enable tty: %v, detach: %v, environment variables: %+v",
			ctx.Args(), containerName, tty, detach, envs)
		Run(tty, detach, containerName, imageTar, volume, envs, ctx.Args(), &cgroup.ResourceConfig{
			MemoryLimit: ctx.String("mem"),
			CPUShare:    ctx.String("cpu-share"),
			CPUSet:      ctx.String("cpu-set"),
		})
		return nil
	},
}

// Run fork 出当前进程，执行 init 命令。
// 它首先会 clone 出来一批 namespace 隔离的进程，然后在子进程中，调用 /proc/self/exe，也就是自己调用自己。
// 发送 init 参数，调用我们写的 init 方法，去初始化容器的一些资源
func Run(tty, detach bool, containerName string, imageTar, volume string, envs, args []string,
	res *cgroup.ResourceConfig) {
	rootDir, err := os.Getwd()
	if err != nil {
		logrus.Fatalf("failed to get current directory: %v", err)
	}
	var writeLayer, workLayer, workspace string
	writeLayer, workLayer, workspace, err = layer.CreateWorkspace(rootDir, imageTar, containerName, volume)
	if err != nil {
		logrus.Fatalf("failed to create workspace: %v", err)
	}
	defer func() {
		if !detach {
			layer.DeleteWorkspace(workspace, workLayer, writeLayer, volume)
		}
	}()

	cgroupManager := cgroup.NewManager("my-docker-cgroup")
	defer func() {
		if !detach {
			if err = cgroupManager.Destroy(); err != nil {
				logrus.Warningf("failed to destroy cgroups: %v", err)
			} else {
				logrus.Info("destroy cgroups successfully")
			}
		}
	}()
	if err = cgroupManager.Set(res); err != nil {
		logrus.Fatalf("failed to set resource config to cgroups for parent process: %v", err)
	}
	logrus.Infof("set resource (%+v) to cgroups for parent process", res)

	var parent *exec.Cmd
	var writePipe *os.File
	parent, writePipe, err = container.NewParentProcess(tty, workspace, containerName, envs)
	if err != nil {
		logrus.Fatalf("failed to build parent process: %v", err)
	}
	logrus.Infof("parent process command: %v", parent.String())
	if err = parent.Start(); err != nil {
		logrus.Fatalf("parent process failed to start: %v", err)
	}

	if err = container.CreateMetadata(parent.Process.Pid, args, containerName, volume); err != nil {
		logrus.Fatalf("failed to record metadata of container (%v): %v", containerName, err)
	}
	defer func() {
		if !detach {
			if err = container.RemoveMetadata(containerName); err != nil {
				logrus.Warningf("failed to remove metadata of container (%v): %v", containerName, err)
			}
		}
	}()

	if err = cgroupManager.Apply(parent.Process.Pid); err != nil {
		logrus.Fatalf("failed to apply pid (%v) of parent process to cgroups; %v", parent.Process.Pid, err)
	}
	logrus.Infof("applied pid (%v) of parent process to cgroups successfully", parent.Process.Pid)

	sendInitArgs(args, writePipe)

	logrus.Infof("parent process started successfully, detach: %v", detach)
	if !detach {
		if err = parent.Wait(); err != nil {
			logrus.Errorf("failed to wait parent process stopping: %v", err)
		}
	}
	logrus.Info("parent process stopped")
}

func sendInitArgs(args []string, writePipe *os.File) {
	if _, err := writePipe.WriteString(strings.Join(args, " ")); err != nil {
		logrus.Fatalf("write string to pipe for args (%+v) failed: %v", args, err)
	}
	if err := writePipe.Close(); err != nil {
		logrus.Errorf("failed to close write pipe: %v", err)
	}
}
