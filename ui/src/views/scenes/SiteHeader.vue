<template>
  <div v-if="site" class="site-header" :style="{ '--hue': color[0], '--sat': color[1] + '%' }">
    <span class="site-logo" :class="{ 'has-img': logoSrc && !logoFailed }">
      <img v-if="logoSrc && !logoFailed" :src="logoURL" alt="" @error="logoFailed = true"/>
      <span v-else>{{ initials }}</span>
    </span>
    <div class="site-text">
      <h2 class="site-name">{{ site.name }}</h2>
      <p class="site-meta">
        {{ site.scene_count.toLocaleString() }} scenes
        <template v-if="site.available"> · {{ site.available.toLocaleString() }} available</template>
        <template v-if="site.latest_release"> · latest {{ site.latest_release }}</template>
      </p>
    </div>
    <a v-if="site.homepage_url" class="site-home" :href="site.homepage_url" target="_blank" rel="noopener noreferrer" :title="$t('Open website')">
      <b-icon pack="mdi" icon="open-in-new" size="is-small"/>
    </a>
    <span v-if="site.upcoming" class="site-upcoming">{{ site.upcoming }} upcoming</span>
    <button type="button" class="site-clear" :title="$t('Show all sites')" :aria-label="$t('Show all sites')" @click="clear">
      <b-icon pack="mdi" icon="close"/>
    </button>
  </div>
</template>

<script>
import { defineComponent } from 'vue'
import { loadSites, siteLogoURL, siteInitials, siteHue, logoColor } from '../../sites'

export default defineComponent({
  name: 'SiteHeader',

  data () {
    return {
      sites: [],
      logoFailed: false,
      logoHueSat: null
    }
  },

  computed: {
    selected () {
      const sites = this.$store.state.sceneList.filters.sites || []
      return sites.length === 1 && !/^[!&]/.test(sites[0]) ? sites[0] : null
    },
    site () {
      return this.selected ? this.sites.find(s => s.name === this.selected) || null : null
    },
    logoSrc () {
      return this.site.logo_url || this.site.avatar_url
    },
    logoURL () {
      return siteLogoURL(this.logoSrc)
    },
    color () {
      return this.logoHueSat || [siteHue(this.site.name), 50]
    },
    initials () {
      return siteInitials(this.site.name)
    },
  },

  watch: {
    site: {
      immediate: true,
      async handler (site) {
        this.logoFailed = false
        this.logoHueSat = site ? await logoColor(site.logo_url || site.avatar_url) : null
      }
    }
  },

  async mounted () {
    this.sites = await loadSites()
  },

  methods: {
    clear () {
      this.$store.state.sceneList.filters.sites = []
      this.$router.push({ name: 'scenes', query: { q: this.$store.getters['sceneList/filterQueryParams'] } })
    }
  }
})
</script>

<style scoped>
.site-header {
  display: flex;
  align-items: center;
  gap: 16px;
  margin: 1.5rem 0;
  padding: 14px 16px;
  border: 1px solid rgba(127, 127, 127, 0.2);
  border-radius: 12px;
  border-color: hsl(var(--hue) var(--sat) 50% / 0.25);
  background: linear-gradient(120deg, hsl(var(--hue) var(--sat) 50% / 0.32), hsl(calc(var(--hue) + 25) var(--sat) 40% / 0.12) 55%, transparent);
  animation: fade-in 0.2s ease-out both;
}
.site-home {
  flex-shrink: 0;
  display: grid;
  place-items: center;
  width: 28px;
  height: 28px;
  border-radius: 8px;
  color: inherit;
  opacity: 0.5;
}
.site-home:hover {
  opacity: 1;
  background: rgba(127, 127, 127, 0.14);
}
.site-logo {
  flex: 0 0 56px;
  width: 56px;
  height: 56px;
  border-radius: 12px;
  overflow: hidden;
  display: grid;
  place-items: center;
  background: hsl(var(--hue) var(--sat) 45% / 0.22);
  color: hsl(var(--hue) var(--sat) 70%);
  font-weight: 700;
  box-shadow: inset 0 0 0 1px rgba(127, 127, 127, 0.18);
}
.site-logo.has-img {
  background: none;
  box-shadow: none;
  border-radius: 0;
}
.site-logo img {
  width: 100%;
  height: 100%;
  object-fit: contain;
}
.site-text {
  flex: 1;
  min-width: 0;
}
.site-name {
  margin: 0;
  font-size: 1.15rem;
  font-weight: 650;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.site-meta {
  margin: 2px 0 0;
  font-size: 0.8rem;
  opacity: 0.65;
  font-variant-numeric: tabular-nums;
}
.site-upcoming {
  flex-shrink: 0;
  padding: 2px 10px;
  border-radius: 999px;
  background: rgba(121, 87, 213, 0.16);
  color: #a78bfa;
  font-size: 0.75rem;
  white-space: nowrap;
}
.site-clear {
  flex-shrink: 0;
  display: grid;
  place-items: center;
  width: 32px;
  height: 32px;
  border: none;
  border-radius: 8px;
  background: rgba(127, 127, 127, 0.18);
  color: inherit;
  cursor: pointer;
  transition: background 0.15s ease;
}
.site-clear:hover {
  background: rgba(127, 127, 127, 0.32);
}
.site-clear .icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
  margin: 0;
}
.site-clear :deep(.mdi),
.site-clear :deep(.mdi::before) {
  font-size: 18px;
  line-height: 1;
}
</style>
