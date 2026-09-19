package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"pollster-backend/internal/middleware"
	"pollster-backend/internal/models"
	"pollster-backend/internal/services"
	"pollster-backend/internal/sse"

	"github.com/gin-gonic/gin"
)

// PollHandler exposes HTTP endpoints for poll management, voting, and SSE streaming.
type PollHandler struct {
	pollSvc *services.PollService
	broker  *sse.Broker
}

// NewPollHandler constructs a PollHandler.
func NewPollHandler(pollSvc *services.PollService, broker *sse.Broker) *PollHandler {
	return &PollHandler{pollSvc: pollSvc, broker: broker}
}

// CreatePoll handles POST /polls
func (h *PollHandler) CreatePoll(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req models.CreatePollRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	poll, err := h.pollSvc.CreatePoll(c.Request.Context(), userID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, poll)
}

// GetPoll handles GET /polls/:id
func (h *PollHandler) GetPoll(c *gin.Context) {
	poll, err := h.pollSvc.GetPoll(c.Request.Context(), c.Param("id"))
	if err != nil {
		if errors.Is(err, services.ErrPollNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "poll not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve poll"})
		return
	}
	c.JSON(http.StatusOK, poll)
}

// GetMyPolls handles GET /polls/mine
func (h *PollHandler) GetMyPolls(c *gin.Context) {
	userID := middleware.GetUserID(c)
	polls, err := h.pollSvc.GetMyPolls(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve polls"})
		return
	}
	if polls == nil {
		polls = []models.Poll{}
	}
	c.JSON(http.StatusOK, polls)
}

// ListRecent handles GET /polls
func (h *PollHandler) ListRecent(c *gin.Context) {
	polls, err := h.pollSvc.ListRecent(c.Request.Context(), 20)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list polls"})
		return
	}
	if polls == nil {
		polls = []models.Poll{}
	}
	c.JSON(http.StatusOK, polls)
}

// Vote handles POST /polls/:id/vote
func (h *PollHandler) Vote(c *gin.Context) {
	pollID := c.Param("id")

	var req models.VoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	fingerprint := c.GetHeader("X-Voter-Fingerprint")
	if fingerprint == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Voter-Fingerprint header required"})
		return
	}
	if len(fingerprint) > 128 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "fingerprint too long"})
		return
	}

	result, err := h.pollSvc.Vote(c.Request.Context(), pollID, fingerprint, req.OptionIDs)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrPollNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "poll not found"})
		case errors.Is(err, services.ErrAlreadyVoted):
			c.JSON(http.StatusConflict, gin.H{"error": "you have already voted on this poll"})
		case errors.Is(err, services.ErrPollClosed):
			c.JSON(http.StatusGone, gin.H{"error": "this poll is closed"})
		case errors.Is(err, services.ErrInvalidOption):
			c.JSON(http.StatusBadRequest, gin.H{"error": "one or more invalid option IDs"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "vote failed"})
		}
		return
	}

	c.JSON(http.StatusOK, result)
}

// ClosePoll handles POST /polls/:id/close
func (h *PollHandler) ClosePoll(c *gin.Context) {
	userID := middleware.GetUserID(c)
	err := h.pollSvc.ClosePoll(c.Request.Context(), c.Param("id"), userID)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrPollNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "poll not found"})
		case errors.Is(err, services.ErrForbidden):
			c.JSON(http.StatusForbidden, gin.H{"error": "only the poll creator can close it"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "close failed"})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "poll closed"})
}

// GetResults handles GET /polls/:id/results
func (h *PollHandler) GetResults(c *gin.Context) {
	result, err := h.pollSvc.GetResults(c.Request.Context(), c.Param("id"))
	if err != nil {
		if errors.Is(err, services.ErrPollNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "poll not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get results"})
		return
	}
	c.JSON(http.StatusOK, result)
}

// CheckVoteStatus handles GET /polls/:id/vote-status
func (h *PollHandler) CheckVoteStatus(c *gin.Context) {
	fingerprint := c.GetHeader("X-Voter-Fingerprint")
	if fingerprint == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Voter-Fingerprint header required"})
		return
	}

	rec, err := h.pollSvc.GetVoterRecord(c.Request.Context(), c.Param("id"), fingerprint)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "status check failed"})
		return
	}

	if rec == nil {
		c.JSON(http.StatusOK, gin.H{"voted": false, "option_ids": []string{}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"voted": true, "option_ids": rec.OptionIDs})
}

// StreamResults handles GET /polls/:id/stream — SSE endpoint.
// The client receives a live "result" event every time a new vote is cast.
func (h *PollHandler) StreamResults(c *gin.Context) {
	pollID := c.Param("id")

	// Verify poll exists before opening the SSE stream.
	if _, err := h.pollSvc.GetPoll(c.Request.Context(), pollID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "poll not found"})
		return
	}

	client := h.broker.Subscribe(pollID)
	defer h.broker.Unsubscribe(client)

	// Set SSE headers.
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no") // Disable Nginx buffering

	// Send an initial snapshot so the client has data while waiting for the first vote.
	if result, err := h.pollSvc.GetResults(c.Request.Context(), pollID); err == nil {
		writeSSEEvent(c, "result", result)
	}

	// Heartbeat ticker to keep the connection alive through proxies.
	ticker := time.NewTicker(25 * time.Second)
	defer ticker.Stop()

	notify := c.Request.Context().Done()
	for {
		select {
		case <-notify:
			return
		case <-ticker.C:
			fmt.Fprintf(c.Writer, ": heartbeat\n\n")
			c.Writer.Flush()
		case data, ok := <-client.Send:
			if !ok {
				return
			}
			fmt.Fprintf(c.Writer, "event: result\ndata: %s\n\n", data)
			c.Writer.Flush()
		}
	}
}

// writeSSEEvent serialises data as JSON and writes a named SSE event.
func writeSSEEvent(c *gin.Context, event string, data interface{}) {
	b, err := json.Marshal(data)
	if err != nil {
		return
	}
	fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", event, b)
	c.Writer.Flush()
}
