package command

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"github.com/wangao1236/my-runc/pkg/cgroup"
	"github.com/wangao1236/my-runc/pkg/container"
	"github.com/wangao1236/my-runc/pkg/layer"
	"github.com/wangao1236/my-runc/pkg/network"
)

var RunCommand = cli.Command{
	Name:  "run",
	Usage: `Create a container with namespace and cgroups limit my-runc run -ti [command]`,
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
		cli.StringSliceFlag{
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
		cli.StringFlag{
			Name:  "network",
			Usage: "Network name used by containers",
		},
		cli.StringSliceFlag{
			Name:  "p",
			Usage: "Port mapping of containers",
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
		portMappings, err := parsePortMappings(ctx.StringSlice("p"))
		if err != nil {
			return fmt.Errorf("invalid port mappings: %v", err)
		}
		tty := ctx.Bool("it")
		detach := ctx.Bool("d")
		containerName := ctx.String("name")
		imageTar := ctx.String("image-tar")
		envs := ctx.StringSlice("e")
		args := ctx.Args()
		volumes := ctx.StringSlice("v")
		networkName := ctx.String("network")
		logrus.Infof("run args: %+v, container name: %v, enable tty: %v, detach: %v, environment variables: %+v",
			args, containerName, tty, detach, envs)
		Run(tty, detach, containerName, imageTar, networkName, envs, args, volumes, portMappings,
			&cgroup.ResourceConfig{
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
func Run(tty, detach bool, containerName string, imageTar, networkName string,
	envs, args, volumes []string, portMappings map[int]int, res *cgroup.ResourceConfig) {
	rootDir, err := os.Getwd()
	if err != nil {
		logrus.Fatalf("failed to get current directory: %v", err)
	}
	var writeLayer, workLayer, workspace string
	writeLayer, workLayer, workspace, err = layer.CreateWorkspace(rootDir, imageTar, containerName, volumes)
	if err != nil {
		logrus.Fatalf("failed to create workspace: %v", err)
	}
	defer func() {
		if !detach && err == nil {
			layer.DeleteWorkspace(workspace, workLayer, writeLayer, volumes)
		}
	}()

	cgroupManager := cgroup.NewManager("my-runc-cgroup")
	defer func() {
		if !detach && err == nil {
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

	if err = container.CreateMetadata(parent.Process.Pid, args, containerName, volumes); err != nil {
		logrus.Fatalf("failed to record metadata of container (%v): %v", containerName, err)
	}
	defer func() {
		if !detach && err == nil {
			if err = container.RemoveMetadata(containerName); err != nil {
				logrus.Warningf("failed to remove metadata of container (%v): %v", containerName, err)
			}
		}
	}()

	if err = cgroupManager.Apply(parent.Process.Pid); err != nil {
		logrus.Fatalf("failed to apply pid (%v) of parent process to cgroups; %v", parent.Process.Pid, err)
	}
	logrus.Infof("applied pid (%v) of parent process to cgroups successfully", parent.Process.Pid)

	if len(networkName) > 0 {
		var metadata *container.Metadata
		metadata, err = container.ReadMetadata(containerName)
		if err != nil {
			logrus.Fatalf("failed to get metadata of container (%v) for setting contaienr network: %v",
				containerName, err)
			return
		}
		if err = network.Connect(networkName, portMappings, metadata); err != nil {
			logrus.Fatalf("failed to connect network (%v) for container (%v): %v", networkName, metadata, err)
		}
		logrus.Infof("succeeded in connecting network (%v) for container (%v)", networkName, metadata.Name)
		defer func() {
			if !detach {
				if err = network.Disconnect(metadata); err != nil {
					logrus.Warningf("failed to disconnect network for container (%v): %v", metadata, err)
				}
			}
		}()
	}

	sendInitArgs(args, writePipe)

	logrus.Infof("parent process started successfully, detach: %v", detach)
	if !detach {
		if err = parent.Wait(); err != nil {
			logrus.Errorf("failed to wait parent process stopping: %v", err)
		}
		logrus.Info("parent process stopped")
	}
}

func sendInitArgs(args []string, writePipe *os.File) {
	if _, err := writePipe.WriteString(strings.Join(args, " ")); err != nil {
		logrus.Fatalf("write string to pipe for args (%+v) failed: %v", args, err)
	}
	if err := writePipe.Close(); err != nil {
		logrus.Errorf("failed to close write pipe: %v", err)
	}
}

func parsePortMappings(portMappings []string) (map[int]int, error) {
	result := make(map[int]int)
	var err error
	var hostPort, containerPort int64
	for _, kv := range portMappings {
		splits := strings.Split(kv, ":")
		if len(splits) < 2 {
			return nil, fmt.Errorf("invalid port mapping: %v", kv)
		}
		hostPort, err = strconv.ParseInt(splits[0], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid host port in port mappings: %v", splits[0])
		}
		containerPort, err = strconv.ParseInt(splits[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid container port in port mappings: %v", splits[1])
		}
		if _, ok := result[int(hostPort)]; ok {
			return nil, fmt.Errorf("duplicate host port in port mappings: %v", hostPort)
		}
		result[int(hostPort)] = int(containerPort)
	}
	return result, nil
}
