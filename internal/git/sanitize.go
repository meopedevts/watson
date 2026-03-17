package git

import (
	"fmt"
	"strings"
)

// maxDiffBytes is the safety-net size limit applied after sanitization.
// Diffs larger than this are truncated with an explanatory note.
const maxDiffBytes = 200 * 1024 // 200 KB

// skipPrefixes lists directory prefixes whose diffs should be excluded.
var skipPrefixes = []string{
	"vendor/",
	"node_modules/",
	".venv/",
	"venv/",
	"dist/",
	"build/",
}

// skipSuffixes lists file name suffixes whose diffs should be excluded.
var skipSuffixes = []string{
	"go.sum",
	"package-lock.json",
	"yarn.lock",
	"pnpm-lock.yaml",
	"Gemfile.lock",
	"Pipfile.lock",
	".pb.go",
	".min.js",
	".min.css",
	".lock", // covers composer.lock, poetry.lock, etc.
}

// SanitizeDiff removes noise from a unified git diff before sending it to
// the LLM. It excludes:
//   - Binary file markers
//   - Lock files (go.sum, *.lock, package-lock.json, …)
//   - Vendor and generated code (vendor/, node_modules/, *.pb.go, …)
//
// Returns the cleaned diff and a human-readable note listing excluded files.
// If the sanitized diff still exceeds maxDiffBytes, it is truncated and
// the note includes a truncation warning.
func SanitizeDiff(diff string) (sanitized string, note string) {
	if diff == "" {
		return diff, ""
	}

	blocks := splitDiffBlocks(diff)

	var kept []string
	var skipped []string

	for _, block := range blocks {
		filename := extractFilename(block)
		if filename == "" {
			kept = append(kept, block)
			continue
		}
		if isBinaryDiff(block) || shouldSkipFile(filename) {
			skipped = append(skipped, filename)
			continue
		}
		kept = append(kept, block)
	}

	result := strings.Join(kept, "")

	var noteParts []string
	if len(skipped) > 0 {
		noteParts = append(noteParts, fmt.Sprintf("Arquivos ignorados (lock/gerados): %s", strings.Join(skipped, ", ")))
	}

	if len(result) > maxDiffBytes {
		result = result[:maxDiffBytes]
		if idx := strings.LastIndex(result, "\n"); idx > 0 {
			result = result[:idx]
		}
		noteParts = append(noteParts, "Diff truncado em 200 KB — PR muito grande para revisão completa")
	}

	return result, strings.Join(noteParts, "; ")
}

// splitDiffBlocks splits a unified diff into per-file blocks.
// Each block starts with a "diff --git a/…" header line.
func splitDiffBlocks(diff string) []string {
	lines := strings.Split(diff, "\n")
	var blocks []string
	var current strings.Builder

	for _, line := range lines {
		if strings.HasPrefix(line, "diff --git ") && current.Len() > 0 {
			blocks = append(blocks, current.String())
			current.Reset()
		}
		current.WriteString(line)
		current.WriteByte('\n')
	}

	if current.Len() > 0 {
		blocks = append(blocks, current.String())
	}

	return blocks
}

// extractFilename parses the b-side filename from a "diff --git a/<f> b/<f>" header.
// Returns empty string if the header is not found.
func extractFilename(block string) string {
	end := strings.IndexByte(block, '\n')
	var firstLine string
	if end < 0 {
		firstLine = block
	} else {
		firstLine = block[:end]
	}

	const prefix = "diff --git "
	if !strings.HasPrefix(firstLine, prefix) {
		return ""
	}
	rest := firstLine[len(prefix):]
	// "a/<path> b/<path>" — take everything after the last " b/"
	idx := strings.LastIndex(rest, " b/")
	if idx < 0 {
		return ""
	}
	return rest[idx+3:]
}

// isBinaryDiff reports whether the block represents a binary file diff.
func isBinaryDiff(block string) bool {
	return strings.Contains(block, "Binary files ")
}

// shouldSkipFile reports whether the file path should be excluded from review.
func shouldSkipFile(filename string) bool {
	for _, prefix := range skipPrefixes {
		if strings.HasPrefix(filename, prefix) {
			return true
		}
	}
	for _, suffix := range skipSuffixes {
		if strings.HasSuffix(filename, suffix) {
			return true
		}
	}
	return false
}
