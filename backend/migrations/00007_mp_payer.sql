-- The payer of a Mercado Pago subscription gets no receipt or operation number, so the
-- owner declares the email and the full name the payment was made with instead. The
-- super admin crosses them with the "new subscriber" mail Mercado Pago sends them.
-- mp_return_params keeps what Mercado Pago appended to the return URL (for instance
-- the plan) only as a hint for the admin; it is never proof of payment.

-- +goose Up
ALTER TABLE businesses
    ADD COLUMN mp_payer_email   text CHECK (char_length(mp_payer_email) <= 254),
    ADD COLUMN mp_payer_name    text CHECK (char_length(mp_payer_name) <= 100),
    ADD COLUMN mp_return_params jsonb CHECK (jsonb_typeof(mp_return_params) = 'object'),
    DROP COLUMN payment_reference;

-- +goose Down
ALTER TABLE businesses
    ADD COLUMN payment_reference text,
    DROP COLUMN mp_return_params,
    DROP COLUMN mp_payer_name,
    DROP COLUMN mp_payer_email;
