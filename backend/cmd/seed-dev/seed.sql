-- Local test data: one public business per category around Río Tercero, so Home, Mapa and
-- the business page have something to show. Safe to run again: a business whose slug
-- already exists is skipped. The owners have no usable password.

DO $$
DECLARE
    r    record;
    uid  bigint;
    bid  bigint;
    brid bigint;
BEGIN
    FOR r IN SELECT * FROM (VALUES
        ('alma-haus-training-club', 'Alma Haus Training Club', 'gimnasios',        0.0040,  0.0020, 'Entrenamiento funcional',          'Av. San Martín 450',   true,  'Pase libre por día'),
        ('pilates-centro',          'Pilates Centro',          'pilates',          0.0070, -0.0040, 'Pilates reformer y mat',            'Belgrano 210',         true,  'Clase de reformer'),
        ('alma-flow-yoga',          'Alma Flow Yoga',          'yoga',            -0.0050,  0.0060, 'Yoga y sonido',                     'Mitre 880',            true,  'Clase de hatha yoga'),
        ('coach-rio-tercero',       'Coach Río Tercero',       'entrenadores',     0.0100,  0.0090, 'Entrenamiento personalizado',       'Sarmiento 120',        false, 'Sesión personalizada'),
        ('palermo-padel-club',      'Palermo Pádel Club',      'padel',           -0.0090, -0.0070, 'Canchas Blindex',                   'Ruta 36 km 5',         true,  'Alquiler de cancha'),
        ('estetica-bienestar',      'Estética y Bienestar',    'estetica',         0.0020, -0.0100, 'Masajes y cosmetología',            'Rivadavia 640',        false, 'Masaje descontracturante'),
        ('natacion-del-lago',       'Natación del Lago',       'natacion',        -0.0130,  0.0030, 'Pileta climatizada',                'Costanera 15',         true,  'Clase de natación'),
        ('sierras-trekking',        'Sierras Trekking',        'trekking-outdoor', 0.0160,  0.0130, 'Salidas guiadas por las sierras',   'Punto de encuentro: Plaza Central', true, 'Salida al Cerro Pelado')
    ) AS t(slug, name, category, dlat, dlng, description, address, verified, service)
    LOOP
        CONTINUE WHEN EXISTS (SELECT 1 FROM businesses WHERE slug = r.slug);

        INSERT INTO users (email, password_hash, role)
        VALUES (r.slug || '@seed.movo.test', '!', 'business_admin') RETURNING id INTO uid;

        INSERT INTO businesses (owner_user_id, category_id, name, slug, description, whatsapp_number, status, verified_at)
        VALUES (uid, (SELECT id FROM categories WHERE slug = r.category), r.name, r.slug, r.description,
                '5493571123456', 'active', CASE WHEN r.verified THEN now() END)
        RETURNING id INTO bid;

        INSERT INTO branches (business_id, name, address, city, location)
        VALUES (bid, 'Centro', r.address, 'Río Tercero',
                ST_SetSRID(ST_MakePoint(-64.1146 + r.dlng, -32.1738 + r.dlat), 4326)::geography)
        RETURNING id INTO brid;

        -- Monday to Friday with a split shift, Saturday mornings.
        INSERT INTO business_hours (branch_id, weekday, opens_at, closes_at)
        SELECT brid, d, '09:00', '13:00' FROM generate_series(1, 6) d;
        INSERT INTO business_hours (branch_id, weekday, opens_at, closes_at)
        SELECT brid, d, '16:00', '21:00' FROM generate_series(1, 5) d;

        INSERT INTO services (business_id, name, description, price)
        VALUES (bid, r.service, 'Servicio de prueba', 12000);
    END LOOP;

    -- The view only shows a pilot or paid business: give every seeded one a current
    -- subscription (verified ones on Pro so the featured stamp shows up). Also covers
    -- databases seeded before subscriptions existed.
    INSERT INTO subscriptions (business_id, plan, paid_until, confirmed_by, notes)
    SELECT b.id, CASE WHEN b.verified_at IS NOT NULL THEN 'pro' ELSE 'starter' END,
           CURRENT_DATE + 30, b.owner_user_id, 'seed-dev'
    FROM businesses b JOIN users u ON u.id = b.owner_user_id
    WHERE u.email LIKE '%@seed.movo.test'
    ON CONFLICT (business_id) DO NOTHING;
END $$;
