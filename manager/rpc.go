package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/exec"
	"syscall"
	"time"

	uplib "github.com/ipfs/ipfs-update/lib"
	uputil "github.com/ipfs/ipfs-update/util"
)

var (
	lg = log.New(os.Stdout, "rpc: ", log.Ldate)
)

type Manager struct {
	ctx  context.Context
	proc *exec.Cmd
}

func NewManager(ctx context.Context) *Manager {
	res := &Manager{ctx: ctx}
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

	lg.Printf("killing PID: %d", mn.proc.Process.Pid)

	// ignore error
	_ = mn.proc.Process.Kill()
	lg.Print("killed")
	mn.proc = nil
}

func (mn *Manager) Start(args []string, _ *struct{}) error {
	mn.kill()
	lg.Println("starting")

	// ipfs is in path
	argsAll := make([]string, 1, len(args)+1)
	argsAll[0] = "daemon"
	argsAll = append(argsAll, args...)
	mn.proc = exec.CommandContext(mn.ctx, "ipfs", argsAll...)
	err := mn.proc.Start()
	if err != nil {
		mn.proc = nil
	}
	lg.Printf("started 'ipfs daemon' with args %v, PID: %d", args, mn.proc.Process.Pid)
	return err
}

// Stops the daemon softly
// Kills it after 30s
func (mn *Manager) Stop(_ struct{}, status *int) error {
	if mn.proc == nil {
		return errors.New("no process is running")
	}

	ctx, cancel := context.WithTimeout(mn.ctx, 30*time.Second)
	defer cancel()

	lg.Println("stopping")
	wait := make(chan error)
	go func() {
		wait <- mn.proc.Wait()
	}()

	lg.Printf("sent SIGINT to PID: %d", mn.proc.Process.Pid)
	mn.proc.Process.Signal(os.Interrupt)
	select {
	case err := <-wait:
		if err == nil {
			lg.Println("stop went fine")
			*status = 0
		} else {
			lg.Printf("wait for stop returned error: %s", err)
			exiterr, ok := err.(*exec.ExitError)
			if ok {
				if st, ok := exiterr.Sys().(syscall.WaitStatus); ok {
					*status = st.ExitStatus()
					return nil
				}
			}
			lg.Print("could not read exit code from error: ", err)
			*status = -2
			return nil
		}
		return nil
	case <-ctx.Done():
		lg.Print("timeout, killing")
		mn.kill()
		*status = 124
		return nil
	}
}
