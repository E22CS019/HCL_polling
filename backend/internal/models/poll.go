package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// PollOption is a single answer choice on a poll.
type PollOption struct {
	ID    string `bson:"id"    json:"id"`
	Text  string `bson:"text"  json:"text"`
	Votes int64  `bson:"votes" json:"votes"`
}

// Poll is the core document stored in MongoDB.
type Poll struct {
	ID           primitive.ObjectID `bson:"_id,omitempty"   json:"id"`
	CreatorID    primitive.ObjectID `bson:"creator_id"      json:"creator_id"`
	Title        string             `bson:"title"           json:"title"`
	Description  string             `bson:"description"     json:"description"`
	Options      []PollOption       `bson:"options"         json:"options"`
	MultiChoice  bool               `bson:"multi_choice"    json:"multi_choice"`
	EndsAt       *time.Time         `bson:"ends_at"         json:"ends_at"`
	CreatedAt    time.Time          `bson:"created_at"      json:"created_at"`
	TotalVotes   int64              `bson:"total_votes"     json:"total_votes"`
	Closed       bool               `bson:"closed"          json:"closed"`
}

// CreatePollRequest is the validated payload for POST /polls.
type CreatePollRequest struct {
	Title       string     `json:"title"        binding:"required,min=3,max=200"`
	Description string     `json:"description"  binding:"max=1000"`
	Options     []string   `json:"options"      binding:"required,min=2,max=10"`
	MultiChoice bool       `json:"multi_choice"`
	EndsAt      *time.Time `json:"ends_at"`
}

// VoteRequest is the validated payload for POST /polls/:id/vote.
type VoteRequest struct {
	OptionIDs []string `json:"option_ids" binding:"required,min=1"`
}

// PollResult carries the live vote counts for SSE and API responses.
type PollResult struct {
	PollID     string             `json:"poll_id"`
	TotalVotes int64              `json:"total_votes"`
	Options    []OptionResult     `json:"options"`
	UpdatedAt  time.Time          `json:"updated_at"`
}

// OptionResult is a single option's live tally.
type OptionResult struct {
	ID         string  `json:"id"`
	Text       string  `json:"text"`
	Votes      int64   `json:"votes"`
	Percentage float64 `json:"percentage"`
}

// VoterRecord tracks which options a voter (by fingerprint) has already chosen.
type VoterRecord struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	PollID    primitive.ObjectID `bson:"poll_id"`
	Fingerprint string           `bson:"fingerprint"`
	OptionIDs []string           `bson:"option_ids"`
	VotedAt   time.Time          `bson:"voted_at"`
}
