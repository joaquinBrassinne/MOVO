package main

import (
	"log/slog"
	"net/http"
	"regexp"
	"slices"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"go.yaml.in/yaml/v3"

	apispec "github.com/joaquinBrassinne/MOVO/api"
	"github.com/joaquinBrassinne/MOVO/internal/identity/adapters/tokens"
	"github.com/joaquinBrassinne/MOVO/internal/platform/config"
	"github.com/joaquinBrassinne/MOVO/internal/platform/dbtest"
	"github.com/joaquinBrassinne/MOVO/internal/platform/httpx"
	"github.com/joaquinBrassinne/MOVO/internal/platform/storage"
)

// apiPrefix is where every module route is mounted; the probes live at the root.
const apiPrefix = "/api/v1"

// openAPIMethods are the keys of a path item that are operations.
var openAPIMethods = []string{"get", "put", "post", "delete", "options", "head", "patch", "trace"}

// routeParam matches a chi parameter with a constraint, such as {id:[0-9]+}.
var routeParam = regexp.MustCompile(`\{([^}:]+):[^}]*\}`)

// TestOpenAPIMatchesTheRouter keeps the contract honest: every route the router serves is
// documented, every documented operation is served, and the document is internally
// consistent (unique operation ids, every reference resolves). Adding a route without
// documenting it, or documenting one that does not exist, fails here.
func TestOpenAPIMatchesTheRouter(t *testing.T) {
	t.Parallel()

	documented, doc := documentedOperations(t)
	served := servedRoutes(t)
	// A walk that finds nothing would make both checks below vacuous.
	if len(served) < 50 {
		t.Fatalf("the router serves only %d routes: %v", len(served), served)
	}
	t.Logf("%d routes served, %d operations documented", len(served), len(documented))

	for _, route := range served {
		if !slices.Contains(documented, route) {
			t.Errorf("%s is served but missing from api/openapi.yaml", route)
		}
	}
	for _, route := range documented {
		if !slices.Contains(served, route) {
			t.Errorf("%s is documented in api/openapi.yaml but the router does not serve it", route)
		}
	}

	checkOperationIDs(t, doc)
	checkReferences(t, doc)
}

// servedRoutes lists "METHOD /path" for every route of the real router, composed with the
// same modules as run() in main.go.
func servedRoutes(t *testing.T) []string {
	t.Helper()

	log := slog.New(slog.DiscardHandler)
	pool := dbtest.New(t)
	signer, err := tokens.NewSigner([]byte(strings.Repeat("s", 32)), time.Hour, nil)
	if err != nil {
		t.Fatalf("signer: %v", err)
	}
	plans := config.Plans{
		Starter: config.PlanSettings{MaxBranches: 2, MaxActiveServices: 3, Price: 19800},
		Pro:     config.PlanSettings{MaxBranches: 4, MaxActiveServices: 0, Price: 29800},
	}
	images := storage.NewMemory()

	businessMount, billingMount, err := newBusinessRoutes(log, pool, signer, plans, images, 6, unusedGeocoder{})
	if err != nil {
		t.Fatalf("newBusinessRoutes: %v", err)
	}
	router := httpx.NewRouter(log, pool,
		newIdentityRoutes(log, pool, config.Auth{RefreshTokenTTL: time.Hour}, signer),
		businessMount,
		billingMount,
		newSearchRoutes(log, pool, images),
		newBenefitsRoutes(log, pool, signer, 5),
		newReviewsRoutes(log, pool, signer),
	)

	routes, ok := router.(chi.Routes)
	if !ok {
		t.Fatalf("the router is a %T, which cannot be walked", router)
	}

	var served []string
	err = chi.Walk(routes, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		route = routeParam.ReplaceAllString(route, "{$1}")
		route = strings.TrimSuffix(route, "/*")
		if route != "/" {
			route = strings.TrimSuffix(route, "/")
		}
		served = append(served, method+" "+route)
		return nil
	})
	if err != nil {
		t.Fatalf("walk the router: %v", err)
	}
	sort.Strings(served)
	return served
}

// documentedOperations lists "METHOD /path" for every operation of the document, with
// the /api/v1 prefix unless the path item overrides the server to the root.
func documentedOperations(t *testing.T) ([]string, map[string]any) {
	t.Helper()

	var doc map[string]any
	if err := yaml.Unmarshal(apispec.Spec, &doc); err != nil {
		t.Fatalf("api/openapi.yaml is not valid YAML: %v", err)
	}
	paths, _ := doc["paths"].(map[string]any)
	if len(paths) == 0 {
		t.Fatal("api/openapi.yaml documents no paths")
	}

	var documented []string
	for path, raw := range paths {
		item, _ := raw.(map[string]any)
		prefix := apiPrefix
		if servers, _ := item["servers"].([]any); len(servers) > 0 {
			if server, _ := servers[0].(map[string]any); server["url"] == "/" {
				prefix = ""
			}
		}
		for _, method := range openAPIMethods {
			if _, ok := item[method]; ok {
				documented = append(documented, strings.ToUpper(method)+" "+prefix+path)
			}
		}
	}
	sort.Strings(documented)
	return documented, doc
}

// checkOperationIDs requires every operation to have an id and every id to be unique:
// generated clients are named after them.
func checkOperationIDs(t *testing.T, doc map[string]any) {
	t.Helper()

	seen := map[string]string{}
	paths, _ := doc["paths"].(map[string]any)
	for path, raw := range paths {
		item, _ := raw.(map[string]any)
		for _, method := range openAPIMethods {
			op, ok := item[method].(map[string]any)
			if !ok {
				continue
			}
			where := strings.ToUpper(method) + " " + path
			id, _ := op["operationId"].(string)
			if id == "" {
				t.Errorf("%s has no operationId", where)
				continue
			}
			if other, dup := seen[id]; dup {
				t.Errorf("operationId %q is used by %s and %s", id, other, where)
			}
			seen[id] = where
			if _, ok := op["responses"].(map[string]any); !ok {
				t.Errorf("%s documents no responses", where)
			}
		}
	}
}

// checkReferences requires every "#/components/..." reference to point at something
// that exists.
func checkReferences(t *testing.T, doc map[string]any) {
	t.Helper()

	var walk func(node any)
	walk = func(node any) {
		switch value := node.(type) {
		case map[string]any:
			for key, child := range value {
				if ref, ok := child.(string); ok && key == "$ref" {
					if !referenceExists(doc, ref) {
						t.Errorf("%s does not resolve", ref)
					}
					continue
				}
				walk(child)
			}
		case []any:
			for _, child := range value {
				walk(child)
			}
		}
	}
	walk(doc)
}

func referenceExists(doc map[string]any, ref string) bool {
	rest, ok := strings.CutPrefix(ref, "#/")
	if !ok {
		return false
	}
	var node any = doc
	for _, part := range strings.Split(rest, "/") {
		part = strings.NewReplacer("~1", "/", "~0", "~").Replace(part)
		mapping, ok := node.(map[string]any)
		if !ok {
			return false
		}
		if node, ok = mapping[part]; !ok {
			return false
		}
	}
	return true
}
