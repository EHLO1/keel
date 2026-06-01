package policy

import "fmt"

type Role int

const (
	RoleUnhealthy Role = iota
	RoleNotReady
	RoleStandby
	RolePrimary
)

func (r Role) String() string {
	switch r {
	case RoleUnhealthy:
		return "UNHEALTHY"
	case RoleNotReady:
		return "NOTREADY"
	case RoleStandby:
		return "STANDBY"
	case RolePrimary:
		return "PRIMARY"
	default:
		return fmt.Sprintf("Role(%d)", int(r))
	}
}

func ParseRole(s string) (Role, error) {
	switch s {
	case "UNHEALTHY":
		return RoleUnhealthy, nil
	case "NOTREADY":
		return RoleNotReady, nil
	case "STANDBY":
		return RoleStandby, nil
	case "PRIMARY":
		return RolePrimary, nil
	default:
		return RoleUnhealthy, fmt.Errorf("invalid role %q", s)
	}
}

func (r Role) MarshalText() ([]byte, error) { return []byte(r.String()), nil }

func (r *Role) UnmarshalText(b []byte) error {
	parsed, err := ParseRole(string(b))
	if err != nil {
		return err
	}
	*r = parsed
	return nil
}
