//+build !windows,!plan9

// Package process manages the lifecyle of processes and process groups
package process

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
	"time"
)

// TODO(nightlyone) remove me, when everything is implemented
var errUnimplemented = errors.New("process: function not implemented")

// Group is a process group we manage.
type Group struct {
	pid   int
	waitc <-chan error
}

// Background runs a command which can fork other commands (e.g. a shell script) building a tree of processes.
// This tree of processes are managed in a their own process group.
func Background(cmd *exec.Cmd) (*Group, error) {
	panic(errUnimplemented)
}

// IsLeader determines, whether g is still the leader of the process group
func (g *Group) IsLeader() (ok bool, err error) {
	if g == nil {
		return false, syscall.ESRCH
	}
	pgid, err := syscall.Getpgid(g.pid)
	if err != nil {
		return false, err
	}

	// Pids 0 and 1 will have special meaning, so don't return them.
	if pgid < 2 {
		return false, nil
	}

	// the process is not the leader?
	if pgid != g.pid {
		return false, nil
	}
	return true, nil
}

// Signal sends POSIX signal sig to this process group
func (g *Group) Signal(sig os.Signal) error {
	if g == nil {
		return syscall.ESRCH
	}

	// This just creates a process object from a Pid in Unix
	// instead of actually searching it.
	grp, _ := os.FindProcess(-g.pid)
	return grp.Signal(sig)
}

// Terminate first tries to gracefully terminate the process, waits patience time, then does final termination and waits for it to exit.
func (g *Group) Terminate(patience time.Duration) error {
	var terminated bool
	var errWait error

	// did we exit in the meantime?
	select {
	case errWait = <-g.waitc:
		return errWait
	default:
	}

	// try to be soft
	if err := g.Signal(syscall.SIGTERM); err != nil {
		return err
	}
	select {
	case errWait = <-g.waitc:
		terminated = true
	case <-time.After(patience):
	}
	if terminated {
		return nil
	}

	// do it the hard way
	if err := g.Signal(syscall.SIGKILL); err != nil {
		return err
	}

	// But we need to wait on the result now
	<-g.waitc
	return nil

}
