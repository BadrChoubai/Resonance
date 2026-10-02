<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { getMe, LOGIN_URL, type Me } from './api'

const loading = ref(true)
const me = ref<Me | null>(null)
const error = ref<string>()
const loginFailed = new URLSearchParams(location.search).get('login') === 'failed'

onMounted(async () => {
  try {
    me.value = await getMe()
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e)
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <main>
    <h1>Résonance</h1>
    <p v-if="loading">Loading…</p>
    <p v-else-if="error" class="error">Something went wrong: {{ error }}</p>
    <p v-else-if="me">Logged in as <strong>{{ me.displayName }}</strong></p>
    <template v-else>
      <p v-if="loginFailed" class="error">Login didn't work. Try again.</p>
      <a class="button" :href="LOGIN_URL">Log in with Spotify</a>
    </template>
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

.button {
  display: inline-block;
  padding: 0.6rem 1.2rem;
  border-radius: 999px;
  background: #1db954;
  color: #000;
  font-weight: 600;
  text-decoration: none;
}
</style>
