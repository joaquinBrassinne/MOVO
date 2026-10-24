// Package ciworkflow_test pins the contract of the CI workflow without running
// it. The pipeline has already failed twice for reasons no local check can see:
// a global working-directory that left make without a Makefile, and a branch
// filter whose single star never crosses the slash in phase-0/foundations.
package ciworkflow_test

import (
	"encoding/json"
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
// point, and a global working-directory default breaks them silently. A
// job-scoped default is fine: the frontend job does not use make at all.
func TestWorkflowKeepsMakeAtTheRepositoryRoot(t *testing.T) {
	t.Parallel()

	workflow := repoFile(t, ".github/workflows/ci.yml")

	if regexp.MustCompile(`(?m)^defaults:`).MatchString(workflow) {
		t.Error("workflow sets a top-level defaults:, which moves make away from the root Makefile")
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

	jobs := []string{"lint", "test", "build", "migrations", "frontend"}
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

// TestFrontendJobRunsEveryGate pins that the frontend gets the same treatment
// as the backend: no commit reaches develop with a lint error, an unformatted
// file, a type error or a build that does not compile.
func TestFrontendJobRunsEveryGate(t *testing.T) {
	t.Parallel()

	workflow := repoFile(t, ".github/workflows/ci.yml")
	block := jobBlock(t, workflow, "frontend")

	for _, want := range []string{
		"working-directory: frontend",
		"run: npm ci",
		"run: npm run lint",
		"run: npm run format:check",
		"run: npm run typecheck",
		"run: npm test",
		"run: npm run build",
	} {
		if !strings.Contains(block, want) {
			t.Errorf("frontend job does not run %q", want)
		}
	}
}

// TestFrontendJobUsesThePinnedNodeVersion pins that CI installs the Node
// version the project declares once, not a moving default that would drift
// away from development.
func TestFrontendJobUsesThePinnedNodeVersion(t *testing.T) {
	t.Parallel()

	workflow := repoFile(t, ".github/workflows/ci.yml")
	block := jobBlock(t, workflow, "frontend")

	if !strings.Contains(block, "node-version-file: frontend/.nvmrc") {
		t.Error("frontend job does not read the Node version from frontend/.nvmrc")
	}
	if strings.Contains(block, "node-version:") {
		t.Error("frontend job hardcodes a node-version instead of reading frontend/.nvmrc")
	}
	for _, want := range []string{"cache: npm", "cache-dependency-path: frontend/package-lock.json"} {
		if !strings.Contains(block, want) {
			t.Errorf("frontend job does not configure %q", want)
		}
	}

	nvmrc := strings.TrimSpace(repoFile(t, "frontend/.nvmrc"))
	if !regexp.MustCompile(`^\d+\.\d+\.\d+$`).MatchString(nvmrc) {
		t.Errorf("frontend/.nvmrc is %q, want an exact version such as 24.22.0", nvmrc)
	}
}

// TestFrontendPinsExactDependencyVersions pins that every frontend dependency
// is an exact version. A caret range in CI means the lockfile is the only pin,
// and `npm install` locally silently moves the tree away from it.
func TestFrontendPinsExactDependencyVersions(t *testing.T) {
	t.Parallel()

	var manifest struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if err := json.Unmarshal([]byte(repoFile(t, "frontend/package.json")), &manifest); err != nil {
		t.Fatalf("parse frontend/package.json: %v", err)
	}

	exact := regexp.MustCompile(`^\d+\.\d+\.\d+$`)
	pinned := 0
	for _, deps := range []map[string]string{manifest.Dependencies, manifest.DevDependencies} {
		for name, version := range deps {
			pinned++
			if !exact.MatchString(version) {
				t.Errorf("dependency %q is %q, want an exact version without a range", name, version)
			}
		}
	}
	if pinned == 0 {
		t.Fatal("frontend/package.json declares no dependencies")
	}

	// `npm ci` in CI depends on a lockfile that matches the manifest.
	if lock := repoFile(t, "frontend/package-lock.json"); !strings.Contains(lock, `"lockfileVersion"`) {
		t.Error("frontend/package-lock.json is not an npm lockfile")
	}
}
