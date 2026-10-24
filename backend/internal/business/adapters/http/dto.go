package businesshttp

import (
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/business/app/queries"
	"github.com/joaquinBrassinne/MOVO/internal/business/domain"
)

type businessJSON struct {
	ID               int64     `json:"id"`
	CategoryID       int64     `json:"category_id"`
	Name             string    `json:"name"`
	Slug             string    `json:"slug"`
	Description      string    `json:"description"`
	WhatsApp         string    `json:"whatsapp"`
	Status           string    `json:"status"`
	Pilot            bool      `json:"pilot"`
	Verified         bool      `json:"verified"`
	PaymentReference string    `json:"payment_reference"`
	CreatedAt        time.Time `json:"created_at"`
}

func toBusinessJSON(b domain.Business) businessJSON {
	return businessJSON{
		ID: b.ID, CategoryID: b.CategoryID, Name: b.Name, Slug: b.Slug, Description: b.Description,
		WhatsApp: b.WhatsApp, Status: string(b.Status), Pilot: b.Pilot, Verified: b.Verified(),
		PaymentReference: b.PaymentReference, CreatedAt: b.CreatedAt,
	}
}

// slotJSON is one opening interval; start and end are "HH:MM".
type slotJSON struct {
	Weekday int    `json:"weekday"`
	Start   string `json:"start"`
	End     string `json:"end"`
}

func toSlotsJSON(slots []domain.Slot) []slotJSON {
	out := make([]slotJSON, 0, len(slots))
	for _, s := range slots {
		out = append(out, slotJSON{Weekday: s.Weekday, Start: s.Start.String(), End: s.End.String()})
	}
	return out
}

type branchJSON struct {
	ID         int64      `json:"id"`
	Name       string     `json:"name"`
	Address    string     `json:"address"`
	City       string     `json:"city"`
	Latitude   float64    `json:"latitude"`
	Longitude  float64    `json:"longitude"`
	BookingURL string     `json:"booking_url"`
	Active     bool       `json:"active"`
	Hours      []slotJSON `json:"hours"`
}

func toBranchJSON(b domain.Branch, hours []domain.Slot) branchJSON {
	return branchJSON{
		ID: b.ID, Name: b.Name, Address: b.Address, City: b.City, Latitude: b.Latitude, Longitude: b.Longitude,
		BookingURL: b.BookingURL, Active: b.Active, Hours: toSlotsJSON(hours),
	}
}

type myBusinessJSON struct {
	Business businessJSON `json:"business"`
	Branches []branchJSON `json:"branches"`
}

func toMyBusinessJSON(v queries.MyBusinessView) myBusinessJSON {
	out := myBusinessJSON{Business: toBusinessJSON(v.Business), Branches: make([]branchJSON, 0, len(v.Branches))}
	for _, b := range v.Branches {
		out.Branches = append(out.Branches, toBranchJSON(b.Branch, b.Hours))
	}
	return out
}

// toSlots parses the request slots; a time that is not "HH:MM" is domain.ErrClockInvalid.
func toSlots(in []slotJSON) ([]domain.Slot, error) {
	out := make([]domain.Slot, 0, len(in))
	for _, s := range in {
		start, err := domain.ParseClock(s.Start)
		if err != nil {
			return nil, err
		}
		end, err := domain.ParseClock(s.End)
		if err != nil {
			return nil, err
		}
		out = append(out, domain.Slot{Weekday: s.Weekday, Start: start, End: end})
	}
	return out, nil
}

// serviceBaseJSON is what a service write answers with. The schedule is left out
// because those writes do not touch it; GET /me/business/services carries it.
type serviceBaseJSON struct {
	ID          int64  `json:"id"`
	BranchID    *int64 `json:"branch_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Active      bool   `json:"active"`
}

type serviceJSON struct {
	serviceBaseJSON
	Schedule []slotJSON `json:"schedule"`
}

func toServiceBaseJSON(s domain.Service) serviceBaseJSON {
	out := serviceBaseJSON{ID: s.ID, Name: s.Name, Description: s.Description, Active: s.Active}
	if s.BranchID != 0 {
		out.BranchID = &s.BranchID
	}
	return out
}

func toServiceJSON(s domain.Service, schedule []domain.Slot) serviceJSON {
	return serviceJSON{serviceBaseJSON: toServiceBaseJSON(s), Schedule: toSlotsJSON(schedule)}
}
