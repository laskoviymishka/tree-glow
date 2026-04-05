package main

import (
	"os"
	"path/filepath"
	"strings"
)

// searchResult holds a file match with its relative path.
type searchResult struct {
	relPath string // relative to root
	absPath string
}

// walkFiles recursively collects all files under root.
func walkFiles(root string, showHidden bool) []searchResult {
	var results []searchResult
	filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		name := d.Name()
		if !showHidden && strings.HasPrefix(name, ".") {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.IsDir() {
			rel, _ := filepath.Rel(root, path)
			results = append(results, searchResult{relPath: rel, absPath: path})
		}
		return nil
	})
	return results
}

// fuzzyMatch returns true if all characters of query appear in s in order (case-insensitive).
func fuzzyMatch(s, query string) bool {
	sr := []rune(strings.ToLower(s))
	qr := []rune(strings.ToLower(query))
	qi := 0
	for i := 0; i < len(sr) && qi < len(qr); i++ {
		if sr[i] == qr[qi] {
			qi++
		}
	}
	return qi == len(qr)
}

// fuzzyScore returns a score for ranking. Lower is better.
// Prefers: exact substring > prefix > fuzzy, shorter paths > longer.
func fuzzyScore(s, query string) int {
	sl := strings.ToLower(s)
	ql := strings.ToLower(query)

	// Exact substring match in filename
	base := strings.ToLower(filepath.Base(s))
	if strings.Contains(base, ql) {
		return len(s) - 100 // strong bonus
	}
	// Substring in full path
	if strings.Contains(sl, ql) {
		return len(s)
	}
	// Fuzzy match — score by path length (longer = worse)
	return len(s) + 100
}

// filterAndSort returns fuzzy-matched results sorted by score.
func filterAndSort(files []searchResult, query string) []searchResult {
	if query == "" {
		return nil
	}
	var matched []searchResult
	for _, f := range files {
		if fuzzyMatch(f.relPath, query) {
			matched = append(matched, f)
		}
	}
	// Sort by score
	for i := 1; i < len(matched); i++ {
		for j := i; j > 0 && fuzzyScore(matched[j].relPath, query) < fuzzyScore(matched[j-1].relPath, query); j-- {
			matched[j], matched[j-1] = matched[j-1], matched[j]
		}
	}
	// Cap results
	if len(matched) > 50 {
		matched = matched[:50]
	}
	return matched
}
