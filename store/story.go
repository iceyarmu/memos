package store

import (
	"context"
	"strings"

	"github.com/pkg/errors"
)

// Story is a private life archive owned by one user.
type Story struct {
	UID         string
	CreatorID   int32
	Title       string
	Description string
	Icon        string
	Color       string
	CreatedTs   int64
	UpdatedTs   int64
}

// CreateStory persists a private archive.
func (s *Store) CreateStory(ctx context.Context, story *Story) error {
	if story == nil || story.CreatorID <= 0 || story.UID == "" || strings.TrimSpace(story.Title) == "" {
		return errors.New("invalid story")
	}
	return s.driver.CreateStory(ctx, story)
}

// ListStories lists only archives belonging to the specified user.
func (s *Store) ListStories(ctx context.Context, creatorID int32) ([]*Story, error) {
	if creatorID <= 0 {
		return nil, errors.New("story owner required")
	}
	return s.driver.ListStories(ctx, creatorID)
}

// StoryEvent is a dated entry in a private life archive.
type StoryEvent struct {
	ParentEvent    string
	MemoUID        string
	AttachmentUIDs []string
	UID            string
	StoryUID       string
	Title          string
	Content        string
	Date           string
	Time           string
	Tags           string
	Revision       int32
}

// ListStoryEvents reads events within an owner's archive.
func (s *Store) ListStoryEvents(ctx context.Context, owner int32, uid string) ([]*StoryEvent, error) {
	return s.driver.ListStoryEvents(ctx, owner, uid)
}

// SaveStoryEvent creates or updates an entry with optimistic concurrency.
func (s *Store) SaveStoryEvent(ctx context.Context, owner int32, event *StoryEvent, create bool) error {
	return s.driver.SaveStoryEvent(ctx, owner, event, create)
}

// DeleteStoryEvent removes an owned entry at the supplied revision.
func (s *Store) DeleteStoryEvent(ctx context.Context, owner int32, uid string, revision int32) error {
	return s.driver.DeleteStoryEvent(ctx, owner, uid, revision)
}

// UpdateStory changes an owned archive's presentation.
func (s *Store) UpdateStory(ctx context.Context, item *Story) error {
	return s.driver.UpdateStory(ctx, item)
}

// DeleteStory removes an owned archive and its entries atomically.
func (s *Store) DeleteStory(ctx context.Context, owner int32, uid string) error {
	return s.driver.DeleteStory(ctx, owner, uid)
}

// SetStoryShare replaces an owner's share token; nil revokes it.
func (s *Store) SetStoryShare(ctx context.Context, owner int32, uid string, token *string) error {
	return s.driver.SetStoryShare(ctx, owner, uid, token)
}

// GetSharedStory resolves a share capability.
func (s *Store) GetSharedStory(ctx context.Context, token string) (*Story, error) {
	return s.driver.GetSharedStory(ctx, token)
}
