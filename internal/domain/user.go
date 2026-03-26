package domain

import (
	"errors"
	"time"
)

type UserStatus string

const (
	UserStatusActive   UserStatus = "active"
	UserStatusInactive UserStatus = "inactive"
	UserStatusBanned   UserStatus = "banned"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrInvalidInput      = errors.New("invalid input")
)

type User struct {
	ID           string
	Email        string
	Nickname     string
	FirstName    string
	LastName     string
	Phone        string
	Address      string
	PasswordHash string
	Status       UserStatus
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time
}

func NewUser(id, email, nickname, firstName, lastName, phone, address, passwordHash string) (*User, error) {
	now := time.Now().UTC()
	u := &User{
		ID:           id,
		Email:        email,
		Nickname:     nickname,
		FirstName:    firstName,
		LastName:     lastName,
		Phone:        phone,
		Address:      address,
		PasswordHash: passwordHash,
		Status:       UserStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := u.Validate(); err != nil {
		return nil, err
	}
	return u, nil
}

func (u *User) Validate() error {
	if u.ID == "" || u.Email == "" || u.FirstName == "" || u.LastName == "" {
		return ErrInvalidInput
	}
	return nil
}

func (u *User) Rename(firstName, lastName string) error {
	if firstName == "" || lastName == "" {
		return ErrInvalidInput
	}
	u.FirstName = firstName
	u.LastName = lastName
	u.UpdatedAt = time.Now().UTC()
	return nil
}

func (u *User) UpdatePhone(phone string) {
	u.Phone = phone
	u.UpdatedAt = time.Now().UTC()
}

func (u *User) UpdateAddress(address string) {
	u.Address = address
	u.UpdatedAt = time.Now().UTC()
}

func (u *User) UpdateNickname(nickname string) {
	u.Nickname = nickname
	u.UpdatedAt = time.Now().UTC()
}

func (u *User) Ban() error {
	if u.Status == UserStatusBanned {
		return ErrInvalidInput
	}
	u.Status = UserStatusBanned
	u.UpdatedAt = time.Now().UTC()
	return nil
}

func (u *User) SoftDelete() {
	now := time.Now().UTC()
	u.DeletedAt = &now
	u.UpdatedAt = now
}
