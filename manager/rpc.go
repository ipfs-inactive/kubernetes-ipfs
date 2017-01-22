package main

import (
	"context"
	"os"
	"os/exec"
	"time"

	uplib "github.com/ipfs/ipfs-update/lib"
	uputil "github.com/ipfs/ipfs-update/util"
)

type Manager struct {
	ctx  context.Context
	proc *exec.Cmd
}

func NewManager(ctx context.Context) *Manager {
	res := &Manager{}
	return res

}

const (
	S_RUNNING RunStatus = iota
	S_STOPPED
	S_ERROR
)

type RunStatus int

func (mn *Manager) Status(_ struct{}, status *RunStatus) error {
	return nil

}

// Removes IPFS_PATH as side effect
// Forcibly kills ipfs daemon as a side effect
func (mn *Manager) ChangeVersion(version string, _ *struct{}) error {
	// TODO: stop existing
	mn.kill()

	repo := uputil.IpfsDir()
	err := os.RemoveAll(repo)
	if err != nil {
		return err
	}

	in, err := uplib.NewInstall(uputil.IpfsVersionPath, version, false)
	if err != nil {
		return err
	}
	return in.Run()
}

func (mn *Manager) kill() {
	if mn.proc == nil {
		return
	}

	// ignore error
	_ = mn.proc.Process.Kill()
	mn.proc = nil
}

func (mn *Manager) Start(args []string, _ *struct{}) error {
	mn.kill()

	// ipfs is in path
	argsAll := make([]string, 1, len(args)+1)
	argsAll[0] = "daemon"
	argsAll = append(argsAll, args...)
	mn.proc = exec.CommandContext(mn.ctx, "ipfs", argsAll...)
	err := mn.proc.Start()
	if err != nil {
		mn.proc = nil
	}
	return err
}

type StopStatus int

const (
	STOP_SOFT StopStatus = iota
	STOP_DEAD
	STOP_KILL
)

// Stops the daemon softly
// Kills it after 30s
func (mn *Manager) Stop(_ struct{}, status *StopStatus) error {
	ctx, cancel := context.WithTimeout(mn.ctx, 30*time.Second)
	defer cancel()

	wait := make(chan error)
	go func() {
		wait <- mn.proc.Wait()
	}()

	mn.proc.Process.Signal(os.Interrupt)
	select {
	case w := <-wait:
		if w == nil {
			*status = STOP_SOFT
		} else {
			*status = STOP_DEAD
		}
		return nil
	case <-ctx.Done():
		mn.kill()
		*status = STOP_KILL
		return nil
	}
}
