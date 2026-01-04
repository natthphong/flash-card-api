package auth

import (
	"context"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pkg/errors"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/utils"
	"golang.org/x/crypto/bcrypt"
	"time"
)

func GenerateJWTForUser(
	ctx context.Context,
	db *pgxpool.Pool,
	username, password string,
	jwtSecret string,
	accessTokenDuration, refreshTokenDuration time.Duration,
	refreshTokenFlag bool,
) (map[string]interface{}, error) {
	var user UserDtoModel
	query := `
			select id,username,name,password,status,email from tbl_users
		where username = $1 and is_deleted='N'
		`
	err := db.QueryRow(ctx, query, username).Scan(
		&user.Id, &user.Username, &user.Name, &user.Password, &user.Status, &user.Email,
	)
	if err != nil {
		return nil, errors.New("User not found")
	}

	if !refreshTokenFlag {
		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
			return nil, errors.New("Invalid password")
		}
	}

	if user.Status == utils.Pending {
		return nil, errors.New("User is pending")
	}
	if user.Status == utils.Rejected {
		return nil, errors.New("User is rejected")
	}
	if user.Status == utils.WaitEmail {
		return nil, errors.New("User is waiting for email")
	}
	if user.Status != utils.APPROVED {
		return nil, errors.New("User is not approved")
	}

	accessTokenClaims := jwt.MapClaims{
		"id":       user.Id,
		"name":     user.Name,
		"username": user.Username,
		"email":    user.Email,
		"exp":      time.Now().Add(accessTokenDuration).Unix(),
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessTokenClaims)
	accessTokenString, err := accessToken.SignedString([]byte(jwtSecret))
	if err != nil {
		return nil, errors.New("Failed to generate access token")
	}

	// Refresh Token
	refreshTokenClaims := jwt.MapClaims{
		"userId":   user.Id,
		"username": user.Username,
		"exp":      time.Now().Add(refreshTokenDuration).Unix(),
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshTokenClaims)
	refreshTokenString, err := refreshToken.SignedString([]byte(jwtSecret))
	if err != nil {
		return nil, errors.New("Failed to generate refresh token")
	}

	// Response
	response := map[string]interface{}{
		"accessToken":  accessTokenString,
		"refreshToken": refreshTokenString,
		"jwtBody":      accessTokenClaims,
	}
	return response, nil
}
