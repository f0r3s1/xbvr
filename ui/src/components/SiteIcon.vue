<template>
  <img v-if="src" :key="src" :src="src" class="site-icon" :style="box" alt="" loading="lazy" decoding="async" @error="step++"/>
  <span v-else class="site-icon site-initial" :style="box" aria-hidden="true">{{ initial }}</span>
</template>

<script>
import { imageURL } from '@/imageURL'

// studio icon: avatar then logo, each through the image cache and then directly; a letter badge once all fail
export default {
  name: 'SiteIcon',
  props: {
    site: { type: Object, default: null },
    name: { type: String, default: '' },
    size: { type: Number, default: 16 }
  },
  data () {
    return { step: 0 }
  },
  computed: {
    candidates () {
      const urls = this.site ? [this.site.avatar_url, this.site.logo_url].filter(Boolean) : []
      return [...new Set(urls.flatMap(u => [imageURL(u, `${this.size * 4}x`), u]))]
    },
    src () {
      return this.candidates[this.step] || null
    },
    initial () {
      return (this.name || this.site?.name || '?').trim().charAt(0).toUpperCase()
    },
    box () {
      return { width: `${this.size}px`, height: `${this.size}px`, fontSize: `${Math.round(this.size * 0.62)}px` }
    }
  },
  watch: {
    candidates () {
      this.step = 0
    }
  }
}
</script>

<style scoped>
.site-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  border-radius: 3px;
  object-fit: cover;
}
.site-initial {
  font-weight: 700;
  line-height: 1;
  color: #fff;
  background: #7957d5;
}
</style>
