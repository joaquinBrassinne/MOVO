// Package benefitshttp exposes the benefit use cases: the owner endpoints under
// /me/business/benefits and the public coupon endpoints.
package benefitshttp

import (
	"encoding/json"
	"errors"
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/app/commands"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/app/queries"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/domain"
	"github.com/joaquinBrassinne/MOVO/internal/platform/clientip"
	"github.com/joaquinBrassinne/MOVO/internal/platform/httpx"
)

// roleOwner is the plain string stored in the token, so this package needs no
// knowledge of the identity module.
const (
	roleOwner      = "business_admin"
	roleSuperAdmin = "super_admin"
)

// Codes specific to the benefit endpoints.
const (
	CodeValidationFailed = "validation_failed"
	CodeBusinessNotFound = "business_not_found"
	CodeBenefitNotFound  = "benefit_not_found"
	CodeBusinessDeleted  = "business_deleted"
	CodePendingCoupon    = "pending_coupon"
	CodeQuotaReached     = "quota_reached"
	CodeRateLimited      = "rate_limited"
	CodeCouponNotFound   = "coupon_not_found"
	CodeCouponRedeemed   = "coupon_redeemed"
	CodeCouponExpired    = "coupon_expired"
)

// Deps are the use cases and the token verifier the handlers need.
type Deps struct {
	Verifier httpx.TokenVerifier
	Log      *slog.Logger

	List      *queries.ListMyBenefits
	Add       *commands.AddBenefit
	Edit      *commands.EditBenefit
	SetActive *commands.SetBenefitActive

	// Generate serves the public coupon endpoint; without it the route is not mounted.
	Generate *commands.GenerateCode
	// Redeem serves the owner's redemption; without it the route is not mounted.
	Redeem *commands.RedeemCode
	// Preview serves the look at a coupon before redeeming it; without it the route is
	// not mounted.
	Preview *queries.PreviewCode
	// Activity serves the owner's recent activity; without it the route is not mounted.
	Activity *queries.RecentActivity
	// View serves the customer's look at their coupon by access token; without it the
	// route is not mounted.
	View *queries.ViewMyCoupon
	// Overview serves the counts of the panel home; without it the route is not mounted.
	Overview *queries.GetOverview
	// Quota and SetQuota serve the monthly coupon cap of the business; each route is
	// mounted only when its use case is set.
	Quota    *queries.GetCouponQuota
	SetQuota *commands.SetCouponQuota
	// Pause serves the super admin's pause of a benefit; without it the route is not mounted.
	Pause *commands.PauseBenefit
}

// Handlers serves the benefit endpoints.
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

// Mount registers the routes. They live under /me/business, which the business module
// also serves; chi picks these static paths over its catch-all. Authentication and the
// role are enforced here, so no handler runs for the wrong kind of caller.
func (h *Handlers) Mount(r chi.Router) {
	owner := r.With(httpx.Authenticate(h.Verifier), httpx.RequireRole(roleOwner), noStore)

	owner.Get("/me/business/benefits", h.handleList)
	owner.Post("/me/business/benefits", h.handleAdd)
	owner.Put("/me/business/benefits/{id}", h.handleEdit)
	owner.Put("/me/business/benefits/{id}/active", h.handleSetActive)

	// The business comes from the session, never from the body.
	if h.Redeem != nil {
		owner.Post("/me/business/codes/redeem", h.handleRedeem)
	}
	if h.Preview != nil {
		owner.Get("/me/business/codes/{code}", h.handlePreview)
	}

	// Public: the customer has no account. The origin limit lives in the use case.
	if h.Generate != nil {
		r.With(noStore).Post("/benefits/{id}/codes", h.handleGenerate)
	}
	if h.Overview != nil {
		owner.Get("/me/business/overview", h.handleOverview)
	}
	if h.Quota != nil {
		owner.Get("/me/business/coupon-quota", h.handleQuota)
	}
	if h.SetQuota != nil {
		owner.Put("/me/business/coupon-quota", h.handleSetQuota)
	}
	if h.Activity != nil {
		owner.Get("/me/business/activity", h.handleActivity)
	}
	// The token in the path is the credential, so the answer must never be cached.
	if h.View != nil {
		r.With(noStore).Get("/access/{token}", h.handleView)
	}
	// Only a super_admin token gets past this group; the owner has no way to reach it.
	if h.Pause != nil {
		admin := r.With(httpx.Authenticate(h.Verifier), httpx.RequireRole(roleSuperAdmin), noStore)
		admin.Put("/admin/benefits/{id}/paused", h.handlePause)
	}
}

type benefitRequest struct {
	Type       domain.Type `json:"type"`
	Percent    int         `json:"percent"`
	AmountARS  int         `json:"amount_ars"`
	Title      string      `json:"title"`
	Conditions string      `json:"conditions"`
}

func (b benefitRequest) input() domain.BenefitInput {
	return domain.BenefitInput{
		Type: b.Type, Percent: b.Percent, AmountARS: b.AmountARS, Title: b.Title,
		Conditions: b.Conditions,
	}
}

type activeRequest struct {
	Active bool `json:"active"`
}

type pausedRequest struct {
	Paused bool `json:"paused"`
}

// claimedBenefitJSON is a benefit as a customer reads it: what it is worth and under
// which conditions, nothing about the business's limits. The fields of the other types
// are null; short_label is what cards and pills show.
type claimedBenefitJSON struct {
	ID         int64       `json:"id"`
	Type       domain.Type `json:"type"`
	Percent    *int        `json:"percent"`
	AmountARS  *int        `json:"amount_ars"`
	Title      *string     `json:"title"`
	Conditions *string     `json:"conditions"`
	ShortLabel string      `json:"short_label"`
}

func toClaimedBenefitJSON(b domain.Benefit) claimedBenefitJSON {
	out := claimedBenefitJSON{ID: b.ID, Type: b.Type, ShortLabel: b.ShortLabel()}
	switch b.Type {
	case domain.TypePercent:
		out.Percent = &b.Percent
	case domain.TypeAmount:
		out.AmountARS = &b.AmountARS
	case domain.TypeCustom:
		out.Title = &b.Title
	}
	if b.Conditions != "" {
		out.Conditions = &b.Conditions
	}
	return out
}

// benefitJSON is a benefit as the panel reads it: what a customer sees, plus the owner's
// switch, whether the super admin paused it, and the monthly limit.
type benefitJSON struct {
	claimedBenefitJSON
	Active      bool `json:"active"`
	AdminPaused bool `json:"admin_paused"`
}

func toBenefitJSON(b domain.Benefit) benefitJSON {
	return benefitJSON{claimedBenefitJSON: toClaimedBenefitJSON(b), Active: b.Active, AdminPaused: b.AdminPaused}
}

// benefitListItemJSON is a benefit of the list, with what it did in the current month
// (Argentina time): coupons generated in it and coupons redeemed in it.
type benefitListItemJSON struct {
	benefitJSON
	GeneratedThisMonth int `json:"generated_this_month"`
	RedeemedThisMonth  int `json:"redeemed_this_month"`
}

func (h *Handlers) handleList(w http.ResponseWriter, r *http.Request) {
	list, err := h.List.Handle(r.Context(), callerID(r))
	if err != nil {
		h.fail(w, r, err)
		return
	}

	out := make([]benefitListItemJSON, 0, len(list))
	for _, b := range list {
		out = append(out, benefitListItemJSON{benefitJSON: toBenefitJSON(b.Benefit), GeneratedThisMonth: b.Generated, RedeemedThisMonth: b.Redeemed})
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"benefits": out})
}

func (h *Handlers) handleAdd(w http.ResponseWriter, r *http.Request) {
	var req benefitRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}

	created, err := h.Add.Handle(r.Context(), commands.AddBenefitInput{OwnerID: callerID(r), Benefit: req.input()})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, toBenefitJSON(created))
}

func (h *Handlers) handleEdit(w http.ResponseWriter, r *http.Request) {
	id, ok := h.pathID(w, r)
	if !ok {
		return
	}
	var req benefitRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}

	updated, err := h.Edit.Handle(r.Context(), commands.EditBenefitInput{OwnerID: callerID(r), BenefitID: id, Benefit: req.input()})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toBenefitJSON(updated))
}

func (h *Handlers) handleSetActive(w http.ResponseWriter, r *http.Request) {
	id, ok := h.pathID(w, r)
	if !ok {
		return
	}
	var req activeRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}

	updated, err := h.SetActive.Handle(r.Context(), commands.SetBenefitActiveInput{OwnerID: callerID(r), BenefitID: id, Active: req.Active})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toBenefitJSON(updated))
}

type generateRequest struct {
	Name  string `json:"name"`
	Phone string `json:"phone"`
}

// generatedJSON is the coupon a customer takes away. The access token appears only
// here: the server keeps its hash, so a lost link cannot be recovered.
type generatedJSON struct {
	Code        string             `json:"code"`
	AccessToken string             `json:"access_token"`
	ExpiresAt   time.Time          `json:"expires_at"`
	Benefit     claimedBenefitJSON `json:"benefit"`
}

// handlePause is the super admin's pause of any benefit. The actor comes from the token.
func (h *Handlers) handlePause(w http.ResponseWriter, r *http.Request) {
	id, ok := h.pathID(w, r)
	if !ok {
		return
	}
	var req pausedRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}

	got, err := h.Pause.Handle(r.Context(), callerID(r), id, req.Paused)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toBenefitJSON(got))
}

type redeemRequest struct {
	Code string `json:"code"`
	// BranchID is where the customer redeemed; null or absent means the business's only
	// active branch.
	BranchID *int64 `json:"branch_id"`
}

type redeemedJSON struct {
	Code         string             `json:"code"`
	RedeemedAt   time.Time          `json:"redeemed_at"`
	CustomerName string             `json:"customer_name"`
	BranchID     int64              `json:"branch_id"`
	Benefit      claimedBenefitJSON `json:"benefit"`
}

// previewJSON is a coupon as the owner sees it before confirming. It has no phone: the
// business only needs the name to greet the customer.
type previewJSON struct {
	Code         string        `json:"code"`
	Status       domain.Status `json:"status"`
	CustomerName string        `json:"customer_name"`
	// CustomerRedeemed is how many coupons this customer already redeemed at the business.
	CustomerRedeemed int                `json:"customer_redeemed_count"`
	ExpiresAt        time.Time          `json:"expires_at"`
	RedeemedAt       *time.Time         `json:"redeemed_at"`
	Benefit          claimedBenefitJSON `json:"benefit"`
}

func (h *Handlers) handlePreview(w http.ResponseWriter, r *http.Request) {
	view, err := h.Preview.Handle(r.Context(), callerID(r), chi.URLParam(r, "code"))
	if err != nil {
		h.fail(w, r, err)
		return
	}

	out := previewJSON{
		Code: view.Code, Status: view.Status, CustomerName: view.CustomerName, CustomerRedeemed: view.CustomerRedeemed,
		ExpiresAt: view.ExpiresAt.UTC(), Benefit: toClaimedBenefitJSON(view.Benefit),
	}
	if view.RedeemedAt != nil {
		redeemedAt := view.RedeemedAt.UTC()
		out.RedeemedAt = &redeemedAt
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

// heldJSON is a coupon as its customer sees it. It has no phone and does not repeat the
// token.
type heldJSON struct {
	Code         string             `json:"code"`
	Status       domain.Status      `json:"status"`
	BusinessName string             `json:"business_name"`
	ExpiresAt    time.Time          `json:"expires_at"`
	RedeemedAt   *time.Time         `json:"redeemed_at"`
	Benefit      claimedBenefitJSON `json:"benefit"`
}

func (h *Handlers) handleView(w http.ResponseWriter, r *http.Request) {
	view, err := h.View.Handle(r.Context(), chi.URLParam(r, "token"))
	if err != nil {
		h.fail(w, r, err)
		return
	}

	out := heldJSON{
		Code: view.Code, Status: view.Status, BusinessName: view.BusinessName,
		ExpiresAt: view.ExpiresAt.UTC(), Benefit: toClaimedBenefitJSON(view.Benefit),
	}
	if view.RedeemedAt != nil {
		redeemedAt := view.RedeemedAt.UTC()
		out.RedeemedAt = &redeemedAt
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *Handlers) handleRedeem(w http.ResponseWriter, r *http.Request) {
	var req redeemRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	in := commands.RedeemCodeInput{OwnerID: callerID(r), Code: req.Code}
	if req.BranchID != nil {
		in.BranchID = *req.BranchID
	}

	got, err := h.Redeem.Handle(r.Context(), in)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, redeemedJSON{
		Code: got.Coupon.Code, RedeemedAt: got.RedeemedAt.UTC(), CustomerName: got.CustomerName,
		BranchID: got.BranchID, Benefit: toClaimedBenefitJSON(got.Benefit),
	})
}

func (h *Handlers) handleGenerate(w http.ResponseWriter, r *http.Request) {
	id, ok := h.pathID(w, r)
	if !ok {
		return
	}
	var req generateRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}

	got, err := h.Generate.Handle(r.Context(), commands.GenerateCodeInput{
		BenefitID: id, Name: req.Name, Phone: req.Phone, ClientIP: clientip.From(r),
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, generatedJSON{
		Code: got.Coupon.Code, AccessToken: got.AccessToken, ExpiresAt: got.Coupon.ExpiresAt.UTC(),
		Benefit: toClaimedBenefitJSON(got.Benefit),
	})
}

// pathID reads {id}; a malformed one answers like a benefit that does not exist.
func (h *Handlers) pathID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		h.fail(w, r, benefits.ErrBenefitNotFound)
		return 0, false
	}
	return id, true
}

// callerID is the authenticated caller. RequireRole already guaranteed it exists; the
// id always comes from the token, never from the body or the URL.
func callerID(r *http.Request) int64 {
	principal, _ := httpx.PrincipalFrom(r.Context())
	return principal.UserID
}

// noStore keeps owner data out of shared caches.
func noStore(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

// validation maps each domain validation error to the field it is about. Anything not
// listed here is not a client mistake.
var validation = map[error]struct{ field, reason string }{
	domain.ErrBenefitTypeInvalid:  {"type", "invalid"},
	domain.ErrPercentInvalid:      {"percent", "invalid"},
	domain.ErrAmountInvalid:       {"amount_ars", "invalid"},
	domain.ErrQuotaLimitInvalid:   {"limit", "invalid"},
	domain.ErrCustomerNameInvalid: {"name", "invalid"},
	domain.ErrPhoneInvalid:        {"phone", "invalid"},
	benefits.ErrBranchRequired:    {"branch_id", "required"},
	benefits.ErrBranchNotFound:    {"branch_id", "unknown"},
}

// fail maps a use case error to its HTTP answer. Anything unexpected is logged with
// the request id and reported as a generic 500: the error text may name internals and
// never reaches the client.
func (h *Handlers) fail(w http.ResponseWriter, r *http.Request, err error) {
	var limited *benefits.RateLimitedError
	if errors.As(err, &limited) {
		seconds := int(math.Ceil(limited.RetryAfter.Seconds()))
		w.Header().Set("Retry-After", strconv.Itoa(seconds))
		httpx.WriteError(w, http.StatusTooManyRequests, CodeRateLimited, "too many requests, try again later", map[string]int{"retry_after_seconds": seconds})
		return
	}

	switch {
	case errors.Is(err, benefits.ErrPendingCoupon):
		httpx.WriteError(w, http.StatusConflict, CodePendingCoupon, "you already have a pending coupon at this business", nil)
		return
	case errors.Is(err, benefits.ErrCouponNotFound):
		httpx.WriteError(w, http.StatusNotFound, CodeCouponNotFound, "coupon not found", nil)
		return
	case errors.Is(err, benefits.ErrCouponRedeemed):
		httpx.WriteError(w, http.StatusConflict, CodeCouponRedeemed, "this coupon was already redeemed", nil)
		return
	case errors.Is(err, benefits.ErrCouponExpired):
		httpx.WriteError(w, http.StatusConflict, CodeCouponExpired, "this coupon has expired", nil)
		return
	case errors.Is(err, benefits.ErrQuotaReached):
		httpx.WriteError(w, http.StatusConflict, CodeQuotaReached, "this business has no coupons left this month", nil)
		return
	case errors.Is(err, benefits.ErrBenefitNotFound):
		httpx.WriteError(w, http.StatusNotFound, CodeBenefitNotFound, "benefit not found", nil)
		return
	case errors.Is(err, benefits.ErrBusinessNotFound):
		httpx.WriteError(w, http.StatusNotFound, CodeBusinessNotFound, "you have not created a business yet", nil)
		return
	case errors.Is(err, benefits.ErrBusinessNotEditable):
		httpx.WriteError(w, http.StatusConflict, CodeBusinessDeleted, "this business was deleted and can no longer change", nil)
		return
	}
	var below *domain.QuotaBelowGeneratedError
	if errors.As(err, &below) {
		httpx.WriteError(w, http.StatusUnprocessableEntity, CodeValidationFailed, "the limit is below the coupons already generated this month",
			map[string]any{"field": "limit", "reason": "below_generated", "generated_this_month": below.Generated})
		return
	}
	var text *domain.TextError
	if errors.As(err, &text) {
		httpx.WriteError(w, http.StatusUnprocessableEntity, CodeValidationFailed, "the "+text.Field+" is not valid",
			map[string]string{"field": text.Field, "reason": string(text.Reason)})
		return
	}
	for known, field := range validation {
		if errors.Is(err, known) {
			httpx.WriteError(w, http.StatusUnprocessableEntity, CodeValidationFailed, "the "+field.field+" is not valid",
				map[string]string{"field": field.field, "reason": field.reason})
			return
		}
	}

	h.log.Error("benefits request failed", "request_id", httpx.RequestID(r.Context()), "path", r.URL.Path, "error", err)
	httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "something went wrong", nil)
}

type activityItemJSON struct {
	Type         benefits.ActivityKind `json:"type"`
	At           time.Time             `json:"at"`
	CustomerName string                `json:"customer_name"`
	Benefit      claimedBenefitJSON    `json:"benefit"`
}

type activityJSON struct {
	Items    []activityItemJSON `json:"items"`
	NewCount int                `json:"new_count"`
}

// handleActivity answers the panel's poll. since is the time of the newest event the
// panel already showed, in RFC 3339; the panel keeps it, the server keeps nothing.
func (h *Handlers) handleActivity(w http.ResponseWriter, r *http.Request) {
	var since *time.Time
	if raw := r.URL.Query().Get("since"); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			httpx.WriteError(w, http.StatusUnprocessableEntity, CodeValidationFailed, "the since is not valid",
				map[string]string{"field": "since", "reason": "invalid"})
			return
		}
		since = &parsed
	}

	got, err := h.Activity.Handle(r.Context(), callerID(r), since)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	out := activityJSON{Items: make([]activityItemJSON, 0, len(got.Events)), NewCount: got.NewCount}
	for _, e := range got.Events {
		out.Items = append(out.Items, activityItemJSON{
			Type: e.Kind, At: e.At.UTC(), CustomerName: e.CustomerName, Benefit: toClaimedBenefitJSON(e.Benefit),
		})
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

// quotaJSON is the monthly coupon quota of the business. A null limit means no limit;
// resets_on is the date the count starts again, in Argentina time.
type quotaJSON struct {
	Limit              *int   `json:"limit"`
	GeneratedThisMonth int    `json:"generated_this_month"`
	ResetsOn           string `json:"resets_on"`
}

func toQuotaJSON(v benefits.QuotaView) quotaJSON {
	return quotaJSON{Limit: v.Limit, GeneratedThisMonth: v.Generated, ResetsOn: domain.DateInArgentina(v.ResetsOn)}
}

// quotaRequest keeps the limit raw so that an absent key (an error) can be told apart
// from an explicit null (no limit).
type quotaRequest struct {
	Limit json.RawMessage `json:"limit"`
}

func (h *Handlers) handleQuota(w http.ResponseWriter, r *http.Request) {
	view, err := h.Quota.Handle(r.Context(), callerID(r))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toQuotaJSON(view))
}

func (h *Handlers) handleSetQuota(w http.ResponseWriter, r *http.Request) {
	var req quotaRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	var limit *int
	switch raw := strings.TrimSpace(string(req.Limit)); raw {
	case "":
		httpx.WriteError(w, http.StatusUnprocessableEntity, CodeValidationFailed, "the limit is required, or null for no limit",
			map[string]string{"field": "limit", "reason": "required"})
		return
	case "null":
	default:
		n, err := strconv.Atoi(raw)
		if err != nil {
			h.fail(w, r, domain.ErrQuotaLimitInvalid)
			return
		}
		limit = &n
	}

	view, err := h.SetQuota.Handle(r.Context(), callerID(r), limit)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toQuotaJSON(view))
}

// overviewJSON is the home of the panel. The views and the clicks are of the last 7 days.
type overviewJSON struct {
	ActiveCoupons    int `json:"active_coupons"`
	RedeemedToday    int `json:"redeemed_today"`
	ProfileViews7d   int `json:"profile_views_7d"`
	WhatsAppClicks7d int `json:"whatsapp_clicks_7d"`
}

func (h *Handlers) handleOverview(w http.ResponseWriter, r *http.Request) {
	got, err := h.Overview.Handle(r.Context(), callerID(r))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, overviewJSON{
		ActiveCoupons: got.ActiveCoupons, RedeemedToday: got.RedeemedToday,
		ProfileViews7d: got.ProfileViews, WhatsAppClicks7d: got.WhatsAppClicks,
	})
}
