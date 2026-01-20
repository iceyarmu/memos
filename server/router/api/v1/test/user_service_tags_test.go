package test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1pb "github.com/usememos/memos/proto/gen/api/v1"
	storepb "github.com/usememos/memos/proto/gen/store"
	"github.com/usememos/memos/store"
)

func TestListUserTags_BasicFunctionality(t *testing.T) {
	ctx := context.Background()

	// Create test service
	ts := NewTestService(t)
	defer ts.Cleanup()

	// Create a test host user
	user, err := ts.CreateHostUser(ctx, "test_user")
	require.NoError(t, err)

	// Create user context for authentication
	userCtx := ts.CreateUserContext(ctx, user.ID)

	// Create memos with various tags
	testCases := []struct {
		memoUID  string
		content  string
		tags     []string
		visible  store.Visibility
	}{
		{"memo-1", "First memo", []string{"work", "project1"}, store.Public},
		{"memo-2", "Second memo", []string{"work", "project2"}, store.Public},
		{"memo-3", "Third memo", []string{"personal", "family"}, store.Protected},
		{"memo-4", "Fourth memo", []string{"personal"}, store.Private},
		{"memo-5", "Fifth memo", []string{"work/subproject", "archive"}, store.Public},
	}

	for _, tc := range testCases {
		memo, err := ts.Store.CreateMemo(ctx, &store.Memo{
			UID:        tc.memoUID,
			CreatorID:  user.ID,
			Content:    tc.content,
			Visibility: tc.visible,
			Payload: &storepb.MemoPayload{
				Tags: tc.tags,
			},
		})
		require.NoError(t, err)
		require.NotNil(t, memo)
	}

	// Test ListUserTags for the user's own tags (should see all tags)
	userName := fmt.Sprintf("users/%d", user.ID)
	response, err := ts.Service.ListUserTags(userCtx, &v1pb.ListUserTagsRequest{
		Parent: userName,
	})
	require.NoError(t, err)
	require.NotNil(t, response)

	// Verify tags are returned correctly
	expectedTags := []string{
		"archive",
		"family", 
		"personal",
		"project1",
		"project2",
		"work",
		"work/subproject",
	}
	
	require.Equal(t, expectedTags, response.Tags, "Tags should be sorted alphabetically and deduplicated")
}

func TestListUserTags_HierarchicalSorting(t *testing.T) {
	ctx := context.Background()

	// Create test service
	ts := NewTestService(t)
	defer ts.Cleanup()

	// Create a test user
	user, err := ts.CreateHostUser(ctx, "hierarchy_user")
	require.NoError(t, err)

	// Create user context for authentication
	userCtx := ts.CreateUserContext(ctx, user.ID)

	// Create memos with hierarchical tags in mixed order
	hierarchicalTags := []string{
		"work/project2/task1", 
		"work", 
		"personal/family", 
		"personal", 
		"work/project1", 
		"archive/2024/q1",
		"archive",
		"archive/2024",
	}

	for i, tag := range hierarchicalTags {
		memo, err := ts.Store.CreateMemo(ctx, &store.Memo{
			UID:        fmt.Sprintf("memo-%d", i+1),
			CreatorID:  user.ID,
			Content:    fmt.Sprintf("Memo with tag %s", tag),
			Visibility: store.Public,
			Payload: &storepb.MemoPayload{
				Tags: []string{tag},
			},
		})
		require.NoError(t, err)
		require.NotNil(t, memo)
	}

	// Test ListUserTags
	userName := fmt.Sprintf("users/%d", user.ID)
	response, err := ts.Service.ListUserTags(userCtx, &v1pb.ListUserTagsRequest{
		Parent: userName,
	})
	require.NoError(t, err)
	require.NotNil(t, response)

	// Verify hierarchical sorting
	expectedTags := []string{
		"archive",
		"archive/2024", 
		"archive/2024/q1",
		"personal",
		"personal/family",
		"work",
		"work/project1",
		"work/project2/task1",
	}
	
	require.Equal(t, expectedTags, response.Tags, "Tags should be sorted by hierarchy with parent tags before child tags")
}

func TestListUserTags_VisibilityPermissions(t *testing.T) {
	ctx := context.Background()

	// Create test service
	ts := NewTestService(t)
	defer ts.Cleanup()

	// Create users
	owner, err := ts.CreateHostUser(ctx, "owner")
	require.NoError(t, err)

	visitor, err := ts.CreateRegularUser(ctx, "visitor")
	require.NoError(t, err)

	// Create memos with different visibility levels
	testMemos := []struct {
		tags       []string
		visibility store.Visibility
	}{
		{[]string{"public-tag"}, store.Public},
		{[]string{"protected-tag"}, store.Protected},
		{[]string{"private-tag"}, store.Private},
	}

	for i, tc := range testMemos {
		memo, err := ts.Store.CreateMemo(ctx, &store.Memo{
			UID:        fmt.Sprintf("memo-%d", i+1),
			CreatorID:  owner.ID,
			Content:    fmt.Sprintf("Memo %d", i+1),
			Visibility: tc.visibility,
			Payload: &storepb.MemoPayload{
				Tags: tc.tags,
			},
		})
		require.NoError(t, err)
		require.NotNil(t, memo)
	}

	ownerName := fmt.Sprintf("users/%d", owner.ID)

	// Test as owner (should see all tags)
	ownerCtx := ts.CreateUserContext(ctx, owner.ID)
	ownerResponse, err := ts.Service.ListUserTags(ownerCtx, &v1pb.ListUserTagsRequest{
		Parent: ownerName,
	})
	require.NoError(t, err)
	require.NotNil(t, ownerResponse)
	
	expectedOwnerTags := []string{"private-tag", "protected-tag", "public-tag"}
	require.Equal(t, expectedOwnerTags, ownerResponse.Tags, "Owner should see all their tags")

	// Test as visitor (should see only public and protected tags)
	visitorCtx := ts.CreateUserContext(ctx, visitor.ID)
	visitorResponse, err := ts.Service.ListUserTags(visitorCtx, &v1pb.ListUserTagsRequest{
		Parent: ownerName,
	})
	require.NoError(t, err)
	require.NotNil(t, visitorResponse)
	
	expectedVisitorTags := []string{"protected-tag", "public-tag"}
	require.Equal(t, expectedVisitorTags, visitorResponse.Tags, "Visitor should see only public and protected tags")

	// Test as unauthenticated user (should see only public tags)
	unauthResponse, err := ts.Service.ListUserTags(ctx, &v1pb.ListUserTagsRequest{
		Parent: ownerName,
	})
	require.NoError(t, err)
	require.NotNil(t, unauthResponse)
	
	expectedPublicTags := []string{"public-tag"}
	require.Equal(t, expectedPublicTags, unauthResponse.Tags, "Unauthenticated user should see only public tags")
}

func TestListUserTags_EmptyTags(t *testing.T) {
	ctx := context.Background()

	// Create test service
	ts := NewTestService(t)
	defer ts.Cleanup()

	// Create a test user
	user, err := ts.CreateHostUser(ctx, "empty_tags_user")
	require.NoError(t, err)

	// Create user context
	userCtx := ts.CreateUserContext(ctx, user.ID)

	// Create memo without tags
	memo, err := ts.Store.CreateMemo(ctx, &store.Memo{
		UID:        "memo-no-tags",
		CreatorID:  user.ID,
		Content:    "Memo without tags",
		Visibility: store.Public,
		Payload: &storepb.MemoPayload{
			Tags: []string{},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, memo)

	// Create memo with empty string tags
	memo2, err := ts.Store.CreateMemo(ctx, &store.Memo{
		UID:        "memo-empty-tags",
		CreatorID:  user.ID,
		Content:    "Memo with empty tags",
		Visibility: store.Public,
		Payload: &storepb.MemoPayload{
			Tags: []string{"", "valid-tag", ""},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, memo2)

	// Test ListUserTags
	userName := fmt.Sprintf("users/%d", user.ID)
	response, err := ts.Service.ListUserTags(userCtx, &v1pb.ListUserTagsRequest{
		Parent: userName,
	})
	require.NoError(t, err)
	require.NotNil(t, response)

	// Should only return the valid tag, empty strings should be filtered out
	expectedTags := []string{"valid-tag"}
	require.Equal(t, expectedTags, response.Tags, "Empty tags should be filtered out")
}

func TestListUserTags_InvalidUserName(t *testing.T) {
	ctx := context.Background()

	// Create test service
	ts := NewTestService(t)
	defer ts.Cleanup()

	// Test with invalid user name format
	_, err := ts.Service.ListUserTags(ctx, &v1pb.ListUserTagsRequest{
		Parent: "invalid-format",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid user name")
}

// Test for the sortTagsByHierarchy helper function
func TestSortTagsByHierarchy(t *testing.T) {
	testCases := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "simple hierarchical tags",
			input:    []string{"work/project2", "work", "personal/family", "personal", "work/project1"},
			expected: []string{"personal", "personal/family", "work", "work/project1", "work/project2"},
		},
		{
			name:     "mixed depth tags",
			input:    []string{"a/b/c", "a", "b", "a/b"},
			expected: []string{"a", "a/b", "a/b/c", "b"},
		},
		{
			name:     "single level tags",
			input:    []string{"zebra", "apple", "banana"},
			expected: []string{"apple", "banana", "zebra"},
		},
		{
			name:     "deep hierarchy",
			input:    []string{"a/b/c/d/e", "a/b", "a/b/c", "a", "a/b/c/d"},
			expected: []string{"a", "a/b", "a/b/c", "a/b/c/d", "a/b/c/d/e"},
		},
		{
			name:     "empty input",
			input:    []string{},
			expected: []string{},
		},
		{
			name:     "chinese tags",
			input:    []string{"工作/项目", "工作", "个人", "个人/家庭"},
			expected: []string{"个人", "个人/家庭", "工作", "工作/项目"},
		},
		{
			name:     "tags with emoji",
			input:    []string{"🎪Resource/🏛️Culture", "🎪Resource", "work", "📚Resource/Books"},
			expected: []string{"🎪Resource", "📚Resource/Books", "🎪Resource/🏛️Culture", "work"},
		},
		{
			name:     "tags with emoji variation selectors",
			input:    []string{"🏄Event/🟡Trivial", "🖊️Area/💪MuscleGuy", "📝Notes"},
			expected: []string{"🖊️Area/💪MuscleGuy", "🏄Event/🟡Trivial", "📝Notes"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Since sortTagsByHierarchy is a private function, we need to test it
			// through the actual API call. But for unit testing the sorting logic,
			// we can create a simple test that verifies the sorting behavior
			// through ListUserTags API.
			
			ctx := context.Background()
			ts := NewTestService(t)
			defer ts.Cleanup()

			user, err := ts.CreateHostUser(ctx, "sort_test_user")
			require.NoError(t, err)

			userCtx := ts.CreateUserContext(ctx, user.ID)

			// Create memos with the test tags
			for i, tag := range tc.input {
				memo, err := ts.Store.CreateMemo(ctx, &store.Memo{
					UID:        fmt.Sprintf("memo-%d", i),
					CreatorID:  user.ID,
					Content:    fmt.Sprintf("Memo %d", i),
					Visibility: store.Public,
					Payload: &storepb.MemoPayload{
						Tags: []string{tag},
					},
				})
				require.NoError(t, err)
				require.NotNil(t, memo)
			}

			// Test ListUserTags to verify sorting
			userName := fmt.Sprintf("users/%d", user.ID)
			response, err := ts.Service.ListUserTags(userCtx, &v1pb.ListUserTagsRequest{
				Parent: userName,
			})
			require.NoError(t, err)
			require.NotNil(t, response)

			assert.Equal(t, tc.expected, response.Tags, "Tags should be sorted correctly for case: %s", tc.name)
		})
	}
}

func TestListUserTags_EmojiHandling(t *testing.T) {
	ctx := context.Background()

	// Create test service
	ts := NewTestService(t)
	defer ts.Cleanup()

	// Create a test user
	user, err := ts.CreateHostUser(ctx, "emoji_user")
	require.NoError(t, err)

	// Create user context for authentication
	userCtx := ts.CreateUserContext(ctx, user.ID)

	// Create memos with emoji tags in mixed order
	emojiTags := []string{
		"🎪Resource兴趣/🏛️Culture", 
		"🎪Resource兴趣", 
		"work", 
		"📚Resource/Books",
		"🎯work/🎨project",
		"🌟personal",
	}

	for i, tag := range emojiTags {
		memo, err := ts.Store.CreateMemo(ctx, &store.Memo{
			UID:        fmt.Sprintf("memo-%d", i+1),
			CreatorID:  user.ID,
			Content:    fmt.Sprintf("Memo with tag %s", tag),
			Visibility: store.Public,
			Payload: &storepb.MemoPayload{
				Tags: []string{tag},
			},
		})
		require.NoError(t, err)
		require.NotNil(t, memo)
	}

	// Test ListUserTags
	userName := fmt.Sprintf("users/%d", user.ID)
	response, err := ts.Service.ListUserTags(userCtx, &v1pb.ListUserTagsRequest{
		Parent: userName,
	})
	require.NoError(t, err)
	require.NotNil(t, response)

	// Verify emoji-based sorting
	// After removing emojis for comparison:
	// "Resource/Books" < "Resource兴趣" < "personal" < "work" < "work/project"
	expectedTags := []string{
		"📚Resource/Books",
		"🎪Resource兴趣",
		"🎪Resource兴趣/🏛️Culture", 
		"🌟personal",
		"work",
		"🎯work/🎨project",
	}
	
	require.Equal(t, expectedTags, response.Tags, "Tags should be sorted by content without emoji, but returned with emoji intact")
}