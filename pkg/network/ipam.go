package network

import (
	"fmt"
	"net"
	"strings"
)

// IPAM 管理地址分配
type IPAM struct {
	// Subnets 保存网段到分配结果位图的映射关系
	Subnets map[string][]byte
}

func NewIPAM() *IPAM {
	return &IPAM{
		Subnets: make(map[string][]byte),
	}
}

func (ipam *IPAM) Allocate(subnet *net.IPNet) (ip net.IP, err error) {
	key := subnet.String()
	if _, ok := ipam.Subnets[key]; !ok {
		one, size := subnet.Mask.Size()
		ipam.Subnets[key] = []byte(strings.Repeat("0", 1<<(size-one)))
	}

	ipNet := ipam.Subnets[key]
	baseIP := subnet.IP
	for idx := range ipNet {
		if ipNet[idx] == '1' {
			continue
		}
		numberToIP(baseIP, uint32(idx+1))
		ipNet[idx] = '1'
		return baseIP, nil
	}
	return nil, fmt.Errorf("no allocatable ip in %v", key)
}

func (ipam *IPAM) Release(subnet *net.IPNet, ip net.IP) (err error) {
	key := subnet.String()
	if _, ok := ipam.Subnets[key]; !ok {
		return fmt.Errorf("%v has not been allocated", key)
	}

	ipNet := ipam.Subnets[key]
	baseIP := subnet.IP
	var base, target uint32
	base, err = ipToNumber(baseIP)
	if err != nil {
		return
	}
	target, err = ipToNumber(ip)
	if err != nil {
		return
	}
	ipNet[target-base-1] = '0'
	return
}

func ipToNumber(ip net.IP) (num uint32, err error) {
	if ip.To4() == nil {
		return 0, fmt.Errorf("%v is not a ipv4", ip)
	}
	num = 0
	for i := 0; i < 4; i++ {
		num |= uint32(ip[i] << ((3 - i) * 8))
	}
	return
}

func numberToIP(baseIP net.IP, num uint32) {
	for i := 0; i < 4; i++ {
		baseIP[i] += uint8(num >> ((3 - i) * 8))
	}
}
