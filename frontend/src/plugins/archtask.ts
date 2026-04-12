// ArchTask Addition - DO NOT MERGE INTO VIKUNJA CORE
import { defineAsyncComponent } from 'vue'

export default {
  name: 'ArchTaskPlugin',
  install(app: any, options: any) {
    // Register the AI Capture globally
    app.component('AICapture', defineAsyncComponent(() => import('../components/AICapture.vue')))
    app.component('ArchPhaseFilter', defineAsyncComponent(() => import('../components/ArchPhaseFilter.vue')))

    // Assuming Vikunja has a plugin event bus or slot registry we trigger on view load
    if (app.config.globalProperties.$pluginRegistry) {
      // Inject AI button into the global top bar or task list view
      app.config.globalProperties.$pluginRegistry.registerSlot('task-list-header', 'AICapture')
      // Inject Arch Filter
      app.config.globalProperties.$pluginRegistry.registerSlot('task-list-filters', 'ArchPhaseFilter')
    }
    
    console.log('🏗️ ArchTask AI Plugin Loaded Successfully! 🚀')
  }
}
