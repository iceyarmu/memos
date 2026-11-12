package test

import (
	"context"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	apipb "github.com/usememos/memos/proto/gen/api"
)

func TestStoriesPrivatePersistence(t *testing.T) {
	ctx := context.Background()
	ts := NewTestService(t)
	defer ts.Cleanup()
	owner, err := ts.CreateRegularUser(ctx, "story-owner")
	require.NoError(t, err)
	other, err := ts.CreateRegularUser(ctx, "story-other")
	require.NoError(t, err)
	ownerCtx := ts.CreateUserContext(ctx, owner.ID)
	_, err = ts.Service.ListStories(ctx, &apipb.ListStoriesRequest{})
	require.Equal(t, codes.Unauthenticated, status.Code(err))
	_, err = ts.Service.CreateStory(ownerCtx, &apipb.CreateStoryRequest{Story: &apipb.Story{Title: "private", Visibility: "PUBLIC"}})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
	created, err := ts.Service.CreateStory(ownerCtx, &apipb.CreateStoryRequest{Story: &apipb.Story{Title: "大学生活", Description: "珍藏经历"}})
	require.NoError(t, err)
	require.Equal(t, "PRIVATE", created.Visibility)
	require.NotEmpty(t, created.Uid)
	listed, err := ts.Service.ListStories(ownerCtx, &apipb.ListStoriesRequest{})
	require.NoError(t, err)
	require.Len(t, listed.Stories, 1)
	require.Equal(t, created.Name, listed.Stories[0].Name)
	require.Equal(t, "珍藏经历", listed.Stories[0].Description)
	hidden, err := ts.Service.ListStories(ts.CreateUserContext(ctx, other.ID), &apipb.ListStoriesRequest{})
	require.NoError(t, err)
	require.Empty(t, hidden.Stories)
}

func TestStoriesEventLifecycle(t *testing.T) {
	ctx := context.Background()
	ts := NewTestService(t)
	defer ts.Cleanup()
	owner, err := ts.CreateRegularUser(ctx, "event-owner")
	require.NoError(t, err)
	other, err := ts.CreateRegularUser(ctx, "event-other")
	require.NoError(t, err)
	own := ts.CreateUserContext(ctx, owner.ID)
	foreign := ts.CreateUserContext(ctx, other.ID)
	archive, err := ts.Service.CreateStory(own, &apipb.CreateStoryRequest{Story: &apipb.Story{Title: "College"}})
	require.NoError(t, err)
	attachment := createTestImageAttachment(own, t, ts, "memory.png")
	attachmentUID := strings.TrimPrefix(attachment.Name, "attachments/")
	second := createTestImageAttachment(own, t, ts, "second.png")
	secondUID := strings.TrimPrefix(second.Name, "attachments/")
	memo, err := ts.Service.CreateMemo(own, &apipb.CreateMemoRequest{Memo: &apipb.Memo{Content: "Original memory", Visibility: apipb.Visibility_PRIVATE}})
	require.NoError(t, err)

	input := &apipb.StoryEvent{Title: "Graduation", Content: "Memories", EventDate: "2018-06-20", Tags: []string{"milestone"}, Memo: memo.Name, AttachmentUids: []string{secondUID, attachmentUID}}
	_, err = ts.Service.CreateStoryEvent(foreign, &apipb.CreateStoryEventRequest{Parent: archive.Name, Event: input})
	require.Error(t, err)
	event, err := ts.Service.CreateStoryEvent(own, &apipb.CreateStoryEventRequest{Parent: archive.Name, Event: input})
	require.NoError(t, err)
	updated, err := ts.Service.UpdateStoryEvent(own, &apipb.UpdateStoryEventRequest{Event: &apipb.StoryEvent{Name: event.Name, Story: archive.Name, Revision: event.Revision, Title: "Ceremony"}, UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"title"}}})
	require.NoError(t, err)
	require.Equal(t, "Memories", updated.Content)
	require.Equal(t, []string{secondUID, attachmentUID}, updated.AttachmentUids)
	require.Equal(t, memo.Name, updated.Memo)
	addition, err := ts.Service.CreateStoryEvent(own, &apipb.CreateStoryEventRequest{Parent: archive.Name, Event: &apipb.StoryEvent{ParentEvent: event.Uid, Content: "Later reflection", EventDate: "2026-11-12"}})
	require.NoError(t, err)
	require.Equal(t, event.Uid, addition.ParentEvent)
	_, err = ts.Service.CreateStoryEvent(own, &apipb.CreateStoryEventRequest{Parent: archive.Name, Event: &apipb.StoryEvent{ParentEvent: addition.Uid, Content: "Nested", EventDate: "2026-11-12"}})
	require.Error(t, err)

	require.Equal(t, int32(1), updated.Revision)
	_, err = ts.Service.DeleteStoryEvent(own, &apipb.DeleteStoryEventRequest{Name: event.Name, Revision: 0})
	require.Error(t, err)
	_, err = ts.Service.DeleteStoryEvent(foreign, &apipb.DeleteStoryEventRequest{Name: event.Name, Revision: 1})
	require.Error(t, err)
	listed, err := ts.Service.ListStoryEvents(own, &apipb.ListStoryEventsRequest{Parent: archive.Name})
	require.NoError(t, err)
	require.Len(t, listed.Events, 2)

	share, err := ts.Service.SetStoryShare(own, &apipb.SetStoryShareRequest{Name: archive.Name, Enabled: true})
	require.NoError(t, err)
	_, err = ts.Service.SetStoryShare(foreign, &apipb.SetStoryShareRequest{Name: archive.Name, Enabled: false})
	require.Error(t, err)
	shared, err := ts.Service.GetSharedStory(ctx, &apipb.GetSharedStoryRequest{Token: share.Token})
	require.NoError(t, err)
	require.Len(t, shared.Events, 2)
	for _, entry := range shared.Events {
		require.Empty(t, entry.Memo)
	}
	rotated, err := ts.Service.SetStoryShare(own, &apipb.SetStoryShareRequest{Name: archive.Name, Enabled: true})
	require.NoError(t, err)
	_, err = ts.Service.GetSharedStory(ctx, &apipb.GetSharedStoryRequest{Token: share.Token})
	require.Equal(t, codes.NotFound, status.Code(err))
	_, err = ts.Service.SetStoryShare(own, &apipb.SetStoryShareRequest{Name: archive.Name, Enabled: false})
	require.NoError(t, err)
	_, err = ts.Service.GetSharedStory(ctx, &apipb.GetSharedStoryRequest{Token: rotated.Token})
	require.Equal(t, codes.NotFound, status.Code(err))
	_, err = ts.Service.DeleteStory(foreign, &apipb.DeleteStoryRequest{Name: archive.Name})
	require.Error(t, err)
	_, err = ts.Service.DeleteStory(own, &apipb.DeleteStoryRequest{Name: archive.Name})
	require.NoError(t, err)
	listed, err = ts.Service.ListStoryEvents(own, &apipb.ListStoryEventsRequest{Parent: archive.Name})
	require.NoError(t, err)
	require.Empty(t, listed.Events)
	var count int
	require.NoError(t, ts.Store.GetDriver().GetDB().QueryRowContext(ctx, "SELECT COUNT(*) FROM story_event").Scan(&count))
	require.Zero(t, count)
}
