<template>
  <OptionsSection :title="$t('Task Schedules')" :loading="isLoading" :value="schedules" :save-label="$t('Save settings')"
                  :saved-note="$t('Saved, schedules are active now')" @save="saveSettings" v-slot="{ dirty }">
      <div class="schedule-list">
        <div v-for="task in tasks" :key="task.key" class="schedule-row"
             :class="{ 'is-off': !schedules[task.key].enabled, 'is-open': open === task.key }">
          <div class="schedule-head">
            <b-tooltip :active="isLocked(task)" :label="$t('Enter a StashApi key to enable')" position="is-right">
              <b-switch v-model="schedules[task.key].enabled" :disabled="isLocked(task)" size="is-small"
                       />
            </b-tooltip>
            <div class="schedule-title">
              <strong>{{ $t(task.name) }}</strong>
              <span class="schedule-summary">{{ summary(task) }}</span>
            </div>
            <span class="schedule-next" :title="nextRuns[task.key]">{{ dirty ? '' : nextLabel(task) }}</span>
            <b-button size="is-small" type="is-text" class="schedule-toggle"
                      :icon-right="open === task.key ? 'chevron-up' : 'chevron-down'"
                      :aria-label="$t('Edit')" @click="open = open === task.key ? null : task.key"/>
          </div>

          <div v-if="open === task.key" class="schedule-body">
            <p v-if="task.note" class="schedule-note">{{ $t(task.note) }}</p>
            <div class="schedule-grid">
              <label>{{ $t('Every') }}</label>
              <b-select size="is-small" v-model.number="schedules[task.key].hourInterval">
                <option v-for="step in intervalOptions(task.key)" :key="step.hours" :value="step.hours">{{ step.label }}</option>
              </b-select>

              <label>{{ $t('Time of day') }}</label>
              <div class="schedule-inline">
                <b-switch size="is-small" v-model="schedules[task.key].useRange" @update:modelValue="onRangeToggle(task.key)">
                  {{ schedules[task.key].useRange ? $t('Only between') : $t('Any time') }}
                </b-switch>
                <template v-if="schedules[task.key].useRange">
                  <b-select size="is-small" v-model.number="schedules[task.key].hourStart">
                    <option v-for="h in 24" :key="h" :value="h - 1">{{ hourLabel(h - 1) }}</option>
                  </b-select>
                  <span>{{ $t('and') }}</span>
                  <b-select size="is-small" v-model.number="schedules[task.key].hourEnd">
                    <option v-for="h in 24" :key="h" :value="h - 1">{{ hourLabel(h - 1) }}</option>
                  </b-select>
                  <span class="has-text-grey" v-if="schedules[task.key].hourStart > schedules[task.key].hourEnd">{{ $t('(next day)') }}</span>
                </template>
              </div>

              <template v-if="schedules[task.key].useRange">
                <label>{{ $t('At minute') }}</label>
                <b-numberinput size="is-small" controls-position="compact" :min="0" :max="59"
                               v-model="schedules[task.key].minuteStart"/>
              </template>

              <label>{{ $t('After startup') }}</label>
              <b-select size="is-small" v-model.number="schedules[task.key].runAtStartDelay">
                <option v-for="d in startDelays(schedules[task.key].runAtStartDelay)" :key="d" :value="d">
                  {{ d === 0 ? $t("Don't run") : $t('Run after {n} min', { n: d }) }}
                </option>
              </b-select>
            </div>
          </div>
        </div>
      </div>

  </OptionsSection>
</template>

<script>
import { defineComponent } from 'vue'
import ky from 'ky'
import { relativeTime } from '@/relativeTime'
import OptionsSection from '../OptionsSection.vue'

// One entry per config.Cron schedule; keys match the Go json keys.
const TASKS = [
  { key: 'rescrapeSchedule', name: 'Rescrape sites' },
  { key: 'rescanSchedule', name: 'Rescan folders' },
  { key: 'previewSchedule', name: 'Preview generation', note: 'CPU heavy. With a time window, no new preview starts after the window ends.' },
  { key: 'actorRescrapeSchedule', name: 'Actor rescrape' },
  { key: 'stashdbRescrapeSchedule', name: 'StashDB rescrape', needsStashKey: true },
  { key: 'linkScenesSchedule', name: 'Link scenes' },
  { key: 'libraryHealthSchedule', name: 'Library health', note: 'Low power background check; it skips a round while other jobs are busy.' }
]

const INTERVALS = [1, 2, 3, 4, 6, 8, 12, 24, 48, 72, 120, 168, 336, 720]

const DELAYS = [0, 1, 2, 5, 10, 15, 30, 60]

export default defineComponent({
  name: 'Schedules',
  components: { OptionsSection },

  data () {
    return {
      isLoading: true,
      open: null,
      tasks: TASKS,
      schedules: Object.fromEntries(TASKS.map(t => [t.key, { enabled: false, hourInterval: 12, useRange: false, minuteStart: 0, hourStart: 0, hourEnd: 23, runAtStartDelay: 0 }])),
      nextRuns: {}
    }
  },

  computed: {
    stashApiKey () {
      return this.$store.state.optionsAdvanced.advanced.stashApiKey
    }
  },

  async mounted () {
    this.$store.dispatch('optionsAdvanced/load')
    await this.loadState()
  },

  methods: {
    apply (data) {
      for (const t of TASKS) {
        if (data.schedules[t.key]) this.schedules[t.key] = { ...data.schedules[t.key] }
      }
      this.nextRuns = data.nextRuns || {}
    },
    async loadState () {
      this.isLoading = true
      try {
        this.apply(await ky.get('/api/options/task-schedule').json())
      } finally {
        this.isLoading = false
      }
    },
    async saveSettings () {
      this.apply(await ky.post('/api/options/task-schedule', { json: this.schedules }).json())
    },
    onRangeToggle (key) {
      // cron steps stay within a day when a time window is set
      const s = this.schedules[key]
      if (s.useRange && s.hourInterval > 23) s.hourInterval = 23
    },
    isLocked (task) {
      return !!task.needsStashKey && !this.stashApiKey && !this.schedules[task.key].enabled
    },
    intervalOptions (key) {
      const s = this.schedules[key]
      let opts = (s.useRange ? [...Array(23)].map((_, i) => i + 1) : INTERVALS).map(h => ({ hours: h, label: this.intervalLabel(h) }))
      if (!opts.some(o => o.hours === s.hourInterval)) {
        // keep values saved by older versions selectable
        opts = [...opts, { hours: s.hourInterval, label: this.intervalLabel(s.hourInterval) }].sort((a, b) => a.hours - b.hours)
      }
      return opts
    },
    startDelays (current) {
      return DELAYS.includes(current) ? DELAYS : [...DELAYS, current].sort((a, b) => a - b)
    },
    hourLabel (h) {
      return `${String(h).padStart(2, '0')}:00`
    },
    intervalLabel (hours) {
      if (hours === 168) return this.$t('1 week')
      if (hours === 336) return this.$t('2 weeks')
      if (INTERVALS.includes(hours) && hours >= 24) return this.$t('{n} day | {n} days', { n: hours / 24 }, hours / 24)
      return this.$t('{n} hour | {n} hours', { n: hours }, hours)
    },
    summary (task) {
      const s = this.schedules[task.key]
      if (!s.enabled) return this.$t('Off')
      let out = this.$t('every {interval}', { interval: this.intervalLabel(s.hourInterval) })
      if (s.useRange) out += this.$t(', {from} to {to}', { from: this.hourLabel(s.hourStart), to: this.hourLabel(s.hourEnd) })
      if (s.runAtStartDelay > 0) out += this.$t(', {n} min after start', { n: s.runAtStartDelay })
      return out
    },
    nextLabel (task) {
      const next = this.nextRuns[task.key]
      if (!next || !this.schedules[task.key].enabled) return ''
      return this.$t('next {time}', { time: relativeTime(next) })
    }
  }
})
</script>

<style scoped>
.schedule-list {
  border: 1px solid var(--border-color, rgba(128, 128, 128, 0.25));
  border-radius: 8px;
  overflow: hidden;
}
.schedule-row + .schedule-row {
  border-top: 1px solid var(--border-color, rgba(128, 128, 128, 0.25));
}
.schedule-head {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.45rem 0.75rem;
}
.schedule-head :deep(.switch) {
  margin-right: 0;
}
.schedule-title {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  column-gap: 0.6rem;
}
.schedule-summary,
.schedule-next {
  font-size: 0.8rem;
  opacity: 0.65;
}
.schedule-next {
  white-space: nowrap;
}
.is-off .schedule-title strong {
  opacity: 0.55;
}
.schedule-body {
  padding: 0.25rem 0.75rem 0.75rem 3.4rem;
}
.schedule-note {
  font-size: 0.8rem;
  opacity: 0.7;
  margin-bottom: 0.5rem !important;
}
.schedule-grid {
  display: grid;
  grid-template-columns: max-content 1fr;
  gap: 0.4rem 0.9rem;
  align-items: center;
}
.schedule-grid > label {
  font-size: 0.85rem;
  opacity: 0.8;
}
.schedule-grid :deep(.b-numberinput) {
  max-width: 8rem;
}
.schedule-inline {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.5rem;
}
@media (max-width: 600px) {
  .schedule-next { display: none; }
  .schedule-body { padding-left: 0.75rem; }
}
</style>
