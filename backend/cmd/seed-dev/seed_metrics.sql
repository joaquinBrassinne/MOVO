-- Fifteen months of coupons and counters for the demo business, so the Pro metrics (conversion,
-- heat map, benefit performance, records...) have something to show, plus the average ticket, an
-- "unredeemed coupons" alert and the super admin account of the admin views. The business is put
-- on Pro; to see the Starter view, change its plan from the admin (Suscripciones).
-- Safe to run again: it does nothing when the history is already there. It must run after
-- seed.sql and seed_coupons.sql.

DO $$
DECLARE
    bid      bigint;
    owner    bigint;
    brid     bigint;
    benefits bigint[];
    clients  bigint[];
BEGIN
    SELECT id, owner_user_id INTO bid, owner FROM businesses WHERE slug = 'alma-haus-training-club';
    IF bid IS NULL OR EXISTS (SELECT 1 FROM benefit_codes WHERE business_id = bid AND code LIKE 'MT%') THEN
        RETURN;
    END IF;
    SELECT id INTO brid FROM branches WHERE business_id = bid ORDER BY id LIMIT 1;
    benefits := ARRAY(SELECT id FROM benefits WHERE business_id = bid ORDER BY id);
    clients  := ARRAY(SELECT id FROM customers WHERE phone LIKE '549357100000%' ORDER BY id);
    IF brid IS NULL OR cardinality(benefits) = 0 OR cardinality(clients) = 0 THEN
        RETURN;
    END IF;

    UPDATE businesses SET average_ticket_ars = 6500 WHERE id = bid;
    UPDATE subscriptions SET plan = 'pro' WHERE business_id = bid;

    -- One to four coupons a day for the last 400 days, two out of three redeemed a few hours later.
    -- The most recent two days are all redeemed (a coupon is valid 48 hours).
    INSERT INTO benefit_codes (code, benefit_id, business_id, customer_id, access_token_hash, status,
                               created_at, expires_at, redeemed_at, redeemed_branch_id, redeemed_by)
    SELECT code, benefits[1 + (n % cardinality(benefits))], bid, clients[1 + (n % cardinality(clients))],
           sha256(convert_to(code, 'UTF8')),
           CASE WHEN redeemed THEN 'redeemed' ELSE 'expired' END,
           created, created + interval '48 hours',
           CASE WHEN redeemed THEN created + interval '3 hours' END,
           CASE WHEN redeemed THEN brid END,
           CASE WHEN redeemed THEN owner END
    FROM (
        SELECT n, created, (d <= 2 OR (d + i) % 3 <> 0) AS redeemed,
               'MT' || (SELECT string_agg(substr('ABCDEFGHJKLMNPQRSTUVWXYZ', ((n / power(24, k)::bigint) % 24)::int + 1, 1), '' ORDER BY k)
                        FROM generate_series(0, 5) k) AS code
        FROM (
            SELECT row_number() OVER (ORDER BY d, i) AS n, d, i,
                   (current_date - d)::timestamp AT TIME ZONE 'America/Argentina/Cordoba'
                     + make_interval(hours => 8 + (i * 5 + d) % 13) AS created
            FROM generate_series(1, 400) d, generate_series(1, 1 + (d % 4)) i
        ) g
    ) c;

    -- Counters for the last 90 days that the seed of coupons does not cover (it has the last 7).
    INSERT INTO metric_daily (business_id, branch_id, date, metric, count)
    SELECT bid, NULL, (current_date - d), 'profile_view', 30 + (d * 7) % 41 FROM generate_series(7, 89) d;
    INSERT INTO metric_daily (business_id, branch_id, date, metric, count)
    SELECT bid, brid, (current_date - d), 'whatsapp_click', 4 + (d * 5) % 9 FROM generate_series(7, 89) d;
    INSERT INTO metric_daily (business_id, branch_id, service_id, date, metric, count)
    SELECT bid, brid, s.id, (current_date - d), 'service_view', 3 + (d * s.id) % 11
    FROM services s, generate_series(0, 89) d
    WHERE s.business_id = bid AND s.deleted_at IS NULL;

    INSERT INTO alerts (business_id, type, payload)
    VALUES (bid, 'unredeemed_coupons',
            jsonb_build_object('period_days', 30, 'coupons_generated', 40, 'coupons_redeemed', 4,
                               'redeem_rate', 0.1, 'threshold', 0.2));
END $$;

-- The super admin of the admin views. main.go sets its password.
INSERT INTO users (email, password_hash, role)
VALUES ('admin@seed.movo.test', 'set-by-seed-dev', 'super_admin')
ON CONFLICT (email) DO NOTHING;
