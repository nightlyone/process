package process

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"
)

func TestAlreadyExecutingFails(t *testing.T) {
	t.Parallel()
	what := "/bin/true"
	cmd := exec.Command(what)
	cmd.Process = new(os.Process)
	_, err := Background(cmd)
	wantErr := fmt.Errorf("process: command already executing: %q", what)
	if err == nil || err.Error() != wantErr.Error() {
		t.Fatalf("got %v, expected error %v", err, wantErr)
	}
}

func TestAlreadyExecutedFails(t *testing.T) {
	t.Parallel()
	what := "/bin/true"
	cmd := exec.Command(what)
	cmd.ProcessState = new(os.ProcessState)
	_, err := Background(cmd)
	wantErr := fmt.Errorf("process: command already executed: %q", what)
	if err == nil || err.Error() != wantErr.Error() {
		t.Fatalf("got %v, expected error %v", err, wantErr)
	}
}

func TestSetsidConflictFails(t *testing.T) {
	t.Parallel()
	what := "true"
	cmd := exec.Command(what)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	_, err := Background(cmd)
	wantErr := errors.New("May not be used with a cmd.SysProcAttr.Setsid = true")
	if err == nil || err.Error() != wantErr.Error() {
		t.Fatalf("got %v, expected error %v", err, wantErr)
	}
}

func TestStartingNonExistingFailsRightAway(t *testing.T) {
	t.Parallel()
	cmd := exec.Command("/var/run/nonexistant")
	_, err := Background(cmd)
	wantErr := errors.New("fork/exec /var/run/nonexistant: no such file or directory")
	if err == nil || err.Error() != wantErr.Error() {
		t.Fatalf("got %v, expected error %v", err, wantErr)
	}
}

func TestBackgroundingWorks(t *testing.T) {
	t.Parallel()
	what := "true"
	cmd := exec.Command(what)
	g, err := Background(cmd)
	if err != nil {
		t.Fatalf("got unexpected error %v", err)
	}
	err = g.Terminate(0)
	if err != nil && err != syscall.ESRCH && err.Error() != errors.New("os: process already finished").Error() {
		t.Fatalf("cannot terminate: %v", err)
	}
}

func TestSoftKillWorks(t *testing.T) {
	t.Parallel()
	what := "sleep"
	cmd := exec.Command(what, "1")
	g, err := Background(cmd)
	if err != nil {
		t.Fatalf("got unexpected error %v", err)
	}
	err = g.Terminate(500 * time.Millisecond)
	if err != nil && err != syscall.ESRCH && err.Error() != errors.New("os: process already finished").Error() {
		t.Fatalf("cannot terminate: %v", err)
	}
}

func TestExitBeforeKill(t *testing.T) {
	t.Parallel()
	what := "false"
	cmd := exec.Command(what)
	g, err := Background(cmd)
	if err != nil {
		t.Fatalf("got unexpected error %v", err)
	}
	time.Sleep(5 * time.Millisecond)
	err = g.Terminate(500 * time.Millisecond)

	want := 1
	if got := getExitCode(err); got != want {
		t.Fatalf("expected error code %d, but got %d, because %v", want, got, err)
	}
}

func getExitCode(err error) int {
	if err == nil {
		return 0
	}
	e, ok := err.(*exec.ExitError)
	if !ok {
		return -1
	}
	status := e.ProcessState.Sys().(syscall.WaitStatus)
	if !status.Exited() {
		return -2
	}
	return status.ExitStatus()

}
