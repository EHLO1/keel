package filesystem

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type VRRPRole struct {
	file string
}

func NewVRRPRole(path string, fileName string) *VRRPRole {
	return &VRRPRole{
		file: filepath.Join(path, fileName),
	}
}

func (v *VRRPRole) Observe() (string, error) {
	f, err := os.ReadFile(v.file)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", fmt.Errorf("vrrp role file does not exist: %w", err)
		}
		return "", err
	}
	return strings.TrimSpace(string(f)), nil
}
