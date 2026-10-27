-- The owner defines the benefit freely (SDD 5.2): a percent from 1 to 100, a fixed amount
-- in pesos, or a custom benefit with its own title. Any of them may carry conditions.
-- This replaces the 10/15/20 percents and the welcome gift of 00014.
--
-- A welcome gift becomes a custom benefit. Its description (up to 200 characters in the
-- application) is split without losing text: the first 80 characters become the title
-- and the rest, at most 120, the conditions.
--
-- admin_paused is the super admin's switch, apart from the owner's active: the owner
-- cannot lift it. Coupons already generated are history and are not touched.

-- +goose Up
ALTER TABLE benefits DROP CONSTRAINT benefits_shape;
ALTER TABLE benefits DROP CONSTRAINT benefits_type_check;

ALTER TABLE benefits
    ADD COLUMN amount_ars   integer,
    ADD COLUMN title        text,
    ADD COLUMN conditions   text,
    ADD COLUMN admin_paused boolean NOT NULL DEFAULT false;

UPDATE benefits
SET type = 'custom',
    title = left(btrim(gift_description), 80),
    conditions = NULLIF(left(btrim(substr(btrim(gift_description), 81)), 120), '')
WHERE type = 'welcome_gift';

ALTER TABLE benefits DROP COLUMN gift_description;

ALTER TABLE benefits
    ADD CONSTRAINT benefits_type_check CHECK (type IN ('percent', 'amount', 'custom')),
    ADD CONSTRAINT benefits_conditions_check CHECK (
        conditions IS NULL OR (btrim(conditions) <> '' AND char_length(conditions) <= 120)
    ),
    ADD CONSTRAINT benefits_shape CHECK (
        -- IS NOT NULL is explicit: a NULL operand makes the whole branch NULL, and a
        -- CHECK that evaluates to NULL passes.
        (type = 'percent' AND percent IS NOT NULL AND percent BETWEEN 1 AND 100 AND amount_ars IS NULL AND title IS NULL)
     OR (type = 'amount'  AND amount_ars IS NOT NULL AND amount_ars > 0 AND percent IS NULL AND title IS NULL)
     OR (type = 'custom'  AND title IS NOT NULL AND btrim(title) <> '' AND char_length(title) <= 80
                          AND percent IS NULL AND amount_ars IS NULL)
    );

-- +goose Down
-- Going back cannot represent an amount or a percent outside 10/15/20: refuse instead of
-- changing what the owners offer.
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM benefits WHERE type = 'amount' OR (type = 'percent' AND percent NOT IN (10, 15, 20))) THEN
        RAISE EXCEPTION 'cannot roll back: there are fixed-amount benefits or percents other than 10, 15 and 20';
    END IF;
END $$;
-- +goose StatementEnd

ALTER TABLE benefits DROP CONSTRAINT benefits_shape;
ALTER TABLE benefits DROP CONSTRAINT benefits_conditions_check;
ALTER TABLE benefits DROP CONSTRAINT benefits_type_check;

ALTER TABLE benefits ADD COLUMN gift_description text;

-- A benefit the admin paused must not come back visible: the owner's switch goes off.
UPDATE benefits SET active = false WHERE admin_paused;
UPDATE benefits
SET type = 'welcome_gift',
    gift_description = left(title || COALESCE(' ' || conditions, ''), 200)
WHERE type = 'custom';

ALTER TABLE benefits
    DROP COLUMN amount_ars,
    DROP COLUMN title,
    DROP COLUMN conditions,
    DROP COLUMN admin_paused;

ALTER TABLE benefits
    ADD CONSTRAINT benefits_type_check CHECK (type IN ('percent', 'welcome_gift')),
    ADD CONSTRAINT benefits_shape CHECK (
        (type = 'percent'      AND percent IS NOT NULL AND percent IN (10, 15, 20) AND gift_description IS NULL)
     OR (type = 'welcome_gift' AND percent IS NULL AND gift_description IS NOT NULL AND btrim(gift_description) <> '')
    );
