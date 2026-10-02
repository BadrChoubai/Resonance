<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { getHello } from './api'

const message = ref<string>()
const error = ref<string>()

onMounted(async () => {
  try {
    message.value = (await getHello()).message
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e)
  }
})
</script>

<template>
  <main>
    <h1>Résonance</h1>
    <p v-if="message">{{ message }}</p>
    <p v-else-if="error" class="error">API unreachable: {{ error }}</p>
    <p v-else>Loading…</p>
  </main>
</template>

<style scoped>
main {
  max-width: 40rem;
  margin: 0 auto;
  padding: 2rem 1rem;
}

.error {
  color: #c0392b;
}
</style>
