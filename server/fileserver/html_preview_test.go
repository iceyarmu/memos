package fileserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/usememos/memos/internal/httphandler"
	"github.com/usememos/memos/internal/testutil/fakes3"
	apipb "github.com/usememos/memos/proto/gen/api"
	"github.com/usememos/memos/server/auth"
	"github.com/usememos/memos/store"
)

func TestHTMLPreview(t *testing.T) {
	for _, storage := range []string{"database", "local", "s3"} {
		t.Run(storage, func(t *testing.T) {
			ctx := context.Background()
			svc, fs, stores, cleanup := newShareAttachmentTestServices(ctx, t)
			defer cleanup()
			if storage == "local" {
				configureLocalStorage(ctx, t, stores)
			} else if storage == "s3" {
				configureFakeS3Storage(ctx, t, stores, fakes3.New(t, "file-server-attachments"))
			}
			creator, err := stores.CreateUser(ctx, &store.User{Username: "preview-owner", Role: store.RoleUser})
			require.NoError(t, err)
			creatorCtx := context.WithValue(ctx, auth.UserIDContextKey, creator.ID)
			content := []byte("<!doctype html><button onclick=\"this.textContent='OK'\">Run</button>")
			attachment, err := svc.CreateAttachment(creatorCtx, &apipb.CreateAttachmentRequest{Attachment: &apipb.Attachment{
				Filename: "report.html", Type: "text/html", Content: content,
			}})
			require.NoError(t, err)
			memo, err := svc.CreateMemo(creatorCtx, &apipb.CreateMemoRequest{Memo: &apipb.Memo{
				Content: "private report", Visibility: apipb.Visibility_PRIVATE, Attachments: []*apipb.Attachment{{Name: attachment.Name}},
			}})
			require.NoError(t, err)
			share, err := svc.CreateMemoShare(creatorCtx, &apipb.CreateMemoShareRequest{Parent: memo.Name, MemoShare: &apipb.MemoShare{}})
			require.NoError(t, err)
			mux := http.NewServeMux()
			fs.RegisterRoutes(mux, http.HandlerFunc(httphandler.NotFound))
			base := "/file/" + attachment.Name + "/report.html"
			for _, mode := range []string{"true", "content"} {
				denied := httptest.NewRecorder()
				mux.ServeHTTP(denied, httptest.NewRequest(http.MethodGet, base+"?preview="+mode, nil))
				require.NotEqual(t, http.StatusOK, denied.Code)
				require.NotContains(t, denied.Body.String(), string(content))
			}
			shared := base + "?share_token=" + share.Name[strings.LastIndex(share.Name, "/")+1:]
			page := httptest.NewRecorder()
			mux.ServeHTTP(page, httptest.NewRequest(http.MethodGet, shared+"&preview=true", nil))
			require.Equal(t, http.StatusOK, page.Code)
			require.Contains(t, page.Body.String(), `sandbox="allow-scripts"`)
			require.Contains(t, page.Body.String(), "preview=content")
			require.Contains(t, page.Body.String(), "share_token=")
			preview := httptest.NewRecorder()
			mux.ServeHTTP(preview, httptest.NewRequest(http.MethodGet, shared+"&preview=content", nil))
			require.Equal(t, http.StatusOK, preview.Code)
			require.Equal(t, content, preview.Body.Bytes())
			require.Equal(t, "text/html; charset=utf-8", preview.Header().Get("Content-Type"))
			require.Equal(t, "inline", preview.Header().Get("Content-Disposition"))
			policy := preview.Header().Get("Content-Security-Policy")
			require.Contains(t, policy, "sandbox allow-scripts;")
			require.NotContains(t, policy, "allow-same-origin")
			require.Contains(t, policy, "connect-src 'none'")
			original := httptest.NewRecorder()
			mux.ServeHTTP(original, httptest.NewRequest(http.MethodGet, shared, nil))
			require.Equal(t, http.StatusOK, original.Code)
			require.Equal(t, content, original.Body.Bytes())
			require.Contains(t, original.Header().Get("Content-Disposition"), "attachment;")
			require.Equal(t, "application/octet-stream", original.Header().Get("Content-Type"))
		})
	}
}
