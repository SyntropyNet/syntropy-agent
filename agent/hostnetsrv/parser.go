package hostnetsrv

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/SyntropyNet/syntropy-agent-go/internal/logger"
	"github.com/google/go-cmp/cmp"
)

// /proc/net/tcp (and friends udp, tcp6, udp6) line entries indexes
const (
	lineNum = iota
	localAddress
	remoteAddress
	state
	queue
	transmitTime
	retransmit
	uid
	timeout
	inode
)

const (
	stateListen = "0A"
)

func convertIpFromHex(hex string) string {
	if len(hex) > 8 {
		logger.Warning().Println(pkgName, "IPv6 is not yet supported")
		// fallback to some invalid IPv6 address
		// So it will result to error everywhere, when IPv6 is started testing
		return "ffff::eeee::0001"
	} else {
		var num [4]int64
		var err error
		printOnce := false
		for i := 0; i < 4; i++ {
			num[i], err = strconv.ParseInt(hex[i*2:i*2+2], 16, 8)
			if err != nil && !printOnce {
				// Parse errors here should never happen.
				logger.Error().Println(pkgName, err)
				printOnce = true
			}
		}
		// Hex is in network order. Reverse it.
		return fmt.Sprintf("%d.%d.%d.%d", num[3], num[2], num[1], num[0])
	}
}

// Parse line by line /proc/net/tcp|udp file
// and store only listen state services
func (obj *hostNetServices) parseProcNetFile(name string, services *[]hostServiceEntry) {
	// true - TCP ports, false = UDP ports
	portTcp := strings.HasPrefix(path.Base(name), "tcp")

	f, err := os.OpenFile(name, os.O_RDONLY, os.ModePerm)
	if err != nil {
		logger.Error().Println(pkgName, name, "open file error: ", err)
		return
	}
	defer f.Close()

	rd := bufio.NewReader(f)
	for {
		entry := hostServiceEntry{
			Subnets: []string{},
			Ports: ports{
				TCP: []int32{},
				UDP: []int32{},
			},
		}
		line, err := rd.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}

			logger.Error().Println(pkgName, name, "read file error: ", err)
			return
		}
		arr := strings.Fields(line)
		if arr[state] != stateListen {
			continue
		}

		ipPort := strings.Split(arr[localAddress], ":")

		// IP is in hex. Convert to printable string IP format
		entry.Subnets = append(entry.Subnets, convertIpFromHex(ipPort[0]))

		// Port is in hex. Convert to dec
		port, err := strconv.ParseInt(ipPort[1], 16, 17)
		if err != nil {
			// Parse errors here should never happen.
			logger.Error().Println(pkgName, err)
			port = 0
		}
		if portTcp {
			entry.Ports.TCP = append(entry.Ports.TCP, int32(port))
		} else {
			entry.Ports.UDP = append(entry.Ports.UDP, int32(port))
		}

		// TODO: add interface name (is it really an interface name, not a service name?)
		entry.IfName = "UnknownIfname"

		*services = append(*services, entry)
	}

}

func (obj *hostNetServices) execute() {
	services := []hostServiceEntry{}

	obj.parseProcNetFile("/proc/net/tcp", &services)
	obj.parseProcNetFile("/proc/net/udp", &services)
	// Not yet
	//	obj.parseProcNetFile("/proc/net/tcp6", &services)
	//	obj.parseProcNetFile("/proc/net/udp6", &services)

	if !cmp.Equal(services, obj.msg.Data) {
		obj.msg.Data = services
		obj.msg.Now()
		raw, err := json.Marshal(obj.msg)
		if err != nil {
			logger.Error().Println(pkgName, "json marshal", err)
			return
		}
		logger.Debug().Println(pkgName, "Sending: ", string(raw))
		obj.writer.Write(raw)
	}
}
