package container

import (
	"os"
	"os/exec"
	"syscall"

	"github.com/sirupsen/logrus"
	"github.com/wangao1236/my-docker/pkg/util"
)

// NewParentProcess 构造出一个 command：
// 1. 调用 /proc/self/exe，使用这种方式对创造出来的进程进行初始化，并隔离新的 namespace 中执行
// 2. 其中 init 是传递给本进程的第一个参数，表示 fork 出的进程会执行我们的 init 命令
// 3. 如果用户指定了 -it 参数，就需要把当前进程的输入输出导入到标准输入输出上
func NewParentProcess(tty bool, args []string) *exec.Cmd {
	newArgs := append([]string{"init"}, args...)
	cmd := exec.Command("/proc/self/exe", newArgs...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS |
			syscall.CLONE_NEWNET | syscall.CLONE_NEWIPC,
	}
	if tty {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	return cmd
}

// RunContainerInitProcess 是在容器内部执行的，会执行一些初始化操作。
// 代码执行到这里时，容器所在的进程其实就已经创建出来了，这是本容器执行的第一个进程。
// 使用 mount 先去挂载 proc 文件系统，
// 然后执行 execve 替换掉 /proc/self/exe，将用户传入的命令参数，作为 1 号进程
func RunContainerInitProcess(args []string) error {
	logrus.Infof("init container for args: %+v", args)

	if err := syscall.Mount("", "/", "", syscall.MS_PRIVATE|syscall.MS_REC, ""); err != nil {
		logrus.Errorf("failed to mount root in private way: %v", err)
		return err
	}

	defaultMountFlags := syscall.MS_NOEXEC | syscall.MS_NOSUID | syscall.MS_NODEV
	if err := syscall.Mount("proc", "/proc", "proc", uintptr(defaultMountFlags), ""); err != nil {
		logrus.Errorf("mount /proc failed: %v", err)
		return err
	}

	processOne, err := util.ShowProcessOne()
	if err != nil {
		logrus.Errorf("failed to get process one: %v", err)
		return err
	}
	logrus.Infof("old process one: \n%v", processOne)

	// 同时，我们使用 lookPath 的方式去查找命令进行执行
	var path string
	path, err = exec.LookPath(args[0])
	if err != nil {
		logrus.Errorf("can't find exec path %v: %v", args[0], err)
		return err
	}
	logrus.Infof("find path: %s", path)
	args[0] = path
	if err = syscall.Exec(args[0], args, os.Environ()); err != nil {
		logrus.Errorf("exec (%+v) failed: %v", args, err)
		return err
	}
	processOne, err = util.ShowProcessOne()
	if err != nil {
		logrus.Errorf("failed to get process one: %v", err)
		return err
	}
	logrus.Infof("new process one: \n%v", processOne)
	return nil
}
