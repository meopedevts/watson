package git

import (
	"strings"
	"testing"
)

const sampleCodeDiff = `diff --git a/internal/auth.go b/internal/auth.go
index abc..def 100644
--- a/internal/auth.go
+++ b/internal/auth.go
@@ -1,5 +1,7 @@
 package auth

+import "fmt"
+
 func Login() {
+	fmt.Println("login")
 }
`

const goSumDiff = `diff --git a/go.sum b/go.sum
index 111..222 100644
--- a/go.sum
+++ b/go.sum
@@ -1,3 +1,4 @@
 github.com/foo/bar v1.0.0 h1:abc=
+github.com/baz/qux v2.0.0 h1:def=
`

const binaryDiff = `diff --git a/assets/logo.png b/assets/logo.png
index 333..444 100644
Binary files a/assets/logo.png and b/assets/logo.png differ
`

const vendorDiff = `diff --git a/vendor/github.com/pkg/errors/errors.go b/vendor/github.com/pkg/errors/errors.go
index 555..666 100644
--- a/vendor/github.com/pkg/errors/errors.go
+++ b/vendor/github.com/pkg/errors/errors.go
@@ -1 +1,2 @@
 package errors
+// vendor change
`

const yarnLockDiff = `diff --git a/yarn.lock b/yarn.lock
index 777..888 100644
--- a/yarn.lock
+++ b/yarn.lock
@@ -1,2 +1,3 @@
 react@18.0.0:
+lodash@4.17.21:
`

func TestSanitizeDiff_EmptyInput(t *testing.T) {
	sanitized, note := SanitizeDiff("")
	if sanitized != "" {
		t.Errorf("expected empty sanitized diff, got %q", sanitized)
	}
	if note != "" {
		t.Errorf("expected empty note, got %q", note)
	}
}

func TestSanitizeDiff_KeepsCodeFiles(t *testing.T) {
	sanitized, note := SanitizeDiff(sampleCodeDiff)

	if !strings.Contains(sanitized, "internal/auth.go") {
		t.Error("expected code diff to be kept")
	}
	if note != "" {
		t.Errorf("expected no note for clean diff, got %q", note)
	}
}

func TestSanitizeDiff_RemovesGoSum(t *testing.T) {
	diff := sampleCodeDiff + goSumDiff
	sanitized, note := SanitizeDiff(diff)

	if strings.Contains(sanitized, "go.sum") {
		t.Error("go.sum diff should be removed")
	}
	if !strings.Contains(sanitized, "internal/auth.go") {
		t.Error("code diff should be kept")
	}
	if !strings.Contains(note, "go.sum") {
		t.Errorf("note should mention go.sum, got %q", note)
	}
}

func TestSanitizeDiff_RemovesBinaryFiles(t *testing.T) {
	diff := sampleCodeDiff + binaryDiff
	sanitized, note := SanitizeDiff(diff)

	if strings.Contains(sanitized, "Binary files") {
		t.Error("binary file diff should be removed")
	}
	if !strings.Contains(sanitized, "internal/auth.go") {
		t.Error("code diff should be kept")
	}
	if !strings.Contains(note, "assets/logo.png") {
		t.Errorf("note should mention excluded binary file, got %q", note)
	}
}

func TestSanitizeDiff_RemovesVendorDir(t *testing.T) {
	diff := sampleCodeDiff + vendorDiff
	sanitized, note := SanitizeDiff(diff)

	if strings.Contains(sanitized, "vendor/") {
		t.Error("vendor diff should be removed")
	}
	if !strings.Contains(note, "vendor/") {
		t.Errorf("note should mention vendor file, got %q", note)
	}
}

func TestSanitizeDiff_RemovesYarnLock(t *testing.T) {
	diff := sampleCodeDiff + yarnLockDiff
	sanitized, note := SanitizeDiff(diff)

	if strings.Contains(sanitized, "yarn.lock") {
		t.Error("yarn.lock diff should be removed")
	}
	if !strings.Contains(note, "yarn.lock") {
		t.Errorf("note should mention yarn.lock, got %q", note)
	}
}

func TestSanitizeDiff_MultipleExcluded(t *testing.T) {
	diff := sampleCodeDiff + goSumDiff + yarnLockDiff + binaryDiff
	sanitized, note := SanitizeDiff(diff)

	if !strings.Contains(sanitized, "internal/auth.go") {
		t.Error("code diff should be kept")
	}
	for _, excluded := range []string{"go.sum", "yarn.lock", "assets/logo.png"} {
		if !strings.Contains(note, excluded) {
			t.Errorf("note should mention %q, got %q", excluded, note)
		}
	}
}

func TestSanitizeDiff_OnlyExcluded_ReturnEmptyDiff(t *testing.T) {
	diff := goSumDiff + yarnLockDiff
	sanitized, note := SanitizeDiff(diff)

	if strings.Contains(sanitized, "go.sum") || strings.Contains(sanitized, "yarn.lock") {
		t.Error("all excluded files should be removed from diff")
	}
	if note == "" {
		t.Error("expected a note when files are excluded")
	}
}

func TestSanitizeDiff_Truncation(t *testing.T) {
	// Build a diff that exceeds maxDiffBytes.
	bigBlock := sampleCodeDiff + strings.Repeat("+line\n", maxDiffBytes/5)
	sanitized, note := SanitizeDiff(bigBlock)

	if len(sanitized) >= maxDiffBytes {
		t.Errorf("expected sanitized diff to be truncated below %d bytes, got %d", maxDiffBytes, len(sanitized))
	}
	if !strings.Contains(note, "truncado") {
		t.Errorf("expected truncation note, got %q", note)
	}
}

func TestExtractFilename(t *testing.T) {
	cases := []struct {
		block string
		want  string
	}{
		{"diff --git a/foo.go b/foo.go\nsome content", "foo.go"},
		{"diff --git a/vendor/pkg/file.go b/vendor/pkg/file.go\n", "vendor/pkg/file.go"},
		{"not a diff header\n", ""},
		{"diff --git a/path/with spaces/file.go b/path/with spaces/file.go\n", "path/with spaces/file.go"},
	}

	for _, tc := range cases {
		got := extractFilename(tc.block)
		if got != tc.want {
			t.Errorf("extractFilename(%q) = %q, want %q", tc.block, got, tc.want)
		}
	}
}

func TestShouldSkipFile(t *testing.T) {
	skipped := []string{
		"go.sum",
		"package-lock.json",
		"yarn.lock",
		"pnpm-lock.yaml",
		"Gemfile.lock",
		"Pipfile.lock",
		"composer.lock",
		"api.pb.go",
		"bundle.min.js",
		"styles.min.css",
		"vendor/github.com/foo/bar.go",
		"node_modules/react/index.js",
		"dist/main.js",
		"build/output.js",
	}

	for _, f := range skipped {
		if !shouldSkipFile(f) {
			t.Errorf("shouldSkipFile(%q) = false, want true", f)
		}
	}

	kept := []string{
		"internal/auth.go",
		"cmd/main.go",
		"README.md",
		"Taskfile.yml",
	}

	for _, f := range kept {
		if shouldSkipFile(f) {
			t.Errorf("shouldSkipFile(%q) = true, want false", f)
		}
	}
}
