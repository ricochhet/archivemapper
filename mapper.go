package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/gen2brain/go-unarr"
)

type (
	SourceFiles      map[string]SourceFile
	DestinationFiles map[string]DestinationFile
	MappedFiles      map[string]MappedFile
)

type Output struct {
	Matches MappedFiles `json:"matches,omitempty"`
	Stray   MappedFiles `json:"stray,omitempty"`
}

type SourceFile struct {
	Path string `json:"path"`
	Name string `json:"name"`
	Size int    `json:"size"`
}

type DestinationFile struct {
	Path string `json:"path"`
	Size int    `json:"size"`
}

type MappedFile struct {
	Src *SourceFile      `json:"src,omitempty"`
	Dst *DestinationFile `json:"dst,omitempty"`
}

type CompareOptions struct {
	MatchHash bool
	MatchPath bool
	PathDepth int
}

func compare(src SourceFiles, dst DestinationFiles, opts CompareOptions) Output {
	result := Output{
		Matches: MappedFiles{},
		Stray:   MappedFiles{},
	}

	matchedSrc := make(map[string]bool)
	matchedDst := make(map[string]bool)

	if opts.MatchHash {
		for hash, s := range src {
			if d, ok := dst[hash]; ok && !matchedDst[hash] {
				sc, dc := s, d

				fmt.Printf("Found match (%s): %s (%s) == %s \n", hash, s.Name, s.Path, d.Path)

				result.Matches[hash] = MappedFile{Src: &sc, Dst: &dc}
				matchedSrc[hash] = true
				matchedDst[hash] = true
			}
		}
	}

	if opts.MatchPath && opts.PathDepth > 0 {
		dstByTail := make(map[string][]string)

		for hash, d := range dst {
			if matchedDst[hash] {
				continue
			}

			tail := pathTail(d.Path, opts.PathDepth)
			dstByTail[tail] = append(dstByTail[tail], hash)
		}

		for hash, s := range src {
			if matchedSrc[hash] {
				continue
			}

			tail := pathTail(s.Name, opts.PathDepth)
			if candidates, ok := dstByTail[tail]; ok && len(candidates) > 0 {
				dstHash := candidates[0]
				d := dst[dstHash]

				fmt.Printf("Found path match (%s): %s (%s) ~~ %s \n", tail, s.Name, s.Path, d.Path)

				result.Matches[hash] = MappedFile{Src: &s, Dst: &d}
				matchedSrc[hash] = true
				matchedDst[dstHash] = true
				dstByTail[tail] = candidates[1:]
			}
		}
	}

	for hash, s := range src {
		if !matchedSrc[hash] {
			result.Stray[hash] = MappedFile{Src: &s}
		}
	}

	for hash, d := range dst {
		if !matchedDst[hash] {
			result.Stray[hash] = MappedFile{Dst: &d}
		}
	}

	return result
}

func writeJSON(src SourceFiles, dst DestinationFiles, path string, opts CompareOptions) error {
	result := compare(src, dst, opts)

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(result, "", "\t")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o644)
}

func pathTail(path string, depth int) string {
	if depth <= 0 {
		return ""
	}

	path = filepath.ToSlash(path)

	parts := strings.Split(path, "/")
	if len(parts) <= depth {
		return path
	}

	return strings.Join(parts[len(parts)-depth:], "/")
}

func walkSource(root string, formats []string) (SourceFiles, error) {
	result := SourceFiles{}

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		if !slices.Contains(formats, filepath.Ext(path)) {
			return nil
		}

		files, err := readArchive(path)
		if err != nil {
			return err
		}

		maps.Copy(result, files)

		return nil
	})

	return result, err
}

func walkDestination(root string) (DestinationFiles, error) {
	result := DestinationFiles{}

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		fmt.Printf("Reading file: %s\n", path)

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		h := md5.New()
		h.Write(data)
		result[hex.EncodeToString(h.Sum(nil))] = DestinationFile{
			Path: path,
			Size: len(data),
		}

		return nil
	})

	return result, err
}

func readArchive(path string) (SourceFiles, error) {
	result := SourceFiles{}

	a, err := unarr.NewArchive(path)
	if err != nil {
		return nil, err
	}
	defer a.Close()

	for {
		if err := a.Entry(); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}

			return nil, err
		}

		data, err := a.ReadAll()
		if err != nil {
			return nil, err
		}

		fmt.Printf("Reading file (%s): %s\n", path, a.Name())

		h := md5.New()
		h.Write(data)
		result[hex.EncodeToString(h.Sum(nil))] = SourceFile{
			Path: path,
			Name: a.Name(),
			Size: a.Size(),
		}
	}

	return result, nil
}
