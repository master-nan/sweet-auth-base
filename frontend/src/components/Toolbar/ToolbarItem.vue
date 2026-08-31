<template>
  <div class="toolbar-actions row items-center no-wrap">
    <q-separator class="toolbar-actions__separator" vertical />
    <LangSelector />
    <q-separator class="toolbar-actions__separator" vertical />
    <q-btn
      class="toolbar-actions__btn"
      round
      dense
      flat
      :icon="$q.fullscreen.isActive ? 'fullscreen_exit' : 'fullscreen'"
      :aria-label="t('layout.fullScreen')"
      @click="$q.fullscreen.toggle()"
      v-if="$q.screen.gt.sm"
    >
      <q-tooltip>{{ t('layout.fullScreen') }}</q-tooltip>
    </q-btn>
    <dark-mode />
    <q-btn
      class="toolbar-actions__btn"
      round
      dense
      flat
      icon="settings"
      :aria-label="t('themeSetting.title')"
      @click="emit('open-settings')"
    >
      <q-tooltip>{{ t('themeSetting.title') }}</q-tooltip>
    </q-btn>
    <q-btn
      class="toolbar-actions__btn"
      round
      dense
      flat
      icon="refresh"
      :aria-label="t('layout.refresh')"
      @click="appStore.reloadPage(200)"
      v-if="$q.screen.gt.sm"
    >
      <q-tooltip>{{ t('layout.refresh') }}</q-tooltip>
    </q-btn>
    <notification-popover />
    <q-btn class="toolbar-actions__avatar-btn" round flat>
      <q-avatar class="toolbar-actions__avatar" color="primary" text-color="white">
        <q-icon name="admin_panel_settings" size="20px" />
      </q-avatar>
      <q-menu>
        <q-list dense>
          <q-item>
            <q-item-section>
              <div>
                {{ t('layout.signedInAs') }} <br /><strong>{{ userStore.getUserName }}</strong>
              </div>
            </q-item-section>
          </q-item>
          <q-separator />
          <!--          <q-item clickable>-->
          <!--            <q-item-section>-->
          <!--              <div>-->
          <!--                <q-icon name="tag_faces" color="blue-9" size="18px" />-->
          <!--              </div>-->
          <!--            </q-item-section>-->
          <!--          </q-item>-->
          <q-separator />

          <q-item clickable>
            <q-item-section @click="logout">{{ t('layout.signOut') }}</q-item-section>
          </q-item>
        </q-list>
      </q-menu>

      <q-tooltip>{{ t('layout.user') }}</q-tooltip>
    </q-btn>
  </div>
</template>

<script lang="ts" setup>
defineOptions({ name: 'ToolbarItem' })

import { useI18n } from 'vue-i18n'
import { useUserStore } from 'src/stores/user'
import DarkMode from 'src/components/Toolbar/DarkMode.vue'
import { useAppStore } from 'src/stores/app'
import LangSelector from 'src/components/Toolbar/LangSelector.vue'
import { useQuasar } from 'quasar'
import NotificationPopover from 'src/components/Notification/NotificationPopover.vue'

const $q = useQuasar()
const emit = defineEmits<{ 'open-settings': [] }>()

const { t } = useI18n()

const userStore = useUserStore()
const appStore = useAppStore()

const logout = async () => {
  await userStore.logout()
}
</script>

<style scoped lang="scss">
.toolbar-actions {
  gap: 8px;

  :deep(.q-btn) {
    color: var(--app-header-text);
  }
}

.toolbar-actions__separator {
  color: var(--app-header-border);
  opacity: 1;
}

.toolbar-actions__btn {
  width: 34px;
  height: 34px;
  border: 1px solid var(--app-header-border);
  border-radius: 8px;
  background: var(--app-header-control-bg);

  &:hover {
    color: var(--q-primary);
    background: var(--app-header-control-hover);
  }
}

.toolbar-actions :deep(.dark-mode-btn) {
  width: 34px;
  height: 34px;
  border: 1px solid var(--app-header-border);
  border-radius: 8px;
  background: var(--app-header-control-bg);

  &:hover {
    color: var(--q-primary);
    background: var(--app-header-control-hover);
  }
}

.toolbar-actions :deep(.language-selector) {
  min-height: 34px;
  padding: 0 9px;
  border: 1px solid var(--app-header-border);
  border-radius: 8px;
  background: var(--app-header-control-bg);
}

.toolbar-actions__avatar-btn {
  width: 38px;
  height: 38px;
  padding: 2px;
  border: 1px solid var(--app-header-border);
  border-radius: 10px;
  background: var(--app-header-control-bg);

  &:hover {
    background: var(--app-header-control-hover);
  }
}

.toolbar-actions__avatar {
  width: 32px;
  height: 32px;
  box-shadow: 0 0 0 1px var(--app-primary-border);
}
</style>
