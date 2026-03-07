<!-- NightInfoLog 夜晚查验信息与间谍魔典历史面板
  [IN]  store/modules/night.js（nightInfoHistory / grimoireHistory）
  [OUT] GameScreen.vue（左侧栏子组件）
  [POS] 展示每夜信息类角色收到的查验结果，以及间谍按夜可回看的魔典 -->
<template>
  <div class="night-log" v-if="hasContent">
    <span class="night-log__title">
      <font-awesome-icon icon="moon" />
      {{ $t('nightLog.title') }}
    </span>
    <div v-if="visibleNightInfoHistory.length" class="night-log__list">
      <div
        v-for="(entry, idx) in visibleNightInfoHistory"
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
    <div v-if="isSpy && sortedGrimoireHistory.length" class="night-log__grimoire">
      <div class="night-log__section-title">{{ $t('nightLog.grimoireTitle') }}</div>
      <div
        v-for="entry in sortedGrimoireHistory"
        :key="entry.nightNumber"
        class="night-log__grimoire-entry"
      >
        <button
          type="button"
          class="night-log__accordion"
          @click="toggleNight(entry.nightNumber)"
        >
          <span>{{ $t('nightLog.grimoireNight', { n: entry.nightNumber }) }}</span>
          <span class="night-log__accordion-indicator">{{ isExpanded(entry.nightNumber) ? '-' : '+' }}</span>
        </button>
        <div v-if="isExpanded(entry.nightNumber)" class="night-log__grimoire-body">
          <div
            v-for="player in normalizedPlayers(entry.content)"
            :key="entry.nightNumber + '-' + player.user_id"
            class="night-log__grimoire-player"
            :class="{ 'is-dead': player.is_alive === false }"
          >
            <span class="night-log__seat">{{ seatLabel(player.seat_index) }}</span>
            <img
              v-if="roleIcon(player.role)"
              class="night-log__icon"
              :src="roleIcon(player.role)"
              :alt="roleName(player.role)"
            />
            <span class="night-log__role">{{ roleName(player.role) }}</span>
            <span v-if="statusLabel(player)" class="night-log__status">{{ statusLabel(player) }}</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import { mapState } from "vuex";

export default {
  name: "NightInfoLog",
  data() {
    return {
      expandedNights: {}
    };
  },
  computed: {
    ...mapState("night", ["nightInfoHistory", "grimoireHistory"]),
    ...mapState("players", ["myRole"]),
    hasContent() {
      return this.visibleNightInfoHistory.length > 0 || this.sortedGrimoireHistory.length > 0;
    },
    isSpy() {
      return this.myRole && this.myRole.roleId === 'spy';
    },
    visibleNightInfoHistory() {
      return this.nightInfoHistory.filter(entry => entry.infoType !== 'grimoire');
    },
    sortedGrimoireHistory() {
      return [...this.grimoireHistory].sort((left, right) => right.nightNumber - left.nightNumber);
    }
  },
  watch: {
    sortedGrimoireHistory: {
      immediate: true,
      handler(entries) {
        if (!entries.length) return;
        const latestNight = entries[0].nightNumber;
        if (this.expandedNights[latestNight] === undefined) {
          this.$set(this.expandedNights, latestNight, true);
        }
      }
    }
  },
  methods: {
    toggleNight(nightNumber) {
      this.$set(this.expandedNights, nightNumber, !this.isExpanded(nightNumber));
    },
    isExpanded(nightNumber) {
      return !!this.expandedNights[nightNumber];
    },
    normalizedPlayers(content) {
      const players = content && Array.isArray(content.players) ? content.players : [];
      return [...players].sort((left, right) => left.seat_index - right.seat_index);
    },
    seatLabel(seatIndex) {
      return this.$t('lobby.seat', { n: seatIndex });
    },
    statusLabel(player) {
      const labels = [];
      if (player.is_drunk) {
        labels.push(this.$t('nightLog.drunk'));
      }
      if (player.poisoned) {
        labels.push(this.$t('nightLog.poisoned'));
      }
      return labels.join(' / ');
    },
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
    margin-bottom: 12px;
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

  &__section-title {
    font-size: 0.72rem;
    opacity: 0.55;
    text-transform: uppercase;
    letter-spacing: 1px;
    margin-bottom: 8px;
  }

  &__grimoire {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  &__grimoire-entry {
    background: rgba(255, 255, 255, 0.05);
    border: 1px solid rgba(255, 255, 255, 0.08);
    border-radius: 10px;
    overflow: hidden;
  }

  &__accordion {
    width: 100%;
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 10px 12px;
    background: rgba(255, 255, 255, 0.04);
    color: inherit;
    border: 0;
    cursor: pointer;
    font-size: 0.82rem;
    font-weight: 600;
    text-align: left;
  }

  &__accordion-indicator {
    opacity: 0.6;
    font-size: 0.9rem;
  }

  &__grimoire-body {
    display: flex;
    flex-direction: column;
    gap: 8px;
    padding: 10px 12px 12px;
  }

  &__grimoire-player {
    display: grid;
    grid-template-columns: auto 24px minmax(0, 1fr) auto;
    align-items: center;
    gap: 8px;
    min-width: 0;

    &.is-dead {
      opacity: 0.55;
    }
  }

  &__seat {
    font-size: 0.78rem;
    white-space: nowrap;
    color: rgba(255, 255, 255, 0.75);
  }

  &__status {
    font-size: 0.7rem;
    padding: 2px 8px;
    border-radius: 999px;
    background: rgba(139, 26, 26, 0.35);
    border: 1px solid rgba(255, 255, 255, 0.12);
    white-space: nowrap;
  }
}
</style>
