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
	file := filepath.Clean(path) + fileName

	return &VRRPRole{
		file: file,
	}
}

func (v *VRRPRole) Observe() (string, error) {
	f, err := os.ReadFile(v.file)
	if errors.Is(err, fs.ErrNotExist) {
		return "", fmt.Errorf("vrrp role file does not exist: %w", err)
	}
	if err != nil {
		return strings.TrimSpace(string(f)), nil
	}
	return "", err
}
