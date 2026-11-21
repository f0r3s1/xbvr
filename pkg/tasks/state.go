package tasks

import (
	"io/fs"
	"slices"
	"strings"

	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/config"
	"github.com/xbapps/xbvr/ui"
)

func UpdateState() {
	// DLNA
	var dlnaImages []string
	fs.WalkDir(ui.Assets, "dist/dlna", func(path string, d fs.DirEntry, err error) error {
		if !d.IsDir() {
			dlnaImages = append(dlnaImages, path)
		}
		return nil
	})

	images := make([]string, 0, len(dlnaImages))
	for _, v := range dlnaImages {
		images = append(images, strings.Replace(strings.Split(v, "/")[2], ".png", "", -1))
	}
	running := IsDMSStarted()

	// GET /api/options/state calls this on every request; only write when
	// something actually changed instead of hitting the DB on each read.
	dlna := &config.State.DLNA
	if dlna.Running == running && slices.Equal(dlna.Images, images) && slices.Equal(dlna.RecentIP, config.RecentIPAddresses) {
		return
	}
	dlna.Images = images
	dlna.Running = running
	dlna.RecentIP = slices.Clone(config.RecentIPAddresses)

	config.SaveState()
}

func CalculateCacheSizes() {
	imgSize, _ := common.DirSize(common.ImgDir)
	heatmapThumbSize, _ := common.DirSize(common.HeatmapThumbnailDir)
	// Combine both image caches (imageproxy + heatmap thumbnails)
	config.State.CacheSize.Images = imgSize + heatmapThumbSize
	config.State.CacheSize.Previews, _ = common.DirSize(common.VideoPreviewDir)
	config.State.CacheSize.SearchIndex, _ = common.DirSize(common.IndexDirV2)

	config.SaveState()
}
