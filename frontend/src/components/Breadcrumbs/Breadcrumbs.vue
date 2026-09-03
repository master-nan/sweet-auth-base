<template>
  <q-breadcrumbs class="breadcrumbs flex items-center" active-color="none">
    <template #separator>
      <span class="breadcrumbs__separator">/</span>
    </template>
    <template
      v-for="(breadcrumb, index) in visibleBreadcrumbs"
      :key="`${breadcrumb.fullPath}-${index}`"
    >
      <q-breadcrumbs-el
        class="breadcrumbs__item"
        :class="{ 'text-primary': index === visibleBreadcrumbs.length - 1 }"
        name="breadcrumb"
        :label="formatBreadcrumbTitle(breadcrumb.title)"
        :icon="showIcon ? breadcrumb.icon : undefined"
      />
    </template>
  </q-breadcrumbs>
</template>

<script lang="ts" setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useBreadcrumbsStore } from '@/stores/breadcrumbs'

defineOptions({ name: 'MyBreadcrumbs' })

interface Props {
  showIcon?: boolean
}
withDefaults(defineProps<Props>(), { showIcon: true })

const { t } = useI18n()
const breadcrumbsStore = useBreadcrumbsStore()
const visibleBreadcrumbs = computed(() =>
  breadcrumbsStore.getBreadCrumbs.filter((breadcrumb) => Boolean(breadcrumb.title)),
)

const formatBreadcrumbTitle = (title: string) => {
  return title.startsWith('router.') ? t(title) : title
}
</script>

<style scoped lang="scss">
.breadcrumbs {
  min-height: 34px;
  line-height: 1;
}

.breadcrumbs__item {
  color: inherit;
  font-size: 14px;
  font-weight: 800;
  letter-spacing: 0;
}

.breadcrumbs__separator {
  display: inline-flex;
  align-items: center;
  align-self: stretch;
  color: currentColor;
  opacity: 0.55;
}
</style>
