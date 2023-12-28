package main

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		fail("Must specify a command.")
	}

	switch args[0] {
	case "init":
		doInit()
	case "annotate":
		doAnnotate()
	case "status":
		doStatus()
	case "ignore":
		fmt.Println("In the future, this will add elements to be ignored (rn I'll edit it with my editor.)")
	default:
		fmt.Println("Unknown command.")
		os.Exit(1)
	}

}

func doInit() {
	// variable reused for errors, because I don't feel like naming each error separately
	var err error

	// create directory
	err = os.Mkdir(".memc", 0755)
	if err != nil {
		fail("Error creating new directory:", err)
	}

	// pre-create the file which will store all of the tags
	err = os.WriteFile(".memc/tags", []byte(""), 0644)
	if err != nil {
		_ = os.RemoveAll(".memc") // for now, ignore this error, although it would be nice to notify the user
		fail("Error creating tag file:", err)
	}

	err = os.WriteFile(".memc/ignore", []byte(""), 0644)
	if err != nil {
		_ = os.RemoveAll(".memc")
		fail("Error creating ignore file:", err)
	}

}

func doAnnotate() {
	ensureInit()
	tags := readTags()
	ignore := readIgnore()
	allFiles := candidateFiles()
	untagged := filterAnnotated(allFiles, tags, ignore)

	for _, file := range untagged {
		// setup image viewer
		display := exec.Command("feh", file)
		displayError := make(chan error)
		go func() {
			err := display.Run()
			displayError <- err
		}()

		// Clear file
		os.WriteFile(".memc/tmp", []byte{}, 0644)

		// we should check here to avoid redundantly opening the editor while not operating on a real image

		// open editor
		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "vi"
		}

		cmd := exec.Command(editor, ".memc/tmp")
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			fail("Error when opening editor:", err)
		}

		// read what the user has written
		data, err := os.ReadFile(".memc/tmp")
		if err != nil {
			fail("Error when reading temporary file:", err)
		}

		_ = file

		tag := strings.Fields(string(data))

		_ = display.Process.Kill() // should we ignore the error?

		// I assume if the tag is left empty, it means that the user wants to stop tagging
		if len(tag) == 0 {
			select {
			case err := <-displayError:
				fmt.Println("Display: ", err)
			default:
			}
			return
		}

		// append to tag file
		tagFile, err := os.OpenFile(".memc/tags", os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			fail("Error opening tag file", err)
		}
		defer tagFile.Close()

		tagFile.WriteString(file + ": " + strings.Join(tag, " ") + "\n")
	}
}

func doStatus() {
	ensureInit()

	tags := readTags()
	ignore := readIgnore()
	allFiles := candidateFiles()

	untagged := filterAnnotated(allFiles, tags, ignore)

	// count overlap (kinda bad, but whatever)
	numIgnored := 0
	for _, elem := range allFiles {
		if _, exists := ignore[elem]; exists {
			numIgnored++
		}
	}

	// count tags/ignore overlap (this is erroneous)
	// simple error checking for the future
	numIgnoreTagOverlap := 0
	for elem := range tags {
		if _, exists := ignore[elem]; exists {
			numIgnoreTagOverlap++
		}
	}

	fmt.Printf(" %d / %d (%d ignored)\n", len(allFiles)-len(untagged)-numIgnored, len(allFiles)-numIgnored, numIgnored)

	if numIgnoreTagOverlap > 0 {
		fmt.Printf("Also, there are %d elements that are both tagged and ignored????\n", numIgnoreTagOverlap)
	}

	fmt.Println(untagged)
}

func filterAnnotated(candidates []string, tags map[string][]string, ignore Set) []string {
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

func candidateFiles() []string {
	// todo: according to docs, filepath.WalkDir passes the path with os-specific separators
	// I'll have to check that on Windows
	var allFiles []string
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
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

	})

	if err != nil {
		fail("Error walking current directory:", err)
	}

	return allFiles
}

func readTags() map[string][]string {
	file, err := os.Open(".memc/tags")
	if err != nil {
		fail("Error while opening tag file for reading:", err)
	}
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

type Set map[string]struct{}

func readIgnore() Set {
	file, err := os.Open(".memc/ignore")
	if err != nil {
		fail("Error while opening ignore file for reading:", err)
	}
	defer file.Close()

	ignore := make(Set)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		s := strings.TrimSpace(scanner.Text())
		ignore[s] = struct{}{} // add a sentinel value
	}

	return ignore
}

func ensureInit() {
	if !initialized() {
		fail("Not initialized. Run `memc init`.")
	}
}

func initialized() bool {
	_, err := os.Stat(".memc")

	if os.IsNotExist(err) {
		return false
	}

	if err != nil {
		fail("Error checking directory: ", err)
	}

	return true
}

func fail(err ...any) {
	fmt.Println(err...)
	os.Exit(1)
}
