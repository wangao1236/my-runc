package network

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path"
	"runtime"
	"sort"
	"text/tabwriter"

	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
	"github.com/wangao1236/my-docker/pkg/container"
	"github.com/wangao1236/my-docker/pkg/types"
	"github.com/wangao1236/my-docker/pkg/util"
)

const (
	DefaultNetworkRootDir = "/var/run/my-docker/network"
	DefaultNetworkDir     = DefaultNetworkRootDir + "/networks"
)

func init() {
	if err := util.EnsureDirectory(DefaultNetworkDir); err != nil {
		logrus.Warningf("faile to ensure network diectory %v: %v", DefaultNetworkDir, err)
	}
}

type Network struct {
	Name    string     `json:"name"`
	Driver  string     `json:"driver"`
	Subnet  *net.IPNet `json:"subnet"`
	Gateway *net.IPNet `json:"gateway"`
}

func (n *Network) String() string {
	body, _ := json.Marshal(n)
	return string(body)
}

func CreateNetwork(driver, subnet, name string) error {
	network, _ := readNetwork(name)
	if network != nil {
		logrus.Warningf("network %v already exists", name)
		return fmt.Errorf("network %v already exists", name)
	}

	if _, ok := drivers[driver]; !ok {
		logrus.Errorf("do not support driver %v", driver)
		return fmt.Errorf("do not support driver %v", driver)
	}

	var err error
	network, err = drivers[driver].CreateNetwork(name, subnet)
	if err != nil {
		logrus.Errorf("failed to create network (%v): %v", name, err)
		return err
	}
	logrus.Infof("network %v has been created successfully", network)

	if err = saveNetwork(network); err != nil {
		logrus.Errorf("failed to save network (%v): %v", name, err)
		return err
	}
	logrus.Infof("network %v has been saved successfully", network)
	return nil
}

func ListNetworks() error {
	files, err := ioutil.ReadDir(DefaultNetworkDir)
	if err != nil {
		logrus.Errorf("failed to read directory (%v): %v", DefaultNetworkDir, err)
		return err
	}

	networks := make([]*Network, len(files))
	var network *Network
	for i := range files {
		if files[i].IsDir() {
			continue
		}
		network, err = readNetwork(files[i].Name())
		if err != nil {
			logrus.Errorf("failed to read network (%v): %v", files[i].Name(), err)
			return err
		}
		networks[i] = network
	}
	sort.Slice(networks, func(i, j int) bool {
		return networks[i].Name < networks[j].Name
	})

	w := tabwriter.NewWriter(os.Stdout, 12, 1, 3, ' ', 0)
	_, _ = fmt.Fprint(w, "NAME\tSUBNET\tGATEWAY\tDRIVER\n")
	for _, n := range networks {
		_, _ = fmt.Fprintf(w, "%v\t%v\t%v\t%v\n", n.Name, n.Subnet, n.Gateway.IP, n.Driver)
	}
	if err = w.Flush(); err != nil {
		return fmt.Errorf("flush ps write err: %v", err)
	}
	return nil
}

func DeleteNetwork(name string) error {
	network, err := readNetwork(name)
	if err != nil {
		logrus.Errorf("failed to get network (%v): %v", name, err)
		return err
	}

	if _, ok := drivers[network.Driver]; !ok {
		logrus.Errorf("do not support driver %v", network.Driver)
		return fmt.Errorf("do not support driver %v", network.Driver)
	}

	if err = drivers[network.Driver].DeleteNetwork(network); err != nil {
		logrus.Errorf("failed to delete network (%v): %v", name, err)
		return err
	}

	if err = ipam.Remove(network.Subnet); err != nil {
		logrus.Errorf("failed to remove network (%v) from IPAM: %v", name, err)
		return err
	}

	if err = removeNetwork(name); err != nil {
		logrus.Errorf("failed to save network (%v): %v", name, err)
		return err
	}
	return nil
}

func Connect(networkName string, metadata *container.Metadata) error {
	network, err := readNetwork(networkName)
	if err != nil {
		logrus.Errorf("failed to get network (%v): %v", networkName, err)
		return err
	}

	if _, ok := drivers[network.Driver]; !ok {
		logrus.Errorf("do not support driver %v", network.Driver)
		return fmt.Errorf("do not support driver %v", network.Driver)
	}

	var containerIP net.IP
	containerIP, err = ipam.Allocate(network.Subnet)
	if err != nil {
		logrus.Errorf("failed to allocate ip for container (%v): %v", metadata, err)
		return err
	}
	logrus.Infof("we will allocate %v to container (%v)", containerIP, metadata.Name)

	endpoint := &types.Endpoint{
		ID:        types.GenerateEndpointID(metadata.ID, networkName),
		IP:        containerIP,
		Network:   networkName,
		GatewayIP: network.Gateway.IP,
		Subnet:    network.Subnet,
	}

	if err = drivers[network.Driver].Connect(network, endpoint); err != nil {
		logrus.Errorf("failed to connect %v for container (%v): %v", network, metadata, err)
		if releaseErr := ipam.Release(network.Subnet, containerIP); releaseErr != nil {
			logrus.Warningf("failed to release %v from %v: %v", containerIP, network, err)
		}
		return err
	}
	logrus.Infof("succeeded in connecting %v", network.Name)

	if err = setContainerNetwork(endpoint, metadata); err != nil {
		logrus.Errorf("failed to set network for container (%v): %v", metadata, err)
		return err
	}
	logrus.Infof("succeeded in setting network for container (%v)->(%v)", metadata, network)

	metadata.Endpoints = append(metadata.Endpoints, endpoint)
	if err = container.SaveMetadata(metadata); err != nil {
		logrus.Errorf("failed to update endpoints in metadata of container (%v)", metadata)
		return err
	}
	return nil
}

func Disconnect(metadata *container.Metadata) error {
	var errs []error
	for i, ep := range metadata.Endpoints {
		if err := disconnectEndpoint(ep); err != nil {
			logrus.Errorf("failed to disconnect endpoint (%v)", ep)
			errs = append(errs, fmt.Errorf("failed to disconnect endpoint (%v)", ep))
		} else {
			metadata.Endpoints[i] = nil
		}
	}
	if len(errs) > 0 {
		logrus.Errorf("failed to disconnect endpoints in container (%v): %+v", metadata, errs)
		return fmt.Errorf("failed to disconnect endpoints in container (%v): %+v", metadata, errs)
	}
	metadata.Endpoints = nil
	return nil
}

func disconnectEndpoint(endpoint *types.Endpoint) error {
	networkName := endpoint.Network
	network, err := readNetwork(networkName)
	if err != nil {
		logrus.Errorf("failed to get network (%v): %v", networkName, err)
		return err
	}

	if _, ok := drivers[network.Driver]; !ok {
		logrus.Errorf("do not support driver %v", network.Driver)
		return fmt.Errorf("do not support driver %v", network.Driver)
	}

	if err = ipam.Release(endpoint.Subnet, endpoint.IP); err != nil {
		logrus.Errorf("failed to release ip (%v) in network (%v)", endpoint.IP, endpoint.Network)
		return err
	}
	return nil
}

func setContainerNetwork(endpoint *types.Endpoint, metadata *container.Metadata) error {
	peerName := endpoint.Device.PeerName
	peerLink, err := netlink.LinkByName(peerName)
	if err != nil {
		logrus.Errorf("failed to get veth pair peer (%v): %v", peerName, err)
		return err
	}

	var file *os.File
	netnsPath := generateNetworkNamespacePath(metadata.PID)
	file, err = os.OpenFile(netnsPath, os.O_RDONLY, 0)
	if err != nil {
		logrus.Errorf("failed to open netns file (%v): %v", netnsPath, err)
		return err
	}

	nsFD := int(file.Fd())

	// 由于 golang 的 GMP 模型，为防止设置网络，执行系统调用期间，goroutine 切换到其他内核态线程，需要锁定当前的线程
	runtime.LockOSThread()

	if err = setPeerNetns(netnsPath, peerName, peerLink, nsFD); err != nil {
		logrus.Errorf("failed to setns (%v) for veth pair peer (%v): %v", netnsPath, peerName, err)
		return err
	}

	var originNetns netns.NsHandle
	originNetns, err = netns.Get()
	if err != nil {
		logrus.Errorf("failed get origin netns: %v", err)
		return err
	}
	logrus.Infof("origin netns is %v", originNetns)

	if err = util.EnterNetns(nsFD); err != nil {
		logrus.Errorf("failed to enter netns (%v) of container (%v): %v", netnsPath, metadata, err)
		return err
	}
	defer func() {
		logrus.Infof("exit netns of container (%v)", metadata)
		if err = util.EnterNetns(int(originNetns)); err != nil {
			logrus.Warningf("failed to return origin netns (%v): %v", originNetns, err)
		}
		if err = originNetns.Close(); err != nil {
			logrus.Warningf("failed to close origin netns (%v): %v", originNetns, err)
		}
		runtime.UnlockOSThread()
		if err = file.Close(); err != nil {
			logrus.Warningf("failed to close netns path (%v): %v", netnsPath, err)
		}
	}()

	if err = setContainerIP(peerName, endpoint, metadata); err != nil {
		logrus.Errorf("failed to set ip of container (%v): %v", metadata, err)
		return err
	}
	logrus.Infof("succeeded in setting ip of container (%v)", metadata)

	if err = setContainerRoute(endpoint.GatewayIP, peerLink.Attrs().Index); err != nil {
		logrus.Errorf("failed to set default route of container (%v): %v", metadata, err)
		return err
	}
	logrus.Infof("succeeded in setting default route of container (%v)", metadata)
	return nil
}

// setPeerNetns 为 veth pair peer 设置 netns，对应执行 `ip link set ${veth-pair-peer} netns ${container-netns}`
func setPeerNetns(netnsPath, peerName string, peerLink netlink.Link, nsFD int) error {
	if err := netlink.LinkSetNsFd(peerLink, nsFD); err != nil {
		logrus.Errorf("failed to set netns (%v) for veth pair peer (%v): %v", netnsPath, peerName, err)
		return err
	}
	logrus.Infof("succeeded in setting netns (%v) for veth pair peer (%v)", netnsPath, peerName)
	return nil
}

// setContainerIP 在容器的 netns 中设置 veth pair 的 ip，并启用 veth pair 和 lo 设备
func setContainerIP(peerName string, endpoint *types.Endpoint, metadata *container.Metadata) error {
	// 设置容器 IP
	if err := util.SetInterfaceIP(peerName, &net.IPNet{
		IP:   endpoint.IP,
		Mask: endpoint.Subnet.Mask,
	}); err != nil {
		logrus.Errorf("failed to set ip of veth pair peer (%v) in container (%v): %v", peerName, metadata, err)
		return err
	}
	logrus.Infof("succeeded in setting ip of veth pair peer (%v) in container (%v): %v",
		peerName, metadata.Name, endpoint.IP)

	// 设置容器的 veth pair peer 为 up
	if err := util.SetInterfaceUp(peerName); err != nil {
		logrus.Errorf("failed to set up veth pair peer (%v) in container (%v): %v", peerName, metadata, err)
		return err
	}
	logrus.Infof("succeeded in setting up veth pair peer (%v) in container (%v)", peerName, metadata.Name)

	peerIface, err := netlink.LinkByName(peerName)
	if err != nil {
		logrus.Errorf("failed to get veth pair peer (%v) in container (%v): %v", peerName, metadata, err)
		return err
	}
	endpoint.Device = peerIface.(*netlink.Veth)
	endpoint.Mac = peerIface.Attrs().HardwareAddr.String()

	// 设置容器的 lo 为 up
	if err = util.SetInterfaceUp("lo"); err != nil {
		logrus.Errorf("failed to set up lo in container (%v): %v", metadata, err)
		return err
	}
	logrus.Infof("succeeded in setting up lo in container (%v)", metadata.Name)
	return nil
}

// setContainerIP 在容器的 netns 中设置默认路由
func setContainerRoute(gatewayIP net.IP, peerLinkIndex int) error {
	_, cidr, _ := net.ParseCIDR("0.0.0.0/0")
	defaultRoute := &netlink.Route{
		Dst:       cidr,
		Gw:        gatewayIP,
		LinkIndex: peerLinkIndex,
	}
	logrus.Infof("try to add default route: %v", defaultRoute)
	if err := netlink.RouteAdd(defaultRoute); err != nil {
		logrus.Errorf("failed to add default route: %v", err)
		return err
	}
	logrus.Infof("succeeded in adding default route: %v", defaultRoute)
	return nil
}

func readNetwork(networkName string) (*Network, error) {
	networkPath := generateNetworkPath(networkName)
	body, err := ioutil.ReadFile(networkPath)
	if err != nil {
		logrus.Errorf("failed to read %v: %v", networkPath, err)
		return nil, err
	}
	network := &Network{}
	if err = json.Unmarshal(body, network); err != nil {
		logrus.Errorf("failed to unmarshal %v: %v", string(body), err)
		return nil, err
	}
	return network, nil
}

func saveNetwork(network *Network) error {
	body, err := json.Marshal(network)
	if err != nil {
		logrus.Errorf("failed to marshal network (%+v): %v", network, err)
		return err
	}

	networkName := network.Name
	if err = util.EnsureDirectory(DefaultNetworkDir); err != nil {
		logrus.Errorf("failed to ensure network directory %v: %v", DefaultNetworkDir, err)
	}

	var file *os.File
	configPath := generateNetworkPath(networkName)
	file, err = os.Create(configPath)
	if err != nil {
		logrus.Errorf("failed to create file of network (%v): %v", configPath, err)
		return err
	}

	if _, err = file.Write(body); err != nil {
		logrus.Errorf("failed to write (%v) to network file (%v): %v", string(body), configPath, err)
		return err
	}
	return nil
}

func removeNetwork(networkName string) error {
	configPath := generateNetworkPath(networkName)
	if err := os.RemoveAll(configPath); err != nil {
		logrus.Errorf("failed to remove network config (%v): %v", configPath, err)
		return err
	}
	return nil
}

func generateNetworkPath(networkName string) string {
	return path.Join(DefaultNetworkDir, networkName)
}

// generateNetworkNamespacePath 生成某一个 PID 的 netns 的文件路径，可以通过 `readlink /proc/${pid}/ns/net` 查看
func generateNetworkNamespacePath(pid int) string {
	return fmt.Sprintf("/proc/%v/ns/net", pid)
}
