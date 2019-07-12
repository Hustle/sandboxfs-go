package main

import (
	"bufio"
	"log"
	"os"
	"strings"

	sandboxfs "github.com/Hustle/sandboxfs-go"
)

// checkAndTrimPrefix checks if s starts with the provided prefix string. If
// it does, it returns s without the leading prefix string, and a boolean true.
//
// If s doesn't start with the provided prefix, it returns s unchanged and a
// boolean false.
func checkAndTrimPrefix(s, prefix string) (string, bool) {
	if strings.HasPrefix(s, prefix) {
		return s[len(prefix):], true
	} else {
		return s, false
	}
}

func mapFilesFromManifest(
	sfs *sandboxfs.Sandboxfs,
	manifest string,
	sourceMappings map[string]string,
) error {
	file, err := os.Open(manifest)
	if err != nil {
		return err
	}
	defer file.Close()

	// keep track of our mappings in a map. this means that if a particular file
	// appears multiple times in the manifest, the last occurence will win.
	mappings := make(map[string]string)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)

		if len(fields) != 2 {
			log.Printf("skipping invalid line in manifest:\n-> %s\n", line)
			continue
		}

		runfilePath, actualFilePath := fields[0], fields[1]

		// check if the path is inside of a node_modules dir
		pathComponents := strings.Split(runfilePath, string(os.PathSeparator))
		for i, component := range pathComponents {
			if component != "node_modules" {
				continue
			}

			// the current component is "node_modules", so if join all components
			// from the current index, we'll get the relative path of this file
			// (eg., "node_modules/X/Y.js")
			relPath := strings.Join(pathComponents[i:], string(os.PathSeparator))

			// map the actual file into our virtual node_modules
			mappings["/"+relPath] = actualFilePath
			break
		}

		// also check if the path matches any of the source mappings
		for sourceDir, sourceTarget := range sourceMappings {
			relPath, isInSourceDir := checkAndTrimPrefix(runfilePath, sourceDir)
			if isInSourceDir {
				relPath = sourceTarget + string(os.PathSeparator) + relPath
				mappings[relPath] = actualFilePath
			}
		}
	}

	// send all mappings to sandboxfs
	for relPath, actualFilePath := range mappings {
		sfs.Map(relPath, actualFilePath, false)
	}

	return nil
}
