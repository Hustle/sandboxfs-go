package main

import (
	"fmt"
	"os"
	"path"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/spf13/pflag"
)

const (
	DefaultNodeJs    = "nodejs/bin/node"
	DefaultSandboxfs = "tools/sandboxfs/bin/sandboxfs"
)

var (
	sourceMappings map[string]string
	manifest       string
	nodejsPath     string
	sandboxfsPath  string
	envFiles       []string

	execArgs []string
)

func defineFlags() {
	pflag.Usage = func() {
		fmt.Fprintf(
			os.Stderr,
			Usage,
			path.Base(os.Args[0]),
		)
		pflag.PrintDefaults()
	}

	pflag.StringToStringVar(
		&sourceMappings,
		"source-mapping",
		map[string]string{},
		"mapping from runfiles path to virtual path",
	)
	pflag.StringVar(
		&manifest,
		"runfiles-manifest",
		"",
		"Bazel-style runfiles manifest",
	)
	pflag.StringVar(
		&sandboxfsPath,
		"sandboxfs",
		DefaultSandboxfs,
		"path to the sandboxfs executable",
	)
	pflag.StringVar(
		&nodejsPath,
		"node",
		DefaultNodeJs,
		"path to the NodeJS executable",
	)
	pflag.StringArrayVar(
		&envFiles,
		"env-file",
		[]string{},
		"file with environment variables to set",
	)
}

func resolvePathToRunfile(pathVar *string) error {
	resolved, err := bazel.Runfile(*pathVar)
	if err != nil {
		return fmt.Errorf("resolving location of %s: %v", *pathVar, err)
	}

	*pathVar = resolved
	return nil
}

func parseFlags() {
	pflag.Parse()

	if pflag.NArg() < 1 {
		pflag.Usage()
		os.Exit(1)
	} else {
		execArgs = pflag.Args()[0:]
	}

	if err := resolvePathToRunfile(&nodejsPath); err != nil {
		logger.Fatal(err)
	}

	if err := resolvePathToRunfile(&sandboxfsPath); err != nil {
		logger.Fatal(err)
	}
}
