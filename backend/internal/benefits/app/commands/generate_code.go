package commands

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
	"github.com/joaquinBrassinne/MOVO/internal/benefits/domain"
)

// maxCodeAttempts bounds the retries when a random code collides. With 32^8 codes a
// second collision is already astronomically unlikely; the bound only keeps a broken
// random source from looping.
const maxCodeAttempts = 5

// GenerateCodeInput is what a customer declares to claim a benefit.
type GenerateCodeInput struct {
	BenefitID int64
	Name      string
	Phone     string
	// ClientIP is the origin the request counts against; it is only a rate limit key
	// and is never stored.
	ClientIP string
}

// GeneratedCode is the coupon the customer takes away. AccessToken is shown only now:
// the database keeps its hash, so it cannot be recovered later.
type GeneratedCode struct {
	Coupon      domain.Coupon
	Benefit     domain.Benefit
	AccessToken string
}

// GenerateCode turns a benefit into a coupon for a customer without an account.
type GenerateCode struct {
	issuer  benefits.CouponIssuer
	limiter benefits.GenerationLimiter
	rand    io.Reader
	now     func() time.Time
}

// NewGenerateCode wires the GenerateCode use case. A nil random source uses
// crypto/rand and a nil clock uses time.Now.
func NewGenerateCode(issuer benefits.CouponIssuer, limiter benefits.GenerationLimiter, random io.Reader, now func() time.Time) *GenerateCode {
	if random == nil {
		random = rand.Reader
	}
	if now == nil {
		now = time.Now
	}
	return &GenerateCode{issuer: issuer, limiter: limiter, rand: random, now: now}
}

// Handle counts the attempt against the origin, validates the contact and issues the
// coupon. It returns *benefits.RateLimitedError when the origin asked too often (every
// attempt counts, valid or not, so guessing ids or numbers is throttled too),
// benefits.ErrBenefitNotFound for an inactive benefit or a business that is not
// visible, benefits.ErrPendingCoupon when the customer already holds one at the
// business, and benefits.ErrQuotaReached when the monthly limit is used up.
func (c *GenerateCode) Handle(ctx context.Context, in GenerateCodeInput) (GeneratedCode, error) {
	if retryAfter, ok := c.limiter.Allow(in.ClientIP); !ok {
		return GeneratedCode{}, &benefits.RateLimitedError{RetryAfter: retryAfter}
	}
	contact, err := domain.NewContact(in.Name, in.Phone)
	if err != nil {
		return GeneratedCode{}, err
	}
	token, hash, err := domain.NewAccessToken(c.rand)
	if err != nil {
		return GeneratedCode{}, err
	}

	now := c.now()
	monthStart, monthEnd := domain.QuotaMonth(now)
	req := benefits.IssueRequest{
		BenefitID:  in.BenefitID,
		Contact:    contact,
		TokenHash:  hash,
		Now:        now,
		ExpiresAt:  domain.ExpiryFrom(now),
		MonthStart: monthStart,
		MonthEnd:   monthEnd,
	}

	for range maxCodeAttempts {
		if req.Code, err = domain.NewCode(c.rand); err != nil {
			return GeneratedCode{}, err
		}
		issued, err := c.issuer.Issue(ctx, req)
		if errors.Is(err, benefits.ErrCodeTaken) {
			continue
		}
		if err != nil {
			return GeneratedCode{}, fmt.Errorf("issue coupon: %w", err)
		}
		return GeneratedCode{Coupon: issued.Coupon, Benefit: issued.Benefit, AccessToken: token}, nil
	}
	return GeneratedCode{}, fmt.Errorf("issue coupon: no free code after %d attempts: %w", maxCodeAttempts, benefits.ErrCodeTaken)
}
