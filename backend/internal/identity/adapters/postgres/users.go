// Package postgres implements the identity ports on top of PostgreSQL.
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/joaquinBrassinne/MOVO/internal/identity"
	"github.com/joaquinBrassinne/MOVO/internal/identity/domain"
)

const uniqueViolation = "23505"

const selectUser = `SELECT id, email, password_hash, role, created_at FROM users`

// Users implements identity.UserRepository.
type Users struct {
	pool *pgxpool.Pool
}

// NewUsers builds the repository over pool.
func NewUsers(pool *pgxpool.Pool) *Users {
	return &Users{pool: pool}
}

// Create stores user and returns it with ID and CreatedAt set.
func (r *Users) Create(ctx context.Context, user domain.User) (domain.User, error) {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO users (email, password_hash, role) VALUES ($1, $2, $3) RETURNING id, created_at`,
		user.Email, user.PasswordHash, string(user.Role),
	).Scan(&user.ID, &user.CreatedAt)

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
		return domain.User{}, identity.ErrEmailTaken
	}
	if err != nil {
		return domain.User{}, fmt.Errorf("postgres: insert user: %w", err)
	}

	return user, nil
}

// FindByEmail looks a user up by its normalized email.
func (r *Users) FindByEmail(ctx context.Context, email string) (domain.User, error) {
	return r.find(ctx, selectUser+` WHERE email = $1`, email)
}

// FindByID looks a user up by id.
func (r *Users) FindByID(ctx context.Context, id int64) (domain.User, error) {
	return r.find(ctx, selectUser+` WHERE id = $1`, id)
}

func (r *Users) find(ctx context.Context, query string, arg any) (domain.User, error) {
	var (
		user domain.User
		role string
	)

	err := r.pool.QueryRow(ctx, query, arg).Scan(&user.ID, &user.Email, &user.PasswordHash, &role, &user.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, identity.ErrUserNotFound
	}
	if err != nil {
		return domain.User{}, fmt.Errorf("postgres: select user: %w", err)
	}

	// The CHECK constraint keeps the column to the known roles, so a failure here
	// means the schema and the domain drifted apart.
	if user.Role, err = domain.ParseRole(role); err != nil {
		return domain.User{}, fmt.Errorf("postgres: stored role %q: %w", role, err)
	}

	return user, nil
}
