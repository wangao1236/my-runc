package types

import (
	"encoding/json"
	"net"

	"github.com/vishvananda/netlink"
)

type Endpoint struct {
	ID        string        `json:"id"`
	Device    *netlink.Veth `json:"device"`
	IP        net.IP        `json:"ipAddress"`
	Mac       string        `json:"mac"`
	Network   string        `json:"network"`
	GatewayIP net.IP        `json:"gatewayIP"`
	Subnet    *net.IPNet    `json:"subnet"`
}

func (e *Endpoint) String() string {
	body, _ := json.Marshal(e)
	return string(body)
}

func (e *Endpoint) GetIPNet() *net.IPNet {
	if e.Subnet == nil || e.Subnet.Mask == nil {
		return nil
	}
	return &net.IPNet{
		IP:   e.IP,
		Mask: e.Subnet.Mask,
	}
}

func GenerateEndpointID(metadataID, networkName string) string {
	return metadataID + "-" + networkName
}
