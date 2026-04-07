<template>
  <div class="container">
    <div class="content">
      <h3>{{ $t("Library Health") }}</h3>
      <hr/>

      <!-- No report yet, not running -->
      <div v-if="!report && !isRunning">
        <p>Scan your library for data integrity issues, orphaned records, missing files, and stale metadata.</p>
        <b-button type="is-primary" @click="startScan">
          Run Health Check
        </b-button>
      </div>

      <!-- Running: progress bar -->
      <div v-if="isRunning" class="progress-section">
        <div style="margin-bottom: 0.5rem;">
          <strong>{{ progress.step }}</strong>
          <span class="has-text-grey" style="margin-left: 0.5rem;">
            Step {{ progress.step_num }} of {{ progress.total_steps }}
          </span>
        </div>
        <b-progress :value="progress.percent" type="is-primary" size="is-medium" show-value>
          {{ Math.round(progress.percent) }}%
        </b-progress>
        <b-button type="is-danger" outlined size="is-small" @click="cancelScan" style="margin-top: 0.5rem;">
          Cancel
        </b-button>
      </div>

      <!-- Report ready -->
      <div v-if="report && !isRunning">
        <div class="columns is-multiline" style="margin-bottom: 1rem;">
          <div class="column is-narrow" v-for="(val, key) in statCards" :key="key">
            <div class="health-stat-card">
              <div class="stat-number">{{ val }}</div>
              <div class="stat-label">{{ key }}</div>
            </div>
          </div>
          <div class="column">
            <div class="is-pulled-right" style="padding-top: 0.5rem;">
              <span class="tag is-light is-small" style="margin-right: 0.3rem;">{{ report.duration }}</span>
              <b-button size="is-small" @click="startScan">Rescan</b-button>
            </div>
          </div>
        </div>

        <div style="margin-bottom: 1rem;">
          <span class="tag is-danger" v-if="report.summary.critical > 0" style="margin-right: 0.4rem;">
            {{ report.summary.critical }} critical
          </span>
          <span class="tag is-warning" v-if="report.summary.warning > 0" style="margin-right: 0.4rem;">
            {{ report.summary.warning }} warnings
          </span>
          <span class="tag is-info" v-if="report.summary.info > 0" style="margin-right: 0.4rem;">
            {{ report.summary.info }} info
          </span>
          <span class="tag is-success" v-if="totalIssues === 0">
            All clear
          </span>
        </div>

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
  </div>
</template>

<script>
import ky from 'ky'
import Vue from 'vue'

export default {
  name: 'LibraryHealth',
  data () {
    return {
      report: null,
      fixingAction: null,
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
    // When progress says done, fetch the report
    'progress.running' (val, oldVal) {
      if (!val && oldVal && this.progress.step === 'done') {
        this.fetchReport()
      }
    }
  },
  async mounted () {
    await this.fetchReport()
  },
  beforeDestroy () {
    if (this._pollTimer) clearInterval(this._pollTimer)
  },
  methods: {
    async fetchReport () {
      try {
        const data = await ky.get('/api/health/report', { timeout: 10000 }).json()
        if (data.report) {
          this.report = data.report
        }
        if (data.running) {
          this.$store.commit('health/setProgress', { ...this.progress, running: true })
        }
      } catch (e) {
        // silent
      }
    },
    async startScan () {
      this.expanded = {}
      this.report = null
      try {
        await ky.post('/api/health/scan', { json: {}, timeout: 10000 }).json()
        // Poll as fallback in case WebSocket progress doesn't arrive
        this.pollForCompletion()
      } catch (e) {
        this.$buefy.toast.open({ message: 'Failed to start health check', type: 'is-danger' })
      }
    },
    pollForCompletion () {
      if (this._pollTimer) clearInterval(this._pollTimer)
      this._pollTimer = setInterval(async () => {
        try {
          const data = await ky.get('/api/health/report', { timeout: 5000 }).json()
          if (data.running) {
            // Update progress from polling if WS isn't delivering
            if (!this.progress.running) {
              this.$store.commit('health/setProgress', { ...this.progress, running: true, step: 'Scanning...' })
            }
          } else if (data.report) {
            this.report = data.report
            this.$store.commit('health/setProgress', { running: false, step: 'done', step_num: 15, total_steps: 15, percent: 100 })
            clearInterval(this._pollTimer)
          }
        } catch (e) { /* silent */ }
      }, 2000)
    },
    async cancelScan () {
      try {
        await ky.post('/api/health/cancel', { json: {}, timeout: 10000 }).json()
      } catch (e) {
        // silent
      }
    },
    toggle (id) {
      Vue.set(this.expanded, id, !this.expanded[id])
    },
    confirmFix (issue) {
      const count = issue.affected_items ? issue.affected_items.length : 0
      this.$buefy.dialog.confirm({
        title: issue.fix_label,
        message: `This will run <strong>${issue.fix_label}</strong> to address <strong>${count}</strong> affected items.<br><br>` +
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
        this.$buefy.toast.open({ message: `${issue.fix_label} started — rescan when complete`, type: 'is-success' })
      } catch (e) {
        this.$buefy.toast.open({ message: 'Fix failed', type: 'is-danger' })
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
  }
}
</script>

<style scoped>
.progress-section {
  max-width: 500px;
}
.health-stat-card {
  text-align: center;
  padding: 0.5rem 1.2rem;
  border: 1px solid #dbdbdb;
  border-radius: 4px;
  min-width: 80px;
}
.stat-number {
  font-size: 1.4rem;
  font-weight: 700;
  line-height: 1.2;
}
.stat-label {
  font-size: 0.75rem;
  color: #888;
  text-transform: uppercase;
}
.health-issue {
  border: 1px solid #dbdbdb;
  border-radius: 4px;
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
  background: #f5f5f5;
}

html[data-theme="dark"] .health-issue-header:hover {
  background: #222230 !important;
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
  color: #666;
  font-style: italic;
}
.health-issue-items {
  border-top: 1px solid #eee;
  max-height: 250px;
  overflow-y: auto;
}
.health-issue-items .table {
  margin-bottom: 0;
}
</style>
