package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"pollster-backend/internal/models"
	"pollster-backend/internal/repository"
	"pollster-backend/internal/sse"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ErrPollNotFound is returned when a poll ID does not exist.
var ErrPollNotFound = errors.New("poll not found")

// ErrAlreadyVoted is returned when the fingerprint has already voted.
var ErrAlreadyVoted = errors.New("already voted on this poll")

// ErrPollClosed is returned when voting on a closed/expired poll.
var ErrPollClosed = errors.New("poll is closed")

// ErrForbidden is returned when an operation is not allowed for the caller.
var ErrForbidden = errors.New("forbidden")

// ErrInvalidOption is returned when an unrecognised option ID is submitted.
var ErrInvalidOption = errors.New("invalid option")

// PollService handles business logic for poll creation, voting, and retrieval.
type PollService struct {
	polls  *repository.PollRepository
	rdb    *redis.Client
	broker *sse.Broker
}

// NewPollService constructs a PollService.
func NewPollService(polls *repository.PollRepository, rdb *redis.Client, broker *sse.Broker) *PollService {
	return &PollService{polls: polls, rdb: rdb, broker: broker}
}

// CreatePoll validates and persists a new poll.
func (s *PollService) CreatePoll(ctx context.Context, creatorID primitive.ObjectID, req models.CreatePollRequest) (*models.Poll, error) {
	if len(req.Options) < 2 {
		return nil, fmt.Errorf("at least 2 options required")
	}

	// Deduplicate option texts.
	seen := make(map[string]struct{}, len(req.Options))
	options := make([]models.PollOption, 0, len(req.Options))
	for _, text := range req.Options {
		if _, dup := seen[text]; dup {
			return nil, fmt.Errorf("duplicate option text: %q", text)
		}
		seen[text] = struct{}{}
		options = append(options, models.PollOption{
			ID:    primitive.NewObjectID().Hex(),
			Text:  text,
			Votes: 0,
		})
	}

	poll := &models.Poll{
		CreatorID:   creatorID,
		Title:       req.Title,
		Description: req.Description,
		Options:     options,
		MultiChoice: req.MultiChoice,
		EndsAt:      req.EndsAt,
		Closed:      false,
	}

	if err := s.polls.Create(ctx, poll); err != nil {
		return nil, fmt.Errorf("creating poll: %w", err)
	}

	// Seed Redis counters to 0 for each option.
	// Key pattern: poll:<id>:votes:<optionID>
	pipe := s.rdb.Pipeline()
	for _, opt := range poll.Options {
		pipe.Set(ctx, redisVoteKey(poll.ID.Hex(), opt.ID), 0, 0)
	}
	pipe.Set(ctx, redisTotalKey(poll.ID.Hex()), 0, 0)
	if _, err := pipe.Exec(ctx); err != nil {
		return nil, fmt.Errorf("seeding redis counters: %w", err)
	}

	return poll, nil
}

// GetPoll returns a poll with live Redis vote counts merged in.
func (s *PollService) GetPoll(ctx context.Context, pollID string) (*models.Poll, error) {
	oid, err := primitive.ObjectIDFromHex(pollID)
	if err != nil {
		return nil, ErrPollNotFound
	}

	poll, err := s.polls.FindByID(ctx, oid)
	if err != nil || poll == nil {
		return nil, ErrPollNotFound
	}

	// Auto-close if the end time has passed.
	if poll.EndsAt != nil && time.Now().After(*poll.EndsAt) && !poll.Closed {
		_ = s.polls.ClosePoll(ctx, poll.ID)
		poll.Closed = true
	}

	// Merge live Redis counts into the poll options.
	s.mergeRedisCounts(ctx, poll)

	return poll, nil
}

// GetMyPolls returns all polls created by a user.
func (s *PollService) GetMyPolls(ctx context.Context, creatorID primitive.ObjectID) ([]models.Poll, error) {
	polls, err := s.polls.FindByCreator(ctx, creatorID)
	if err != nil {
		return nil, err
	}
	for i := range polls {
		s.mergeRedisCounts(ctx, &polls[i])
	}
	return polls, nil
}

// Vote records a vote, updating MongoDB and Redis atomically (at the service level),
// then publishes a live result update to the SSE broker.
func (s *PollService) Vote(ctx context.Context, pollID string, fingerprint string, optionIDs []string) (*models.PollResult, error) {
	oid, err := primitive.ObjectIDFromHex(pollID)
	if err != nil {
		return nil, ErrPollNotFound
	}

	poll, err := s.polls.FindByID(ctx, oid)
	if err != nil || poll == nil {
		return nil, ErrPollNotFound
	}

	// Check if poll is closed or expired.
	if poll.Closed || (poll.EndsAt != nil && time.Now().After(*poll.EndsAt)) {
		return nil, ErrPollClosed
	}

	// Validate all submitted option IDs exist on this poll.
	validOptions := make(map[string]bool, len(poll.Options))
	for _, opt := range poll.Options {
		validOptions[opt.ID] = true
	}
	for _, oid2 := range optionIDs {
		if !validOptions[oid2] {
			return nil, ErrInvalidOption
		}
	}

	// Enforce single-choice constraint.
	if !poll.MultiChoice && len(optionIDs) > 1 {
		optionIDs = optionIDs[:1]
	}

	// Idempotency check — one vote per fingerprint.
	voted, err := s.polls.HasVoted(ctx, oid, fingerprint)
	if err != nil {
		return nil, fmt.Errorf("checking vote status: %w", err)
	}
	if voted {
		return nil, ErrAlreadyVoted
	}

	// Record voter to MongoDB first (provides the uniqueness guarantee).
	voterRec := &models.VoterRecord{
		PollID:      oid,
		Fingerprint: fingerprint,
		OptionIDs:   optionIDs,
	}
	if err := s.polls.RecordVote(ctx, voterRec); err != nil {
		// Duplicate key from a race condition.
		return nil, ErrAlreadyVoted
	}

	// Increment counters in Redis — fast path for live display.
	pipe := s.rdb.Pipeline()
	for _, optID := range optionIDs {
		pipe.Incr(ctx, redisVoteKey(pollID, optID))
	}
	pipe.Incr(ctx, redisTotalKey(pollID))
	if _, err := pipe.Exec(ctx); err != nil {
		return nil, fmt.Errorf("incrementing redis counters: %w", err)
	}

	// Also persist to MongoDB for durability.
	updatedPoll, err := s.polls.IncrementVotes(ctx, oid, optionIDs)
	if err != nil {
		return nil, fmt.Errorf("persisting vote to mongo: %w", err)
	}

	// Build a PollResult from Redis (always-fresh counts).
	result := s.buildResult(ctx, updatedPoll)

	// Publish via Redis Pub/Sub → broker fans out to all SSE clients.
	if err := s.broker.Publish(ctx, pollID, result); err != nil {
		// Non-fatal: the vote is already recorded; just log.
		fmt.Printf("warn: publish to broker failed: %v\n", err)
	}

	return result, nil
}

// ClosePoll marks a poll as closed (only the creator may do this).
func (s *PollService) ClosePoll(ctx context.Context, pollID string, requesterID primitive.ObjectID) error {
	oid, err := primitive.ObjectIDFromHex(pollID)
	if err != nil {
		return ErrPollNotFound
	}

	poll, err := s.polls.FindByID(ctx, oid)
	if err != nil || poll == nil {
		return ErrPollNotFound
	}
	if poll.CreatorID != requesterID {
		return ErrForbidden
	}

	return s.polls.ClosePoll(ctx, oid)
}

// GetResults returns a PollResult populated from Redis counts (falls back to Mongo).
func (s *PollService) GetResults(ctx context.Context, pollID string) (*models.PollResult, error) {
	poll, err := s.GetPoll(ctx, pollID)
	if err != nil {
		return nil, err
	}
	return s.buildResult(ctx, poll), nil
}

// GetVoterRecord returns the voting record for a given fingerprint on a poll.
func (s *PollService) GetVoterRecord(ctx context.Context, pollID string, fingerprint string) (*models.VoterRecord, error) {
	oid, err := primitive.ObjectIDFromHex(pollID)
	if err != nil {
		return nil, ErrPollNotFound
	}
	return s.polls.GetVoterRecord(ctx, oid, fingerprint)
}

// ListRecent returns the most recent polls.
func (s *PollService) ListRecent(ctx context.Context, limit int) ([]models.Poll, error) {
	polls, err := s.polls.ListRecent(ctx, limit)
	if err != nil {
		return nil, err
	}
	for i := range polls {
		s.mergeRedisCounts(ctx, &polls[i])
	}
	return polls, nil
}

// ----- helpers -----

func redisVoteKey(pollID, optionID string) string {
	return fmt.Sprintf("poll:%s:votes:%s", pollID, optionID)
}

func redisTotalKey(pollID string) string {
	return fmt.Sprintf("poll:%s:total", pollID)
}

// mergeRedisCounts overwrites the vote counts in a poll's options with the
// live Redis values. Falls back silently to MongoDB counts if Redis is unavailable.
func (s *PollService) mergeRedisCounts(ctx context.Context, poll *models.Poll) {
	for i, opt := range poll.Options {
		key := redisVoteKey(poll.ID.Hex(), opt.ID)
		val, err := s.rdb.Get(ctx, key).Int64()
		if err == nil {
			poll.Options[i].Votes = val
		}
	}
	totalKey := redisTotalKey(poll.ID.Hex())
	total, err := s.rdb.Get(ctx, totalKey).Int64()
	if err == nil {
		poll.TotalVotes = total
	}
}

// buildResult constructs a PollResult with percentage calculations.
func (s *PollService) buildResult(ctx context.Context, poll *models.Poll) *models.PollResult {
	s.mergeRedisCounts(ctx, poll)

	opts := make([]models.OptionResult, len(poll.Options))
	for i, opt := range poll.Options {
		pct := 0.0
		if poll.TotalVotes > 0 {
			pct = float64(opt.Votes) / float64(poll.TotalVotes) * 100
		}
		opts[i] = models.OptionResult{
			ID:         opt.ID,
			Text:       opt.Text,
			Votes:      opt.Votes,
			Percentage: pct,
		}
	}
	return &models.PollResult{
		PollID:     poll.ID.Hex(),
		TotalVotes: poll.TotalVotes,
		Options:    opts,
		UpdatedAt:  time.Now(),
	}
}
