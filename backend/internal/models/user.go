package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// User represents a registered account that can create and manage polls.
type User struct {
	ID           primitive.ObjectID `bson:"_id,omitempty"   json:"id"`
	Username     string             `bson:"username"        json:"username"`
	Email        string             `bson:"email"           json:"email"`
	PasswordHash string             `bson:"password_hash"   json:"-"`
	CreatedAt    time.Time          `bson:"created_at"      json:"created_at"`
}

// RegisterRequest is the validated payload for POST /auth/register.
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=30"`
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

// LoginRequest is the validated payload for POST /auth/login.
type LoginRequest struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// AuthResponse is returned after a successful login or registration.
type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}
