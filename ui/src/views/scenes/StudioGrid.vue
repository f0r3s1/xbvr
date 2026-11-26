<template>
  <div class="studio-grid">
    <button v-for="site in sites" :key="site.scraper_id + site.name" type="button" class="studio-tile"
            :style="{ '--hue': (colors[site.name] || [hue(site.name)])[0], '--sat': (colors[site.name] || [0, 50])[1] + '%' }" @click="$router.push(siteFilterQuery($store, site.name))">
      <span class="studio-logo" :class="{ 'has-img': (site.logo_url || site.avatar_url) && !failed[site.name] }">
        <img v-if="(site.logo_url || site.avatar_url) && !failed[site.name]" :src="logo(site.logo_url || site.avatar_url)" alt="" @error="failed[site.name] = true"/>
        <span v-else>{{ initials(site.name) }}</span>
      </span>
      <span class="studio-body">
        <span class="studio-name">{{ site.name }}</span>
        <span class="studio-meta">{{ $t('{n} scene | {n} scenes', { n: site.scene_count.toLocaleString() }, site.scene_count) }}<template v-if="site.available"> · {{ $t('{n} available', { n: site.available.toLocaleString() }) }}</template></span>
        <span class="studio-foot">
          <span v-if="site.latest_release">{{ $t('Latest {date}', { date: site.latest_release }) }}</span>
          <span v-if="site.upcoming" class="studio-upcoming">{{ $t('{n} upcoming', { n: site.upcoming }) }}</span>
        </span>
      </span>
    </button>
  </div>
</template>

<script>
import { defineComponent } from 'vue'
import { loadSites, siteLogoURL, siteInitials, siteHue, siteFilterQuery, logoColor } from '../../sites'

export default defineComponent({
  name: 'StudioGrid',

  data () {
    return { sites: [], failed: {}, colors: {} }
  },

  async mounted () {
    const sites = await loadSites()
    this.sites = sites.filter(s => s.scene_count > 0).sort((a, b) => (b.latest_release || '').localeCompare(a.latest_release || ''))
    for (const site of this.sites) {
      logoColor(site.logo_url || site.avatar_url).then(c => { if (c) this.colors[site.name] = c })
    }
  },

  methods: {
    logo: siteLogoURL,
    initials: siteInitials,
    hue: siteHue,
    siteFilterQuery
  }
})
</script>

<style scoped>
.studio-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(min(100%, 280px), 1fr));
  gap: 12px;
}
.studio-tile {
  position: relative;
  display: flex;
  align-items: center;
  gap: 14px;
  min-width: 0;
  padding: 14px;
  overflow: hidden;
  border: 1px solid rgba(127, 127, 127, 0.2);
  border-radius: 12px;
  border-color: hsl(var(--hue) var(--sat) 50% / 0.22);
  background: linear-gradient(120deg, hsl(var(--hue) var(--sat) 50% / 0.28), hsl(calc(var(--hue) + 25) var(--sat) 40% / 0.1) 55%, transparent);
  color: inherit;
  font: inherit;
  text-align: left;
  cursor: pointer;
  transition: border-color 0.15s ease, transform 0.15s ease, box-shadow 0.15s ease;
}
.studio-tile:hover {
  border-color: rgba(121, 87, 213, 0.6);
  transform: translateY(-1px);
  box-shadow: 0 6px 18px rgba(0, 0, 0, 0.12);
}
.studio-tile:focus-visible {
  outline: 2px solid var(--bulma-primary, #7957d5);
  outline-offset: 2px;
}
.studio-logo {
  position: relative;
  flex: 0 0 52px;
  width: 52px;
  height: 52px;
  border-radius: 12px;
  overflow: hidden;
  display: grid;
  place-items: center;
  background: hsl(var(--hue) var(--sat) 45% / 0.22);
  color: hsl(var(--hue) var(--sat) 70%);
  font-weight: 700;
  box-shadow: inset 0 0 0 1px rgba(127, 127, 127, 0.18);
}
.studio-logo.has-img {
  background: none;
  box-shadow: none;
  border-radius: 0;
}
.studio-logo img {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: contain;
}
.studio-body {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}
.studio-name {
  font-weight: 600;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.studio-meta,
.studio-foot {
  font-size: 0.78rem;
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.studio-meta {
  opacity: 0.7;
}
.studio-foot {
  display: flex;
  align-items: center;
  gap: 8px;
  opacity: 0.85;
}
.studio-foot > span:first-child:not(.studio-upcoming) {
  opacity: 0.6;
}
.studio-upcoming {
  padding: 0 6px;
  border-radius: 999px;
  background: rgba(121, 87, 213, 0.16);
  color: #a78bfa;
}
</style>
