<template>
  <div class="modal is-active">
    <GlobalEvents
      :filter="e => !['INPUT', 'TEXTAREA'].includes(e.target.tagName)"
      @keyup.esc="handleEscape"
      @keydown.left="handleLeftArrow"
      @keydown.right="handleRightArrow"
      @keydown.o="prevActor"
      @keydown.p="nextActor"
      @keydown.f="$store.commit('actorList/toggleActorList', {actor_id: actor.id, list: 'favourite'})"
      @keydown.exact.w="$store.commit('actorList/toggleActorList', {actor_id: actor.id, list: 'watchlist'})"
      @keydown.e="$store.commit('overlay/editActorDetails', {actor: actor})"
      @keydown.s="$store.commit('overlay/showSearchStashdbActors', {actor: item})"
      @keydown.g="toggleGallery"
      @keydown="onZeroKey"
    />

    <div class="modal-background" @click="close"></div>

    <div class="modal-card">
      <section class="modal-card-body">
        <div class="columns">

          <div class="column is-half">
            <b-tabs v-model="activeMedia" position="is-centered" :animated="false">
              <b-tab-item :label="$t('Gallery')">
                <div class="carousel-wrapper" :style="{ '--photo-ratio': activePhotoRatio }">
                  <b-carousel v-model="carouselSlide" @change="onCarouselChange" :autoplay="false" :indicator-inside="false">
                    <b-carousel-item v-for="(carousel, i) in images" :key="i">
                      <div class="image is-1by1 is-full carousel-image-container">
                        <img 
                          ref="carouselImages"
                          :src="getCarouselImageURL(carousel, i)" 
                          @load="onCarouselImageLoad(i)"
                          @error="onCarouselImageError(i)"
                          class="carousel-image"
                          :class="{ 'is-loaded': carouselImagesLoaded[i] }"
                          @click="openFullscreenGallery(i)"
                          style="cursor: pointer;"
                        />
                        <span class="carousel-type-tag" v-if="carouselImageInfo[i]">{{ carouselImageInfo[i] }}</span>
                      </div>
                    </b-carousel-item>
                    <template #indicators="props">
                        <span class="al image" style="width:max-content;">
                          <vue-load-image>
                            <template #image><img :src="getIndicatorURL(props.i)" style="height:85px;"/></template>
                            <template #preloader><img :src="getImageURL('https://i.stack.imgur.com/kOnzy.gif')" style="height:25px;"/></template>
                            <template #error><img src="/ui/images/blank_female_profile.png" style="height:85px;"/></template>
                          </vue-load-image>
                        </span>
                    </template>
                  </b-carousel>
                  <button class="fullscreen-btn" @click="openFullscreenGallery(carouselSlide)" title="Fullscreen (G)">
                    <b-icon pack="mdi" icon="fullscreen" size="is-small"></b-icon>
                  </button>
                </div>
                <div class="flexcentre">
                <b-button class="button is-primary is-small" style="display: flex; justify-content: center;" v-on:click="setActorImage()">{{$t('Set Main Image')}}</b-button>
                <b-button v-if="images.length != 0" class="button is-primary is-small" style="display: flex; justify-content: center;margin-left: 1em;" v-on:click="deleteActorImage()">{{$t('Delete Image')}}</b-button>
                </div>
              </b-tab-item>
            </b-tabs>
          </div>

          <div class="column is-half">
            <div class="block-info block">
              <div class="content">
                <h3>
                  <span>
                    {{ actor.name }}
                    <b-tooltip position="is-right" :label="$t('Delete Aka Group')" multilined :delay="200" v-if="actor.name.startsWith('aka:')">
                      <button class="button is-small is-outlined" @click="deleteAkaGroup" >
                        <b-icon pack="mdi" icon="delete-outline"></b-icon>
                      </button>
                    </b-tooltip>
                    <b-tooltip v-if="enableNewAkaGroup()" position="is-right" :label="$t('Create a new Aka Group')" multilined :delay="200">
                      <button class="button is-small is-outlined" @click="createAkaGroup">
                        <b-icon pack="mdi" icon="account-multiple-plus-outline"></b-icon>
                      </button>
                    </b-tooltip>
                  </span>
                  <small v-if="actor.birth_date != '0001-01-01T00:00:00Z'" class="is-pulled-right">
                    {{ actor.birth_date.slice(0, 10) }}
                  </small>
                </h3>
                <div class="columns">
                  <div class="column pb-0">
                  </div>
                </div>
                <div class="columns is-vcentered">
                  <div class="column pt-0">
                    <b-field>
                      <strong style="width: 8em;">{{ $t('Your Rating') }}</strong>
                      <star-rating :key="actor.id" v-model="actor.star_rating" :rating="actor.star_rating" @rating-selected="setRating"
                                   :increment="0.5" :star-size="20" :show-rating="true" />
                      <b-tooltip :label="$t('Reset Rating')" position="is-right" :delay="250">
                        <b-icon pack="mdi" icon="autorenew" size="is-small" @click="setRating(0)" style="padding-left: 1em;padding-top: .5em;"/>
                      </b-tooltip>
                    </b-field>
                    <b-field>
                      <strong style="width: 8em;">{{ $t('Scene Average') }}</strong>
                    <star-rating :key="actor.id" :rating="Math.round(actor.scene_rating_average * 4) / 4" read-only :increment="0.25" :star-size="20" :show-rating="true" active-color="#7957d5"/>
                    </b-field>

                  </div>
                  <div class="column pt-0">
                    <div class="is-pulled-right">
                      <actor-favourite-button :actor="actor"/>&nbsp;
                      <actor-watchlist-button :actor="actor"/>&nbsp;
                      <actor-edit-button :actor="actor"/>&nbsp;
                      <link-stashdb-button :item="actor" objectType="actor" />
                    </div>
                  </div>
                </div>
                <div class="actor-stats">
                  <div class="actor-stat" v-for="s in actorStats" :key="s.label">
                    <span class="actor-stat-value">{{ s.value }}</span>
                    <span class="actor-stat-label">{{ s.label }}</span>
                  </div>
                </div>
              </div>
            </div>
                        
            <div class="block-opts block">
              <b-tabs v-model="activeTab" :animated="false">
                <b-tab-item :label="$t('Details')">
                  <dl class="actor-facts">
                    <div class="fact" v-if="actor.start_year + actor.end_year != 0">
                      <dt>{{ $t('Active') }}</dt><dd>{{ getYearsActive() }}</dd>
                    </div>
                    <div class="fact" v-if="actor.nationality">
                      <dt>{{ $t('Nationality') }}</dt>
                      <dd class="fact-flag">
                        <vue-load-image>
                          <template #image><img :src="getImageURL(getCountryFlag(actor.nationality))" class="fact-flag-img" alt=""/></template>
                        </vue-load-image>
                        <span>{{ getCountryName(actor.nationality) }}</span>
                      </dd>
                    </div>
                    <div class="fact" v-if="actor.ethnicity">
                      <dt>{{ $t('Ethnicity') }}</dt><dd>{{ actor.ethnicity }}</dd>
                    </div>
                    <div class="fact" v-if="actor.hair_color">
                      <dt>{{ $t('Hair Color') }}</dt><dd>{{ actor.hair_color }}</dd>
                    </div>
                    <div class="fact" v-if="actor.eye_color">
                      <dt>{{ $t('Eye Color') }}</dt><dd>{{ actor.eye_color }}</dd>
                    </div>
                    <div class="fact" v-if="actor.height">
                      <dt>{{ $t('Height') }}</dt><dd>{{ getHeight(actor.height) }}</dd>
                    </div>
                    <div class="fact" v-if="actor.weight">
                      <dt>{{ $t('Weight') }}</dt><dd>{{ getWeight(actor.weight) }}</dd>
                    </div>
                    <div class="fact" v-if="measurements() != ''">
                      <dt>{{ $t('Measurements') }}</dt><dd>{{ measurements() }}</dd>
                    </div>
                    <div class="fact" v-if="actor.breast_type">
                      <dt>{{ $t('Breast Type') }}</dt><dd>{{ actor.breast_type }}</dd>
                    </div>
                    <div class="fact is-wide" v-if="actor.aliases && actor.aliases != '[]'">
                      <dt>{{ $t('Aliases') }}</dt><dd>{{ joinArray(actor.aliases) }}</dd>
                    </div>
                    <div class="fact is-wide" v-if="actor.tattoos && actor.tattoos != '[]'">
                      <dt>{{ $t('Tattoos') }}</dt><dd>{{ joinArray(actor.tattoos) }}</dd>
                    </div>
                    <div class="fact is-wide" v-if="actor.piercings && actor.piercings != '[]'">
                      <dt>{{ $t('Piercings') }}</dt><dd>{{ joinArray(actor.piercings) }}</dd>
                    </div>
                  </dl>
                  <div class="actor-bio" v-if="actor.biography">
                    <p :class="{ 'is-clamped': !bioExpanded }" ref="bio">{{ actor.biography }}</p>
                    <button v-if="bioLong" type="button" class="bio-toggle" @click="bioExpanded = !bioExpanded">
                      {{ bioExpanded ? 'Show less' : 'Show more' }}
                    </button>
                  </div>
                  <p v-if="!hasDetails" class="actor-empty-details">
                    No profile details yet. Use Edit, or link the actor on StashDB, to fill them in.
                  </p>
                </b-tab-item>
                <b-tab-item>
                  <template #header>                    
                    <span class="tab-label">Scenes ({{ actor.scenes.length }})<a v-if="showOpenInNewWindow" :href='getCastScenesUrl([actor.name])' target="_blank" class="tab-ext" title="Open scenes in a new window" @click.stop><b-icon pack="mdi" icon="open-in-new" custom-size="mdi-14px"/></a></span>
                  </template>
                  <div v-show="activeTab == 1" :class="['columns', 'is-multiline', actor.scenes.length > 6 ? 'scroll' : '']">
                    <!-- list entries carry scene summaries; cards wait for the full scenes -->
                    <div :class="['column', 'is-multiline', 'is-one-third']"
                      v-for="(scene, idx) in (actor.slim ? [] : actor.scenes)" :key="idx" class="image-wrapper">
                      <SceneCard :item="scene" :reRead=true />
                    </div>
                  </div>
                </b-tab-item>
                <b-tab-item :label="$t('Akas')" :visible="akas.aka_groups != null || akas.actors != null || akas.possible_akas != null">
                  <div v-show="activeTab == 2">
                    <b-field :label="$t('Aka Groups')" v-if="akas.aka_groups != null &&  akas.aka_groups.length!=0">
                      <div  class="columns is-multiline">
                        <div :class="['column', 'is-multiline', 'is-one-third']"
                          v-for="(akaactor, idx) in akas.aka_groups" :key="idx" class="image-wrapper">
                          <ActorCard :actor="akaactor"/>
                        </div>
                      </div>
                    </b-field>
                    <b-field :label="$t('Other Actors In Groups')" v-if="akas.actors != null &&  akas.actors.length!=0">
                      <div  class="columns is-multiline">
                        <div :class="['column', 'is-multiline', 'is-one-third']"
                          v-for="(akaactor, idx) in akas.actors" :key="idx" class="image-wrapper">
                          <ActorCard :actor="akaactor"/>
                          <b-tooltip position="is-bottom" :label="$t('Remove Cast from Aka Group. Select the Aka group and Actors to remove in the Cast Filter')" multilined :delay="200">
                            <button class="button is-small is-outlined" @click="removeFromAkaGroup(akaactor.name)" v-if="actor.name.startsWith('aka:')">
                              <b-icon pack="mdi" icon="account-minus-outline"></b-icon>
                            </button>
                          </b-tooltip>
                        </div>
                      </div>
                    </b-field>
                    <b-field :label="$t('Possible Matches')" v-if="akas.possible_akas != null &&  akas.possible_akas.length!=0">
                      <div class="columns is-multiline">
                        <div :class="['column', 'is-multiline', 'is-one-third']"
                          v-for="(akaactor, idx) in akas.possible_akas" :key="idx" class="image-wrapper">
                          <ActorCard :actor="akaactor"/>
                          <b-tooltip position="is-bottom" :label="$t('Add Cast to Aka Group. Select the Aka group and Actors to add in the Cast Filter')" multilined :delay="200">
                            <button class="button is-small is-outlined" @click="addToAkaGroup(akaactor.name)" v-if='actor.name.startsWith("aka:")'>
                              <b-icon pack="mdi" icon="account-plus-outline"></b-icon>
                            </button>
                          </b-tooltip>
                        </div>
                      </div>
                    </b-field>
                  </div>
                </b-tab-item>                
                <b-tab-item :visible="colleagues.length != 0" :label="`Colleagues (${colleagues.length})`">
                  <div v-show="activeTab == 3" class="columns is-multiline scroll">
                    <div :class="['column', 'is-multiline', 'is-one-third']"
                      v-for="(colleague, idx) in colleagues" :key="idx" class="image-wrapper">
                      <ActorCard :actor="colleague" :colleague="actor.name" />
                    </div>
                  </div>
                </b-tab-item>
                <b-tab-item :label="`Links (${getActorUrls().length})`" :visible="getActorUrls().length != 0">
                  <div v-show="activeTab == 4">
                    <div >                    
                      <b-field :label="$t('Links')" >
                        <div >                       
                          <div 
                            v-for="(urllink, idx) in getActorUrls()" :key="idx">
                            <a class="tag is-info" :href="urllink.url" target="_blank" rel="noreferrer" style="margin-bottom: .5em;">{{urllink.url}}</a>                            
                          </div>                        
                        </div>
                      </b-field>
                    </div>
                  </div>
                </b-tab-item>
                <b-tab-item  :label="`Scrapers (${extrefs.length})`" :visible="extrefs.length != 0">
                  <div v-show="activeTab == 5">
                    <div >                    
                      <b-field :label="$t('Actor Scrapers')" >
                        <div >                       
                          <div v-for="(extref, idx) in extrefs" :key="idx">
                            <b-field grouped>
                              <a @click="refreshScraper(extref.external_reference.external_url)" :title="'Rescrape Actor Details now'">
                                <b-icon pack="mdi" icon="refresh" size="is-small" style="margin-right: 1em;"/>
                              </a>
                            <a class="tag is-info" :href="extref.external_reference.external_url" target="_blank" rel="noreferrer" style="margin-bottom: .5em;">{{extref.external_source}} - Updated: {{format(parseISO(extref.external_reference.external_date), "yyyy-MM-dd") }}</a>                            
                            </b-field>
                          </div>                        
                        </div>
                      </b-field>
                    </div>
                  </div>
                </b-tab-item>
              </b-tabs>
            </div>

          </div>
        </div>
      </section>
    </div>
    <button class="modal-close is-large" aria-label="close" @click="close()"></button>
    <a class="prev" @click="prevActor"
       title="Keyboard shortcut: O">&#10094;</a>
    <a class="next" @click="nextActor"
       title="Keyboard shortcut: P">&#10095;</a>

    <!-- Fullscreen Gallery Modal -->
    <div v-if="fullscreenGallery" ref="fullscreenGallery" class="fullscreen-gallery" :class="{ 'is-zoomed': fullscreenZoomed }" @click.self="handleGalleryClick">
      <button class="fullscreen-close" @click="closeFullscreenGallery" title="Close (Esc)" v-show="!fullscreenZoomed">
        <b-icon pack="mdi" icon="close" size="is-medium"></b-icon>
      </button>
      <button class="fullscreen-nav fullscreen-prev" @click="fullscreenPrev" v-if="images.length > 1" v-show="!fullscreenZoomed" title="Previous (←)">
        <b-icon pack="mdi" icon="chevron-left" size="is-large"></b-icon>
      </button>
      <button class="fullscreen-nav fullscreen-next" @click="fullscreenNext" v-if="images.length > 1" v-show="!fullscreenZoomed" title="Next (→)">
        <b-icon pack="mdi" icon="chevron-right" size="is-large"></b-icon>
      </button>
      <img 
        ref="fullscreenImage"
        :src="getFullscreenImageURL(images[fullscreenIndex])" 
        class="fullscreen-image"
        :class="{ 'is-loaded': fullscreenImageLoaded, 'is-zoomed': fullscreenZoomed, 'can-zoom': fullscreenCanZoom }"
        @load="onFullscreenImageLoad"
        @click="handleImageClick($event)"
      />
      <div v-if="!fullscreenImageLoaded" class="fullscreen-loader">
        <b-loading :is-full-page="false" :active="true"></b-loading>
      </div>
      <div class="fullscreen-info" v-if="!fullscreenZoomed">
        <span class="fullscreen-counter">{{ fullscreenIndex + 1 }} / {{ images.length }}</span>
        <span class="fullscreen-type" v-if="fullscreenImageInfo">{{ fullscreenImageInfo }}</span>
      </div>
    </div>
  </div>
</template>

<script>
import { defineComponent, nextTick } from 'vue';

import ky from 'ky'
import { format, parseISO } from 'date-fns'
import VueLoadImage from 'vue-load-image'
import { GlobalEvents } from 'vue-global-events'
import StarRating from 'vue-star-rating'
import ActorFavouriteButton from '../../components/ActorFavouriteButton'
import ActorWatchlistButton from '../../components/ActorWatchlistButton'
import ActorEditButton from '../../components/ActorEditButton'
import LinkStashdbButton from '../../components/LinkStashdbButton'
import SceneCard from '../scenes/SceneCard'
import ActorCard from './ActorCard'

export default defineComponent({
  name: 'ActorDetails',
  components: { VueLoadImage, GlobalEvents, StarRating, ActorWatchlistButton, ActorFavouriteButton, SceneCard, ActorEditButton,  ActorCard, LinkStashdbButton },

  data () {
    return {
      index: 1,
      activeTab: 0,
      activeMedia: 0,
      bioExpanded: false,
      photoRatios: {},
      bioLong: false,
      carouselSlide: 0,
      sortMultiple: true,
      countries: [],
      akas: [],
      extrefs: [],
      colleagues: [],
      carouselImagesLoaded: {},
      carouselImageRetries: {},
      carouselImageKeys: {},
      fullscreenGallery: false,
      fullscreenIndex: 0,
      fullscreenImageLoaded: false,
      fullscreenZoomed: false,
      fullscreenCanZoom: false,
      fullscreenImageInfo: null,
      carouselImageInfo: {},
    }
  },

  computed: {
    // height/width of the photo on screen, so the box fits portrait and landscape
    // shots (clamped between 3:2 landscape and 2:3 portrait)
    activePhotoRatio () {
      const r = this.photoRatios[this.carouselSlide] || 1
      return Math.min(1.5, Math.max(0.66, r))
    },
    actor () {      
      const actor = this.$store.state.overlay.actordetails.actor
      ky.get(`/api/actor/akas/${actor.id}`)
      .json()
      .then(list => {          
        this.akas = list
      })
      ky.get(`/api/actor/colleagues/${actor.id}`)
      .json()
      .then(list => {          
        this.colleagues = list
      })
      ky.get(`/api/actor/extrefs/${actor.id}`)
      .json()
      .then(list => {          
        this.extrefs = list          
      })
      return actor
    },
    // Properties for gallery
    images () {
      if (this.actor.image_arr==undefined || this.actor.image_arr=="") {
        return []
      }      
      return JSON.parse(this.actor.image_arr).filter(im => im != "")      
    },
    showEdit () {
      return this.$store.state.overlay.actoredit.show
    },
    showOpenInNewWindow () {
      return this.$store.state.optionsWeb.web.showOpenInNewWindow
    },
    // At-a-glance numbers from the actor's scenes (already loaded, no extra request).
    actorStats () {
      const scenes = this.actor.scenes || []
      const latest = scenes.map(s => s.release_date).filter(d => d && !d.startsWith('0001')).sort().pop()
      const rated = scenes.filter(s => s.star_rating > 0)
      const born = this.actor.birth_date
      const age = born && !born.startsWith('0001') ? this.calcAge(born) : null
      return [
        ...(age ? [{ label: 'Age', value: age }] : []),
        { label: 'Scenes', value: scenes.length },
        { label: 'Available', value: scenes.filter(s => s.is_available).length },
        { label: 'Scripted', value: scenes.filter(s => s.is_scripted).length },
        { label: 'Watched', value: scenes.filter(s => s.is_watched).length },
        { label: 'Avg rating', value: rated.length ? (rated.reduce((a, s) => a + s.star_rating, 0) / rated.length).toFixed(1) : '–' },
        { label: 'Latest', value: latest ? latest.slice(0, 7) : '–' }
      ]
    },
    hasDetails () {
      const a = this.actor
      const filled = v => v && v !== '[]'
      return a.birth_date !== '0001-01-01T00:00:00Z' || (a.start_year + a.end_year) !== 0 || filled(a.nationality) ||
        filled(a.ethnicity) || filled(a.hair_color) || filled(a.eye_color) || a.height || a.weight ||
        this.measurements() !== '' || filled(a.breast_type) || filled(a.aliases) || filled(a.tattoos) ||
        filled(a.piercings) || filled(a.biography)
    },
  },

  mounted () {    
      ky.get('/api/actor/countrylist')
        .json()
        .then(list => {
          this.countries=list
        })
  },

  watch: {
    'actor.id': {
      handler () {
        this.activeTab = this.hasDetails ? 0 : 1
        this.bioExpanded = false
        this.$nextTick(() => this.measureBio())
        // The actor list only carries scene summaries; fetch the full scenes for
        // the Scenes tab. Merged in place so the akas/colleagues lookups don't rerun.
        const shown = this.$store.state.overlay.actordetails.actor
        if (shown && shown.slim) {
          ky.get('/api/actor/' + shown.id).json().then(full => {
            if (this.$store.state.overlay.actordetails.actor !== shown || !full.id) return
            shown.scenes = full.scenes || []
            shown.slim = false
          }).catch(() => {})
        }
      },
      immediate: true,
    },
    images: {
      handler () {
        // Reset loaded state when images change
        this.carouselImagesLoaded = {}
        this.photoRatios = {}
        this.carouselImageRetries = {}
        this.carouselImageKeys = {}
      },

      immediate: false,
      deep: true,
    }
  },

  methods: {
    // "Show more" only when the clamped biography actually hides text
    measureBio () {
      const el = this.$refs.bio
      this.bioLong = !!el && el.scrollHeight > el.clientHeight + 2
    },
  getCarouselImageURL (url, index) {
    // Add a cache-busting key when retrying
    const key = this.carouselImageKeys[index] || 0
    const baseUrl = this.getImageURL(url, '1200x')
    return key > 0 ? baseUrl + (baseUrl.includes('?') ? '&' : '?') + '_retry=' + key : baseUrl
  },
  getFullscreenImageURL (url) {
    // Full resolution for fullscreen view
    return this.getImageURL(url, '0x0')
  },
  openFullscreenGallery (index) {
    this.fullscreenIndex = index
    this.fullscreenImageLoaded = false
    this.fullscreenGallery = true
    // Disable body scroll when fullscreen is open
    document.body.style.overflow = 'hidden'
  },
  closeFullscreenGallery () {
    this.fullscreenGallery = false
    this.fullscreenZoomed = false
    // Re-enable body scroll
    document.body.style.overflow = ''
    // Sync carousel with fullscreen position
    this.carouselSlide = this.fullscreenIndex
  },
  fullscreenPrev () {
    this.fullscreenImageLoaded = false
    this.fullscreenImageInfo = null
    this.fullscreenZoomed = false
    this.fullscreenCanZoom = false
    this.fullscreenIndex = (this.fullscreenIndex - 1 + this.images.length) % this.images.length
  },
  fullscreenNext () {
    this.fullscreenImageLoaded = false
    this.fullscreenImageInfo = null
    this.fullscreenZoomed = false
    this.fullscreenCanZoom = false
    this.fullscreenIndex = (this.fullscreenIndex + 1) % this.images.length
  },
  handleGalleryClick () {
    if (this.fullscreenZoomed) {
      this.fullscreenZoomed = false
    } else {
      this.closeFullscreenGallery()
    }
  },
  handleImageClick (event) {
    if (!this.fullscreenCanZoom) return
    
    const gallery = this.$refs.fullscreenGallery
    const img = this.$refs.fullscreenImage
    
    if (this.fullscreenZoomed) {
      // Zoom out
      this.fullscreenZoomed = false
    } else {
      // Capture click position BEFORE zoom (as ratio 0-1)
      const rect = img.getBoundingClientRect()
      const clickRatioX = (event.clientX - rect.left) / rect.width
      const clickRatioY = (event.clientY - rect.top) / rect.height
      
      // Enable zoom
      this.fullscreenZoomed = true
      
      // Use requestAnimationFrame + setTimeout to ensure layout is complete
      requestAnimationFrame(() => {
        setTimeout(() => {
          // Image natural size (full resolution)
          const natW = img.naturalWidth
          const natH = img.naturalHeight
          // Where the clicked point is in the full-size image
          const pointX = natW * clickRatioX
          const pointY = natH * clickRatioY
          // Scroll so that point is at center of viewport
          const scrollX = Math.max(0, pointX - (window.innerWidth / 2))
          const scrollY = Math.max(0, pointY - (window.innerHeight / 2))
          
          gallery.scrollTo(scrollX, scrollY)
        }, 50)
      })
    }
  },
  handleEscape () {
    if (this.fullscreenGallery) {
      this.closeFullscreenGallery()
    } else {
      this.close()
    }
  },
  toggleGallery () {
    if (this.fullscreenGallery) {
      this.closeFullscreenGallery()
    } else {
      this.openFullscreenGallery(this.carouselSlide)
    }
  },
  handleLeftArrow () {
    if (this.fullscreenGallery) {
      this.fullscreenPrev()
    } else {
      this.carouselSlide = this.carouselSlide - 1
    }
  },
  handleRightArrow () {
    if (this.fullscreenGallery) {
      this.fullscreenNext()
    } else {
      this.carouselSlide = this.carouselSlide + 1
    }
  },
  onCarouselChange (index) {
    this.scrollToActiveIndicator(index)
    // Check if the new slide's image is already loaded
    nextTick(() => {
      if (this.$refs.carouselImages && this.$refs.carouselImages[index]) {
        const img = this.$refs.carouselImages[index]
        if (img.complete && img.naturalWidth > 0) {
          this.carouselImagesLoaded[index] = true
        }
      }
    })
  },
  onCarouselImageLoad (index) {
    // Verify the image actually loaded with valid dimensions
    nextTick(() => {
      if (this.$refs.carouselImages && this.$refs.carouselImages[index]) {
        const img = this.$refs.carouselImages[index]
        if (img.naturalWidth > 100) {
          // Image loaded successfully with reasonable size
          this.carouselImagesLoaded[index] = true
          this.carouselImageRetries[index] = 0
          this.photoRatios[index] = img.naturalHeight / img.naturalWidth
          // Fetch image info (format and size) for the badge
          this.fetchCarouselImageInfo(img.src, index)
        } else if ((this.carouselImageRetries[index] || 0) < 3) {
          // Image too small, might be an error - retry
          this.retryCarouselImage(index)
        } else {
          // Give up, show whatever we have
          this.carouselImagesLoaded[index] = true
        }
      } else {
        this.carouselImagesLoaded[index] = true
      }
    })
  },
  onFullscreenImageLoad () {
    this.fullscreenImageLoaded = true
    if (this.$refs.fullscreenImage) {
      const img = this.$refs.fullscreenImage
      // Can zoom if image is larger than 90% of viewport in either dimension
      const fitsWidth = img.naturalWidth <= window.innerWidth * 0.9
      const fitsHeight = img.naturalHeight <= window.innerHeight * 0.9
      this.fullscreenCanZoom = !(fitsWidth && fitsHeight)
      // Fetch image info for fullscreen
      this.fetchFullscreenImageInfo(img.src)
    }
  },
  fetchCarouselImageInfo (url, index) {
    // Use Range request to get first 16 bytes for magic number detection
    fetch(url, { 
      method: 'GET',
      headers: { 'Range': 'bytes=0-15' }
    })
      .then(response => {
        // Get file size from Content-Range header (format: bytes 0-15/totalSize)
        const contentRange = response.headers.get('Content-Range')
        let fileSize = null
        if (contentRange) {
          const match = contentRange.match(/\/(\d+)$/)
          if (match) {
            fileSize = parseInt(match[1])
          }
        }
        return response.arrayBuffer().then(buffer => ({ buffer, fileSize }))
      })
      .then(({ buffer, fileSize }) => {
        const bytes = new Uint8Array(buffer)
        const format = this.detectFormatFromBytes(bytes)
        const size = fileSize ? this.formatBytes(fileSize) : ''
        const info = size ? `${format} · ${size}` : format
        this.carouselImageInfo[index] = info
      })
      .catch(() => {
        // Silently fail - no info displayed
      })
  },
  fetchFullscreenImageInfo (url) {
    this.fullscreenImageInfo = null
    // Use Range request to get first 16 bytes for magic number detection + full response for size
    fetch(url, { 
      method: 'GET',
      headers: { 'Range': 'bytes=0-15' }
    })
      .then(response => {
        // Get file size from Content-Range header (format: bytes 0-15/totalSize)
        const contentRange = response.headers.get('Content-Range')
        let fileSize = null
        if (contentRange) {
          const match = contentRange.match(/\/(\d+)$/)
          if (match) {
            fileSize = parseInt(match[1])
          }
        }
        // Fallback to Content-Length if no range support
        if (!fileSize) {
          const cl = response.headers.get('Content-Length')
          if (cl) fileSize = parseInt(cl)
        }
        return response.arrayBuffer().then(buffer => ({ buffer, fileSize }))
      })
      .then(({ buffer, fileSize }) => {
        const bytes = new Uint8Array(buffer)
        const format = this.detectFormatFromBytes(bytes)
        const size = fileSize ? this.formatBytes(fileSize) : ''
        this.fullscreenImageInfo = size ? `${format} · ${size}` : format
      })
      .catch(() => {
        // Silently fail
      })
  },
  detectFormatFromBytes (bytes) {
    if (bytes.length < 12) return 'IMG'
    
    // AVIF: check for 'ftyp' at offset 4 and 'avif', 'avis', or 'mif1' at offset 8
    if (bytes[4] === 0x66 && bytes[5] === 0x74 && bytes[6] === 0x79 && bytes[7] === 0x70) {
      const brand = String.fromCharCode(bytes[8], bytes[9], bytes[10], bytes[11])
      if (brand === 'avif' || brand === 'avis' || brand === 'mif1') return 'AVIF'
      if (brand === 'heic' || brand === 'heix') return 'HEIC'
    }
    
    // JPEG: starts with FF D8 FF
    if (bytes[0] === 0xFF && bytes[1] === 0xD8 && bytes[2] === 0xFF) return 'JPEG'
    
    // PNG: starts with 89 50 4E 47 (‰PNG)
    if (bytes[0] === 0x89 && bytes[1] === 0x50 && bytes[2] === 0x4E && bytes[3] === 0x47) return 'PNG'
    
    // GIF: starts with GIF87a or GIF89a
    if (bytes[0] === 0x47 && bytes[1] === 0x49 && bytes[2] === 0x46) return 'GIF'
    
    // WebP: starts with RIFF....WEBP
    if (bytes[0] === 0x52 && bytes[1] === 0x49 && bytes[2] === 0x46 && bytes[3] === 0x46 &&
        bytes[8] === 0x57 && bytes[9] === 0x45 && bytes[10] === 0x42 && bytes[11] === 0x50) return 'WebP'
    
    return 'IMG'
  },
  formatBytes (bytes) {
    if (bytes === 0) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i]
  },
  onCarouselImageError (index) {
    const retries = this.carouselImageRetries[index] || 0
    if (retries < 3) {
      this.retryCarouselImage(index)
    } else {
      // Give up, remove blur anyway
      this.carouselImagesLoaded[index] = true
    }
  },
  retryCarouselImage (index) {
    const retries = (this.carouselImageRetries[index] || 0) + 1
    this.carouselImageRetries[index] = retries
    const delay = retries * 1500 // 1.5s, 3s, 4.5s
    setTimeout(() => {
      // Increment the key to force a new request
      this.carouselImageKeys[index] = (this.carouselImageKeys[index] || 0) + 1
    }, delay)
  },
  getImageURL (u, size) {
    if (u.startsWith('http') || u.startsWith('https')) {
      return '/img/' + size + '/' + u.replace('://', ':/')
    } else {
      return u
    }
  },
  getIndicatorURL (idx) {      
    if (this.images[idx] !== undefined) {
      return this.getImageURL(this.images[idx], 'x85')
    } else {
      return '/ui/images/blank_female_profile.png'
    }
  },
  close () {      
    this.$store.commit('overlay/hideActorDetails')
  },
  setRating (val) {
    ky.post(`/api/actor/rate/${this.actor.id}`, { json: { rating: val } })
    const updatedActor = Object.assign({}, this.actor)
    updatedActor.star_rating = val
    this.actor.star_rating = val
    this.$store.commit('actorList/updateActor', updatedActor)
  },
  onZeroKey (e) {
    if (e.key === '0') this.setRating(0)
  },
  // Next/previous follow the lazily loaded list: at the end, load the next batch
  // if there is one, otherwise wrap around to the other end of what is loaded.
  showActor (data) {
    this.$store.commit('overlay/showActorDetails', { actor: data })
    this.activeMedia = 0
    this.carouselSlide = 0
  },
  async nextActor () {
    let data = this.$store.getters['actorList/nextActor'](this.actor)
    if (data === null) {
      const list = this.$store.state.actorList
      if (list.actors.length < list.total) {
        await this.$store.dispatch('actorList/load', { offset: list.actors.length })
        data = this.$store.getters['actorList/nextActor'](this.actor)
      }
      if (data === null) data = this.$store.getters['actorList/firstActor']()
    }
    if (data) this.showActor(data)
  },
  async prevActor () {
    let data = this.$store.getters['actorList/prevActor'](this.actor)
    if (data === null) data = this.$store.getters['actorList/lastActor']()
    if (data) this.showActor(data)
  },
  scrollToActiveIndicator (value) {
    const indicators = document.querySelector('.carousel-indicator')
    const active = indicators.children[value]
    indicators.scrollTo({
      top: 0,
      left: active.offsetLeft + active.offsetWidth / 2 - indicators.offsetWidth / 2,
      behavior: 'smooth'
    })
  },
  calcAge(birthdate){       
    const birthdateObj = new Date(birthdate);
    const now = new Date();
    const diffInMs = now - birthdateObj;
    const msPerYear = 1000 * 60 * 60 * 24 * 365.25; // average milliseconds per year, accounting for leap years
    const age = Math.floor(diffInMs / msPerYear);      
    return age
  },
  getYearsActive(){      
    let active = ""
    if (this.actor.start_year > 0) {
      active = this.actor.start_year 
    }      
    active +=  "-"      
    if (this.actor.end_year > 0) {
      active += this.actor.end_year 
    }
    return active
  },
  measurements(){      
    let metric_measurements=""
    let imperial_measurements=""
    if (this.actor.band_size != 0) {
      metric_measurements=this.actor.band_size
      imperial_measurements=Math.round(this.actor.band_size / 2.54)
    }
    if (this.actor.cup_size != ''){
      metric_measurements +=  this.actor.cup_size        
      imperial_measurements += this.actor.cup_size        
    }
    if (this.actor.waist_size != 0) {
      if (metric_measurements!='') {
        metric_measurements += '-'
        imperial_measurements += '-'
      }
      metric_measurements += this.actor.waist_size
      imperial_measurements += Math.round(this.actor.waist_size  / 2.54)
    }
    if (this.actor.hip_size != 0) {
      if (metric_measurements!='') {
        metric_measurements += '-'
        imperial_measurements += '-'
      }
      metric_measurements += this.actor.hip_size
      imperial_measurements += Math.round(this.actor.hip_size / 2.54)
    } 
    if (metric_measurements==''){
      return ''
    }
    return imperial_measurements + " / " + metric_measurements
  },
  joinArray(jsonArr){
    const arr = JSON.parse(jsonArr);
    return  arr.join(", ");       
  },
  setActorImage () {
    ky.post('/api/actor/setimage', {
    json: {
      actor_id: this.actor.id,
      url: this.images[this.carouselSlide]
    }}).json().then(data => {        
      this.$store.state.overlay.actordetails.actor = data
      this.carouselSlide=0
      this.$store.dispatch('actorList/load', { offset: this.$store.state.actorList.offset - this.$store.state.actorList.limit })
    })    
  },
  deleteActorImage () {
    ky.delete('/api/actor/delimage', {
    json: {
      actor_id: this.actor.id,
      url: this.images[this.carouselSlide]
    }}).json().then(data => {
      this.$store.state.overlay.actordetails.actor = data
    })    
  },
  getCountryName(countryCode){
    const country = this.countries.find(c => c.code === countryCode)
    if (country == undefined) {
      return countryCode
    }
    return country.name
  },
  getCountryFlag(countryCode){
    const country = this.countries.find(c => c.code === countryCode)
    if (country == undefined) {
      return 'https://flagcdn.com/' + countryCode.toLowerCase() +'.svg'
    }
    return country.flag_url
  },
  getWeight(kg) {
      return kg + " kg - " + Math.round( kg * 2.20462) + " lbs"
  },
  getHeight(cm){
    const totalInches = Math.round(cm / 2.54)
    let feet = Math.floor(totalInches / 12)
    return cm + " cm - " + feet + "' " +  Math.round(totalInches - (feet*12)) + '"'
  },
  getActorUrls() {
    if (this.actor.urls=="")
    {
      return []
    }      
    let array = JSON.parse(this.actor.urls)      
    return array
  },
  createAkaGroup () {
    this.$store.state.actorList.isLoading = true
    let actorlist = [this.actor.name]
    for (let idx = 0; idx < this.akas.possible_akas.length; idx++) {
      actorlist.push(this.akas.possible_akas[idx].name)
    }
    ky.post('/api/aka/create', {json: {actorList: actorlist}}).json().then(data => {
      if (data.status != '') {
        this.$buefy.toast.open({message: `Warning:  ${data.status}`, type: 'is-warning', duration: 5000})
      }
      ky.get('/api/actor/'+this.actor.id).json().then(data => {
        if (data.id != 0){
          this.$store.state.overlay.actordetails.actor = data          
        }          
      })
      this.$store.state.actorList.isLoading = false
      this.$store.dispatch('actorList/load', { offset: this.$store.state.actorList.offset - this.$store.state.actorList.limit })

    })
  },
  deleteAkaGroup () {
    this.$store.state.actorList.isLoading = true
    ky.post('/api/aka/delete', {json: {name: this.actor.name}}).json().then(() => {
      this.$store.state.actorList.isLoading = false
    }).then(() => {
      this.$store.dispatch('actorList/load', { offset: this.$store.state.actorList.offset - this.$store.state.actorList.limit })
      this.close()
    }
    )
  },
  addToAkaGroup (newMember) {
    this.$store.state.actorList.isLoading = true
    ky.post('/api/aka/add', {json: {actorList: [this.actor.name, newMember]}}).json().then(data => {        
      // delete old aka & add new name
      if (data.status != '') {
        this.$buefy.toast.open({message: `Warning:  ${data.status}`, type: 'is-warning', duration: 5000})
      }
      ky.get('/api/actor/'+this.actor.id).json().then(data => {
        if (data.id != 0){
          this.$store.state.overlay.actordetails.actor = data          
        }          
      })
      this.$store.state.actorList.isLoading = false
    })
    
  },
  removeFromAkaGroup (memberToRemove) {
    this.$store.state.actorList.isLoading = true
    ky.post('/api/aka/remove', {json: {actorList: [this.actor.name, memberToRemove]}}).json().then(data => {        
      // delete old aka & add new name
      if (data.status != '') {
        this.$buefy.toast.open({message: `Warning:  ${data.status}`, type: 'is-warning', duration: 5000})
      }
      ky.get('/api/actor/'+this.actor.id).json().then(data => {          
        if (data.id != 0){
          this.$store.state.overlay.actordetails.actor = data          
        }          
      })
      this.$store.state.actorList.isLoading = false
    })
  },
  enableNewAkaGroup () {
    if (this.actor.name.startsWith("aka:")){
      return false
    }
    if (this.akas.aka_groups != null)
    {
      return false
    }
    if (this.akas.possible_akas == null)
    {
      return false
    }
    for (let idx = 0; idx < this.akas.possible_akas.length; idx++) {
      if (this.akas.possible_akas[idx].name.startsWith("aka:")) {
        return false
      }
    }      
    return true
  },
  getCastScenesUrl(actor) {
    let newfilters = Object.assign({}, this.$store.state.sceneList.filters);
    newfilters.cast = actor;
    newfilters.dlState = "any"
    newfilters.isAvailable=null
    newfilters.isAccessible=null
    return this.$router.resolve({
      name: 'scenes',
      query: { q: btoa(unescape(encodeURIComponent(JSON.stringify(newfilters)))) }
    }).href
  },
  refreshScraper(url){
    if (url.includes('stashdb')) {
      this.$store.state.actorList.isLoading = true
      const lastSlashIndex = url.lastIndexOf('/');
      ky.get('/api/extref/stashdb/refresh_performer/'+url.substring(lastSlashIndex + 1)).then(() => {
        ky.get('/api/actor/'+this.actor.id).json().then(data => {
          if (data.id != 0){
            this.$store.state.overlay.actordetails.actor = data
            this.$store.state.actorList.isLoading = false
            this.$store.dispatch('actorList/load', { offset: this.$store.state.actorList.offset - this.$store.state.actorList.limit })
          }
        })
      })
    } else {
      this.$store.state.actorList.isLoading = true
      ky.post('/api/extref/generic/scrape_single', { json: {id: this.actor.id,url: url}})
        .then(() => {
          ky.get('/api/actor/'+this.actor.id).json().then(data => {
            if (data.id != 0){
              this.$store.state.overlay.actordetails.actor = data
              this.$store.state.actorList.isLoading = false
              this.$store.dispatch('actorList/load', { offset: this.$store.state.actorList.offset - this.$store.state.actorList.limit })
            }
          })
        })
    }
    this.$store.state.actorList.isLoading = false
  },
  format,
  parseISO
},
});
</script>

<style lang="less" scoped>
.actor-stats {
  display: flex;
  gap: 0.75rem;
  margin: 0 0 0.5rem;
  padding: 0.4rem 0.75rem;
  border: 1px solid rgba(127, 127, 127, 0.18);
  border-radius: 8px;
  font-variant-numeric: tabular-nums;
}
.actor-stat {
  display: flex;
  flex-direction: column;
  flex: 1 1 0;
  min-width: 0;
}
.actor-stat-value {
  font-size: 1rem;
  font-weight: 600;
  line-height: 1.25;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.actor-stat-label {
  font-size: 0.65rem;
  white-space: nowrap;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  opacity: 0.6;
}
.actor-empty-details {
  padding: 0.75rem 0;
  opacity: 0.65;
  font-size: 0.9rem;
}

.bbox {
  flex: 1 0 calc(25%);
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  padding: 0;
  line-height: 0;
}

/* Square photo area, but never taller than the modal can show next to the
   tabs, thumbnails and buttons (90vh modal minus roughly 380px of chrome). */
/* Photo box follows the current photo's shape, but never taller than the modal
   can show next to the tabs, thumbnails and buttons (90vh minus ~260px). */
.is-1by1 {
  padding-top: min(calc(var(--photo-ratio, 1) * 100%), calc(90vh - 260px)) !important;
  transition: padding-top 0.25s ease;
}

.modal-card {
  width: 85%;
  max-height: 90vh;
  overflow-y: auto;
}

@media (max-width: 768px) {
  .modal-card {
    width: 96vw;
    margin: 0 2vw;
  }
}

.missing {
  opacity: 0.6;
}

.block-tab-content {
  flex: 1 1 auto;
}

.block-info {
}

.block-tags {
  max-height: 200px;
  overflow: scroll;
  scrollbar-width: none;
}

.block-tags::-webkit-scrollbar {
  display: none;
}

.block-opts {
}

.vue-star-rating {
    line-height: 0;
}

.scene-id {
  position: absolute;
  right:10px;
  bottom: 5px;
  font-size: 11px;
  color: #b0b0b0;
}

.prev, .next {
  cursor: pointer;
  position: absolute;
  top: 50%;
  width: auto;
  padding: 16px;
  margin-top: -50px;
  color: white;
  font-weight: bold;
  font-size: 24px;
  border-radius: 0 3px 3px 0;
  user-select: none;
  -webkit-user-select: none;
}

.next {
  right: 0;
  border-radius: 3px 0 0 3px;
}

.prev {
  left: 0;
  border-radius: 3px 0 0 3px;
}

span.is-active img {
  border: 2px;
}

.pathDetails {
  color: #b0b0b0;
}

.heatmapFunscript {
  width: 100%;
  padding: 0;
  margin-top: 0.5em;
}

.heatmapFunscript img {
  border: 1px #888 solid;
  width: 100%;
  height: 20px;
  margin: 0;
  padding: 0;
}
.videosize {
  color: rgb(60, 60, 60);
  font-weight: 550;
}

:deep(.carousel .carousel-indicator .indicator-item img) {
  border-radius: 4px;
}
:deep(.carousel .carousel-indicator) {
  justify-content: flex-start;
  width: 100%;
  max-width: min-content;
  margin-left: auto;
  margin-right: auto;
  overflow: auto;
}
:deep(.carousel .carousel-indicator .indicator-item:not(.is-active)) {
  opacity: 0.5;
}
.is-divider {
  margin: .8rem 0;
}
.image-row {
  display: flex;  
}
.image-wrapper {
  position: relative;
}
.thumbnail {
  height: 100px;
  margin-right: .5em;
  object-fit: cover;
}
.tooltip {
  position: absolute;
  z-index: 1;
  top: 50px;
  right: 100%;
  width: 400px;
  height: 400px;
  background-color: white;
  box-shadow: 0 2px 5px rgba(0, 0, 0, 0.3);
  display: flex;
  justify-content: center;
  align-items: center;
  padding: 10px;
  transform: translateX(10px);
}
.tooltip img {
  max-width: 100%;
  max-height: 100%;
}
div.scroll {
  height: 1000px;
  overflow-x: hidden;
  overflow-y: auto;
  text-align: center;
}
/* Scenes tab header: the open-in-new link sits inline, same height as the other tabs */
.tab-label {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  line-height: 1.5;
}
.tab-ext {
  display: inline-flex;
  padding: 0 !important;
  border: none !important;
  margin: 0 !important;
  opacity: 0.6;
}
.tab-ext:hover {
  opacity: 1;
}
.tab-ext :deep(.icon) {
  width: 14px;
  height: 14px;
}

/* Details: label/value pairs in two columns; long lists span both and wrap */
.actor-facts {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px 20px;
  margin: 0 0 12px;
}
.fact {
  min-width: 0;
}
.fact.is-wide {
  grid-column: 1 / -1;
}
.fact dt {
  font-size: 0.68rem;
  font-weight: 600;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  opacity: 0.55;
  margin-bottom: 1px;
}
.fact dd {
  margin: 0;
  font-size: 0.9rem;
  line-height: 1.35;
  overflow-wrap: anywhere;
}
.fact-flag {
  display: flex;
  align-items: center;
  gap: 6px;
}
.fact-flag-img {
  display: block;
  height: 12px;
  border-radius: 2px;
  box-shadow: 0 0 0 1px rgba(0, 0, 0, 0.15);
}
@media (max-width: 520px) {
  .actor-facts { grid-template-columns: minmax(0, 1fr); }
}
.actor-bio {
  padding: 10px 12px;
  border-radius: 8px;
  background: rgba(127, 127, 127, 0.08);
  font-size: 0.85rem;
  line-height: 1.5;
}
.actor-bio p {
  margin: 0;
  white-space: pre-line;
  overflow-wrap: anywhere;
}
.actor-bio p.is-clamped {
  display: -webkit-box;
  -webkit-line-clamp: 5;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.bio-toggle {
  margin-top: 4px;
  padding: 0;
  border: none;
  background: none;
  font: inherit;
  font-size: 0.75rem;
  color: var(--accent, #7957d5);
  cursor: pointer;
}
.flexcentre {
  display: flex;
  justify-content: center;
}

/* Carousel image blur-to-sharp effect */
.carousel-image-container {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
}

.carousel-image {
  border-radius: 8px; /* same radius as cards and the player */
  max-width: 100%;
  max-height: 100%;
  width: auto;
  height: auto;
  object-fit: contain;
  filter: blur(10px);
  transform: scale(1.05);
  transition: filter 0.4s ease-out, transform 0.4s ease-out;
}

.carousel-image.is-loaded {
  filter: blur(0);
  transform: scale(1);
}

/* Fullscreen Gallery Styles */
.carousel-wrapper {
  position: relative;
  /* Height comes from the capped photo area above; the old fixed 2:3 ratio made
     the gallery column taller than the modal and forced a scrollbar. */
  min-height: 180px;
  overflow: hidden;
}
.fullscreen-btn {
  position: absolute;
  top: 10px;
  right: 10px;
  z-index: 10;
  background: rgba(0, 0, 0, 0.6);
  border: none;
  border-radius: 4px;
  padding: 6px 8px;
  cursor: pointer;
  color: white;
  transition: background 0.2s;
}
.fullscreen-btn:hover {
  background: rgba(0, 0, 0, 0.8);
}
.fullscreen-gallery {
  position: fixed;
  top: 0;
  left: 0;
  width: 100vw;
  height: 100vh;
  background: rgba(0, 0, 0, 0.95);
  z-index: 9999;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
}
.fullscreen-gallery.is-zoomed {
  display: block;
  overflow: auto;
  /* Dark scrollbars */
  scrollbar-width: thin;
  scrollbar-color: rgba(255,255,255,0.3) transparent;
}
.fullscreen-gallery.is-zoomed::-webkit-scrollbar {
  width: 8px;
  height: 8px;
}
.fullscreen-gallery.is-zoomed::-webkit-scrollbar-track {
  background: transparent;
}
.fullscreen-gallery.is-zoomed::-webkit-scrollbar-thumb {
  background: rgba(255,255,255,0.3);
  border-radius: 4px;
}
.fullscreen-gallery.is-zoomed::-webkit-scrollbar-thumb:hover {
  background: rgba(255,255,255,0.5);
}
.fullscreen-close {
  position: absolute;
  top: 20px;
  right: 20px;
  background: rgba(255, 255, 255, 0.1);
  border: none;
  border-radius: 50%;
  width: 50px;
  height: 50px;
  cursor: pointer;
  color: white;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background 0.2s;
  z-index: 10001;
}
.fullscreen-close:hover {
  background: rgba(255, 255, 255, 0.2);
}
.fullscreen-nav {
  position: absolute;
  top: 50%;
  transform: translateY(-50%);
  background: rgba(255, 255, 255, 0.1);
  border: none;
  border-radius: 50%;
  width: 60px;
  height: 60px;
  cursor: pointer;
  color: white;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background 0.2s;
  z-index: 10001;
}
.fullscreen-nav:hover {
  background: rgba(255, 255, 255, 0.2);
}
.fullscreen-prev {
  left: 20px;
}
.fullscreen-next {
  right: 20px;
}
.fullscreen-image {
  max-width: 90vw;
  max-height: 90vh;
  object-fit: contain;
  opacity: 0;
  cursor: default;
  transition: opacity 0.3s ease;
}
.fullscreen-image.can-zoom {
  cursor: zoom-in;
}
.fullscreen-image.is-zoomed {
  max-width: none;
  max-height: none;
  cursor: zoom-out;
}
.fullscreen-image.is-loaded {
  opacity: 1;
}
.fullscreen-loader {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
}
.fullscreen-info {
  position: absolute;
  bottom: 20px;
  left: 50%;
  transform: translateX(-50%);
  display: flex;
  gap: 10px;
  align-items: center;
}
.fullscreen-counter {
  color: white;
  font-size: 14px;
  background: rgba(0, 0, 0, 0.5);
  padding: 8px 16px;
  border-radius: 20px;
}
.fullscreen-type {
  color: white;
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  background: rgba(100, 100, 100, 0.5);
  padding: 6px 12px;
  border-radius: 12px;
}

/* Carousel image type tag */
.carousel-type-tag {
  position: absolute;
  bottom: 10px;
  right: 10px;
  font-size: 10px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  color: rgba(255, 255, 255, 0.9);
  background: rgba(0, 0, 0, 0.5);
  padding: 4px 10px;
  border-radius: 10px;
  pointer-events: none;
  z-index: 10;
}

</style>