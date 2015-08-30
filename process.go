//+build !windows,!plan9

// Package process manages the lifecyle of processes and process groups
package process

import (
	"errors"
	"fmt"
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
// NOTE: It overwrites cmd.SysProcAttr
func Background(cmd *exec.Cmd) (*Group, error) {
	if cmd.ProcessState != nil {
		return nil, fmt.Errorf("process: command already executed: %q", cmd.Path)
	}
	if cmd.Process != nil {
		return nil, fmt.Errorf("process: command already executing: %q", cmd.Path)
	}

	startc := make(chan startResult)
	waitc := make(chan error, 1)

	// NOTE: Cannot setsid and and setpgid in one child. Would need double fork or exec,
	// which makes things very hard.
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Setpgid: true,
		}

	} else {
		return nil, errUnimplemented
	}

	// Try to start process
	go startProcess(cmd, startc, waitc)
	res := <-startc

	if res.err != nil {
		return nil, res.err
	}

	// Now running in the background
	return &Group{
		pid:   res.pid,
		waitc: waitc,
	}, nil
}

// ErrNotLeader is returned when we request actions for a process group, but are not their process group leader
var ErrNotLeader = errors.New("process is not process group leader")

// Signal sends POSIX signal sig to this process group
func (g *Group) Signal(sig os.Signal) error {
	if g == nil || g.waitc == nil {
		return syscall.ESRCH
	}
	if leader, err := g.isLeader(); err != nil {
		return err
	} else if !leader {
		return ErrNotLeader
	}

	// This just creates a process object from a Pid in Unix
	// instead of actually searching it.
	grp, _ := os.FindProcess(-g.pid)
	return grp.Signal(sig)
}

// Terminate first tries to gracefully terminate the process, waits patience time, then does final termination and waits for it to exit.
func (g *Group) Terminate(patience time.Duration) error {
	var terminated bool

	// did we exit in the meantime?
	select {
	case errWait, opened := <-g.waitc:
		if opened {
			<-g.waitc
			g.waitc = nil
			return errWait
		}
	default:
	}

	// try to be soft
	if err := g.Signal(syscall.SIGTERM); err != nil {
		return err
	}

	// wait at most patience time for exit
	select {
	case _, opened := <-g.waitc:
		if opened {
			<-g.waitc
			g.waitc = nil
			terminated = true
		}
	case <-time.After(patience):
	}

	// exited gracefully
	if terminated {
		return nil
	}

	// do it the hard way
	if err := g.Signal(syscall.SIGKILL); err != nil {
		return err
	}

	// But we need to wait on the result now
	<-g.waitc
	<-g.waitc
	g.waitc = nil
	return nil

}

// isLeader determines, whether g is still the leader of the process group
func (g *Group) isLeader() (ok bool, err error) {
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

type startResult struct {
	pid int
	err error
}

// startProcess tries to a new process. Start result in startc, exit result in waitc.
func startProcess(cmd *exec.Cmd, startc chan<- startResult, waitc chan<- error) {
	var res startResult

	// Startup new process
	if err := cmd.Start(); err != nil {
		res.err = err
		startc <- res
		return
	}
	res.pid = cmd.Process.Pid
	startc <- res
	close(startc)

	// No wait until we finish or get killed
	waitc <- cmd.Wait()
	close(waitc)
}
