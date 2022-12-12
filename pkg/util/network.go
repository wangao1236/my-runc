package util

import (
	"net"
	"strings"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
)

// EnsureBridgeInterface 检查 bridge 接口是否存在，不存在则创建。
func EnsureBridgeInterface(name string) error {
	_, err := net.InterfaceByName(name)
	if err == nil || !strings.Contains(err.Error(), "no such network interface") {
		return err
	}

	la := netlink.NewLinkAttrs()
	la.Name = name
	bridge := &netlink.Bridge{
		LinkAttrs: la,
	}
	if err = netlink.LinkAdd(bridge); err != nil {
		logrus.Errorf("faild to execute `ip link add %v`: %v", name, err)
		return err
	}
	return nil
}

// DeleteInterface 删除特定的 network interface。
func DeleteInterface(name string) error {
	iface, err := netlink.LinkByName(name)
	if err != nil {
		logrus.Errorf("faild to get iterface (%v): %v", name, err)
		return err
	}

	if err = netlink.LinkDel(iface); err != nil {
		logrus.Errorf("failed to execute `ip link del %v`: %v", name, err)
		return err
	}
	return nil
}

// SetInterfaceIP 检查 network interface 是否存在，如果不存在则报错；如果存在，则检查 interface 是否有期望的网络地址。
// 如果期望地址不存在，就给 interface 添加该地址。
func SetInterfaceIP(name string, ipAddr *net.IPNet) error {
	var iface netlink.Link
	var err error
	retries := 2
	for i := 0; i < retries; i++ {
		iface, err = netlink.LinkByName(name)
		if err == nil {
			break
		}
		logrus.Warningf("failed to get link %v, we will try again: %v", name, err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		logrus.Errorf("failed to get link %v: %v", name, err)
		return err
	}

	targetAddr := &netlink.Addr{
		IPNet: ipAddr,
		Peer:  ipAddr,
	}

	var addrList []netlink.Addr
	addrList, err = netlink.AddrList(iface, syscall.AF_INET)
	for i := range addrList {
		if addrList[i].Equal(*targetAddr) {
			logrus.Infof("iface %v has already been set ip address %v", name, ipAddr)
			return err
		}
	}

	if err = netlink.AddrAdd(iface, targetAddr); err != nil {
		logrus.Errorf("failed to execute `ip addr add %v dev %v`: %v", ipAddr, name, err)
		return err
	}
	return nil
}

// SetInterfaceUp 检查 network interface 是否存在，如果不存在则报错；如果存在，则设置 iface ip。
func SetInterfaceUp(name string) error {
	iface, err := netlink.LinkByName(name)
	if err != nil {
		logrus.Errorf("failed to get link %v: %v", name, err)
		return err
	}
	if err = netlink.LinkSetUp(iface); err != nil {
		logrus.Errorf("failed to execute `ip link set %v uo`: %v", name, err)
		return err
	}
	return nil
}
