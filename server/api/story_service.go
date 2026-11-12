package api

import (
	"context"
	"encoding/json"
	"google.golang.org/protobuf/types/known/emptypb"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/lithammer/shortuuid/v4"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	apipb "github.com/usememos/memos/proto/gen/api"
	"github.com/usememos/memos/store"
)

// ListStories returns only the authenticated user's life archives.
func (s *APIService) ListStories(ctx context.Context, _ *apipb.ListStoriesRequest) (*apipb.ListStoriesResponse, error) {
	user, err := s.fetchCurrentUser(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get user")
	}
	if user == nil {
		return nil, status.Errorf(codes.Unauthenticated, "authentication required")
	}
	items, err := s.Store.ListStories(ctx, user.ID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list stories")
	}
	result := &apipb.ListStoriesResponse{}
	for _, item := range items {
		result.Stories = append(result.Stories, convertStoryFromStore(item))
	}
	return result, nil
}

// CreateStory creates a private archive for the authenticated user.
func (s *APIService) CreateStory(ctx context.Context, request *apipb.CreateStoryRequest) (*apipb.Story, error) {
	user, err := s.fetchCurrentUser(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get user")
	}
	if user == nil {
		return nil, status.Errorf(codes.Unauthenticated, "authentication required")
	}
	input := request.GetStory()
	if input == nil || strings.TrimSpace(input.Title) == "" || utf8.RuneCountInString(input.Title) > 120 || utf8.RuneCountInString(input.Description) > 12000 {
		return nil, status.Errorf(codes.InvalidArgument, "title is required (up to 120 characters); description is limited to 12000 characters")
	}
	if input.Visibility != "" && input.Visibility != "PRIVATE" {
		return nil, status.Errorf(codes.InvalidArgument, "only private archives are supported")
	}
	unit := input.IntervalUnit
	if unit == "" {
		unit = "days"
	}
	if unit != "days" && unit != "hours" {
		return nil, status.Errorf(codes.InvalidArgument, "invalid interval unit")
	}
	icon, color := input.Icon, input.Color
	if icon == "" {
		icon = "📖"
	}
	if color == "" {
		color = "#55A9BE"
	}
	if utf8.RuneCountInString(icon) > 16 || !regexp.MustCompile(`^#[0-9a-fA-F]{6}$`).MatchString(color) {
		return nil, status.Errorf(codes.InvalidArgument, "invalid icon or color")
	}
	item := &store.Story{UID: shortuuid.New(), CreatorID: user.ID, Title: strings.TrimSpace(input.Title), Description: input.Description, Icon: icon, Color: color, IntervalUnit: unit}
	if err := s.Store.CreateStory(ctx, item); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create story")
	}
	return convertStoryFromStore(item), nil
}

func (s *APIService) storyOwner(ctx context.Context) (int32, error) {
	u, err := s.fetchCurrentUser(ctx)
	if err != nil {
		return 0, status.Errorf(codes.Internal, "failed to get user")
	}
	if u == nil {
		return 0, status.Errorf(codes.Unauthenticated, "authentication required")
	}
	return u.ID, nil
}
func storyUID(name, prefix string) (string, error) {
	uid, ok := strings.CutPrefix(name, prefix+"/")
	if !ok || uid == "" || strings.Contains(uid, "/") {
		return "", status.Errorf(codes.InvalidArgument, "invalid resource name")
	}
	return uid, nil
}

// ListStoryEvents returns entries belonging to an owned archive.
func (s *APIService) ListStoryEvents(ctx context.Context, r *apipb.ListStoryEventsRequest) (*apipb.ListStoryEventsResponse, error) {
	owner, err := s.storyOwner(ctx)
	if err != nil {
		return nil, err
	}
	uid, err := storyUID(r.Parent, "stories")
	if err != nil {
		return nil, err
	}
	items, err := s.Store.ListStoryEvents(ctx, owner, uid)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list events")
	}
	out := &apipb.ListStoryEventsResponse{}
	for _, e := range items {
		out.Events = append(out.Events, convertStoryEvent(e))
	}
	return out, nil
}
func convertStoryEvent(e *store.StoryEvent) *apipb.StoryEvent {
	tags := []string{}
	_ = json.Unmarshal([]byte(e.Tags), &tags)
	memo := ""
	if e.MemoUID != "" {
		memo = "memos/" + e.MemoUID
	}
	return &apipb.StoryEvent{ParentEvent: e.ParentEvent, Memo: memo, Name: "storyEvents/" + e.UID, Uid: e.UID, Story: "stories/" + e.StoryUID, Title: e.Title, Content: e.Content, EventDate: e.Date, EventTime: e.Time, Tags: tags, Revision: e.Revision, AttachmentUids: e.AttachmentUIDs}
}

// CreateStoryEvent records a historical experience.
func (s *APIService) CreateStoryEvent(ctx context.Context, r *apipb.CreateStoryEventRequest) (*apipb.StoryEvent, error) {
	return s.saveStoryEvent(ctx, r.Event, r.Parent, true)
}

// UpdateStoryEvent replaces editable event fields using its revision.
func (s *APIService) UpdateStoryEvent(ctx context.Context, r *apipb.UpdateStoryEventRequest) (*apipb.StoryEvent, error) {
	input := r.GetEvent()
	if input == nil {
		return nil, status.Errorf(codes.InvalidArgument, "event required")
	}
	if r.UpdateMask != nil {
		owner, err := s.storyOwner(ctx)
		if err != nil {
			return nil, err
		}
		uid, err := storyUID(input.Story, "stories")
		if err != nil {
			return nil, err
		}
		events, err := s.Store.ListStoryEvents(ctx, owner, uid)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to load event")
		}
		var merged *apipb.StoryEvent
		for _, e := range events {
			if "storyEvents/"+e.UID == input.Name {
				merged = convertStoryEvent(e)
			}
		}
		if merged == nil {
			return nil, status.Errorf(codes.NotFound, "event unavailable")
		}
		merged.Revision = input.Revision
		for _, p := range r.UpdateMask.Paths {
			switch p {
			case "title":
				merged.Title = input.Title
			case "content":
				merged.Content = input.Content
			case "event_date":
				merged.EventDate = input.EventDate
			case "event_time":
				merged.EventTime = input.EventTime
			case "memo":
				merged.Memo = input.Memo
			case "attachment_uids":
				merged.AttachmentUids = input.AttachmentUids
			case "tags":
				merged.Tags = input.Tags
			default:
				return nil, status.Errorf(codes.InvalidArgument, "unsupported field")
			}
		}
		input = merged
	}
	return s.saveStoryEvent(ctx, input, input.Story, false)
}

func (s *APIService) saveStoryEvent(ctx context.Context, input *apipb.StoryEvent, parent string, create bool) (*apipb.StoryEvent, error) {
	owner, err := s.storyOwner(ctx)
	if err != nil {
		return nil, err
	}
	uid, err := storyUID(parent, "stories")
	if err != nil {
		return nil, err
	}
	if input == nil {
		return nil, status.Errorf(codes.InvalidArgument, "event required")
	}
	if _, err = time.Parse("2006-01-02", input.EventDate); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid date")
	}
	if input.EventTime != "" {
		if _, err = time.Parse("15:04", input.EventTime); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid time")
		}
	}
	if strings.TrimSpace(input.Title+input.Content) == "" || utf8.RuneCountInString(input.Title) > 120 || utf8.RuneCountInString(input.Content) > 12000 || len(input.Tags) > 20 {
		return nil, status.Errorf(codes.InvalidArgument, "invalid event content")
	}
	for _, tag := range input.Tags {
		if strings.TrimSpace(tag) == "" || utf8.RuneCountInString(tag) > 40 {
			return nil, status.Errorf(codes.InvalidArgument, "invalid tag")
		}
	}
	if input.ParentEvent != "" && strings.Contains(input.ParentEvent, "/") {
		return nil, status.Errorf(codes.InvalidArgument, "parent event must be a UID")
	}
	memoUID := ""
	if input.Memo != "" {
		memoUID, err = storyUID(input.Memo, "memos")
		if err != nil {
			return nil, err
		}
	}

	if len(input.AttachmentUids) > 6 {
		return nil, status.Errorf(codes.InvalidArgument, "at most six images")
	}
	seen := map[string]bool{}
	for _, uid := range input.AttachmentUids {
		if seen[uid] {
			return nil, status.Errorf(codes.InvalidArgument, "duplicate image")
		}
		seen[uid] = true
		a, err := s.Store.GetAttachment(ctx, &store.FindAttachment{UID: &uid})
		if err != nil || a == nil || a.CreatorID != owner || a.MemoID != nil || !strings.HasPrefix(a.Type, "image/") {
			return nil, status.Errorf(codes.InvalidArgument, "image unavailable")
		}
	}
	eventUID := shortuuid.New()
	if !create {
		eventUID, err = storyUID(input.Name, "storyEvents")
		if err != nil {
			return nil, err
		}
	}
	tags, _ := json.Marshal(input.Tags)
	e := &store.StoryEvent{UID: eventUID, StoryUID: uid, Title: input.Title, Content: input.Content, Date: input.EventDate, Time: input.EventTime, Tags: string(tags), Revision: input.Revision, AttachmentUIDs: input.AttachmentUids, MemoUID: memoUID, ParentEvent: input.ParentEvent}
	if err = s.Store.SaveStoryEvent(ctx, owner, e, create); err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "archive unavailable or event changed; reload before retrying")
	}
	return convertStoryEvent(e), nil
}

// DeleteStoryEvent deletes an owned event at the supplied revision.
func (s *APIService) DeleteStoryEvent(ctx context.Context, r *apipb.DeleteStoryEventRequest) (*emptypb.Empty, error) {
	owner, err := s.storyOwner(ctx)
	if err != nil {
		return nil, err
	}
	uid, err := storyUID(r.Name, "storyEvents")
	if err != nil {
		return nil, err
	}
	if err = s.Store.DeleteStoryEvent(ctx, owner, uid, r.Revision); err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "event changed or unavailable")
	}
	return &emptypb.Empty{}, nil
}

// DeleteStory deletes an owned archive and its events.
func (s *APIService) DeleteStory(ctx context.Context, r *apipb.DeleteStoryRequest) (*emptypb.Empty, error) {
	owner, err := s.storyOwner(ctx)
	if err != nil {
		return nil, err
	}
	uid, err := storyUID(r.Name, "stories")
	if err != nil {
		return nil, err
	}
	if err = s.Store.DeleteStory(ctx, owner, uid); err != nil {
		return nil, status.Errorf(codes.NotFound, "archive unavailable")
	}
	return &emptypb.Empty{}, nil
}

// UpdateStory updates an owned archive's presentation.
func (s *APIService) UpdateStory(ctx context.Context, r *apipb.UpdateStoryRequest) (*apipb.Story, error) {
	owner, err := s.storyOwner(ctx)
	if err != nil {
		return nil, err
	}
	input := r.GetStory()
	if input == nil {
		return nil, status.Errorf(codes.InvalidArgument, "story required")
	}
	uid, err := storyUID(input.Name, "stories")
	if err != nil {
		return nil, err
	}
	items, err := s.Store.ListStories(ctx, owner)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to read archive")
	}
	var item *store.Story
	for _, v := range items {
		if v.UID == uid {
			item = v
		}
	}
	if item == nil {
		return nil, status.Errorf(codes.NotFound, "archive unavailable")
	}
	paths := []string{"title", "description", "icon", "color", "interval_unit"}
	if r.UpdateMask != nil {
		paths = r.UpdateMask.Paths
	}
	for _, p := range paths {
		switch p {
		case "title":
			item.Title = input.Title
		case "description":
			item.Description = input.Description
		case "icon":
			item.Icon = input.Icon
		case "interval_unit":
			if input.IntervalUnit != "" {
				item.IntervalUnit = input.IntervalUnit
			}
		case "color":
			item.Color = input.Color
		default:
			return nil, status.Errorf(codes.InvalidArgument, "unsupported field")
		}
	}
	if strings.TrimSpace(item.Title) == "" || utf8.RuneCountInString(item.Title) > 120 || utf8.RuneCountInString(item.Description) > 12000 || utf8.RuneCountInString(item.Icon) > 16 || !regexp.MustCompile(`^#[0-9a-fA-F]{6}$`).MatchString(item.Color) {
		return nil, status.Errorf(codes.InvalidArgument, "invalid archive presentation")
	}
	if item.IntervalUnit != "days" && item.IntervalUnit != "hours" {
		return nil, status.Errorf(codes.InvalidArgument, "invalid interval unit")
	}
	if err = s.Store.UpdateStory(ctx, item); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update archive")
	}
	return convertStoryFromStore(item), nil
}

// SetStoryShare creates a fresh read-only capability or revokes the existing one.
func (s *APIService) SetStoryShare(ctx context.Context, r *apipb.SetStoryShareRequest) (*apipb.StoryShare, error) {
	owner, err := s.storyOwner(ctx)
	if err != nil {
		return nil, err
	}
	uid, err := storyUID(r.Name, "stories")
	if err != nil {
		return nil, err
	}
	var token *string
	if r.Enabled {
		v := shortuuid.New() + shortuuid.New()
		token = &v
	}
	if err = s.Store.SetStoryShare(ctx, owner, uid, token); err != nil {
		return nil, status.Errorf(codes.NotFound, "archive unavailable")
	}
	out := &apipb.StoryShare{}
	if token != nil {
		out.Token = *token
	}
	return out, nil
}

// GetSharedStory reads only the archive named by an active capability.
func (s *APIService) GetSharedStory(ctx context.Context, r *apipb.GetSharedStoryRequest) (*apipb.GetSharedStoryResponse, error) {
	if len(r.Token) != 44 {
		return nil, status.Errorf(codes.NotFound, "share unavailable")
	}
	archive, err := s.Store.GetSharedStory(ctx, r.Token)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "share unavailable")
	}
	events, err := s.Store.ListStoryEvents(ctx, archive.CreatorID, archive.UID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to read archive")
	}
	out := &apipb.GetSharedStoryResponse{Story: convertStoryFromStore(archive)}
	for _, e := range events {
		v := convertStoryEvent(e)
		v.Memo = ""
		out.Events = append(out.Events, v)
	}
	// Recheck after reading so a revoked or rotated token cannot finish a fresh lookup.
	if _, err = s.Store.GetSharedStory(ctx, r.Token); err != nil {
		return nil, status.Errorf(codes.NotFound, "share unavailable")
	}
	return out, nil
}
