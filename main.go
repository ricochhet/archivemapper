package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
)

var (
	name        = "archivemapper"
	buildDate   string
	gitHash     string
	buildOn     string
	versionFlag bool
)

func version() {
	fmt.Printf("%s-%s\n", name, gitHash)
	fmt.Printf("Build date: %s\n", buildDate)
	fmt.Printf("Build on: %s\n", buildOn)
	os.Exit(0)
}

func usage() {
	fmt.Println("Run without -h or -version flags to start.")
	flag.PrintDefaults()
}

func main() {
	flag.BoolVar(&versionFlag, "version", false, "show version")
	flag.BoolVar(&versionFlag, "v", false, "show version")

	output := flag.String("out", "output.json", "set output path")
	matchHash := flag.Bool("match-hash", true, "use file hash for matching")
	matchPath := flag.Bool("match-path", false, "use file path for matching")
	pathDepth := flag.Int("path-depth", 2, "file path depth to use for matching")
	formats := flag.String("formats", ".7z,.rar,.zip", "archive formats to search for")
	flag.Usage = usage

	flag.Parse()

	if versionFlag {
		version()
	}

	args := flag.Args()
	if len(args) < 2 {
		log.Fatalln("not enough arguments (wants: 2)")
	}

	src := args[0]
	dst := args[1]

	s, err := walkSource(src, strings.Split(*formats, ","))
	if err != nil {
		log.Fatalf("failed to walk source directory: %v\n", err)
	}

	d, err := walkDestination(dst)
	if err != nil {
		log.Fatalf("failed to walk destination directory: %v\n", err)
	}

	if err := writeJSON(*s, *d, *output, &CompareOptions{
		MatchHash: *matchHash,
		MatchPath: *matchPath,
		PathDepth: *pathDepth,
	}); err != nil {
		log.Fatalf("failed to write json: %v\n", err)
	}
}
