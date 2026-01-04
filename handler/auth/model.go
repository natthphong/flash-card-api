package auth

import "github.com/pkg/errors"

type RegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Name     string `json:"name"`
	Email    string `json:"email"`
}

func (r RegisterRequest) Validate() error {
	if r.Username == "" {
		return errors.New("username is required")
	}
	if r.Password == "" {
		return errors.New("password is required")
	}
	if r.Name == "" {
		return errors.New("name is required")
	}
	if r.Email == "" {
		return errors.New("email is required")
	}
	return nil
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (r LoginRequest) Validate() error {
	if r.Username == "" {
		return errors.New("username is required")
	}
	if r.Password == "" {
		return errors.New("password is required")
	}
	return nil
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refreshToken"` // Refresh token from the client
}

// UserDtoModel dto
type UserDtoModel struct {
	Id       string `json:"id"`
	Username string `json:"username"`
	Password string `json:"password"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Status   string `json:"status"`
}
