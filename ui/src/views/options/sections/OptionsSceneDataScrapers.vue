<template>
  <div class="container">
    <b-loading :is-full-page="true" :active.sync="isLoading"></b-loading>
    <div class="content">
      <!-- Header -->
      <div class="scraper-header">
        <h3>{{$t('Scrape scenes from studios')}}</h3>
        <div class="header-right">
          <b-button type="is-primary" icon-left="play" @click="taskScrape('_enabled')">
            {{$t('Run selected scrapers')}}
          </b-button>
        </div>
      </div>
      <hr/>
    
    <!-- Table -->
    <b-table :data="scraperList" ref="scraperTable" class="scrapers-table" :mobile-cards="true">
      <!-- Enable Toggle -->
      <b-table-column field="is_enabled" :label="$t('On')" v-slot="props" width="50" sortable>
        <b-switch v-model="props.row.is_enabled" @input="$store.dispatch('optionsSites/toggleSite', {id: props.row.id})" size="is-small"/>
      </b-table-column>
      
      <!-- Studio Info (Icon + Name combined for mobile) -->
      <b-table-column field="sitename" :label="$t('Studio')" sortable searchable v-slot="props">
        <div class="studio-cell">
          <span class="studio-icon image is-24x24">
            <vue-load-image>
              <img slot="image" :src="getImageURL(props.row.avatar_url ? props.row.avatar_url : '/ui/images/blank.png')"/>
              <img slot="preloader" src="/ui/images/blank.png"/>
              <img slot="error" src="/ui/images/blank.png"/>
            </vue-load-image>
          </span>
          <span class="studio-info">
            <b-tooltip class="is-warning" :active="props.row.has_scraper == false" :label="$t('Scraper does not exist')" :delay="250">
              <a @click="navigateToStudio(props.row.sitename)" :class="[props.row.has_scraper ? 'has-text-link' : 'has-text-danger']">
                {{ props.row.sitename }}
              </a>
            </b-tooltip>
            <span class="source-tag" v-if="props.row.master_site_id">{{ getMasterSiteName(props.row.master_site_id) }}</span>
            <span class="custom-tag" v-if="!props.row.is_builtin">{{$t('Custom')}}</span>
          </span>
        </div>
      </b-table-column>
      
      <!-- Last Scrape -->
      <b-table-column field="last_update" :label="$t('Last scrape')" sortable v-slot="props">
        <span v-if="!runningScrapers.includes(props.row.id)">
          <span v-if="props.row.last_update !== '0001-01-01T00:00:00Z'" class="last-scrape">
            {{formatDistanceToNow(parseISO(props.row.last_update))}}
          </span>
          <span v-else class="never-scraped">{{$t('Never')}}</span>
        </span>
        <span v-else class="pulsate is-info">{{$t('Running...')}}</span>
      </b-table-column>
      
      <!-- Settings (consolidated toggles) -->
      <b-table-column field="settings" :label="$t('Settings')" v-slot="props" width="200">
        <div class="settings-badges">
          <!-- Limit scraping - all sites -->
          <b-tooltip :label="$t('Limit to newest scenes')" :delay="250">
            <span class="setting-badge" :class="{ 'is-active': props.row.limit_scraping }" 
                  @click="$store.dispatch('optionsSites/toggleLimitScraping', {id: props.row.id})">
              <b-icon icon="filter" size="is-small"/>
            </span>
          </b-tooltip>
          <!-- Main sites only: subscribed, stash, flaresolverr -->
          <template v-if="props.row.master_site_id==''">
            <b-tooltip :label="$t('Subscribed')" :delay="250">
              <span class="setting-badge" :class="{ 'is-active': props.row.subscribed }" 
                    @click="$store.dispatch('optionsSites/toggleSubscribed', {id: props.row.id})">
                <b-icon icon="star" size="is-small"/>
              </span>
            </b-tooltip>
            <b-tooltip :label="$t('Scrape Stashdb')" :delay="250">
              <span class="setting-badge" :class="{ 'is-active': props.row.scrape_stash }" 
                    @click="$store.dispatch('optionsSites/toggleScrapeStash', {id: props.row.id})">
                <b-icon icon="database" size="is-small"/>
              </span>
            </b-tooltip>
            <b-tooltip :label="$t('Use FlareSolverr')" :delay="250">
              <span class="setting-badge flare-badge" :class="{ 'is-active': props.row.use_flaresolverr }" 
                    @click="$store.dispatch('optionsSites/toggleUseFlareSolverr', {id: props.row.id})">
                <b-icon icon="fire" size="is-small"/>
              </span>
            </b-tooltip>
          </template>
          <!-- Alternate sites: link to master + edit params -->
          <template v-else>
            <b-tooltip :label="$t('Use FlareSolverr')" :delay="250">
              <span class="setting-badge flare-badge" :class="{ 'is-active': props.row.use_flaresolverr }" 
                    @click="$store.dispatch('optionsSites/toggleUseFlareSolverr', {id: props.row.id})">
                <b-icon icon="fire" size="is-small"/>
              </span>
            </b-tooltip>
            <b-tooltip :label="$t('Edit Matching Parameters')" :delay="250">
              <span class="setting-badge" @click="editMatchParams(props.row)">
                <b-icon icon="cog" size="is-small"/>
              </span>
            </b-tooltip>
          </template>
          <!-- Matching params for main sites that have them -->
          <b-tooltip :label="$t('Matching Parameters')" :delay="250" v-if="props.row.master_site_id=='' && props.row.matching_params">
            <span class="setting-badge" @click="editMatchParams(props.row)">
              <b-icon icon="cog" size="is-small"/>
            </span>
          </b-tooltip>
        </div>
      </b-table-column>
      
      <!-- Actions Menu -->
      <b-table-column field="options" v-slot="props" width="40">
        <b-dropdown aria-role="list" position="is-bottom-left">
          <template #trigger>
            <b-button icon-right="dots-vertical" size="is-small" type="is-text"/>
          </template>
          <b-dropdown-item v-if="props.row.has_scraper" aria-role="listitem" @click="taskScrape(props.row.id)">
            <b-icon icon="play" size="is-small"/> {{$t('Run scraper')}}
          </b-dropdown-item>
          <b-dropdown-item v-if="props.row.has_scraper" aria-role="listitem" @click="taskScrapeQuick(props.row.id)">
            <b-icon icon="lightning-bolt" size="is-small"/> {{$t('Quick scrape (first page)')}}
          </b-dropdown-item>
          <b-dropdown-item v-if="props.row.has_scraper && props.row.id != 'baberoticavr'" aria-role="listitem" @click="taskScrapeScene(props.row.id)">
            <b-icon icon="file-document-outline" size="is-small"/> {{$t('Single scene')}}
          </b-dropdown-item>
          <hr class="dropdown-divider" v-if="props.row.has_scraper">
          <b-dropdown-item v-if="props.row.has_scraper && props.row.master_site_id==''" aria-role="listitem" @click="forceSiteUpdate(props.row.name, props.row.id)">
            <b-icon icon="refresh" size="is-small"/> {{$t('Force update')}}
          </b-dropdown-item>
          <b-dropdown-item v-if="props.row.has_scraper && props.row.master_site_id!=''" aria-role="listitem" @click="removeSceneLinks(props.row, true)">
            <b-icon icon="link-off" size="is-small"/> {{$t('Remove links')}}
          </b-dropdown-item>
          <b-dropdown-item v-if="props.row.has_scraper && props.row.master_site_id!=''" aria-role="listitem" @click="removeSceneLinks(props.row, false)">
            <b-icon icon="link-variant-remove" size="is-small"/> {{$t('Remove links (keep edits)')}}
          </b-dropdown-item>
          <b-dropdown-item aria-role="listitem" @click="deleteScenes(props.row)" class="has-text-danger">
            <b-icon icon="delete" size="is-small"/> {{$t('Delete scenes')}}
          </b-dropdown-item>
          <hr class="dropdown-divider" v-if="props.row.master_site_id==''">
          <b-dropdown-item aria-role="listitem" @click="scrapeActors(props.row.name, props.row.id)" v-if="props.row.master_site_id==''">
            <b-icon icon="account-search" size="is-small"/> {{$t('Scrape actors')}}
          </b-dropdown-item>
        </b-dropdown>
      </b-table-column>
    </b-table>
    
    <!-- Footer Actions -->
    <div class="footer-actions">
      <b-button size="is-small" @click="toggleAllLimitScraping()" icon-left="filter">
        {{$t('Toggle Limit All')}}
      </b-button>
      <b-button size="is-small" @click="toggleAllSubscriptions()" icon-left="star">
        {{$t('Toggle Subscribe All')}}
      </b-button>
    </div>

    <b-modal :active.sync="isSingleScrapeModalActive"
             has-modal-card
             trap-focus
             aria-role="dialog"
             aria-modal>
      <div class="modal-card" style="width: auto">
        <header class="modal-card-head">
          <p class="modal-card-title">{{$t('Additional Details Required')}}</p>
        </header>
        <section class="modal-card-body">
          <b-field v-if="additionalInfoIdx == 0 && this.scraperwarning != ''"><span>{{this.scraperwarning}}</span></b-field>
          <b-field v-if="additionalInfoIdx == 0 && this.scraperwarning2 != ''"><span>{{this.scraperwarning2}}</span></b-field>          
          <b-field :label=this.additionalInfo[additionalInfoIdx].fieldPrompt>
            <b-input v-if="additionalInfo[additionalInfoIdx].type != 'checkbox'"
              :type=additionalInfo[additionalInfoIdx].type
              v-model='additionalInfo[additionalInfoIdx].fieldValue'
              :required=additionalInfo[additionalInfoIdx].required
              :placeholder=additionalInfo[additionalInfoIdx].placeholder                            
              ref="additionInfoInput"
              >
            </b-input>
            <b-checkbox v-if="additionalInfo[additionalInfoIdx].type == 'checkbox'" v-model="additionalInfo[additionalInfoIdx].fieldValue">{{this.additionalInfo[additionalInfoIdx].fieldPrompt}}</b-checkbox>
          </b-field>
        </section>
        <footer class="modal-card-foot">
          <button class="button is-primary" :disabled="this.additionalInfo[additionalInfoIdx].required && this.additionalInfo[additionalInfoIdx].fieldValue == ''" @click="taskScrapeSceneInfoEntered()">Continue
          </button>
        </footer>
      </div>
    </b-modal>

    </div>
  </div>

</template>

<script>
import ky from 'ky'
import VueLoadImage from 'vue-load-image'
import { formatDistanceToNow, parseISO } from 'date-fns'

export default {
  name: 'OptionsSites',
  components: { VueLoadImage },
  data () {
    return {
      javrQuery: '',
      tpdbSceneUrl: '',
      isLoading: false,
      sceneUrl: '',
      isSingleScrapeModalActive: false,
      additionalInfo: [{fieldName: "scene_url", fieldPrompt: "Scene Url", placeholder: "eg https://www.mysite.com/scenes/my scene", fieldValue: '', required: true, type: 'url' }],
      additionalInfoIdx: 0,
      currentScraper: '',
      scraperwarning: '',
      scraperwarning2: '',
    }
  },
  mounted () {
    this.$store.dispatch('optionsSites/load')
  },
  methods: {
    getImageURL (u) {
      if (u.startsWith('http')) {
        return '/img/128x/' + u.replace('://', ':/')
      } else {
        return u
      }
    },
    taskScrape (scraper) {
      ky.get(`/api/task/scrape?site=${scraper}`)
    },
    taskScrapeQuick (scraper) {
      ky.get(`/api/task/scrape?site=${scraper}&quick=true`)
    },
    taskScrapeScene (scraper) {
      this.currentScraper=scraper      
      this.additionalInfo = [{fieldName: "scene_url", fieldPrompt: "Scene Url", placeholder: "Enter the url for a VR Scene", fieldValue: '', required: true, type: 'url'}]      
      this.scraperwarning = "Take care to only use scene urls for the " + scraper + " Scraper"
      this.scraperwarning2 = ""
      switch (scraper) {
        case 'wankzvr':
        case 'milfvr':
        case 'herpovr':
        case  'brasilvr':
        case 'tranzvr':
          this.scraperwarning = "Only use povr.com urls for the " + scraper + " Scraper"
          break
        case 'tonightsgirlfriend':
          this.scraperwarning2 = "Warning " + scraper + " also includes 2d scenes, only select scenes from their VR section"
        case 'naughtyamericavr':
          this.scraperwarning2 = "Warning The NaughtyAmerica site also includes 2d scenes, only select scenes from their VR section"
          break
    }
      this.additionalInfoIdx=0
      this.isSingleScrapeModalActive = true      
    },
    taskScrapeSceneInfoEntered () {      
      const inputElement = this.$refs.additionInfoInput
      if (!inputElement.isValid) {
        // get the field again
        this.isSingleScrapeModalActive = true
        return
      }

      this.isSingleScrapeModalActive = false      
      var fieldCheckMsg = ""
      if (this.additionalInfo[0].fieldValue.toLowerCase().includes('fuckpassvr.com')) {
        var fieldCheckMsg="Note: Video Previews are not available when scraping single scenes from FuckpassVR"
      }
      if (this.additionalInfo[0].fieldValue.toLowerCase().includes('lethalhardcorevr.com')) {
        var fieldCheckMsg=`Please check the Site if the scene was for WhorecraftVR. Please check the Release Date`
      }
      if (this.additionalInfo[0].fieldValue.toLowerCase().includes('littlecaprice-dreams.com')) {
        var fieldCheckMsg=`Please specify a URL for the cover image`
      }
      if (this.additionalInfo[0].fieldValue.toLowerCase().includes('sexbabesvr.com')) {
        var fieldCheckMsg="Please check the Release Date"
      }
      if (this.additionalInfo[0].fieldValue.toLowerCase().includes('stasyqvr.com')) {
        var fieldCheckMsg=`Please specify a Duration if required`
      }
      if (this.additionalInfo[0].fieldValue.toLowerCase().includes('tonightsgirlfriend.com')) {
        var fieldCheckMsg="Please check the Release Date"
      }
      if (this.additionalInfo[0].fieldValue.toLowerCase().includes('virtualporn.com')) {
        var fieldCheckMsg=`Please check the Release Date and specify a Duration if required`
      }
      if (this.additionalInfo[0].fieldValue.toLowerCase().includes('wetvr.com')) {        
        var fieldCheckMsg="Please check the Release Date"
      }

      if (this.additionalInfoIdx == 0) {
        if (this.additionalInfo[0].fieldValue.toLowerCase().includes('wetvr.com')) {
          this.additionalInfo.push({fieldName: "scene_id", fieldPrompt: "Scene Id", placeholder: "eg 69037 (excl site prefix)", fieldValue: '', required: true, type: 'number'})
        }
      }
      
      this.additionalInfo[this.additionalInfoIdx].fieldValue = this.additionalInfo[this.additionalInfoIdx].fieldValue.trim()
      if (this.additionalInfoIdx + 1 < this.additionalInfo.length) {          
        this.additionalInfoIdx = this.additionalInfoIdx +1
        this.isSingleScrapeModalActive = true      
      } else {
        if (fieldCheckMsg != "") {
          this.$buefy.toast.open({message: `Scene scraping in progress, please wait for the Scene Detail popup`, type: 'is-warning', duration: 5000})
        } else {
          this.$buefy.toast.open({message: `Scene scraping in progress`, type: 'is-warning', duration: 5000})
        }
        ky.post(`/api/task/singlescrape`, {timeout: false, json: { site: this.currentScraper, sceneurl: this.additionalInfo[0].fieldValue, additionalinfo: this.additionalInfo.slice(1)}})
        .json()
        .then(data => { 
          if (data.status == 'OK') {          
            this.$store.commit('overlay/editDetails', { scene: data.scene })
            if (fieldCheckMsg != "") {
              this.$buefy.toast.open({message: fieldCheckMsg, type: 'is-warning', duration: 10000})
            }
          }
        })
      }
    },
    forceSiteUpdate (site, scraper) {
      ky.post('/api/options/scraper/force-site-update', {
        json: { scraper_id: scraper }
      })
      this.$buefy.toast.open(`Scenes from ${site} will be updated on next scrape`)
    },
    deleteScenes (site) {
      this.$buefy.dialog.confirm({
        title: this.$t('Delete scraped scenes'),
        message: `You're about to delete scraped scenes for <strong>${site.name}</strong>.`,
        type: 'is-danger',
        hasIcon: true,
        onConfirm: function () {
          if (site.master_site_id==""){
            ky.post('/api/options/scraper/delete-scenes', {
              json: { scraper_id: site.id }
            })
          } else {
            const external_source = 'alternate scene ' + site.id
            ky.delete(`/api/extref/delete_extref_source`, {
              json: {external_source: external_source}
            });
          }
        }
      })
    },
    removeSceneLinks (site, all) {
      this.$buefy.dialog.confirm({
        title: this.$t('Remove Scene Links'),
        message: `You're about to remove links for scenes from <strong>${site.name}</strong>. Scenes will be relinked after the next scrape.`,
        type: 'is-warning',
        hasIcon: true,
        onConfirm: function () {
          const external_source = 'alternate scene ' + site.id          
          if (all) {
            ky.delete(`/api/extref/delete_extref_source_links/all`, {
              json: {external_source: external_source}
            });
          } else {
            ky.delete(`/api/extref/delete_extref_source_links/keep_manual`, {
              json: {external_source: external_source}
            });
          }
        }
      })
    },
    scrapeActors(site, scraper) {      
      ky.get('/api/extref/generic/scrape_by_site/' + scraper)
      this.$buefy.toast.open(`Scraping Actor Details from ${site}`)
    },
    async toggleAllSubscriptions(){
      const table = this.$refs.scraperTable;
      this.isLoading=true
      for (let i=0; i<table.newData.length; i++) {
        await ky.put(`/api/options/sites/subscribed/${table.newData[i].id}`, { json: {} }).json()
        this.$store.dispatch('optionsSites/load')
      }
      this.isLoading=false
    },
    async toggleAllLimitScraping(){
      const table = this.$refs.scraperTable;
      this.isLoading=true
      for (let i=0; i<table.newData.length; i++) {
        await ky.put(`/api/options/sites/limit_scraping/${table.newData[i].id}`, { json: {} }).json()
        this.$store.dispatch('optionsSites/load')
      }
      this.isLoading=false
    },    
    editMatchParams(site){
      this.$store.commit('overlay/showSceneMatchParams', { site: site })
    },
    getMasterSiteName(siteId){
      if (siteId=="") {
        return ""
      }
      const site = this.scraperList.find(element => element.id === siteId);
      return site ? site.name : siteId;
    },
    getMasterSiteNameShort(siteId){
      if (siteId=="") {
        return ""
      }
      const site = this.scraperList.find(element => element.id === siteId);
      if (!site) return siteId;
      // Return shortened name (first word or first 15 chars)
      const name = site.sitename || site.name;
      return name.length > 15 ? name.substring(0, 15) + '...' : name;
    },
    navigateToStudio(studioName) {
      // Set the site filter and navigate to scenes page
      this.$store.state.sceneList.filters.sites = [studioName]
      this.$store.state.sceneList.filters.tags = []
      this.$store.state.sceneList.filters.attributes = []
      this.$router.push({
        name: 'scenes',
        query: { q: this.$store.getters['sceneList/filterQueryParams'] }
      })
    },
    parseISO,
    formatDistanceToNow
  },
  computed: {
    scraperList() {
      var items = this.$store.state.optionsSites.items;
      let re = /(.*)\s+\((.+)\)$/;
      for (let i=0; i < items.length; i++) {
        items[i].sitename = items[i].name;
        items[i].source = "";

        var m = re.exec(items[i].name);
        if (m) {
          items[i].sitename = m[1];
          items[i].source = m[2];
        }
      }
      return items;
    },
    items () {
      return this.$store.state.optionsSites.items
    },
    runningScrapers () {
      this.$store.dispatch('optionsSites/load')
      return this.$store.state.messages.runningScrapers
    }
  }
}
</script>

<style scoped>
  .scraper-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    flex-wrap: wrap;
    gap: 1rem;
  }

  .scraper-header h3 {
    margin-bottom: 0;
  }

  .scrapers-table {
    font-size: 0.95rem;
  }

  .studio-cell {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }

  .studio-icon {
    flex-shrink: 0;
    width: 24px;
    height: 24px;
    min-width: 24px;
    min-height: 24px;
  }

  .studio-icon img {
    border-radius: 4px;
    width: 24px;
    height: 24px;
    object-fit: cover;
  }

  .studio-info {
    display: flex;
    flex-direction: row;
    align-items: center;
    gap: 0.5rem;
    min-width: 0;
    flex: 1;
    overflow: hidden;
  }

  .source-tag {
    font-size: 0.7rem;
    color: #888;
    background: #f0f0f0;
    padding: 0.15rem 0.4rem;
    border-radius: 3px;
    flex-shrink: 1;
    max-width: 100%;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .custom-tag {
    font-size: 0.7rem;
    color: #fff;
    background: #9b59b6;
    padding: 0.15rem 0.4rem;
    border-radius: 3px;
    flex-shrink: 1;
    max-width: 100%;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .last-scrape {
    font-size: 0.85rem;
    color: #666;
  }

  .never-scraped {
    font-size: 0.85rem;
    color: #999;
    font-style: italic;
  }

  .settings-badges {
    display: flex;
    gap: 0.3rem;
  }

  .setting-badge {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 28px;
    height: 28px;
    border-radius: 4px;
    background: #f0f0f0;
    color: #888;
    cursor: pointer;
    transition: all 0.2s ease;
  }

  .setting-badge:hover {
    background: #e0e0e0;
    color: #666;
  }

  .setting-badge.is-active {
    background: #3273dc;
    color: white;
  }

  .setting-badge.flare-badge.is-active {
    background: #ff6b35;
    color: white;
  }

  .setting-badge.flare-badge:hover {
    background: #ff8c5a;
  }

  .setting-badge.is-active:hover {
    background: #2366d1;
  }

  .linked-site {
    font-size: 0.85rem;
  }

  .linked-site a {
    display: flex;
    align-items: center;
    gap: 0.3rem;
    color: #666;
  }

  .linked-site a:hover {
    color: #3273dc;
  }

  .footer-actions {
    display: flex;
    justify-content: flex-end;
    gap: 0.5rem;
    margin-top: 1rem;
    padding-top: 1rem;
    border-top: 1px solid #eee;
  }

  .running {
    opacity: 0.6;
    pointer-events: none;
  }

  .pulsate {
    animation: pulsate 0.8s linear infinite;
    color: #3273dc;
    font-weight: 500;
  }

  @keyframes pulsate {
    0%, 100% { opacity: 0.5; }
    50% { opacity: 1.0; }
  }

  /* Responsive adjustments */
  @media screen and (max-width: 768px) {
    .scraper-header {
      flex-direction: column;
      align-items: stretch;
    }

    .header-right {
      display: flex;
      justify-content: center;
    }

    .settings-badges {
      flex-wrap: wrap;
    }

    .footer-actions {
      flex-direction: column;
    }

    .footer-actions .button {
      width: 100%;
    }
  }
</style>

<style>
  .scrapers-table .table td {
    vertical-align: middle;
  }

  .scrapers-table .b-table .table-wrapper {
    overflow-x: auto;
  }
</style>
