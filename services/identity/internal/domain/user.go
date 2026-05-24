package domain

import (
	"context"
	"errors"
)

type Role string

const (
	RoleRider  Role = "rider"
	RoleDriver Role = "driver"
)

// ErrDuplicateEmail is returned when a user with the same email already exists.
var ErrDuplicateEmail = errors.New("duplicate email")

// ErrUserNotFound is returned when a user cannot be found.
var ErrUserNotFound = errors.New("user not found")

type User struct {
	ID    string
	Email string
	Role  Role
}

type UserRepository interface {
	Create(ctx context.Context, u *User) error
	GetByID(ctx context.Context, id string) (*User, error)
}
