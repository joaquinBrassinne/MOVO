package main

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/joaquinBrassinne/MOVO/internal/benefits"
	"github.com/joaquinBrassinne/MOVO/internal/billing"
	"github.com/joaquinBrassinne/MOVO/internal/billing/adapters/configcatalog"
	billinghttp "github.com/joaquinBrassinne/MOVO/internal/billing/adapters/http"
	"github.com/joaquinBrassinne/MOVO/internal/billing/adapters/pilotplan"
	billingpostgres "github.com/joaquinBrassinne/MOVO/internal/billing/adapters/postgres"
	billingcommands "github.com/joaquinBrassinne/MOVO/internal/billing/app/commands"
	billingqueries "github.com/joaquinBrassinne/MOVO/internal/billing/app/queries"
	"github.com/joaquinBrassinne/MOVO/internal/business"
	businesshttp "github.com/joaquinBrassinne/MOVO/internal/business/adapters/http"
	businesspostgres "github.com/joaquinBrassinne/MOVO/internal/business/adapters/postgres"
	"github.com/joaquinBrassinne/MOVO/internal/business/app/commands"
	"github.com/joaquinBrassinne/MOVO/internal/business/app/queries"
	"github.com/joaquinBrassinne/MOVO/internal/platform/config"
	"github.com/joaquinBrassinne/MOVO/internal/platform/httpx"
)

// entitlementsOf adapts the billing query to the port the business module
// declares: the two modules never import each other, only this root joins them.
type entitlementsOf struct {
	query *billingqueries.EntitlementsFor
}

func (e entitlementsOf) For(ctx context.Context, businessID int64) (business.Entitlements, error) {
	return e.query.Handle(ctx, businessID)
}

// metricsPermission adapts the billing query to the port the benefits module declares: a
// business may read the Pro metrics when its entitlements (a pilot gets the Pro ones) say so.
type metricsPermission struct {
	query *billingqueries.EntitlementsFor
}

func (m metricsPermission) AdvancedMetrics(ctx context.Context, businessID int64) (bool, error) {
	granted, err := m.query.Handle(ctx, businessID)
	if err != nil {
		return false, err
	}
	return granted.AdvancedMetrics(), nil
}

// newMetricsPermission builds the Pro metrics permission from the plan settings and the same
// resolver the business routes use, so both answer alike.
func newMetricsPermission(pool *pgxpool.Pool, plans config.Plans) (benefits.MetricsPermission, error) {
	catalog, err := configcatalog.New(plans)
	if err != nil {
		return nil, err
	}
	resolver, _ := newPlanResolver(pool)
	return metricsPermission{query: billingqueries.NewEntitlementsFor(catalog, resolver)}, nil
}

// newPlanResolver tells which plan a business is on: pilot businesses get the Pro limits for
// free; everyone else is on the plan of their subscription (Starter until the admin confirms
// the payment). Dropping pilotplan ends the pilot. It also returns the subscriptions store.
func newPlanResolver(pool *pgxpool.Pool) (billing.PlanResolver, *billingpostgres.Subscriptions) {
	subscriptions := billingpostgres.NewSubscriptions(pool)
	return pilotplan.New(businesspostgres.NewBusinesses(pool), subscriptions), subscriptions
}

// ownerPlans answers which plan an owner's business is on. It joins the two modules
// here so billing never reads businesses.
type ownerPlans struct {
	businesses *businesspostgres.Businesses
	resolver   billing.PlanResolver
	catalog    billing.PlanCatalog
}

func (o ownerPlans) PlanOfOwner(ctx context.Context, ownerID int64) (billing.PlanInfo, error) {
	b, err := o.businesses.FindByOwner(ctx, ownerID)
	if errors.Is(err, business.ErrBusinessNotFound) {
		return billing.PlanInfo{}, billing.ErrOwnerHasNoBusiness
	}
	if err != nil {
		return billing.PlanInfo{}, err
	}

	plan, err := o.resolver.PlanOf(ctx, b.ID)
	if err != nil {
		return billing.PlanInfo{}, err
	}
	for _, info := range o.catalog.Plans() {
		if info.Plan == plan {
			return info, nil
		}
	}
	return billing.PlanInfo{}, billing.ErrUnknownPlan
}

// ownerSubscriptions answers the payment state of an owner's business, joining the two
// modules here so billing never reads businesses.
type ownerSubscriptions struct {
	businesses    *businesspostgres.Businesses
	subscriptions *billingpostgres.Subscriptions
}

func (o ownerSubscriptions) SubscriptionOfOwner(ctx context.Context, ownerID int64) (billing.OwnerSubscription, error) {
	b, err := o.businesses.FindByOwner(ctx, ownerID)
	if errors.Is(err, business.ErrBusinessNotFound) {
		return billing.OwnerSubscription{}, billing.ErrOwnerHasNoBusiness
	}
	if err != nil {
		return billing.OwnerSubscription{}, err
	}

	own := billing.OwnerSubscription{Pilot: b.Pilot}
	sub, err := o.subscriptions.Find(ctx, b.ID)
	switch {
	case errors.Is(err, billing.ErrSubscriptionNotFound):
	case err != nil:
		return billing.OwnerSubscription{}, err
	default:
		own.Subscription = &sub
	}
	return own, nil
}

// newBusinessRoutes composes the business and billing modules: adapters, use
// cases and HTTP. It returns the mounts of both.
func newBusinessRoutes(
	log *slog.Logger,
	pool *pgxpool.Pool,
	verifier httpx.TokenVerifier,
	plans config.Plans,
	images business.ImageStorage,
	maxImages int,
	geocoder business.Geocoder,
) (businessMount, billingMount func(chi.Router), err error) {
	catalog, err := configcatalog.New(plans)
	if err != nil {
		return nil, nil, err
	}

	businesses := businesspostgres.NewBusinesses(pool)
	branches := businesspostgres.NewBranches(pool)
	services := businesspostgres.NewServices(pool)
	admin := businesspostgres.NewAdmin(pool)
	photos := businesspostgres.NewImages(pool)

	resolver, subscriptions := newPlanResolver(pool)
	entitlements := entitlementsOf{query: billingqueries.NewEntitlementsFor(catalog, resolver)}

	handlers := businesshttp.New(businesshttp.Deps{
		Verifier:  verifier,
		MaxImages: maxImages,

		MyBusiness:     queries.NewGetMyBusiness(businesses, branches, photos, images),
		CreateBusiness: commands.NewCreateBusiness(businesses),
		UpdateBusiness: commands.NewUpdateBusiness(businesses, entitlements),
		SubmitBusiness: commands.NewSubmitBusiness(businesses),
		PresignImage:   commands.NewPresignImageUpload(businesses, images, nil),
		Geocode:        commands.NewGeocodeAddress(businesses, geocoder),
		RegisterImage:  commands.NewRegisterImage(businesses, photos, images, maxImages),
		ReorderImages:  commands.NewReorderImages(businesses, photos),
		RemoveImage:    commands.NewRemoveImage(businesses, photos, images, businesspostgres.NewCleanup(pool), log),

		AddBranch:       commands.NewAddBranch(businesses, branches, entitlements),
		EditBranch:      commands.NewEditBranch(businesses, branches),
		RetireBranch:    commands.NewRetireBranch(businesses, branches),
		SetBranchHours:  commands.NewSetBranchHours(businesses, branches),
		SetBranchActive: commands.NewSetBranchActive(businesses, branches, entitlements),

		ListServices:       queries.NewListMyServices(businesses, services),
		AddService:         commands.NewAddService(businesses, branches, services, entitlements),
		EditService:        commands.NewEditService(businesses, branches, services),
		RemoveService:      commands.NewRemoveService(businesses, services),
		SetServiceActive:   commands.NewSetServiceActive(businesses, services, entitlements),
		SetServiceSchedule: commands.NewSetServiceSchedule(businesses, services),

		Categories:     queries.NewListCategories(businesspostgres.NewCategories(pool)),
		ListBusinesses: queries.NewListBusinesses(admin),
		AdminAction:    commands.NewApplyAdminAction(admin),
		AdminDetail:    queries.NewGetBusinessDetail(admin, branches, services),
	})

	planHandlers := billinghttp.New(catalog).
		WithOwnerPlans(verifier, ownerPlans{businesses: businesses, resolver: resolver, catalog: catalog}).
		WithOwnerSubscription(verifier, ownerSubscriptions{businesses: businesses, subscriptions: subscriptions}).
		WithAlerts(verifier, billingpostgres.NewAlerts(pool))
	adminHandlers := billinghttp.NewAdmin(log, billinghttp.AdminDeps{
		Verifier:      verifier,
		Subscriptions: subscriptions,
		Confirm:       billingcommands.NewConfirmPayment(subscriptions, time.Now),
		Register:      billingcommands.NewRegisterPayment(subscriptions, time.Now),
		ChangePlan:    billingcommands.NewChangePlan(subscriptions, catalog),
		List:          billingqueries.NewListSubscriptions(subscriptions),
	})
	billingMount = func(r chi.Router) {
		planHandlers.Mount(r)
		adminHandlers.Mount(r)
	}
	return handlers.Mount, billingMount, nil
}
