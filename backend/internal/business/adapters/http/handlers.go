// Package businesshttp exposes the business use cases: the owner endpoints under
// /me/business and the super admin endpoints under /admin.
package businesshttp

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/joaquinBrassinne/MOVO/internal/business/app/commands"
	"github.com/joaquinBrassinne/MOVO/internal/business/app/queries"
	"github.com/joaquinBrassinne/MOVO/internal/platform/httpx"
)

// Roles allowed on each group of routes. They are the plain strings stored in the
// token, so this package needs no knowledge of the identity module.
const (
	roleOwner      = "business_admin"
	roleSuperAdmin = "super_admin"
)

// Deps are the use cases and the token verifier the handlers need.
type Deps struct {
	Verifier httpx.TokenVerifier
	Log      *slog.Logger

	MyBusiness     *queries.GetMyBusiness
	CreateBusiness *commands.CreateBusiness
	UpdateBusiness *commands.UpdateBusiness
	SubmitBusiness *commands.SubmitBusiness
	PresignImage   *commands.PresignImageUpload

	Geocode         *commands.GeocodeAddress
	RegisterImage   *commands.RegisterImage
	ReorderImages   *commands.ReorderImages
	RemoveImage     *commands.RemoveImage
	AddBranch       *commands.AddBranch
	EditBranch      *commands.EditBranch
	RemoveBranch    *commands.RemoveBranch
	SetBranchHours  *commands.SetBranchHours
	SetBranchActive *commands.SetBranchActive

	ListServices       *queries.ListMyServices
	AddService         *commands.AddService
	EditService        *commands.EditService
	RemoveService      *commands.RemoveService
	SetServiceActive   *commands.SetServiceActive
	SetServiceSchedule *commands.SetServiceSchedule

	Categories *queries.ListCategories

	ListBusinesses *queries.ListBusinesses
	AdminAction    *commands.ApplyAdminAction
	AdminDetail    *queries.GetBusinessDetail
}

// Handlers serves the business endpoints.
type Handlers struct {
	Deps
	log *slog.Logger
}

// New wires the handlers. A nil logger discards logs.
func New(deps Deps) *Handlers {
	log := deps.Log
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}
	return &Handlers{Deps: deps, log: log}
}

// Mount registers the routes. Authentication and the role are enforced here, per
// group, so no handler runs for the wrong kind of caller.
func (h *Handlers) Mount(r chi.Router) {
	r.Route("/me/business", func(me chi.Router) {
		me.Use(httpx.Authenticate(h.Verifier), httpx.RequireRole(roleOwner), noStore)

		me.Get("/", h.handleGetMine)
		me.Post("/", h.handleCreate)
		me.Put("/", h.handleUpdate)
		me.Post("/submit", h.handleSubmit)
		me.Post("/images/presign", h.handlePresign)
		me.Post("/geocode", h.handleGeocode)
		h.mountPhotos(me)
		h.mountBranches(me)
		h.mountServices(me)
	})
	h.mountCategories(r)
	h.mountAdmin(r)
}

type businessRequest struct {
	CategoryID  int64  `json:"category_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	WhatsApp    string `json:"whatsapp"`
}

type submitRequest struct {
	PaymentReference string `json:"payment_reference"`
}

type presignRequest struct {
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
}

type presignResponse struct {
	ObjectKey string `json:"object_key"`
	Upload    struct {
		URL     string            `json:"url"`
		Method  string            `json:"method"`
		Headers map[string]string `json:"headers"`
	} `json:"upload"`
}

// callerID is the authenticated caller. RequireRole already guaranteed it exists;
// the id always comes from the token, never from the body or the URL.
func callerID(r *http.Request) int64 {
	principal, _ := httpx.PrincipalFrom(r.Context())
	return principal.UserID
}

func (h *Handlers) handleGetMine(w http.ResponseWriter, r *http.Request) {
	view, err := h.MyBusiness.Handle(r.Context(), callerID(r))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toMyBusinessJSON(view))
}

func (h *Handlers) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req businessRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}

	created, err := h.CreateBusiness.Handle(r.Context(), commands.CreateBusinessInput{
		OwnerID: callerID(r), CategoryID: req.CategoryID, Name: req.Name, Description: req.Description, WhatsApp: req.WhatsApp,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, toBusinessJSON(created))
}

func (h *Handlers) handleUpdate(w http.ResponseWriter, r *http.Request) {
	var req businessRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}

	updated, err := h.UpdateBusiness.Handle(r.Context(), commands.UpdateBusinessInput{
		OwnerID: callerID(r), CategoryID: req.CategoryID, Name: req.Name, Description: req.Description, WhatsApp: req.WhatsApp,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toBusinessJSON(updated))
}

func (h *Handlers) handleSubmit(w http.ResponseWriter, r *http.Request) {
	var req submitRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}

	submitted, err := h.SubmitBusiness.Handle(r.Context(), commands.SubmitBusinessInput{OwnerID: callerID(r), PaymentReference: req.PaymentReference})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toBusinessJSON(submitted))
}

func (h *Handlers) handlePresign(w http.ResponseWriter, r *http.Request) {
	var req presignRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}

	presigned, err := h.PresignImage.Handle(r.Context(), commands.PresignImageUploadInput{OwnerID: callerID(r), ContentType: req.ContentType, Size: req.Size})
	if err != nil {
		h.fail(w, r, err)
		return
	}

	var resp presignResponse
	resp.ObjectKey = presigned.ObjectKey
	resp.Upload.URL, resp.Upload.Method, resp.Upload.Headers = presigned.Upload.URL, presigned.Upload.Method, presigned.Upload.Headers
	httpx.WriteJSON(w, http.StatusOK, resp)
}

// noStore keeps owner data out of shared caches.
func noStore(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}
