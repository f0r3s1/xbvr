<template>
  <OptionsSection :title="$t('Library Health')">
    <div class="content">

      <div v-if="!report && !isRunning">
        <p>
          {{ $t('Your library is checked automatically in the background once a week (change or turn it off in Task Schedules) for data integrity issues, duplicate actors, orphaned records, missing files and stale metadata.') }}
        </p>
        <b-button type="is-primary" @click="startScan">
          {{ $t('Check now') }}
        </b-button>
      </div>

      <div v-if="isRunning" class="progress-section">
        <div style="margin-bottom: 0.5rem;">
          <span v-if="progress.background" class="tag is-success is-light is-small" style="margin-right: 0.4rem;"
                :title="$t('Throttled to about 20% of one CPU core')">{{ $t('Background · low power') }}</span>
          <strong>{{ progress.step }}</strong>
          <span class="has-text-grey" style="margin-left: 0.5rem;">
            {{ $t('Step {n} of {total}', { n: progress.step_num, total: progress.total_steps }) }}
          </span>
        </div>
        <b-progress :modelValue="progress.percent" type="is-primary" size="is-medium" show-value>
          {{ Math.round(progress.percent) }}%
        </b-progress>
        <b-button type="is-danger" outlined size="is-small" @click="cancelScan" style="margin-top: 0.5rem;">
          {{ $t('Cancel') }}
        </b-button>
      </div>

      <div v-if="report && !isRunning">
        <div class="columns is-multiline" style="margin-bottom: 1rem;">
          <div class="column is-narrow" v-for="(val, key) in statCards" :key="key">
            <div class="health-stat-card">
              <div class="stat-number">{{ val }}</div>
              <div class="stat-label">{{ key }}</div>
            </div>
          </div>
          <div class="column">
            <div class="is-pulled-right health-when">
              <span class="health-when-text" :title="report.generated_at">
                {{ $t(report.auto ? 'Checked automatically {time}' : 'Checked {time}', { time: checkedAgo }) }}
              </span>
              <b-button size="is-small" @click="startScan">{{ $t('Check now') }}</b-button>
            </div>
          </div>
        </div>

        <div style="margin-bottom: 1rem;">
          <span class="tag is-danger" v-if="report.summary.critical > 0" style="margin-right: 0.4rem;">
            {{ $t('{n} critical', { n: report.summary.critical }) }}
          </span>
          <span class="tag is-warning" v-if="report.summary.warning > 0" style="margin-right: 0.4rem;">
            {{ $t('{n} warning | {n} warnings', { n: report.summary.warning }, report.summary.warning) }}
          </span>
          <span class="tag is-info" v-if="report.summary.info > 0" style="margin-right: 0.4rem;">
            {{ $t('{n} info', { n: report.summary.info }) }}
          </span>
          <span class="tag is-success" v-if="totalIssues === 0">
            {{ $t('All clear') }}
          </span>
        </div>

        <DuplicateActors v-if="showDuplicates" ref="dups" class="health-dups"/>

        <div v-for="issue in sortedIssues" :key="issue.id" class="health-issue">
          <div class="health-issue-header" @click="toggle(issue.id)">
            <span class="health-issue-icon">
              <b-icon pack="mdi" :icon="severityIcon(issue.severity)" :type="severityType(issue.severity)" size="is-small"></b-icon>
            </span>
            <span class="health-issue-desc">
              <strong>{{ issue.description }}</strong>
              <span class="tag is-light is-small" style="margin-left: 0.5rem;">{{ issue.category }}</span>
            </span>
            <span class="health-issue-actions">
              <b-button v-if="issue.fixable" size="is-small" type="is-primary" outlined
                        @click.stop="confirmFix(issue)" :loading="fixingAction === issue.fix_action">
                {{ issue.fix_label }}
              </b-button>
              <b-icon pack="mdi" :icon="expanded[issue.id] ? 'chevron-up' : 'chevron-down'" size="is-small"
                      class="expand-icon" v-if="issue.affected_items && issue.affected_items.length > 0"></b-icon>
            </span>
          </div>
          <div class="health-issue-detail" v-if="issue.detail && expanded[issue.id]">
            {{ issue.detail }}
          </div>
          <div class="health-issue-items" v-if="expanded[issue.id] && issue.affected_items && issue.affected_items.length > 0">
            <table class="table is-narrow is-fullwidth is-size-7">
              <tbody>
                <tr v-for="(item, idx) in issue.affected_items" :key="idx">
                  <td class="has-text-grey" style="width: 50px;" v-if="item.id">{{ item.id }}</td>
                  <td>{{ item.label || '—' }}</td>
                  <td class="has-text-grey" v-if="item.extra">{{ item.extra }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </div>
  </OptionsSection>
</template>

<script>
import OptionsSection from '../OptionsSection.vue'
import ky from 'ky'
import { defineComponent } from 'vue';
import { parseISO } from 'date-fns'
import { relativeTime } from '@/relativeTime'
import DuplicateActors from './DuplicateActors.vue'

export default defineComponent({
  name: 'LibraryHealth',
  components: { OptionsSection, DuplicateActors },

  data () {
    return {
      report: null,
      fixingAction: null,
      showDuplicates: false,
      now: Date.now(),
      expanded: {}
    }
  },

  computed: {
    progress () {
      return this.$store.state.health.progress
    },
    isRunning () {
      return this.progress.running
    },
    totalIssues () {
      if (!this.report) return 0
      return this.report.issues.length
    },
    sortedIssues () {
      if (!this.report) return []
      const order = { critical: 0, warning: 1, info: 2 }
      return [...this.report.issues].sort((a, b) => order[a.severity] - order[b.severity])
    },
    checkedAgo () {
      if (!this.report || !this.report.generated_at) return ''
      void this.now // re-evaluated every minute
      try {
        // the server clock can be a little ahead of the browser: never say "in ..."
        const t = Math.min(parseISO(this.report.generated_at).getTime(), this.now)
        return relativeTime(t)
      } catch {
        return ''
      }
    },
    statCards () {
      if (!this.report) return {}
      return {
        Scenes: this.report.stats.total_scenes,
        Files: this.report.stats.total_files,
        Actors: this.report.stats.total_actors,
        Tags: this.report.stats.total_tags
      }
    }
  },

  watch: {
    'progress.running' (val, oldVal) {
      if (!val && oldVal && this.progress.step === 'done') {
        this.fetchReport()
      }
    }
  },

  async mounted () {
    await this.fetchReport()
    this._clock = setInterval(() => { this.now = Date.now() }, 60000)
  },

  beforeUnmount () {
    if (this._pollTimer) clearInterval(this._pollTimer)
    if (this._clock) clearInterval(this._clock)
  },

  methods: {
    async fetchReport () {
      try {
        const data = await ky.get('/api/health/report', { timeout: 10000 }).json()
        if (data.report) {
          this.report = data.report
          this.$store.commit('health/setReport', data.report)
        }
        if (data.running) {
          this.$store.commit('health/setProgress', { ...this.progress, running: true })
        }
      } catch { /* silent */ }
    },
    async startScan () {
      this.expanded = {}
      this.report = null
      try {
        await ky.post('/api/health/scan', { json: {}, timeout: 10000 }).json()
        this.pollForCompletion()
      } catch {
        this.$buefy.toast.open({ message: this.$t('Failed to start health check'), type: 'is-danger' })
      }
    },
    pollForCompletion () {
      if (this._pollTimer) clearInterval(this._pollTimer)
      this._pollTimer = setInterval(async () => {
        try {
          const data = await ky.get('/api/health/report', { timeout: 5000 }).json()
          if (data.running) {
            if (!this.progress.running) {
              this.$store.commit('health/setProgress', { ...this.progress, running: true, step: this.$t('Scanning...') })
            }
          } else if (data.report) {
            this.report = data.report
            this.$store.commit('health/setReport', data.report)
            this.$store.commit('health/setProgress', { running: false, step: 'done', step_num: 16, total_steps: 16, percent: 100 })
            clearInterval(this._pollTimer)
          }
        } catch { /* silent */ }
      }, 2000)
    },
    async cancelScan () {
      try {
        await ky.post('/api/health/cancel', { json: {}, timeout: 10000 }).json()
      } catch { /* silent */ }
    },
    toggle (id) {
      this.expanded[id] = !this.expanded[id]
    },
    confirmFix (issue) {
      // not a one-click fix: open the review list instead
      if (issue.fix_action === 'review-duplicate-actors') {
        this.showDuplicates = true
        this.$nextTick(() => this.$refs.dups && this.$refs.dups.$el.scrollIntoView({ behavior: 'smooth', block: 'start' }))
        return
      }
      const count = issue.affected_items ? issue.affected_items.length : 0
      this.$buefy.dialog.confirm({
        title: issue.fix_label,
        message: this.$t('This will run {action} to address {count} affected items.', { action: `<strong>${issue.fix_label}</strong>`, count: `<strong>${count}</strong>` }) + '<br><br>' +
                 (issue.detail ? `<em>${issue.detail}</em>` : ''),
        type: 'is-warning',
        hasIcon: true,
        onConfirm: () => this.fix(issue)
      })
    },
    async fix (issue) {
      this.fixingAction = issue.fix_action
      try {
        await ky.post('/api/health/fix', { json: { action: issue.fix_action }, timeout: 10000 }).json()
        this.$buefy.toast.open({ message: this.$t('{action} started — rescan when complete', { action: issue.fix_label }), type: 'is-success' })
      } catch {
        this.$buefy.toast.open({ message: this.$t('Fix failed'), type: 'is-danger' })
      }
      this.fixingAction = null
    },
    severityIcon (severity) {
      switch (severity) {
        case 'critical': return 'alert-circle'
        case 'warning': return 'alert'
        case 'info': return 'information'
        default: return 'information'
      }
    },
    severityType (severity) {
      switch (severity) {
        case 'critical': return 'is-danger'
        case 'warning': return 'is-warning'
        case 'info': return 'is-info'
        default: return 'is-info'
      }
    }
  },
});
</script>

<style scoped>
.health-when {
  display: flex;
  align-items: center;
  gap: 8px;
  padding-top: 0.5rem;
}
.health-when-text {
  font-size: 0.78rem;
  opacity: 0.65;
}
.health-dups {
  margin-bottom: 1rem;
}
.health-stat-card {
  text-align: center;
  padding: 0.5rem 1.2rem;
  border: 1px solid rgba(127, 127, 127, 0.25);
  border-radius: 8px;
  min-width: 80px;
}
.stat-number {
  font-size: 1.4rem;
  font-weight: 700;
  line-height: 1.2;
}
.stat-label {
  font-size: 0.75rem;
  opacity: 0.6;
  text-transform: uppercase;
}
.health-issue {
  border: 1px solid rgba(127, 127, 127, 0.25);
  border-radius: 8px;
  margin-bottom: 0.5rem;
  overflow: hidden;
}
.health-issue-header {
  display: flex;
  align-items: center;
  padding: 0.6rem 0.8rem;
  cursor: pointer;
  user-select: none;
}
.health-issue-header:hover {
  background: rgba(127, 127, 127, 0.08);
}
.health-issue-icon {
  flex-shrink: 0;
  margin-right: 0.6rem;
}
.health-issue-desc {
  flex: 1;
}
.health-issue-actions {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-shrink: 0;
}
.expand-icon {
  cursor: pointer;
  opacity: 0.5;
}
.health-issue-detail {
  padding: 0 0.8rem 0.4rem 2.4rem;
  font-size: 0.85rem;
  opacity: 0.7;
  font-style: italic;
}
.health-issue-items {
  border-top: 1px solid rgba(127, 127, 127, 0.2);
  max-height: 250px;
  overflow-y: auto;
}
.health-issue-items .table {
  margin-bottom: 0;
}
</style>
