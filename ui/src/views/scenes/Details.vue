<template>
  <SplitModal :body-style="paletteStyle" @close="close">
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
      @keydown.space.prevent="togglePlay"
      @keydown.k="togglePlay"
      @keydown="onZeroKey"
    />

    <template #media>
            <b-tabs v-model="activeMedia" position="is-centered" :animated="false" class="media-tabs">

              <b-tab-item :label="$t('Gallery')">
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
                        <button class="fullscreen-btn" @click.stop="openFullscreenGallery(i)" :title="$t('Fullscreen (G)')">
                          <b-icon pack="mdi" icon="fullscreen" size="is-small"></b-icon>
                        </button>
                      </div>
                    </b-carousel-item>
                    <template #indicators="props">
                        <span class="al image" style="width:max-content;">
                          <vue-load-image>
                            <template #image><img :src="getIndicatorURL(props.i)" style="height:40px;"/></template>
                            <template #preloader><img src="/ui/images/blank.png" style="height:40px;"/></template>
                            <template #error><img :src="originalAt(props.i) || '/ui/images/blank.png'" @error="$event.target.src = '/ui/images/blank.png'" style="height:40px;"/></template>
                          </vue-load-image>
                        </span>
                    </template>
                  </b-carousel>
                </div>
              </b-tab-item>

              <b-tab-item :label="$t('Player')" v-if="!displayingAlternateSource && fileCount > 0">
                <!-- keyed per scene: each scene gets a fresh player -->
                <div class="player-wrapper" :key="'player-' + item.scene_id">
                  <VrPlayer ref="player" aspect-ratio="16 / 9" :script="playerScriptURL" :scripts="playerScripts" @select-script="id => selectScript({ id })"/>
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
                <div class="shortcut-legend">
                  <span v-for="k in playerShortcuts" :key="k.keys"><kbd>{{ k.keys }}</kbd> {{ k.label }}</span>
                </div>
             </b-tab-item>

            </b-tabs>

    </template>

    <template #details>

            <div class="detail-header">
              <h3 class="detail-title">
                <span v-if="item.title">{{ item.title }}</span>
                <span v-else class="missing">{{ $t('(no title)') }}</span>
              </h3>

              <div class="detail-meta">
                <span class="detail-meta-left">
                  <span class="site-link">
                    <a @click.stop="showSiteScenes([item.site])">{{ item.site }}</a>
                    <a :href="item.scene_url" target="_blank" rel="noreferrer" @click.stop class="site-external"><b-icon pack="mdi" icon="open-in-new" custom-size="mdi-12px"/></a>
                  </span>
                  <a v-if="item.members_url != ''" :href="item.members_url" target="_blank" rel="noreferrer" @click.stop class="members-link">
                    <b-icon pack="mdi" icon="link-lock" custom-size="mdi-14px"/>
                  </a>
                  <span v-if="item.duration" class="detail-duration">{{ $t('{n} min', { n: item.duration }) }}</span>
                </span>
                <span class="detail-date">
                  {{ format(parseISO(item.release_date), "yyyy-MM-dd") }}
                </span>
              </div>

              <div class="detail-actions-row">
                <div class="detail-rating" v-if="!displayingAlternateSource">
                  <star-rating :key="item.id" v-model="item.star_rating" :rating="item.star_rating" @rating-selected="setRating"
                               :increment="0.5" :star-size="18" :show-rating="false" />
                  <b-icon pack="mdi" icon="autorenew" size="is-small" @click="setRating(0)" class="rating-reset" :data-tooltip="$t('Reset rating')"/>
                </div>
                <div v-if="displayingAlternateSource" class="detail-rating">
                  <strong>{{ $t('Linked scene, Not an XBVR Scene') }}</strong>
                </div>
                <div class="detail-actions">
                  <a class="button is-primary is-outlined is-small" @click="searchAlternateSourceScene()" :data-tooltip="$t('Search for a different scene')" v-if="displayingAlternateSource">
                    <b-icon pack="mdi" icon="movie-search-outline" size="is-small"/>
                  </a>
                  <a class="button is-primary is-outlined is-small" @click="scrapeScene()" :data-tooltip="$t('Scrape and create an XBVR scene')" v-if="displayingAlternateSource">
                    <b-icon pack="mdi" icon="plus" size="is-medium"/>
                  </a>
                  <a class="button is-primary is-outlined is-small" @click="refreshExtRef()" :data-tooltip="$t('Refresh scene data and relink')" v-if="displayingAlternateSource">
                    <b-icon pack="mdi" icon="refresh" size="is-small"/>
                  </a>
                  <a class="button is-danger is-outlined is-small" @click="flagExtRefDeleted()" :data-tooltip="$t('Unlink scene permanently')" v-if="displayingAlternateSource">
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
                  <div v-for="(altsrc, idx) in alternateSourcesWithTitles" :key="idx" class="altsrc-image-wrapper" @click="showExtRefScene(altsrc)">
                    <b-tooltip type="is-light" :label="altsrc.title" :delay="0" append-to-body>
                      <vue-load-image>
                        <template #image><img :src="getImageURL(altsrc.site_icon)" :alt="$t('Image')" width="28px"/></template>
                        <template #error><b-icon pack="mdi" icon="link" size="is-small"/></template>
                      </vue-load-image>
                    </b-tooltip>
                  </div>
                </div>
              </div>
            </div>

            <div class="cast-section" v-if="!displayingAlternateSource && castimages.length > 0">
              <div class="cast-row">
                <div v-for="(image, idx) in castimages" :key="idx" class="cast-item"
                     :data-tooltip="image.actor_label"
                     @click='showActorDetail([image.actor_id])'>
                  <vue-load-image>
                    <template #image><img :src="getImageURL(image.src)" class="cast-thumb"/></template>
                    <template #preloader><img src="/ui/images/actor_placeholder.svg" class="cast-thumb"/></template>
                    <template #error><img src="/ui/images/actor_placeholder.svg" class="cast-thumb"/></template>
                  </vue-load-image>
                  <span class="cast-caption">
                    <span class="cast-name">{{ image.first_name }}</span>
                    <span v-if="castHasAge" class="cast-age">{{ image.age ?? '' }}</span>
                  </span>
                </div>
              </div>
            </div>

            <div class="tags-section" :class="{ 'no-accent': !accentColorTags }" v-if="item.tags && item.tags.length > 0">
              <div class="tag-group" :ref="watchTagGroup" :class="{ 'is-collapsed': tagsOverflow && !tagsExpanded }">
                <a v-for="(tag, idx) in item.tags" :key="'tag' + idx"
                   @click='showTagScenes([tag.name])' class="tag is-info is-small tag-pill">
                  <span>{{ tag.name }} ({{ tag.count }})</span>
                  <b-icon v-if="showOpenInNewWindow" pack="mdi" icon="open-in-new" size="is-small"
                    class="tag-external" @click.stop="openTagInNewWindow(tag.name)"/>
                </a>
              </div>
              <button v-if="tagsOverflow" type="button" class="tags-toggle" @click="tagsExpanded = !tagsExpanded">
                <b-icon pack="mdi" :icon="tagsExpanded ? 'chevron-up' : 'chevron-down'" size="is-small"/>
                <span>{{ tagsExpanded ? $t('Show less') : $t('+{n} more', { n: hiddenTags }) }}</span>
              </button>
            </div>

            <div class="block-opts block">
              <b-tabs v-model="activeTab" :animated="false">

                <b-tab-item :label="$t('Files ({n})', { n: fileCount })" v-if="!displayingAlternateSource">
                  <div class="block-tab-content block">
                    <p class="tab-empty" v-if="!filesByType.length">{{ $t('No files for this scene') }}</p>
                    <div class="content media is-small" v-for="(f, idx) in filesByType" :key="idx">
                      <div class="media-left">
                        <button rounded class="button is-success is-small" @click='playFile(f)'
                                v-show="f.type === 'video'">
                          <b-icon pack="mdi" icon="play" size="is-small"></b-icon>
                        </button>
                        <b-tooltip :label="$t('Select this script for export')" position="is-right">
                        <button rounded class="button is-info is-small is-outlined" @click='selectScript(f)'
                          v-show="f.type === 'script'" v-bind:class="{ 'is-success': f.is_selected_script, 'is-info' :!f.is_selected_script }">
                          <b-icon pack="mdi" :icon="isAiScript(f.filename) ? 'robot-outline' : 'pulse'" :title="isAiScript(f.filename) ? $t('AI script') : $t('Script')"></b-icon>
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
                      <div class="media-content file-content">
                        <strong class="scene-file-name" :title="f.filename">{{ f.filename }}</strong>
                        <small>
                          <span class="pathDetails scene-file-path" :title="f.path">{{ f.path }}</span>
                          {{ prettyBytes(f.size) }}<span v-if="f.type === 'video'"> ({{ prettyBytes(f.video_bitrate, { bits: true })  }}/s)</span>,
                          <span v-if="f.type === 'video'"><span class="videosize">{{ f.video_width }}x{{ f.video_height }} {{ f.video_codec_name }}</span>, {{ f.projection }},&nbsp;</span>
                          <span v-if="f.duration > 1">{{ humanizeSeconds(f.duration) }},</span>
                          {{ format(parseISO(f.created_time), "yyyy-MM-dd") }}
                        </small>
                        <div v-if="f.type === 'script' && f.has_heatmap" class="heatmapFunscript">
                          <img :src="getHeatmapURL(f.id)"/>
                        </div>
                        <div v-if="f.type === 'script' && f.script_actions > 0" class="script-props">
                          <span v-if="scriptLicense(f.script_license) === 'paid'" class="sp-badge is-paid" :title="$t('Paid script')">
                            <b-icon pack="mdi" icon="cash" custom-size="mdi-14px"/>{{ $t('Paid') }}
                          </span>
                          <span v-else-if="scriptLicense(f.script_license) === 'free'" class="sp-badge is-free" :title="$t('Free script')">{{ $t('Free') }}</span>
                          <span v-if="scriptGenerator(f.script_description)" class="sp-badge is-gen" :title="f.script_description">
                            {{ scriptGenerator(f.script_description).name }} v{{ scriptGenerator(f.script_description).version }}
                          </span>
                          <span v-if="f.script_creator" class="sp-badge" :title="$t('Creator')">
                            <b-icon pack="mdi" icon="account-outline" custom-size="mdi-14px"/>{{ f.script_creator }}
                          </span>
                          <span class="sp-stats">
                            <span :title="$t('Average speed while moving (position units per second)')"><b>{{ Math.round(f.script_avg_speed) }}</b> {{ $t('avg') }}</span>
                            <span :title="$t('Fastest movement (position units per second)')"><b>{{ Math.round(f.script_max_speed) }}</b> {{ $t('max') }}</span>
                            <span :title="$t('Up and down cycles per minute')"><b>{{ Math.round(f.script_strokes_per_min) }}</b> {{ $t('spm') }}</span>
                            <span :title="$t('Average stroke depth')" class="sp-depth"><b-icon pack="mdi" icon="arrow-expand-vertical" custom-size="mdi-12px"/><b>{{ Math.round(f.script_depth) }}%</b></span>
                          </span>
                        </div>
                      </div>
                      <div class="media-right">
                        <button class="button is-dark is-small is-outlined" :data-tooltip="$t('Unmatch file from scene')" @click='unmatchFile(f)'>
                          <b-icon pack="mdi" icon="link-off" size="is-small"></b-icon>
                        </button>&nbsp;
                        <button class="button is-danger is-small is-outlined" :data-tooltip="$t('Delete file from disk')" @click='removeFile(f)'>
                          <b-icon pack="mdi" icon="delete" size="is-small"></b-icon>
                        </button>
                      </div>
                    </div>
                  </div>
                </b-tab-item>

                <b-tab-item :label="$t('Cuepoints ({n})', { n: sortedCuepoints.length })" v-if="!displayingAlternateSource">
                  <div class="block-tab-content block">
                  <div class="cuepoint-editor">
                  <div class="block-tags block">
                   <b-taglist>
                    <b-tooltip  type="is-danger" :label="disableSaveMsg()" position="is-right" :delay=250 :active="disableSaveButtons()">
                      <b-button @click="updateCuepoint(false)" class="tag is-info is-small is-warning" accesskey="a" :disabled="disableSaveButtons()" >
                        {{ $t('Add New') }}
                      </b-button>
                    </b-tooltip>
                      <b-button @click="vidPosition = new Date(0,0,0,0,0, 0, player.currentTime() * 1000)" class="tag is-info is-small is-warning" accesskey="t">{{ $t('Current Time') }}</b-button>
                    <b-tooltip type="is-danger" :label="disableSaveMsg()" position="is-right" :delay=250 :active="disableSaveButtons()">
                      <b-button v-if="currentCuepointId > 0" @click="updateCuepoint(true)" class="tag is-info is-small is-warning" accesskey="s"
                        :disabled="disableSaveButtons()" >
                        {{ $t('Save Edit') }}
                      </b-button>
                    </b-tooltip>
                      <b-button v-if="cuepointName!=''" @click='cuepointName=""' class="tag is-info is-small is-warning" >{{ $t('Clear Cuepoint Name') }}</b-button>
                      <b-button v-if="tagAct!=''" @click='setCuepointName("")' class="tag is-info is-small is-warning" accesskey="c">{{ $t('Clear Action') }}</b-button>
                    </b-taglist>
                  </div>

                  <div class="is-divider" :data-content="$t('Cuepoint Positions')"></div>
                  <div class="block-tags block">
                    <b-taglist>
                      <b-button v-for="(c, idx) in cuepointPositionTags.slice(1)" :key="'pos' + idx" @click='setCuepointName([c])' class="tag is-info is-small">{{c}}</b-button>
                    </b-taglist>
                  </div>
                  <div class="is-divider" :data-content="$t('Default Cuepoint Actions')"></div>
                  <div class="block-tags block">
                    <b-taglist>
                      <b-button v-for="(c, idx) in cuepointActTags.slice(1)" :key="'action' + idx" @click='setCuepointName([c])' class="tag is-info is-small">{{c}}</b-button>
                    </b-taglist>
                  </div>
                  <div class="is-divider" :data-content="$t('Cast Cuepoints')"></div>
                  <div class="block-tags block">
                    <b-taglist>
                      <b-button v-for="(c, idx) in item.cast" :key="'cast' + idx" @click='setCuepointName([c.name])' class="tag is-info is-small">{{c.name}}</b-button>
                    </b-taglist>
                  </div>
                  <div class="is-divider" :data-content="$t('Scene Tag Cuepoints')"></div>
                  <div class="block-tags block">
                    <b-taglist>
                      <b-button v-for="(tag, idx) in item.tags" :key="'tag' + idx" @click='setCuepointName([tag.name])'
                         class="tag is-info is-small">{{ tag.name }}</b-button>
                    </b-taglist>
                  </div>
                  </div>
                    <div class="block" >
                      <div class="columns">
                        <div class="column is-2">
                        <b-field :label="$t('Track')" width="7.25em" label-position="on-border">
                          <b-input v-model="track" width="7.25em"></b-input>
                        </b-field>
                        </div>
                        <div class="column">
                        <b-field :label="$t('Name')" label-position="on-border">
                          <b-autocomplete v-model="cuepointName" :data="filteredCuepointPositionList" :open-on-focus="true"></b-autocomplete>
                        </b-field>
                        </div>
                        <div class="column is-2">
                        <b-field :label="$t('Start')" label-position="on-border">
                          <b-timepicker v-model="vidPosition" rounded editable :placeholder="$t('Defaults to player position')" hour-format="24" :enable-seconds="true" :max-time="maxTime" :time-formatter="timeFormatter" :time-parser="timeParser" >
                          <b-button
                            :label="$t('Current Time')"
                            type="is-primary"
                            @click="vidPosition = new Date(0,0,0,0,0, 0, player.currentTime() * 1000)" />
                          </b-timepicker>
                        </b-field>
                        </div>
                        <div class="column is-2">
                          <b-field :label="$t('End')" label-position="on-border">
                          <b-timepicker v-model="endTime" rounded editable :placeholder="$t('Defaults to player position')" hour-format="24" :enable-seconds="true" :max-time="maxTime" :time-formatter="timeFormatter" :time-parser="timeParser" >
                          <b-button
                            :label="$t('Current Time')"
                            type="is-primary"
                            @click="endTime = new Date(0,0,0,0,0, 0, player.currentTime() * 1000)" />
                          </b-timepicker>
                        </b-field>
                        </div>
                      </div>
                    </div>
                    <div>
                        <b-table :data="sortedCuepoints" :narrowed=true focusable striped
                          @select="cuepointSelected">
                          <b-table-column field="track" :label="$t('Track')" width="7.25em" v-slot="props" >
                            {{ props.row.track ==null ? "" :  props.row.track }}
                          </b-table-column>
                          <b-table-column field="name" :label="$t('Name')" v-slot="props"  is-small>
                            {{ props.row.name }}
                          </b-table-column>
                          <b-table-column field="time_start" :label="$t('Start')" v-slot="props" width="6.5em"  >
                            {{ humanizeSeconds1DP(props.row.time_start) }}
                          </b-table-column>
                          <b-table-column field="time_end" :label="$t('End')" v-slot="props" width="6.5em"  >
                            {{ props.row.time_end==null ? "" :  humanizeSeconds1DP(props.row.time_end) }}
                          </b-table-column>
                          <b-table-column field="rating" v-slot="props" width="7em"  >
                            <b-field v-if="props.row.track!=null">
                              <star-rating :key="props.row.id" v-model="props.row.rating" :rating="props.row.rating" @rating-selected="setCuepointRating(props.row)" :increment="0.5" :star-size="10" />
                              <b-icon v-if="props.row.rating>0" pack="mdi" icon="autorenew" size="is-small" @click="clearCuepointRating(props.row)" style="padding-left: .25em;padding-top: .5em;"/>
                            </b-field>
                          </b-table-column>
                          <b-table-column v-slot="props" width="1em" >
                            <button class="button is-danger is-outlined is-small" @click="deleteCuepoint(props.row.id)" :data-tooltip="$t('Delete cuepoint')">
                              <b-icon pack="mdi" icon="delete" />
                            </button>
                          </b-table-column>
                        </b-table>
                    </div>
                  </div>
                </b-tab-item>

                <b-tab-item :label="$t('Description')" v-if="item.synopsis">
                  <div class="block-tab-content block">
                    <div class="description-text">{{ item.synopsis }}</div>
                  </div>
                </b-tab-item>

                <b-tab-item :label="$t('Watch history')" v-if="!displayingAlternateSource">
                  <div class="block-tab-content block">
                    <div>
                      {{ $t('{n} view session, total duration {duration} | {n} view sessions, total duration {duration}', { n: historySessionsCount, duration: humanizeSeconds(historySessionsDuration) }, historySessionsCount) }}
                    </div>
                    <div class="content is-small">
                      <div class="block" v-for="(session, idx) in item.history" :key="idx">
                        <strong>{{ format(parseISO(session.time_start), "yyyy-MM-dd kk:mm:ss") }} -
                          {{ humanizeSeconds(session.duration) }}</strong>
                      </div>
                    </div>
                  </div>
                </b-tab-item>

                <b-tab-item v-if="this.$store.state.optionsAdvanced.advanced.showSceneSearchField && !displayingAlternateSource" :label="$t('Search fields')">
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

    </template>
    <template #footer>
      <div class="scene-id">
        {{ item.scene_id }}
        <span  v-if="this.$store.state.optionsAdvanced.advanced.showInternalSceneId">{{ $t('Internal ID') }}: {{item.id}}</span>
        <a v-if="this.$store.state.optionsAdvanced.advanced.showHSPApiLink" :href="`/heresphere/${item.id}`" target="_blank" rel="noreferrer" style="margin-left:0.5em">
          <img src="/ui/icons/heresphere_24.png" style="height:15px;"/>
        </a>
      </div>
    </template>
    <button class="modal-close is-large" :aria-label="$t('close')" @click="close()"></button>
    <a class="prev" @click="prevScene" v-if="$store.getters['sceneList/prevScene'](item) !== null && !displayingAlternateSource"
       :title="$t('Keyboard shortcut: O')">&#10094;</a>
    <a class="next" @click="nextScene" v-if="hasNextScene && !displayingAlternateSource"
       :title="$t('Keyboard shortcut: P')">&#10095;</a>

    <div v-if="fullscreenGallery" 
         ref="fullscreenGallery"
         class="fullscreen-gallery" 
         :class="{ 'is-zoomed': fullscreenZoomed }"
         @click.self="handleGalleryClick">
      <button class="fullscreen-close" @click="closeFullscreenGallery" :title="$t('Close (Esc)')">
        <b-icon pack="mdi" icon="close" size="is-medium"></b-icon>
      </button>
      <button class="fullscreen-nav fullscreen-prev" @click.stop="fullscreenPrev" v-if="images.length > 1 && !fullscreenZoomed" :title="$t('Previous (←)')">
        <b-icon pack="mdi" icon="chevron-left" size="is-large"></b-icon>
      </button>
      <button class="fullscreen-nav fullscreen-next" @click.stop="fullscreenNext" v-if="images.length > 1 && !fullscreenZoomed" :title="$t('Next (→)')">
        <b-icon pack="mdi" icon="chevron-right" size="is-large"></b-icon>
      </button>
      <img 
        ref="fullscreenImage"
        :src="getFullscreenImageURL(images[fullscreenIndex].url)" 
        class="fullscreen-image"
        :class="{ 'is-loaded': fullscreenImageLoaded, 'is-zoomed': fullscreenZoomed, 'can-zoom': fullscreenCanZoom }"
        @load="onFullscreenImageLoad"
        @error="onFullscreenImageError"
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
  </SplitModal>
</template>

<script>
import ky from 'ky'
import { getAlternateSources } from '@/api'
import VrPlayer from '../../components/VrPlayer.vue'
import SplitModal from '../../components/SplitModal.vue'
import { format, parseISO } from 'date-fns'
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
import { isAiScript, scriptGenerator, scriptLicense } from '@/scripts'
import gallery from '@/gallery'

export default {
  name: 'SceneDetails',
  components: { SplitModal, VrPlayer, VueLoadImage, GlobalEvents, StarRating, WatchlistButton, FavouriteButton, LinkStashdbButton, WishlistButton, WatchedButton, EditButton, RefreshButton, RescrapeButton, TrailerlistButton, HiddenButton },
  mixins: [gallery],
  data () {
    return {
      tagsExpanded: false,
      tagsOverflow: false,
      hiddenTags: 0,
      index: 1,
      activeTab: 0,
      activeMedia: 0,
      playerRequest: null,
      player: {},
      vrPlugin: null,
      tagAct: '',
      cuepointName: '',
      cuepointRating: 0,
      cuepointPositionTags: ['', 'standing', 'sitting', 'laying', 'kneeling'],
      cuepointActTags: ['', 'handjob', 'blowjob', 'doggy', 'cowgirl', 'revcowgirl', 'missionary', 'titfuck', 'anal', 'cumshot', '69', 'facesit'],
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
    castHasAge () {
      return this.castimages.some(c => c.age != null)
    },
    // all scripts of the scene, for the overlay's picker (selected one marked)
    playerScripts () {
      const files = (this.item && this.item.file) || []
      const scripts = files.filter(f => f.type === 'script')
      const sel = scripts.find(f => f.is_selected_script) || scripts[0]
      return scripts.map(f => ({ id: f.id, name: f.filename, selected: f === sel, ai: isAiScript(f.filename), gen: scriptGenerator(f.script_description) }))
    },
    playerScriptURL () {
      const files = (this.item && this.item.file) || []
      const scripts = files.filter(f => f.type === 'script')
      const f = scripts.find(f => f.is_selected_script) || scripts[0]
      return f ? '/api/dms/file/' + f.id + '?dnt=true' : ''
    },
    playerShortcuts () {
      return [
        { keys: 'Space / K', label: this.$t('play / pause') },
        { keys: '← →', label: this.$t('skip') },
        { keys: this.$t('Scroll'), label: this.$t('zoom') },
        { keys: 'O / P', label: this.$t('prev / next scene') },
        { keys: 'G', label: this.$t('gallery') },
        { keys: 'F', label: this.$t('favourite') },
        { keys: 'W', label: this.$t('watchlist') },
        { keys: '⇧W', label: this.$t('watched') },
        { keys: 'E', label: this.$t('edit') },
        { keys: 'Esc', label: this.$t('close') }
      ]
    },
    item () {
      const item = this.$store.state.overlay.details.scene
      if (this.$store.state.optionsWeb.web.tagSort === 'alphabetically') {
        item.tags.sort((a, b) => a.name < b.name ? -1 : 1)
      }
      // age at release; an unknown release date (e.g. undated upcoming scenes) falls back to today
      let releasedate = parseISO(item.release_date)
      if (!(releasedate.getFullYear() > 1900)) releasedate = new Date()
      let imgs = item.cast.map((actor) => {
        let birthdate = parseISO(actor.birth_date)
        let label = actor.name
        let age = null
        if (birthdate.getFullYear() > 1900) {
          age = releasedate.getFullYear() - birthdate.getFullYear()
          if ((releasedate.getMonth() < birthdate.getMonth()) || (releasedate.getMonth() == birthdate.getMonth() && releasedate.getDate() < birthdate.getDate())) {
            age -= 1
          }
          if (age < 18 || age > 99) age = null
          else label += `, ${this.$t('{age} in scene', { age })}`
        }
        let img = actor.image_url
        if (img == "" ){
          img = "/ui/images/actor_placeholder.svg"
        }
        if (actor.name.startsWith("aka:")) {
          img = ""
        }
        return { src: img, visible: false, actor_name: actor.name, actor_label: label, actor_id: actor.id, first_name: actor.name.split(' ')[0], age }
      });

      this.castimages =  imgs.filter((img) => {
        return img.src !== '';
        });
      return item
    },
    searchFieldsSceneId () {
      const show = this.$store.state.optionsAdvanced.advanced.showSceneSearchField && !this.displayingAlternateSource
      return show ? this.item.id : null
    },
    images () {
      if (this.item.images=="null") {
        return "[]"
      }
      return JSON.parse(this.item.images).filter(im => im && im.url)
    },
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
      // an alternate source is scene data stored in external_references, so lists and ratings do not apply
      if ( this.$store.state.overlay.details.altsrc != null) return true
      return false
    },
    async getAlternateSceneSources() {
      this.alternateSources = [];
      if (this.displayingAlternateSource) return 0
      try {
        const response = await getAlternateSources(this.item.id);
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
          title: extdata.scene?.title || this.$t('No Title')
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

    ky.get('/api/options/cuepoints').json().then(data => {
      this.cuepointActTags = data.actions
      this.cuepointPositionTags = data.positions
      this.cuepointActTags.unshift("")
      this.cuepointPositionTags.unshift("")
      })    
},
watch:{
    searchFieldsSceneId: {
      handler (id) { this.getSearchFields(id) },
      immediate: true
    },
  quickFindOverlayState(newVal){
    if (newVal == true) {
      return
    }
    if (this.waitingForQuickFind){
      this.waitingForQuickFind = false
      if (this.$store.state.overlay.quickFind.selectedScene != null && this.$store.state.overlay.quickFind.selectedScene.id > 0) {
        this.$buefy.dialog.confirm({
          title: this.$t('Relink scene'),
          message: this.$t('Do you wish to link this scene to {title}', { title: `<strong>${this.$store.state.overlay.quickFind.selectedScene.title}</strong>` }),
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
    if (newVal === 1) this.$nextTick(() => this.loadPlayer())
  },
  'item.scene_id'(newVal, oldVal) {
    // the keyed wrapper renders a fresh <video> for the new scene
    if (!newVal || oldVal === undefined || newVal === oldVal) return
    this.disposePlayer()
    this.playerRequest = null
    this.carouselSlide = 0
    this.tagsExpanded = false
    if (this.activeMedia === 1 && this.firstVideoFile()) {
      this.$nextTick(() => this.loadPlayer())
    } else {
      this.activeMedia = 0
    }
  },
  images: {
    handler () {
      this.carouselImagesLoaded = {}
      this.carouselImageRetries = {}
      this.directImages = {}
      this.carouselImageKeys = {}
      this.scenePalette = null
    },
    immediate: false
  }
},
  methods: {
    isAiScript,
    scriptGenerator,
    scriptLicense,
    watchTagGroup (el) {
      if (el === this.tagGroupEl) return
      if (this.tagObserver) this.tagObserver.disconnect()
      this.tagGroupEl = el
      if (!el) { this.tagsOverflow = false; return }
      this.tagObserver = new ResizeObserver(() => this.measureTags())
      this.tagObserver.observe(el)
      this.measureTags()
    },
    measureTags () {
      const el = this.tagGroupEl
      if (!el) return
      const limit = parseFloat(getComputedStyle(el).getPropertyValue('--tags-collapsed')) || 80
      this.tagsOverflow = el.scrollHeight > limit + 2
      const top = el.getBoundingClientRect().top
      this.hiddenTags = [...el.children].filter(c => c.getBoundingClientRect().top - top >= limit).length
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
      this.$nextTick(() => {
        if (this.$refs.carouselImages && this.$refs.carouselImages[index]) {
          const img = this.$refs.carouselImages[index]
          if (img.naturalWidth > 100) {
            this.carouselImagesLoaded[index] = true
            this.carouselImageRetries[index] = 0
            this.fetchCarouselImageInfo(img.src, index)
            if (index === 0 && !this.scenePalette) {
              this.extractSceneColor(img)
            }
          } else if ((this.carouselImageRetries[index] || 0) < 3) {
            this.retryCarouselImage(index)
          } else {
            this.carouselImagesLoaded[index] = true
          }
        } else {
          this.carouselImagesLoaded[index] = true
        }
      })
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
    firstVideoFile () {
      if (this.displayingAlternateSource) return null
      return this.filesByType.find(f => f.type === 'video') || null
    },
    disposePlayer () {
      // the keyed <VrPlayer> disposes itself on unmount; only drop our handle
      this.player = {}
      this.vrPlugin = null
    },
    // the Player tab watcher loads it, so this cannot race the default first-video load
    openPlayer (request) {
      this.playerRequest = request
      if (this.activeMedia === 1) this.$nextTick(() => this.loadPlayer())
      else this.activeMedia = 1
    },
    loadPlayer () {
      if (this.activeMedia !== 1) return
      if (!this.vrPlugin) this.setupPlayer()
      if (!this.vrPlugin) return // Player tab not rendered (e.g. scene has no files)
      let req = this.playerRequest
      this.playerRequest = null
      if (!req) {
        const v = this.firstVideoFile()
        if (!v) return
        req = { src: '/api/dms/file/' + v.id + '?dnt=true', projection: v.projection, play: false }
      }
      this.updatePlayer(req.src, req.projection)
      if (req.play) this.player.play()
    },
    setupPlayer () {
      const p = this.$refs.player
      if (!p || !p.el() || !p.el().isConnected) return
      this.player = p
      this.vrPlugin = true // player is ready
    },
    updatePlayer (src, projection) {
      if (!this.vrPlugin) return // player tab not mounted yet
      this.player.poster('') // black until playback starts
      if (src) this.player.load(src, projection)
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
      this.openPlayer({ src: '/api/dms/preview/' + this.item.scene_id, projection: 'flat', play: true })
    },
    playFile (file) {
      this.openPlayer({ src: '/api/dms/file/' + file.id + '?dnt=true', projection: file.projection, play: true })
    },
    unmatchFile (file) {
      this.$buefy.dialog.confirm({
        title: this.$t('Unmatch file'),
        message: this.$t("You're about to unmatch the file {file} from this scene. Afterwards, it can be matched again to this or any other scene.", { file: `<strong>${file.filename}</strong>` }),
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
        title: this.$t('Remove file'),
        message: this.$t("You're about to remove file {file} from {disk}.", { file: `<strong>${file.filename}</strong>`, disk: `<strong>${this.$t('disk')}</strong>` }),
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

        centers.sort((a, b) => (0.299*a[0]+0.587*a[1]+0.114*a[2]) - (0.299*b[0]+0.587*b[1]+0.114*b[2]))

        const rgb = c => `rgb(${c[0]}, ${c[1]}, ${c[2]})`
        const pastel = (c, amt) => c.map(v => Math.round(v + (255 - v) * amt))

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

        let bestSat = -1, accent = centers[1]
        for (const c of centers) {
          const mx = Math.max(...c), mn = Math.min(...c)
          const sat = mx > 0 ? (mx - mn) / mx : 0
          if (sat > bestSat) { bestSat = sat; accent = c }
        }

        let [h, s, l] = rgbToHsl(accent[0], accent[1], accent[2])
        if (s < 0.5) s = s + (0.5 - s) * 0.6
        s = Math.min(s, 0.85)
        l = Math.max(0.35, Math.min(0.55, l))
        accent = hslToRgb(h, s, l)

        const luminance = c => {
          const toLinear = x => { const v = x/255; return v <= 0.03928 ? v/12.92 : ((v+0.055)/1.055)**2.4 }
          return 0.2126*toLinear(c[0]) + 0.7152*toLinear(c[1]) + 0.0722*toLinear(c[2])
        }
        const lum = luminance(accent)
        const accentText = lum > 0.179 ? '#1a1a1a' : '#ffffff'

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
      this.player.currentTime(cuepoint.time_start)
      this.player.play()
    },
    updateCuepoint (editCuepoint) {
      if (this.disableSaveButtons()) return
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
        this.$store.dispatch('overlay/openDetails', { scene: data })
      }
    },
    prevScene () {
      const data = this.$store.getters['sceneList/prevScene'](this.item)
      if (data !== null && !this.displayingAlternateSource) {
        this.$store.dispatch('overlay/openDetails', { scene: data })
      }
    },
    // keys inside the player are handled by its own shortcuts
    fromPlayer (e) {
      return !!(e && e.target && e.target.closest && e.target.closest('video-player'))
    },
    togglePlay (e) {
      if (this.fromPlayer(e)) return
      if (this.activeMedia === 0 || !this.player || typeof this.player.paused !== 'function') return
      if (this.player.paused()) this.player.play()
      else this.player.pause()
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
        this.closeFullscreenGallery()
      } else if (this.activeMedia == 0) {
        this.openFullscreenGallery(this.carouselSlide)
      } else {
        this.activeMedia = 0
      }
    },
    handleLeftArrow (e) {
      if (this.activeMedia !== 0 && this.fromPlayer(e)) return
      if (this.fullscreenGallery) {
        this.fullscreenPrev()
      } else if (this.activeMedia === 0) {
        this.carouselSlide = this.carouselSlide - 1
      } else {
        this.playerStepBack(this.lastSkipBackInterval)
      }
    },
    handleRightArrow (e) {
      if (this.activeMedia !== 0 && this.fromPlayer(e)) return
      if (this.fullscreenGallery) {
        this.fullscreenNext()
      } else if (this.activeMedia === 0) {
        this.carouselSlide = this.carouselSlide + 1
      } else {
        this.playerStepForward(this.lastSkipFowardInterval)
      }
    },
    timeFormatter(time) {
       return new Intl.DateTimeFormat('en', { hourCycle: 'h23', hour: "2-digit", minute: "2-digit", second: "2-digit", fractionalSecondDigits: 1 }).format(time)
    },
    timeParser(inputString) {
      let items = inputString.split(":")
      return new Date(0, 0, 0, items[0],items[1], 0, items[2]*1000)
    },
    cuepointSelected(cuepoint) {
      this.vidPosition = new Date(0, 0, 0, 0, 0, 0, cuepoint.time_start*1000)
      this.endTime = new Date(0, 0, 0, 0, 0, 0, cuepoint.time_end*1000)
      this.currentCuepointId = cuepoint.id
      this.cuepointName = cuepoint.name
      this.track=cuepoint.track
      this.cuepointRating=cuepoint.rating
      this.player.currentTime(cuepoint.time_start)
      this.player.play()
    },
    disableSaveButtons() {
      if (this.track!=null && this.track!="" && (isNaN(this.endTime) || this.endTime==null)) return true
      if ((this.track==null || this.track==="") && !isNaN(this.endTime) && this.endTime!=null) return true
      return false
    },
    disableSaveMsg() {
      if (this.track!=null && this.track!="" && (isNaN(this.endTime) || this.endTime==null)) return this.$t('Specify a End Time')
      if ((this.track==null || this.track==="") && !isNaN(this.endTime) && this.endTime!=null) return this.$t('End Time is only valid for HSP Cuepoints')
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
    getSearchFields (id) {
      this.searchfields = []
      if (id === null) return
      ky.get('/api/scene/searchfields', { searchParams: { q: id } }).json().then(data => {
        if (id === this.searchFieldsSceneId) this.searchfields = data
      })
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
        this.$buefy.toast.open({ message: this.$t('The scene was sucessfully relinked to a new Scene'), type: 'is-primary', duration: 3000 });
      }
    },
    async scrapeScene() {
      this.$buefy.dialog.confirm({
        title: this.$t('Scrape & Create Scene'),
        message: this.$t('Do you wish to create a seperate XBVR scene from this linked scene {url}', { url: `<strong>${this.$store.state.overlay.details.altsrc.url}</strong>` }),
        type: 'is-info is-wide',
        hasIcon: true,
        id: 'heh',
        onConfirm: () => {
          const url = this.$store.state.overlay.details.altsrc.url
          this.$store.state.overlay.details.altsrc = null
          this.$store.commit('overlay/hideDetails')
          this.$store.commit('optionsSceneCreate/setScrapeScene', url )
          this.$store.commit('optionsSceneCreate/showSceneCreate', true )
          this.$router.push({ path: '/options'})
        }
      })

    },
    async refreshExtRef() {
      this.$buefy.dialog.confirm({
        title: this.$t('Continue?'),
        message: this.$t('This will remove the scene, rescrape the site to relink it to an XBVR scene'),
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
        this.$buefy.toast.open({ message: this.$t('The scene was removed, ready to rescan'), type: 'is-primary', duration: 3000 });
      }
    },
    flagExtRefDeleted() {
      this.$buefy.dialog.confirm({
        title: this.$t('Continue?'),
        message: this.$t('This will unlink the scene and prevent it from relinking to any scene. This cannot be undone'),
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
        this.$buefy.toast.open({ message: this.$t('The scene was unlinked and will not be relinked to any scene'), type: 'is-primary', duration: 3000 });
      }
    },    
    format,
    parseISO,
    prettyBytes
  }
}
</script>

<style lang="less" scoped>
.cuepoint-editor {
  margin-bottom: 1.5rem;
}
.cuepoint-editor .block:last-child {
  margin-bottom: 0;
}
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
  padding-top: 56.25% !important;
  background: transparent !important;
}




/* Files tab: long names and paths stay on one line with an ellipsis (full text
   on hover) instead of making the modal scroll sideways. */
.file-content {
  min-width: 0;
  overflow: hidden;
}
.scene-file-name,
.scene-file-path {
  display: block;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.player-wrapper {
  width: 100%;
  margin-bottom: 8px;
}

.shortcut-legend {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 4px 12px;
  margin-top: 6px;
  font-size: 0.72rem;
  color: var(--text-muted, #7a7a7a);
}

.shortcut-legend kbd {
  display: inline-block;
  min-width: 1.5em;
  padding: 0 4px;
  margin-right: 2px;
  border: 1px solid currentColor;
  border-radius: 4px;
  font-family: inherit;
  font-size: 0.68rem;
  line-height: 1.4;
  text-align: center;
  opacity: 0.85;
}

.skip-controls {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: center;
  gap: 4px 0.75em;
  margin-top: 6px;
  max-width: 100%;
}

.skip-group {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  align-items: center;
  gap: 2px;
}
.skip-group :deep(.button.tag) {
  height: 1.6rem;
  min-width: 2.8em;
  background: transparent !important;
  border: 1px solid rgba(127, 127, 127, 0.3) !important;
  color: inherit !important;
  font-variant-numeric: tabular-nums;
}
.skip-group :deep(.button.tag:hover) {
  border-color: var(--accent, #7957d5) !important;
  color: var(--accent, #7957d5) !important;
}



:deep(.modal-card-body) {
  overflow-y: auto;
  max-height: calc(100vh - 40px);
  scrollbar-width: none;
  border-top: 3px solid var(--accent, #7957d5);
  border-radius: 8px;
  transition: border-color 0.8s ease;
}
.tab-empty {
  padding: 0.75rem 0;
  opacity: 0.6;
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

.detail-rating :deep(.vue-star-rating) {
  display: flex;
  align-items: center;
  line-height: 1;
}
.rating-reset {
  display: inline-flex;
  align-items: center;
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
/* name, plus an age line on every card when any actor has an age, so the row stays even */
.cast-caption {
  display: flex;
  flex-direction: column;
  width: 0;
  min-width: 100%;
  margin-top: 4px;
  line-height: 1.25;
  text-align: center;
}
.cast-name {
  font-size: 0.72rem;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.cast-age {
  min-height: 1.25em;
  font-size: 0.65rem;
  opacity: 0.6;
  font-variant-numeric: tabular-nums;
}


.cast-thumb {
  height: clamp(80px, 7vw, 100px);
  aspect-ratio: 2 / 3;
  background: rgba(127, 127, 127, 0.12);
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
  --tags-collapsed: 52px;
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}
.tag-group.is-collapsed {
  max-height: var(--tags-collapsed);
  overflow: hidden;
}
.tags-toggle {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  margin-top: 4px;
  padding: 2px 6px 2px 2px;
  border: none;
  border-radius: 4px;
  background: none;
  font: inherit;
  font-size: 0.75rem;
  color: var(--accent, #7957d5);
  cursor: pointer;
}
.tags-toggle:hover {
  background: rgba(121, 87, 213, 0.1);
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

.script-props {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 4px 5px;
  margin-top: 6px;
  font-size: 0.7rem;
  line-height: 18px;
}
.sp-badge {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  height: 18px;
  padding: 0 5px;
  border-radius: 999px;
  background: rgba(127, 127, 127, 0.12);
  white-space: nowrap;
}
.sp-badge :deep(.icon) {
  width: 13px;
  height: 13px;
}
.sp-badge.is-gen {
  background: rgba(62, 142, 208, 0.15);
  color: #3e8ed0;
}
.sp-badge.is-paid {
  background: rgba(255, 183, 77, 0.18);
  color: #b7791f;
}
html[data-theme="dark"] .sp-badge.is-paid {
  color: #e0a03c;
}
.sp-badge.is-free {
  background: rgba(72, 199, 142, 0.16);
  color: #3fbf86;
  font-weight: 600;
}
/* the numbers stay together as one block: "150 avg · 213 max · 73 spm · 59% depth" */
.sp-stats {
  display: inline-flex;
  white-space: nowrap;
  opacity: 0.75;
  font-variant-numeric: tabular-nums;
}
.sp-stats > span + span::before {
  content: '·';
  margin: 0 4px;
  opacity: 0.6;
}
.sp-stats b {
  font-weight: 600;
}
.sp-depth {
  display: inline-flex;
  align-items: center;
}
.sp-depth :deep(.icon) {
  width: 12px;
  height: 12px;
  margin-right: 1px;
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
.altsrc-image-wrapper {
  display: inline-block;
  margin-left: 5px;  
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
.cast-item[data-tooltip]::after {
  left: 0;
  right: auto;
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
