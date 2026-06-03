//go:build linux

package cmdrunner

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/EHLO1/keel/internal/adapter/proc"
)

type Client struct {
	log       *slog.Logger
	waitDelay time.Duration
}

func NewClient(log *slog.Logger, waitDelay time.Duration) *Client {
	return &Client{
		log:       log,
		waitDelay: waitDelay,
	}
}

func (c *Client) Run(ctx context.Context, dir string, env []string, name string, args ...string) (proc.Result, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Dir = dir
	cmd.Env = env

	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb

	cmd.Cancel = func() error {
		err := syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
		if err == syscall.ESRCH {
			return os.ErrProcessDone
		}
		return err
	}
	// Grace period after Cancel before SIGKILL
	cmd.WaitDelay = c.waitDelay

	if err := cmd.Start(); err != nil {
		return proc.Result{}, err
	}
	pgid := cmd.Process.Pid

	waitErr := cmd.Wait()

	if ctx.Err() != nil {
		_ = syscall.Kill(-pgid, syscall.SIGKILL)
	}

	res := proc.Result{Stdout: out.Bytes(), Stderr: errb.Bytes()}
	var ee *exec.ExitError
	if errors.As(waitErr, &ee) {
		res.Code = ee.ExitCode()
	}
	return res, waitErr
}
