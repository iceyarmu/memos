package api

import (
	apipb "github.com/usememos/memos/proto/gen/api"
	"github.com/usememos/memos/store"
)

func convertStoryFromStore(item *store.Story) *apipb.Story {
	return &apipb.Story{IntervalUnit: item.IntervalUnit, Name: "stories/" + item.UID, Uid: item.UID, Title: item.Title, Description: item.Description, Icon: item.Icon, Color: item.Color, Visibility: "PRIVATE", CreatedTs: item.CreatedTs, UpdatedTs: item.UpdatedTs}
}
