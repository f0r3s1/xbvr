package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/emicklei/go-restful/v3"

	"github.com/xbapps/xbvr/pkg/config"
	"github.com/xbapps/xbvr/pkg/models"
	"github.com/xbapps/xbvr/pkg/session"
)

type HeresphereScan struct {
	ScanData []HeresphereScanData `json:"scanData"`
}

type HeresphereScanData struct {
	Link         string          `json:"link"`
	Title        string          `json:"title"`
	DateReleased string          `json:"dateReleased,omitempty"`
	DateAdded    string          `json:"dateAdded,omitempty"`
	Duration     float64         `json:"duration,omitempty"`
	Rating       float64         `json:"rating,omitempty"`
	IsFavorite   bool            `json:"isFavorite,omitempty"`
	Tags         []HeresphereTag `json:"tags,omitempty"`
}

type HerespherePlaybackEvent struct {
	Username string  `json:"username"`
	ID       string  `json:"id"`
	Event    int     `json:"event"`
	Time     float64 `json:"time"`
	Speed    float64 `json:"speed"`
}

type heresphereScanRow struct {
	ID          uint
	Title       string
	ReleaseDate time.Time
	AddedDate   time.Time
	Duration    int
	StarRating  float64
	Favourite   bool
	Watchlist   bool
	IsScripted  bool
	Site        string
}

type heresphereScanName struct {
	SceneID uint
	Name    string
}

func (i HeresphereResource) getHeresphereScan(req *restful.Request, resp *restful.Response) {
	if !config.Config.Interfaces.DeoVR.Enabled {
		return
	}
	resp.WriteHeaderAndEntity(http.StatusOK, HeresphereScan{ScanData: buildHeresphereScan(requestBaseURL(req))})
}

func buildHeresphereScan(baseURL string) []HeresphereScanData {
	db, _ := models.GetDB()
	defer db.Close()

	cond := "deleted_at is null and is_available = ? and is_accessible = ?"
	args := []interface{}{true, true}
	if config.Config.Web.SceneTrailerlist {
		cond = "deleted_at is null and ((is_available = ? and is_accessible = ?) or (is_available = ? and trailerlist = ? and trailer_source != ''))"
		args = append(args, false, true)
	}
	inScan := "in (select id from scenes where " + cond + ")"

	var rows []heresphereScanRow
	db.Table("scenes").Where(cond, args...).
		Select("id, title, release_date, added_date, duration, star_rating, favourite, watchlist, is_scripted, site").Scan(&rows)
	if len(rows) == 0 {
		return []HeresphereScanData{}
	}

	var durations []struct {
		SceneID  uint
		Duration float64
	}
	db.Table("files").Select("scene_id, max(video_duration) as duration").
		Where("type = 'video' and scene_id "+inScan, args...).Group("scene_id").Scan(&durations)
	videoDuration := make(map[uint]float64, len(durations))
	for _, d := range durations {
		videoDuration[d.SceneID] = d.Duration
	}

	tags := make(map[uint][]HeresphereTag, len(rows))
	var cast []heresphereScanName
	db.Table("scene_cast").Select("scene_cast.scene_id, actors.name").
		Joins("join actors on actors.id = scene_cast.actor_id").
		Where("scene_cast.scene_id "+inScan, args...).Scan(&cast)
	for _, c := range cast {
		tags[c.SceneID] = append(tags[c.SceneID], HeresphereTag{Name: "Talent:" + c.Name})
	}
	var categories []heresphereScanName
	db.Table("scene_tags").Select("scene_tags.scene_id, tags.name").
		Joins("join tags on tags.id = scene_tags.tag_id").
		Where("scene_tags.scene_id "+inScan, args...).Scan(&categories)
	for _, c := range categories {
		tags[c.SceneID] = append(tags[c.SceneID], HeresphereTag{Name: "Category:" + c.Name})
	}

	var plays []struct {
		SceneID uint
		Count   int
	}
	db.Table("histories").Select("scene_id, count(*) as count").
		Where("scene_id "+inScan, args...).Group("scene_id").Scan(&plays)
	playCount := make(map[uint]int, len(plays))
	for _, p := range plays {
		playCount[p.SceneID] = p.Count
	}

	out := make([]HeresphereScanData, 0, len(rows))
	for _, r := range rows {
		scene := models.Scene{Title: r.Title}
		scene.ID = r.ID
		title := r.Title
		if r.IsScripted {
			title = scene.GetFunscriptTitle()
		}
		sceneTags := append([]HeresphereTag{{Name: "Studio:" + r.Site}}, tags[r.ID]...)
		if r.Watchlist {
			sceneTags = append(sceneTags, HeresphereTag{Name: "Feature:Watchlist"})
		}
		sceneTags = append(sceneTags, heresphereTagPlayed(playCount[r.ID]))
		duration := videoDuration[r.ID]
		if duration == 0 {
			duration = float64(r.Duration * 60)
		}
		item := HeresphereScanData{
			Link:       fmt.Sprintf("%v/heresphere/%v", baseURL, r.ID),
			Title:      title,
			Duration:   duration * 1000,
			Rating:     r.StarRating,
			IsFavorite: r.Favourite,
			Tags:       sceneTags,
		}
		if r.ReleaseDate.Year() > 1900 {
			item.DateReleased = r.ReleaseDate.Format("2006-01-02")
		}
		if r.AddedDate.Year() > 1900 {
			item.DateAdded = r.AddedDate.Format("2006-01-02")
		}
		out = append(out, item)
	}
	return out
}

func (i HeresphereResource) postHeresphereEvent(req *restful.Request, resp *restful.Response) {
	if !config.Config.Interfaces.DeoVR.Enabled || !config.Config.Interfaces.DeoVR.TrackWatchTime {
		resp.WriteHeader(http.StatusNoContent)
		return
	}

	var event HerespherePlaybackEvent
	if err := json.Unmarshal(heresphereRequestBody(req), &event); err != nil {
		resp.WriteHeader(http.StatusBadRequest)
		return
	}
	// events carry the username but no password, so only the name can be checked
	if isDeoAuthEnabled() && event.Username != config.Config.Interfaces.DeoVR.Username {
		resp.WriteHeader(http.StatusUnauthorized)
		return
	}

	sceneID, err := strconv.ParseUint(req.PathParameter("scene-id"), 10, 32)
	if err != nil {
		resp.WriteHeader(http.StatusBadRequest)
		return
	}
	session.TrackSessionFromHereSphere(uint(sceneID), event.Event)
	resp.WriteHeader(http.StatusNoContent)
}
