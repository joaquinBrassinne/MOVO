package businesshttp

import (
	"net/http"

	"github.com/joaquinBrassinne/MOVO/internal/business/app/commands"
	"github.com/joaquinBrassinne/MOVO/internal/platform/httpx"
)

type geocodeRequest struct {
	Address string `json:"address"`
	City    string `json:"city"`
}

type geocodeResponse struct {
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	DisplayName string  `json:"display_name"`
}

// handleGeocode finds the pin position of a typed address. The owner confirms or
// drags the pin afterwards; nothing is stored here.
func (h *Handlers) handleGeocode(w http.ResponseWriter, r *http.Request) {
	var req geocodeRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}

	place, err := h.Geocode.Handle(r.Context(), commands.GeocodeAddressInput{OwnerID: callerID(r), Address: req.Address, City: req.City})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, geocodeResponse{Latitude: place.Latitude, Longitude: place.Longitude, DisplayName: place.DisplayName})
}
