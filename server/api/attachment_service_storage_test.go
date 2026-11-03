package api

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/usememos/memos/internal/profile"
	storepb "github.com/usememos/memos/proto/gen/store"
	"github.com/usememos/memos/store"
)

func TestConvertAttachmentFromStoreStorageReferences(t *testing.T) {
	dataDir := t.TempDir()
	localAttachment := &store.Attachment{
		UID:         "local-image",
		Filename:    "image.png",
		Type:        "image/png",
		StorageType: storepb.AttachmentStorageType_LOCAL,
		Reference:   "assets/image.png",
	}
	require.Equal(t, localAttachment.Reference, convertAttachmentFromStore(localAttachment, dataDir).Reference)

	s3Attachment := &store.Attachment{
		UID:         "s3-image",
		Filename:    "image.png",
		Type:        "image/png",
		StorageType: storepb.AttachmentStorageType_S3,
		Reference:   "https://s3.example.com/presigned-secret",
	}
	require.Empty(t, convertAttachmentFromStore(s3Attachment, dataDir).ExternalLink)
	require.Empty(t, convertAttachmentFromStore(s3Attachment, dataDir).Reference)

	externalAttachment := &store.Attachment{
		UID:         "external-image",
		Filename:    "image.png",
		Type:        "image/png",
		StorageType: storepb.AttachmentStorageType_EXTERNAL,
		Reference:   "https://cdn.example.com/image.png",
	}
	require.Equal(t, externalAttachment.Reference, convertAttachmentFromStore(externalAttachment, dataDir).ExternalLink)
}

func TestConvertAttachmentFromStoreLocalReference(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), "memos")
	for _, tc := range []struct{ name, reference, want string }{
		{"relative", "assets/bear/photo.jpg", "assets/bear/photo.jpg"},
		{"absolute within data", filepath.Join(dataDir, "assets/bear/photo.jpg"), "assets/bear/photo.jpg"},
		{"absolute outside data", filepath.Join(dataDir, "../other/photo.jpg"), ""},
		{"sibling with same prefix", filepath.Join(dataDir+"-other", "photo.jpg"), ""},
		{"relative outside data", "../other/photo.jpg", ""},
		{"empty", "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			attachment := &store.Attachment{StorageType: storepb.AttachmentStorageType_LOCAL, Reference: tc.reference}
			require.Equal(t, tc.want, convertAttachmentFromStore(attachment, dataDir).Reference)
			require.Equal(t, tc.reference, attachment.Reference, "conversion must not alter the persisted path")
		})
	}
}

func TestSaveAttachmentContentPreservesTemplatePathKind(t *testing.T) {
	dataDir := t.TempDir()
	outsideDir := t.TempDir()
	for _, tc := range []struct{ name, template, want string }{
		{"relative", "assets/{filename}", "assets/photo.jpg"},
		{"absolute within data", filepath.Join(dataDir, "absolute/{filename}"), filepath.Join(dataDir, "absolute/photo.jpg")},
		{"absolute outside data", filepath.Join(outsideDir, "{filename}"), filepath.Join(outsideDir, "photo.jpg")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			content := []byte("image content")
			attachment := &store.Attachment{UID: "photo-id", Filename: "photo.jpg"}
			setting := &storepb.InstanceStorageSetting{
				FilepathTemplate: tc.template, DefaultStorageId: "local",
				Storages: []*storepb.Storage{{Id: "local", Type: storepb.StorageType_STORAGE_TYPE_LOCAL}},
			}
			err := saveAttachmentContent(context.Background(), &profile.Profile{Data: dataDir}, nil, attachment, setting, bytes.NewReader(content))
			require.NoError(t, err)
			require.Equal(t, filepath.ToSlash(tc.want), attachment.Reference)
			path := filepath.FromSlash(attachment.Reference)
			if !filepath.IsAbs(path) {
				path = filepath.Join(dataDir, path)
			}
			stored, err := os.ReadFile(path)
			require.NoError(t, err)
			require.Equal(t, content, stored)
		})
	}
}

func TestReplaceFilenameWithPathTemplateSupportsUsername(t *testing.T) {
	path := replaceFilenameWithPathTemplate("assets/{username}/{filename}", "photo.png", "alice")
	require.Equal(t, "assets/alice/photo.png", path)
}
