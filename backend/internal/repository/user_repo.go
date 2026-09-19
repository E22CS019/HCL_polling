package repository

import (
	"context"
	"errors"
	"time"

	"pollster-backend/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// UserRepository handles all MongoDB operations for users.
type UserRepository struct {
	col *mongo.Collection
}

// NewUserRepository creates a UserRepository and ensures required indexes exist.
func NewUserRepository(db *mongo.Database) (*UserRepository, error) {
	col := db.Collection("users")

	// Unique index on email for fast lookups and duplicate prevention.
	idx := mongo.IndexModel{
		Keys:    bson.D{{Key: "email", Value: 1}},
		Options: options.Index().SetUnique(true),
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if _, err := col.Indexes().CreateOne(ctx, idx); err != nil {
		return nil, err
	}

	// Unique index on username.
	idxU := mongo.IndexModel{
		Keys:    bson.D{{Key: "username", Value: 1}},
		Options: options.Index().SetUnique(true),
	}
	if _, err := col.Indexes().CreateOne(ctx, idxU); err != nil {
		return nil, err
	}

	return &UserRepository{col: col}, nil
}

// Create inserts a new user document.
func (r *UserRepository) Create(ctx context.Context, u *models.User) error {
	u.ID = primitive.NewObjectID()
	u.CreatedAt = time.Now()
	_, err := r.col.InsertOne(ctx, u)
	return err
}

// FindByEmail looks up a user by email address.
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	var u models.User
	err := r.col.FindOne(ctx, bson.M{"email": email}).Decode(&u)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	return &u, err
}

// FindByID looks up a user by their ObjectID.
func (r *UserRepository) FindByID(ctx context.Context, id primitive.ObjectID) (*models.User, error) {
	var u models.User
	err := r.col.FindOne(ctx, bson.M{"_id": id}).Decode(&u)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	return &u, err
}
