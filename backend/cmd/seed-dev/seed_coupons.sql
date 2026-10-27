-- Benefits, coupons and counters for the demo business (the one whose owner can sign in), so
-- the benefits view, the home of the panel and the redeem view have something to show.
-- Safe to run again: it does nothing when the business already has benefits.
--
-- Codes to try in "Canjear cupón" (the owner of the demo business can redeem them):
--   ALMA2345  pending, 15 % off, a customer who already redeemed one coupon here
--   ALMA6789  pending, custom benefit
--   ALMA7777  pending, fixed amount
--   ALMAVENC  expired
--   ALMACAN5  already redeemed

DO $$
DECLARE
    bid     bigint;
    owner   bigint;
    brid    bigint;
    pct     bigint;
    amt     bigint;
    gift    bigint;
    lucia   bigint;
    tomas   bigint;
    ana     bigint;
    beto    bigint;
    carla   bigint;
BEGIN
    SELECT id, owner_user_id INTO bid, owner FROM businesses WHERE slug = 'alma-haus-training-club';
    IF bid IS NULL OR EXISTS (SELECT 1 FROM benefits WHERE business_id = bid) THEN
        RETURN;
    END IF;
    SELECT id INTO brid FROM branches WHERE business_id = bid ORDER BY id LIMIT 1;

    UPDATE businesses SET monthly_coupon_limit = 30 WHERE id = bid;

    INSERT INTO benefits (business_id, type, percent, conditions)
    VALUES (bid, 'percent', 15, 'Válido para alumnos nuevos en Musculación o Funcional.') RETURNING id INTO pct;
    INSERT INTO benefits (business_id, type, amount_ars, conditions)
    VALUES (bid, 'amount', 2000, 'Solo de lunes a viernes.') RETURNING id INTO amt;
    INSERT INTO benefits (business_id, type, title, conditions)
    VALUES (bid, 'custom', 'Regalo de bienvenida: una botella de agua', 'Se entrega al validar el primer acceso.') RETURNING id INTO gift;

    INSERT INTO customers (name, phone) VALUES ('Lucía Martínez', '5493571000001') RETURNING id INTO lucia;
    INSERT INTO customers (name, phone) VALUES ('Tomás Rojas',    '5493571000002') RETURNING id INTO tomas;
    INSERT INTO customers (name, phone) VALUES ('Ana Pérez',      '5493571000003') RETURNING id INTO ana;
    INSERT INTO customers (name, phone) VALUES ('Beto Gómez',     '5493571000004') RETURNING id INTO beto;
    INSERT INTO customers (name, phone) VALUES ('Carla Díaz',     '5493571000005') RETURNING id INTO carla;

    -- Redeemed: Lucía twice (so she shows as a returning customer), Beto yesterday.
    INSERT INTO benefit_codes (code, benefit_id, business_id, customer_id, access_token_hash, status,
                               created_at, expires_at, redeemed_at, redeemed_branch_id, redeemed_by)
    VALUES
      ('ALMACAN5', pct,  bid, lucia, sha256(convert_to('ALMACAN5', 'UTF8')), 'redeemed',
       now() - interval '3 hours',  now() + interval '45 hours', now() - interval '1 hour',  brid, owner),
      ('ALMACAN6', pct,  bid, lucia, sha256(convert_to('ALMACAN6', 'UTF8')), 'redeemed',
       now() - interval '30 hours', now() + interval '18 hours', now() - interval '26 hours', brid, owner),
      ('ALMACAN7', gift, bid, beto,  sha256(convert_to('ALMACAN7', 'UTF8')), 'redeemed',
       now() - interval '40 hours', now() + interval '8 hours',  now() - interval '30 hours', brid, owner);

    -- Pending: they expire 48 hours after they were generated.
    INSERT INTO benefit_codes (code, benefit_id, business_id, customer_id, access_token_hash, status, created_at, expires_at)
    VALUES
      ('ALMA2345', pct,  bid, lucia, sha256(convert_to('ALMA2345', 'UTF8')), 'pending', now() - interval '2 minutes', now() + interval '48 hours' - interval '2 minutes'),
      ('ALMA6789', gift, bid, tomas, sha256(convert_to('ALMA6789', 'UTF8')), 'pending', now() - interval '3 hours',   now() + interval '45 hours'),
      ('ALMA7777', amt,  bid, ana,   sha256(convert_to('ALMA7777', 'UTF8')), 'pending', now() - interval '20 hours',  now() + interval '28 hours');

    -- Expired: generated three days ago and never redeemed.
    INSERT INTO benefit_codes (code, benefit_id, business_id, customer_id, access_token_hash, status, created_at, expires_at)
    VALUES ('ALMAVENC', amt, bid, carla, sha256(convert_to('ALMAVENC', 'UTF8')), 'expired',
            now() - interval '72 hours', now() - interval '24 hours');

    -- Seven days of counters, the last one being today.
    INSERT INTO metric_daily (business_id, branch_id, date, metric, count)
    SELECT bid, NULL, (current_date - d), 'profile_view', 30 + (d * 7) % 41 FROM generate_series(0, 6) d;
    INSERT INTO metric_daily (business_id, branch_id, date, metric, count)
    SELECT bid, brid, (current_date - d), 'whatsapp_click', 4 + (d * 5) % 9 FROM generate_series(0, 6) d;
END $$;
