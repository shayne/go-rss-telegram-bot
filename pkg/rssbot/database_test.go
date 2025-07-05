package rssbot

import (
	"fmt"
	"os"
	"testing"
)

func TestDatabase(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-db-*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	db, err := NewDatabase(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	sub := &Subscription{
		UserID:  123,
		ChatID:  456,
		FeedURL: "https://example.com/feed.xml",
		FeedInfo: FeedInfo{
			Title:       "Example Feed",
			Description: "Test feed",
			Link:        "https://example.com",
		},
	}

	t.Run("AddSubscription", func(t *testing.T) {
		err := db.AddSubscription(sub)
		if err != nil {
			t.Errorf("Failed to add subscription: %v", err)
		}

		err = db.AddSubscription(sub)
		if err == nil || err.Error() != "already subscribed to this feed" {
			t.Errorf("Expected 'already subscribed' error, got: %v", err)
		}
	})

	t.Run("GetUserSubscriptions", func(t *testing.T) {
		subs, err := db.GetUserSubscriptions(123)
		if err != nil {
			t.Errorf("Failed to get user subscriptions: %v", err)
		}
		if len(subs) != 1 {
			t.Errorf("Expected 1 subscription, got %d", len(subs))
		}
		if subs[0].FeedURL != "https://example.com/feed.xml" {
			t.Errorf("Unexpected feed URL: %s", subs[0].FeedURL)
		}
	})

	t.Run("GetAllSubscriptions", func(t *testing.T) {
		subs, err := db.GetAllSubscriptions()
		if err != nil {
			t.Errorf("Failed to get all subscriptions: %v", err)
		}
		if len(subs) != 1 {
			t.Errorf("Expected 1 subscription, got %d", len(subs))
		}
	})

	t.Run("UpdateLastChecked", func(t *testing.T) {
		err := db.UpdateLastChecked(123, "https://example.com/feed.xml", "guid-123")
		if err != nil {
			t.Errorf("Failed to update last checked: %v", err)
		}

		subs, _ := db.GetUserSubscriptions(123)
		if subs[0].LastItemGUID != "guid-123" {
			t.Errorf("Expected LastItemGUID to be 'guid-123', got '%s'", subs[0].LastItemGUID)
		}
	})

	t.Run("RemoveSubscription", func(t *testing.T) {
		err := db.RemoveSubscription(123, "https://example.com/feed.xml")
		if err != nil {
			t.Errorf("Failed to remove subscription: %v", err)
		}

		subs, _ := db.GetUserSubscriptions(123)
		if len(subs) != 0 {
			t.Errorf("Expected 0 subscriptions after removal, got %d", len(subs))
		}
	})

	t.Run("FeedErrors", func(t *testing.T) {
		testErr := fmt.Errorf("test error")
		err := db.RecordFeedError("https://example.com/feed.xml", testErr)
		if err != nil {
			t.Errorf("Failed to record feed error: %v", err)
		}

		feedErr, exists := db.GetFeedError("https://example.com/feed.xml")
		if !exists {
			t.Error("Expected feed error to exist")
		}
		if feedErr.ErrorCount != 1 {
			t.Errorf("Expected error count 1, got %d", feedErr.ErrorCount)
		}

		err = db.ClearFeedError("https://example.com/feed.xml")
		if err != nil {
			t.Errorf("Failed to clear feed error: %v", err)
		}

		_, exists = db.GetFeedError("https://example.com/feed.xml")
		if exists {
			t.Error("Expected feed error to be cleared")
		}
	})
}

func TestDatabasePersistence(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-db-persist-*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	db1, err := NewDatabase(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	sub := &Subscription{
		UserID:  123,
		ChatID:  456,
		FeedURL: "https://example.com/feed.xml",
		FeedInfo: FeedInfo{
			Title: "Example Feed",
		},
	}

	db1.AddSubscription(sub)

	db2, err := NewDatabase(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	subs, err := db2.GetUserSubscriptions(123)
	if err != nil {
		t.Fatal(err)
	}
	if len(subs) != 1 {
		t.Errorf("Expected 1 subscription after reload, got %d", len(subs))
	}
}