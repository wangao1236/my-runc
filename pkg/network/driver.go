package network

import (
	"net"
	"os/exec"

	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"github.com/wangao1236/my-docker/pkg/types"
	"github.com/wangao1236/my-docker/pkg/util"
)

const (
	DriverBridge = "bridge"
)

var (
	_       Driver = &bridgeDriver{}
	drivers        = make(Drivers)
	ipam    *IPAM
)

func init() {
	bd := &bridgeDriver{}
	drivers[bd.Name()] = bd
	ipam = NewIPAM(true)
}

type Drivers map[string]Driver

type Driver interface {
	Name() string
	CreateNetwork(networkName, subnet string) (*Network, error)
	DeleteNetwork(network *Network) error
	Connect(network *Network, endpoint *types.Endpoint) error
	Disconnect(network *Network, endpoint *types.Endpoint) error
}

type bridgeDriver struct {
}

func (bd *bridgeDriver) Name() string {
	return DriverBridge
}

func (bd *bridgeDriver) CreateNetwork(networkName, subnet string) (*Network, error) {
	_, ipNet, err := net.ParseCIDR(subnet)
	if err != nil {
		logrus.Errorf("invalid subnet %v: %v", subnet, err)
		return nil, err
	}
	var gatewayIP net.IP
	gatewayIP, err = ipam.Allocate(ipNet)
	if err != nil {
		logrus.Errorf("failed to allocate gateway ip for network (%v): %v", networkName, err)
		return nil, err
	}
	logrus.Infof("allocated %v to gateway of network %v", gatewayIP, networkName)

	network := &Network{
		Name:   networkName,
		Driver: bd.Name(),
		Subnet: ipNet,
		Gateway: &net.IPNet{
			IP:   gatewayIP,
			Mask: ipNet.Mask,
		},
	}
	if err = bd.initBridge(network); err != nil {
		logrus.Errorf("failed to init bridge (%v): %v", network.Name, err)
		return nil, err
	}
	logrus.Infof("network %v has been initialized", network)

	return network, nil
}

func (bd *bridgeDriver) DeleteNetwork(network *Network) error {
	if err := bd.clearInterfaceIPTables(network.Name, network.Subnet); err != nil {
		logrus.Errorf("failed to clear iptables for bridge (%v): %v", network.Name, err)
		return err
	}
	logrus.Infof("succeed in clearing iptables for bridge (%v)", network.Name)

	if err := util.DeleteInterface(network.Name); err != nil {
		logrus.Errorf("failed to delete bridge (%v): %v", network.Name, err)
		return err
	}
	logrus.Infof("succeed in deleting bridge (%v)", network.Name)

	if err := ipam.Release(network.Subnet, network.Gateway.IP); err != nil {
		logrus.Errorf("failed to release gateway (%v) of bridge (%v): %v", network.Gateway, network.Name, err)
		return err
	}
	logrus.Infof("succeed in releasing gateway (%v) of bridge (%v)", network.Gateway, network.Name)
	return nil
}

func (bd *bridgeDriver) Connect(network *Network, endpoint *types.Endpoint) error {
	br, err := netlink.LinkByName(network.Name)
	if err != nil {
		logrus.Errorf("failed to get interface (%v): %v", network.Name, err)
		return err
	}

	// 创建 veth pair 的配置
	la := netlink.NewLinkAttrs()
	la.Name = endpoint.ID[:5]
	// 相当于执行 `ip link set dev ${peer-name} master ${bridge-name}`，将 veth pair 的 peer 端插在 bridge 上
	la.MasterIndex = br.Attrs().Index
	// 初始化 veth pair 设备的基本配置
	endpoint.Device = &netlink.Veth{
		LinkAttrs: la,
		PeerName:  generateVethPairPeerName(la.Name),
	}

	if err = netlink.LinkAdd(endpoint.Device); err != nil {
		logrus.Errorf("failed to add veth pair (%v/%v): %v", endpoint.Device.Name, endpoint.Device.PeerName, err)
		return err
	}
	logrus.Infof("succeed in adding veth pair (%v/%v)", endpoint.Device.Name, endpoint.Device.PeerName)

	if err = netlink.LinkSetUp(endpoint.Device); err != nil {
		logrus.Errorf("failed to set up veth pair (%v/%v): %v", endpoint.Device.Name, endpoint.Device.PeerName, err)
		return err
	}
	logrus.Infof("succeed in setting up veth pair (%v/%v)", endpoint.Device.Name, endpoint.Device.PeerName)
	return nil
}

func (bd *bridgeDriver) Disconnect(network *Network, endpoint *types.Endpoint) error {
	panic("implement me")
}

func (bd *bridgeDriver) initBridge(network *Network) error {
	if err := util.EnsureBridgeInterface(network.Name); err != nil {
		logrus.Errorf("failed to ensure bridge interface %v: %v", network.Name, err)
		return err
	}

	if err := util.SetInterfaceIP(network.Name, network.Gateway); err != nil {
		logrus.Errorf("failed to set ip (%v) of bridge interface %v: %v", network.Gateway, network.Name, err)
		return err
	}

	if err := util.SetInterfaceUp(network.Name); err != nil {
		logrus.Errorf("failed to set up (%v) of bridge interface %v: %v", network.Gateway, network.Name, err)
		return err
	}

	if err := bd.setInterfaceIPTables(network.Name, network.Subnet); err != nil {
		logrus.Errorf("failed to set iptables for bridge (%v): %v", network.Name, err)
		return err
	}

	return nil
}

func (bd *bridgeDriver) setInterfaceIPTables(name string, subnet *net.IPNet) error {
	// 设置 iptables 对容器网络出流量做 SNAT，发生在 nat 表的 POSTROUTING 链上
	// 参考：https://chanjarster.github.io/post/network/ip-forwarding-masq-nat/
	args := []string{
		"-t", "nat", "-A", "POSTROUTING",
		"-s", subnet.String(), "!", "-o", name, "-j", "MASQUERADE",
	}
	cmd := exec.Command("iptables", args...)
	if output, err := cmd.Output(); err != nil {
		logrus.Errorf("failed to set iptables (%+v), output (%v): %v", args, string(output), err)
		return err
	}
	return nil
}

func (bd *bridgeDriver) clearInterfaceIPTables(name string, subnet *net.IPNet) error {
	args := []string{
		"-t", "nat", "-D", "POSTROUTING",
		"-s", subnet.String(), "!", "-o", name, "-j", "MASQUERADE",
	}
	cmd := exec.Command("iptables", args...)
	if output, err := cmd.Output(); err != nil {
		logrus.Errorf("failed to set iptables (%+v), output (%v): %v", args, string(output), err)
		return err
	}
	return nil
}

func generateVethPairPeerName(vethPairName string) string {
	return "vp-" + vethPairName
}
