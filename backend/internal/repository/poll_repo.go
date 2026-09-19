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

// PollRepository handles all MongoDB operations for polls and voter records.
type PollRepository struct {
	polls   *mongo.Collection
	voters  *mongo.Collection
}

// NewPollRepository creates indexes and returns a PollRepository.
func NewPollRepository(db *mongo.Database) (*PollRepository, error) {
	polls := db.Collection("polls")
	voters := db.Collection("voters")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Index polls by creator for the "my polls" listing.
	_, err := polls.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "creator_id", Value: 1}},
	})
	if err != nil {
		return nil, err
	}

	// Compound unique index prevents the same fingerprint from voting twice on the same poll.
	_, err = voters.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "poll_id", Value: 1}, {Key: "fingerprint", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return nil, err
	}

	return &PollRepository{polls: polls, voters: voters}, nil
}

// Create inserts a new poll document.
func (r *PollRepository) Create(ctx context.Context, p *models.Poll) error {
	p.ID = primitive.NewObjectID()
	p.CreatedAt = time.Now()
	_, err := r.polls.InsertOne(ctx, p)
	return err
}

// FindByID returns a poll by its ObjectID.
func (r *PollRepository) FindByID(ctx context.Context, id primitive.ObjectID) (*models.Poll, error) {
	var p models.Poll
	err := r.polls.FindOne(ctx, bson.M{"_id": id}).Decode(&p)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	return &p, err
}

// FindByCreator returns all polls created by a user, newest first.
func (r *PollRepository) FindByCreator(ctx context.Context, creatorID primitive.ObjectID) ([]models.Poll, error) {
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	cur, err := r.polls.Find(ctx, bson.M{"creator_id": creatorID}, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var polls []models.Poll
	if err := cur.All(ctx, &polls); err != nil {
		return nil, err
	}
	return polls, nil
}

// IncrementVotes atomically increments vote counts for the given option IDs
// and the poll-level total, returning the updated poll.
func (r *PollRepository) IncrementVotes(ctx context.Context, pollID primitive.ObjectID, optionIDs []string) (*models.Poll, error) {
	// Build $inc update: one entry per option and one for total_votes.
	inc := bson.D{{Key: "total_votes", Value: 1}}
	for _, oid := range optionIDs {
		inc = append(inc, bson.E{Key: "options.$[opt" + oid + "].votes", Value: 1})
	}

	// Build array filters so only matching options are updated.
	var arrayFilters []interface{}
	for _, oid := range optionIDs {
		arrayFilters = append(arrayFilters, bson.M{"opt" + oid + ".id": oid})
	}

	opts := options.FindOneAndUpdate().
		SetArrayFilters(options.ArrayFilters{Filters: arrayFilters}).
		SetReturnDocument(options.After)

	var updated models.Poll
	err := r.polls.FindOneAndUpdate(
		ctx,
		bson.M{"_id": pollID},
		bson.M{"$inc": inc},
		opts,
	).Decode(&updated)
	return &updated, err
}

// ClosePoll marks a poll as closed.
func (r *PollRepository) ClosePoll(ctx context.Context, pollID primitive.ObjectID) error {
	_, err := r.polls.UpdateOne(
		ctx,
		bson.M{"_id": pollID},
		bson.M{"$set": bson.M{"closed": true}},
	)
	return err
}

// HasVoted returns true if the fingerprint has already voted on this poll.
func (r *PollRepository) HasVoted(ctx context.Context, pollID primitive.ObjectID, fingerprint string) (bool, error) {
	count, err := r.voters.CountDocuments(ctx, bson.M{
		"poll_id":     pollID,
		"fingerprint": fingerprint,
	})
	return count > 0, err
}

// RecordVote stores the voter record to prevent duplicate votes.
func (r *PollRepository) RecordVote(ctx context.Context, rec *models.VoterRecord) error {
	rec.ID = primitive.NewObjectID()
	rec.VotedAt = time.Now()
	_, err := r.voters.InsertOne(ctx, rec)
	return err
}

// GetVoterRecord returns the voter record for a fingerprint on a poll, or nil if not found.
func (r *PollRepository) GetVoterRecord(ctx context.Context, pollID primitive.ObjectID, fingerprint string) (*models.VoterRecord, error) {
	var rec models.VoterRecord
	err := r.voters.FindOne(ctx, bson.M{
		"poll_id":     pollID,
		"fingerprint": fingerprint,
	}).Decode(&rec)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	return &rec, err
}

// ListRecent returns polls ordered by creation date (for a public feed).
func (r *PollRepository) ListRecent(ctx context.Context, limit int) ([]models.Poll, error) {
	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetLimit(int64(limit))

	cur, err := r.polls.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var polls []models.Poll
	if err := cur.All(ctx, &polls); err != nil {
		return nil, err
	}
	return polls, nil
}
