<template>
  <OptionsSection :title="$t('DLNA interface')" :loading="isLoading" :value="editable" :save-label="$t('Save and apply changes')" @save="save"
                  :description="$t('Standard protocol that works with players such as: Skybox, Pigasus, Mobile Station VR, and others.') + ' ' + $t('Since it is broadcasted accross the whole local network, you might want to restrict access to selected IP addresses or disable it completely.')">
    <b-field label="DLNA server">
      <b-switch v-model="dlna.enabled">Enabled</b-switch>
    </b-field>

    <b-field label="Visible name">
      <b-input v-model="dlna.name" class="input-short"></b-input>
    </b-field>

    <b-field label="Icon" grouped>
      <b-select placeholder="Select image" v-model="dlna.image">
        <option v-for="s in dlna.availableImages" :value="s" :key="s">{{ s }}</option>
      </b-select>
      <img :src="`/ui/dlna/${dlna.image}.png`" width="64" class="dlna-icon" v-if="dlna.image"/>
    </b-field>

    <b-field label="Allowed IP addresses">
      <b-taginput v-model="dlna.allowedIp" :allow-new="true" placeholder="Type in a IP address"></b-taginput>
    </b-field>

    <p v-if="!isLoading" class="recent-ip">
      Recent IP addresses:
      <template v-if="dlna.recentIp.length">
        <b-tag rounded v-for="ip in dlna.recentIp" :key="ip" type="is-info" class="recent-ip-tag" @click="addIP(ip)">{{ ip }}</b-tag>
      </template>
      <span v-else>none (connect to DLNA at least once to find out device's IP address)</span>
    </p>
  </OptionsSection>
</template>

<script>
import { defineComponent } from 'vue'
import OptionsSection from '../OptionsSection.vue'

export default defineComponent({
  name: 'InterfaceDLNA',
  components: { OptionsSection },

  mounted () {
    this.$store.dispatch('optionsDLNA/load')
  },

  computed: {
    dlna () {
      return this.$store.state.optionsDLNA.dlna
    },
    editable () {
      const { enabled, name, image, allowedIp } = this.dlna
      return { enabled, name, image, allowedIp }
    },
    isLoading () {
      return this.$store.state.optionsDLNA.loading
    }
  },

  methods: {
    save () {
      return this.$store.dispatch('optionsDLNA/save')
    },
    addIP (ip) {
      if (!this.dlna.allowedIp.includes(ip)) this.dlna.allowedIp = [...this.dlna.allowedIp, ip]
    }
  }
})
</script>

<style scoped>
.input-short {
  max-width: 240px;
}
.dlna-icon {
  margin-left: 1rem;
}
.recent-ip-tag {
  margin-right: 0.25em;
  cursor: pointer;
}
</style>
