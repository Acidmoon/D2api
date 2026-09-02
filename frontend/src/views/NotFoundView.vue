<template>
  <div
    class="relative flex min-h-screen items-center justify-center overflow-hidden bg-background px-4 text-foreground"
  >
    <!-- Soft brand glow -->
    <div
      class="pointer-events-none absolute -top-32 left-1/2 h-80 w-[36rem] -translate-x-1/2 rounded-full bg-brand/10 blur-3xl"
      aria-hidden="true"
    ></div>

    <div class="relative z-10 w-full max-w-md text-center">
      <!-- 404 Display -->
      <div class="mb-8">
        <div class="relative inline-block">
          <span class="text-[7rem] font-bold leading-none text-brand/10 sm:text-[10rem] md:text-[12rem]"
            >404</span
          >
          <div class="absolute inset-0 flex items-center justify-center">
            <div
              class="flex h-20 w-20 items-center justify-center rounded-3xl bg-card shadow-nm sm:h-24 sm:w-24"
            >
              <svg
                class="h-10 w-10 text-brand sm:h-12 sm:w-12"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                stroke-width="1.5"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126zM12 15.75h.007v.008H12v-.008z"
                />
              </svg>
            </div>
          </div>
        </div>
      </div>

      <!-- Text Content -->
      <div class="mb-8">
        <h1 class="mb-3 text-2xl font-bold text-foreground">
          {{ t('errors.pageNotFound') }}
        </h1>
        <p class="empty-state-description mx-auto">
          {{ t('errors.pageNotFoundDescription') }}
        </p>
      </div>

      <!-- Action Buttons -->
      <div class="flex flex-col justify-center gap-3 sm:flex-row">
        <button @click="goBack" class="btn btn-secondary">
          <Icon name="arrowLeft" size="md" class="mr-2" />
          {{ t('errors.goBack') }}
        </button>
        <router-link to="/dashboard" class="btn btn-primary">
          <Icon name="home" size="md" class="mr-2" />
          {{ t('errors.backToDashboard') }}
        </router-link>
      </div>

      <!-- Help Link -->
      <p class="mt-8 text-sm text-muted-foreground">
        {{ t('errors.needHelp') }}
        <a
          href="#"
          class="font-medium text-brand transition-colors hover:underline"
        >
          {{ t('common.contactSupport') }}
        </a>
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()
const router = useRouter()

function goBack(): void {
  router.back()
}
</script>
