package cgroup

var (
	Subsystems = []Subsystem{
		&MemorySubsystem{},
	}
)

type ResourceConfig struct {
	MemoryLimit string
	CPUShare    string
	CPUSet      string
}

// Subsystem 对应 linux cgroup 的每一个 subsystem：
type Subsystem interface {
	// Name 返回 名称，如 memory、cpuset、cpushare
	Name() string
	// Set 写入配置文件，对资源进行限制
	Set(cgroupName string, res *ResourceConfig) error
	// Apply 将 PID 加入当前 Cgroup
	Apply(cgroupName string, pid int) error
	// Remove 将 PID 移出当前 Cgroup
	Remove(cgroupName string) error
}

type Manager struct {
	// CgroupName 表示当前进程的 Cgroup 名称：在 Cgroup 下建立的子文件夹名称
	CgroupName string
	Resource   *ResourceConfig
}

func NewManager(cgroupName string) *Manager {
	return &Manager{
		CgroupName: cgroupName,
	}
}

// Apply 将 PID 加入Cgroup
func (m *Manager) Apply(pid int) error {
	for _, ss := range Subsystems {
		err := ss.Apply(m.CgroupName, pid)
		if err != nil {
			return err
		}
	}
	return nil
}

// Set 设置资源限制
func (m *Manager) Set(res *ResourceConfig) error {
	for _, ss := range Subsystems {
		err := ss.Set(m.CgroupName, res)
		if err != nil {
			return err
		}
	}
	return nil
}

// Destroy 释放 Cgroup
func (m *Manager) Destroy() error {
	for _, ss := range Subsystems {
		err := ss.Remove(m.CgroupName)
		if err != nil {

			return err
		}
	}
	return nil
}
