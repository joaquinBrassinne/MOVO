package businesshttp

import (
	"errors"
	"net/http"

	"github.com/joaquinBrassinne/MOVO/internal/business"
	"github.com/joaquinBrassinne/MOVO/internal/business/app/commands"
	"github.com/joaquinBrassinne/MOVO/internal/business/app/queries"
	"github.com/joaquinBrassinne/MOVO/internal/business/domain"
	"github.com/joaquinBrassinne/MOVO/internal/platform/httpx"
	"github.com/joaquinBrassinne/MOVO/internal/platform/storage"
)

// Codes specific to the business endpoints.
const (
	CodeValidationFailed = "validation_failed"
	CodeBusinessNotFound = "business_not_found"
	CodeBranchNotFound   = "branch_not_found"
	CodeServiceNotFound  = "service_not_found"
	CodeAddressNotFound  = "address_not_found"
	CodeImageNotFound    = "image_not_found"
	CodeImageLimit       = "image_limit_reached"
	CodeImageExists      = "image_exists"
	CodeGeocoderBusy     = "geocoder_busy"
	CodeGeocoderDown     = "geocoder_unavailable"
	CodeBusinessExists   = "business_exists"
	CodeSlugUnavailable  = "slug_unavailable"
	CodePlanLimitReached = "plan_limit_reached"
	CodeBusinessDeleted  = "business_deleted"
	CodeInvalidState     = "invalid_state"
	CodeNoChange         = "no_change"
	CodeConflict         = "conflict"
)

// fieldError is a validation failure: which field and why, in stable words.
type fieldError struct {
	field  string
	reason string
}

// validation maps each domain or storage validation error to the field it is
// about. Anything not listed here is not a client mistake.
var validation = map[error]fieldError{
	domain.ErrNameInvalid:               {"name", "invalid"},
	domain.ErrSlugInvalid:               {"name", "no_letters_or_digits"},
	domain.ErrDescriptionInvalid:        {"description", "too_long"},
	domain.ErrWhatsAppInvalid:           {"whatsapp", "invalid"},
	domain.ErrCategoryRequired:          {"category_id", "required"},
	business.ErrCategoryNotFound:        {"category_id", "unknown"},
	domain.ErrPayerEmailInvalid:         {"mp_payer_email", "invalid"},
	domain.ErrPayerNameInvalid:          {"mp_payer_name", "invalid"},
	domain.ErrBranchNameInvalid:         {"name", "invalid"},
	domain.ErrAddressInvalid:            {"address", "invalid"},
	domain.ErrCityInvalid:               {"city", "invalid"},
	domain.ErrLocationInvalid:           {"location", "invalid"},
	domain.ErrBookingURLInvalid:         {"booking_url", "invalid"},
	domain.ErrReferenceInvalid:          {"reference", "too_long"},
	domain.ErrPriceInvalid:              {"price", "invalid"},
	domain.ErrServiceNameInvalid:        {"name", "invalid"},
	domain.ErrServiceDescriptionInvalid: {"description", "too_long"},
	domain.ErrBranchNotFoundForService:  {"branch_id", "invalid"},
	domain.ErrClockInvalid:              {"slots", "invalid_time"},
	domain.ErrSlotInvalid:               {"slots", "invalid"},
	domain.ErrSlotsOverlap:              {"slots", "overlap"},
	storage.ErrContentTypeNotAllowed:    {"content_type", "not_allowed"},
	storage.ErrSizeInvalid:              {"size", "invalid"},
	domain.ErrImageKeyInvalid:           {"object_key", "invalid"},
	business.ErrImageNotUploaded:        {"object_key", "not_uploaded"},
	business.ErrImageOrderInvalid:       {"ids", "invalid"},
	commands.ErrUnknownAdminAction:      {"action", "unknown"},
	queries.ErrAdminFilterInvalid:       {"filter", "invalid"},
}

// conflicts maps the state errors to their 409 answer.
var conflicts = []struct {
	err     error
	code    string
	message string
	details any
}{
	{business.ErrOwnerHasBusiness, CodeBusinessExists, "you already have a business", nil},
	{commands.ErrSlugExhausted, CodeSlugUnavailable, "no free address could be built from that name", nil},
	{business.ErrImageLimitReached, CodeImageLimit, "you already have the maximum number of photos", nil},
	{business.ErrImageExists, CodeImageExists, "that photo is already registered", nil},
	{business.ErrBranchLimitReached, CodePlanLimitReached, "your plan has no room for another branch", map[string]string{"resource": "branches"}},
	{business.ErrServiceLimitReached, CodePlanLimitReached, "your plan has no room for another active service", map[string]string{"resource": "services"}},
	{domain.ErrNotEditable, CodeBusinessDeleted, "this business was deleted and can no longer change", nil},
	{domain.ErrInvalidTransition, CodeInvalidState, "the business status does not allow this action", nil},
	{commands.ErrNoChange, CodeNoChange, "the business is already in that state", nil},
	{business.ErrStatusConflict, CodeConflict, "the business was changed by someone else, reload and try again", nil},
}

// notFound maps the missing-resource errors to their 404 answer.
var notFound = []struct {
	err     error
	code    string
	message string
}{
	{business.ErrBusinessNotFound, CodeBusinessNotFound, "you have not created a business yet"},
	{business.ErrBranchNotFound, CodeBranchNotFound, "branch not found"},
	{business.ErrServiceNotFound, CodeServiceNotFound, "service not found"},
	{business.ErrImageNotFound, CodeImageNotFound, "photo not found"},
	{business.ErrAddressNotFound, CodeAddressNotFound, "we could not find that address, check it or place the pin by hand"},
}

// fail maps a use case error to its HTTP answer. Anything unexpected is logged
// with the request id and reported as a generic 500: the error text may name
// internals and never reaches the client.
func (h *Handlers) fail(w http.ResponseWriter, r *http.Request, err error) {
	for _, nf := range notFound {
		if errors.Is(err, nf.err) {
			httpx.WriteError(w, http.StatusNotFound, nf.code, nf.message, nil)
			return
		}
	}
	for _, c := range conflicts {
		if errors.Is(err, c.err) {
			details := c.details
			if errors.Is(err, business.ErrImageLimitReached) {
				details = map[string]int{"max": h.MaxImages}
			}
			httpx.WriteError(w, http.StatusConflict, c.code, c.message, details)
			return
		}
	}
	switch {
	case errors.Is(err, business.ErrGeocoderBusy):
		w.Header().Set("Retry-After", "5")
		httpx.WriteError(w, http.StatusTooManyRequests, CodeGeocoderBusy, "the address search is busy, try again in a few seconds", nil)
		return
	case errors.Is(err, business.ErrGeocoderUnavailable):
		h.log.Warn("geocoder unavailable", "request_id", httpx.RequestID(r.Context()), "error", err)
		httpx.WriteError(w, http.StatusBadGateway, CodeGeocoderDown, "the address search is not available right now, place the pin by hand", nil)
		return
	}
	for known, field := range validation {
		if errors.Is(err, known) {
			invalid(w, field.field, field.reason)
			return
		}
	}

	h.log.Error("business request failed", "request_id", httpx.RequestID(r.Context()), "path", r.URL.Path, "error", err)
	httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "something went wrong", nil)
}

func invalid(w http.ResponseWriter, field, reason string) {
	httpx.WriteError(w, http.StatusUnprocessableEntity, CodeValidationFailed, "the "+field+" is not valid",
		map[string]string{"field": field, "reason": reason})
}
