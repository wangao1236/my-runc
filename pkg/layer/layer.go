package layer

import (
	"fmt"
	"os"
	"os/exec"
	"path"

	"github.com/sirupsen/logrus"
	"github.com/wangao1236/my-docker/pkg/util"
)

func CreateWorkspace(rootDir, imageTar, readonlyDir, writeDir, workDir, mountDir string) (
	writeLayer string, workLayer string, workspace string, err error) {
	var readonlyPath string
	readonlyPath, err = createReadonlyLayer(rootDir, readonlyDir, imageTar)
	if err != nil {
		logrus.Errorf("failed to create readonly layer: %v", err)
		return "", "", "", err
	}

	writeLayer, err = createWriteLayer(rootDir, writeDir)
	if err != nil {
		logrus.Errorf("failed to create write layer: %v", err)
		return "", "", "", err
	}

	workLayer, err = createWorkLayer(rootDir, workDir)
	if err != nil {
		logrus.Errorf("failed to create work layer: %v", err)
		return "", "", "", err
	}

	workspace, err = createWorkspace(rootDir, mountDir, readonlyPath, writeLayer, workLayer)
	if err != nil {
		logrus.Errorf("failed to create mount point: %v", err)
		return "", "", "", err
	}

	return
}

func DeleteWorkspace(workspace, workLayer, writeLayer string) {
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

func createReadonlyLayer(rootDir, imageDir, imageTar string) (string, error) {
	// 创建 overlayFs 只读层目录
	targetPath := path.Join(rootDir, imageDir)
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

func createWriteLayer(rootDir, writeDir string) (string, error) {
	// 创建 overlayFs 可读写层目录
	targetPath := path.Join(rootDir, writeDir)
	if err := util.EnsureDirectory(targetPath); err != nil {
		logrus.Errorf("failed to ensure directory %v: %v", targetPath, err)
		return "", err
	}
	return targetPath, nil
}

func createWorkLayer(rootDir, workDir string) (string, error) {
	// 创建 overlayFs 临时文件层目录
	targetPath := path.Join(rootDir, workDir)
	if err := util.EnsureDirectory(targetPath); err != nil {
		logrus.Errorf("failed to ensure directory %v: %v", targetPath, err)
		return "", err
	}
	return targetPath, nil
}

func createWorkspace(rootDir, mountDir, lowerPath, upperPath, workPath string) (string, error) {
	// 创建 overlayFs 挂载点目录
	targetPath := path.Join(rootDir, mountDir)
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