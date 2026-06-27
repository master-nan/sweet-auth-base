<template>
  <q-breadcrumbs class="flex items-center" active-color="none">
    <template
      v-for="(breadcrumb, index) in breadcrumbsStore.getBreadCrumbs"
      :key="`${breadcrumb.fullPath}-${breadcrumb.title}-${index}`"
    >
      <q-breadcrumbs-el
        v-if="breadcrumb.title"
        name="breadcrumb"
        :label="formatBreadcrumbTitle(breadcrumb.title)"
        :icon="showIcon ? breadcrumb.icon : undefined"
      />
    </template>
  </q-breadcrumbs>
</template>

<script lang="ts" setup>
import { useI18n } from 'vue-i18n'
import { useBreadcrumbsStore } from 'src/stores/breadcrumbs'

defineOptions({ name: 'MyBreadcrumbs' })

interface Props {
  showIcon?: boolean
}
withDefaults(defineProps<Props>(), { showIcon: true })

const { t } = useI18n()
const breadcrumbsStore = useBreadcrumbsStore()

const formatBreadcrumbTitle = (title: string) => {
  return title.startsWith('router.') ? t(title) : title
}
</script>
