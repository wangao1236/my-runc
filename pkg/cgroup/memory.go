package cgroup

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"

	"github.com/sirupsen/logrus"
	"github.com/wangao1236/my-docker/pkg/util"
)

var _ Subsystem = &MemorySubsystem{}

type MemorySubsystem struct {
}

func (s *MemorySubsystem) Name() string {
	return "memory"
}

func (s *MemorySubsystem) Set(cgroupName string, res *ResourceConfig) error {
	// 获取自定义Cgroup的路径，没有则创建，如：/sys/fs/cgroup/memory/mydocker-cgroup
	cgroupPath, err := util.GetCgroupPath(s.Name(), cgroupName)
	if err != nil {
		logrus.Infof("failed to get path of cgroup (%v) in (%v): %v", cgroupName, s.Name(), err)
		return err
	}
	logrus.Infof("path of cgroup (%v) in (%v) is %v", cgroupName, s.Name(), cgroupPath)

	// 将资源限制写入
	limitFilePath := path.Join(cgroupPath, "memory.limit_in_bytes")
	if err = ioutil.WriteFile(limitFilePath, []byte(res.MemoryLimit), 0644); err != nil {
		logrus.Errorf("failed to set memory limit of %v: %v", cgroupName, err)
		return fmt.Errorf("failed to set memory limit of %v: %v", cgroupName, err)
	}
	return nil
}

func (s MemorySubsystem) Apply(cgroupName string, pid int) error {
	// 获取自定义 Cgroup 的路径，没有则创建，如：/sys/fs/cgroup/memory/my-docker-cgroup
	cgroupPath, err := util.GetCgroupPath(s.Name(), cgroupName)
	if err != nil {
		logrus.Infof("failed to get path of cgroup (%v) in (%v): %v", cgroupName, s.Name(), err)
		return err
	}
	logrus.Infof("path of cgroup (%v) in (%v) is %v", cgroupName, s.Name(), cgroupPath)

	// 将 PID 加入该 cgroup
	limitFilePath := path.Join(cgroupPath, "tasks")
	if err = ioutil.WriteFile(limitFilePath, []byte(strconv.Itoa(pid)), 0644); err != nil {
		logrus.Errorf("failed to add pid to %v: %v", cgroupName, err)
		return fmt.Errorf("failed to add pid to %v: %v", cgroupName, err)
	}
	return nil
}

func (s *MemorySubsystem) Remove(cgroupName string) error {
	// 获取自定义 Cgroup 的路径，没有则创建，如：/sys/fs/cgroup/memory/my-docker-cgroup
	cgroupPath, err := util.GetCgroupPath(s.Name(), cgroupName)
	if err != nil {
		logrus.Errorf("failed to get path of cgroup (%v) in (%v): %v", cgroupName, s.Name(), err)
		return err
	}
	logrus.Infof("try to remove path of cgroup (%v) in (%v) is %v", cgroupName, s.Name(), cgroupPath)
	return os.RemoveAll(cgroupPath)
}
