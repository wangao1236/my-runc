package network

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIPAMAllocate(t *testing.T) {
	testIPAM := NewIPAM(false)
	// 每次释放和分配ip时，都需要重新调用下面的函数进行IPNet的获取，因为函数调用后，IPNet的值会发生变化
	_, ipNet, _ := net.ParseCIDR("192.168.0.0/24")
	// 第一次分配
	ip1, err := testIPAM.Allocate(ipNet)
	assert.Equal(t, nil, err)
	assert.Equal(t, "192.168.0.1", ip1.String())
	t.Logf("%v allocated", ip1)

	// 第二个ip分配
	var ip2 net.IP
	_, ipNet, _ = net.ParseCIDR("192.168.0.0/24")
	ip2, err = testIPAM.Allocate(ipNet)
	assert.Equal(t, nil, err)
	assert.Equal(t, "192.168.0.2", ip2.String())
	t.Logf("%v allocated", ip2)

	// 释放调第一个IP
	_, ipNet, _ = net.ParseCIDR("192.168.0.0/24")
	assert.Equal(t, nil, testIPAM.Release(ipNet, ip1))

	// 能分配得第一个IP
	var ip3 net.IP
	_, ipNet, _ = net.ParseCIDR("192.168.0.0/24")
	ip3, err = testIPAM.Allocate(ipNet)
	assert.Equal(t, nil, err)
	assert.Equal(t, "192.168.0.1", ip3.String())
	t.Logf("%v allocated", ip3)

	// 分配第三个IP
	var ip4 net.IP
	_, ipNet, _ = net.ParseCIDR("192.168.0.0/24")
	ip4, err = testIPAM.Allocate(ipNet)
	assert.Equal(t, nil, err)
	assert.Equal(t, "192.168.0.3", ip4.String())
	t.Logf("%v allocated", ip4)
}
