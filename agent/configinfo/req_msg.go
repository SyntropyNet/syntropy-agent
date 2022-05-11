package configinfo

import (
	"net/netip"
	"strings"

	"github.com/SyntropyNet/syntropy-agent/agent/common"
	"github.com/SyntropyNet/syntropy-agent/agent/swireguard"
	"github.com/SyntropyNet/syntropy-agent/internal/env"
)

type configInfoNetworkEntry struct {
	IP        string `json:"internal_ip"`
	PublicKey string `json:"public_key,omitempty"`
	Port      int    `json:"listen_port,omitempty"`
}

func (e *configInfoNetworkEntry) asInterfaceInfo(ifaceName string) (*swireguard.InterfaceInfo, error) {
	var ifname string
	if strings.HasPrefix(ifaceName, env.InterfaceNamePrefix) {
		ifname = ifaceName
	} else {
		ifname = env.InterfaceNamePrefix + ifaceName
	}

	addr, err := netip.ParseAddr(e.IP)
	if err != nil {
		return nil, err
	}

	return &swireguard.InterfaceInfo{
		IfName:    ifname,
		IP:        addr,
		PublicKey: e.PublicKey,
		Port:      e.Port,
	}, nil
}

func (e *configInfoVpnEntry) asPeerInfo() *swireguard.PeerInfo {
	var ifname string
	if strings.HasPrefix(e.Args.IfName, env.InterfaceNamePrefix) {
		ifname = e.Args.IfName
	} else {
		ifname = env.InterfaceNamePrefix + e.Args.IfName
	}
	return &swireguard.PeerInfo{
		IfName:       ifname,
		IP:           e.Args.EndpointIPv4,
		PublicKey:    e.Args.PublicKey,
		ConnectionID: e.Metadata.ConnectionID,
		GroupID:      e.Metadata.GroupID,
		AgentID:      e.Metadata.AgentID,
		Port:         e.Args.EndpointPort,
		Gateway:      e.Args.GatewayIPv4,
		AllowedIPs:   e.Args.AllowedIPs,
	}
}

func (e *configInfoVpnEntry) asInterfaceInfo() (*swireguard.InterfaceInfo, error) {
	var ifname string
	if strings.HasPrefix(e.Args.IfName, env.InterfaceNamePrefix) {
		ifname = e.Args.IfName
	} else {
		ifname = env.InterfaceNamePrefix + e.Args.IfName
	}

	addr, err := netip.ParseAddr(e.Args.InternalIP)
	if err != nil {
		return nil, err
	}
	return &swireguard.InterfaceInfo{
		IfName:    ifname,
		IP:        addr,
		PublicKey: e.Args.PublicKey,
		Port:      e.Args.ListenPort,
	}, nil
}

func (e *configInfoVpnEntry) asNetworkPath() *common.SdnNetworkPath {
	var gw string
	if len(e.Args.AllowedIPs) > 0 {
		// Controller sends first IP as connected peers internal IP address
		// Use this IP as internal routing gateway
		parts := strings.Split(e.Args.AllowedIPs[0], "/")
		gw = parts[0]
	}
	return &common.SdnNetworkPath{
		Ifname:       e.Args.IfName,
		PublicKey:    e.Args.PublicKey,
		Gateway:      gw,
		ConnectionID: e.Metadata.ConnectionID,
		GroupID:      e.Metadata.GroupID,
	}
}

type configInfoSubnetworksEntry struct {
	Name   string `json:"name"`
	Subnet string `json:"subnet"`
	Type   string `json:"type"`
}

type configInfoVpnEntry struct {
	Function string `json:"fn"`

	Args struct {
		// Common fields
		IfName    string `json:"ifname"`
		PublicKey string `json:"public_key,omitempty"`
		// create_interface
		InternalIP string `json:"internal_ip,omitempty"`
		ListenPort int    `json:"listen_port,omitempty"`
		// add_peer
		AllowedIPs   []string `json:"allowed_ips,omitempty"`
		EndpointIPv4 string   `json:"endpoint_ipv4,omitempty"`
		EndpointPort int      `json:"endpoint_port,omitempty"`
		GatewayIPv4  string   `json:"gw_ipv4,omitempty"`
	} `json:"args,omitempty"`

	Metadata struct {
		// create_interface
		NetworkID int `json:"network_id,omitempty"`
		// add_peer
		DeviceID         string `json:"device_id,omitempty"`
		DeviceName       string `json:"device_name,omitempty"`
		DevicePublicIPv4 string `json:"device_public_ipv4,omitempty"`
		ConnectionID     int    `json:"connection_id,omitempty"`
		GroupID          int    `json:"connection_group_id,omitempty"`
		AgentID          int    `json:"agent_id,omitempty"`
	} `json:"metadata,omitempty"`
}

type configInfoMsg struct {
	common.MessageHeader
	Data struct {
		AgentID int `json:"agent_id"`
		Network struct {
			Public *configInfoNetworkEntry `json:"PUBLIC,omitempty"`
			Sdn1   *configInfoNetworkEntry `json:"SDN1,omitempty"`
			Sdn2   *configInfoNetworkEntry `json:"SDN2,omitempty"`
			Sdn3   *configInfoNetworkEntry `json:"SDN3,omitempty"`
		}
		VPN         []configInfoVpnEntry         `json:"vpn,omitempty"`
		Subnetworks []configInfoSubnetworksEntry `json:"subnetworks,omitempty"`
	} `json:"data"`
}
