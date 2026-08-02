# Pilot — Frontend Design

**Framework**: Nuxt 3 + Vue 3 + TypeScript  
**Styling**: Tailwind CSS  
**State**: Pinia  
**Status**: 🟡 Design Ready  

---

## Project Structure

```
pilot-frontend/
├── components/
│   ├── common/
│   │   ├── Header.vue
│   │   ├── Sidebar.vue
│   │   ├── LoadingSpinner.vue
│   │   └── ErrorAlert.vue
│   ├── dashboard/
│   │   ├── EarningsCard.vue
│   │   ├── GoalProgress.vue
│   │   ├── RecentTrips.vue
│   │   └── StatsGrid.vue
│   ├── trips/
│   │   ├── TripsTable.vue
│   │   ├── TripFilters.vue
│   │   └── TripDetails.vue
│   ├── analytics/
│   │   ├── EarningsChart.vue
│   │   └── PerformanceMetrics.vue
│   ├── goals/
│   │   ├── GoalForm.vue
│   │   └── GoalHistory.vue
│   └── profile/
│       ├── ProfileForm.vue
│       └── ProfileStats.vue
├── pages/
│   ├── index.vue (Dashboard)
│   ├── login.vue (OAuth)
│   ├── analytics.vue
│   ├── trips.vue
│   ├── goals.vue
│   ├── profile.vue
│   └── 404.vue
├── composables/
│   ├── useAuth.ts
│   ├── useTrips.ts
│   ├── useStats.ts
│   ├── useGoals.ts
│   └── useNotification.ts
├── stores/
│   ├── auth.ts (Pinia)
│   ├── driver.ts
│   ├── trips.ts
│   ├── stats.ts
│   ├── goals.ts
│   └── ui.ts
├── utils/
│   ├── api.ts (API client)
│   ├── format.ts (Formatação)
│   ├── validation.ts
│   └── constants.ts
├── app.vue
├── nuxt.config.ts
├── package.json
└── .env.example
```

---

## Pages

### pages/index.vue (Dashboard)

```vue
<template>
  <div class="min-h-screen bg-gray-50">
    <Header />
    <div class="container mx-auto px-4 py-8">
      <!-- Earnings Card -->
      <EarningsCard 
        :loading="statsLoading"
        :total-earned="stats?.total_earned || 0"
        :total-trips="stats?.total_trips || 0"
        :earnings-per-hour="stats?.earnings_per_hour || 0"
      />
      
      <!-- Goal Progress -->
      <GoalProgress 
        :goal="goalProgress"
        @update="setNewGoal"
      />
      
      <!-- Stats Grid -->
      <StatsGrid :stats="stats" />
      
      <!-- Recent Trips -->
      <RecentTrips 
        :trips="recentTrips"
        @refresh="syncTrips"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useStats } from '~/composables/useStats'
import { useTrips } from '~/composables/useTrips'
import { useAuth } from '~/composables/useAuth'

const { isAuthenticated } = useAuth()
const { stats, goalProgress, fetchDailyStats, fetchGoalProgress } = useStats()
const { recentTrips, fetchRecentTrips, syncTrips } = useTrips()

const statsLoading = ref(false)

onMounted(async () => {
  if (!isAuthenticated.value) navigateTo('/login')
  
  statsLoading.value = true
  await Promise.all([
    fetchDailyStats(),
    fetchGoalProgress(),
    fetchRecentTrips(5)
  ])
  statsLoading.value = false
  
  setInterval(fetchDailyStats, 30000)
})

const setNewGoal = async (amount: number) => {
  await useGoals().setDailyGoal(amount)
  await fetchGoalProgress()
}
</script>
```

### pages/login.vue

```vue
<template>
  <div class="min-h-screen bg-gradient-to-br from-indigo-600 to-blue-800 flex items-center justify-center">
    <div class="bg-white rounded-lg shadow-xl p-8 w-96">
      <img src="~/assets/logo.svg" alt="Pilot" class="h-12 mb-6 mx-auto" />
      
      <h1 class="text-2xl font-bold text-center mb-8">Pilot</h1>
      
      <button 
        @click="loginWithUber"
        class="w-full px-4 py-3 bg-black text-white rounded-lg hover:bg-gray-900"
      >
        Continuar com Uber
      </button>
      
      <p v-if="error" class="mt-4 p-3 bg-red-50 text-red-800 rounded-lg">
        {{ error }}
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useAuth } from '~/composables/useAuth'

const { loginWithUber, error } = useAuth()
</script>
```

### pages/analytics.vue

```vue
<template>
  <div class="min-h-screen bg-gray-50">
    <Header />
    <div class="container mx-auto px-4 py-8">
      <h1 class="text-3xl font-bold mb-8">Analytics</h1>
      
      <!-- Tabs -->
      <div class="flex gap-4 mb-6">
        <button 
          @click="selectedTab = 'week'"
          :class="{ 'border-b-2 border-indigo-600': selectedTab === 'week' }"
        >
          Semana
        </button>
        <button 
          @click="selectedTab = 'month'"
          :class="{ 'border-b-2 border-indigo-600': selectedTab === 'month' }"
        >
          Mês
        </button>
      </div>
      
      <!-- Charts -->
      <EarningsChart v-if="selectedTab === 'week'" :data="weeklyStats" />
      <EarningsChart v-else :data="monthlyStats" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useStats } from '~/composables/useStats'

const selectedTab = ref('week')
const { weeklyStats, monthlyStats, fetchWeeklyStats, fetchMonthlyStats } = useStats()

onMounted(async () => {
  await Promise.all([
    fetchWeeklyStats(),
    fetchMonthlyStats()
  ])
})
</script>
```

---

## Components

### EarningsCard.vue

```vue
<template>
  <div class="bg-white rounded-2xl shadow-lg p-8 mb-6">
    <p class="text-gray-500 text-sm mb-2">Ganho do dia</p>
    <div class="text-5xl font-bold text-green-600 mb-4">
      {{ formatCurrency(totalEarned) }}
    </div>
    <p class="text-gray-400">
      {{ totalTrips }} corridas · {{ formatCurrency(earningsPerHour) }}/h
    </p>
  </div>
</template>

<script setup lang="ts">
import { formatCurrency } from '~/utils/format'

defineProps({
  totalEarned: { type: Number, default: 0 },
  totalTrips: { type: Number, default: 0 },
  earningsPerHour: { type: Number, default: 0 },
  loading: Boolean
})
</script>
```

### GoalProgress.vue

```vue
<template>
  <div class="bg-white rounded-2xl shadow-lg p-6 mb-6">
    <div class="flex justify-between items-center mb-4">
      <h2 class="text-lg font-semibold">Meta do dia</h2>
      <span class="text-2xl font-bold text-indigo-600">
        {{ goal?.percent_complete?.toFixed(0) || 0 }}%
      </span>
    </div>
    
    <!-- Progress Bar -->
    <div class="w-full bg-gray-200 rounded-full h-3 mb-4">
      <div 
        class="bg-indigo-600 h-3 rounded-full"
        :style="{ width: `${Math.min(100, goal?.percent_complete || 0)}%` }"
      ></div>
    </div>

    <!-- Status -->
    <p v-if="goal?.remaining > 0" class="text-yellow-600">
      Faltam {{ formatCurrency(goal.remaining) }} 🚗
    </p>
    <p v-else class="text-green-600">🎉 Meta atingida!</p>

    <!-- Button -->
    <button 
      @click="showForm = true"
      class="mt-4 w-full px-4 py-2 bg-indigo-600 text-white rounded-lg"
    >
      Ajustar Meta
    </button>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { formatCurrency } from '~/utils/format'

const props = defineProps({
  goal: Object,
  loading: Boolean
})

const emit = defineEmits(['update'])
const showForm = ref(false)
const newAmount = ref(0)

const submit = () => {
  emit('update', newAmount.value)
  showForm.value = false
}
</script>
```

---

## Composables

### useAuth.ts

```typescript
import { ref, computed } from 'vue'

export const useAuth = () => {
  const token = ref<string | null>(null)
  const driver = ref(null)
  const loading = ref(false)
  const error = ref<string | null>(null)

  const isAuthenticated = computed(() => !!token.value)

  const loginWithUber = () => {
    const config = useRuntimeConfig()
    const clientId = config.public.uberClientId
    const redirectUri = `${window.location.origin}/auth/callback`
    
    const authUrl = new URL('https://login.uber.com/oauth/v2/authorize')
    authUrl.searchParams.set('client_id', clientId)
    authUrl.searchParams.set('response_type', 'code')
    authUrl.searchParams.set('redirect_uri', redirectUri)
    authUrl.searchParams.set('scope', 'profile trips.read payments.read')
    
    window.location.href = authUrl.toString()
  }

  const handleCallback = async (code: string) => {
    try {
      loading.value = true
      const { token: newToken, driver: newDriver } = await $fetch('/api/auth/uber-login', {
        method: 'POST',
        body: { code }
      })
      
      token.value = newToken
      driver.value = newDriver
      localStorage.setItem('auth_token', newToken)
      
      navigateTo('/')
    } catch (err) {
      error.value = 'Erro ao fazer login'
    } finally {
      loading.value = false
    }
  }

  const logout = () => {
    token.value = null
    driver.value = null
    localStorage.removeItem('auth_token')
    navigateTo('/login')
  }

  return {
    token, driver, loading, error, isAuthenticated,
    loginWithUber, handleCallback, logout
  }
}
```

### useStats.ts

```typescript
export const useStats = () => {
  const config = useRuntimeConfig()
  const { token } = useAuth()
  
  const stats = ref(null)
  const weeklyStats = ref(null)
  const monthlyStats = ref(null)
  const goalProgress = ref(null)

  const fetchDailyStats = async () => {
    stats.value = await $fetch(`${config.public.apiBase}/api/stats/today`, {
      headers: { 'Authorization': `Bearer ${token.value}` }
    })
  }

  const fetchWeeklyStats = async () => {
    weeklyStats.value = await $fetch(`${config.public.apiBase}/api/stats/week`, {
      headers: { 'Authorization': `Bearer ${token.value}` }
    })
  }

  const fetchMonthlyStats = async () => {
    monthlyStats.value = await $fetch(`${config.public.apiBase}/api/stats/month`, {
      headers: { 'Authorization': `Bearer ${token.value}` }
    })
  }

  const fetchGoalProgress = async () => {
    goalProgress.value = await $fetch(`${config.public.apiBase}/api/goals/progress`, {
      headers: { 'Authorization': `Bearer ${token.value}` }
    })
  }

  return {
    stats, weeklyStats, monthlyStats, goalProgress,
    fetchDailyStats, fetchWeeklyStats, fetchMonthlyStats, fetchGoalProgress
  }
}
```

---

## Pinia Stores

### stores/auth.ts

```typescript
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export const useAuthStore = defineStore('auth', () => {
  const token = ref<string | null>(null)
  const driver = ref(null)

  const isAuthenticated = computed(() => !!token.value)

  const setToken = (newToken: string) => {
    token.value = newToken
    localStorage.setItem('auth_token', newToken)
  }

  const setDriver = (newDriver: any) => {
    driver.value = newDriver
  }

  const clearAuth = () => {
    token.value = null
    driver.value = null
    localStorage.removeItem('auth_token')
  }

  const restore = () => {
    const saved = localStorage.getItem('auth_token')
    if (saved) token.value = saved
  }

  return { token, driver, isAuthenticated, setToken, setDriver, clearAuth, restore }
})
```

---

## Nuxt Config

```typescript
export default defineNuxtConfig({
  modules: ['@nuxtjs/tailwindcss'],
  
  css: ['~/assets/css/main.css'],
  
  runtimeConfig: {
    public: {
      apiBase: process.env.API_BASE_URL || 'http://localhost:8080',
      uberClientId: process.env.UBER_CLIENT_ID || ''
    }
  },
  
  ssr: true,
  
  app: {
    head: {
      title: 'Pilot - Seu Copiloto de Ganhos',
      meta: [
        { name: 'description', content: 'Analytics de ganhos para motoristas' }
      ]
    }
  }
})
```

