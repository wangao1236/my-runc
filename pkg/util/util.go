package util

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
	"syscall"

	"github.com/sirupsen/logrus"
)

func ShowProcessOne() (string, error) {
	cmd := exec.Command("ps", "-c", "1")
	output, err := cmd.Output()
	return string(output), err
}

func ShowProcessesInSpecifyPath(outputPath string) (string, error) {
	if _, err := os.Stat(outputPath); err == nil {
		if err = os.RemoveAll(outputPath); err != nil {
			return "", err
		}
	}
	if _, err := os.Create(outputPath); err != nil {
		return "", err
	}
	if err := ioutil.WriteFile(outputPath, []byte(""), 0777); err != nil {

	}
	cmd := exec.Command("ps")
	cmd.Stdin = os.NewFile(uintptr(syscall.Stdin), outputPath)
	cmd.Stderr = os.NewFile(uintptr(syscall.Stderr), outputPath)
	cmd.Stdout = os.NewFile(uintptr(syscall.Stdout), outputPath)
	if err := cmd.Run(); err != nil {
		logrus.Errorf("failed to run cmd toward %v: %v", outputPath, err)
		return "", err
	}
	output, err := ioutil.ReadFile(outputPath)
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

func EnsureDirectory(targetPath string) error {
	if fi, err := os.Stat(targetPath); err == nil && fi != nil {
		return nil
	}
	if err := os.MkdirAll(targetPath, 0777); err != nil {
		logrus.Errorf("failed to mkdir target path %v: %v", targetPath, err)
		return err
	}
	return nil
}
