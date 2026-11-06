package domain

// Category is what a business is about (gimnasios, yoga, pádel...). The catalog
// is seeded by a migration and read-only for owners.
type Category struct {
	ID   int64
	Slug string
	Name string
}
