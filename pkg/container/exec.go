package container

import (
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
	_ "github.com/wangao1236/my-docker/pkg/nsenter"
)

const (
	EnvMyDockerPID = "my_docker_pid"
	EnvMyDockerCMD = "my_docker_cmd"
)

// ExecContainer 进入容器内的 Namespace
func ExecContainer(containerName string, args []string) error {
	metadata, err := ReadMetadata(containerName)
	if err != nil {
		logrus.Errorf("failed to read metadata of container (%v): %v", containerName, err)
		return err
	}

	logrus.Infof("exec container with pid: %v", metadata.PID)
	logrus.Infof("exec container with args: %+v", args)

	if err = os.Setenv(EnvMyDockerPID, strconv.Itoa(metadata.PID)); err != nil {
		logrus.Errorf("failed to set env of (%v): %v", EnvMyDockerPID, err)
		return err
	}

	if err = os.Setenv(EnvMyDockerCMD, strings.Join(args, " ")); err != nil {
		logrus.Errorf("failed to set env of (%v): %v", EnvMyDockerCMD, err)
		return err
	}

	cmd := exec.Command("/proc/self/exe", "exec")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err = cmd.Run(); err != nil {
		logrus.Errorf("failed to exec container %v with %+v: %v", containerName, args, err)
		return err
	}
	return nil
}
