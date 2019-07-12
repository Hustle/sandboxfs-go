package main

import (
	"fmt"
	"log"
	"os"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/joho/godotenv"
)

const (
	Usage = `Run a NodeJS script in a sandboxfs tree

Usage:
  %s [flags] <entry point> [arguments ...]

This utility will run a NodeJS script in a sandboxfs tree with sources and
node_modules present. By using sandboxfs, all sources and modules appear to be
real, local files to NodeJS and other tools -- even if they are actually located
in various locations on the real filesystem.

The list of potential files to mount into the tree is read from a Bazel runfiles
manifest, provided via command-line flags or the RUNFILES_MANIFEST_FILE environment
variable. Each line of this file should look like this:

  <file path in runfiles directory> <absolute path to real file>

For each line in the file, we check if the runfiles path is in a node_modules
directory. If so, we map the real file into the sandboxfs mount under /node_modules.

Alternatively, if the runfiles path is a descendant of a source mapping specified
with --source-mapping, we map the real file into the mount under the provided
virtual path.

Finally, we execute the specified entry point within the sandbox root.

Flags:
`
)

var (
	// logger without timestamps
	logger = log.New(os.Stderr, "[sandboxfs_node] ", 0)
)

func findRunfilesManifest() {
	if manifest == "" {
		manifest = os.Getenv("RUNFILES_MANIFEST_FILE")
		runfilesDir := os.Getenv("RUNFILES_DIR")

		if runfilesDir == "" && manifest == "" {
			logger.Fatal("need a runfiles manifest")
		} else if manifest == "" {
			manifest = fmt.Sprintf("%s_manifest", runfilesDir)
		}
	}
}

func loadEnvFiles() {
	for _, f := range envFiles {
		path, err := bazel.Runfile(f)
		if err != nil {
			logger.Fatalf("resolving location of %s: %v", f, err)
		}

		// load values from env file into the environment, overriding any
		// existing environment variables
		if err := godotenv.Overload(path); err != nil {
			logger.Fatalf("loading %s: %v", path, err)
		} else {
			logger.Println("loaded env file:", path)
		}
	}
}

func main() {
	defineFlags()
	parseFlags()
	findRunfilesManifest()
	loadEnvFiles()

	// do the thing, and exit with the wrapped command's exit status
	os.Exit(sandboxfsExec())
}
