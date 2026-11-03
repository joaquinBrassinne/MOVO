-- Average ticket (SDD 6.7): what a typical customer spends at the business, in whole pesos. The
-- owner types it in "Datos del negocio" and only the estimated return of the Pro metrics uses
-- it. Optional: NULL means the owner never loaded one, and the estimate is not shown.

-- +goose Up
ALTER TABLE businesses ADD COLUMN average_ticket_ars integer
    CHECK (average_ticket_ars IS NULL OR average_ticket_ars > 0);

-- +goose Down
ALTER TABLE businesses DROP COLUMN average_ticket_ars;
