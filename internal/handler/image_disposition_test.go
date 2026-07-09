package handler

import (
	"net/url"
	"strings"
	"testing"
)

func TestAttachmentDispositionUTF8(t *testing.T) {
	name := "七姓婚摄y1_DSC_4919-白底.webp"
	got := attachmentDisposition(name)
	if !strings.Contains(got, "filename*=") {
		t.Fatalf("missing filename*: %s", got)
	}
	idx := strings.Index(got, "filename*=UTF-8''")
	if idx < 0 {
		t.Fatalf("bad disposition: %s", got)
	}
	encoded := strings.TrimPrefix(got[idx:], "filename*=UTF-8''")
	if semi := strings.Index(encoded, ";"); semi >= 0 {
		encoded = encoded[:semi]
	}
	decoded, err := url.PathUnescape(strings.TrimSpace(encoded))
	if err != nil {
		t.Fatal(err)
	}
	if decoded != name {
		t.Fatalf("got %q want %q", decoded, name)
	}
}
