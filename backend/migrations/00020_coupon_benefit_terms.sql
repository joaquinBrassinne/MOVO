-- A coupon keeps the terms it was issued under. Until now it read them from its benefit,
-- so when the owner edited the benefit, every coupon already in a customer's hands changed
-- with it: a "-50%" held by a customer became "-5%" at the counter, and the history of
-- past redemptions was rewritten. The terms are copied into the coupon when it is issued;
-- benefit_id still points at the benefit, which keeps the statistics per benefit.
--
-- Coupons that already exist are filled from their benefit as it is today: it is the best
-- that can be rebuilt, and from here on the copy never moves.

-- +goose Up
ALTER TABLE benefit_codes
    ADD COLUMN benefit_type       text,
    ADD COLUMN benefit_percent    smallint,
    ADD COLUMN benefit_amount_ars integer,
    ADD COLUMN benefit_title      text,
    ADD COLUMN benefit_conditions text;

UPDATE benefit_codes c
SET benefit_type       = b.type,
    benefit_percent    = b.percent,
    benefit_amount_ars = b.amount_ars,
    benefit_title      = b.title,
    benefit_conditions = b.conditions
FROM benefits b
WHERE b.id = c.benefit_id;

ALTER TABLE benefit_codes
    ALTER COLUMN benefit_type SET NOT NULL,
    -- Same rule as benefits_shape: the fields of each type stay coherent.
    ADD CONSTRAINT benefit_codes_terms_shape CHECK (
        (benefit_type = 'percent' AND benefit_percent IS NOT NULL AND benefit_percent BETWEEN 1 AND 100
                                  AND benefit_amount_ars IS NULL AND benefit_title IS NULL)
     OR (benefit_type = 'amount'  AND benefit_amount_ars IS NOT NULL AND benefit_amount_ars > 0
                                  AND benefit_percent IS NULL AND benefit_title IS NULL)
     OR (benefit_type = 'custom'  AND benefit_title IS NOT NULL AND btrim(benefit_title) <> ''
                                  AND char_length(benefit_title) <= 80
                                  AND benefit_percent IS NULL AND benefit_amount_ars IS NULL)
    ),
    ADD CONSTRAINT benefit_codes_terms_conditions_check CHECK (
        benefit_conditions IS NULL OR (btrim(benefit_conditions) <> '' AND char_length(benefit_conditions) <= 120)
    );

-- Any writer gets the copy: when an insert does not bring the terms, they are taken from
-- the benefit. Issue sends them explicitly, so a coupon carries exactly what the customer
-- was shown even if the owner edits the benefit in between; this is the net for the rest
-- (seeds, tests, future code), so a coupon can never exist without its terms. A BEFORE
-- trigger runs ahead of the NOT NULL and CHECK constraints.
-- +goose StatementBegin
CREATE FUNCTION benefit_codes_fill_terms() RETURNS trigger AS $$
BEGIN
    IF NEW.benefit_type IS NULL THEN
        SELECT type, percent, amount_ars, title, conditions
        INTO NEW.benefit_type, NEW.benefit_percent, NEW.benefit_amount_ars, NEW.benefit_title, NEW.benefit_conditions
        FROM benefits WHERE id = NEW.benefit_id;
    END IF;
    RETURN NEW;
END
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER benefit_codes_fill_terms BEFORE INSERT ON benefit_codes
    FOR EACH ROW EXECUTE FUNCTION benefit_codes_fill_terms();

-- +goose Down
DROP TRIGGER benefit_codes_fill_terms ON benefit_codes;
DROP FUNCTION benefit_codes_fill_terms();

ALTER TABLE benefit_codes
    DROP CONSTRAINT benefit_codes_terms_conditions_check,
    DROP CONSTRAINT benefit_codes_terms_shape,
    DROP COLUMN benefit_conditions,
    DROP COLUMN benefit_title,
    DROP COLUMN benefit_amount_ars,
    DROP COLUMN benefit_percent,
    DROP COLUMN benefit_type;
