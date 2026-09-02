package v1

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/usememos/memos/internal/webhook"
	v1pb "github.com/usememos/memos/proto/gen/api/v1"
	"github.com/usememos/memos/store"
)

func TestCommentReactionWebhookNotifiesCommentAndParentOwners(t *testing.T) {
	ctx := context.Background()
	service := newIntegrationService(t)
	parentOwner := createSpaceTestUser(ctx, t, service, "reaction-parent-owner", store.RoleUser)
	commentOwner := createSpaceTestUser(ctx, t, service, "reaction-comment-owner", store.RoleUser)
	reactor := createSpaceTestUser(ctx, t, service, "reaction-webhook-reactor", store.RoleUser)

	parent, err := service.CreateMemo(userCtx(ctx, parentOwner.ID), &v1pb.CreateMemoRequest{
		Memo: &v1pb.Memo{Content: "parent memo", Visibility: v1pb.Visibility_PUBLIC},
	})
	require.NoError(t, err)
	comment, err := service.CreateMemoComment(userCtx(ctx, commentOwner.ID), &v1pb.CreateMemoCommentRequest{
		Name:    parent.Name,
		Comment: &v1pb.Memo{Content: "comment memo", Visibility: v1pb.Visibility_PUBLIC},
	})
	require.NoError(t, err)

	received := make(chan []byte, 4)
	receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, readErr := io.ReadAll(r.Body)
		if readErr != nil {
			http.Error(w, readErr.Error(), http.StatusBadRequest)
			return
		}
		received <- body
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0}`))
	}))
	defer receiver.Close()

	receiverHost, _, err := net.SplitHostPort(receiver.Listener.Addr().String())
	require.NoError(t, err)
	require.NoError(t, webhook.ConfigurePrivateDestinationAllowlist([]string{receiverHost}))
	defer func() {
		require.NoError(t, webhook.ConfigurePrivateDestinationAllowlist(nil))
	}()

	for _, owner := range []*store.User{parentOwner, commentOwner} {
		_, err = service.CreateUserWebhook(userCtx(ctx, owner.ID), &v1pb.CreateUserWebhookRequest{
			Parent: BuildUserName(owner.Username),
			Webhook: &v1pb.UserWebhook{
				DisplayName: "reaction-capture",
				Url:         receiver.URL,
			},
		})
		require.NoError(t, err)
	}

	reaction, err := service.UpsertMemoReaction(userCtx(ctx, reactor.ID), &v1pb.UpsertMemoReactionRequest{
		Name:     comment.Name,
		Reaction: &v1pb.Reaction{ReactionType: "like"},
	})
	require.NoError(t, err)

	for range 2 {
		var body []byte
		select {
		case body = <-received:
		case <-time.After(3 * time.Second):
			require.FailNow(t, "reaction webhook was not delivered to both memo owners")
		}
		payload := &webhook.WebhookRequestPayload{}
		require.NoError(t, json.Unmarshal(body, payload))
		require.Equal(t, "memos.memo.reacted", payload.ActivityType)
		require.NotNil(t, payload.Memo)
		require.Equal(t, comment.Name, payload.Memo.Name)
		require.Equal(t, parent.Name, payload.Memo.GetParent())
		require.NotNil(t, payload.Reaction)
		require.Equal(t, reaction.Name, payload.Reaction.Name)
		require.Equal(t, BuildUserName(reactor.Username), payload.Reaction.Creator)
	}

	select {
	case extra := <-received:
		require.Failf(t, "unexpected additional reaction webhook", "received %s", extra)
	case <-time.After(200 * time.Millisecond):
	}
}
