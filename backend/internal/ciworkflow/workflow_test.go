// Package ciworkflow_test pins the contract of the CI workflow without running
// it. The pipeline has already failed twice for reasons no local check can see:
// a global working-directory that left make without a Makefile, and a branch
// filter whose single star never crosses the slash in phase-0/foundations.
package ciworkflow_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// repoFile reads a file from the repository root, three levels above this
// package: backend/internal/ciworkflow.
func repoFile(t *testing.T, name string) string {
	t.Helper()

	path := filepath.Join("..", "..", "..", name)
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	return string(body)
}

// jobBlock returns the body of one job. A job body is indented four or more
// spaces, so the next job header is the first line at exactly two spaces.
func jobBlock(t *testing.T, workflow, name string) string {
	t.Helper()

	block := regexp.MustCompile(`(?ms)^  ` + regexp.QuoteMeta(name) + `:\n(.*?)(?:\n  [a-z-]+:\n|\z)`).
		FindStringSubmatch(workflow)
	if block == nil {
		t.Fatalf("workflow has no %q job", name)
	}

	return block[1]
}

// TestWorkflowTriggersOnWorkingBranches pins that a branch with a slash in its
// name actually fires: `phase-*` silently matched nothing.
func TestWorkflowTriggersOnWorkingBranches(t *testing.T) {
	t.Parallel()

	workflow := repoFile(t, ".github/workflows/ci.yml")

	for _, want := range []string{`- "phase-**"`, "- develop", "- main", "pull_request:"} {
		if !strings.Contains(workflow, want) {
			t.Errorf("workflow does not trigger on %q", want)
		}
	}
	for _, rejected := range []string{`- "phase-*"`} {
		if strings.Contains(workflow, rejected) {
			t.Errorf("workflow uses %q, which never matches a branch with a slash", rejected)
		}
	}
}

// TestWorkflowKeepsMakeAtTheRepositoryRoot pins that no run step is moved into
// backend/: the make targets of the root Makefile are the pipeline's entry
// point, and a working-directory default breaks them silently.
func TestWorkflowKeepsMakeAtTheRepositoryRoot(t *testing.T) {
	t.Parallel()

	workflow := repoFile(t, ".github/workflows/ci.yml")

	if strings.Contains(workflow, "defaults:") {
		t.Error("workflow sets defaults:, which moves make away from the root Makefile")
	}
	for _, job := range []string{"test", "build", "migrations"} {
		if block := jobBlock(t, workflow, job); !strings.Contains(block, "run: make ") {
			t.Errorf("job %q does not invoke a Makefile target", job)
		}
	}
}

// TestWorkflowPinsEveryAction pins that an action cannot change under the
// pipeline: a tag or a moving major would.
func TestWorkflowPinsEveryAction(t *testing.T) {
	t.Parallel()

	workflow := repoFile(t, ".github/workflows/ci.yml")
	uses := regexp.MustCompile(`(?m)^\s*-?\s*uses: (\S+)$`)
	semver := regexp.MustCompile(`@v\d+\.\d+\.\d+$`)

	matches := uses.FindAllStringSubmatch(workflow, -1)
	if len(matches) == 0 {
		t.Fatal("workflow declares no actions")
	}

	for _, match := range matches {
		if !semver.MatchString(match[1]) {
			t.Errorf("action %q is not pinned to an exact version", match[1])
		}
	}
}

// TestWorkflowBoundsEveryJob pins that a hung job dies by itself instead of
// burning the default six hours.
func TestWorkflowBoundsEveryJob(t *testing.T) {
	t.Parallel()

	workflow := repoFile(t, ".github/workflows/ci.yml")

	jobs := []string{"lint", "test", "build", "migrations"}
	for _, job := range jobs {
		if block := jobBlock(t, workflow, job); !strings.Contains(block, "timeout-minutes:") {
			t.Errorf("job %q has no timeout-minutes", job)
		}
	}
}

// TestMigrationsJobProvesTheState pins that the job verifies what the database
// looks like, not only that the commands exited 0.
func TestMigrationsJobProvesTheState(t *testing.T) {
	t.Parallel()

	workflow := repoFile(t, ".github/workflows/ci.yml")
	block := jobBlock(t, workflow, "migrations")

	for _, want := range []string{
		"run: make migrate-up",
		"run: make migrate-status",
		"run: make migrate-down",
		"assert postgis is installed",
		"assert postgis is gone",
		"extname = 'postgis'",
		"ON_ERROR_STOP=1",
	} {
		if !strings.Contains(block, want) {
			t.Errorf("migrations job does not contain %q", want)
		}
	}
	if got := strings.Count(block, "run: make migrate-up"); got < 2 {
		t.Errorf("migrations job applies the migrations %d time(s), want at least 2 for a roundtrip", got)
	}
}

// TestDevAndCIUseTheSameDatabaseImage pins that CI validates the migrations
// against the server development runs, not a different PostgreSQL.
func TestDevAndCIUseTheSameDatabaseImage(t *testing.T) {
	t.Parallel()

	compose := repoFile(t, "deploy/docker-compose.yml")
	workflow := repoFile(t, ".github/workflows/ci.yml")

	image := regexp.MustCompile(`image: (postgis/postgis:\S+)`).FindStringSubmatch(compose)
	if image == nil {
		t.Fatal("deploy/docker-compose.yml declares no postgis image")
	}
	if !strings.Contains(workflow, "image: "+image[1]) {
		t.Errorf("CI does not use the compose image %q", image[1])
	}
}

// TestLintVersionMatchesTheMakefile pins that the pipeline and `make lint` run
// the same linter, so a local green is worth something.
func TestLintVersionMatchesTheMakefile(t *testing.T) {
	t.Parallel()

	makefile := repoFile(t, "Makefile")
	workflow := repoFile(t, ".github/workflows/ci.yml")

	version := regexp.MustCompile(`GOLANGCI_LINT_VERSION := (\S+)`).FindStringSubmatch(makefile)
	if version == nil {
		t.Fatal("Makefile declares no GOLANGCI_LINT_VERSION")
	}
	if !strings.Contains(workflow, "version: "+version[1]) {
		t.Errorf("CI does not run golangci-lint %s, the version `make lint` expects", version[1])
	}
}
