package api

import (
	"context"

	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/usememos/memos/core/access"
	"github.com/usememos/memos/internal/webhook"
	apipb "github.com/usememos/memos/proto/gen/api"
	storepb "github.com/usememos/memos/proto/gen/store"
	"github.com/usememos/memos/server/auth"
	"github.com/usememos/memos/store"
)

// DispatchMemoCreatedWebhook dispatches a webhook when a memo is created.
func (s *APIService) DispatchMemoCreatedWebhook(ctx context.Context, memo *apipb.Memo) error {
	return s.dispatchMemoRelatedWebhook(ctx, memo, "memos.memo.created")
}

// DispatchMemoUpdatedWebhook dispatches webhook when memo is updated.
func (s *APIService) DispatchMemoUpdatedWebhook(ctx context.Context, memo *apipb.Memo) error {
	return s.dispatchMemoRelatedWebhook(ctx, memo, "memos.memo.updated")
}

// DispatchMemoDeletedWebhook dispatches webhook when memo is deleted.
func (s *APIService) DispatchMemoDeletedWebhook(ctx context.Context, memo *apipb.Memo) error {
	return s.dispatchMemoRelatedWebhook(ctx, memo, "memos.memo.deleted")
}

// DispatchMemoReactedWebhook dispatches reaction webhooks to the memo owner and,
// for comments, the related memo owner.
func (s *APIService) DispatchMemoReactedWebhook(ctx context.Context, memo *store.Memo, reaction *apipb.Reaction) error {
	if memo == nil || reaction == nil {
		return status.Error(codes.InvalidArgument, "memo and reaction are required")
	}

	receivers := map[int32]struct{}{memo.CreatorID: {}}
	var relatedMemo *store.Memo
	if memo.ParentUID != nil && *memo.ParentUID != "" {
		var err error
		relatedMemo, err = s.Store.GetMemo(ctx, &store.FindMemo{UID: memo.ParentUID})
		if err != nil {
			return errors.Wrap(err, "failed to get related memo for reaction webhook")
		}
		if relatedMemo != nil {
			receivers[relatedMemo.CreatorID] = struct{}{}
		}
	}

	for receiverID := range receivers {
		if err := s.dispatchMemoReactionWebhookToUser(ctx, receiverID, memo, relatedMemo, reaction); err != nil {
			return err
		}
	}
	return nil
}

func (s *APIService) dispatchMemoReactionWebhookToUser(
	ctx context.Context,
	receiverID int32,
	memo, relatedMemo *store.Memo,
	reaction *apipb.Reaction,
) error {
	receiver, err := s.Store.GetUser(ctx, &store.FindUser{ID: &receiverID})
	if err != nil {
		return err
	}
	if receiver == nil || receiver.RowStatus != store.Normal {
		return nil
	}
	for _, subject := range []*store.Memo{memo, relatedMemo} {
		if subject == nil {
			continue
		}
		readContext, err := s.buildMemoReadContextForViewer(ctx, subject, receiver, false, nil)
		if err != nil {
			return errors.Wrap(err, "failed to resolve reaction webhook subject access")
		}
		if !access.CheckMemoReadContext(readContext).Allowed() {
			return nil
		}
	}

	receiverCtx := auth.SetUserInContext(ctx, receiver, "")
	reactions, err := s.Store.ListReactions(receiverCtx, &store.FindReaction{MemoID: &memo.ID})
	if err != nil {
		return err
	}
	attachments, err := s.Store.ListAttachments(receiverCtx, &store.FindAttachment{MemoID: &memo.ID})
	if err != nil {
		return err
	}
	relations, err := s.loadMemoRelations(receiverCtx, memo)
	if err != nil {
		return err
	}
	memoMessage, err := s.convertMemoFromStore(receiverCtx, memo, reactions, attachments, relations)
	if err != nil {
		return err
	}
	webhooks, err := s.Store.GetUserWebhooks(ctx, receiverID)
	if err != nil {
		return err
	}
	for _, hook := range webhooks {
		request := newMemoWebhookRequest(hook, "memos.memo.reacted", memoMessage)
		request.Payload.(*memoWebhookPayload).Reaction = reaction
		webhook.PostAsync(request)
	}
	return nil
}

// DispatchMemoCommentCreatedWebhook dispatches webhook to the related memo owner when a comment is created.
func (s *APIService) DispatchMemoCommentCreatedWebhook(
	ctx context.Context,
	comment, relatedMemo *store.Memo,
	relatedMemoCreatorID int32,
) error {
	receiver, err := s.Store.GetUser(ctx, &store.FindUser{ID: &relatedMemoCreatorID})
	if err != nil {
		return err
	}
	if receiver == nil || receiver.RowStatus != store.Normal {
		return nil
	}
	for _, subject := range []*store.Memo{comment, relatedMemo} {
		readContext, err := s.buildMemoReadContextForViewer(ctx, subject, receiver, false, nil)
		if err != nil {
			return errors.Wrap(err, "failed to resolve webhook subject access")
		}
		decision := access.CheckMemoReadContext(readContext)
		if !decision.Allowed() {
			return nil
		}
	}
	// The request context belongs to the comment author, but the webhook belongs
	// to the context memo's author. Build the prepared payload under the receiver's
	// identity before it enters the existing asynchronous delivery queue.
	receiverCtx := auth.SetUserInContext(ctx, receiver, "")
	reactions, err := s.Store.ListReactions(receiverCtx, &store.FindReaction{MemoID: &comment.ID})
	if err != nil {
		return err
	}
	attachments, err := s.Store.ListAttachments(receiverCtx, &store.FindAttachment{MemoID: &comment.ID})
	if err != nil {
		return err
	}
	relations, err := s.loadMemoRelations(receiverCtx, comment)
	if err != nil {
		return err
	}
	commentMessage, err := s.convertMemoFromStore(receiverCtx, comment, reactions, attachments, relations)
	if err != nil {
		return err
	}
	webhooks, err := s.Store.GetUserWebhooks(ctx, relatedMemoCreatorID)
	if err != nil {
		return err
	}
	for _, hook := range webhooks {
		webhook.PostAsync(newMemoWebhookRequest(hook, "memos.memo.comment.created", commentMessage))
	}
	return nil
}

func (s *APIService) dispatchMemoRelatedWebhook(ctx context.Context, memo *apipb.Memo, activityType string) error {
	creator, err := ResolveUserByName(ctx, s.Store, memo.Creator)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "invalid memo creator")
	}
	if creator == nil {
		return status.Errorf(codes.NotFound, "memo creator not found")
	}
	creatorID := creator.ID
	webhooks, err := s.Store.GetUserWebhooks(ctx, creatorID)
	if err != nil {
		return err
	}
	for _, hook := range webhooks {
		webhook.PostAsync(newMemoWebhookRequest(hook, activityType, memo))
	}
	return nil
}

// memoWebhookPayload is the JSON body delivered to user webhooks for memo
// activity. Field names are part of the webhook contract.
type memoWebhookPayload struct {
	// URL is the destination the payload was sent to.
	URL string `json:"url"`
	// ActivityType names the event, for example "memos.memo.created".
	ActivityType string `json:"activityType"`
	// Creator is the resource name of the memo creator. Format: users/{user}
	Creator string `json:"creator"`
	// Memo is the memo that triggered the event.
	Memo     *apipb.Memo     `json:"memo"`
	Reaction *apipb.Reaction `json:"reaction,omitempty"`
}

func newMemoWebhookRequest(hook *storepb.WebhooksUserSetting_Webhook, activityType string, memo *apipb.Memo) *webhook.Request {
	return &webhook.Request{
		URL:           hook.Url,
		Label:         activityType,
		SigningSecret: hook.SigningSecret,
		Payload: &memoWebhookPayload{
			URL:          hook.Url,
			ActivityType: activityType,
			Creator:      memo.Creator,
			Memo:         memo,
		},
	}
}

func (s *APIService) getMemoContentSnippet(content string) (string, error) {
	// Use goldmark service for snippet generation
	snippet, err := s.MarkdownService.GenerateSnippet([]byte(content), 64)
	if err != nil {
		return "", errors.Wrap(err, "failed to generate snippet")
	}
	return snippet, nil
}

// parseMemoOrderBy parses the order_by field and sets the appropriate ordering in memoFind.
// Follows AIP-132: supports comma-separated list of fields with optional "desc" suffix.
// Example: "pinned desc, create_time desc" or "update_time asc".
