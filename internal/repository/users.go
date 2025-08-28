package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sefa-b/go-banking-sim/internal/domain"
)

// usersRepo implements the UsersRepo interface.
type usersRepo struct {
	db *pgxpool.Pool
}

// NewUsersRepo creates a new users repository.
func NewUsersRepo(db *pgxpool.Pool) UsersRepo {
	return &usersRepo{db: db}
}

// Create creates a new user.
func (r *usersRepo) Create(ctx context.Context, user *domain.User) error {
	query := `
		INSERT INTO users (id, username, email, password_hash, role, created_at, updated_at, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	now := time.Now()
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}
	user.CreatedAt = now
	user.UpdatedAt = now
	user.IsActive = true // New users are active by default

	_, err := r.db.Exec(ctx, query, user.ID, user.Username, user.Email, user.PasswordHash, user.Role, user.CreatedAt, user.UpdatedAt, user.IsActive)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// GetByID retrieves a user by ID.
func (r *usersRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	query := `
		SELECT id, username, email, password_hash, role, created_at, updated_at, is_active
		FROM users
		WHERE id = $1 AND is_active = TRUE`

	var user domain.User
	err := r.db.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.IsActive,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user by ID: %w", err)
	}

	return &user, nil
}

// GetByEmail retrieves a user by email.
func (r *usersRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
		SELECT id, username, email, password_hash, role, created_at, updated_at, is_active
		FROM users
		WHERE email = $1 AND is_active = TRUE`

	var user domain.User
	err := r.db.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.IsActive,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return &user, nil
}

// GetByUsername retrieves a user by username.
func (r *usersRepo) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	query := `
		SELECT id, username, email, password_hash, role, created_at, updated_at, is_active
		FROM users
		WHERE username = $1 AND is_active = TRUE`

	var user domain.User
	err := r.db.QueryRow(ctx, query, username).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.IsActive,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user by username: %w", err)
	}

	return &user, nil
}

// Update updates an existing user.
func (r *usersRepo) Update(ctx context.Context, user *domain.User) error {
	query := `
		UPDATE users
		SET username = $2, email = $3, password_hash = $4, role = $5, updated_at = $6, is_active = $7
		WHERE id = $1`

	user.UpdatedAt = time.Now()

	result, err := r.db.Exec(ctx, query, user.ID, user.Username, user.Email, user.PasswordHash, user.Role, user.UpdatedAt, user.IsActive)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// Delete deletes a user by ID.
func (r *usersRepo) Delete(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE users SET is_active = FALSE, updated_at = NOW() WHERE id = $1 AND is_active = TRUE`

	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to soft delete user: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("user not found or already inactive")
	}

	return nil
}

// ListPaginated retrieves users with pagination.
func (r *usersRepo) ListPaginated(ctx context.Context, limit, offset int) ([]*domain.User, error) {
	baseQuery := `
		SELECT id, username, email, password_hash, role, created_at, updated_at, is_active
		FROM users
		WHERE is_active = TRUE
		ORDER BY created_at DESC`

	queryArgs := []interface{}{}
	paramCount := 0

	if limit > 0 {
		baseQuery += fmt.Sprintf(" LIMIT $%d", paramCount+1)
		queryArgs = append(queryArgs, limit)
		paramCount++
	}

	baseQuery += fmt.Sprintf(" OFFSET $%d", paramCount+1)
	queryArgs = append(queryArgs, offset)

	rows, err := r.db.Query(ctx, baseQuery, queryArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	var users []*domain.User
	for rows.Next() {
		var user domain.User
		err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.Email,
			&user.PasswordHash,
			&user.Role,
			&user.CreatedAt,
			&user.UpdatedAt,
			&user.IsActive,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, &user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate users: %w", err)
	}

	return users, nil
}

// Count returns the total number of users.
func (r *usersRepo) Count(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM users WHERE is_active = TRUE`

	var count int
	err := r.db.QueryRow(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count users: %w", err)
	}

	return count, nil
}

// ListAll retrieves all users without pagination (for testing purposes).
func (r *usersRepo) ListAll(ctx context.Context) ([]*domain.User, error) {
	query := `
		SELECT id, username, email, password_hash, role, created_at, updated_at, is_active
		FROM users
		WHERE is_active = TRUE
		ORDER BY created_at DESC`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list all users: %w", err)
	}
	defer rows.Close()

	var users []*domain.User
	for rows.Next() {
		var user domain.User
		err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.Email,
			&user.PasswordHash,
			&user.Role,
			&user.CreatedAt,
			&user.UpdatedAt,
			&user.IsActive,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, &user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate users: %w", err)
	}

	return users, nil
}
