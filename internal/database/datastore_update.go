package database

import (
	"context"
	"fmt"

	"cloud.google.com/go/datastore"
)

func (db *DatastoreDB) UpdateUserMaxArticlesOnFeedAdd(userID int, maxArticles int) error {
	ctx := context.Background()

	// Get the user key
	userKey := datastore.IDKey("User", int64(userID), nil)

	// Get current user entity
	var user UserEntity
	if err := db.client.Get(ctx, userKey, &user); err != nil {
		return fmt.Errorf("failed to get user for update: %w", err)
	}

	// Update the max articles setting
	user.MaxArticlesOnFeedAdd = maxArticles

	// Save back to datastore
	_, err := db.client.Put(ctx, userKey, &user)
	if err != nil {
		return fmt.Errorf("failed to update user max articles setting: %w", err)
	}

	return nil
}
