package util

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/sirupsen/logrus"
)

func ShowProcessOne() (string, error) {
	cmd := exec.Command("ps", "-c", "1")
	output, err := cmd.Output()
	return string(output), err
}

func GetCgroupPath(subsystem, cgroupName string) (string, error) {
	// 找到 Cgroup 的根目录，如：/sys/fs/cgroup/memory
	cgroupRoot, err := FindCgroupMountPoint(subsystem)
	if err != nil {
		return "", err
	}

	cgroupPath := path.Join(cgroupRoot, cgroupName)
	_, err = os.Stat(cgroupPath)
	if err != nil {
		if !os.IsNotExist(err) {
			logrus.Errorf("failed to get cgroup path (%v): %v", cgroupPath, err)
			return "", fmt.Errorf("get cgroup path (%v) failed: %v", cgroupPath, err)
		}
		if err = os.Mkdir(cgroupPath, os.ModePerm); err != nil {
			logrus.Errorf("failed to mkdir (%v): %v", cgroupPath, err)
			return "", fmt.Errorf("failed to mkdir (%v): %v", cgroupPath, err)
		}
		//return cgroupPath, ioutil.WriteFile(path.Join(cgroupPath, "memory.oom_control"), []byte("1"), 0644)
	}
	return cgroupPath, nil
}

func FindCgroupMountPoint(subsystem string) (string, error) {
	f, err := os.Open("/proc/self/mountinfo")
	if err != nil {
		return "", fmt.Errorf("open /proc/self/mountinfo err: %v", err)
	}
	defer func() {
		if err = f.Close(); err != nil {
			logrus.Warningf("close mountinfo failed: %v", err)
		}
	}()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		txt := scanner.Text()
		fields := strings.Split(txt, " ")
		lastField := fields[len(fields)-1]
		for _, opt := range strings.Split(lastField, ",") {
			if opt == subsystem {
				return fields[4], nil
			}
		}
	}
	if err = scanner.Err(); err != nil {
		logrus.Errorf("file scanner err: %v", err)
		return "", fmt.Errorf("file scanner err: %v", err)
	}
	logrus.Errorf("subsystem %v is not found", subsystem)
	return "", fmt.Errorf("subsystem %v is not found", subsystem)
}
