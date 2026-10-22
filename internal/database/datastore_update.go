package database

import (
	"fmt"
)

func (db *DatastoreDB) UpdateUserMaxArticlesOnFeedAdd(userID int, maxArticles int) error {
	if maxArticles <= 0 || maxArticles > 10000 {
		return fmt.Errorf("maxArticles must be between 1 and 10000, got %d", maxArticles)
	}

	err := db.updateUser(userID, func(user *UserEntity) error {
		user.MaxArticlesOnFeedAdd = maxArticles
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to update user max articles setting: %w", err)
	}
	return nil
}
