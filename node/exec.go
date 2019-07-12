package main

import (
	"context"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"time"

	sandboxfs "github.com/Hustle/sandboxfs-go"
)

func sandboxfsExec() int {

	// create a temp directory to use as a mount point
	mountPoint, err := ioutil.TempDir("", "sfs_exec-")
	if err != nil {
		logger.Fatalln("creating mount point:", err)
	}
	defer os.Remove(mountPoint)
	logger.Println("created mount point:", mountPoint)

	// try to start sandboxfs
	sfs := &sandboxfs.Sandboxfs{
		SandboxfsPath: sandboxfsPath,
		MountPoint:    mountPoint,
	}

	sfsCtx, cancelSfsCtx := context.WithCancel(context.Background())
	if err := sfs.Run(sfsCtx); err != nil {
		logger.Fatalf("starting sandboxfs: %v", err)
	}

	// do our best to shutdown sandboxfs cleanly when we exit
	defer cancelSfsCtx()
	sfs.UnmountOnExit()

	// map things into the sandbox
	logger.Println("processing runfiles manifest")
	if err := mapFilesFromManifest(sfs, manifest, sourceMappings); err != nil {
		logger.Fatalf("while processing runfiles manifest: %v", err)
	}

	// also map a node_modules/.cache dir for things to use
	cacheDir, err := ioutil.TempDir("", "sfs_exec-cache-")
	if err != nil {
		logger.Fatalln("creating node_modules/.cache dir:", err)
	}
	defer os.RemoveAll(cacheDir)

	if err := sfs.Map("/node_modules/.cache", cacheDir, true); err != nil {
		logger.Fatalf("mapping node_modules/.cache dir:", err)
	}

	logger.Println("waiting for sandboxfs...")
	if err := sfs.WaitUntilReady(10 * time.Second); err != nil {
		logger.Fatal(err)
	}
	logger.Println("sandboxfs is ready")

	// try running the actual command
	logger.Println("starting:", strings.Join(execArgs, " "))
	scriptCmd := &exec.Cmd{
		Path: nodejsPath,
		Args: append([]string{"node"}, execArgs...),
		Dir:  mountPoint,

		Stdin:  os.Stdin,
		Stderr: os.Stderr,
		Stdout: os.Stdout,
	}
	if err := scriptCmd.Start(); err != nil {
		logger.Fatal(err)
	}
	defer scriptCmd.Process.Kill()

	// wait for the script to finish running
	if err := scriptCmd.Wait(); err != nil {
		if _, ok := err.(*exec.ExitError); !ok {
			// this is an error that is *not* ExitError
			logger.Fatal(err)
		}
	}

	// return the command's exit status
	exit := scriptCmd.ProcessState.ExitCode()
	if exit == 0 {
		logger.Println("command completed successfully")
	} else {
		logger.Println("command exited with non-zero status:", exit)
	}
	return exit
}
