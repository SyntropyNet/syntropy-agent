package router

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/SyntropyNet/syntropy-agent-go/internal/env"
	"github.com/SyntropyNet/syntropy-agent-go/internal/logger"
	"github.com/SyntropyNet/syntropy-agent-go/pkg/common"
	"github.com/SyntropyNet/syntropy-agent-go/pkg/netcfg"
)

/**
 * Hugh.... I'm a little in doubt here...
 * I do not like GOs standard `net` package IP structs and interfaces
 * vishvananda's netlink package is either too low level or tries reusing net packages interfaces
 * An option could be to use tailscale's inet.af/netaddr, but this needs more investigation
 * what benefits tradeoffs we will get
 * So for now lets stick to plain strings (TODO)
 **/

func (r *Router) RouteAdd(netpath *common.SdnNetworkPath, dest ...string) []common.RouteResult {
	res := []common.RouteResult{}

	// add a single route helper function (makes results formating easier)
	singleRouteAdd := func(ip string) (entry common.RouteResult) {
		entry.IP = ip

		// Keep a list of active SDN routes
		if r.routes[ip] == nil {
			r.routes[ip] = &routeList{
				// when adding new destination - always start with the first route active
				// Yes, I know this is zero by default, but I wanted it to be explicitely clear
				active: 0,
			}
		}
		r.routes[ip].Add(&routeEntry{
			ifname:       netpath.Ifname,
			gateway:      netpath.Gateway,
			connectionID: netpath.ConnectionID,
			groupID:      netpath.GroupID,
		})

		if r.routes[ip].Count() > 1 {
			logger.Debug().Println(pkgName, "skip existing SDN route to", ip)
			return
		}

		if netcfg.RouteExists(ip) {
			entry.Error = fmt.Errorf("route to %s already exists", ip)
			logger.Warning().Println(pkgName, entry.Error)
			return
		}

		logger.Info().Println(pkgName, "Route add ", ip, " via ", netpath.Gateway)
		entry.Error = netcfg.RouteAdd(netpath.Ifname, netpath.Gateway, ip)
		if entry.Error != nil {
			logger.Error().Println(pkgName, "route add error", entry.Error)
		}

		return
	}

	r.Lock()
	defer r.Unlock()

	for _, ip := range dest {
		res = append(res, singleRouteAdd(ip))
	}

	return res
}

func (r *Router) RouteDel(netpath *common.SdnNetworkPath, ips ...string) []common.RouteResult {
	res := []common.RouteResult{}
	r.Lock()
	defer r.Unlock()

	for _, ip := range ips {
		if r.routes[ip] != nil {
			delete(r.routes, ip)
			entry := common.RouteResult{
				IP: ip,
			}
			entry.Error = netcfg.RouteDel(netpath.Ifname, ip)
			if entry.Error != nil {
				logger.Error().Println(pkgName, ip, "route delete error", entry.Error)
			}
			res = append(res, entry)

		}
	}

	return res
}

func (r *Router) Reroute(newgw string) error {
	errIPs := []string{}
	resp := newRespMsg()

	r.Lock()

	for dest, routes := range r.routes {
		if routes.Count() <= 1 {
			// cannot do smart routing on only one route list
			continue
		}

		for idx, route := range routes.list {
			if newgw == route.gateway {
				if idx == routes.active {
					break
				}
				logger.Info().Printf("%s change route to %s via %s [id:%d]\n",
					pkgName, dest, newgw, route.groupID)
				logger.Info().Println(pkgName, idx, routes.active)
				routes.Print()
				routes.active = idx
				err := netcfg.RouteReplace(route.ifname, newgw, dest)
				if err == nil {
					resp.Data = append(resp.Data,
						peerActiveDataEntry{
							ConnectionID: route.connectionID,
							GroupID:      route.groupID,
							Timestamp:    time.Now().Format(env.TimeFormat),
						})
				} else {
					logger.Error().Println(pkgName, err)
					errIPs = append(errIPs, dest)
				}
			}
		}
	}

	r.Unlock()

	// TODO thing about sending errors to controller
	if len(resp.Data) > 0 {
		resp.Now()
		raw, err := json.Marshal(resp)
		if err != nil {
			return err
		}

		r.writer.Write(raw)
	}

	if len(errIPs) > 0 {
		return fmt.Errorf("could not change routes to %s via %s", strings.Join(errIPs, ","), newgw)
	}

	return nil
}
