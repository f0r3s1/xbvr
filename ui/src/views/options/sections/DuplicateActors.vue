<template>
  <div class="dup-actors">
    <div class="dup-head">
      <div>
        <strong>Duplicate actors</strong>
        <span class="dup-sub" v-if="!loading">{{ pairs.length }} possible {{ pairs.length === 1 ? 'pair' : 'pairs' }}</span>
      </div>
      <div class="dup-tools" v-if="pairs.length">
        <span class="dup-tools-label">Select</span>
        <button type="button" class="dup-chip" @click="selectFrom(100)">100% <small>{{ countFrom(100) }}</small></button>
        <button type="button" class="dup-chip" @click="selectFrom(95)">≥ 95% <small>{{ countFrom(95) }}</small></button>
        <button type="button" class="dup-chip" @click="selectFrom(90)">≥ 90% <small>{{ countFrom(90) }}</small></button>
        <button type="button" class="dup-chip" @click="selected = {}">None</button>
        <b-button type="is-primary" size="is-small" :disabled="!selectedCount" :loading="merging" @click="confirmMerge">
          Merge selected ({{ selectedCount }})
        </b-button>
      </div>
    </div>

    <p class="dup-help">
      The actor marked <b>Keep</b> stays; the other one is merged into it (scenes, links, aliases and empty profile
      fields move over, its name becomes an alias). Click the arrow to swap. Scrapers keep using the kept actor.
    </p>

    <b-loading :is-full-page="false" :model-value="loading"/>
    <p v-if="!loading && !pairs.length" class="dup-empty">No duplicates found.</p>

    <div class="dup-list" v-if="pairs.length">
      <div v-for="p in pairs" :key="p.key" class="dup-row" :class="{ 'is-selected': selected[p.key] }">
        <label class="dup-check">
          <input type="checkbox" :checked="!!selected[p.key]" @change="toggle(p.key)"/>
        </label>
        <div class="dup-actor" :class="{ 'is-keep': keepOf(p) === p.a.id }">
          <img :src="img(p.a.image_url)" alt="" loading="lazy"/>
          <div class="dup-name">
            <span class="dup-n" :title="p.a.name">{{ p.a.name }}</span>
            <small>{{ p.a.count }} scenes<template v-if="p.a.stash"> · StashDB</template></small>
          </div>
          <span class="dup-role">{{ keepOf(p) === p.a.id ? 'Keep' : 'Merge' }}</span>
        </div>
        <button type="button" class="dup-swap" title="Swap which actor is kept" @click="swap(p)">
          <b-icon pack="mdi" :icon="keepOf(p) === p.a.id ? 'arrow-left' : 'arrow-right'" size="is-small"/>
        </button>
        <div class="dup-actor" :class="{ 'is-keep': keepOf(p) === p.b.id }">
          <img :src="img(p.b.image_url)" alt="" loading="lazy"/>
          <div class="dup-name">
            <span class="dup-n" :title="p.b.name">{{ p.b.name }}</span>
            <small>{{ p.b.count }} scenes<template v-if="p.b.stash"> · StashDB</template></small>
          </div>
          <span class="dup-role">{{ keepOf(p) === p.b.id ? 'Keep' : 'Merge' }}</span>
        </div>
        <div class="dup-score">
          <span class="dup-pct" :class="scoreClass(p.score)">{{ p.score }}%</span>
          <small>{{ p.reason }}</small>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
// Review list for likely duplicate actors (Library Health): quick bulk selection
// by confidence, choose which record to keep, merge in one go.
import { defineComponent } from 'vue'
import ky from 'ky'

export default defineComponent({
  name: 'DuplicateActors',
  emits: ['merged'],

  data () {
    return {
      pairs: [],
      selected: {}, // key -> true
      keepOverride: {}, // key -> actor id
      loading: false,
      merging: false
    }
  },

  computed: {
    selectedCount () {
      return Object.values(this.selected).filter(Boolean).length
    }
  },

  mounted () {
    this.load()
  },

  methods: {
    async load () {
      this.loading = true
      try {
        const data = await ky.get('/api/actor/duplicates', { timeout: 60000 }).json()
        this.pairs = (data || []).map(p => ({ ...p, key: p.a.id + '-' + p.b.id }))
        this.selected = {}
        this.keepOverride = {}
      } catch {
        this.$buefy.toast.open({ message: 'Could not load duplicate actors', type: 'is-danger' })
      }
      this.loading = false
    },
    keepOf (p) {
      return this.keepOverride[p.key] || p.keep
    },
    swap (p) {
      this.keepOverride = { ...this.keepOverride, [p.key]: this.keepOf(p) === p.a.id ? p.b.id : p.a.id }
    },
    toggle (key) {
      this.selected = { ...this.selected, [key]: !this.selected[key] }
    },
    countFrom (min) {
      return this.pairs.filter(p => p.score >= min).length
    },
    selectFrom (min) {
      const sel = {}
      for (const p of this.pairs) if (p.score >= min) sel[p.key] = true
      this.selected = sel
    },
    scoreClass (s) {
      return s >= 100 ? 'is-certain' : s >= 90 ? 'is-high' : 'is-review'
    },
    img (u) {
      if (!u) return '/ui/images/actor_placeholder.svg'
      return u.startsWith('http') ? '/img/200x/' + u.replace('://', ':/') : u
    },
    confirmMerge () {
      const n = this.selectedCount
      this.$buefy.dialog.confirm({
        title: 'Merge actors',
        message: `Merge <b>${n}</b> ${n === 1 ? 'pair' : 'pairs'}? Each merged actor is folded into the one marked Keep and then deleted. This cannot be undone.`,
        confirmText: `Merge ${n}`,
        type: 'is-warning',
        hasIcon: true,
        onConfirm: () => this.merge()
      })
    },
    async merge () {
      const pairs = this.pairs.filter(p => this.selected[p.key]).map(p => {
        const keep = this.keepOf(p)
        return { keep, merge: keep === p.a.id ? p.b.id : p.a.id }
      })
      this.merging = true
      try {
        const res = await ky.post('/api/actor/merge', { json: { pairs }, timeout: 300000 }).json()
        const errs = (res.errors || []).length
        this.$buefy.toast.open({
          message: `Merged ${res.merged} ${res.merged === 1 ? 'actor' : 'actors'}` + (errs ? `, ${errs} skipped` : ''),
          type: errs ? 'is-warning' : 'is-success'
        })
        this.$emit('merged')
        await this.load()
      } catch {
        this.$buefy.toast.open({ message: 'Merge failed', type: 'is-danger' })
      }
      this.merging = false
    }
  }
})
</script>

<style scoped>
.dup-actors {
  position: relative;
  min-height: 80px;
  margin-top: 1.5rem;
  padding: 1rem;
  border: 1px solid rgba(127, 127, 127, 0.25);
  border-radius: 8px;
}
.dup-head {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}
.dup-sub {
  margin-left: 8px;
  font-size: 0.8rem;
  opacity: 0.6;
}
.dup-tools {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
}
.dup-tools-label {
  font-size: 0.75rem;
  opacity: 0.6;
}
.dup-chip {
  height: 28px;
  padding: 0 10px;
  border: 1px solid rgba(127, 127, 127, 0.3);
  border-radius: 999px;
  background: transparent;
  color: inherit;
  font: inherit;
  font-size: 0.78rem;
  cursor: pointer;
}
.dup-chip:hover {
  background: rgba(127, 127, 127, 0.12);
}
.dup-chip small {
  opacity: 0.6;
  margin-left: 2px;
}
.dup-help {
  margin: 0.5rem 0 0.75rem;
  font-size: 0.78rem;
  opacity: 0.7;
}
.dup-empty {
  opacity: 0.6;
}
.dup-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
  max-height: 60vh;
  overflow-y: auto;
}
.dup-row {
  display: grid;
  grid-template-columns: 24px minmax(0, 1fr) 28px minmax(0, 1fr) 150px;
  align-items: center;
  gap: 8px;
  padding: 6px 8px;
  border-radius: 8px;
  background: rgba(127, 127, 127, 0.05);
}
.dup-row.is-selected {
  background: rgba(121, 87, 213, 0.1);
}
.dup-check input {
  accent-color: #7957d5;
  cursor: pointer;
}
.dup-actor {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  padding: 4px 6px;
  border: 1px solid transparent;
  border-radius: 6px;
}
.dup-actor.is-keep {
  border-color: rgba(72, 199, 142, 0.5);
  background: rgba(72, 199, 142, 0.08);
}
.dup-actor img {
  flex: 0 0 auto;
  width: 36px;
  height: 36px;
  border-radius: 6px;
  object-fit: cover;
  background: rgba(127, 127, 127, 0.15);
}
.dup-name {
  display: flex;
  flex-direction: column;
  min-width: 0;
  line-height: 1.25;
}
.dup-n {
  font-weight: 500;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.dup-name small {
  font-size: 0.72rem;
  opacity: 0.6;
}
.dup-role {
  margin-left: auto;
  flex: 0 0 auto;
  font-size: 0.68rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  opacity: 0.55;
}
.dup-actor.is-keep .dup-role {
  color: #3fbf86;
  opacity: 1;
}
.dup-swap {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border: 1px solid rgba(127, 127, 127, 0.3);
  border-radius: 6px;
  background: transparent;
  color: inherit;
  cursor: pointer;
}
.dup-swap:hover {
  background: rgba(127, 127, 127, 0.12);
}
.dup-score {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  line-height: 1.25;
  text-align: right;
}
.dup-score small {
  font-size: 0.7rem;
  opacity: 0.6;
}
.dup-pct {
  font-weight: 700;
  font-variant-numeric: tabular-nums;
}
.dup-pct.is-certain { color: #3fbf86; }
.dup-pct.is-high { color: #3e8ed0; }
.dup-pct.is-review { color: #e0a03c; }
@media (max-width: 900px) {
  .dup-row {
    grid-template-columns: 24px minmax(0, 1fr) 28px minmax(0, 1fr);
  }
  .dup-score {
    grid-column: 2 / -1;
    flex-direction: row;
    gap: 6px;
    justify-content: flex-end;
  }
}
</style>
