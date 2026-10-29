-- Reviews for the demo business (the one whose owner can sign in), so the stars on the cards,
-- "Reseñas del local", the review form and the moderation page have something to show.
-- Safe to run again: it does nothing when the business already has reviews.
--
-- Access tokens to try in /cupon/{token}/resena (each is 43 characters):
--   seed-review-open-00000000000000000000000000   redeemed 2 hours ago, can be reviewed
--   seed-review-closed-000000000000000000000000   redeemed 40 days ago, the 30 days are over
--   seed-review-pending-00000000000000000000000   not redeemed yet
--   seed-reviewed-01000000000000000000000000000   already reviewed
-- The moderation page (/admin/reviews) finds 2 reported reviews, 1 hidden a few hours ago
-- (it can still be restored) and 1 hidden and already purged (it is not listed).

DO $$
DECLARE
    bid      bigint;
    owner    bigint;
    brid     bigint;
    benefit  bigint;
    cid      bigint;
    code_id  bigint;
    i        int;
    token    text;
    redeemed timestamptz;
    stars    int[] := ARRAY[5, 5, 4, 5, 3, 4, 5, 2, 5, 4, 1, 5];
    names    text[] := ARRAY['Lucía Martínez', 'Tomás Rojas', 'Ana Pérez', 'Beto Gómez', 'Carla Díaz', 'Martín Suárez'];
    shown    text[] := ARRAY['Lucía M.', 'Tomás R.', 'Ana P.', 'Beto G.', 'Carla D.', 'Martín S.'];
    comments text[] := ARRAY[
        'Increíble lugar, el equipamiento es de primera y los profes súper atentos.',
        'Muy buen ambiente y re limpio. Un poco lleno a las 19 hs pero se puede entrenar bien.',
        'Me encantó la clase de prueba. El staff de recepción re buena onda con la lectura del QR.',
        'Con el cupón entré sin vueltas y sin pedir nada raro.',
        'Buena atención, las duchas podrían estar mejor.'];
    state    text;
    hidden   timestamptz;
BEGIN
    SELECT id, owner_user_id INTO bid, owner FROM businesses WHERE slug = 'alma-haus-training-club';
    IF bid IS NULL OR EXISTS (SELECT 1 FROM reviews WHERE business_id = bid) THEN
        RETURN;
    END IF;
    SELECT id INTO brid FROM branches WHERE business_id = bid ORDER BY id LIMIT 1;
    SELECT id INTO benefit FROM benefits WHERE business_id = bid ORDER BY id LIMIT 1;
    IF brid IS NULL OR benefit IS NULL THEN
        RETURN;
    END IF;

    -- 24 redeemed coupons, a day apart, each with its review (two pages of the public list).
    FOR i IN 1..24 LOOP
        INSERT INTO customers (name, phone)
        VALUES (names[(i - 1) % 6 + 1], '5493571' || lpad((100 + i)::text, 6, '0')) RETURNING id INTO cid;
        redeemed := now() - make_interval(days => i) - interval '2 hours';
        token := rpad('seed-reviewed-' || lpad(i::text, 2, '0'), 43, '0');
        INSERT INTO benefit_codes (code, benefit_id, business_id, customer_id, access_token_hash, status,
                                   created_at, expires_at, redeemed_at, redeemed_branch_id, redeemed_by)
        -- A coupon code is 8 characters of A-H J-N P-Z 2-9: the 24 letters give 24 distinct codes.
        VALUES ('RESE' || substr('ABCDEFGHJKLMNPQRSTUVWXYZ', i, 1) || '222', benefit, bid, cid, sha256(convert_to(token, 'UTF8')), 'redeemed',
                redeemed - interval '3 hours', redeemed + interval '45 hours', redeemed, brid, owner)
        RETURNING id INTO code_id;

        state := CASE WHEN i IN (4, 11) THEN 'reported' WHEN i IN (7, 15) THEN 'hidden' ELSE 'visible' END;
        hidden := CASE WHEN i = 7 THEN now() - interval '10 hours' WHEN i = 15 THEN now() - interval '5 days' END;
        INSERT INTO reviews (branch_id, business_id, benefit_code_id, customer_id, rating, comment, display_name,
                             show_name, status, hidden_at, purged_at, created_at, updated_at)
        VALUES (brid, bid, code_id, cid, stars[(i - 1) % 12 + 1],
                CASE WHEN i = 15 OR i % 3 = 0 THEN NULL ELSE comments[(i - 1) % 5 + 1] END,
                CASE WHEN i = 15 OR i % 5 = 0 THEN 'Cliente de MOVO' ELSE shown[(i - 1) % 6 + 1] END,
                i <> 15 AND i % 5 <> 0, state, hidden,
                CASE WHEN i = 15 THEN now() - interval '2 days' END,
                redeemed + interval '1 hour', redeemed + interval '1 hour');
    END LOOP;

    -- The rating a card shows, counted like the reviews module does: a reported review counts
    -- and a hidden one does not.
    UPDATE branches SET rating_count = s.total, rating_avg = s.average
    FROM (SELECT count(*)::int AS total, COALESCE(round(avg(rating), 2), 0) AS average
          FROM reviews WHERE branch_id = brid AND status <> 'hidden') s
    WHERE branches.id = brid;

    -- The three other states of the review form.
    INSERT INTO customers (name, phone) VALUES ('Nico Vega', '5493571000201') RETURNING id INTO cid;
    INSERT INTO benefit_codes (code, benefit_id, business_id, customer_id, access_token_hash, status,
                               created_at, expires_at, redeemed_at, redeemed_branch_id, redeemed_by)
    VALUES ('RESEABRE', benefit, bid, cid, sha256(convert_to(rpad('seed-review-open-', 43, '0'), 'UTF8')), 'redeemed',
            now() - interval '5 hours', now() + interval '43 hours', now() - interval '2 hours', brid, owner);

    INSERT INTO customers (name, phone) VALUES ('Sofía Ibarra', '5493571000202') RETURNING id INTO cid;
    INSERT INTO benefit_codes (code, benefit_id, business_id, customer_id, access_token_hash, status,
                               created_at, expires_at, redeemed_at, redeemed_branch_id, redeemed_by)
    VALUES ('RESEVENC', benefit, bid, cid, sha256(convert_to(rpad('seed-review-closed-', 43, '0'), 'UTF8')), 'redeemed',
            now() - interval '41 days', now() - interval '39 days', now() - interval '40 days', brid, owner);

    INSERT INTO customers (name, phone) VALUES ('Pablo Ríos', '5493571000203') RETURNING id INTO cid;
    INSERT INTO benefit_codes (code, benefit_id, business_id, customer_id, access_token_hash, status, created_at, expires_at)
    VALUES ('RESEPEND', benefit, bid, cid, sha256(convert_to(rpad('seed-review-pending-', 43, '0'), 'UTF8')), 'pending',
            now() - interval '1 hour', now() + interval '47 hours');
END $$;
