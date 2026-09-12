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
		u.id,
		u.first_name,
		u.last_name,
		u.email,
		u.password,
		u.role_id,
		u.created_at,
		u.updated_at,
		r.id,
		r.name,
		r.level,
		r.description
		FROM users u
	JOIN roles r ON u.role_id = r.id
	WHERE u.id = $1`
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()
	var user User
	var lastName sql.NullString
	err := s.db.QueryRowContext(ctx, q, id).Scan(
		&user.Id,
		&user.FirstName,
		&lastName,
		&user.Email,
		&user.Password.hash,
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
			return nil, ErrResourceNotFound
		default:
			return nil, err
		}
	}
	if lastName.Valid {
		user.LastName = lastName.String
	}
	return &user, nil
}

func (s *UserStore) Create(ctx context.Context, u *User) error {
	q := `INSERT INTO users(
		first_name, last_name, email, role_id, password
	) VALUES(
		$1, 
		$2, 
		$3, 
		(SELECT id FROM roles WHERE name = $4), 
		$5
	) RETURNING users.id, created_at, updated_at`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()
	err := s.db.QueryRowContext(
		ctx,
		q,
		nullString(u.FirstName),
		nullString(u.LastName),
		nullString(u.Email),
		u.Role.Name,
		nullString(string(u.Password.hash)),
	).Scan(
		&u.Id,
		&u.CreateAt,
		&u.UpdatedAt,
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

func (s *UserStore) GetByEmail(ctx context.Context, email string) (*User, error) {
	q := `SELECT 
		u.id,
		u.first_name,
		u.last_name,
		u.email,
		u.password,
		u.role_id,
		u.created_at,
		u.updated_at,
		r.id,
		r.name,
		r.level,
		r.description
		FROM users u
	JOIN roles r ON u.role_id = r.id
	WHERE u.email = $1`
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()
	var user User
	var lastName sql.NullString
	err := s.db.QueryRowContext(ctx, q, email).Scan(
		&user.Id,
		&user.FirstName,
		&lastName,
		&user.Email,
		&user.Password.hash,
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
			return nil, ErrResourceNotFound
		default:
			return nil, err
		}
	}
	if lastName.Valid {
		user.LastName = lastName.String
	}
	return &user, nil
}
