package sandboxfs

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"
)

// Sandboxfs represents an instance of sandboxfs being prepared or run. It
// submits mapping requests to the sandboxfs instance, and keeps track of
// the number of outstanding requests to track whether the filesystem is ready.
//
// A Sandboxfs cannot be reused after calling its Run method.
type Sandboxfs struct {
	// Path to the sandboxfs binary
	SandboxfsPath string

	// Path to use as a mount point
	MountPoint string

	// Writer for the sandboxfs process's stderr
	Stderr io.Writer

	ctx    context.Context
	cancel context.CancelFunc

	requests            chan<- []byte
	outstandingRequests sync.WaitGroup

	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser

	started       bool
	unmountHandle *os.File
}

// Run starts sandboxfs and connects to it.
//
// The sandboxfs process is detached from the process group of the current
// process. Because of this, it is imperative to use UnmountOnExit or Unmount
// to shut down the sandboxfs process cleanly.
func (sfs *Sandboxfs) Run(ctx context.Context) error {
	if sfs.started {
		return fmt.Errorf("instance has already been started")
	}
	if sfs.SandboxfsPath == "" {
		return fmt.Errorf("SandboxfsPath must be specified")
	}
	if sfs.MountPoint == "" {
		return fmt.Errorf("MountPoint must be specified")
	}

	sfs.ctx, sfs.cancel = context.WithCancel(ctx)
	sfs.started = true

	requests := make(chan []byte)
	sfs.requests = requests

	sfs.cmd = exec.Command(sfs.SandboxfsPath, sfs.MountPoint)
	sfs.cmd.Stderr = sfs.Stderr

	// prevent sandboxfs from being terminated by detaching it from our
	// process group. this is necessary for UnmountOnExit to work.
	sfs.cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	var err error
	sfs.stdin, err = sfs.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("creating stdin pipe for sandboxfs: %v", err)
	}

	sfs.stdout, err = sfs.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("creating stdout pipe for sandboxfs: %v", err)
	}

	if err := sfs.cmd.Start(); err != nil {
		return fmt.Errorf("starting sandboxfs: %v", err)
	}

	go sfs.mapHandler(requests)
	go sfs.stdoutHandler()

	return nil
}
