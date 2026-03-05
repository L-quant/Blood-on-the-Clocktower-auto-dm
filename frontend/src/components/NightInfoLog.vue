<!-- NightInfoLog 夜晚查验信息历史记录面板
  [IN]  store/modules/night.js（nightInfoHistory）
  [OUT] GameScreen.vue（左侧栏子组件）
  [POS] 展示每夜信息类角色收到的查验结果，持久可查 -->
<template>
  <div class="night-log" v-if="nightInfoHistory.length">
    <span class="night-log__title">
      <font-awesome-icon icon="moon" />
      {{ $t('nightLog.title') }}
    </span>
    <div class="night-log__list">
      <div
        v-for="(entry, idx) in nightInfoHistory"
        :key="idx"
        class="night-log__entry"
      >
        <div class="night-log__entry-header">
          <img
            v-if="roleIcon(entry.roleId)"
            class="night-log__icon"
            :src="roleIcon(entry.roleId)"
            :alt="roleName(entry.roleId)"
          />
          <span class="night-log__role">{{ roleName(entry.roleId) }}</span>
          <span class="night-log__night">{{ $t('game.nightN', { n: entry.nightNumber }) }}</span>
        </div>
        <p class="night-log__message">{{ entry.message }}</p>
      </div>
    </div>
  </div>
</template>

<script>
import { mapState } from "vuex";

export default {
  name: "NightInfoLog",
  computed: {
    ...mapState("night", ["nightInfoHistory"])
  },
  methods: {
    roleName(roleId) {
      if (!roleId) return '';
      const key = 'roles.' + roleId;
      return this.$te(key) ? this.$t(key) : roleId;
    },
    roleIcon(roleId) {
      if (!roleId) return '';
      try {
        return require(`../assets/icons/${roleId}.png`);
      } catch (_e) {
        return '';
      }
    }
  }
};
</script>

<style lang="scss" scoped>
@import "../vars";

.night-log {
  margin-bottom: 16px;

  &__title {
    font-size: 0.7rem;
    opacity: 0.4;
    text-transform: uppercase;
    letter-spacing: 1px;
    display: flex;
    align-items: center;
    gap: 6px;
    margin-bottom: 8px;
  }

  &__list {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  &__entry {
    background: rgba(255, 255, 255, 0.06);
    border: 1px solid rgba(255, 255, 255, 0.08);
    border-radius: 8px;
    padding: 10px 12px;

    &:hover {
      background: rgba(255, 255, 255, 0.09);
    }
  }

  &__entry-header {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 4px;
  }

  &__icon {
    width: 24px;
    height: 24px;
    object-fit: contain;
  }

  &__role {
    font-size: 0.8rem;
    font-weight: 600;
    flex: 1;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  &__night {
    font-size: 0.65rem;
    opacity: 0.4;
    white-space: nowrap;
  }

  &__message {
    font-size: 0.8rem;
    line-height: 1.4;
    margin: 0;
    color: rgba(255, 255, 255, 0.8);
    font-family: Papyrus, serif;
  }
}
</style>
