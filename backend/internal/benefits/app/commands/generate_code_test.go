package commands_test

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"testing/iotest"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/app/commands"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/domain"
)

// fakeIssuer records the requests and answers from a script, one entry per call.
type fakeIssuer struct {
	errs  []error
	calls []benefits.IssueRequest
}

func (f *fakeIssuer) Issue(_ context.Context, req benefits.IssueRequest) (benefits.Issued, error) {
	f.calls = append(f.calls, req)
	if i := len(f.calls) - 1; i < len(f.errs) && f.errs[i] != nil {
		return benefits.Issued{}, f.errs[i]
	}
	return benefits.Issued{
		Coupon:  domain.Coupon{ID: 1, BenefitID: req.BenefitID, Code: req.Code, Status: domain.StatusPending, ExpiresAt: req.ExpiresAt},
		Benefit: domain.Benefit{ID: req.BenefitID, Type: domain.TypePercent, Percent: 10},
	}, nil
}

// entropy gives n bytes that differ per call so each draw is a different code.
func entropy(n int) *bytes.Reader {
	raw := make([]byte, n)
	for i := range raw {
		raw[i] = byte(i * 7)
	}
	return bytes.NewReader(raw)
}

var fixedNow = func() time.Time { return time.Date(2028, 3, 15, 12, 0, 0, 0, time.UTC) }

func TestGenerateCodeIssuesWithTheNormalizedContactAndA48HourExpiry(t *testing.T) {
	t.Parallel()

	issuer := &fakeIssuer{}
	got, err := commands.NewGenerateCode(issuer, entropy(1024), fixedNow).Handle(t.Context(),
		commands.GenerateCodeInput{BenefitID: 9, Name: "  Ana ", Phone: "+54 9 11 1234-5678"})
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}

	req := issuer.calls[0]
	if req.BenefitID != 9 || req.Contact.Name != "Ana" || req.Contact.Phone != "5491112345678" {
		t.Errorf("request = %+v, want the trimmed name and normalized phone", req)
	}
	if want := fixedNow().Add(48 * time.Hour); !req.ExpiresAt.Equal(want) {
		t.Errorf("ExpiresAt = %v, want %v", req.ExpiresAt, want)
	}
	if start, end := domain.QuotaMonth(fixedNow()); !req.MonthStart.Equal(start) || !req.MonthEnd.Equal(end) {
		t.Errorf("month = [%v, %v), want the quota month", req.MonthStart, req.MonthEnd)
	}
	if len(got.Coupon.Code) != domain.CodeLength || got.Coupon.Code != req.Code {
		t.Errorf("code = %q, want the %d-symbol code that was issued", got.Coupon.Code, domain.CodeLength)
	}
	if got.AccessToken == "" || !bytes.Equal(req.TokenHash, domain.HashAccessToken(got.AccessToken)) {
		t.Error("the issuer must receive the hash of the token the customer gets, never the token")
	}
	if bytes.Contains(req.TokenHash, []byte(got.AccessToken)) {
		t.Error("the plain token must not travel to the issuer")
	}
}

func TestGenerateCodeRejectsAnInvalidContactWithoutIssuing(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name, customer, phone string
		want                  error
	}{
		{"blank name", " ", "5491112345678", domain.ErrCustomerNameInvalid},
		{"no country code", "Ana", "011 1234 5678", domain.ErrPhoneInvalid},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			issuer := &fakeIssuer{}
			_, err := commands.NewGenerateCode(issuer, entropy(1024), fixedNow).Handle(t.Context(),
				commands.GenerateCodeInput{BenefitID: 1, Name: tc.customer, Phone: tc.phone})
			if !errors.Is(err, tc.want) || len(issuer.calls) != 0 {
				t.Errorf("error = %v with %d issue calls, want %v and none", err, len(issuer.calls), tc.want)
			}
		})
	}
}

func TestGenerateCodeDrawsAnotherCodeOnACollision(t *testing.T) {
	t.Parallel()

	issuer := &fakeIssuer{errs: []error{benefits.ErrCodeTaken, benefits.ErrCodeTaken}}
	got, err := commands.NewGenerateCode(issuer, entropy(1024), fixedNow).Handle(t.Context(),
		commands.GenerateCodeInput{BenefitID: 1, Name: "Ana", Phone: "5491112345678"})
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if len(issuer.calls) != 3 {
		t.Fatalf("issue calls = %d, want 3 (two collisions, then success)", len(issuer.calls))
	}
	codes := map[string]bool{}
	for _, c := range issuer.calls {
		codes[c.Code] = true
	}
	if len(codes) != 3 || got.Coupon.Code != issuer.calls[2].Code {
		t.Errorf("codes = %v, want a new code on each attempt and the last one returned", codes)
	}
	// The token and its hash are drawn once: a retry must not change what the customer holds.
	if !bytes.Equal(issuer.calls[0].TokenHash, issuer.calls[2].TokenHash) {
		t.Error("the token hash must be the same across retries")
	}
}

func TestGenerateCodeGivesUpWhenEveryCodeCollides(t *testing.T) {
	t.Parallel()

	taken := benefits.ErrCodeTaken
	issuer := &fakeIssuer{errs: []error{taken, taken, taken, taken, taken, taken, taken}}
	_, err := commands.NewGenerateCode(issuer, entropy(1024), fixedNow).Handle(t.Context(),
		commands.GenerateCodeInput{BenefitID: 1, Name: "Ana", Phone: "5491112345678"})
	if !errors.Is(err, benefits.ErrCodeTaken) || len(issuer.calls) != 5 {
		t.Errorf("error = %v after %d calls, want ErrCodeTaken after 5", err, len(issuer.calls))
	}
}

func TestGenerateCodePassesTheRuleErrorsThroughWithoutRetrying(t *testing.T) {
	t.Parallel()

	for _, want := range []error{benefits.ErrBenefitNotFound, benefits.ErrPendingCoupon, benefits.ErrQuotaReached} {
		issuer := &fakeIssuer{errs: []error{want}}
		_, err := commands.NewGenerateCode(issuer, entropy(1024), fixedNow).Handle(t.Context(),
			commands.GenerateCodeInput{BenefitID: 1, Name: "Ana", Phone: "5491112345678"})
		if !errors.Is(err, want) || len(issuer.calls) != 1 {
			t.Errorf("error = %v after %d calls, want %v after exactly one", err, len(issuer.calls), want)
		}
	}
}

func TestGenerateCodeFailsWhenThereIsNoEntropy(t *testing.T) {
	t.Parallel()

	issuer := &fakeIssuer{}
	_, err := commands.NewGenerateCode(issuer, iotest.ErrReader(errors.New("no entropy")), fixedNow).Handle(t.Context(),
		commands.GenerateCodeInput{BenefitID: 1, Name: "Ana", Phone: "5491112345678"})
	if err == nil || len(issuer.calls) != 0 {
		t.Errorf("error = %v with %d issue calls, want a failure and no coupon", err, len(issuer.calls))
	}
}
