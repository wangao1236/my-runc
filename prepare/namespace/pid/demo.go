package main

import (
	"os"
	"os/exec"
	"syscall"
)

func main() {
	// fork 出新进程的初试命令
	cmd := exec.Command("sh")
	// 设置系统调佣参数
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWIPC | syscall.CLONE_NEWPID,
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		panic(err)
	}
}
