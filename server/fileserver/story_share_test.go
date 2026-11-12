package fileserver

import (
	"context"
	"github.com/stretchr/testify/require"
	apipb "github.com/usememos/memos/proto/gen/api"
	"github.com/usememos/memos/server/auth"
	"github.com/usememos/memos/store"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestStoryImageShareScopeAndRevocation(t *testing.T) {
	ctx := context.Background()
	svc, fs, _, cleanup := newShareAttachmentTestServices(ctx, t)
	defer cleanup()
	user, err := svc.Store.CreateUser(ctx, &store.User{Username: "story-reader", Role: store.RoleUser})
	require.NoError(t, err)
	own := context.WithValue(ctx, auth.UserIDContextKey, user.ID)
	image, err := svc.CreateAttachment(own, &apipb.CreateAttachmentRequest{Attachment: &apipb.Attachment{Filename: "image.png", Type: "image/png", Content: []byte("test image")}})
	require.NoError(t, err)
	uid := strings.TrimPrefix(image.Name, "attachments/")
	archive, err := svc.CreateStory(own, &apipb.CreateStoryRequest{Story: &apipb.Story{Title: "Private archive"}})
	require.NoError(t, err)
	share, err := svc.SetStoryShare(own, &apipb.SetStoryShareRequest{Name: archive.Name, Enabled: true})
	require.NoError(t, err)
	attachment, err := svc.Store.GetAttachment(ctx, &store.FindAttachment{UID: &uid})
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodGet, "/file/attachments/"+uid+"?story_share_token="+share.Token, nil)
	_, err = fs.checkAttachmentPermission(ctx, req, attachment)
	require.Error(t, err, "unrelated image must not be shared")
	_, err = svc.CreateStoryEvent(own, &apipb.CreateStoryEventRequest{Parent: archive.Name, Event: &apipb.StoryEvent{Title: "Memory", EventDate: "2020-01-01", AttachmentUids: []string{uid}}})
	require.NoError(t, err)
	_, err = fs.checkAttachmentPermission(ctx, req, attachment)
	require.NoError(t, err)
	_, err = svc.SetStoryShare(own, &apipb.SetStoryShareRequest{Name: archive.Name, Enabled: false})
	require.NoError(t, err)
	_, err = fs.checkAttachmentPermission(ctx, req, attachment)
	require.Error(t, err, "revoked token must not authorize image")
}
