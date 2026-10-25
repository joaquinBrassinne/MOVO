package businesshttp

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/joaquinBrassinne/MOVO/internal/business"
	"github.com/joaquinBrassinne/MOVO/internal/business/app/commands"
	"github.com/joaquinBrassinne/MOVO/internal/business/app/queries"
	"github.com/joaquinBrassinne/MOVO/internal/business/domain"
	"github.com/joaquinBrassinne/MOVO/internal/platform/httpx"
)

type adminActionRequest struct {
	Action string `json:"action"`
}

// adminBusinessJSON is a business as the super admin list shows it.
type adminBusinessJSON struct {
	businessJSON
	OwnerEmail   string `json:"owner_email"`
	CategoryName string `json:"category_name"`
	BranchCount  int    `json:"branch_count"`
}

// mountAdmin registers the super admin routes under /admin. Only a super_admin
// token gets past the group middleware; the use cases do not check the role.
func (h *Handlers) mountAdmin(r chi.Router) {
	r.Route("/admin/businesses", func(admin chi.Router) {
		admin.Use(httpx.Authenticate(h.Verifier), httpx.RequireRole(roleSuperAdmin), noStore)

		admin.Get("/", h.handleAdminList)
		admin.Get("/{id}", h.handleAdminDetail)
		admin.Post("/{id}/actions", h.handleAdminAction)
	})
}

func (h *Handlers) handleAdminList(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	limit, okLimit := intParam(query.Get("limit"))
	offset, okOffset := intParam(query.Get("offset"))
	category, okCategory := intParam(query.Get("category_id"))
	if !okLimit || !okOffset || !okCategory {
		h.fail(w, r, queries.ErrAdminFilterInvalid)
		return
	}

	page, err := h.ListBusinesses.Handle(r.Context(), business.AdminFilter{
		Status: domain.Status(query.Get("status")), Query: query.Get("q"),
		CategoryID: int64(category), Limit: limit, Offset: offset,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}

	rows := make([]adminBusinessJSON, 0, len(page.Rows))
	for _, row := range page.Rows {
		rows = append(rows, adminBusinessJSON{
			businessJSON: toBusinessJSON(row.Business), OwnerEmail: row.OwnerEmail,
			CategoryName: row.CategoryName, BranchCount: row.BranchCount,
		})
	}
	counts := make(map[string]int, len(page.Counts))
	for status, n := range page.Counts {
		counts[string(status)] = n
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"businesses": rows, "total": page.Total, "counts": counts})
}

func (h *Handlers) handleAdminAction(w http.ResponseWriter, r *http.Request) {
	id, ok := h.pathID(w, r, business.ErrBusinessNotFound)
	if !ok {
		return
	}
	var req adminActionRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}

	updated, err := h.AdminAction.Handle(r.Context(), commands.ApplyAdminActionInput{
		ActorID: callerID(r), BusinessID: id, Action: commands.AdminAction(req.Action),
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toBusinessJSON(updated))
}

// intParam reads an optional non-negative integer query parameter; an absent one
// is 0, which the query turns into its default.
func intParam(raw string) (int, bool) {
	if raw == "" {
		return 0, true
	}
	n, err := strconv.Atoi(raw)
	return n, err == nil
}

type auditJSON struct {
	ActorEmail string         `json:"actor_email"`
	Action     string         `json:"action"`
	At         time.Time      `json:"at"`
	Metadata   map[string]any `json:"metadata"`
}

type adminDetailJSON struct {
	Business       adminBusinessJSON `json:"business"`
	Branches       []branchJSON      `json:"branches"`
	ServicesTotal  int               `json:"services_total"`
	ServicesActive int               `json:"services_active"`
	History        []auditJSON       `json:"history"`
}

func (h *Handlers) handleAdminDetail(w http.ResponseWriter, r *http.Request) {
	id, ok := h.pathID(w, r, business.ErrBusinessNotFound)
	if !ok {
		return
	}

	detail, err := h.AdminDetail.Handle(r.Context(), id)
	if err != nil {
		h.fail(w, r, err)
		return
	}

	out := adminDetailJSON{
		Business: adminBusinessJSON{
			businessJSON: toBusinessJSON(detail.Row.Business), OwnerEmail: detail.Row.OwnerEmail,
			CategoryName: detail.Row.CategoryName, BranchCount: detail.Row.BranchCount,
		},
		Branches:      make([]branchJSON, 0, len(detail.Branches)),
		ServicesTotal: len(detail.Services),
		History:       make([]auditJSON, 0, len(detail.History)),
	}
	for _, branch := range detail.Branches {
		out.Branches = append(out.Branches, toBranchJSON(branch, nil))
	}
	for _, service := range detail.Services {
		if service.Active {
			out.ServicesActive++
		}
	}
	for _, record := range detail.History {
		out.History = append(out.History, auditJSON{ActorEmail: record.ActorEmail, Action: record.Action, At: record.At, Metadata: record.Metadata})
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}
