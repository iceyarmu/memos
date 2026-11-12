package api

import (
	"connectrpc.com/connect"
	"context"
	apipb "github.com/usememos/memos/proto/gen/api"
	"google.golang.org/protobuf/types/known/emptypb"
)

// ListStories adapts the StoryService RPC for Connect.
func (s *ConnectServiceHandler) ListStories(ctx context.Context, req *connect.Request[apipb.ListStoriesRequest]) (*connect.Response[apipb.ListStoriesResponse], error) {
	resp, err := s.APIService.ListStories(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

// CreateStory adapts the StoryService RPC for Connect.
func (s *ConnectServiceHandler) CreateStory(ctx context.Context, req *connect.Request[apipb.CreateStoryRequest]) (*connect.Response[apipb.Story], error) {
	resp, err := s.APIService.CreateStory(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

// UpdateStory adapts the StoryService RPC for Connect.
func (s *ConnectServiceHandler) UpdateStory(ctx context.Context, req *connect.Request[apipb.UpdateStoryRequest]) (*connect.Response[apipb.Story], error) {
	resp, err := s.APIService.UpdateStory(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

// DeleteStory adapts the StoryService RPC for Connect.
func (s *ConnectServiceHandler) DeleteStory(ctx context.Context, req *connect.Request[apipb.DeleteStoryRequest]) (*connect.Response[emptypb.Empty], error) {
	resp, err := s.APIService.DeleteStory(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

// ListStoryEvents adapts the StoryService RPC for Connect.
func (s *ConnectServiceHandler) ListStoryEvents(ctx context.Context, req *connect.Request[apipb.ListStoryEventsRequest]) (*connect.Response[apipb.ListStoryEventsResponse], error) {
	resp, err := s.APIService.ListStoryEvents(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

// CreateStoryEvent adapts the StoryService RPC for Connect.
func (s *ConnectServiceHandler) CreateStoryEvent(ctx context.Context, req *connect.Request[apipb.CreateStoryEventRequest]) (*connect.Response[apipb.StoryEvent], error) {
	resp, err := s.APIService.CreateStoryEvent(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

// UpdateStoryEvent adapts the StoryService RPC for Connect.
func (s *ConnectServiceHandler) UpdateStoryEvent(ctx context.Context, req *connect.Request[apipb.UpdateStoryEventRequest]) (*connect.Response[apipb.StoryEvent], error) {
	resp, err := s.APIService.UpdateStoryEvent(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

// DeleteStoryEvent adapts the StoryService RPC for Connect.
func (s *ConnectServiceHandler) DeleteStoryEvent(ctx context.Context, req *connect.Request[apipb.DeleteStoryEventRequest]) (*connect.Response[emptypb.Empty], error) {
	resp, err := s.APIService.DeleteStoryEvent(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

// SetStoryShare adapts the story RPC for Connect.
func (s *ConnectServiceHandler) SetStoryShare(ctx context.Context, r *connect.Request[apipb.SetStoryShareRequest]) (*connect.Response[apipb.StoryShare], error) {
	v, err := s.APIService.SetStoryShare(ctx, r.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(v), nil
}

// GetSharedStory adapts the story RPC for Connect.
func (s *ConnectServiceHandler) GetSharedStory(ctx context.Context, r *connect.Request[apipb.GetSharedStoryRequest]) (*connect.Response[apipb.GetSharedStoryResponse], error) {
	v, err := s.APIService.GetSharedStory(ctx, r.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(v), nil
}
