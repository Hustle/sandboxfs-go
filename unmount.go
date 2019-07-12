package sandboxfs

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"syscall"
	"time"
)

// UnmountOnExit attempts to ensure that the sandboxfs mount is unmounted when
// this process exits.
//
// To do this, we open a file in the mount point, and then send SIGTERM to
// sandboxfs. According to spec, sandboxfs will not actually unmount and exit
// until the file is closed.
func (sfs *Sandboxfs) UnmountOnExit() error {
	if sfs.unmountHandle != nil {
		// already called
		return nil
	}

	// create a temp file to serve as our sentinel
	tmp, err := ioutil.TempFile("", "sfs-unmount-")
	if err != nil {
		return fmt.Errorf("opening mount point: %v", err)
	}

	// map that temp file into the sandbox
	sfs.Map("/.sfs-unmount", tmp.Name(), false)
	sfs.outstandingRequests.Wait()

	// hold open a handle on the sentinel
	sfs.unmountHandle, err = os.Open(path.Join(sfs.MountPoint, "/.sfs-unmount"))
	if err != nil {
		return fmt.Errorf("opening unmount sentinel in sandbox: %v", err)
	}

	// we should be able to unlink the underlying file now
	os.Remove(tmp.Name())

	// send sandboxfs a SIGTERM so it exits whenever we close the handle, and
	// release it so that it survives us
	sfs.cmd.Process.Signal(syscall.SIGTERM)
	sfs.cmd.Process.Release()

	return nil
}

// Unmount tries to cleanly stop the sandboxfs instance, and then kills it
// forcefully.
func (sfs *Sandboxfs) Unmount(timeout time.Duration) error {
	if sfs.unmountHandle != nil {
		return fmt.Errorf("Unmount cannot be called after UnmountOnExit")
	}

	sfs.cancel()
	sfs.cmd.Process.Signal(syscall.SIGTERM)

	stopped := make(chan error)
	go func() {
		stopped <- sfs.cmd.Wait()
		close(stopped)
	}()

	select {
	case err := <-stopped:
		if _, ok := err.(*exec.ExitError); !ok {
			// this is an error that is *not* ExitError
			return err
		} else {
			return nil
		}
	case <-time.After(timeout):
		sfs.cmd.Process.Kill()
		return fmt.Errorf("killed sandboxfs after timeout")
	}
}
