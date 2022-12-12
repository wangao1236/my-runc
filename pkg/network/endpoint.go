package network

import (
	"net"

	"github.com/vishvananda/netlink"
)

type Endpoint struct {
	ID     string       `json:"id"`
	Device netlink.Veth `json:"device"`
	IP     net.IPNet    `json:"ipAddress"`
	Mac    string       `json:"mac"`
}
