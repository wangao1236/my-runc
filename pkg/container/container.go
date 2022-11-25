package container

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
	"syscall"

	"github.com/sirupsen/logrus"
	"github.com/wangao1236/my-docker/pkg/util"
)

// NewParentProcess 构造出一个 command：
// 1. 调用 /proc/self/exe，使用这种方式对创造出来的进程进行初始化，并隔离新的 namespace 中执行
// 2. 其中 init 是传递给本进程的第一个参数，表示 fork 出的进程会执行我们的 init 命令
// 3. 如果用户指定了 -it 参数，就需要把当前进程的输入输出导入到标准输入输出上
func NewParentProcess(tty bool, workspace string) (*exec.Cmd, *os.File, error) {
	readPipe, writePipe, err := newPipe()
	if err != nil {
		return nil, nil, err
	}
	cmd := exec.Command("/proc/self/exe", "init")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS |
			syscall.CLONE_NEWNET | syscall.CLONE_NEWIPC,
	}
	if tty {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	cmd.Dir = workspace
	// 将管道的一端传入 fork 的进程中
	cmd.ExtraFiles = []*os.File{readPipe}
	return cmd, writePipe, nil
}

// RunContainerInitProcess 是在容器内部执行的，会执行一些初始化操作。
// 代码执行到这里时，容器所在的进程其实就已经创建出来了，这是本容器执行的第一个进程。
// 使用 mount 先去挂载 proc 文件系统，
// 然后执行 execve 替换掉 /proc/self/exe，将用户传入的命令参数，作为 1 号进程
func RunContainerInitProcess() error {
	if err := setUpMount(); err != nil {
		logrus.Errorf("failed to set up mount: %v", err)
		return err
	}

	args := readArgs()
	logrus.Infof("init container for args: %+v", args)

	processOne, err := util.ShowProcessesInSpecifyPath("./old-process-one")
	if err != nil {
		logrus.Errorf("failed to get process one: %v", err)
		return err
	}
	logrus.Infof("old process one: \n%v", processOne)

	// 同时，我们使用 lookPath 的方式去查找命令进行执行
	var execPath string
	execPath, err = exec.LookPath(args[0])
	if err != nil {
		logrus.Errorf("can't find exec path %v: %v", args[0], err)
		return err
	}
	logrus.Infof("find exec path: %s", execPath)
	args[0] = execPath
	if err = syscall.Exec(args[0], args, os.Environ()); err != nil {
		logrus.Errorf("exec (%+v) failed: %v", args, err)
		return err
	}
	processOne, err = util.ShowProcessesInSpecifyPath("./new-process-one")
	if err != nil {
		logrus.Errorf("failed to get process one: %v", err)
		return err
	}
	logrus.Infof("new process one: \n%v", processOne)
	return nil
}

func CommitContainer(imageName string) error {
	rootDir, err := os.Getwd()
	if err != nil {
		logrus.Fatalf("failed to get current directory: %v", err)
	}
	workspace := path.Join(rootDir, ".merge")
	imageTar := path.Join(rootDir, imageName)
	logrus.Infof("try to commit image to %v", imageTar)
	if _, err = exec.Command("tar", "-czf", imageTar, "-C", workspace, ".").CombinedOutput(); err != nil {
		return fmt.Errorf("failed to commit image to %v: %v", workspace, err)
	}
	return nil
}

func newPipe() (*os.File, *os.File, error) {
	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		logrus.Errorf("failed to create io pipe: %v", err)
		return nil, nil, err
	}
	return readPipe, writePipe, nil
}

func readArgs() []string {
	readPipe := os.NewFile(uintptr(3), "pipe")
	msg, err := ioutil.ReadAll(readPipe)
	if err != nil {
		logrus.Errorf("failed to read args from pipe: %v", err)
		return nil
	}
	return strings.Split(string(msg), " ")
}

func setUpMount() error {
	if err := syscall.Mount("/", "/", "", syscall.MS_REC|syscall.MS_PRIVATE, ""); err != nil {
		logrus.Errorf("failed to mount root in private way: %v", err)
		return err
	}

	pwd, err := os.Getwd()
	if err != nil {
		logrus.Errorf("get current directory failed: %v", err)
		return fmt.Errorf("get current directory failed: %v", err)
	}
	logrus.Infof("current directory is %v", pwd)

	if err = pivotRoot(pwd); err != nil {
		logrus.Errorf("failed to change pivot root: %v", err)
		return fmt.Errorf("failed to change pivot root: %v", err)
	}

	pwd, err = os.Getwd()
	if err != nil {
		logrus.Errorf("get current directory failed: %v", err)
		return fmt.Errorf("get current directory failed: %v", err)
	}
	logrus.Infof("current directory is %v", pwd)

	defaultMountFlags := syscall.MS_NOEXEC | syscall.MS_NOSUID | syscall.MS_NODEV
	if err = syscall.Mount("proc", "/proc", "proc", uintptr(defaultMountFlags), ""); err != nil {
		logrus.Errorf("mount /proc failed: %v", err)
		return err
	}
	return syscall.Mount("tmpfs", "/dev", "tmpfs", syscall.MS_NOSUID|syscall.MS_STRICTATIME, "mode=755")
}

func pivotRoot(root string) error {
	// 参考：https://blog.csdn.net/weixin_43988498/article/details/121307202
	// privot_root new_root 和 put_old 不能与当前根目录在同一个挂载上，且 new_root 本身也需要是一个挂载点。
	// 因此，为了使当前 root 的老 root 和新 root 不在同一个文件系统下，我们把 root 重新 mount 一次。
	// bind mount 是把相同的内容换了一个挂载点的挂载方法。
	if err := syscall.Mount(root, root, "bind", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
		logrus.Errorf("failed to bind mount rootfs to itself: %v", err)
		return fmt.Errorf("failed to bind mount rootfs to itself: %v", err)
	}

	putOld := path.Join(root, ".pivot_root")
	if _, err := os.Stat(putOld); err == nil {
		if err = os.RemoveAll(putOld); err != nil {
			logrus.Errorf("failed to remove existing put_old directory failed: %v", err)
			return fmt.Errorf("failed to remove existing put_old directory: %v", err)
		}
	}

	if err := os.MkdirAll(putOld, 0777); err != nil {
		logrus.Errorf("mkdir put_old directory failed: %v", err)
		return fmt.Errorf("failed to mkdir put_old directory: %v", err)
	}

	// pivot_root 到新的 rootfs，老的 old_root 现在挂载在 rootfs/.pivot_root 上
	// 挂载点目前依然可以在 mount 命令中看到
	if err := syscall.PivotRoot(root, putOld); err != nil {
		logrus.Errorf("failed to pivot_root: %v", err)
		return fmt.Errorf("failed to pivot_root: %v", err)
	}

	//mount, err := util.ShowMount()
	//if err != nil {
	//	logrus.Errorf("failed to get mount: %v", err)
	//	return fmt.Errorf("failed to get mount: %v", err)
	//}
	//logrus.Infof("old mount: \n%v", mount)

	if err := syscall.Chdir("/"); err != nil {
		logrus.Errorf("failed to chdir root: %v", err)
		return fmt.Errorf("failed to chdir root: %v", err)
	}

	//mount, err = util.ShowMount()
	//if err != nil {
	//	logrus.Errorf("failed to get mount: %v", err)
	//	return fmt.Errorf("failed to get mount: %v", err)
	//}
	//logrus.Infof("new mount: \n%v", mount)

	// 取消临时文件 .pivot_root 的挂载并删除它
	// 注意当前已经在根目录下，所以临时文件的目录也改变了
	putOld = path.Join("/", ".pivot_root")
	if err := syscall.Unmount(putOld, syscall.MNT_DETACH); err != nil {
		logrus.Errorf("failed to umount put_old directory: %v", err)
		return fmt.Errorf("failed to umount put_old directory: %v", err)
	}
	return os.Remove(putOld)
}
