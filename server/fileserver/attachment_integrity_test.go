package fileserver

import (
	"context"
	"crypto/md5"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/usememos/memos/internal/httphandler"
	"github.com/usememos/memos/internal/testutil"
	"github.com/usememos/memos/internal/testutil/fakes3"
	apipb "github.com/usememos/memos/proto/gen/api"
	"github.com/usememos/memos/server/auth"
	"github.com/usememos/memos/store"
)

func TestAttachmentUploadDownloadIntegrity(t *testing.T) {
	// A valid JPEG with an APP1 metadata segment catches both metadata removal
	// and lossy re-encoding, even when the client declares a generic MIME type.
	jpeg := testutil.BuildJPEG(20, 10)
	content := append([]byte{}, jpeg[:2]...)
	content = append(content, []byte("\xff\xe1\x00\x0cExif\x00\x00test")...)
	content = append(content, jpeg[2:]...)
	for _, storage := range []string{"local", "s3"} {
		for _, chunked := range []bool{false, true} {
			for _, mimeType := range []string{"image/jpeg", "application/octet-stream"} {
				t.Run(fmt.Sprintf("%s/chunked=%t/%s", storage, chunked, mimeType), func(t *testing.T) {
					ctx := context.Background()
					svc, fs, stores, cleanup := newShareAttachmentTestServices(ctx, t)
					defer cleanup()
					defer svc.CloseUploads()
					if storage == "s3" {
						configureFakeS3Storage(ctx, t, stores, fakes3.New(t, "file-server-attachments"))
					} else {
						configureLocalStorage(ctx, t, stores)
					}
					user, err := stores.CreateUser(ctx, &store.User{Username: "integrity", Role: store.RoleUser})
					require.NoError(t, err)
					userCtx := context.WithValue(ctx, auth.UserIDContextKey, user.ID)
					metadata := &apipb.Attachment{Filename: "original.jpg", Type: mimeType}
					var attachment *apipb.Attachment
					if chunked {
						split := len(content) / 2
						first, err := svc.UploadAttachment(userCtx, &apipb.UploadAttachmentRequest{
							Upload: &apipb.UploadAttachmentRequest_Spec{Spec: &apipb.UploadAttachmentSpec{Attachment: metadata, TotalSize: int64(len(content))}},
							Data:   content[:split],
						})
						require.NoError(t, err)
						last, err := svc.UploadAttachment(userCtx, &apipb.UploadAttachmentRequest{
							Upload:      &apipb.UploadAttachmentRequest_UploadId{UploadId: first.UploadId},
							WriteOffset: int64(split), Data: content[split:], FinishWrite: true,
						})
						require.NoError(t, err)
						attachment = last.Attachment
					} else {
						metadata.Content = content
						attachment, err = svc.CreateAttachment(userCtx, &apipb.CreateAttachmentRequest{Attachment: metadata})
						require.NoError(t, err)
					}
					require.EqualValues(t, len(content), attachment.Size)
					_, err = svc.CreateMemo(userCtx, &apipb.CreateMemoRequest{Memo: &apipb.Memo{
						Content: "original attachment", Visibility: apipb.Visibility_PUBLIC,
						Attachments: []*apipb.Attachment{{Name: attachment.Name}},
					}})
					require.NoError(t, err)
					mux := http.NewServeMux()
					fs.RegisterRoutes(mux, http.HandlerFunc(httphandler.NotFound))
					rec := httptest.NewRecorder()
					mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/file/"+attachment.Name+"/"+attachment.Filename, nil))
					require.Equal(t, http.StatusOK, rec.Code)
					require.Equal(t, md5.Sum(content), md5.Sum(rec.Body.Bytes()))
					require.Equal(t, content, rec.Body.Bytes())
				})
			}
		}
	}
}
