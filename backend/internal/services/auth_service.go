package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"pollster-backend/internal/models"
	"pollster-backend/internal/repository"

	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

// ErrInvalidCredentials is returned when email/password don't match.
var ErrInvalidCredentials = errors.New("invalid credentials")

// ErrUserExists is returned when email or username is already taken.
var ErrUserExists = errors.New("user already exists")

// AuthService handles registration, login, and token validation.
type AuthService struct {
	users      *repository.UserRepository
	jwtSecret  []byte
	expiryHours int
}

// NewAuthService constructs an AuthService.
func NewAuthService(users *repository.UserRepository, secret string, expiryHours int) *AuthService {
	return &AuthService{
		users:       users,
		jwtSecret:   []byte(secret),
		expiryHours: expiryHours,
	}
}

// Register creates a new user account and returns a signed JWT.
func (s *AuthService) Register(ctx context.Context, req models.RegisterRequest) (*models.AuthResponse, error) {
	existing, err := s.users.FindByEmail(ctx, req.Email)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrUserExists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hashing password: %w", err)
	}

	user := &models.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: string(hash),
	}
	if err := s.users.Create(ctx, user); err != nil {
		return nil, ErrUserExists // duplicate username hits the unique index
	}

	token, err := s.signToken(user.ID)
	if err != nil {
		return nil, err
	}
	return &models.AuthResponse{Token: token, User: *user}, nil
}

// Login validates credentials and returns a signed JWT.
func (s *AuthService) Login(ctx context.Context, req models.LoginRequest) (*models.AuthResponse, error) {
	user, err := s.users.FindByEmail(ctx, req.Email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	token, err := s.signToken(user.ID)
	if err != nil {
		return nil, err
	}
	return &models.AuthResponse{Token: token, User: *user}, nil
}

// ValidateToken parses and validates a JWT, returning the user ID claim.
func (s *AuthService) ValidateToken(tokenStr string) (primitive.ObjectID, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.jwtSecret, nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil || !token.Valid {
		return primitive.NilObjectID, errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return primitive.NilObjectID, errors.New("invalid claims")
	}

	sub, err := claims.GetSubject()
	if err != nil {
		return primitive.NilObjectID, errors.New("missing subject claim")
	}

	return primitive.ObjectIDFromHex(sub)
}

// GetUserByID retrieves a user by their ObjectID (used by /auth/me).
func (s *AuthService) GetUserByID(ctx context.Context, id primitive.ObjectID) (*models.User, error) {
	return s.users.FindByID(ctx, id)
}

func (s *AuthService) signToken(userID primitive.ObjectID) (string, error) {
	claims := jwt.RegisteredClaims{
		Subject:   userID.Hex(),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(s.expiryHours) * time.Hour)),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.jwtSecret)
}
