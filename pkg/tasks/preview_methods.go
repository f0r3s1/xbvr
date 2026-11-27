package tasks

import (
	"github.com/xbapps/xbvr/pkg/config"
)

// PreviewOptions are the preview settings every method receives; each method uses what applies to it.
type PreviewOptions struct {
	StartTime     int
	SnippetLength float64
	SnippetAmount int
	Resolution    int
	ExtraSnippet  bool
	Codec         string
}

type previewMethod func(inputFile, destFile, projection string, o PreviewOptions) error

// previewMethods lists the ways a card preview can be made; a method can be removed by deleting its entry and file.
var previewMethods = map[string]previewMethod{
	"classic": func(in, dest, projection string, o PreviewOptions) error {
		return RenderPreview(in, dest, projection, o.StartTime, o.SnippetLength, o.SnippetAmount, o.Resolution, o.ExtraSnippet)
	},
	"flat":  renderFlatPreview,
	"smart": renderSmartPreview,
}

// PreviewMethodValid reports whether a preview method exists.
func PreviewMethodValid(name string) bool {
	_, ok := previewMethods[name]
	return ok
}

// RenderPreviewWith renders with the named method, falling back to classic for an unknown name.
func RenderPreviewWith(method, inputFile, destFile, projection string, o PreviewOptions) (string, error) {
	if !PreviewMethodValid(method) {
		method = "classic"
	}
	return method, previewMethods[method](inputFile, destFile, projection, o)
}

// PreviewOptionsFromConfig returns the saved preview settings.
func PreviewOptionsFromConfig() PreviewOptions {
	p := config.Config.Library.Preview
	return PreviewOptions{StartTime: p.StartTime, SnippetLength: p.SnippetLength, SnippetAmount: p.SnippetAmount, Resolution: p.Resolution, ExtraSnippet: p.ExtraSnippet, Codec: p.Codec}
}
