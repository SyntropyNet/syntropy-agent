package servicemon

import (
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
)

// Group or routes. Destination will be map key
type routeList struct {
	list []*routeEntry
}

func newRouteList() *routeList {
	return &routeList{}
}
func (rl *routeList) Dump() {
	for _, r := range rl.list {
		logger.Debug().Println(pkgName, r)
	}
}

// Returns total count of entries in this service route list
func (rl *routeList) Count() int {
	return len(rl.list)
}

// Returns pending entries to be added and/or deleted
func (rl *routeList) Pending() (add, del int) {
	for _, r := range rl.list {
		if r.CheckFlag(rfPendingAdd) {
			add++
		}
		if r.CheckFlag(rfPendingDel) {
			del++
		}
	}
	return add, del
}

func (rl *routeList) resetPending() {
	for _, r := range rl.list {
		r.ClearFlags(rfPendingAdd | rfPendingDel)
	}
}

// Searches for Public link.
// If not found - returns first in list
func (rl *routeList) GetDefault() *routeEntry {
	// TODO implement Public search.
	// Shorcut now - choose active link
	re := rl.GetActive()
	if re != nil {
		return re
	}

	// Fallback to first non deleted route, if no active set yet
	for _, re = range rl.list {
		if !re.CheckFlag(rfPendingDel) {
			return re
		}
	}

	return nil
}

// Returns active route from the set
func (rl *routeList) GetActive() *routeEntry {
	for _, r := range rl.list {
		if r.CheckFlag(rfActive) {
			return r
		}
	}

	return nil
}

func (rl *routeList) Add(re *routeEntry) {
	// Dupplicate entries happen when WSS connection was lost
	// and after reconnecting controller sends whole config
	for _, r := range rl.list {
		if r.gateway == re.gateway {
			// skip dupplicate entry
			return
		}
	}

	re.SetFlag(rfPendingAdd)
	// Note: active flag will be marked on apply

	rl.list = append(rl.list, re)
}

func (rl *routeList) MarkDel(gateway string) {
	// Dupplicate entries happen when WSS connection was lost
	// and after reconnecting controller sends whole config
	for _, r := range rl.list {
		if r.gateway == gateway {
			r.SetFlag(rfPendingDel)
			return
		}
	}
}

func (rl *routeList) Find(gateway string) *routeEntry {
	for _, r := range rl.list {
		if r.gateway == gateway {
			return r
		}
	}
	return nil
}
