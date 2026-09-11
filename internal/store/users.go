package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)


var (
	ErrUserNotFound   = errors.New("user not found")
	ErrDuplicateEmail = errors.New("a user with that email already exists")
)


type UserStore struct {
	db *sql.DB
}

type User struct {
	Id        uuid.UUID `json:"id"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Email 	 string 	`json:"email"`
	Password  password  `json:"-"`
	CreateAt  time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	RoleId    int 		`json:"role_id"`
	Role 	  Role   	`json:"role"`
}

type password struct {
	text *string
	hash []byte
}

func (p *password) Set(text string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(text), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	p.text = &text
	p.hash = hash
	return nil
}

func (p *password) Compare(text string) error {
	return bcrypt.CompareHashAndPassword(p.hash, []byte(text))
}

func (s *UserStore) GetById(ctx context.Context, id string) (*User, error) {
	q := `SELECT 
			u.*, 
			r.* 
		FROM users u JOIN roles r ON u.role_id = r.id WHERE id = $1`
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()
	var user User
	err := s.db.QueryRowContext(ctx, q, id).Scan(
		&user.Id,
		&user.FirstName,
		&user.LastName,
		&user.Email,
		&user.Password,
		&user.RoleId,
		&user.CreateAt,
		&user.UpdatedAt,
		&user.Role.ID,
		&user.Role.Name,
		&user.Role.Level,
		&user.Role.Description,
	)
	if err != nil {
		switch err {
		case sql.ErrNoRows:
			return nil, ErrUserNotFound
		default:
			return nil, err
		}
	}
	return &user, nil
}

func (s *UserStore) Create(ctx context.Context, u *User) error {
	q := `INSER INTO users(
		first_name, last_name, email, role_id, password
	) VALUES($1, $2, $3, $4, $5) RETURNING *`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()
	err := s.db.QueryRowContext(
		ctx,
		q,
		nullString(u.FirstName),
		nullString(u.LastName),
		nullString(u.Email),
		u.RoleId,
		nullString(string(u.Password.hash)),
	).Scan(
		&u.Id,
	)
	if err != nil {
		switch {
		case err.Error() == `pq: duplicate key value violates unique constraint "users_email_key"`:
			return ErrDuplicateEmail
		default:
			return err
		}
	}

	return nil
}


// CREATE TABLE IF NOT EXISTS roles (
//   id BIGSERIAL PRIMARY KEY,
//   name VARCHAR(255) NOT NULL UNIQUE,
//   level int NOT NULL DEFAULT 0,
//   description TEXT
// );

// CREATE TABLE users(
//     id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
//     first_name VARCHAR(255) NOT NULL,
//     last_name VARCHAR(255),
//     email VARCHAR(255) NOT NULL,
//     password VARCHAR(255) NOT NULL,
//     role_id INT REFERENCES roles(id) DEFAULT 1,
//     created_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),
//     updated_at timestamp(0) with time zone NOT NULL DEFAULT NOW()
// );

func (s *UserStore) GetByEmail(ctx context.Context, email string) (*User, error) {
	q := `SELECT 
			u.*, 
			r.* 
		FROM users u JOIN roles r ON u.role_id = r.id WHERE email = $1`
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()
	var user User
	err := s.db.QueryRowContext(ctx, q, email).Scan(
		&user.Id,
		&user.FirstName,
		&user.LastName,
		&user.Email,
		&user.Password,
		&user.RoleId,
		&user.CreateAt,
		&user.UpdatedAt,
		&user.Role.ID,
		&user.Role.Name,
		&user.Role.Level,
		&user.Role.Description,
	)
	if err != nil {
		switch err {
		case sql.ErrNoRows:
			return nil, ErrUserNotFound
		default:
			return nil, err
		}
	}
	return &user, nil
}
