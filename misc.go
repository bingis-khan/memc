package main

import (
	"bufio"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type Ignores map[string]struct{}
type Tags map[string][]string

func filterAnnotated(candidates []string, tags Tags, ignore Ignores) []string {
	var untagged []string
	for _, elem := range candidates {
		_, isInTags := tags[elem]
		_, isInIgnore := ignore[elem]
		if !isInTags && !isInIgnore {
			untagged = append(untagged, elem)
		}
	}

	return untagged
}

func gatherCandidateFiles() []string {
	// todo: according to docs, filepath.WalkDir passes the path with os-specific separators
	// I'll have to check that on Windows
	var allFiles []string
	failWhen("Error walking current directory")(filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if path == "." {
			return nil
		}

		if strings.HasPrefix(d.Name(), ".") {
			return fs.SkipDir
		}

		if d.IsDir() {
			return nil
		}

		allFiles = append(allFiles, path)
		return nil

	}))

	return allFiles
}

// Open the tags file.
func openTags(tagfile string) Tags {
	file := failIf(os.Open(tagfile))("Error while opening tag file for reading")
	defer file.Close()

	allTags := make(map[string][]string)
	scanner := bufio.NewScanner(file)

	// apparently, there's a line limit equal to 64k characters per line, because Scanner does not allocate resources
	for scanner.Scan() {
		split := strings.Split(scanner.Text(), ":")
		path, unfieldedTags := split[0], split[1]
		tags := strings.Fields(unfieldedTags)
		allTags[path] = tags
	}

	return allTags
}

// An ignore file is composed of lines.
// In each line there is a path to a file or directory that should be ignored.
func openIgnore(ignorefile string) Ignores {
	file := failIf(os.Open(ignorefile))("Error while opening ignore file for reading")
	defer file.Close()

	ignore := make(Ignores)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		s := strings.TrimSpace(scanner.Text())
		ignore[s] = struct{}{} // add a sentinel value - no sets in Go
	}

	return ignore
}

// Checks if a path is a subpath of another.
// TODO: Actually implement it. Apparently it's not that straightforward in Go.
//
//	it's not exactly required, since I won't be abusing it anyway, so I'll leave it for later.
func isSubpath(base, p string) bool {
	return true
}
