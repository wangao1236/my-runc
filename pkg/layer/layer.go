package layer

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
	"syscall"

	"github.com/sirupsen/logrus"
	"github.com/wangao1236/my-docker/pkg/util"
)

func CreateWorkspace(rootDir, imageTar, containerName, volume string) (
	writeLayer, workLayer, workspace string, err error) {
	var readonlyPath string
	readonlyPath, err = createReadonlyLayer(rootDir, imageTar)
	if err != nil {
		logrus.Errorf("failed to create readonly layer: %v", err)
		return "", "", "", err
	}

	writeLayer, err = createWriteLayer(rootDir, containerName)
	if err != nil {
		logrus.Errorf("failed to create write layer: %v", err)
		return "", "", "", err
	}

	workLayer, err = createWorkLayer(rootDir, containerName)
	if err != nil {
		logrus.Errorf("failed to create work layer: %v", err)
		return "", "", "", err
	}

	workspace, err = createWorkspace(rootDir, containerName, readonlyPath, writeLayer, workLayer)
	if err != nil {
		logrus.Errorf("failed to create mount point: %v", err)
		return "", "", "", err
	}

	if err = mountVolume(workspace, volume); err != nil {
		logrus.Errorf("failed to mount volume %v: %v", volume, err)
		return "", "", "", err
	}
	return
}

func DeleteWorkspace(workspace, workLayer, writeLayer, volume string) {
	umountVolume(workspace, volume)
	cmd := exec.Command("umount", workspace)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		logrus.Errorf("failed to umount %v: %v", workspace, err)
	}
	if err := os.RemoveAll(workspace); err != nil {
		logrus.Errorf("failed to remove workspace %v: %v", workspace, err)
	}
	if err := os.RemoveAll(workLayer); err != nil {
		logrus.Errorf("failed to remove work layer %v: %v", workLayer, err)
	}
	if err := os.RemoveAll(writeLayer); err != nil {
		logrus.Errorf("failed to remove write layer %v: %v", writeLayer, err)
	}
}

func createReadonlyLayer(rootDir, imageTar string) (string, error) {
	// 创建 overlayFs 只读层目录
	targetPath := generateReadyOnlyDir(rootDir)
	if err := util.EnsureDirectory(targetPath); err != nil {
		logrus.Errorf("failed to ensure directory %v: %v", targetPath, err)
		return "", err
	}

	imagePath := path.Join(rootDir, imageTar)
	if _, err := exec.Command("tar", "-xvf", imagePath, "-C", targetPath).CombinedOutput(); err != nil {
		logrus.Errorf("failed to untar image %v: %v", imagePath, err)
		return "", err
	}
	return targetPath, nil
}

func createWriteLayer(rootDir, containerName string) (string, error) {
	// 创建 overlayFs 可读写层目录
	targetPath := GenerateWriteDir(rootDir, containerName)
	if err := util.EnsureDirectory(targetPath); err != nil {
		logrus.Errorf("failed to ensure directory %v: %v", targetPath, err)
		return "", err
	}
	return targetPath, nil
}

func createWorkLayer(rootDir, containerName string) (string, error) {
	// 创建 overlayFs 临时文件层目录
	targetPath := GenerateWorkDir(rootDir, containerName)
	if err := util.EnsureDirectory(targetPath); err != nil {
		logrus.Errorf("failed to ensure directory %v: %v", targetPath, err)
		return "", err
	}
	return targetPath, nil
}

func createWorkspace(rootDir, containerName, lowerPath, upperPath, workPath string) (string, error) {
	// 创建 overlayFs 挂载点目录
	targetPath := GenerateWorkSpaceDir(rootDir, containerName)
	if err := util.EnsureDirectory(targetPath); err != nil {
		logrus.Errorf("failed to ensure directory %v: %v", targetPath, err)
		return "", err
	}

	dirs := fmt.Sprintf("lowerdir=%v,upperdir=%v,workdir=%v", lowerPath, upperPath, workPath)
	cmd := exec.Command("mount", "-t", "overlay", "overlay", "-o", dirs, targetPath)
	if err := cmd.Run(); err != nil {
		logrus.Errorf("failed to mount overlayFs (%v): %v", cmd.String(), err)
		return "", err
	}
	return targetPath, nil
}

func mountVolume(workspace, volume string) error {
	if len(volume) == 0 {
		return nil
	}
	dirs := strings.Split(volume, ":")
	if len(dirs) < 2 {
		logrus.Warningf("invalid volume %v, need host path and in-container path", volume)
		return nil
	}

	hostPath := dirs[0]
	if err := util.EnsureDirectory(hostPath); err != nil {
		logrus.Errorf("failed to ensure host path %v when mounting volume for containers", hostPath)
		return err
	}

	inContainerPath := path.Join(workspace, dirs[1])
	if err := util.EnsureDirectory(inContainerPath); err != nil {
		logrus.Errorf("failed to ensure in-container path %v when mounting volume for containers", inContainerPath)
		return err
	}

	err := syscall.Mount(hostPath, inContainerPath, "bind", syscall.MS_BIND|syscall.MS_REC, "")
	if err != nil {
		logrus.Errorf("failed to mount %v to %v: %v", hostPath, inContainerPath, err)
	}
	return err
}

func umountVolume(workspace, volume string) {
	if len(volume) == 0 {
		return
	}
	dirs := strings.Split(volume, ":")
	if len(dirs) < 2 {
		logrus.Warningf("invalid volume %v, need host path and in-container path", volume)
		return
	}

	inContainerPath := path.Join(workspace, dirs[1])
	cmd := exec.Command("umount", inContainerPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		logrus.Errorf("failed to umount volume %v: %v", inContainerPath, err)
	}
}

func generateReadyOnlyDir(rootDir string) string {
	return path.Join(rootDir, ".read")
}

// GenerateWriteDir 生成可读写层目录的路径
func GenerateWriteDir(rootDir, containerName string) string {
	return path.Join(rootDir, ".write", containerName)
}

// GenerateWorkDir 生成临时文件层目录的路径
func GenerateWorkDir(rootDir, containerName string) string {
	return path.Join(rootDir, ".work", containerName)
}

// GenerateWorkSpaceDir 生成 container workspace 目录的路径
func GenerateWorkSpaceDir(rootDir, containerName string) string {
	return path.Join(rootDir, ".merge", containerName)
}

// GenerateImageTarPath 生成 container 的镜像的压缩文件的路径
func GenerateImageTarPath(rootDir, containerName string) string {
	return path.Join(rootDir, containerName+".tar")
}
