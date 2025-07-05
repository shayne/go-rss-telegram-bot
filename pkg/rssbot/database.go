package rssbot

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

//go:generate go run tailscale.com/cmd/viewer -type=Subscription,FeedInfo,FeedError

type Database struct {
	mu            sync.RWMutex
	path          string
	Subscriptions map[string]map[string]*Subscription `json:"subscriptions"`
	FeedErrors    map[string]*FeedError               `json:"feed_errors"`
}

type Subscription struct {
	UserID       int64    `json:"user_id"`
	ChatID       int64    `json:"chat_id"`
	FeedURL      string   `json:"feed_url"`
	FeedInfo     FeedInfo `json:"feed_info"`
	LastChecked  string   `json:"last_checked"`
	LastItemGUID string   `json:"last_item_guid"`
}

type FeedInfo struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Link        string `json:"link"`
}

type FeedError struct {
	FeedURL      string `json:"feed_url"`
	ErrorCount   int    `json:"error_count"`
	LastError    string `json:"last_error"`
	LastErrorAt  string `json:"last_error_at"`
	FirstErrorAt string `json:"first_error_at"`
}

func NewDatabase(path string) (*Database, error) {
	db := &Database{
		path:          path,
		Subscriptions: make(map[string]map[string]*Subscription),
		FeedErrors:    make(map[string]*FeedError),
	}

	if _, err := os.Stat(path); err == nil {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read database: %w", err)
		}

		if len(data) > 0 {
			if err := json.Unmarshal(data, db); err != nil {
				return nil, fmt.Errorf("failed to unmarshal database: %w", err)
			}
		}
	}

	return db, nil
}

func (db *Database) save() error {
	data, err := json.MarshalIndent(db, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal database: %w", err)
	}

	if err := os.WriteFile(db.path, data, 0644); err != nil {
		return fmt.Errorf("failed to write database: %w", err)
	}

	return nil
}

func (db *Database) AddSubscription(sub *Subscription) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	userKey := fmt.Sprintf("%d", sub.UserID)
	if db.Subscriptions[userKey] == nil {
		db.Subscriptions[userKey] = make(map[string]*Subscription)
	}

	if _, exists := db.Subscriptions[userKey][sub.FeedURL]; exists {
		return fmt.Errorf("already subscribed to this feed")
	}

	sub.LastChecked = time.Now().Format(time.RFC3339)
	db.Subscriptions[userKey][sub.FeedURL] = sub

	return db.save()
}

func (db *Database) RemoveSubscription(userID int64, feedURL string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	userKey := fmt.Sprintf("%d", userID)
	if userSubs, ok := db.Subscriptions[userKey]; ok {
		delete(userSubs, feedURL)
		if len(userSubs) == 0 {
			delete(db.Subscriptions, userKey)
		}
	}

	return db.save()
}

func (db *Database) GetUserSubscriptions(userID int64) ([]*Subscription, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	userKey := fmt.Sprintf("%d", userID)
	userSubs, ok := db.Subscriptions[userKey]
	if !ok {
		return []*Subscription{}, nil
	}

	subs := make([]*Subscription, 0, len(userSubs))
	for _, sub := range userSubs {
		subs = append(subs, sub)
	}

	return subs, nil
}

func (db *Database) GetAllSubscriptions() ([]*Subscription, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	var subs []*Subscription
	for _, userSubs := range db.Subscriptions {
		for _, sub := range userSubs {
			subs = append(subs, sub)
		}
	}

	return subs, nil
}

func (db *Database) UpdateLastChecked(userID int64, feedURL string, lastItemGUID string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	userKey := fmt.Sprintf("%d", userID)
	if sub, ok := db.Subscriptions[userKey][feedURL]; ok {
		sub.LastChecked = time.Now().Format(time.RFC3339)
		sub.LastItemGUID = lastItemGUID
		return db.save()
	}

	return fmt.Errorf("subscription not found")
}

func (db *Database) RecordFeedError(feedURL string, err error) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	feedErr, exists := db.FeedErrors[feedURL]
	if !exists {
		feedErr = &FeedError{
			FeedURL:      feedURL,
			FirstErrorAt: time.Now().Format(time.RFC3339),
		}
		db.FeedErrors[feedURL] = feedErr
	}

	feedErr.ErrorCount++
	feedErr.LastError = err.Error()
	feedErr.LastErrorAt = time.Now().Format(time.RFC3339)

	return db.save()
}

func (db *Database) ClearFeedError(feedURL string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	delete(db.FeedErrors, feedURL)
	return db.save()
}

func (db *Database) GetFeedError(feedURL string) (*FeedError, bool) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	feedErr, exists := db.FeedErrors[feedURL]
	return feedErr, exists
}
