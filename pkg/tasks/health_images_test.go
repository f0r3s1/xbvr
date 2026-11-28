package tasks

import (
	"testing"

	"github.com/xbapps/xbvr/pkg/models"
)

func TestDedupeURLs(t *testing.T) {
	got, n := dedupeURLs([]string{"https://a/main.jpg", "https://a/x.jpg", "http://a/x.jpg", " https://a/y.jpg ", "https://a/y.jpg/"})
	if n != 2 || len(got) != 3 || got[0] != "https://a/main.jpg" {
		t.Fatalf("got %v, %d removed", got, n)
	}
	imgs, n := dedupeSceneImages([]models.Image{{URL: "https://a/1.jpg"}, {URL: "https://a/1.jpg", Type: "gallery"}, {URL: "https://a/2.jpg"}})
	if n != 1 || len(imgs) != 2 {
		t.Fatalf("got %v, %d removed", imgs, n)
	}
}
