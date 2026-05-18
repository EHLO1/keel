package icmp

import (
	"net"
	"time"
)

type ICMPTargets struct {
	ObservedAt time.Time

	Targets []Target
}

type Target struct {
	IP        net.IP
	Reachable bool
}
