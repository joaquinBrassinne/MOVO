package postgres_test

import (
	"testing"

	"github.com/joaquinBrassinne/MOVO/internal/benefits/adapters/postgres"
	"github.com/joaquinBrassinne/MOVO/internal/platform/dbtest"
)

// TestTheFirstNameOfACustomerIsKept pins that a phone is one person: the name given the
// first time stays, so a typo or a joke at another business cannot rename the customer
// in every business that already knows them.
func TestTheFirstNameOfACustomerIsKept(t *testing.T) {
	t.Parallel()

	pool := dbtest.New(t)
	ctx := t.Context()
	first := newShop(ctx, t, pool, "studio", "active", 0)
	second := newShop(ctx, t, pool, "gym", "active", 0)
	store := postgres.NewCoupons(pool)

	if _, err := store.Issue(ctx, request(first, "5491112345678", "ABCDEFGH")); err != nil {
		t.Fatalf("first issue: %v", err)
	}
	other := request(second, "5491112345678", "JKLMNPQR")
	other.Contact.Name = "Otro Nombre"
	if _, err := store.Issue(ctx, other); err != nil {
		t.Fatalf("second issue: %v", err)
	}

	for _, s := range []shop{first, second} {
		code := map[int64]string{first.business: "ABCDEFGH", second.business: "JKLMNPQR"}[s.business]
		preview, err := store.Preview(ctx, s.business, code)
		if err != nil {
			t.Fatalf("Preview: %v", err)
		}
		if preview.CustomerName != "Ana" {
			t.Errorf("customer name at business %d = %q, want the first one, Ana", s.business, preview.CustomerName)
		}
	}
}
