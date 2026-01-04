package admin

import "github.com/pkg/errors"

type ConfirmUserRequest struct {
	Username string `json:"username"`
}

func (r *ConfirmUserRequest) Validate() error {
	if r.Username == "" {
		return errors.New("username is required")
	}
	return nil
}
