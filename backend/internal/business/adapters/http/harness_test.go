package businesshttp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/joaquinBrassinne/MOVO/internal/business"
	businesshttp "github.com/joaquinBrassinne/MOVO/internal/business/adapters/http"
	"github.com/joaquinBrassinne/MOVO/internal/business/adapters/postgres"
	"github.com/joaquinBrassinne/MOVO/internal/business/app/commands"
	"github.com/joaquinBrassinne/MOVO/internal/business/app/queries"
	"github.com/joaquinBrassinne/MOVO/internal/platform/dbtest"
	"github.com/joaquinBrassinne/MOVO/internal/platform/httpx"
	"github.com/joaquinBrassinne/MOVO/internal/platform/storage"
)

// tokens maps a bearer token to its caller, standing in for the identity module.
type tokens map[string]httpx.Principal

func (t tokens) Verify(raw string) (httpx.Principal, error) {
	p, ok := t[raw]
	if !ok {
		return httpx.Principal{}, errors.New("unknown token")
	}
	return p, nil
}

// planLimits gives every business the same limits; 0 means unlimited.
type planLimits struct{ branches, services int }

func (l planLimits) CanAddBranch(current int) bool { return l.branches == 0 || current < l.branches }

func (l planLimits) CanActivateService(current int) bool {
	return l.services == 0 || current < l.services
}

type fixedPlan struct{ limits planLimits }

func (f fixedPlan) For(context.Context, int64) (business.Entitlements, error) { return f.limits, nil }

// server is the whole stack over a real database: handlers, use cases and
// Postgres adapters, with the object storage faked.
type server struct {
	t       *testing.T
	url     string
	pool    *pgxpool.Pool
	tokens  tokens
	storage *storage.Memory
}

func newServer(t *testing.T, limits planLimits) *server {
	t.Helper()

	pool := dbtest.New(t)
	businesses := postgres.NewBusinesses(pool)
	s := &server{t: t, pool: pool, tokens: tokens{}, storage: storage.NewMemory()}

	branches := postgres.NewBranches(pool)
	services := postgres.NewServices(pool)
	plan := fixedPlan{limits: limits}

	handlers := businesshttp.New(businesshttp.Deps{
		Verifier:       s.tokens,
		MyBusiness:     queries.NewGetMyBusiness(businesses, branches),
		CreateBusiness: commands.NewCreateBusiness(businesses),
		UpdateBusiness: commands.NewUpdateBusiness(businesses),
		SubmitBusiness: commands.NewSubmitBusiness(businesses),
		PresignImage:   commands.NewPresignImageUpload(businesses, s.storage, nil),
		AddBranch:      commands.NewAddBranch(businesses, branches, plan),
		EditBranch:     commands.NewEditBranch(businesses, branches),
		RemoveBranch:   commands.NewRemoveBranch(businesses, branches),
		SetBranchHours: commands.NewSetBranchHours(businesses, branches),

		ListServices:       queries.NewListMyServices(businesses, services),
		AddService:         commands.NewAddService(businesses, branches, services, plan),
		EditService:        commands.NewEditService(businesses, branches, services),
		RemoveService:      commands.NewRemoveService(businesses, services),
		SetServiceActive:   commands.NewSetServiceActive(businesses, services, plan),
		SetServiceSchedule: commands.NewSetServiceSchedule(businesses, services),
	})

	ts := httptest.NewServer(httpx.NewRouter(nil, nil, handlers.Mount))
	t.Cleanup(ts.Close)
	s.url = ts.URL + "/api/v1"
	return s
}

// user creates a user of the given role and returns its id and bearer token.
func (s *server) user(email, role string) (int64, string) {
	s.t.Helper()

	var id int64
	err := s.pool.QueryRow(s.t.Context(),
		`INSERT INTO users (email, password_hash, role) VALUES ($1, 'hash', $2) RETURNING id`, email, role).Scan(&id)
	if err != nil {
		s.t.Fatalf("create user %s: %v", email, err)
	}
	token := "token-" + email
	s.tokens[token] = httpx.Principal{UserID: id, Role: role}
	return id, token
}

func (s *server) categoryID(slug string) int64 {
	s.t.Helper()

	var id int64
	if err := s.pool.QueryRow(s.t.Context(), `SELECT id FROM categories WHERE slug = $1`, slug).Scan(&id); err != nil {
		s.t.Fatalf("category %s: %v", slug, err)
	}
	return id
}

// reply is a decoded response.
type reply struct {
	status int
	header http.Header
	body   map[string]any
}

// do sends a request with an optional bearer token and JSON body.
func (s *server) do(method, path, token string, body any) reply {
	s.t.Helper()

	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			s.t.Fatalf("encode body: %v", err)
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(s.t.Context(), method, s.url+path, reader)
	if err != nil {
		s.t.Fatalf("build request: %v", err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		s.t.Fatalf("%s %s: %v", method, path, err)
	}
	defer func() { _ = res.Body.Close() }()

	out := reply{status: res.StatusCode, header: res.Header}
	raw, _ := io.ReadAll(res.Body)
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &out.body); err != nil {
			s.t.Fatalf("%s %s: response is not a JSON object: %s", method, path, raw)
		}
	}
	return out
}

// errorCode returns the stable code of the error envelope.
func (r reply) errorCode() string {
	e, _ := r.body["error"].(map[string]any)
	code, _ := e["code"].(string)
	return code
}

// errorField returns details.field of the error envelope.
func (r reply) errorField() string {
	e, _ := r.body["error"].(map[string]any)
	d, _ := e["details"].(map[string]any)
	field, _ := d["field"].(string)
	return field
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }
