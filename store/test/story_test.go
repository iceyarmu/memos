package test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/usememos/memos/store"
)

func TestStoryPersistenceAndOwnerIsolation(t *testing.T) {
	ctx := context.Background()
	ts := NewTestingStore(ctx, t)
	item := &store.Story{UID: "life-archive", CreatorID: 1, Title: "人生档案", Description: "图片与回忆", Icon: "📖", Color: "#55A9BE"}
	require.NoError(t, ts.CreateStory(ctx, item))
	rows, err := ts.ListStories(ctx, 1)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, item, rows[0])
	rows, err = ts.ListStories(ctx, 2)
	require.NoError(t, err)
	require.Empty(t, rows)
	require.Error(t, ts.CreateStory(ctx, item), "duplicate UID must not overwrite an archive")
	_, err = ts.ListStories(ctx, 0)
	require.Error(t, err, "missing owner must never list all archives")
}
