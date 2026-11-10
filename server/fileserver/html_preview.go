package fileserver

import (
	"html/template"
	"mime"
	"net/http"
	"strings"

	"github.com/usememos/memos/store"
)

func isHTMLAttachment(attachment *store.Attachment) bool {
	mimeType, _, _ := mime.ParseMediaType(attachment.Type)
	return strings.EqualFold(mimeType, "text/html")
}

// serveHTMLPreviewPage keeps the controls outside the untrusted document.
// Both this page and its content pass through the normal attachment permission check.
func serveHTMLPreviewPage(w http.ResponseWriter, r *http.Request, attachment *store.Attachment) error {
	downloadURL := *r.URL
	query := downloadURL.Query()
	query.Del("preview")
	downloadURL.RawQuery = query.Encode()
	contentURL := downloadURL
	query.Set("preview", "content")
	contentURL.RawQuery = query.Encode()

	setSecurityHeaders(w)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; frame-src 'self'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'")
	page := template.Must(template.New("preview").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>{{.Filename}} — HTML preview</title>
<style>body{margin:0;font:14px system-ui;display:flex;flex-direction:column;height:100dvh;background:#fff;color:#222}header{padding:12px 16px;border-bottom:1px solid #ddd;display:flex;gap:16px;align-items:center}strong{overflow:hidden;text-overflow:ellipsis;white-space:nowrap;flex:1}a{color:#2563eb}p{margin:0;padding:8px 16px;background:#f5f5f5;font-size:12px}iframe{border:0;width:100%;flex:1;min-height:0}</style></head>
<body><header><strong>{{.Filename}}</strong><a href="{{.DownloadURL}}" download>Download original / 下载原文件</a></header>
<p>Supports embedded scripts. External libraries, images and network requests are blocked. / 支持内嵌脚本；外部库、图片及网络请求被禁用。</p>
<iframe title="HTML preview" sandbox="allow-scripts" referrerpolicy="no-referrer" src="{{.ContentURL}}"></iframe></body></html>`))
	return page.Execute(w, struct {
		Filename, DownloadURL, ContentURL string
	}{attachment.Filename, downloadURL.String(), contentURL.String()})
}
