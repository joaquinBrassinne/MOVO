package main

import (
	"context"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/joaquinBrassinne/MOVO/internal/billing/adapters/configcatalog"
	"github.com/joaquinBrassinne/MOVO/internal/billing/adapters/fixedplan"
	billinghttp "github.com/joaquinBrassinne/MOVO/internal/billing/adapters/http"
	"github.com/joaquinBrassinne/MOVO/internal/billing/adapters/pilotplan"
	billingqueries "github.com/joaquinBrassinne/MOVO/internal/billing/app/queries"
	billingdomain "github.com/joaquinBrassinne/MOVO/internal/billing/domain"
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

// newBusinessRoutes composes the business and billing modules: adapters, use
// cases and HTTP. It returns the mounts of both.
func newBusinessRoutes(
	pool *pgxpool.Pool,
	verifier httpx.TokenVerifier,
	plans config.Plans,
	images business.ImageStorage,
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

	// Pilot businesses get the Pro limits for free; everyone else is on Starter
	// until subscriptions exist (phase 5). Dropping pilotplan ends the pilot.
	resolver := pilotplan.New(businesses, fixedplan.New(billingdomain.PlanStarter))
	entitlements := entitlementsOf{query: billingqueries.NewEntitlementsFor(catalog, resolver)}

	handlers := businesshttp.New(businesshttp.Deps{
		Verifier: verifier,

		MyBusiness:     queries.NewGetMyBusiness(businesses, branches),
		CreateBusiness: commands.NewCreateBusiness(businesses),
		UpdateBusiness: commands.NewUpdateBusiness(businesses),
		SubmitBusiness: commands.NewSubmitBusiness(businesses),
		PresignImage:   commands.NewPresignImageUpload(businesses, images, nil),
		Geocode:        commands.NewGeocodeAddress(businesses, geocoder),

		AddBranch:      commands.NewAddBranch(businesses, branches, entitlements),
		EditBranch:     commands.NewEditBranch(businesses, branches),
		RemoveBranch:   commands.NewRemoveBranch(businesses, branches),
		SetBranchHours: commands.NewSetBranchHours(businesses, branches),

		ListServices:       queries.NewListMyServices(businesses, services),
		AddService:         commands.NewAddService(businesses, branches, services, entitlements),
		EditService:        commands.NewEditService(businesses, branches, services),
		RemoveService:      commands.NewRemoveService(businesses, services),
		SetServiceActive:   commands.NewSetServiceActive(businesses, services, entitlements),
		SetServiceSchedule: commands.NewSetServiceSchedule(businesses, services),

		Categories:     queries.NewListCategories(businesspostgres.NewCategories(pool)),
		ListBusinesses: queries.NewListBusinesses(admin),
		AdminAction:    commands.NewApplyAdminAction(admin),
	})

	return handlers.Mount, billinghttp.New(catalog).Mount, nil
}
