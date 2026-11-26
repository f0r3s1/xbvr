package models

import (
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/jinzhu/gorm"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// ActorMerge remembers an actor name that was merged into another actor, so a
// later scrape that still reports the old name links the kept actor instead of
// creating the duplicate again.
type ActorMerge struct {
	ID        uint      `gorm:"primary_key" json:"id" xbvrbackup:"-"`
	CreatedAt time.Time `json:"-" xbvrbackup:"-"`
	Name      string    `gorm:"unique_index" json:"name" xbvrbackup:"name"`
	ActorID   uint      `gorm:"index" json:"actor_id" xbvrbackup:"-"`
}

// FindOrCreateActorByName is what scrapers use to turn a cast name into an actor:
// a merged-away name resolves to the actor it was merged into.
func FindOrCreateActorByName(db *gorm.DB, name string, out *Actor) {
	var m ActorMerge
	if db.Where("name = ?", name).First(&m).Error == nil {
		if db.First(out, m.ActorID).Error == nil {
			return
		}
	}
	db.Where(&Actor{Name: name}).FirstOrCreate(out)
}

type ActorBrief struct {
	ID         uint      `json:"id"`
	Name       string    `json:"name"`
	ImageUrl   string    `json:"image_url"`
	Count      int       `json:"count"`
	AvailCount int       `json:"avail_count"`
	Stash      bool      `json:"stash"`
	BirthDate  time.Time `json:"birth_date"`
	filled     int
	aliases    []string
}

type DuplicateActorPair struct {
	A      ActorBrief `json:"a"`
	B      ActorBrief `json:"b"`
	Score  int        `json:"score"`
	Reason string     `json:"reason"`
	Keep   uint       `json:"keep"` // suggested actor to keep
}

var stripMarks = transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)

// actorNameTokens: lower case, accents removed, split on anything that is not a
// letter or digit ("Avá  Brooks-Lee." -> ava, brooks, lee).
func actorNameTokens(s string) []string {
	r, _, err := transform.String(stripMarks, s)
	if err != nil {
		r = s
	}
	return strings.FieldsFunc(strings.ToLower(r), func(c rune) bool { return !unicode.IsLetter(c) && !unicode.IsDigit(c) })
}

// jaroWinkler similarity of two strings, 0..1.
func jaroWinkler(a, b string) float64 {
	ar, br := []rune(a), []rune(b)
	la, lb := len(ar), len(br)
	if la == 0 || lb == 0 {
		return 0
	}
	window := la
	if lb > window {
		window = lb
	}
	window = window/2 - 1
	if window < 0 {
		window = 0
	}
	ma := make([]bool, la)
	mb := make([]bool, lb)
	matches := 0
	for i := 0; i < la; i++ {
		lo, hi := i-window, i+window+1
		if lo < 0 {
			lo = 0
		}
		if hi > lb {
			hi = lb
		}
		for j := lo; j < hi; j++ {
			if !mb[j] && ar[i] == br[j] {
				ma[i], mb[j] = true, true
				matches++
				break
			}
		}
	}
	if matches == 0 {
		return 0
	}
	trans, k := 0, 0
	for i := 0; i < la; i++ {
		if !ma[i] {
			continue
		}
		for !mb[k] {
			k++
		}
		if ar[i] != br[k] {
			trans++
		}
		k++
	}
	m := float64(matches)
	jaro := (m/float64(la) + m/float64(lb) + (m-float64(trans)/2)/m) / 3
	prefix := 0
	for i := 0; i < la && i < lb && i < 4 && ar[i] == br[i]; i++ {
		prefix++
	}
	return jaro + float64(prefix)*0.1*(1-jaro)
}

func validBirth(t time.Time) bool { return !t.IsZero() && t.Year() > 1900 }

// FindDuplicateActors lists actor pairs that are probably the same person, most
// certain first. Aka groups ("aka:...") are never proposed.
//
//	100 same StashDB performer, or same name ignoring case/accents/punctuation/spacing
//	 97 same letters without spaces ("Mary Jane" / "MaryJane")
//	 95 same words in another order
//	 92 one is listed as an alias of the other
//	70-89 very similar spelling (same word count and initials); needs a look
//
// Two actors with different known birth dates are not proposed, except for a
// shared StashDB performer.
//
// yield (may be nil) is called regularly so a background caller can throttle the
// CPU use of the name comparisons.
func FindDuplicateActors(yield func()) []DuplicateActorPair {
	if yield == nil {
		yield = func() {}
	}
	db, _ := GetCommonDB()

	var rows []Actor
	db.Model(&Actor{}).Select("id, name, image_url, count, avail_count, birth_date, aliases, nationality, height, biography, urls").
		Where("name NOT LIKE 'aka:%'").Find(&rows)

	var links []struct {
		InternalDbId uint
		ExternalId   string
	}
	db.Model(&ExternalReferenceLink{}).Select("internal_db_id, external_id").
		Where("internal_table = 'actors' AND external_source = 'stashdb performer'").Scan(&links)
	stashOf := map[uint][]string{}
	for _, l := range links {
		stashOf[l.InternalDbId] = append(stashOf[l.InternalDbId], l.ExternalId)
	}

	briefs := make(map[uint]*ActorBrief, len(rows))
	tokens := make(map[uint][]string, len(rows))
	for i := range rows {
		a := &rows[i]
		b := &ActorBrief{ID: a.ID, Name: a.Name, ImageUrl: a.ImageUrl, Count: a.Count, AvailCount: a.AvailCount,
			Stash: len(stashOf[a.ID]) > 0, BirthDate: a.BirthDate}
		for _, v := range []bool{validBirth(a.BirthDate), a.Nationality != "", a.Height > 0, a.Biography != "", a.ImageUrl != "", a.URLs != "" && a.URLs != "[]"} {
			if v {
				b.filled++
			}
		}
		var al []string
		if json.Unmarshal([]byte(a.Aliases), &al) == nil {
			b.aliases = al
		}
		briefs[a.ID] = b
		tokens[a.ID] = actorNameTokens(a.Name)
	}

	type key struct{ a, b uint }
	best := map[key]DuplicateActorPair{}
	add := func(x, y uint, score int, reason string) {
		if x == y {
			return
		}
		if x > y {
			x, y = y, x
		}
		a, b := briefs[x], briefs[y]
		if a == nil || b == nil {
			return
		}
		if score < 100 || reason != "Same StashDB performer" {
			if validBirth(a.BirthDate) && validBirth(b.BirthDate) && !sameDay(a.BirthDate, b.BirthDate) {
				return // different people, same-ish name
			}
			if score < 100 && validBirth(a.BirthDate) && sameDay(a.BirthDate, b.BirthDate) {
				score += 3
				if score > 99 {
					score = 99
				}
			}
		}
		k := key{x, y}
		if cur, ok := best[k]; ok && cur.Score >= score {
			return
		}
		best[k] = DuplicateActorPair{A: *a, B: *b, Score: score, Reason: reason}
	}
	pairsIn := func(groups map[string][]uint, score int, reason string) {
		for _, ids := range groups {
			for i := 0; i < len(ids); i++ {
				for j := i + 1; j < len(ids); j++ {
					add(ids[i], ids[j], score, reason)
				}
			}
		}
	}

	yield()
	byStash := map[string][]uint{}
	for id, ext := range stashOf {
		if briefs[id] == nil {
			continue
		}
		for _, e := range ext {
			byStash[e] = append(byStash[e], id)
		}
	}
	pairsIn(byStash, 100, "Same StashDB performer")

	byName := map[string][]uint{}
	byJoined := map[string][]uint{}
	bySorted := map[string][]uint{}
	for id, t := range tokens {
		if len(t) == 0 {
			continue
		}
		byName[strings.Join(t, " ")] = append(byName[strings.Join(t, " ")], id)
		joined := strings.Join(t, "")
		if len(joined) >= 6 {
			byJoined[joined] = append(byJoined[joined], id)
		}
		if len(t) > 1 {
			s := append([]string{}, t...)
			sort.Strings(s)
			bySorted[strings.Join(s, " ")] = append(bySorted[strings.Join(s, " ")], id)
		}
	}
	pairsIn(byName, 100, "Same name")
	pairsIn(byJoined, 97, "Same name without spaces")
	pairsIn(bySorted, 95, "Same name, other word order")

	for id, b := range briefs {
		for _, al := range b.aliases {
			for _, other := range byName[strings.Join(actorNameTokens(al), " ")] {
				add(id, other, 92, "Listed as an alias")
			}
		}
	}

	// similar spelling, compared only within the same word count and initials
	blocks := map[string][]uint{}
	for id, t := range tokens {
		if len(t) < 2 {
			continue
		}
		var initials []rune
		for _, w := range t {
			initials = append(initials, []rune(w)[0])
		}
		k := string(initials)
		blocks[k] = append(blocks[k], id)
	}
	compared := 0
	for _, ids := range blocks {
		if len(ids) > 3000 {
			continue // keep it cheap on huge buckets
		}
		for i := 0; i < len(ids); i++ {
			ji := strings.Join(tokens[ids[i]], " ")
			for j := i + 1; j < len(ids); j++ {
				if compared++; compared%512 == 0 {
					yield()
				}
				jj := strings.Join(tokens[ids[j]], " ")
				if ji == jj || len(ji) < 7 || len(jj) < 7 {
					continue
				}
				if s := jaroWinkler(ji, jj); s >= 0.93 {
					add(ids[i], ids[j], 70+int((s-0.93)/0.07*19), "Similar spelling")
				}
			}
		}
	}

	out := make([]DuplicateActorPair, 0, len(best))
	for _, p := range best {
		p.Keep = preferredActor(p.A, p.B)
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		return strings.ToLower(out[i].A.Name) < strings.ToLower(out[j].A.Name)
	})
	return out
}

func sameDay(a, b time.Time) bool {
	return a.Year() == b.Year() && a.YearDay() == b.YearDay()
}

// preferredActor: the record to keep, StashDB link first, then more scenes, then
// the more complete profile, then the older record.
func preferredActor(a, b ActorBrief) uint {
	switch {
	case a.Stash != b.Stash:
		if a.Stash {
			return a.ID
		}
		return b.ID
	case a.Count != b.Count:
		if a.Count > b.Count {
			return a.ID
		}
		return b.ID
	case a.filled != b.filled:
		if a.filled > b.filled {
			return a.ID
		}
		return b.ID
	}
	if a.ID < b.ID {
		return a.ID
	}
	return b.ID
}

// MergeActors folds actor dropID into keepID: scenes, aka groups, external links
// and edit history move over, empty profile fields are filled from the merged
// actor, its name becomes an alias and is remembered for scrapers, and the
// merged actor is deleted. Returns the ids of the scenes whose cast changed.
func MergeActors(keepID, dropID uint) ([]uint, error) {
	if keepID == 0 || dropID == 0 || keepID == dropID {
		return nil, errors.New("pick two different actors")
	}
	db, _ := GetCommonDB()
	var keep, drop Actor
	if db.First(&keep, keepID).Error != nil || db.First(&drop, dropID).Error != nil {
		return nil, errors.New("actor not found")
	}
	if strings.HasPrefix(keep.Name, "aka:") || strings.HasPrefix(drop.Name, "aka:") {
		return nil, errors.New("aka groups cannot be merged")
	}

	tx := db.Begin()
	fail := func(err error) ([]uint, error) {
		tx.Rollback()
		return nil, err
	}

	var dropScenes, keepScenes []uint
	if err := tx.Table("scene_cast").Where("actor_id = ?", dropID).Pluck("scene_id", &dropScenes).Error; err != nil {
		return fail(err)
	}
	if err := tx.Table("scene_cast").Where("actor_id = ?", keepID).Pluck("scene_id", &keepScenes).Error; err != nil {
		return fail(err)
	}
	have := map[uint]bool{}
	for _, s := range keepScenes {
		have[s] = true
	}
	for _, s := range dropScenes {
		if !have[s] {
			if err := tx.Exec("INSERT INTO scene_cast (scene_id, actor_id) VALUES (?, ?)", s, keepID).Error; err != nil {
				return fail(err)
			}
		}
	}
	if err := tx.Exec("DELETE FROM scene_cast WHERE actor_id = ?", dropID).Error; err != nil {
		return fail(err)
	}

	var dropAkas, keepAkas []uint
	if err := tx.Table("actor_akas").Where("actor_id = ?", dropID).Pluck("aka_id", &dropAkas).Error; err != nil {
		return fail(err)
	}
	if err := tx.Table("actor_akas").Where("actor_id = ?", keepID).Pluck("aka_id", &keepAkas).Error; err != nil {
		return fail(err)
	}
	inAka := map[uint]bool{}
	for _, a := range keepAkas {
		inAka[a] = true
	}
	for _, a := range dropAkas {
		if !inAka[a] {
			if err := tx.Exec("INSERT INTO actor_akas (aka_id, actor_id) VALUES (?, ?)", a, keepID).Error; err != nil {
				return fail(err)
			}
		}
	}
	if err := tx.Exec("DELETE FROM actor_akas WHERE actor_id = ?", dropID).Error; err != nil {
		return fail(err)
	}

	// external links move unless the kept actor already has them
	var dropLinks []ExternalReferenceLink
	if err := tx.Where("internal_table = 'actors' AND internal_db_id = ?", dropID).Find(&dropLinks).Error; err != nil {
		return fail(err)
	}
	for _, l := range dropLinks {
		var n int
		if err := tx.Model(&ExternalReferenceLink{}).Where("internal_table = 'actors' AND internal_db_id = ? AND external_reference_id = ?", keepID, l.ExternalReferenceID).Count(&n).Error; err != nil {
			return fail(err)
		}
		var err error
		if n > 0 {
			err = tx.Delete(&l).Error
		} else {
			err = tx.Model(&ExternalReferenceLink{}).Where("id = ?", l.ID).Updates(map[string]interface{}{"internal_db_id": keepID, "internal_name_id": keep.Name}).Error
		}
		if err != nil {
			return fail(err)
		}
	}

	if err := tx.Exec("UPDATE action_actors SET actor_id = ? WHERE actor_id = ?", keepID, dropID).Error; err != nil {
		return fail(err)
	}

	mergeActorFields(&keep, &drop)
	if err := tx.Save(&keep).Error; err != nil {
		return fail(err)
	}

	// remember the merged-away name (and anything previously merged into it)
	if err := tx.Exec("UPDATE actor_merges SET actor_id = ? WHERE actor_id = ?", keepID, dropID).Error; err != nil {
		return fail(err)
	}
	var existing ActorMerge
	var err error
	if tx.Where("name = ?", drop.Name).First(&existing).Error == nil {
		err = tx.Model(&existing).Update("actor_id", keepID).Error
	} else {
		err = tx.Create(&ActorMerge{Name: drop.Name, ActorID: keepID}).Error
	}
	if err != nil {
		return fail(err)
	}

	if err := tx.Delete(&drop).Error; err != nil {
		return fail(err)
	}
	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	for _, akaID := range dropAkas {
		var aka Aka
		if aka.GetIfExistByPK(akaID) == nil {
			aka.UpdateAkaSceneCastRecords()
			aka.RefreshAkaActorNames()
		}
	}
	return dropScenes, nil
}

func mergeActorFields(keep, drop *Actor) {
	if !validBirth(keep.BirthDate) && validBirth(drop.BirthDate) {
		keep.BirthDate = drop.BirthDate
	}
	str := func(k *string, d string) {
		if strings.TrimSpace(*k) == "" {
			*k = d
		}
	}
	str(&keep.ImageUrl, drop.ImageUrl)
	str(&keep.Nationality, drop.Nationality)
	str(&keep.Ethnicity, drop.Ethnicity)
	str(&keep.EyeColor, drop.EyeColor)
	str(&keep.HairColor, drop.HairColor)
	str(&keep.CupSize, drop.CupSize)
	str(&keep.BreastType, drop.BreastType)
	str(&keep.Gender, drop.Gender)
	str(&keep.Biography, drop.Biography)
	num := func(k *int, d int) {
		if *k == 0 {
			*k = d
		}
	}
	num(&keep.Height, drop.Height)
	num(&keep.Weight, drop.Weight)
	num(&keep.BandSize, drop.BandSize)
	num(&keep.WaistSize, drop.WaistSize)
	num(&keep.HipSize, drop.HipSize)
	num(&keep.StartYear, drop.StartYear)
	num(&keep.EndYear, drop.EndYear)
	if keep.StarRating == 0 {
		keep.StarRating = drop.StarRating
	}
	keep.Favourite = keep.Favourite || drop.Favourite
	keep.Watchlist = keep.Watchlist || drop.Watchlist

	var imgs []string
	if json.Unmarshal([]byte(drop.ImageArr), &imgs) == nil {
		for _, u := range imgs {
			keep.AddToImageArray(u)
		}
	}
	keep.AddToImageArray(drop.ImageUrl)
	var urls []ActorLink
	if json.Unmarshal([]byte(drop.URLs), &urls) == nil {
		for _, u := range urls {
			keep.AddToActorUrlArray(u)
		}
	}
	keep.Aliases = unionJSONList(keep.Aliases, drop.Aliases, drop.Name, keep.Name)
	keep.Tattoos = unionJSONList(keep.Tattoos, drop.Tattoos, "", "")
	keep.Piercings = unionJSONList(keep.Piercings, drop.Piercings, "", "")
}

// unionJSONList merges two JSON string arrays (case-insensitive), optionally adding
// one more value, never adding `exclude`.
func unionJSONList(a, b, extra, exclude string) string {
	var la, lb []string
	json.Unmarshal([]byte(a), &la)
	json.Unmarshal([]byte(b), &lb)
	seen := map[string]bool{strings.ToLower(strings.TrimSpace(exclude)): exclude != ""}
	var out []string
	for _, v := range append(append(la, lb...), extra) {
		v = strings.TrimSpace(v)
		k := strings.ToLower(v)
		if v == "" || seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, v)
	}
	if len(out) == 0 {
		if a == "" {
			return ""
		}
		return "[]"
	}
	j, _ := json.Marshal(out)
	return string(j)
}
