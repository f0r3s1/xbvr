<template>
  <div class="modal is-active">
    <GlobalEvents
      :filter="e => !['INPUT', 'TEXTAREA'].includes(e.target.tagName)"
      @keyup.esc="close"
    />
    <div class="modal-background"></div>
    <div class="modal-card" v-if="site != null">
      <header class="modal-card-head">
        <p class="modal-card-title">{{ $t("Matching parameters")}}: {{ site.name }}</p>        
        <button class="delete" @click="close" aria-label="close"></button>
      </header>
      <section class="modal-card-body" v-if="params != null">
        
        <!-- Selection Criteria -->
        <div class="card">
          <header class="card-header">
            <p class="card-header-title">Selection Criteria</p>
          </header>
          <div class="card-content">
            <div class="columns is-multiline">
              <div class="column is-one-third">
                <b-tooltip :label="$t('Days to wait after the release date, before linking. Useful where the main site releases after SLR/VRPorn/POVR, eg LethalHardcore')" 
                  :delay="500" type="is-primary" multilined size="is-large" position="is-bottom">
                  <b-field :label="$t('Delay linking (days)')">
                    <b-numberinput v-model="params.delay_linking"></b-numberinput>
                  </b-field>
                </b-tooltip>
              </div>
              <div class="column is-one-third">
                <b-tooltip :label="$t('Number of days to keep re-linking scenes after the release date')" :delay="500" type="is-primary" multilined>
                  <b-field :label="$t('Keep Re-linking (days)')">
                    <b-numberinput v-model="params.reprocess_links"></b-numberinput>
                  </b-field>
                </b-tooltip>
              </div>
              <div class="column is-one-third">
                <b-tooltip :label="$t('Do not link scenes prior to the specified date. The quality of metadata of older scenes is often poor and causes mismatches')" 
                  :delay="500" type="is-primary" multilined size="is-large" position="is-bottom">
                  <b-field :label="$t('Ignore Before Date')">
                    <b-datepicker v-model="ignoreReleasedBefore" :icon-right="ignoreReleasedBefore ? 'close-circle' : ''" icon-right-clickable @icon-right-click="clearDate">                    
                      <b-button
                          label="Today"
                          type="is-primary"
                          icon-left="calendar-today"
                          @click="ignoreReleasedBefore = new Date()" />
                      <b-button
                          label="Clear"
                          type="is-danger"
                          icon-left="close"
                          outlined
                          @click="ignoreReleasedBefore = null" />
                    </b-datepicker>
                  </b-field>
                </b-tooltip>
              </div>
            </div>
          </div>
        </div>      
        
        <!-- Release Date Searching -->
        <div class="card">
          <header class="card-header">
            <p class="card-header-title">Release Date Searching</p>
          </header>
          <div class="card-content">
            <div class="columns is-multiline">
              <div class="column is-one-third">
                <b-field label="Match Type">
                  <b-select required v-model="params.released_match_type">
                    <option value="should">Should match</option>
                    <option value="must">Must</option>
                    <option value="do not">Do not</option>
                  </b-select>
                </b-field>
              </div>
              <div class="column is-one-third">
                <b-tooltip :label="$t('Weighting of release date matches (vs Duration=1)')" 
                  :delay="500" type="is-primary" multilined size="is-large" position="is-bottom">
                  <b-field :label="$t('Boost Value')">
                    <b-numberinput v-model="params.boost_released" step="0.05"></b-numberinput>
                  </b-field>
                </b-tooltip>
              </div>
              <div class="column is-one-third"></div>
              <div class="column is-one-third">
                <b-tooltip :label="$t('Days prior to release date to search. If both prior and after are 0, range is not used.')" 
                  :delay="500" type="is-primary" multilined size="is-large" position="is-bottom">
                  <b-field :label="$t('Days Prior')">
                    <b-numberinput v-model="params.released_prior"></b-numberinput>
                  </b-field>
                </b-tooltip>
              </div>
              <div class="column is-one-third">
                <b-tooltip :label="$t('Days after release date to search. Usually 0. If both prior and after are 0, range is not used.')" 
                  :delay="500" type="is-primary" multilined size="is-large" position="is-bottom">
                  <b-field :label="$t('Days After')">
                    <b-numberinput v-model="params.released_after"></b-numberinput>
                  </b-field>
                </b-tooltip>
              </div>
            </div>
          </div>
        </div>

        <!-- Title Searching -->
        <div class="card">
          <header class="card-header">
            <p class="card-header-title">Title Searching</p>
          </header> 
          <div class="card-content">
            <div class="columns is-multiline">
              <div class="column is-one-third">
                <b-field :label="$t('Exact Match Boost')">
                  <b-numberinput v-model="params.boost_title" step="0.05"></b-numberinput>
                </b-field>
              </div>
              <div class="column is-one-third">
                <b-field :label="$t('Word Match Boost')">
                  <b-numberinput v-model="params.boost_title_any_words" step="0.05"></b-numberinput>
                </b-field>
              </div>
            </div>
          </div>
        </div>

        <!-- Duration Searching -->
        <div class="card">
          <header class="card-header">
            <p class="card-header-title">Duration Searching</p>
          </header>
          <div class="card-content">
            <div class="columns is-multiline">
              <div class="column is-one-third">
                <b-field :label="$t('Match Type')">
                  <b-select required v-model="params.duration_match_type">
                    <option value="should">Should match</option>
                    <option value="must">Must</option>
                    <option value="do not">Do not</option>
                  </b-select>
                </b-field>
              </div>
              <div class="column is-one-third">
                <b-field :label="$t('Minimum Duration')">
                  <b-numberinput v-model="params.duration_min"></b-numberinput>
                </b-field>
              </div>
              <div class="column is-one-third"></div>
              <div class="column is-one-third">
                <b-field :label="$t('Lower Range')">
                  <b-numberinput v-model="params.duration_range_less"></b-numberinput>
                </b-field>
              </div>
              <div class="column is-one-third">
                <b-field :label="$t('Upper Range')">
                  <b-numberinput v-model="params.duration_range_more"></b-numberinput>
                </b-field>
              </div>
            </div>
          </div>
        </div>

        <!-- Cast Searching -->
        <div class="card">
          <header class="card-header">
            <p class="card-header-title">Cast Searching</p>
          </header>
          <div class="card-content">
            <div class="columns is-multiline">
              <div class="column is-one-third">
                <b-field :label="$t('Match Type')">
                  <b-select required v-model="params.cast_match_type">
                    <option value="should">Should match</option>
                    <option value="must">Must</option>
                    <option value="do not">Do not</option>
                  </b-select>
                </b-field>
              </div>
              <div class="column is-one-third">
                <b-field :label="$t('Exact Match Boost')">
                  <b-numberinput v-model="params.boost_cast" step="0.05"></b-numberinput>
                </b-field>
              </div>
            </div>
          </div>
        </div>

        <!-- Description Searching -->
        <div class="card">
          <header class="card-header">
            <p class="card-header-title">Description Searching</p>
          </header>
          <div class="card-content">
            <div class="columns is-multiline">
              <div class="column is-one-third">
                <b-field :label="$t('Match Type')">
                  <b-select required v-model="params.desc_match_type">
                    <option value="should">Should match</option>
                    <option value="must">Must</option>
                    <option value="do not">Do not</option>
                  </b-select>
                </b-field>
              </div>
              <div class="column is-one-third">
                <b-field :label="$t('Exact Match Boost')">
                  <b-numberinput v-model="params.boost_description" step="0.05"></b-numberinput>
                </b-field>
              </div>
            </div>
          </div>
        </div>        

      </section>
      <footer class="modal-card-foot">
        <b-button type="is-primary" @click="saveSettings">Save settings</b-button>
      </footer>
    </div>
  </div>
</template>

<script>
import ky from 'ky'
import { format, parseISO } from 'date-fns'
import prettyBytes from 'pretty-bytes'
import GlobalEvents from 'vue-global-events'

export default {
  name: 'SceneMatchParams',
  components: { GlobalEvents },
  data () {
    return {
      site: null,
      params: null,
      ignoreReleasedBefore: null,
      format,
      parseISO
    }
  },
  computed: {
  },
  mounted () {    
    this.initView()
  },
  methods: {
    initView () {
      this.site=this.$store.state.overlay.sceneMatchParams.site
      ky.get('/api/options/site/match_params/' + this.site.id).json().then(data => {
        this.params = data
        this.ignoreReleasedBefore = new Date(this.params.ignore_released_before);
      })
    },
    close () {
      this.$store.commit('overlay/hideSceneMatchParams')
    },
    clearDate() {
      this.ignoreReleasedBefore = null
    },
    saveSettings() {      
      this.params.ignore_released_before=this.ignoreReleasedBefore
      ky.post(`/api/options/site/save_match_params`, { json: { site: this.site.id, match_params: this.params } })
      if (this.ignoreReleasedBefore != null) {        
        const formattedDate = this.ignoreReleasedBefore.toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric',});
        this.$buefy.dialog.confirm({
          title: 'Clear existing links',
          message: `Do you also wish to clear links from <strong>${formattedDate}</strong>`,
          type: 'is-info is-wide',
          hasIcon: true,
          id: 'heh',
          onConfirm: () => {
            ky.delete(`/api/extref/delete_extref_source_links/keep_manual`, { json: {external_source: 'alternate scene ' + this.site.id, delete_date: this.ignoreReleasedBefore} });
          }
        })
        
      }
    },
    prettyBytes
  }
}
</script>

<style scoped>
.modal-card {
  position: absolute;
  top: 2em;
  width: 60%;
  max-width: 900px;
}

.modal-card-body {
  padding: 1.5rem;
}

.card {
  margin-bottom: 1rem;
}

.card:first-child {
  margin-top: 0;
}

.card-header {
  background: #f5f5f5;
}

.card-header-title {
  padding: 0.5rem 1rem;
  font-size: 0.95rem;
}

.card-content {
  padding: 1rem;
}

.columns {
  margin: 0;
}

.column {
  padding: 0.5rem;
}
</style>
