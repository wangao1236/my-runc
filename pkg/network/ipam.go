package network

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
)

const (
	DefaultSubnetPath = DefaultNetworkRootDir + "/subnets.json"
)

// IPAM 管理地址分配
type IPAM struct {
	save bool
	// Subnets 保存网段到分配结果位图的映射关系
	Subnets map[string]string
}

func NewIPAM(save bool) *IPAM {
	return &IPAM{
		save:    save,
		Subnets: make(map[string]string),
	}
}

func (ipam *IPAM) Allocate(subnet *net.IPNet) (ip net.IP, err error) {
	if ipam.save {
		var subnets map[string]string
		if subnets, err = readSubnets(); err != nil {
			logrus.Errorf("failed to read subnets: %v", err)
			return
		}
		logrus.Info("succeeded in reading subnets")
		ipam.Subnets = subnets
		defer func() {
			if err == nil {
				if saveErr := saveSubnets(ipam.Subnets); saveErr != nil {
					logrus.Warningf("failed to save subnets: %v", saveErr)
				} else {
					logrus.Info("succeeded in saving subnets")
				}
			} else {
				logrus.Errorf("can not save subnets for error: %v", err)
			}
		}()
	}

	key := subnet.String()
	if _, ok := ipam.Subnets[key]; !ok {
		one, size := subnet.Mask.Size()
		ipam.Subnets[key] = strings.Repeat("0", 1<<(size-one))
	}

	ipNet := []byte(ipam.Subnets[key])
	allocatedIP := make([]byte, 4)
	copy(allocatedIP, subnet.IP)
	for idx := range ipNet {
		if ipNet[idx] == '1' {
			continue
		}
		numberToIP(allocatedIP, uint32(idx+1))
		ipNet[idx] = '1'
		ipam.Subnets[key] = string(ipNet)
		return allocatedIP, nil
	}
	return nil, fmt.Errorf("no allocatable ip in %v", key)
}

func (ipam *IPAM) Release(subnet *net.IPNet, ip net.IP) (err error) {
	if ipam.save {
		var subnets map[string]string
		if subnets, err = readSubnets(); err != nil {
			logrus.Errorf("failed to read subnets: %v", err)
			return
		}
		logrus.Info("succeeded in reading subnets")
		ipam.Subnets = subnets
		defer func() {
			if err == nil {
				if saveErr := saveSubnets(ipam.Subnets); saveErr != nil {
					logrus.Warningf("failed to save subnets: %v", saveErr)
				} else {
					logrus.Info("succeeded in saving subnets")
				}
			} else {
				logrus.Errorf("can not save subnets for error: %v", err)
			}
		}()
	}

	key := subnet.String()
	if _, ok := ipam.Subnets[key]; !ok {
		return fmt.Errorf("%v has not been allocated", key)
	}

	ipNet := []byte(ipam.Subnets[key])
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
	ipam.Subnets[key] = string(ipNet)
	return
}

func readSubnets() (map[string]string, error) {
	subnets := make(map[string]string)
	if _, err := os.Stat(DefaultSubnetPath); err != nil {
		if os.IsNotExist(err) {
			return subnets, nil
		}
		logrus.Errorf("failed to get file of subnets (%v): %v", DefaultSubnetPath, err)
		return nil, err
	}
	body, err := ioutil.ReadFile(DefaultSubnetPath)
	if err != nil {
		logrus.Errorf("failed to read %v: %v", DefaultSubnetPath, err)
		return nil, err
	}
	if err = json.Unmarshal(body, &subnets); err != nil {
		logrus.Errorf("failed to unmarshal %v: %v", string(body), err)
		return nil, err
	}
	return subnets, nil
}

func saveSubnets(subnets map[string]string) error {
	body, err := json.Marshal(subnets)
	if err != nil {
		logrus.Errorf("failed to marshal subnets (%+v): %v", subnets, err)
		return err
	}

	var file *os.File
	file, err = os.Create(DefaultSubnetPath)
	if err != nil {
		logrus.Errorf("failed to create file of subnets (%v): %v", DefaultSubnetPath, err)
		return err
	}

	if _, err = file.Write(body); err != nil {
		logrus.Errorf("failed to write (%v) to subnets file (%v): %v", string(body), DefaultSubnetPath, err)
		return err
	}
	return nil
}

func ipToNumber(ip net.IP) (num uint32, err error) {
	ipv4 := ip.To4()
	if ipv4 == nil {
		return 0, fmt.Errorf("%v is not a ipv4", ip)
	}
	num = 0
	for i := 0; i < 4; i++ {
		num |= uint32(ipv4[i]) << ((3 - i) * 8)
	}
	return
}

func numberToIP(baseIP net.IP, num uint32) {
	for i := 0; i < 4; i++ {
		baseIP[i] += uint8(num >> ((3 - i) * 8))
	}
}
