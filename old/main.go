package main

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/agnivade/levenshtein"
)

/*
	TODO: Recursively find the repository in parent paths.
		Idea: make a struct which will contain the absolute paths after being found, then pass it around.
		Should be returned by `ensureInit` or something.
*/

func main() {
	if len(os.Args) < 2 {
		fail("Must specify a command. Type 'help'.")
	}

	switch os.Args[1] {
	case "i":
	case "init":
		doInit()
	case "a":
	case "annotate":
		doAnnotate()
	case "s":
	case "status":
		doStatus()
	case "g":
	case "ignore":
		doIgnore(os.Args[2:])
	case "f":
	case "find": // this should also launch some sort of interactive mode where you can preview the maymays
		doFind(os.Args[2:])
	default:
		fail("Unknown command.")
	}

}

/* Constants */
const (
	DIRECTORY            = ".memc"
	TAGS_FILE            = DIRECTORY + "/tags"
	IGNORE_FILE          = DIRECTORY + "/ignore"
	DEFAULT_EDITOR       = "vi"
	DEFAULT_IMAGE_VIEWER = "feh"
)

/* Command line functions */
// Initializes a new "image repository"
func doInit() {
	// create directory
	failWhen("Error creating new directory")(os.Mkdir(DIRECTORY, 0755))

	// pre-create the file which will store all of the tags
	failCleanupWhen("Error creating tag file")(func() { _ = os.RemoveAll(DIRECTORY) })(os.WriteFile(TAGS_FILE, []byte(""), 0644))

	// create ignore fire
	failCleanupWhen("Error creating ignore file")(func() { _ = os.RemoveAll(DIRECTORY) })(os.WriteFile(IGNORE_FILE, []byte(""), 0644))
}

// Begin annotation process
// opens an image viewer and an editor
func doAnnotate() {
	ensureInit()

	tags := openTags(TAGS_FILE)                         // these ones are already tagged files
	ignore := openIgnore(IGNORE_FILE)                   // these ones are the files to ignore
	allFiles := gatherCandidateFiles()                  // all the files that can be checked
	untagged := filterAnnotated(allFiles, tags, ignore) // this way, we can get the untagged files

	for _, file := range untagged {
		// setup image viewer
		display := exec.Command(DEFAULT_IMAGE_VIEWER, file)
		displayError := make(chan error)
		go func() {
			err := display.Run()
			displayError <- err
		}()

		const TMP = DIRECTORY + "/tmp"

		// Clear file
		os.WriteFile(TMP, []byte{}, 0644)

		// we should check here to avoid redundantly opening the editor while not operating on a real image

		// open editor
		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = DEFAULT_EDITOR
		}

		cmd := exec.Command(editor, TMP)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		failWhen("Error opening editor")(cmd.Run())

		// read what the user has written
		data := failIf(os.ReadFile(TMP))("Error reading temporary file.")

		tags := strings.Fields(string(data))

		_ = display.Process.Kill() // should we ignore the error?

		// If the 'tags' is left empty, it means that the user wants to stop tagging
		if len(tags) == 0 {
			select {
			case err := <-displayError:
				fmt.Println("Display: ", err)
			default:
			}
			return
		}

		/* Add to tag file */
		tagFile := failIf(os.OpenFile(TAGS_FILE, os.O_APPEND|os.O_WRONLY, 0644))("Error opening tag file")
		defer tagFile.Close()

		// Append to tag file (separate by strings)
		failIf(tagFile.WriteString(file + ": " + strings.Join(tags, " ")))("Error writing to tags file")
	}
}

/* Check "image repo" status. */
func doStatus() {
	ensureInit()

	tags := openTags(TAGS_FILE)
	ignore := openIgnore(IGNORE_FILE)
	allFiles := gatherCandidateFiles()

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

	// Printing output
	fmt.Printf(" %d / %d (%d ignored)\n", len(allFiles)-len(untagged)-numIgnored, len(allFiles)-numIgnored, numIgnored)

	if numIgnoreTagOverlap > 0 {
		fmt.Printf("Also, there are %d elements that are both tagged and ignored????\n", numIgnoreTagOverlap)
	}

	fmt.Println(untagged)
}

// Find an image with a search query
func doFind(searchTerms []string) {
	ensureInit()

	tags := openTags(TAGS_FILE)

	// slow, O(n^3) implementation
	// also, it works badly. I should find a simpler way to do it or at least a one that works better.
	scores := []Candidate{}
	for filename, description := range tags {
		cumDistance := 0
		for _, queryPart := range searchTerms {
			minDistance := levenshtein.ComputeDistance(queryPart, filename)
			for _, tag := range description {
				distance := levenshtein.ComputeDistance(queryPart, tag)
				minDistance = min(minDistance, distance)
			}

			cumDistance += minDistance
		}

		scores = append(scores, Candidate{filename, cumDistance})
	}

	/* sort scores and return max 10 results */
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].difference < scores[j].difference
	})

	trimlen := min(10, len(scores))

	for _, cand := range scores[:trimlen] {
		fmt.Printf("%s: %d\n", cand.filename, cand.difference)
	}
}

func doIgnore(files []string) {
	ensureInit()

	ignores := openIgnore(IGNORE_FILE)
	tags := openTags(TAGS_FILE)

	ignoreFile := failIf(os.OpenFile(IGNORE_FILE, os.O_APPEND|os.O_WRONLY, 0644))("Error opening ignore file")
	defer ignoreFile.Close()

loop:
	for _, filename := range files {
		// check if it's a subpath to repository
		if !isSubpath(".", filename) {
			fmt.Printf("ERROR: %s is not in this repository.", filename)
			continue
		}

		// check if it's ignored
		for ign := range ignores {
			if ign == filename {
				fmt.Printf("WARNING: %s is already ignored.", filename)
				continue loop
			}
		}

		// check if it's already tagged
		// file will still be added to "ignored"
		for k := range tags {
			if k == filename {
				fmt.Printf("WARNING: %s is tagged. This will exclude it from search.", filename)
				break
			}
		}

		// Append to ignore file (separate by lines)
		failIf(ignoreFile.WriteString(filename + "\n"))("Error writing to ignore file")
	}
}

/* Small util stuff */
type Candidate struct {
	filename   string
	difference int
}

func ensureInit() {
	if !initialized() {
		fail("Not initialized. Run `memc init`.")
	}
}

func initialized() bool {
	_, err := os.Stat(DIRECTORY)

	if os.IsNotExist(err) {
		return false
	}

	// User might want to know about this, so we exit.
	if err != nil {
		fail("Error checking directory: ", err)
	}

	return true
}
