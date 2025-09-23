package shared

import (
	"context"
	"fmt"

	"cloud.google.com/go/datastore"
	"goread2/internal/database"
)

// UserArticleCompat provides methods to work with UserArticle entities
// regardless of their field naming conventions in datastore
type UserArticleCompat struct {
	client *datastore.Client
}

// NewUserArticleCompat creates a new compatibility handler
func NewUserArticleCompat(client *datastore.Client) *UserArticleCompat {
	return &UserArticleCompat{client: client}
}

// UserArticleData represents the unified structure we want to work with
type UserArticleData struct {
	UserID    int64
	ArticleID int64
	IsRead    bool
	IsStarred bool
}

// QueryUserArticles attempts to query UserArticle entities and convert them to a unified format
func (c *UserArticleCompat) QueryUserArticles(ctx context.Context) ([]*datastore.Key, []UserArticleData, error) {
	query := datastore.NewQuery("UserArticle")

	// Try current format first (snake_case)
	var currentEntities []database.UserArticleEntity
	keys, err := c.client.GetAll(ctx, query, &currentEntities)
	if err == nil {
		// Success with current format
		data := make([]UserArticleData, len(currentEntities))
		for i, entity := range currentEntities {
			data[i] = UserArticleData{
				UserID:    entity.UserID,
				ArticleID: entity.ArticleID,
				IsRead:    entity.IsRead,
				IsStarred: entity.IsStarred,
			}
		}
		return keys, data, nil
	}

	// If current format fails, use PropertyList approach for maximum flexibility
	var propertyLists []datastore.PropertyList
	keys, err = c.client.GetAll(ctx, query, &propertyLists)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query UserArticle entities even with PropertyList: %w", err)
	}

	// Convert PropertyLists to our unified format
	data := make([]UserArticleData, len(propertyLists))
	for i, props := range propertyLists {
		data[i] = c.extractUserArticleData(props)
	}

	return keys, data, nil
}

// extractUserArticleData extracts data from a PropertyList using flexible field name matching
func (c *UserArticleCompat) extractUserArticleData(props datastore.PropertyList) UserArticleData {
	var result UserArticleData

	for _, prop := range props {
		switch prop.Name {
		// UserID variants
		case "UserID", "user_id", "userId":
			if val, ok := prop.Value.(int64); ok {
				result.UserID = val
			}

		// ArticleID variants
		case "ArticleID", "article_id", "articleId":
			if val, ok := prop.Value.(int64); ok {
				result.ArticleID = val
			}

		// IsRead variants
		case "IsRead", "is_read", "isRead":
			if val, ok := prop.Value.(bool); ok {
				result.IsRead = val
			}

		// IsStarred variants
		case "IsStarred", "is_starred", "isStarred":
			if val, ok := prop.Value.(bool); ok {
				result.IsStarred = val
			}
		}
	}

	return result
}

// FindOrphanedByArticles finds UserArticle entities that reference non-existent articles
func (c *UserArticleCompat) FindOrphanedByArticles(ctx context.Context, existingArticleIDs map[int64]bool) ([]*datastore.Key, error) {
	keys, data, err := c.QueryUserArticles(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query user articles: %w", err)
	}

	var orphanedKeys []*datastore.Key
	for i, userArticle := range data {
		if userArticle.ArticleID > 0 && !existingArticleIDs[userArticle.ArticleID] {
			orphanedKeys = append(orphanedKeys, keys[i])
		}
	}

	return orphanedKeys, nil
}

// FindOrphanedByUsers finds UserArticle entities that reference non-existent users
func (c *UserArticleCompat) FindOrphanedByUsers(ctx context.Context, existingUserIDs map[int64]bool) ([]*datastore.Key, error) {
	keys, data, err := c.QueryUserArticles(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query user articles: %w", err)
	}

	var orphanedKeys []*datastore.Key
	for i, userArticle := range data {
		if userArticle.UserID > 0 && !existingUserIDs[userArticle.UserID] {
			orphanedKeys = append(orphanedKeys, keys[i])
		}
	}

	return orphanedKeys, nil
}

// BackupUserArticles creates backup copies of UserArticle entities
func (c *UserArticleCompat) BackupUserArticles(ctx context.Context, suffix string) error {
	keys, data, err := c.QueryUserArticles(ctx)
	if err != nil {
		return fmt.Errorf("failed to query user articles for backup: %w", err)
	}

	if len(keys) == 0 {
		fmt.Printf("  No UserArticle entities found to backup\n")
		return nil
	}

	// Store as backup using current standard format
	for i, userArticleData := range data {
		entity := database.UserArticleEntity{
			UserID:    userArticleData.UserID,
			ArticleID: userArticleData.ArticleID,
			IsRead:    userArticleData.IsRead,
			IsStarred: userArticleData.IsStarred,
		}

		originalKey := keys[i]
		backupKeyName := fmt.Sprintf("%s_backup%s", originalKey.Name, suffix)
		backupKey := datastore.NameKey("UserArticle_backup", backupKeyName, nil)

		_, err = c.client.Put(ctx, backupKey, &entity)
		if err != nil {
			return fmt.Errorf("failed to backup UserArticle entity %s: %w", originalKey.Name, err)
		}
	}

	fmt.Printf("  Backed up %d UserArticle entities (using flexible format detection)\n", len(data))
	return nil
}