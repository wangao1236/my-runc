package main

import (
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"strconv"
	"syscall"
	"time"
)

const cgroupMemoryHierarchyMount = "/sys/fs/cgroup/memory"

func main() {
	if os.Args[0] == "/proc/self/exe" {
		log.Printf("inner pid: %v, ppid: %v\n", syscall.Getpid(), syscall.Getppid())
		cmd := exec.Command("stress", "--vm-bytes", "200m", "--vm-keep", "-m", "1")
		cmd.SysProcAttr = &syscall.SysProcAttr{}
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		go func() {
			time.Sleep(time.Second)
			log.Printf("stress pid: %v\n", cmd.Process.Pid)
		}()
		if err := cmd.Run(); err != nil {
			log.Printf("run (%v) error: %v\n", cmd, err)
		}
	}

	cmd := exec.Command("/proc/self/exe")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWNS,
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		log.Printf("start error: %v\n", err)
		os.Exit(1)
	} else {
		log.Printf("outer pid: %v -> %v\n", cmd, cmd.Process.Pid)
		_ = os.MkdirAll(path.Join(cgroupMemoryHierarchyMount, "testMemoryLimit"), 0755)
		log.Println("create cgroup")
		_ = ioutil.WriteFile(path.Join(cgroupMemoryHierarchyMount, "testMemoryLimit", "tasks"),
			[]byte(strconv.Itoa(cmd.Process.Pid)), 0644)
		// TODO: exec `echo 1 >> memory.oom_control` in new cgroup
		log.Println("move current process to new cgroup")
		_ = ioutil.WriteFile(path.Join(cgroupMemoryHierarchyMount, "testMemoryLimit", "memory.limit_in_bytes"),
			[]byte("100m"), 0644)
		log.Println("limit memory")
	}
	_, err := cmd.Process.Wait()
	if err != nil {
		log.Printf("wait error: %v\n", err)
		os.Exit(1)
	}
	log.Println("wait exit")
}
