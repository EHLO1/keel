package policy

import "time"

type grace struct {
	trip  time.Duration
	clear time.Duration
}

func gracePeriod(q Qualifier, role Role) grace {
	switch q {
	case QPostgres:
		switch role {
		case RolePrimary:
			return grace{trip: 2 * time.Second, clear: 6 * time.Second}
		case RoleStandby:
			return grace{trip: 6 * time.Second, clear: 6 * time.Second}
		default:
			return grace{trip: 10 * time.Second, clear: 6 * time.Second}
		}
	case QVRRP:
		if role == RolePrimary {
			return grace{trip: 6 * time.Second, clear: 8 * time.Second}
		}
		return grace{trip: 10 * time.Second, clear: 8 * time.Second}
	case QValkey:
		switch role {
		case RolePrimary:
			return grace{trip: 4 * time.Second, clear: 4 * time.Second}
		case RoleStandby:
			return grace{trip: 8 * time.Second, clear: 4 * time.Second}
		default:
			return grace{trip: 10 * time.Second, clear: 4 * time.Second}
		}
	case QLoadBalancer:
		switch role {
		case RolePrimary:
			return grace{trip: 8 * time.Second, clear: 6 * time.Second}
		case RoleStandby:
			return grace{trip: 10 * time.Second, clear: 6 * time.Second}
		default:
			return grace{trip: 15 * time.Second, clear: 6 * time.Second}
		}
	case QSystemd:
		switch role {
		case RolePrimary:
			return grace{trip: 4 * time.Second, clear: 4 * time.Second}
		case RoleStandby:
			return grace{trip: 4 * time.Second, clear: 4 * time.Second}
		default:
			return grace{trip: 4 * time.Second, clear: 4 * time.Second}
		}
	default:
		return grace{trip: 5 * time.Second, clear: 5 * time.Second}
	}
}

type debouncer struct {
	confirmed Verdict // Qualifier is confirmed (to be) OK or is confirmed (to be) !OK
	lastOK    bool
	rawSince  time.Time
}

func (d *debouncer) observe(v Verdict, now time.Time, g grace) Verdict {
	if v.OK != d.lastOK {
		d.lastOK = v.OK
		d.rawSince = now
	}
	held := now.Sub(d.rawSince)

	switch {
	case d.confirmed.OK && !v.OK && held >= g.trip:
		d.confirmed = v // Failure is confirmed (trip)
	case !d.confirmed.OK && v.OK && held >= g.clear:
		d.confirmed = v // Recovery is confirmed (clear)
	}

	return d.confirmed
}
