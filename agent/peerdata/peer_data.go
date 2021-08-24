package peerdata

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/SyntropyNet/syntropy-agent-go/controller"
	"github.com/SyntropyNet/syntropy-agent-go/multiping"
	"github.com/SyntropyNet/syntropy-agent-go/wireguard"
)

const cmd = "IFACE_PEERS_BW_DATA"
const pkgName = "Peer_Data. "

const (
	periodInit = time.Second
	periodRun  = time.Minute
)

type peerDataEntry struct {
	IP      string  `json:"internal_ip"`
	Latency int     `json:"latency_ms"`
	Loss    float32 `json:"packet_loss"`
	Status  string  `json:"status"`
	Reason  string  `json:"status_reason,omitempty"`
}

type ifaceBwEntry struct {
	IfName    string             `json:"iface"`
	PublicKey string             `json:"iface_public_key"`
	Peers     []peerDataEntry    `json:"peers"`
	channel   chan *ifaceBwEntry `json:"-"`
}

type peerBwData struct {
	controller.MessageHeader
	Data []ifaceBwEntry `json:"data"`
}

type wgPeerWatcher struct {
	wg      *wireguard.Wireguard
	writer  io.Writer
	ticker  *time.Ticker
	timeout time.Duration
	stop    chan bool
}

func New(writer io.Writer, wgctl *wireguard.Wireguard) controller.Service {
	return &wgPeerWatcher{
		wg:      wgctl,
		writer:  writer,
		timeout: periodInit,
		stop:    make(chan bool),
	}
}

func (ie *ifaceBwEntry) ProcessPingResults(pr []multiping.PingResult) {
	for _, pingres := range pr {
		entry := peerDataEntry{
			IP:      pingres.IP,
			Latency: pingres.Latency,
			Loss:    pingres.Loss,
		}

		switch {
		case entry.Loss >= 1:
			entry.Status = "OFFLINE"
			entry.Reason = "Packet loss 100%"
		case entry.Loss >= 0.01 && entry.Loss < 1:
			entry.Status = "WARNING"
			entry.Reason = "Packet loss higher than 1%"
		case entry.Latency > 500:
			entry.Status = "WARNING"
			entry.Reason = "Latency higher than 500ms"
		default:
			entry.Status = "CONNECTED"
		}

		ie.Peers = append(ie.Peers, entry)
	}
	ie.channel <- ie
}

func (obj *wgPeerWatcher) execute() error {
	wg := obj.wg
	resp := peerBwData{}
	resp.ID = "-"
	resp.MsgType = cmd

	wgdevs, err := wg.Devices()
	if err != nil {
		return err
	}

	count := len(wgdevs)

	// If no interfaces are created yet - I send nothing to controller and wait a short time
	// When interfaces are created - switch to less frequently check
	if count == 0 {
		if obj.timeout != periodInit {
			obj.timeout = periodInit
			obj.ticker.Reset(obj.timeout)
		}
		return nil
	} else if obj.timeout != periodRun {
		obj.timeout = periodRun
		obj.ticker.Reset(obj.timeout)
	}

	// The pinger runs in background, so I will create a channel with number of interfaces
	// And pinger callback will put interface entry (with peers ping info) into channel
	c := make(chan *ifaceBwEntry, count)

	for _, wgdev := range wgdevs {
		ifaceData := ifaceBwEntry{
			IfName:    wgdev.Name,
			PublicKey: wgdev.PublicKey.String(),
			Peers:     []peerDataEntry{},
			channel:   c,
		}
		ping := multiping.New(&ifaceData)

		for _, p := range wgdev.Peers {
			if len(p.AllowedIPs) == 0 {
				continue
			}
			ip := p.AllowedIPs[0]
			ping.AddHost(ip.IP.String())
		}

		ping.Ping()
	}

	for count > 0 {
		entry := <-c
		if len(entry.Peers) > 0 {
			resp.Data = append(resp.Data, *entry)
		}
		count--
	}
	close(c)

	if len(resp.Data) > 0 {
		resp.Now()
		raw, err := json.Marshal(resp)
		if err != nil {
			return err
		}
		obj.writer.Write(raw)
	}

	return nil
}

func (obj *wgPeerWatcher) Name() string {
	return cmd
}

func (obj *wgPeerWatcher) Start() error {
	// I'm not doing concurency prevention here,
	// because this Start() should not be called concurently and this is only a sanity check
	if obj.ticker != nil {
		return fmt.Errorf("%s is already running", pkgName)
	}

	obj.ticker = time.NewTicker(periodInit)
	go func() {
		for {
			select {
			case <-obj.stop:
				return
			case <-obj.ticker.C:
				obj.execute()

			}
		}
	}()
	return nil
}

func (obj *wgPeerWatcher) Stop() error {
	// Cannot stop not running instance
	if obj.ticker == nil {
		return fmt.Errorf("%s is not running", pkgName)

	}

	obj.ticker.Stop()
	obj.stop <- true

	return nil
}
