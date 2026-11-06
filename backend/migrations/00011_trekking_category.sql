-- Adds the category for trekking guides and outdoor companies: trekking, horse riding,
-- kayak, mountain bike and climbing. The seed of 00003 stays as it was; a new category
-- is a new migration.

-- +goose Up
INSERT INTO categories (slug, name) VALUES ('trekking-outdoor', 'Trekking y outdoor');

-- +goose Down
-- Fails while a business uses the category (the foreign key), which is the safe answer.
DELETE FROM categories WHERE slug = 'trekking-outdoor';
