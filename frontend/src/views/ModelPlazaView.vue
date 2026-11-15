<template>
  <!-- 后台内嵌形态:?embedded=1 且已登录,套完整后台布局 -->
  <AppLayout v-if="isEmbedded">
    <ModelPlazaContent :response="data" :loading="loading" :error="loadFailed" embedded />
  </AppLayout>

  <!-- 独立形态使用与首页一致的公开站点导航。 -->
  <div v-else class="public-model-plaza">
    <PublicSiteHeader />
    <main>
      <ModelPlazaContent :response="data" :loading="loading" :error="loadFailed" />
    </main>
    <PublicSiteFooter />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import PublicSiteHeader from '@/components/public/PublicSiteHeader.vue'
import PublicSiteFooter from '@/components/public/PublicSiteFooter.vue'
import ModelPlazaContent from '@/components/modelPlaza/ModelPlazaContent.vue'
import { getModelPlaza, type ModelPlazaResponse } from '@/api/modelPlaza'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'

const route = useRoute()
const appStore = useAppStore()
const authStore = useAuthStore()

// embedded=1 但未登录(如转发的链接)自动降级为独立形态。
const isEmbedded = computed(() => route.query.embedded === '1' && authStore.isAuthenticated)

const data = ref<ModelPlazaResponse | null>(null)
const loading = ref(true)
const loadFailed = ref(false)

onMounted(async () => {
  // 独立形态导航条需要站点名/Logo;有 __APP_CONFIG__ 注入时同步命中缓存。
  void appStore.fetchPublicSettings()
  try {
    data.value = await getModelPlaza()
  } catch {
    loadFailed.value = true
  } finally {
    loading.value = false
  }
})
</script>

<style scoped>
.public-model-plaza {
  display: flex;
  flex-direction: column;
  min-height: 100vh;
  background: #f5f5f7;
  color: #1d1d1f;
}

.public-model-plaza > main { flex: 1; }

.dark .public-model-plaza {
  background: #101011;
  color: #f5f5f7;
}
</style>
