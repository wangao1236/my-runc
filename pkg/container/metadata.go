package container

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/wangao1236/my-docker/pkg/util"
)

const (
	StatusRunning = "running"
	StatusStopped = "stopped"
	StatusExited  = "exited"

	DefaultMetadataRootDir = "/var/run/my-docker"
	defaultContainerDir    = "default"
	configName             = "config.json"
	logName                = "container.log"
)

func init() {
	if err := util.EnsureDirectory(DefaultMetadataRootDir); err != nil {
		logrus.Fatalf("faile to ensure metadata root diectory %v: %v", DefaultMetadataRootDir, err)
	}
}

// Metadata 表示容器元数据
type Metadata struct {
	PID        int       `json:"pid"`
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Command    string    `json:"command"`
	CreateTime time.Time `json:"createTime"`
	Status     string    `json:"status"`
	Volume     string    `json:"volume"`
}

// CreateMetadata 在容器创建时，将元数据存入配置文件中
func CreateMetadata(pid int, args []string, containerName, volume string) error {
	return SaveMetadata(&Metadata{
		PID:        pid,
		ID:         util.RandomString(10),
		Name:       containerName,
		Command:    strings.Join(args, " "),
		CreateTime: time.Now(),
		Status:     StatusRunning,
		Volume:     volume,
	})
}

// ReadMetadata 读取容器元数据
func ReadMetadata(containerName string) (*Metadata, error) {
	currentPath := generateConfigPath(containerName)
	body, err := ioutil.ReadFile(currentPath)
	if err != nil {
		logrus.Errorf("failed to read %v: %v", currentPath, err)
		return nil, err
	}
	metadata := &Metadata{}
	if err = json.Unmarshal(body, metadata); err != nil {
		logrus.Errorf("failed to unmarshal %v: %v", string(body), err)
		return nil, err
	}
	return metadata, nil
}

// SaveMetadata saves metadata locally.
func SaveMetadata(metadata *Metadata) error {
	body, err := json.Marshal(metadata)
	if err != nil {
		logrus.Errorf("failed to marshal metadata (%+v): %v", metadata, err)
		return err
	}

	containerName := metadata.Name
	metadataDir := generateMetadataDir(containerName)
	if err = util.EnsureDirectory(metadataDir); err != nil {
		logrus.Errorf("failed to ensure metadata directory %v: %v", metadataDir, err)
	}

	var file *os.File
	configPath := generateConfigPath(containerName)
	file, err = os.Create(configPath)
	if err != nil {
		logrus.Errorf("failed to create file of metadata (%v): %v", configPath, err)
		return err
	}

	if _, err = file.Write(body); err != nil {
		logrus.Errorf("failed to write (%v) to metadata file (%v): %v", string(body), configPath, err)
		return err
	}
	return nil
}

// RemoveMetadata 在容器退出时，把元数据删除
func RemoveMetadata(containerName string) error {
	metadataDir := generateMetadataDir(containerName)
	if err := os.RemoveAll(metadataDir); err != nil {
		logrus.Errorf("failed to remove metadata directory (%v): %v", metadataDir, err)
		return err
	}
	return nil
}

// CreateLogFile 创建日志文件
func CreateLogFile(containerName string) (*os.File, error) {
	metadataDir := generateMetadataDir(containerName)
	if err := util.EnsureDirectory(metadataDir); err != nil {
		logrus.Errorf("failed to ensure metadata directory %v: %v", metadataDir, err)
	}

	logPath := generateLogPath(containerName)
	file, err := os.Create(logPath)
	if err != nil {
		logrus.Errorf("failed to create file of metadata (%v): %v", logPath, err)
	}
	return file, err
}

func generateMetadataDir(containerName string) string {
	if len(containerName) == 0 {
		containerName = defaultContainerDir
	}
	return path.Join(DefaultMetadataRootDir, containerName)
}

func generateConfigPath(containerName string) string {
	return path.Join(generateMetadataDir(containerName), configName)
}

func generateLogPath(containerName string) string {
	return path.Join(generateMetadataDir(containerName), logName)
}
