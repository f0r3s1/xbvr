<template>
  <div class="modal is-active">
    <GlobalEvents
      :filter="e => !['INPUT', 'TEXTAREA'].includes(e.target.tagName)"
      @keyup.esc="handleEscape"
      @keydown.left="handleLeftArrow"
      @keydown.right="handleRightArrow"
      @keydown.o="prevScene"
      @keydown.p="nextScene"
      @keydown.f="$store.commit('sceneList/toggleSceneList', {scene_id: item.scene_id, list: 'favourite'})"
      @keydown.exact.w="$store.commit('sceneList/toggleSceneList', {scene_id: item.scene_id, list: 'watchlist'})"
      @keydown.shift.w="$store.commit('sceneList/toggleSceneList', {scene_id: item.scene_id, list: 'watched'})"
      @keydown.t="$store.commit('sceneList/toggleSceneList', {scene_id: item.scene_id, list: 'trailerlist'})"
      @keydown.e="$store.commit('overlay/editDetails', {scene: item})"
      @keydown.s="$store.commit('overlay/showSearchStashdbScenes', {scene: item})"
      @keydown.g="toggleGallery"
      @keydown="onZeroKey"
    />

    <div class="modal-background" @click="close"></div>

    <div class="modal-card">
      <section class="modal-card-body" :style="paletteStyle">
        <div class="columns">

          <div class="column is-half">
            <b-tabs v-model="activeMedia" position="is-centered" :animated="false" class="media-tabs">

              <b-tab-item label="Gallery">
                <div class="carousel-wrapper">
                  <b-carousel v-model="carouselSlide" @change="onCarouselChange" :autoplay="false" :indicator-inside="false">
                    <b-carousel-item v-for="(carousel, i) in images" :key="i">
                      <div class="image is-1by1 is-full carousel-image-container">
                        <img 
                          ref="carouselImages"
                          :src="getCarouselImageURL(carousel.url, i)" 
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
                            <template #image><img :src="getIndicatorURL(props.i)" style="height:40px;"/></template>
                            <template #preloader><img src="/ui/images/blank.png" style="height:40px;"/></template>
                            <template #error><img src="/ui/images/blank.png" style="height:40px;"/></template>
                          </vue-load-image>
                        </span>
                    </template>
                  </b-carousel>
                  <button class="fullscreen-btn" @click="openFullscreenGallery(carouselSlide)" title="Fullscreen (G)">
                    <b-icon pack="mdi" icon="fullscreen" size="is-small"></b-icon>
                  </button>
                </div>
              </b-tab-item>

              <b-tab-item label="Player" v-if="!displayingAlternateSource && fileCount > 0">
                <div class="player-wrapper">
                  <video ref="player" class="video-js vjs-default-skin" controls playsinline preload="none"/>
                </div>
                <div class="skip-controls">
                  <div class="skip-group">
                    <b-tooltip v-for="(skipBack, i) in skipBackIntervals" :key="i" :active="skipBack == lastSkipBackInterval" :label="$t('Keyboard shortcut: Left Arrow')"
                        position="is-top" type="is-primary is-light">
                      <b-button class="tag is-small is-outlined is-info is-light" @click="playerStepBack(skipBack)">
                        <b-icon v-if="skipBack == lastSkipBackInterval" pack="mdi" icon="arrow-left-thin" size="is-small"></b-icon> {{ skipBack }}
                      </b-button>
                    </b-tooltip>
                  </div>
                  <div class="skip-group">
                    <b-tooltip v-for="(skipForward, i) in skipForwardIntervals" :key="i" :active="skipForward == lastSkipFowardInterval" :label="$t('Keyboard shortcut: Right Arrow')"
                        position="is-top" type="is-primary is-light">
                      <b-button class="tag is-small is-outlined is-info is-light" @click="playerStepForward(skipForward)">
                        <b-icon v-if="skipForward == lastSkipFowardInterval" pack="mdi" icon="arrow-right-thin" size="is-small"></b-icon> +{{ skipForward }}
                      </b-button>
                    </b-tooltip>
                  </div>
                </div>
             </b-tab-item>

            </b-tabs>

          </div>

          <div class="column is-half">

            <div class="detail-header">
              <!-- Title -->
              <h3 class="detail-title">
                <span v-if="item.title">{{ item.title }}</span>
                <span v-else class="missing">(no title)</span>
              </h3>

              <!-- Meta line: site, duration, date -->
              <div class="detail-meta">
                <span class="detail-meta-left">
                  <span class="site-link">
                    <a @click.stop="showSiteScenes([item.site])">{{ item.site }}</a>
                    <a :href="item.scene_url" target="_blank" rel="noreferrer" @click.stop class="site-external"><b-icon pack="mdi" icon="open-in-new" custom-size="mdi-12px"/></a>
                  </span>
                  <a v-if="item.members_url != ''" :href="item.members_url" target="_blank" rel="noreferrer" @click.stop class="members-link">
                    <b-icon pack="mdi" icon="link-lock" custom-size="mdi-14px"/>
                  </a>
                  <span v-if="item.duration" class="detail-duration">{{ item.duration }} min</span>
                </span>
                <span class="detail-date">
                  {{ format(parseISO(item.release_date), "yyyy-MM-dd") }}
                </span>
              </div>

              <!-- Rating + Actions -->
              <div class="detail-actions-row">
                <div class="detail-rating" v-if="!displayingAlternateSource">
                  <star-rating :key="item.id" v-model="item.star_rating" :rating="item.star_rating" @rating-selected="setRating"
                               :increment="0.5" :star-size="18" :show-rating="false" />
                  <b-icon pack="mdi" icon="autorenew" size="is-small" @click="setRating(0)" class="rating-reset" data-tooltip="Reset rating"/>
                </div>
                <div v-if="displayingAlternateSource" class="detail-rating">
                  <strong>Linked scene, Not an XBVR Scene</strong>
                </div>
                <div class="detail-actions">
                  <a class="button is-primary is-outlined is-small" @click="searchAlternateSourceScene()" data-tooltip="Search for a different scene" v-if="displayingAlternateSource">
                    <b-icon pack="mdi" icon="movie-search-outline" size="is-small"/>
                  </a>
                  <a class="button is-primary is-outlined is-small" @click="scrapeScene()" data-tooltip="Scrape and create an XBVR scene" v-if="displayingAlternateSource">
                    <b-icon pack="mdi" icon="plus" size="is-medium"/>
                  </a>
                  <a class="button is-primary is-outlined is-small" @click="refreshExtRef()" data-tooltip="Refresh scene data and relink" v-if="displayingAlternateSource">
                    <b-icon pack="mdi" icon="refresh" size="is-small"/>
                  </a>
                  <a class="button is-danger is-outlined is-small" @click="flagExtRefDeleted()" data-tooltip="Unlink scene permanently" v-if="displayingAlternateSource">
                    <b-icon pack="mdi" icon="delete" size="is-small"/>
                  </a>
                  <hidden-button :item="item" v-if="!displayingAlternateSource"/>
                  <watchlist-button :item="item" v-if="!displayingAlternateSource"/>
                  <trailerlist-button :item="item" v-if="!displayingAlternateSource"/>
                  <favourite-button :item="item" v-if="!displayingAlternateSource"/>
                  <wishlist-button :item="item" v-if="!displayingAlternateSource"/>
                  <watched-button :item="item" v-if="!displayingAlternateSource"/>
                  <edit-button :item="item"/>
                  <refresh-button :item="item" v-if="!displayingAlternateSource"/>
                  <rescrape-button :item="item" v-if="!displayingAlternateSource"/>
                  <link-stashdb-button :item="item" objectType="scene" />
                  <!-- Alternate source icons inline -->
                  <div v-for="(altsrc, idx) in alternateSourcesWithTitles" :key="idx" class="altsrc-image-wrapper" @click="showExtRefScene(altsrc)">
                    <b-tooltip type="is-light" :label="altsrc.title" :delay="0" append-to-body>
                      <vue-load-image>
                        <template #image><img :src="getImageURL(altsrc.site_icon)" alt="Image" width="28px"/></template>
                        <template #error><b-icon pack="mdi" icon="link" size="is-small"/></template>
                      </vue-load-image>
                    </b-tooltip>
                  </div>
                </div>
              </div>
            </div>

            <!-- Cast -->
            <div class="cast-section" v-if="activeTab != 1 && !displayingAlternateSource && castimages.length > 0">
              <div class="cast-row">
                <div v-for="(image, idx) in castimages" :key="idx" class="cast-item"
                     :data-tooltip="image.actor_label"
                     @click='showActorDetail([image.actor_id])'>
                  <vue-load-image>
                    <template #image><img :src="getImageURL(image.src)" class="cast-thumb"/></template>
                    <template #preloader><img src="/ui/images/blank_female_profile.png" class="cast-thumb"/></template>
                    <template #error><img src="/ui/images/blank_female_profile.png" class="cast-thumb"/></template>
                  </vue-load-image>
                </div>
              </div>
            </div>

            <!-- Tags: content only (cast shown above, site in meta) -->
            <div class="tags-section" :class="{ 'no-accent': !accentColorTags }" v-if="activeTab != 1 && item.tags && item.tags.length > 0">
              <div class="tag-group">
                <a v-for="(tag, idx) in item.tags" :key="'tag' + idx"
                   @click='showTagScenes([tag.name])' class="tag is-info is-small tag-pill">
                  <span>{{ tag.name }} ({{ tag.count }})</span>
                  <b-icon v-if="showOpenInNewWindow" pack="mdi" icon="open-in-new" size="is-small"
                    class="tag-external" @click.stop="openTagInNewWindow(tag.name)"/>
                </a>
              </div>
            </div>

            <div class="block-tags block" v-if="activeTab == 1">
             <b-taglist>
              <b-tooltip  type="is-danger" :label="disableSaveMsg()" position="is-right" :delay=250 :active="disableSaveButtons()">
                <b-button @click="updateCuepoint(false)" class="tag is-info is-small is-warning" accesskey="a" :disabled="disableSaveButtons()" >
                  <u>A</u>dd New
                </b-button>
              </b-tooltip>
                <b-button @click="vidPosition = new Date(0,0,0,0,0, 0, player.currentTime() * 1000)" class="tag is-info is-small is-warning" accesskey="t">Current <u>T</u>ime</b-button>
              <b-tooltip type="is-danger" :label="$t(disableSaveMsg())" position="is-right" :delay=250 :active="disableSaveButtons()">
                <b-button v-if="currentCuepointId > 0" @click="updateCuepoint(true)" class="tag is-info is-small is-warning" accesskey="s"
                  :disabled="disableSaveButtons()" >
                  <u>S</u>ave Edit
                </b-button>
              </b-tooltip>
                <b-button v-if="cuepointName!=''" @click='cuepointName=""' class="tag is-info is-small is-warning" >Clear Cuepoint Name</b-button>
                <b-button v-if="tagAct!=''" @click='setCuepointName("")' class="tag is-info is-small is-warning" accesskey="c"><u>C</u>lear Action</b-button>
              </b-taglist>
            </div>

            <div class="is-divider" data-content="Cuepoint Positions" v-if="activeTab == 1"></div>
            <div class="block-tags block" v-if="activeTab == 1">
              <b-taglist>
                <b-button v-for="(c, idx) in cuepointPositionTags.slice(1)" :key="'pos' + idx" @click='setCuepointName([c])' class="tag is-info is-small">{{c}}</b-button>
              </b-taglist>
            </div>
            <div class="is-divider" data-content="Default Cuepoint Actions" v-if="activeTab == 1"></div>
            <div class="block-tags block" v-if="activeTab == 1">
              <b-taglist>
                <b-button v-for="(c, idx) in cuepointActTags.slice(1)" :key="'action' + idx" @click='setCuepointName([c])' class="tag is-info is-small">{{c}}</b-button>
              </b-taglist>
            </div>
            <div class="is-divider" data-content="Cast Cuepoints" v-if="activeTab == 1"></div>
            <div class="block-tags block" v-if="activeTab == 1">
              <b-taglist>
                <b-button v-for="(c, idx) in item.cast" :key="'cast' + idx" @click='setCuepointName([c.name])' class="tag is-info is-small">{{c.name}}</b-button>
              </b-taglist>
            </div>
            <div class="is-divider" data-content="Scene Tag Cuepoints" v-if="activeTab == 1"></div>
            <div class="block-tags block" v-if="activeTab == 1">
              <b-taglist>
                <b-button v-for="(tag, idx) in item.tags" :key="'tag' + idx" @click='setCuepointName([tag.name])'
                   class="tag is-info is-small">{{ tag.name }}</b-button>
              </b-taglist>
            </div>


            <div class="block-opts block">
              <b-tabs v-model="activeTab" :animated="false">

                <b-tab-item :label="`Files (${fileCount})`" v-if="!displayingAlternateSource">
                  <div class="block-tab-content block">
                    <div class="content media is-small" v-for="(f, idx) in filesByType" :key="idx">
                      <div class="media-left">
                        <button rounded class="button is-success is-small" @click='playFile(f)'
                                v-show="f.type === 'video'">
                          <b-icon pack="mdi" icon="play" size="is-small"></b-icon>
                        </button>
                        <b-tooltip :label="$t('Select this script for export')" position="is-right">
                        <button rounded class="button is-info is-small is-outlined" @click='selectScript(f)'
                          v-show="f.type === 'script'" v-bind:class="{ 'is-success': f.is_selected_script, 'is-info' :!f.is_selected_script }">
                          <b-icon pack="mdi" icon="pulse"></b-icon>
                        </button>
                        </b-tooltip>
                        <button rounded class="button is-info is-small is-outlined" disabled
                                v-show="f.type === 'hsp'">
                          <b-icon pack="mdi" icon="safety-goggles"></b-icon>
                        </button>
                        <button rounded class="button is-info is-small is-outlined" disabled
                                v-show="f.type === 'subtitles'">
                          <b-icon pack="mdi" icon="subtitles"></b-icon>
                        </button>
                      </div>
                      <div class="media-content" style="overflow-wrap: break-word;">
                        <strong>{{ f.filename }}</strong><br/>
                        <small>
                          <span class="pathDetails">{{ f.path }}</span>
                          <br/>
                          {{ prettyBytes(f.size) }}<span v-if="f.type === 'video'"> ({{ prettyBytes(f.video_bitrate, { bits: true })  }}/s)</span>,
                          <span v-if="f.type === 'video'"><span class="videosize">{{ f.video_width }}x{{ f.video_height }} {{ f.video_codec_name }}</span>, {{ f.projection }},&nbsp;</span>
                          <span v-if="f.duration > 1">{{ humanizeSeconds(f.duration) }},</span>
                          {{ format(parseISO(f.created_time), "yyyy-MM-dd") }}
                        </small>
                        <div v-if="f.type === 'script' && f.has_heatmap" class="heatmapFunscript">
                          <img :src="getHeatmapURL(f.id)"/>
                        </div>
                      </div>
                      <div class="media-right">
                        <button class="button is-dark is-small is-outlined" data-tooltip="Unmatch file from scene" @click='unmatchFile(f)'>
                          <b-icon pack="mdi" icon="link-off" size="is-small"></b-icon>
                        </button>&nbsp;
                        <button class="button is-danger is-small is-outlined" data-tooltip="Delete file from disk" @click='removeFile(f)'>
                          <b-icon pack="mdi" icon="delete" size="is-small"></b-icon>
                        </button>
                      </div>
                    </div>
                  </div>
                </b-tab-item>

                <b-tab-item :label="`Cuepoints (${sortedCuepoints.length})`" v-if="!displayingAlternateSource">
                  <div class="block-tab-content block">
                    <div class="block" >
                      <div class="columns">
                        <div class="column is-2">
                        <b-field label="Track" width="7.25em" label-position="on-border">
                          <b-input v-model="track" width="7.25em"></b-input>
                        </b-field>
                        </div>
                        <div class="column">
                        <b-field label="Name" label-position="on-border">
                          <b-autocomplete v-model="cuepointName" :data="filteredCuepointPositionList" :open-on-focus="true"></b-autocomplete>
                        </b-field>
                        </div>
                        <div class="column is-2">
                        <b-field label="Start" label-position="on-border">
                          <b-timepicker v-model="vidPosition" rounded editable placeholder="Defaults to player position" hour-format="24" :enable-seconds="true" :max-time="maxTime" :time-formatter="timeFormatter" :time-parser="timeParser" >
                          <b-button
                            label="Current Time"
                            type="is-primary"
                            @click="vidPosition = new Date(0,0,0,0,0, 0, player.currentTime() * 1000)" />
                          </b-timepicker>
                        </b-field>
                        </div>
                        <div class="column is-2">
                          <b-field label="End" label-position="on-border">
                          <b-timepicker v-model="endTime" rounded editable placeholder="Defaults to player position" hour-format="24" :enable-seconds="true" :max-time="maxTime" :time-formatter="timeFormatter" :time-parser="timeParser" >
                          <b-button
                            label="Current Time"
                            type="is-primary"
                            @click="endTime = new Date(0,0,0,0,0, 0, player.currentTime() * 1000)" />
                          </b-timepicker>
                        </b-field>
                        </div>
                      </div>
                    </div>
                    <div>
                      <!-- :sort-multiple="sortMultiple" :sort-multiple-data="cuepointSorting" -->
                        <b-table :data="sortedCuepoints"  :narrowed=true :per-page=7 focusable striped sticky-header
                          @select="cuepointSelected">
                          <!-- paginated  pagination-position="top" :pagination-rounded=true pagination-size="is-small" -->
                          <b-table-column field="track" label="Track" width="7.25em" v-slot="props" >
                            {{ props.row.track ==null ? "" :  props.row.track }}
                          </b-table-column>
                          <b-table-column field="name" label="Name" v-slot="props"  is-small>
                            {{ props.row.name }}
                          </b-table-column>
                          <b-table-column field="time_start" label="Start" v-slot="props" width="6.5em"  >
                            {{ humanizeSeconds1DP(props.row.time_start) }}
                          </b-table-column>
                          <b-table-column field="time_end" label="End" v-slot="props" width="6.5em"  >
                            {{ props.row.time_end==null ? "" :  humanizeSeconds1DP(props.row.time_end) }}
                          </b-table-column>
                          <b-table-column field="rating" v-slot="props" width="7em"  >
                            <b-field v-if="props.row.track!=null">
                              <star-rating :key="props.row.id" v-model="props.row.rating" :rating="props.row.rating" @rating-selected="setCuepointRating(props.row)" :increment="0.5" :star-size="10" />
                              <b-icon v-if="props.row.rating>0" pack="mdi" icon="autorenew" size="is-small" @click="clearCuepointRating(props.row)" style="padding-left: .25em;padding-top: .5em;"/>
                            </b-field>
                          </b-table-column>
                          <b-table-column v-slot="props" width="1em" >
                            <button class="button is-danger is-outlined is-small" @click="deleteCuepoint(props.row.id)" data-tooltip="Delete cuepoint">
                              <b-icon pack="mdi" icon="delete" />
                            </button>
                          </b-table-column>
                        </b-table>
                    </div>
                  </div>
                </b-tab-item>

                <b-tab-item label="Description" v-if="item.synopsis">
                  <div class="block-tab-content block">
                    <div class="description-text">{{ item.synopsis }}</div>
                  </div>
                </b-tab-item>

                <b-tab-item label="Watch history" v-if="!displayingAlternateSource">
                  <div class="block-tab-content block">
                    <div>
                      {{ historySessionsCount }} view sessions, total duration
                      {{ humanizeSeconds(historySessionsDuration) }}
                    </div>
                    <div class="content is-small">
                      <div class="block" v-for="(session, idx) in item.history" :key="idx">
                        <strong>{{ format(parseISO(session.time_start), "yyyy-MM-dd kk:mm:ss") }} -
                          {{ humanizeSeconds(session.duration) }}</strong>
                      </div>
                    </div>
                  </div>
                </b-tab-item>

                <b-tab-item v-if="this.$store.state.optionsAdvanced.advanced.showSceneSearchField && !displayingAlternateSource" label="Search fields">
                  <div class="block-tab-content block">
                    <div class="content is-small">
                      <div class="block" v-for="(field, idx) in searchfields" :key="idx">
                        <strong>{{ field.fieldName }} - </strong> {{ field.fieldValue }}
                      </div>
                    </div>
                  </div>
                </b-tab-item>

              </b-tabs>
            </div>

          </div>
        </div>
      </section>
      <div class="scene-id">
        {{ item.scene_id }}
        <span  v-if="this.$store.state.optionsAdvanced.advanced.showInternalSceneId">{{ $t('Internal ID') }}: {{item.id}}</span>
        <a v-if="this.$store.state.optionsAdvanced.advanced.showHSPApiLink" :href="`/heresphere/${item.id}`" target="_blank" rel="noreferrer" style="margin-left:0.5em">
          <img src="/ui/icons/heresphere_24.png" style="height:15px;"/>
        </a>
      </div>
    </div>
    <button class="modal-close is-large" aria-label="close" @click="close()"></button>
    <a class="prev" @click="prevScene" v-if="$store.getters['sceneList/prevScene'](item) !== null && !displayingAlternateSource"
       title="Keyboard shortcut: O">&#10094;</a>
    <a class="next" @click="nextScene" v-if="hasNextScene && !displayingAlternateSource"
       title="Keyboard shortcut: P">&#10095;</a>

    <!-- Fullscreen Gallery Modal -->
    <div v-if="fullscreenGallery" 
         ref="fullscreenGallery"
         class="fullscreen-gallery" 
         :class="{ 'is-zoomed': fullscreenZoomed }"
         @click.self="handleGalleryClick">
      <button class="fullscreen-close" @click="closeFullscreenGallery" title="Close (Esc)">
        <b-icon pack="mdi" icon="close" size="is-medium"></b-icon>
      </button>
      <button class="fullscreen-nav fullscreen-prev" @click.stop="fullscreenPrev" v-if="images.length > 1 && !fullscreenZoomed" title="Previous (←)">
        <b-icon pack="mdi" icon="chevron-left" size="is-large"></b-icon>
      </button>
      <button class="fullscreen-nav fullscreen-next" @click.stop="fullscreenNext" v-if="images.length > 1 && !fullscreenZoomed" title="Next (→)">
        <b-icon pack="mdi" icon="chevron-right" size="is-large"></b-icon>
      </button>
      <img 
        ref="fullscreenImage"
        :src="getFullscreenImageURL(images[fullscreenIndex].url)" 
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
import ky from 'ky'
import videojs from 'video.js'
import 'videojs-vr'
import 'videojs-hotkeys'
import { format, formatDistance, parseISO } from 'date-fns'
import prettyBytes from 'pretty-bytes'
import VueLoadImage from 'vue-load-image'
import { GlobalEvents } from 'vue-global-events'
import StarRating from 'vue-star-rating'
import FavouriteButton from '../../components/FavouriteButton'
import LinkStashdbButton from '../../components/LinkStashdbButton'
import WatchlistButton from '../../components/WatchlistButton'
import WishlistButton from '../../components/WishlistButton'
import WatchedButton from '../../components/WatchedButton'
import EditButton from '../../components/EditButton'
import RefreshButton from '../../components/RefreshButton'
import RescrapeButton from '../../components/RescrapeButton'
import TrailerlistButton from '../../components/TrailerlistButton'
import HiddenButton from '../../components/HiddenButton'

export default {
  name: 'SceneDetails',
  components: { VueLoadImage, GlobalEvents, StarRating, WatchlistButton, FavouriteButton, LinkStashdbButton, WishlistButton, WatchedButton, EditButton, RefreshButton, RescrapeButton, TrailerlistButton, HiddenButton },
  data () {
    return {
      index: 1,
      activeTab: 0,
      activeMedia: 0,
      player: {},
      vrPlugin: null,
      tagAct: '',
      cuepointName: '',
      cuepointRating: 0,
      cuepointPositionTags: ['', 'standing', 'sitting', 'laying', 'kneeling'],
      cuepointActTags: ['', 'handjob', 'blowjob', 'doggy', 'cowgirl', 'revcowgirl', 'missionary', 'titfuck', 'anal', 'cumshot', '69', 'facesit'],
      carouselSlide: 0,
      carouselImagesLoaded: {},
      carouselImageRetries: {},
      carouselImageKeys: {},
      carouselImageInfo: {},
      fullscreenGallery: false,
      fullscreenIndex: 0,
      fullscreenImageLoaded: false,
      fullscreenImageInfo: null,
      fullscreenZoomed: false,
      fullscreenCanZoom: false,
      vidPosition: null,
      skipForwardIntervals: [5, 10, 30, 60, 120, 300],
      skipBackIntervals: [-300, -120, -60, -30, -10, -5],
      lastSkipFowardInterval: 5,
      lastSkipBackInterval: -5,
      currentCuepointId: 0,
      maxTime: new Date(0, 0, 0, 5, 0, 0),
      cuepointSorting: [{ field: "is_hsp", order: "asc" },{ field: "time_start", order: "desc" }, {field: "track", order: "desc"}, {field: "time_end", order: "desc"}],
      trackInput: '',
      track: null,
      endTime: null,
      sortMultiple: true,
      castimages: [],
      searchfields: [],
      alternateSources: [],
      waitingForQuickFind: false,
      scenePalette: null,
    }
  },
  computed: {
    item () {
      const item = this.$store.state.overlay.details.scene
      if (this.$store.state.optionsWeb.web.tagSort === 'alphabetically') {
        item.tags.sort((a, b) => a.name < b.name ? -1 : 1)
      }
      let releasedate = parseISO(item.release_date)
      let imgs = item.cast.map((actor) => {
        let birthdate = parseISO(actor.birth_date)
        let label = actor.name
        if (birthdate.getFullYear() > 0) {
          let age = releasedate.getFullYear() - birthdate.getFullYear()
          if ((releasedate.getMonth() < birthdate.getMonth()) || (releasedate.getMonth() == birthdate.getMonth() && releasedate.getDate() < birthdate.getDate())) {
            age -= 1
          }
          label += `, ${age} in scene`
        }
        let img = actor.image_url
        if (img == "" ){
          img = "/ui/images/blank_female_profile.png"
        }
        if (actor.name.startsWith("aka:")) {
          img = ""
        }
        return {src: img, visible: false, actor_name: actor.name, actor_label: label, actor_id: actor.id};
      });

      this.castimages =  imgs.filter((img) => {
        return img.src !== '';
        });
      this.getSearchFields(item.id)
      return item
    },
    // Properties for gallery
    images () {
      if (this.item.images=="null") {
        return "[]"
      }
      return JSON.parse(this.item.images).filter(im => im && im.url)
    },
    // Tab: cuepoints
    sortedCuepoints () {
      if (this.item.cuepoints !== null) {
        for (let i = 0; i < this.item.cuepoints.length; i++) {
          this.item.cuepoints[i].is_hsp = this.item.cuepoints[i].track == null ? 0 : 1
        }
        const x = this.item.cuepoints.slice().sort((a, b) => {
          let compare = (a.is_hsp<b.is_hsp) ? -1 : (a.is_hsp>b.is_hsp) ? 1 : 0
          if (compare!=0) {
            return compare
          }
          compare = (a.time_start<b.time_start) ? -1 : (a.time_start>b.time_start) ? 1 : 0
          if (compare!=0) {
            return compare
          }
          compare = (a.track<b.track) ? -1 : (a.track>b.track) ? 1 : 0
          if (compare!=0) {
            return compare
          }
          return  (a.time_end<b.time_end) ? -1 : (a.time_end>b.time_end) ? 1 : 0
        })
        return x
      }
      return []
    },
    // Tab: files
    fileCount () {
      if (this.item.file !== null) {
        return this.item.file.length
      }
      return 0
    },
    filesByType () {
      if (this.item.file !== null) {
        return this.item.file.slice().sort((a) => (a.type === 'video') ? -1 : 1)
      }
      return []
    },
    // Tab: history
    historySessionsCount () {
      if (this.item.history !== null) {
        return this.item.history.length
      }
      return 0
    },
    historySessionsDuration () {
      if (this.item.history !== null) {
        let total = 0
        this.item.history.slice().map(i => {
          total = total + i.duration
          return 0
        })
        return total
      }
      return 0
    },
    showEdit () {
      return this.$store.state.overlay.edit.show
    },
    filteredCuepointPositionList () {
      // filter the list of positions based on what has been entered so far
      let list=this.cuepointActTags.concat(this.cuepointPositionTags)
      return list.filter((option) => {
        return option
          .toString()
          .toLowerCase()
          .trim()
          .indexOf(this.cuepointName.toString().toLowerCase()) >= 0
      })
    },
    displayingAlternateSource () {
      // displayingAlternateSource indicates we aren't displaying a real xbvr scene from the scenes table,
      //  so functions like watchlist, ratings, etc don't apply
      // we are displaying scene data serialized and saved in the external_references table
      if ( this.$store.state.overlay.details.altsrc != null) return true
      return false
    },
    async getAlternateSceneSources() {
      this.alternateSources = [];
      if (this.displayingAlternateSource) return 0
      try {
        const response = await ky.get('/api/scene/alternate_source/' + this.item.id).json();
        if (response==null){
          return 0
        }
        response.forEach(altsrc => {
          if (altsrc.external_source.startsWith("alternate scene ")) {
            this.alternateSources.push(altsrc)
          }
        });
        return this.alternateSources.length;
      } catch {
        return 0;
      }
    },
    changeDetailsTab() {      
      return this.$store.state.overlay.changeDetailsTab
    },
    quickFindOverlayState() {
      return this.$store.state.overlay.quickFind.show
    },
    showOpenInNewWindow () {
      return this.$store.state.optionsWeb.web.showOpenInNewWindow
    },
    accentColorTags () {
      return this.$store.state.optionsWeb.web.accentColorTags !== false
    },
    alternateSourcesWithTitles() {
      return this.alternateSources.map(altsrc => {
        const extdata = JSON.parse(altsrc.external_data);
        return {
          ...altsrc,
          title: extdata.scene?.title || 'No Title'
        };
      });
    },
    hasNextScene () {
      const next = this.$store.getters['sceneList/nextScene'](this.item)
      if (next !== null) return true
      const st = this.$store.state.sceneList
      return st.items.length < st.total
    },
    paletteStyle () {
      if (!this.scenePalette) return {}
      const p = this.scenePalette
      return {
        '--accent': p.accent,
        '--accent-bg': p.accentBg,
        '--accent-text': p.accentText,
        '--accent-light': p.accentLight,
        '--accent-faint': p.accentFaint
      }
    }
  },
  mounted () {
    this.setupPlayer()

    // load default cuepoint actions & positions from kv entry in the db
    ky.get('/api/options/cuepoints').json().then(data => {
      this.cuepointActTags = data.actions
      this.cuepointPositionTags = data.positions
      this.cuepointActTags.unshift("")
      this.cuepointPositionTags.unshift("")
      })    
},
watch:{
  quickFindOverlayState(newVal){
    if (newVal == true) {
      return
    }
    if (this.waitingForQuickFind){
      this.waitingForQuickFind = false
      if (this.$store.state.overlay.quickFind.selectedScene != null && this.$store.state.overlay.quickFind.selectedScene.id > 0) {
        this.$buefy.dialog.confirm({
          title: 'Relink scene',
          message: `Do you wish to link this scene to <strong>${this.$store.state.overlay.quickFind.selectedScene.title}</strong>`,
          type: 'is-info is-wide',
          hasIcon: true,
          id: 'heh',
          onConfirm: () => {
            this.handleRelinkExtRef()
          }
        })
      }
    }
  },
  changeDetailsTab(newVal){
    if (newVal == -1 ) {
      return
    }
    this.activeTab = newVal
    this.$store.commit('overlay/changeDetailsTab', { tab: -1 })
  },
  activeMedia(newVal) {
    // Auto-load first video when Player tab is opened (without auto-playing)
    if (newVal === 1 && !this.displayingAlternateSource) {
      const videoFiles = this.filesByType.filter(f => f.type === 'video')
      if (videoFiles.length > 0) {
        this.updatePlayer('/api/dms/file/' + videoFiles[0].id + '?dnt=true', (videoFiles[0].projection == 'flat' ? 'NONE' : '180'))
      }
    }
  },
  'item.scene_id'(newVal, oldVal) {
    // Scene changed via prev/next nav — fully rebuild the player so the
    // next time the Player tab is opened it loads cleanly for the new scene.
    if (!newVal || oldVal === undefined || newVal === oldVal) return
    if (this.player && typeof this.player.dispose === 'function') {
      try { this.player.dispose() } catch { /* already disposed */ }
    }
    this.player = {}
    this.vrPlugin = null
    this.activeMedia = 0
    this.carouselSlide = 0
    this.$nextTick(() => this.setupPlayer())
  },
  images: {
    handler () {
      // Reset loaded state when images change
      this.carouselImagesLoaded = {}
      this.carouselImageRetries = {}
      this.carouselImageKeys = {}
      this.scenePalette = null
    },
    immediate: false
  }
},
  methods: {
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
    onCarouselChange (index) {
      this.scrollToActiveIndicator(index)
      // Check if the new slide's image is already loaded
      this.$nextTick(() => {
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
      this.$nextTick(() => {
        if (this.$refs.carouselImages && this.$refs.carouselImages[index]) {
          const img = this.$refs.carouselImages[index]
          if (img.naturalWidth > 100) {
            // Image loaded successfully with reasonable size
            this.carouselImagesLoaded[index] = true
            this.carouselImageRetries[index] = 0
            // Fetch image info (format and size) for the badge
            this.fetchCarouselImageInfo(img.src, index)
            // Extract dominant color from first image
            if (index === 0 && !this.scenePalette) {
              this.extractSceneColor(img)
            }
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
    getFormatFromContentType (contentType) {
      if (contentType.includes('avif')) return 'AVIF'
      if (contentType.includes('webp')) return 'WebP'
      if (contentType.includes('png')) return 'PNG'
      if (contentType.includes('gif')) return 'GIF'
      if (contentType.includes('jpeg') || contentType.includes('jpg')) return 'JPEG'
      if (contentType.includes('svg')) return 'SVG'
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
    setupPlayer () {
      if (!this.$refs.player) return
      this.player = videojs(this.$refs.player, {
        aspectRatio: '1:1',
        fluid: true,
        loop: true
      })

      if (typeof this.player.hotkeys === 'function') this.player.hotkeys({
        alwaysCaptureHotkeys: true,
        volumeStep: 0.1,
        seekStep: 5,
        enableModifiersForNumbers: false,
        enableVolumeScroll: false,
        customKeys: {
          closeModal: {
            key: function (event) {
              return event.which === 27
            },
            handler: () => {
              this.close()
            }
          },
          zoomIn: {
            handler: () => {
              this.zoomHandler(true)
            }
          },
          zoomOut: {
            handler: () => {
              this.zoomHandler(false)
            }
          }
        }
      })

      const videoElement = this.player.el();
      videoElement.addEventListener('wheel', this.zoomHandlerWeb.bind(this))

      // Initialize VR once; projection is updated per-source via updatePlayer
      this.vrPlugin = this.player.vr({ projection: '180', forceCardboard: false })
    },

    zoomHandlerWeb(event) {
      event.preventDefault();
      this.zoomHandler(event.deltaY < 0)
    },

    zoomHandler(isZoomingIn) {
      const vr = this.player.vr()
      if (!vr || !vr.camera) return
      const minFov = 30
      const maxFov = 130
      let fov = vr.camera.fov + (isZoomingIn ? -1 : 1)

      if (fov < minFov) {
        fov = minFov
      }

      if (fov > maxFov) {
        fov = maxFov
      }

      vr.camera.fov = fov;
      vr.camera.updateProjectionMatrix()
    },
    updatePlayer (src, projection) {
      if (!this.vrPlugin) return  // player tab not mounted yet
      // setProjection queues the projection for the next loadedmetadata → init() cycle
      const vr = this.player.vr()
      if (vr) {
        vr.setProjection(projection)
      }

      if (src) {
        this.player.src({ src: src, type: 'video/mp4' })
      }
      this.player.poster(this.getImageURL(this.item.cover_url, '1200x'))
    },
    showCastScenes (actor) {
      this.$store.state.sceneList.filters.cast = actor
      this.$store.state.sceneList.filters.sites = []
      this.$store.state.sceneList.filters.tags = []
      this.$store.state.sceneList.filters.attributes = []
      this.$router.push({
        name: 'scenes',
        query: { q: this.$store.getters['sceneList/filterQueryParams'] }
      })
      this.close()
    },
    getCastScenesUrl(actor) {
      let newfilters = Object.assign({}, this.$store.state.sceneList.filters);
      newfilters.cast = actor;       
      newfilters.sites = []
      newfilters.tags = []
      newfilters.attributes = []
      return this.$router.resolve({
        name: 'scenes',
        query: { q: btoa(unescape(encodeURIComponent(JSON.stringify(newfilters)))) }
      }).href
    },
    showTagScenes (tag) {
      this.$store.state.sceneList.filters.cast = []
      this.$store.state.sceneList.filters.sites = []
      this.$store.state.sceneList.filters.tags = tag
      this.$store.state.sceneList.filters.attributes = []
      this.$router.push({
        name: 'scenes',
        query: { q: this.$store.getters['sceneList/filterQueryParams'] }
      })
      this.close()
    },
    openTagInNewWindow (tagName) {
      const url = this.getTagScenesUrl([tagName])
      window.open(url, '_blank')
    },
    getTagScenesUrl(tag) {
      let newfilters = Object.assign({}, this.$store.state.sceneList.filters);      
      newfilters.tags = tag;       
      newfilters.cast = []       
      newfilters.sites = []
      newfilters.attributes = []
      return this.$router.resolve({
        name: 'scenes',
        query: { q: btoa(unescape(encodeURIComponent(JSON.stringify(newfilters)))) }
      }).href
    },
    showSiteScenes (site) {
      this.$store.state.sceneList.filters.cast = []
      this.$store.state.sceneList.filters.sites = site
      this.$store.state.sceneList.filters.tags = []
      this.$store.state.sceneList.filters.attributes = []
      this.$router.push({
        name: 'scenes',
        query: { q: this.$store.getters['sceneList/filterQueryParams'] }
      })
      this.close()
    },
    getSiteScenesUrl(site) {
      let newfilters = Object.assign({}, this.$store.state.sceneList.filters);
      newfilters.sites = site;       
      newfilters.cast = []       
      newfilters.tags = []
      newfilters.attributes = []
      return this.$router.resolve({
        name: 'scenes',
        query: { q: btoa(unescape(encodeURIComponent(JSON.stringify(newfilters)))) }
      }).href
    },
    showActorDetail (actor_id) {
      ky.get('/api/actor/'+actor_id).json().then(data => {
        if (data.id != 0){
          this.$store.commit('overlay/showActorDetails', { actor: data })
          this.close()
        }
      })
    },
    playPreview () {
      this.activeMedia = 1
      this.updatePlayer('/api/dms/preview/' + this.item.scene_id, 'NONE')
      this.player.play()
    },
    playFile (file) {
      this.activeMedia = 1
      this.$nextTick(() => {
        if (!this.vrPlugin) this.setupPlayer()
        this.updatePlayer('/api/dms/file/' + file.id + '?dnt=true', (file.projection == 'flat' ? 'NONE' : '180'))
        if (this.vrPlugin) this.player.play()
      })
    },
    unmatchFile (file) {
      this.$buefy.dialog.confirm({
        title: 'Unmatch file',
        message: `You're about to unmatch the file <strong>${file.filename}</strong> from this scene. Afterwards, it can be matched again to this or any other scene.`,
        type: 'is-info is-wide',
        hasIcon: true,
        id: 'heh',
        onConfirm: () => {
          ky.post(`/api/files/unmatch`, {json:{file_id: file.id}}).json().then(data => {
            this.$store.commit('overlay/showDetails', { scene: data })
          })
        }
      })
    },
    removeFile (file) {
      this.$buefy.dialog.confirm({
        title: 'Remove file',
        message: `You're about to remove file <strong>${file.filename}</strong> from <strong>disk</strong>.`,
        type: 'is-danger',
        hasIcon: true,
        onConfirm: () => {
          ky.delete(`/api/files/file/${file.id}`).json().then(data => {
            this.$store.commit('overlay/showDetails', { scene: data })
          })
        }
      })
    },
    selectScript (file) {
      ky.post(`/api/scene/selectscript/${this.item.id}`, {
        json: {
          file_id: file.id,
        }
      }).json().then(data => {
          this.$store.commit('overlay/showDetails', { scene: data })
      })
    },
    getImageURL (u, size) {
      if (u==undefined) {
        return u
      }
      try {
        if (u.startsWith('http')) {
          if (u.search("%") == -1) {
            return '/img/' + size + '/' + encodeURI(u)
          } else {
            return '/img/' + size + '/' + encodeURI(decodeURI(u))
          }
        }
      } catch {
        return u
      }
      return u
    },
    extractSceneColor (imgEl) {
      try {
        const canvas = document.createElement('canvas')
        const size = 30
        canvas.width = size
        canvas.height = size
        const ctx = canvas.getContext('2d')
        ctx.drawImage(imgEl, 0, 0, size, size)
        const data = ctx.getImageData(0, 0, size, size).data

        // Collect all pixels as [r,g,b]
        const pixels = []
        for (let i = 0; i < data.length; i += 4) {
          pixels.push([data[i], data[i+1], data[i+2]])
        }

        // Simple k-means with 4 clusters
        const k = 4
        let centers = pixels.filter((_, i) => i % Math.floor(pixels.length / k) === 0).slice(0, k)
        for (let iter = 0; iter < 8; iter++) {
          const clusters = centers.map(() => [])
          for (const px of pixels) {
            let minD = Infinity, best = 0
            for (let c = 0; c < centers.length; c++) {
              const d = (px[0]-centers[c][0])**2 + (px[1]-centers[c][1])**2 + (px[2]-centers[c][2])**2
              if (d < minD) { minD = d; best = c }
            }
            clusters[best].push(px)
          }
          centers = clusters.map((cl, i) => {
            if (cl.length === 0) return centers[i]
            const avg = [0, 0, 0]
            for (const px of cl) { avg[0] += px[0]; avg[1] += px[1]; avg[2] += px[2] }
            return avg.map(v => Math.round(v / cl.length))
          })
        }

        // Sort by luminance
        centers.sort((a, b) => (0.299*a[0]+0.587*a[1]+0.114*a[2]) - (0.299*b[0]+0.587*b[1]+0.114*b[2]))

        const rgb = c => `rgb(${c[0]}, ${c[1]}, ${c[2]})`
        const pastel = (c, amt) => c.map(v => Math.round(v + (255 - v) * amt))

        // RGB <-> HSL helpers
        const rgbToHsl = (r, g, b) => {
          r /= 255; g /= 255; b /= 255
          const mx = Math.max(r, g, b), mn = Math.min(r, g, b)
          let h = 0, s = 0, l = (mx + mn) / 2
          if (mx !== mn) {
            const d = mx - mn
            s = l > 0.5 ? d / (2 - mx - mn) : d / (mx + mn)
            if (mx === r) h = ((g - b) / d + (g < b ? 6 : 0)) / 6
            else if (mx === g) h = ((b - r) / d + 2) / 6
            else h = ((r - g) / d + 4) / 6
          }
          return [h, s, l]
        }
        const hslToRgb = (h, s, l) => {
          if (s === 0) { const v = Math.round(l * 255); return [v, v, v] }
          const hue2rgb = (p, q, t) => {
            if (t < 0) t += 1; if (t > 1) t -= 1
            if (t < 1/6) return p + (q - p) * 6 * t
            if (t < 1/2) return q
            if (t < 2/3) return p + (q - p) * (2/3 - t) * 6
            return p
          }
          const q = l < 0.5 ? l * (1 + s) : l + s - l * s
          const p = 2 * l - q
          return [Math.round(hue2rgb(p, q, h + 1/3) * 255), Math.round(hue2rgb(p, q, h) * 255), Math.round(hue2rgb(p, q, h - 1/3) * 255)]
        }

        // Pick the most saturated cluster as accent
        let bestSat = -1, accent = centers[1]
        for (const c of centers) {
          const mx = Math.max(...c), mn = Math.min(...c)
          const sat = mx > 0 ? (mx - mn) / mx : 0
          if (sat > bestSat) { bestSat = sat; accent = c }
        }

        // Convert to HSL and normalize: ensure minimum saturation, target nice lightness
        let [h, s, l] = rgbToHsl(accent[0], accent[1], accent[2])
        // Gently boost saturation only if it's weak (< 0.5), cap at 0.7
        if (s < 0.5) s = s + (0.5 - s) * 0.6
        s = Math.min(s, 0.85)
        // Clamp lightness to a pleasant range (not too dark, not too bright)
        l = Math.max(0.35, Math.min(0.55, l))
        accent = hslToRgb(h, s, l)

        // Compute luminance for contrast decisions
        const luminance = c => {
          const toLinear = x => { const v = x/255; return v <= 0.03928 ? v/12.92 : ((v+0.055)/1.055)**2.4 }
          return 0.2126*toLinear(c[0]) + 0.7152*toLinear(c[1]) + 0.0722*toLinear(c[2])
        }
        const lum = luminance(accent)
        const accentText = lum > 0.179 ? '#1a1a1a' : '#ffffff'

        // Background variant: slightly darker for solid fills
        const accentBg = hslToRgb(h, Math.min(s * 1.1, 0.9), Math.max(l - 0.08, 0.25))

        this.scenePalette = {
          accent: rgb(accent),
          accentBg: rgb(accentBg),
          accentText: accentText,
          accentLight: rgb(pastel(accentBg, 0.35)),
          accentFaint: rgb(pastel(accentBg, 0.88))
        }
      } catch (e) {
        console.warn('Color extraction failed:', e.message)
      }
    },
    getIndicatorURL (idx) {
      if (this.images[idx] !== undefined) {
        return this.getImageURL(this.images[idx].url, 'x40')
      } else {
        return '/ui/images/blank.png'
      }
    },
    getHeatmapURL (fileId) {
      return `/api/dms/heatmap/${fileId}`
    },
    playCuepoint (cuepoint) {
      // populate the cuepoint edit fields
      this.vidPosition = new Date(0, 0, 0, 0, 0, 0, cuepoint.time_start*1000)
      this.endTime = new Date(0, 0, 0, 0, 0, 0, cuepoint.time_end*1000)
      this.currentCuepointId = cuepoint.id
      this.cuepointRating = cuepoint.rating
      if (cuepoint.name.indexOf('-') > 0) {
        this.cuepointName = cuepoint.name.substr(0, cuepoint.name.indexOf('-'))
        this.tagAct = cuepoint.name.substr(cuepoint.name.indexOf('-') + 1)
      } else {
        this.tagAct = cuepoint.name
        this.cuepointName = ''
      }
      // now mow the player position
      this.player.currentTime(cuepoint.time_start)
      this.player.play()
    },
    updateCuepoint (editCuepoint) {
      if (this.disableSaveButtons()) return
      // if edit choosen, delete existing cuepoint before add
      if (editCuepoint && this.currentCuepointId > 0) {
        this.deleteCuepoint(this.currentCuepointId)
      }
      let name =  this.cuepointName
      let pos = this.player.currentTime()
      let endpos=null
      this.track=parseInt(this.track)
      if (this.vidPosition != null) {
        pos = (this.vidPosition.getMilliseconds() / 1000) + this.vidPosition.getSeconds() + (this.vidPosition.getMinutes() * 60) + (this.vidPosition.getHours() * 60 * 60)
      }
      if (this.endTime != null) {
        endpos = (this.endTime.getMilliseconds() / 1000) + this.endTime.getSeconds() + (this.endTime.getMinutes() * 60) + (this.endTime.getHours() * 60 * 60)
      }
      this.currentCuepointId = 0

      ky.post(`/api/scene/${this.item.id}/cuepoint`, {
        json: {
          track: this.track,
          name: name,
          time_start: pos,
          time_end: endpos,
          rating: this.cuepointRating
        }
      }).json().then(data => {
        this.vidPosition = null
        this.endTime = null
        this.cuepointName=''
        this.track = null
        this.$store.commit('sceneList/updateScene', data)
        this.$store.commit('overlay/showDetails', { scene: data })
      })
    },
    deleteCuepoint (cuepointid) {
      ky.delete(`/api/scene/${this.item.id}/cuepoint/${cuepointid}`)
        .json().then(data => {
          this.$store.commit('sceneList/updateScene', data)
          this.$store.commit('overlay/showDetails', { scene: data })
        })
    },
    close () {
      if (!this.displayingAlternateSource && this.player && typeof this.player.dispose === 'function') this.player.dispose()
      this.$store.commit('overlay/hideDetails')
    },
    humanizeSeconds (seconds) {
      return new Date(seconds * 1000).toISOString().substr(11, 8)
    },
    humanizeSeconds1DP (seconds) {
      return new Date(seconds * 1000).toISOString().substr(11, 10)
    },
    setRating (val) {
      ky.post(`/api/scene/rate/${this.item.id}`, { json: { rating: val } })

      const updatedScene = Object.assign({}, this.item)
      updatedScene.star_rating = val
      this.item.star_rating = val
      this.$store.commit('sceneList/updateScene', updatedScene)
    },
    onZeroKey (e) {
      if (e.key === '0') this.setRating(0)
    },
    async nextScene () {
      if (this.displayingAlternateSource) return
      let data = this.$store.getters['sceneList/nextScene'](this.item)
      if (data === null) {
        const st = this.$store.state.sceneList
        if (st.items.length < st.total && !st.isLoading) {
          await this.$store.dispatch('sceneList/load', { offset: st.offset })
          data = this.$store.getters['sceneList/nextScene'](this.item)
        }
      }
      if (data !== null) {
        this.$store.commit('overlay/showDetails', { scene: data })
      }
    },
    prevScene () {
      const data = this.$store.getters['sceneList/prevScene'](this.item)
      if (data !== null && !this.displayingAlternateSource) {
        this.$store.commit('overlay/showDetails', { scene: data })
      }
    },
    playerStepBack (interval) {
      const wasPlaying = !this.player.paused()
      if (wasPlaying) {
        this.player.pause()
      }
      let seekTime = this.player.currentTime() + interval
      if (seekTime <= 0) {
        seekTime = 0
      }
      this.player.currentTime(seekTime)
      if (wasPlaying) {
        this.player.play()
      }
      this.lastSkipBackInterval = interval
    },
    playerStepForward (interval) {
      const duration = this.player.duration()
      const wasPlaying = !this.player.paused()
      if (wasPlaying) {
        this.player.pause()
      }
      let seekTime = this.player.currentTime() + interval
      if (seekTime >= duration) {
        seekTime = wasPlaying ? duration - 0.001 : duration
      }
      this.player.currentTime(seekTime)
      if (wasPlaying) {
        this.player.play()
      }
      this.lastSkipFowardInterval = interval
    },
    setCuepointName (param) {
      if (this.activeTab === 1) {
        if (this.cuepointName=='') {
          this.cuepointName = param.toString()
        }else{
          this.cuepointName = this.cuepointName+'-'+param.toString()
        }
      }
    },
    toggleGallery () {
      if (this.fullscreenGallery) {
        // G key opens/closes fullscreen when gallery tab is active
        this.closeFullscreenGallery()
      } else if (this.activeMedia == 0) {
        // Open fullscreen gallery
        this.openFullscreenGallery(this.carouselSlide)
      } else {
        // Switch to gallery tab
        this.activeMedia = 0
      }
    },
    handleLeftArrow () {
      if (this.fullscreenGallery) {
        this.fullscreenPrev()
      } else if (this.activeMedia === 0) {
        this.carouselSlide = this.carouselSlide - 1
      } else {
        this.playerStepBack(this.lastSkipBackInterval)
      }
    },
    handleRightArrow () {
      if (this.fullscreenGallery) {
        this.fullscreenNext()
      } else if (this.activeMedia === 0) {
        this.carouselSlide = this.carouselSlide + 1
      } else {
        this.playerStepForward(this.lastSkipFowardInterval)
      }
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
    timeFormatter(time) {
       return new Intl.DateTimeFormat('en', { hourCycle: 'h23', hour: "2-digit", minute: "2-digit", second: "2-digit", fractionalSecondDigits: 1 }).format(time)
    },
    timeParser(inputString) {
      let items = inputString.split(":")
      return new Date(0, 0, 0, items[0],items[1], 0, items[2]*1000)
    },
    cuepointSelected(cuepoint) {
      // populate the cuepoint edit fields
      this.vidPosition = new Date(0, 0, 0, 0, 0, 0, cuepoint.time_start*1000)
      this.endTime = new Date(0, 0, 0, 0, 0, 0, cuepoint.time_end*1000)
      this.currentCuepointId = cuepoint.id
      this.cuepointName = cuepoint.name
      this.track=cuepoint.track
      this.cuepointRating=cuepoint.rating
      // now mow the player position
      this.player.currentTime(cuepoint.time_start)
      this.player.play()
    },
    disableSaveButtons() {
      if (this.track!=null && this.track!="" && (isNaN(this.endTime) || this.endTime==null)) return true
      if ((this.track==null || this.track==="") && !isNaN(this.endTime) && this.endTime!=null) return true
      return false
    },
    disableSaveMsg() {
      if (this.track!=null && this.track!="" && (isNaN(this.endTime) || this.endTime==null)) return "Specify a End Time"
      if ((this.track==null || this.track==="") && !isNaN(this.endTime) && this.endTime!=null) return "End Time is only valid for HSP Cuepoints"
      return ""
    },
    setCuepointRating (row) {
      this.cuepointSelected(row)
      this.updateCuepoint(true)
    },
    clearCuepointRating (row) {
      row.rating=0
      this.cuepointSelected(row)
      this.updateCuepoint(true)
    },
    showTooltip(idx) {
      this.castimages[idx].visible = true;
    },
    hideTooltip(idx) {
      this.castimages[idx].visible = false;
    },
    getSearchFields(id) {
      // load search fields
      this.searchfields = []      
      if (this.$store.state.optionsAdvanced.advanced.showSceneSearchField && !this.displayingAlternateSource) {
        ky.get('/api/scene/searchfields', {
          searchParams: {
            q: id
          },
          }).json().then(data => {
            this.searchfields = data
          })
      }
    },
    showExtRefScene (altsrc) {      
      const extdata = JSON.parse(altsrc.external_data);      
      if (extdata.scene.cast == null) 
      {
        extdata.scene.cast = []
      }
      this.$store.commit('overlay/showDetails', { scene: extdata.scene, altsrc: altsrc, prevscene: this.item, query_for_altsrc: extdata.query })
      this.activeTab = 0      
    },
    searchAlternateSourceScene() {
      // search for a new scene to link to the alternate source scene
      const  q = this.$store.state.overlay.details.query_for_altsrc == "" ? this.item.title : this.$store.state.overlay.details.query_for_altsrc      
      this.$store.commit('overlay/showQuickFind', { searchString:  q, displaySelectedScene: false })
      this.waitingForQuickFind = true
    }, 
    async handleRelinkExtRef() {
      const response = await ky.post(`/api/extref/edit_link`, {
        json: {
          external_source: this.$store.state.overlay.details.altsrc.external_source,
          external_id: this.$store.state.overlay.details.altsrc.external_id,
          internal_table: "scenes",
          internal_db_id: this.$store.state.overlay.quickFind.selectedScene.id,
          internal_name_id: this.$store.state.overlay.quickFind.selectedScene.scene_id,
          match_type: 99999
        }
      });
      if (response.status === 200) {
        this.$store.state.overlay.details.prevscene = this.$store.state.overlay.quickFind.selectedScene;
        this.$buefy.toast.open({ message: `The scene was sucessfully relinked to a new Scene`, type: 'is-primary', duration: 3000 });
      }
    },
    async scrapeScene() {
      this.$buefy.dialog.confirm({
        title: 'Scrape & Create Scene',
        message: `Do you wish to create a seperate XBVR scene from this linked scene <strong>${this.$store.state.overlay.details.altsrc.url}</strong>`,
        type: 'is-info is-wide',
        hasIcon: true,
        id: 'heh',
        onConfirm: () => {
          const url = this.$store.state.overlay.details.altsrc.url
          this.$store.state.overlay.details.altsrc = null
          this.$store.commit('overlay/hideDetails')
          // call the options screen passing the url in state   
          this.$store.commit('optionsSceneCreate/setScrapeScene', url )
          this.$store.commit('optionsSceneCreate/showSceneCreate', true )
          this.$router.push({ path: '/options'})
        }
      })

    },
    async refreshExtRef() {
      this.$buefy.dialog.confirm({
        title: 'Continue?',
        message: `This will remove the scene, rescrape the site to relink it to an XBVR scene`,
        type: 'is-info is-wide',
        hasIcon: true,
        id: 'heh',
        onConfirm: () => {          
          this.handleRefreshExtRef()
        }
      })
    },
    async handleRefreshExtRef() {
      const response = await ky.delete(`/api/extref/delete_extref`, {
        json: {
          external_source: this.$store.state.overlay.details.altsrc.external_source,
          external_id: this.$store.state.overlay.details.altsrc.external_id,
        }
      });
      if (response.status === 200) {
        this.$store.state.overlay.details.prevscene = this.$store.state.overlay.quickFind.selectedScene;
        this.$buefy.toast.open({ message: `The scene was removed, ready to rescan`, type: 'is-primary', duration: 3000 });
      }
    },
    flagExtRefDeleted() {
      this.$buefy.dialog.confirm({
        title: 'Continue?',
        message: `This will unlink the scene and prevent it from relinking to any scene. This cannot be undone`,
        type: 'is-danger is-wide',
        hasIcon: true,
        id: 'heh',
        onConfirm: () => {          
          this.handleFlagExtRefDeleted()
        },
      })    
    },    
    async handleFlagExtRefDeleted() {
      const response = await ky.post(`/api/extref/edit_link`, {
        json: {
          external_source: this.$store.state.overlay.details.altsrc.external_source,
          external_id: this.$store.state.overlay.details.altsrc.external_id,
          internal_table: "scenes",
          internal_db_id: 0,
          internal_name_id: "deleted",
          match_type: -1
        }
      });
      if (response.status === 200) {
        this.$store.state.overlay.details.prevscene = this.$store.state.overlay.quickFind.selectedScene;
        this.$buefy.toast.open({ message: `The scene was unlinked and will not be relinked to any scene`, type: 'is-primary', duration: 3000 });
      }
    },    
    format,
    parseISO,
    prettyBytes,
    formatDistance
  }
}
</script>

<style lang="less" scoped>
.carousel-image-container {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
}

.carousel-image {
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

.bbox {
  flex: 1 0 calc(25%);
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  padding: 0;
  line-height: 0;
}

.is-1by1 {
  padding-top: calc(100% - 40px - 1em) !important;
  background: transparent !important;
}




.player-wrapper {
  width: 100%;
  background: #000;
  margin-bottom: 8px;
}

.skip-controls {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.75em;
  margin-top: 6px;
}

.skip-group {
  display: flex;
  align-items: center;
  gap: 2px;
}

.player-wrapper .video-js {
  width: 100%;
}

.video-js {
  margin: 0 auto;
}

:deep(.video-js .vjs-big-play-button) {
  top: 50% !important;
  left: 50% !important;
  margin-top: 0 !important;
  margin-left: 0 !important;
  transform: translate(-50%, -50%) !important;
}

.modal-card {
  width: 85%;
  max-height: calc(100vh - 40px);
  margin: 0 auto;
  transition: background 1.2s ease;
}

:deep(.modal-card-body) {
  overflow-y: auto;
  max-height: calc(100vh - 40px);
  scrollbar-width: none;
  border-top: 3px solid var(--accent, #7957d5);
  transition: border-color 0.8s ease;
}

:deep(.modal-card-body)::-webkit-scrollbar {
  display: none;
}

.missing {
  opacity: 0.6;
}

.block-tab-content {
  flex: 1 1 auto;
}

/* Detail Header */
.detail-header {
  margin-bottom: 12px;
}

.detail-title {
  font-size: 1.2rem;
  font-weight: 600;
  margin-bottom: 4px !important;
  line-height: 1.3;
}

.detail-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 0.85rem;
  color: #666;
  margin-bottom: 8px;
}

.detail-meta-left {
  display: flex;
  align-items: center;
  gap: 8px;
}

.detail-meta-left a {
  color: #485fc7;
  text-decoration: none;
}

.detail-meta-left a:hover {
  text-decoration: underline;
}

.site-link {
  display: inline-flex;
  align-items: center;
  gap: 3px;
}

.site-external {
  color: #999;
  line-height: 0;
  transition: color 0.15s;
}

.site-external:hover {
  color: #485fc7;
}

.members-link {
  display: inline-flex;
  align-items: center;
}

.detail-duration {
  color: #999;
}

.detail-date {
  color: #999;
  font-size: 0.8rem;
  flex-shrink: 0;
}

.detail-actions-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.detail-rating {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
}

.rating-reset {
  cursor: pointer;
  opacity: 0.4;
  transition: opacity 0.15s;
}

.rating-reset:hover {
  opacity: 1;
}

/* Cast Section */
.cast-section {
  margin-bottom: 10px;
}

.cast-row {
  display: flex;
  gap: 6px;
  overflow: visible;
  padding-bottom: 4px;
  flex-wrap: wrap;
}

.cast-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  flex-shrink: 0;
  cursor: pointer;
  position: relative;
  line-height: 0;
}

.cast-item :deep(> div) {
  line-height: 0;
}


.cast-thumb {
  height: clamp(70px, 6vw, 90px);
  border-radius: 4px;
  object-fit: cover;
  border: 2px solid transparent;
  transition: border-color 0.15s;
  display: block;
}

.cast-item:hover .cast-thumb {
  border-color: #7957d5;
}

/* Tags Section */
.tags-section {
  margin-bottom: 10px;
}

.tag-group {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

.tag-pill {
  display: inline-flex !important;
  align-items: center !important;
  gap: 4px;
  cursor: pointer;
}

.tag-external {
  opacity: 0.6;
  margin-left: 0 !important;
  margin-right: -2px !important;
  transition: opacity 0.15s;
}
.tag-external:hover {
  opacity: 1;
}

.tags-section :deep(.tag) {
  margin-bottom: 0 !important;
  font-size: 0.75rem !important;
  padding: 0 8px !important;
  height: 24px !important;
  border-radius: 4px !important;
}

/* Description */
.description-text {
  font-size: 0.85rem;
  line-height: 1.5;
  color: #444;
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
.altsrc-image-wrapper {
  display: inline-block;
  margin-left: 5px;  
}

/* Carousel background — match modal */
:deep(.carousel) {
  background: transparent !important;
}
:deep(.carousel .carousel-items),
:deep(.carousel-item) {
  background: transparent !important;
}

/* Fullscreen Gallery Styles */
.carousel-wrapper {
  position: relative;
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

/* Detail action buttons — match SceneCard size */
.detail-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  align-items: center;
}

.detail-actions .button,
.detail-actions :deep(.button) {
  width: 28px !important;
  height: 28px !important;
  min-width: 28px !important;
  min-height: 28px !important;
  padding: 0 !important;
  margin: 0 !important;
  border-radius: 5px !important;
  display: inline-flex !important;
  align-items: center !important;
  justify-content: center !important;
  line-height: 1 !important;
}

.detail-actions :deep(.button .icon) {
  margin: 0 !important;
  width: 16px !important;
  height: 16px !important;
  display: inline-flex !important;
  align-items: center !important;
  justify-content: center !important;
}

.detail-actions :deep(.button .icon .mdi) {
  font-size: 16px !important;
  line-height: 1 !important;
}

.detail-actions :deep(.button span:not(.icon)) {
  display: none !important;
}

/* Instant CSS tooltips for all buttons with data-tooltip */
[data-tooltip] {
  position: relative;
}

[data-tooltip]::after {
  content: attr(data-tooltip);
  position: absolute;
  bottom: calc(100% + 6px);
  right: 0;
  padding: 4px 8px;
  background: rgba(0,0,0,0.85);
  color: #fff;
  font-size: 0.75rem;
  font-weight: 400;
  line-height: normal;
  white-space: nowrap;
  border-radius: 4px;
  pointer-events: none;
  opacity: 0;
  transition: opacity 0.1s ease;
  z-index: 30;
}

[data-tooltip]:hover::after {
  opacity: 1;
}

/* ── Detail View Dark Mode ── */
html[data-theme="dark"] .detail-title { color: #e8e8ec; }
html[data-theme="dark"] .detail-meta,
html[data-theme="dark"] .detail-duration,
html[data-theme="dark"] .detail-date { color: #a0a0b0; }
html[data-theme="dark"] .detail-meta-left a { color: #8ab4f8; }
html[data-theme="dark"] .site-external { color: #888; }
html[data-theme="dark"] .description-text { color: #c8c8d0; }
html[data-theme="dark"] .pathDetails { color: #999; }
html[data-theme="dark"] .scene-id { color: #666; }
html[data-theme="dark"] .rating-reset { color: #aaa; }
html[data-theme="dark"] .videosize { color: #b0b0c0; font-weight: 500; }
html[data-theme="dark"] .prev,
html[data-theme="dark"] .next { color: #ddd; }
html[data-theme="dark"] .members-link { color: #888; }
html[data-theme="dark"] .cast-thumb { border-color: #2a2a35; }
html[data-theme="dark"] .is-divider { background-color: #2a2a35 !important; }
html[data-theme="dark"] .heatmapFunscript img { border-color: #3a3a48; }

/* ── Scene Accent Color ── */
.detail-meta-left a {
  color: var(--accent, #485fc7);
}
.detail-meta-left a:hover {
  color: var(--accent, #485fc7);
  filter: brightness(0.85);
}
.site-external:hover {
  color: var(--accent, #485fc7);
}
.cast-item:hover .cast-thumb {
  border-color: var(--accent, #7957d5);
}
.tags-section:not(.no-accent) :deep(.tag.is-info) {
  background-color: var(--accent-bg, #485fc7) !important;
  color: var(--accent-text, #fff) !important;
}
.tags-section:not(.no-accent) :deep(.tag.is-info:hover) {
  filter: brightness(1.15);
}
:deep(.tabs li.is-active a) {
  color: var(--accent, #7957d5) !important;
  border-bottom-color: var(--accent, #7957d5) !important;
}
.detail-actions :deep(.button.is-primary) {
  border-color: var(--accent, #7957d5) !important;
  color: var(--accent, #7957d5) !important;
}
.detail-actions :deep(.button.is-primary:hover) {
  background-color: var(--accent-bg, #7957d5) !important;
  color: var(--accent-text, #fff) !important;
}
</style>
