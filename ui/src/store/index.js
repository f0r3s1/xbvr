import { createStore } from 'vuex'

import sceneList from './sceneList'
import actorList from './actorList'
import messages from './messages'
import overlay from './overlay'
import files from './files'
import remote from './remote'
import optionsStorage from './optionsStorage'
import optionsWeb from './optionsWeb'
import optionsDLNA from './optionsDLNA'
import optionsDeoVR from './optionsDeoVR'
import optionsSites from './optionsSites'
import optionsPreviews from './optionsPreviews'
import optionsFunscripts from './optionsFunscripts'
import optionsVendor from './optionsVendor'
import optionsAdvanced from './optionsAdvanced'
import optionsSceneCreate from './optionsSceneCreate'
import health from './health'

export default createStore({
  modules: {
    sceneList,
    actorList,
    messages,
    overlay,
    files,
    remote,
    optionsStorage,
    optionsDLNA,
    optionsDeoVR,
    optionsWeb,
    optionsSites,
    optionsPreviews,
    optionsFunscripts,
    optionsVendor,
    optionsAdvanced,
    optionsSceneCreate,
    health,
  }
})
