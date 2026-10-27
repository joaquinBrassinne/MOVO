package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
)

// ExpireCoupons is the daily sweep that marks overdue coupons as expired (SDD 5.2).
type ExpireCoupons struct {
	expirer benefits.CouponExpirer
	now     func() time.Time
}

// NewExpireCoupons wires the ExpireCoupons use case. A nil clock uses time.Now.
func NewExpireCoupons(expirer benefits.CouponExpirer, now func() time.Time) *ExpireCoupons {
	if now == nil {
		now = time.Now
	}
	return &ExpireCoupons{expirer: expirer, now: now}
}

// Handle marks the overdue coupons and returns how many it marked.
func (c *ExpireCoupons) Handle(ctx context.Context) (int64, error) {
	n, err := c.expirer.ExpireOverdue(ctx, c.now())
	if err != nil {
		return 0, fmt.Errorf("expire coupons: %w", err)
	}
	return n, nil
}
