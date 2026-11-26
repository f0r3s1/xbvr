<template>
  <OptionsSection :title="$t('Storage')">
    <template #actions>
      <b-button type="is-primary" size="is-small" @click="taskRescan">{{ $t('Rescan all folders') }}</b-button>
    </template>
  <div class="content">
    <div v-if="items.length > 0">
      <b-table :data="items"
               ref="table" default-sort="is_available" default-sort-direction="desc">
        <b-table-column field="path" :label="$t('Path')" sortable v-slot="props">
          {{ props.row.path }}
        </b-table-column>
        <b-table-column field="type" :label="$t('Type')" sortable v-slot="props">
          <b-icon pack="mdi" icon="cloud-outline" size="is-small" v-if="props.row.type !== 'local'"/>
          <b-icon pack="mdi" icon="folder-outline" size="is-small" v-else/>
        </b-table-column>
        <b-table-column field="is_available" :label="$t('Avail')" sortable v-slot="props">
          <b-icon pack="mdi" icon="check" size="is-small" v-if="props.row.is_available"></b-icon>
        </b-table-column>
        <b-table-column field="file_count" :label="$t('# of files')" sortable v-slot="props">
          {{ props.row.file_count }}
        </b-table-column>
        <b-table-column field="unmatched_count" :label="$t('Not matched')" sortable v-slot="props">
          {{ props.row.unmatched_count }}
        </b-table-column>
        <b-table-column field="total_size" :label="$t('Total size')" sortable v-slot="props">
          {{ prettyBytes(props.row.total_size) }}
        </b-table-column>
        <b-table-column field="last_scan" :label="$t('Last scan')" sortable v-slot="props">
            <span v-if="props.row.last_scan !== '0001-01-01T00:00:00Z'">
              {{ relativeTime(props.row.last_scan) }}
            </span>
          <span v-else>{{ $t('never') }}</span>
        </b-table-column>
        <b-table-column field="actions" v-slot="props">
          <b-field grouped>
            <button class="button is-small is-outlined" v-on:click='rescanFolder(props.row)' style="margin-right:1em" :title="$t('rescan folder')">
              <b-icon pack="mdi" icon="folder-refresh-outline"></b-icon>
            </button>
            <button class="button is-danger is-small is-outlined" v-on:click='removeFolder(props.row)' :title="$t('remove folder')">
              <b-icon pack="mdi" icon="close-circle" size="is-small"></b-icon>
            </button>
          </b-field>
        </b-table-column>
        <template #footer>
          <td></td>
          <td></td>
          <td></td>
          <td>{{ total.files }}</td>
          <td>{{ total.unmatched }}</td>
          <td>{{ prettyBytes(total.size) }}</td>
          <td></td>
          <td></td>
        </template>
      </b-table>
    </div>
    <div v-else>
      <section class="hero">
        <div class="hero-body">
          <div class="container has-text-centered">
            <h1 class="title">
              <span class="icon">
                <b-icon pack="mdi" icon="folder-outline" size="is-large"></b-icon>
              </span>
            </h1>
            <h2 class="subtitle">
              {{ $t('Add folders with VR videos') }}
            </h2>
          </div>
        </div>
      </section>
    </div>

    <hr/>

    <div class="columns">
      <div class="column">
        <h4 class="options-subtitle">{{ $t('Add local folder') }}</h4>
        <div class="field">
          <label class="label">{{ $t('Path to folder with content') }}</label>
          <div class="control">
            <input class="input" type="text" v-model='newVolumePath'>
          </div>
        </div>
        <div class="control">
          <button class="button is-link" v-on:click='addFolder'>{{ $t('Add new folder') }}</button>
        </div>
      </div>
      <div class="column">
        <h4 class="options-subtitle">{{ $t('Add cloud storage') }}</h4>
        <b-field grouped>
          <b-field :label="$t('Service')">
            <b-select :placeholder="$t('Select one')" v-model="serviceSelected">
              <option v-for="option in serviceOpts" :value="option.id" :key="option.id">
                {{ option.name }}
              </option>
            </b-select>
          </b-field>
          <b-field :label="$t('Token')" expanded>
            <b-input v-model='serviceToken'/>
          </b-field>
        </b-field>
        <div class="control">
          <button class="button is-link" v-on:click='addCloudStorage'
                  :disabled="serviceSelected === null || serviceToken === ''">{{ $t('Add service') }}
          </button>
        </div>
      </div>
    </div>

    <hr/>

  <div>
    <h4 class="options-subtitle">{{ $t('Options') }}</h4>
    <b-field>
      <b-switch v-model="match_ohash" type="is-default">
        {{ $t('Match StashDB Hashes') }}
      </b-switch>
    </b-field>

    <hr/>

    <b-field :label="$t('Video File Extensions')">
      <b-tooltip :label="$t('Only add video file extensions!')" position="is-top" style="width: 100%;">
        <b-taginput
            ref="videoExtInput"
            v-model="video_ext"
            :allow-new="true"
            @add="OnExtAdded"
            @remove="saveExtensions"
            :placeholder="$t('Add a video extension (e.g. .mp4)')">
        </b-taginput>
      </b-tooltip>
    </b-field>
    <b-field>
      <b-tooltip :label="$t('This only resets the Video File Extensions list.')" position="is-right">
        <b-button type="is-warning" @click="resetToDefaults">{{ $t('Reset') }}</b-button>
      </b-tooltip>
    </b-field>
  </div>
  </div>
  </OptionsSection>
</template>

<script>
import OptionsSection from '../OptionsSection.vue'
import ky from 'ky'
import prettyBytes from 'pretty-bytes'
import { parseISO } from 'date-fns'
import { relativeTime } from '@/relativeTime'

export default {
  name: 'Storage',
  components: { OptionsSection },
  data () {
    return {
      volumes: [],
      serviceOpts: [{ name: 'Put.io', id: 'putio' }],
      serviceToken: '',
      serviceSelected: null,
      newVolumePath: '',
      prettyBytes,
      parseISO,
      relativeTime,
      lastAddedTag: null,
      lastAddedTime: 0
    }
  },
  mounted () {
    this.$store.dispatch('optionsStorage/load')
  },
  methods: {
    async taskRescan () {
      try {
        await this.$store.dispatch('optionsStorage/save');
        await ky.get('/api/task/rescan');
      } catch {
        this.$buefy.dialog.alert({
          title: this.$t('Error'),
          message: this.$t('Failed to save options. Rescan was not started.'),
          type: 'is-danger',
          hasIcon: true,
          ariaRole: 'alertdialog',
          ariaModal: true
        });
      }
    },
    addFolder: async function () {
      await ky.post('/api/options/storage', { json: { path: this.newVolumePath, type: 'local' } })
    },
    addCloudStorage: async function () {
      await ky.post('/api/options/storage', { json: { token: this.serviceToken, type: this.serviceSelected } })
    },
    removeFolder: function (folder) {
      this.$buefy.dialog.confirm({
        title: this.$t('Remove folder'),
        message: this.$t("You're about to remove storage location {path} and its files from local database - files will remain intact at the location.", { path: `<strong>${folder.path}</strong>` }),
        type: 'is-danger',
        hasIcon: true,
        onConfirm: function () {
          ky.delete(`/api/options/storage/${folder.id}`)
        }
      })
    },
    rescanFolder: function (folder) {
      ky.get(`/api/task/rescan/${folder.id}`)
    },
    saveExtensions () {
      this.$store.dispatch('optionsStorage/save')
    },
    OnExtAdded(tag) {
      // Debounce the add event as it also triggers on blur
      const now = Date.now();
      if (this.lastAddedTag === tag && now - this.lastAddedTime < 200) {
        return;
      }
      this.lastAddedTag = tag;
      this.lastAddedTime = now;

      const index = this.video_ext.indexOf(tag);
      if (index > -1) {
        this.video_ext.splice(index, 1);
      }

      let cleanTag = tag.trim().toLowerCase();

      // Reject if it contains whitespace in the middle
      if (/\s/.test(cleanTag)) {
        this.$buefy.dialog.alert({
          title: this.$t('Invalid Extension'),
          message: this.$t('File extensions cannot contain whitespace.'),
          type: 'is-danger',
          hasIcon: true,
          ariaRole: 'alertdialog',
          ariaModal: true
        });
        if (this.$refs.videoExtInput) {
          this.$refs.videoExtInput.newTag = '';
        }
        return;
      }

      if (cleanTag.length === 0 || cleanTag === '.') {
        return;
      }

      if (!cleanTag.startsWith('.')) {
        cleanTag = '.' + cleanTag;
      }

      if (this.forbidden_video_ext && this.forbidden_video_ext.includes(cleanTag)) {
        this.$buefy.dialog.alert({
          title: this.$t('Invalid Extension'),
          message: this.$t('The file extension {ext} is a reserved extension and cannot be added.', { ext: `<strong>${cleanTag}</strong>` }),
          type: 'is-danger',
          hasIcon: true,
          ariaRole: 'alertdialog',
          ariaModal: true
        });
        if (this.$refs.videoExtInput) {
          this.$refs.videoExtInput.newTag = '';
        }
        return;
      }

      if (!/^\.[a-z0-9]+$/.test(cleanTag)) {
        this.$buefy.dialog.alert({
          title: this.$t('Invalid Extension'),
          message: this.$t('File extensions must only contain letters and numbers after the dot.'),
          type: 'is-danger',
          hasIcon: true,
          ariaRole: 'alertdialog',
          ariaModal: true
        });
        if (this.$refs.videoExtInput) {
          this.$refs.videoExtInput.newTag = '';
        }
        return;
      }

      if (!this.video_ext.includes(cleanTag)) {
        this.video_ext.push(cleanTag);
        this.saveExtensions();
      }
    },
    resetToDefaults () {
      this.video_ext = this.default_video_ext.slice()
      this.saveExtensions()
    },
  },
  computed: {
    match_ohash: {
      get () {        
        return this.$store.state.optionsStorage.options.match_ohash
      },
      set (value) {
        this.$store.state.optionsStorage.options.match_ohash = value
      },
    },
    total () {
      let files = 0; let unmatched = 0; let size = 0
      this.$store.state.optionsStorage.items.map(v => {
        files = files + v.file_count
        unmatched = unmatched + v.unmatched_count
        size = size + v.total_size
      })
      return { files, unmatched, size }
    },
    items () {
      return this.$store.state.optionsStorage.items
    },
    video_ext: {
      get () {return this.$store.state.optionsStorage.options.video_ext},
      set (value) {this.$store.state.optionsStorage.options.video_ext = value},
    },
    forbidden_video_ext: {
      get () {return this.$store.state.optionsStorage.options.forbidden_video_ext}
    },
    default_video_ext: {
      get () {return this.$store.state.optionsStorage.options.default_video_ext}
    },
  }
}
</script>
