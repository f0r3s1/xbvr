package models

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/jinzhu/gorm"
	"github.com/markphelps/optional"
)

type Actor struct {
	ID        uint      `gorm:"primary_key" json:"id" xbvrbackup:"-"`
	CreatedAt time.Time `json:"-" xbvrbackup:"-"`
	UpdatedAt time.Time `json:"-" xbvrbackup:"-"`

	Name   string  `gorm:"unique_index" json:"name" xbvrbackup:"name"`
	Scenes []Scene `gorm:"many2many:scene_cast;" json:"scenes" xbvrbackup:"-"`
	Count  int     `json:"count" xbvrbackup:"-"`

	AvailCount int `json:"avail_count" xbvrbackup:"-"`

	ImageUrl   string  `json:"image_url" xbvrbackup:"image_url"`
	ImageArr   string  `json:"image_arr" sql:"type:text;" xbvrbackup:"image_arr"`
	StarRating float64 `json:"star_rating" xbvrbackup:"star_rating"`
	Favourite  bool    `json:"favourite" gorm:"default:false" xbvrbackup:"favourite"`
	Watchlist  bool    `json:"watchlist" gorm:"default:false" xbvrbackup:"watchlist"`

	BirthDate   time.Time `json:"birth_date" xbvrbackup:"birth_date"`
	Nationality string    `json:"nationality" xbvrbackup:"nationality"`
	Ethnicity   string    `json:"ethnicity" xbvrbackup:"ethnicity"`
	EyeColor    string    `json:"eye_color" xbvrbackup:"eyeColor"`
	HairColor   string    `json:"hair_color" xbvrbackup:"hairColor"`
	Height      int       `json:"height" xbvrbackup:"height"`
	Weight      int       `json:"weight" xbvrbackup:"weight"`
	CupSize     string    `json:"cup_size" xbvrbackup:"cup_size"`
	BandSize    int       `json:"band_size" xbvrbackup:"band_size"`
	WaistSize   int       `json:"waist_size" xbvrbackup:"waist_size"`
	HipSize     int       `json:"hip_size" xbvrbackup:"hip_size"`
	BreastType  string    `json:"breast_type" xbvrbackup:"breast_type"`
	StartYear   int       `json:"start_year" xbvrbackup:"start_year"`
	EndYear     int       `json:"end_year" xbvrbackup:"end_year"`
	Tattoos     string    `json:"tattoos" sql:"type:text;"  xbvrbackup:"tattoos"`
	Piercings   string    `json:"piercings" sql:"type:text;" xbvrbackup:"biercings"`

	Biography string `json:"biography" sql:"type:text;" xbvrbackup:"biography"`
	Aliases   string `json:"aliases" gorm:"size:1000"  xbvrbackup:"aliases"`
	Gender    string `json:"gender" xbvrbackup:"gender"`
	URLs      string `json:"urls" sql:"type:text;" xbvrbackup:"urls"`

	SceneRatingAverage string `json:"scene_rating_average" gorm:"-" `
	VisibleSceneCount  int    `json:"-" gorm:"-"`
	AvailableSceneCnt  int    `json:"-" gorm:"-"`
	AkaGroups          []Aka  `gorm:"many2many:actor_akas;" json:"aka_groups" xbvrbackup:"-"`
}

type RequestActorList struct {
	DlState        optional.String   `json:"dlState"`
	Limit          optional.Int      `json:"limit"`
	Offset         optional.Int      `json:"offset"`
	Lists          []optional.String `json:"lists"`
	Cast           []optional.String `json:"cast"`
	Sites          []optional.String `json:"sites"`
	Tags           []optional.String `json:"tags"`
	Attributes     []optional.String `json:"attributes"`
	JumpTo         optional.String   `json:"jumpTo"`
	Name           optional.String   `json:"name"`
	MinAge         optional.Int      `json:"min_age"`
	MaxAge         optional.Int      `json:"max_age"`
	MinHeight      optional.Int      `json:"min_height"`
	MaxHeight      optional.Int      `json:"max_height"`
	MinWeight      optional.Int      `json:"min_weight"`
	MaxWeight      optional.Int      `json:"max_weight"`
	MinCount       optional.Int      `json:"min_count"`
	MaxCount       optional.Int      `json:"max_count"`
	MinAvail       optional.Int      `json:"min_avail"`
	MaxAvail       optional.Int      `json:"max_avail"`
	MinRating      optional.Float64  `json:"min_rating"`
	MaxRating      optional.Float64  `json:"max_rating"`
	MinSceneRating optional.Float64  `json:"min_scene_rating"`
	MaxSceneRating optional.Float64  `json:"max_scene_rating"`
	Sort           optional.String   `json:"sort"`
}
type ResponseActorList struct {
	Results            int     `json:"results"`
	Actors             []Actor `json:"actors"`
	CountAny           int     `json:"count_any"`
	CountAvailable     int     `json:"count_available"`
	CountDownloaded    int     `json:"count_downloaded"`
	CountNotDownloaded int     `json:"count_not_downloaded"`
	CountHidden        int     `json:"count_hidden"`
	Offset             int     `json:"offset"`
}

type ActorLink struct {
	Url  string `json:"url"`
	Type string `json:"type"`
}

func (i *Actor) Save() error {
	commonDb, _ := GetCommonDB()

	var err error = retry.Do(
		func() error {
			err := commonDb.Save(&i).Error
			if err != nil {
				return err
			}
			return nil
		},
	)

	if err != nil {
		log.Fatal("Failed to save ", err)
	}

	return nil
}

func (i *Actor) CountActorTags() {
	commonDb, _ := GetCommonDB()

	type CountResults struct {
		ID            int
		Cnt           int
		Existingcnt   int
		IsAvailable   int
		Existingavail int
	}

	var results []CountResults

	commonDb.Model(&Actor{}).
		Select("actors.id, actors.count as existingcnt, count(scenes.id) cnt, coalesce(sum(scenes.is_available), 0) is_available, actors.avail_count as existingavail").
		Group("actors.id").
		Joins("left join scene_cast on scene_cast.actor_id = actors.id").
		Joins("left join scenes on scenes.id=scene_cast.scene_id and scenes.deleted_at is null").
		Scan(&results)

	for i := range results {
		var actor Actor
		if results[i].Cnt != results[i].Existingcnt || results[i].IsAvailable != results[i].Existingavail {
			commonDb.First(&actor, results[i].ID)
			actor.Count = results[i].Cnt
			actor.AvailCount = results[i].IsAvailable
			actor.Save()
		}
	}
}
func QueryActorFull(r RequestActorList) ResponseActorList {
	var actors []Actor
	r.Limit = optional.NewInt(100)
	r.Offset = optional.NewInt(0)

	q := QueryActors(r, true)
	actors = q.Actors

	for len(actors) < q.Results {
		r.Offset = optional.NewInt(len(actors))
		q := QueryActors(r, true)
		actors = append(actors, q.Actors...)
	}

	q.Actors = actors
	return q
}

// The unary plus on is_hidden keeps SQLite from driving these subqueries through the low-selectivity is_hidden index.
const (
	actorScenesSQL        = "from scene_cast sc join scenes s on s.id = sc.scene_id where sc.actor_id = actors.id"
	actorVisibleScenesSQL = actorScenesSQL + " and +s.is_hidden = 0"
	actorSceneRatingSQL   = "(select avg(s.star_rating) " + actorVisibleScenesSQL + " and s.star_rating > 0)"
	actorStashLinksSQL    = "(select count(*) from external_reference_links erl where erl.internal_db_id = actors.id and erl.external_source = 'stashdb performer')"
)

var actorAttributeColumns = [][2]string{{"Cup Size ", "cup_size"}, {"Hair Color ", "hair_color"}, {"Eye Color ", "eye_color"}, {"Nationality ", "nationality"}, {"Gender ", "gender"}, {"Ethnicity ", "ethnicity"}, {"Breast Type ", "breast_type"}}

// actorAttributeCond returns the SQL condition for an attribute filter name (without its "!"/"&" prefix), or "" for an unknown name.
func actorAttributeCond(name string) (string, []interface{}) {
	for _, col := range actorAttributeColumns {
		if value, ok := strings.CutPrefix(name, col[0]); ok {
			if col[1] == "nationality" {
				for _, c := range GetCountryList() {
					if c.Name == value {
						value = c.Code
					}
				}
			}
			return "actors." + col[1] + " = ?", []interface{}{value}
		}
	}
	if rating, ok := strings.CutPrefix(name, "Rating "); ok {
		if v, err := strconv.ParseFloat(rating, 64); err == nil {
			return "actors.star_rating = ?", []interface{}{v}
		}
	}
	switch name {
	case "In Watchlist":
		return "actors.watchlist = 1", nil
	case "Is Favourite":
		return "actors.favourite = 1", nil
	case "Is Scripted":
		return "exists (select 1 " + actorVisibleScenesSQL + " and s.is_scripted = 1)", nil
	case "Has Rating":
		return "actors.star_rating > 0", nil
	case "Aka Group":
		return "actors.name like 'aka:%'", nil
	case "In An Aka Group":
		return "exists (select 1 from actor_akas where actor_akas.actor_id = actors.id)", nil
	case "Has Image":
		return "coalesce(actors.image_url, '') <> ''", nil
	case "Possible Aka":
		// the StashDB performer is linked to more than one actor here
		return "exists (select 1 from external_reference_links erl join external_reference_links erl2 on erl2.external_id = erl.external_id and erl2.internal_db_id <> erl.internal_db_id where erl.internal_db_id = actors.id and erl.external_source = 'stashdb performer')", nil
	case "Has Stashdb Link":
		return actorStashLinksSQL + " > 0", nil
	case "Multiple Stashdb Links":
		// linked to more than one StashDB performer, i.e. duplicates in StashDB
		return actorStashLinksSQL + " > 1", nil
	case "SLR Scraper":
		return "exists (select 1 from external_reference_links erl where erl.internal_db_id = actors.id and erl.external_source = 'slr scrape')", nil
	case "VRPorn Scraper":
		return "exists (select 1 from external_reference_links erl where erl.internal_db_id = actors.id and erl.external_source = 'vrporn scrape')", nil
	case "Has Tattoo":
		return "coalesce(actors.tattoos, '') not in ('', '[]')", nil
	case "Has Piercing":
		return "coalesce(actors.piercings, '') not in ('', '[]')", nil
	case "No Cup Size":
		return "coalesce(actors.cup_size, '') = ''", nil
	}
	return "", nil
}

// actorLetterCond is the first-letter filter ("#" = anything not starting with A-Z).
func actorLetterCond(letter string) (string, []interface{}) {
	if letter == "#" {
		return "(actors.name < 'A' or (actors.name >= '[' and actors.name < 'a') or actors.name >= '{')", nil
	}
	c := letter[0]
	if c < 'A' || c > 'Z' {
		return "", nil
	}
	lower := c + ('a' - 'A')
	return "((actors.name >= ? and actors.name < ?) or (actors.name >= ? and actors.name < ?))",
		[]interface{}{string(c), string(c + 1), string(lower), string(lower + 1)}
}

func QueryActors(r RequestActorList, enablePreload bool) ResponseActorList {
	limit := r.Limit.OrElse(100)
	offset := r.Offset.OrElse(0)

	commonDb, _ := GetCommonDB()
	tx := commonDb.Model(&Actor{})

	for _, i := range r.Lists {
		switch i.OrElse("") {
		case "watchlist":
			tx = tx.Where("actors.watchlist = ?", true)
		case "favourite":
			tx = tx.Where("actors.favourite = ?", true)
		}
	}

	if strings.EqualFold(r.DlState.OrElse(""), "available") {
		tx = tx.Where("actors.avail_count > 0")
	}
	if getConfig(commonDb).Web.HideMaleActors {
		tx = tx.Where("lower(coalesce(actors.gender, '')) <> 'male'")
	}

	// Attributes: plain ones are OR-ed, "&" (must have) and "!" (must not) are AND-ed.
	var orConds, andConds []string
	var orArgs, andArgs []interface{}
	for _, attribute := range r.Attributes {
		name := attribute.OrElse("")
		cond, args := actorAttributeCond(strings.TrimLeft(name, "!&"))
		if cond == "" {
			continue
		}
		switch {
		case strings.HasPrefix(name, "!"):
			andConds = append(andConds, "not ("+cond+")")
			andArgs = append(andArgs, args...)
		case strings.HasPrefix(name, "&"):
			andConds = append(andConds, "("+cond+")")
			andArgs = append(andArgs, args...)
		default:
			orConds = append(orConds, "("+cond+")")
			orArgs = append(orArgs, args...)
		}
	}
	if len(orConds) > 0 {
		tx = tx.Where("("+strings.Join(orConds, " or ")+")", orArgs...)
	}
	if len(andConds) > 0 {
		tx = tx.Where(strings.Join(andConds, " and "), andArgs...)
	}

	var cast, excludedCast []string
	for _, i := range r.Cast {
		if name := i.OrElse(""); strings.HasPrefix(name, "!") {
			excludedCast = append(excludedCast, name[1:])
		} else if name != "" {
			cast = append(cast, name)
		}
	}
	if len(cast) > 0 {
		tx = tx.Where("actors.name IN (?)", cast)
	}
	if len(excludedCast) > 0 {
		tx = tx.Where("actors.name NOT IN (?)", excludedCast)
	}

	if r.MinAge.OrElse(0) > 18 || r.MaxAge.OrElse(100) < 100 {
		startRange := time.Now().AddDate(r.MinAge.OrElse(0)*-1, 0, 0)
		endRange := time.Now().AddDate(r.MaxAge.OrElse(0)*-1, 0, 0)
		tx = tx.Where("actors.birth_date <= ? and actors.birth_date >= ?", startRange, endRange)
	}
	if r.MinHeight.OrElse(120) > 120 {
		tx = tx.Where("actors.height >= ?", r.MinHeight.OrElse(120))
	}
	if r.MaxHeight.OrElse(220) < 220 {
		tx = tx.Where("actors.height <= ?", r.MaxHeight.OrElse(220))
	}
	if r.MinWeight.OrElse(25) > 25 {
		tx = tx.Where("actors.weight >= ?", r.MinWeight.OrElse(25))
	}
	if r.MaxWeight.OrElse(150) < 150 {
		tx = tx.Where("actors.weight <= ?", r.MaxWeight.OrElse(150))
	}
	if r.MinCount.OrElse(0) > 0 {
		tx = tx.Where("actors.`count` >= ?", r.MinCount.OrElse(0))
	}
	if r.MaxCount.OrElse(150) < 150 {
		tx = tx.Where("actors.`count` <= ?", r.MaxCount.OrElse(150))
	}
	if r.MinAvail.OrElse(0) > 0 {
		tx = tx.Where("actors.avail_count >= ?", r.MinAvail.OrElse(0))
	}
	if r.MaxAvail.OrElse(150) < 150 {
		tx = tx.Where("actors.avail_count <= ?", r.MaxAvail.OrElse(150))
	}
	if r.MinRating.OrElse(0) > 0 {
		tx = tx.Where("actors.star_rating >= ?", r.MinRating.OrElse(0))
	}
	if r.MaxRating.OrElse(5) < 5 {
		tx = tx.Where("actors.star_rating <= ?", r.MaxRating.OrElse(5))
	}
	if r.MinSceneRating.OrElse(0) > 0 {
		tx = tx.Where(actorSceneRatingSQL+" >= ?", r.MinSceneRating.OrElse(0))
	}
	if r.MaxSceneRating.OrElse(5) < 5 {
		tx = tx.Where(actorSceneRatingSQL+" <= ?", r.MaxSceneRating.OrElse(5))
	}

	if name := strings.TrimSpace(r.Name.OrElse("")); name != "" {
		tx = tx.Where("LOWER(actors.name) LIKE ?", "%"+strings.ToLower(name)+"%")
	}

	if letter := strings.ToUpper(strings.TrimSpace(r.JumpTo.OrElse(""))); letter != "" {
		if cond, args := actorLetterCond(letter[:1]); cond != "" {
			tx = tx.Where(cond, args...)
		}
	}

	// Site filters are semi-joins (IN subquery driven by the scraper_id index), so an actor never appears twice and the query needs no GROUP BY / DISTINCT.
	const actorSiteSQL = "actors.id in (select sc.actor_id from scenes s join scene_cast sc on sc.scene_id = s.id where s.scraper_id in (select id from sites where name in (?)))"
	var sites []string
	for _, i := range r.Sites {
		site := i.OrElse("")
		switch {
		case strings.HasPrefix(site, "&"):
			tx = tx.Where(actorSiteSQL, []string{site[1:]})
		case strings.HasPrefix(site, "!"):
			tx = tx.Where("not "+actorSiteSQL, []string{site[1:]})
		case site != "":
			sites = append(sites, site)
		}
	}
	if len(sites) > 0 {
		tx = tx.Where(actorSiteSQL, sites)
	}

	// Ratings-based sorts only list rated actors, so they belong to the filter.
	sort := r.Sort.OrElse("")
	if sort == "rating_desc" || sort == "rating_asc" {
		tx = tx.Where("actors.star_rating > 0")
	}

	out := ResponseActorList{Offset: offset}
	tx.Count(&out.Results)

	switch sort {
	case "name_desc":
		tx = tx.Order("actors.name desc")
	case "rating_desc":
		tx = tx.Order("actors.star_rating desc, actors.name")
	case "rating_asc":
		tx = tx.Order("actors.star_rating asc, actors.name")
	case "scene_rating_desc":
		tx = tx.Order(actorSceneRatingSQL + " desc, (select count(*) " + actorVisibleScenesSQL + " and s.star_rating > 0) desc, actors.`count` desc")
	case "scene_release_desc":
		tx = tx.Order("coalesce((select max(s.release_date) " + actorScenesSQL + "), '1970-01-01') desc, actors.avail_count desc, actors.`count` desc")
	case "scene_added_desc":
		tx = tx.Order("coalesce((select max(s.created_at) " + actorScenesSQL + "), '1970-01-01') desc, actors.avail_count desc, actors.`count` desc")
	case "file_added_desc":
		tx = tx.Order("coalesce((select max(s.added_date) " + actorScenesSQL + "), '1970-01-01') desc, actors.avail_count desc, actors.`count` desc")
	case "birthday_desc":
		tx = tx.Order("actors.birth_date desc")
	case "birthday_asc":
		tx = tx.Order("actors.birth_date asc")
	case "added_desc":
		tx = tx.Order("actors.created_at desc")
	case "added_asc":
		tx = tx.Order("actors.created_at asc")
	case "modified_desc":
		tx = tx.Order("actors.updated_at desc")
	case "modified_asc":
		tx = tx.Order("actors.updated_at asc")
	case "random":
		if dbConn.Driver == "mysql" {
			tx = tx.Order("rand()")
		} else {
			tx = tx.Order("random()")
		}
	case "scene_count_desc":
		tx = tx.Order("actors.`count` desc, actors.name")
	case "scene_available_desc":
		tx = tx.Order("actors.avail_count desc, actors.name")
	default:
		tx = tx.Order("actors.name asc")
	}

	if enablePreload {
		tx = tx.Preload("Scenes", func(db *gorm.DB) *gorm.DB {
			return db.Order("release_date DESC").Where("+scenes.is_hidden = 0")
		})
	}

	tx.Select("actors.*").Limit(limit).Offset(offset).Find(&out.Actors)
	fillSceneRatingAverages(out.Actors)
	return out
}

// fillSceneRatingAverages sets the visible scene stats for a page of actors in one query.
func fillSceneRatingAverages(actors []Actor) {
	if len(actors) == 0 {
		return
	}
	ids := make([]uint, len(actors))
	for i, a := range actors {
		ids[i] = a.ID
	}
	var rows []struct {
		ActorID   uint
		Cnt       int
		Available int
		Avg       float64
	}
	commonDb, _ := GetCommonDB()
	commonDb.Raw(`select sc.actor_id, count(*) as cnt, count(case when s.is_available then 1 end) as available,
		coalesce(avg(case when s.star_rating > 0 then s.star_rating end), 0) as avg
		from scene_cast sc join scenes s on s.id = sc.scene_id
		where sc.actor_id in (?) and +s.is_hidden = 0 and s.deleted_at is null group by sc.actor_id`, ids).Scan(&rows)
	byID := make(map[uint]int, len(rows))
	for i, row := range rows {
		byID[row.ActorID] = i
	}
	for i := range actors {
		actors[i].SceneRatingAverage = "0"
		if j, ok := byID[actors[i].ID]; ok {
			actors[i].SceneRatingAverage = strconv.FormatFloat(rows[j].Avg, 'f', -1, 64)
			actors[i].VisibleSceneCount = rows[j].Cnt
			actors[i].AvailableSceneCnt = rows[j].Available
		}
	}
}

func (o *Actor) GetIfExist(id string) error {
	commonDb, _ := GetCommonDB()

	return commonDb.
		Preload("Scenes", func(db *gorm.DB) *gorm.DB {
			return db.Where("is_hidden = 0")
		}).
		Where(&Actor{Name: id}).First(o).Error
}

func (o *Actor) GetIfExistByPK(id uint) error {
	commonDb, _ := GetCommonDB()

	return commonDb.
		Preload("Scenes", func(db *gorm.DB) *gorm.DB {
			return db.Where("is_hidden = 0")
		}).
		Where(&Actor{ID: id}).First(o).Error
}

func (o *Actor) GetIfExistByPKWithSceneAvg(id uint) error {
	commonDb, _ := GetCommonDB()

	err := commonDb.
		Preload("Scenes", func(db *gorm.DB) *gorm.DB {
			return db.Order("release_date DESC").Where("+scenes.is_hidden = 0")
		}).
		Where(&Actor{ID: id}).First(o).Error
	if err == nil {
		actors := []Actor{*o}
		fillSceneRatingAverages(actors)
		o.SceneRatingAverage = actors[0].SceneRatingAverage
	}
	return err
}

func (i *Actor) AddToImageArray(newValue string) bool {
	var array []string
	if newValue == "" {
		return false
	}
	newValue = strings.TrimSuffix(newValue, "/")
	if i.ImageArr == "" {
		i.ImageArr = "[]"
	}

	json.Unmarshal([]byte(i.ImageArr), &array)
	for idx, item := range array {
		if strings.EqualFold(item, newValue) {
			// if we are adding an image that is the main actor image, put it at the begining
			if strings.EqualFold(newValue, i.ImageUrl) && idx > 0 {
				array = append(array[:idx], array[idx+1:]...)
				array = append([]string{item}, array...)
				jsonString, _ := json.Marshal(array)
				i.ImageArr = string(jsonString)
			}
			return false
		}
	}
	if newValue == i.ImageUrl {
		array = append([]string{newValue}, array...)
	} else {
		array = append(array, newValue)
	}
	jsonString, _ := json.Marshal(array)
	i.ImageArr = string(jsonString)
	return true
}
func (i *Actor) AddToActorUrlArray(newValue ActorLink) bool {
	if newValue.Url == "" {
		return false
	}
	newValue.Url = strings.TrimSuffix(newValue.Url, "/")
	var array []ActorLink
	if i.URLs == "" {
		i.URLs = "[]"
	}
	json.Unmarshal([]byte(i.URLs), &array)
	for _, item := range array {
		if strings.EqualFold(item.Url, newValue.Url) {
			return false
		}
	}
	array = append(array, newValue)
	jsonString, _ := json.Marshal(array)
	i.URLs = string(jsonString)
	return true
}
func (i *Actor) AddToTattoos(newValue string) bool {
	updated := false
	if newValue != "" {
		i.Tattoos, updated = addToStringArray(i.Tattoos, newValue)
	}
	return updated
}
func (i *Actor) AddToPiercings(newValue string) bool {
	updated := false
	if newValue != "" {
		i.Piercings, updated = addToStringArray(i.Piercings, newValue)
	}
	return updated
}
func (i *Actor) AddToAliases(newValue string) bool {
	updated := false
	if newValue != "" {
		i.Aliases, updated = addToStringArray(i.Aliases, newValue)
	}
	return updated
}
func addToStringArray(inputArray string, newValue string) (string, bool) {
	var array []string
	if inputArray == "" {
		inputArray = "[]"
	}
	json.Unmarshal([]byte(inputArray), &array)
	for _, item := range array {
		if strings.EqualFold(item, newValue) {
			return inputArray, false
		}
	}
	array = append(array, newValue)
	jsonString, _ := json.Marshal(array)
	return string(jsonString), true
}

func (a *Actor) CheckForSetImage() bool {
	// check if the field was deleted by the user,
	commonDb, _ := GetCommonDB()
	var action ActionActor
	commonDb.Where("source = 'edit_actor' and actor_id = ? and changed_column = 'image_url' and action_type = 'setimage'", a.ID).Order("ID desc").First(&action)
	return action.ID != 0
}

func (a *Actor) CheckForUserDeletes(fieldName string, newValue string) bool {
	// check if the field was deleted by the user,
	commonDb, _ := GetCommonDB()
	var action ActionActor
	commonDb.Where("source = 'edit_actor' and actor_id = ? and changed_column = ? and new_value = ?", a.ID, fieldName, newValue).Order("ID desc").First(&action)
	if action.ID != 0 && action.ActionType == "delete" {
		return true
	}
	return false
}
