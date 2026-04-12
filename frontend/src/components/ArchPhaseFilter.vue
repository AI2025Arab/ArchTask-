<!-- ArchTask Addition - DO NOT MERGE INTO VIKUNJA CORE -->
<template>
  <div class="arch-phase-filter">
    <button 
      v-for="phase in phases" 
      :key="phase.value"
      class="filter-btn"
      :class="{ active: currentPhase === phase.value }"
      @click="setPhase(phase.value)"
    >
      {{ phase.label }}
    </button>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'

const router = useRouter()
const route = useRoute()

const phases = [
  { label: 'الكل', value: '' },
  { label: 'SD', value: 'SD' },
  { label: 'DD', value: 'DD' },
  { label: 'CD', value: 'CD' },
  { label: 'CA', value: 'CA' },
  { label: 'Academic', value: 'Academic' }
]

const currentPhase = ref(route.query.arch_phase || '')

const setPhase = (phase: string) => {
  currentPhase.value = phase
  const query = { ...route.query }
  if (phase) {
    query.arch_phase = phase
  } else {
    delete query.arch_phase
  }
  router.push({ query })
}
</script>

<style scoped>
.arch-phase-filter {
  display: flex;
  gap: 10px;
  margin-bottom: 20px;
  justify-content: center;
  flex-wrap: wrap;
}
.filter-btn {
  padding: 8px 16px;
  border-radius: 20px;
  border: 1px solid rgba(255, 255, 255, 0.2);
  background: transparent;
  color: inherit;
  cursor: pointer;
  font-weight: bold;
  transition: all 0.2s ease;
}
.filter-btn:hover {
  background: rgba(255, 255, 255, 0.1);
}
.filter-btn.active {
  background: #4caf50;
  color: white;
  border-color: #4caf50;
}
</style>
