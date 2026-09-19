// Package sse implements a per-poll Server-Sent Events (SSE) broker.
//
// Architecture:
//   - Each poll has its own topic channel in the broker.
//   - When a vote arrives the vote service publishes to Redis Pub/Sub on
//     channel "poll:<id>:results". The SSE broker subscribes to that Redis
//     channel and fans the message out to every connected HTTP client over
//     their individual response channels.
//   - Using Redis Pub/Sub means multiple backend replicas stay in sync
//     automatically — a vote processed by replica A is still delivered to
//     a browser connected to replica B.
package sse

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/redis/go-redis/v9"
)

// Client represents a single SSE connection.
type Client struct {
	PollID string
	Send   chan []byte
}

// Broker manages SSE subscriptions for all active polls.
type Broker struct {
	mu      sync.RWMutex
	// topics maps pollID -> set of clients.
	topics  map[string]map[*Client]struct{}
	rdb     *redis.Client
	// subs tracks active Redis Pub/Sub subscriptions per pollID.
	subs    map[string]*redis.PubSub
}

// NewBroker creates a Broker and starts the Redis listener loop.
func NewBroker(rdb *redis.Client) *Broker {
	return &Broker{
		topics: make(map[string]map[*Client]struct{}),
		rdb:    rdb,
		subs:   make(map[string]*redis.PubSub),
	}
}

// Subscribe registers a new SSE client for a given poll and returns the client.
// It also ensures a Redis Pub/Sub subscription exists for the poll topic.
func (b *Broker) Subscribe(pollID string) *Client {
	client := &Client{
		PollID: pollID,
		Send:   make(chan []byte, 8),
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if _, ok := b.topics[pollID]; !ok {
		b.topics[pollID] = make(map[*Client]struct{})
	}
	b.topics[pollID][client] = struct{}{}

	// Ensure there is exactly one Redis subscription per poll topic.
	if _, exists := b.subs[pollID]; !exists {
		channel := fmt.Sprintf("poll:%s:results", pollID)
		sub := b.rdb.Subscribe(context.Background(), channel)
		b.subs[pollID] = sub
		go b.listenRedis(pollID, sub)
	}

	return client
}

// Unsubscribe removes a client and cleans up resources if the topic is empty.
func (b *Broker) Unsubscribe(client *Client) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if clients, ok := b.topics[client.PollID]; ok {
		delete(clients, client)
		close(client.Send)

		if len(clients) == 0 {
			delete(b.topics, client.PollID)
			if sub, ok := b.subs[client.PollID]; ok {
				_ = sub.Close()
				delete(b.subs, client.PollID)
			}
		}
	}
}

// Publish sends a result payload to all SSE clients watching a poll.
// It is also responsible for writing the update to Redis Pub/Sub so that
// every replica's broker picks it up.
func (b *Broker) Publish(ctx context.Context, pollID string, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	channel := fmt.Sprintf("poll:%s:results", pollID)
	return b.rdb.Publish(ctx, channel, data).Err()
}

// listenRedis reads messages from a Redis Pub/Sub channel and fans them out
// to all SSE clients subscribed to that poll.
func (b *Broker) listenRedis(pollID string, sub *redis.PubSub) {
	ch := sub.Channel()
	for msg := range ch {
		b.mu.RLock()
		clients := b.topics[pollID]
		// Copy client set to avoid holding the lock while sending.
		targets := make([]*Client, 0, len(clients))
		for c := range clients {
			targets = append(targets, c)
		}
		b.mu.RUnlock()

		payload := []byte(msg.Payload)
		for _, c := range targets {
			select {
			case c.Send <- payload:
			default:
				// Slow client — drop the message rather than block.
				log.Printf("sse: dropped message for slow client on poll %s", pollID)
			}
		}
	}
}
