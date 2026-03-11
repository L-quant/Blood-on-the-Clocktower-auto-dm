<!-- TimelineView 事件时间线（含过滤按钮）
  [OUT] GameScreen.vue（主布局子组件）
  [POS] 游戏事件历史展示，按类型过滤 -->
<template>
  <div class="timeline-view">
    <!-- Filters -->
    <div class="timeline-view__filters">
      <button
        v-for="f in filterOptions"
        :key="f.id"
        class="timeline-view__filter"
        :class="{ active: filters.includes(f.id) }"
        @click="toggleFilter(f.id)"
      >{{ $t('timeline.' + f.id) }}</button>
    </div>

    <!-- Event list -->
    <div class="timeline-view__list">
      <div v-if="events.length === 0" class="timeline-view__empty">
        {{ $t('timeline.empty') }}
      </div>
      <div
        v-for="event in events"
        :key="event.id"
        class="timeline-view__item"
        :class="event.type"
      >
        <div class="timeline-view__dot" :class="event.type"></div>
        <div class="timeline-view__content">
          <span class="timeline-view__type">{{ getTypeLabel(event) }}</span>
          <span class="timeline-view__detail">{{ getDetail(event) }}</span>
          <span class="timeline-view__time">{{ formatTime(event.timestamp) }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import { mapState, mapGetters } from "vuex";

export default {
  name: "TimelineView",
  computed: {
    ...mapState("timeline", ["filters"]),
    ...mapGetters("timeline", ["filtered"]),
    filterOptions() {
      return [
        { id: 'all' },
        { id: 'phases' },
        { id: 'deaths' },
        { id: 'votes' },
        { id: 'abilities' }
      ];
    },
    events() {
      return this.filtered;
    }
  },
  methods: {
    toggleFilter(id) {
      if (id === 'all') {
        this.$store.commit("timeline/setFilters", ['all']);
      } else {
        let current = [...this.filters].filter(f => f !== 'all');
        if (current.includes(id)) {
          current = current.filter(f => f !== id);
        } else {
          current.push(id);
        }
        if (current.length === 0) current = ['all'];
        // Map display names to event types
        const typeMap = { phases: 'phase_change', deaths: 'death', votes: 'vote_result', abilities: 'ability' };
        this.$store.commit("timeline/setFilters", current.map(c => typeMap[c] || c));
      }
    },
    getTypeLabel(event) {
      const labels = {
        phase_change: this.$t('timeline.phaseChange'),
        death: '☠️',
        nomination: '⚖️',
        vote_result: '🗳️',
        ability: '✨',
        system: 'ℹ️'
      };
      return labels[event.type] || event.type;
    },
    getDetail(event) {
      if (event.type === 'phase_change' && event.data) {
        return this.getPhaseChangeDetail(event);
      }
      if (event.type === 'death' && event.data) {
        return this.$t('timeline.playerDied', { n: event.data.seatIndex });
      }
      if (event.type === 'vote_result' && event.data) {
        return this.$t('timeline.voteResult', {
          result: event.data.result === 'executed' ? this.$t('vote.executed') : this.$t('vote.safe')
        });
      }
      if (event.type === 'ability' && event.data && event.data.ability === 'slayer_shot') {
        const params = {
          shooter: event.data.shooterSeat,
          target: event.data.targetSeat
        };
        if (event.data.result === 'killed_night') {
          return this.$t('timeline.slayerShotKillNight', params);
        }
        if (event.data.result === 'killed') {
          return this.$t('timeline.slayerShotKill', params);
        }
        return this.$t('timeline.slayerShotMiss', params);
      }
      return '';
    },
    getPhaseChangeDetail(event) {
      const phase = event.data && event.data.phase ? event.data.phase : '';
      if (phase === 'first_night') {
        return this.$t('timeline.phaseFirstNight');
      }
      if (phase === 'day') {
        return this.$t('timeline.phaseNthDay', { ordinal: this.toChineseOrdinal(event.dayCount) });
      }
      if (phase === 'night') {
        return this.$t('timeline.phaseNthNight', { ordinal: this.toChineseOrdinal(event.dayCount) });
      }
      return this.$t('game.phases.' + phase);
    },
    toChineseOrdinal(value) {
      const number = parseInt(value, 10);
      if (!number || number < 1) return '一';

      const digits = ['零', '一', '二', '三', '四', '五', '六', '七', '八', '九'];
      if (number < 10) return digits[number];
      if (number === 10) return '十';
      if (number < 20) return '十' + digits[number % 10];
      if (number < 100) {
        const tens = Math.floor(number / 10);
        const ones = number % 10;
        return digits[tens] + '十' + (ones ? digits[ones] : '');
      }
      return String(number);
    },
    formatTime(ts) {
      const d = new Date(ts);
      return d.getHours().toString().padStart(2, '0') + ':' +
             d.getMinutes().toString().padStart(2, '0') + ':' +
             d.getSeconds().toString().padStart(2, '0');
    }
  }
};
</script>

<style lang="scss" scoped>
@import "../vars";

.timeline-view {
  height: 100%;
  display: flex;
  flex-direction: column;

  &__filters {
    display: flex;
    gap: 4px;
    padding: 8px 12px;
    flex-shrink: 0;
  }

  &__filter {
    padding: 4px 12px;
    border: 1px solid rgba(255, 255, 255, 0.15);
    border-radius: 12px;
    background: none;
    color: rgba(255, 255, 255, 0.4);
    font-size: 0.65rem;
    cursor: pointer;
    transition: all 200ms;

    &.active {
      border-color: $townsfolk;
      color: $townsfolk;
      background: rgba($townsfolk, 0.1);
    }
  }

  &__list {
    flex: 1;
    overflow-y: auto;
    padding: 0 12px 12px;
    -webkit-overflow-scrolling: touch;
  }

  &__empty {
    text-align: center;
    opacity: 0.3;
    padding: 40px 0;
    font-size: 0.85rem;
  }

  &__item {
    display: flex;
    gap: 12px;
    padding: 10px 0;
    border-bottom: 1px solid rgba(255, 255, 255, 0.05);
  }

  &__dot {
    width: 10px;
    height: 10px;
    border-radius: 50%;
    margin-top: 4px;
    flex-shrink: 0;
    background: rgba(255, 255, 255, 0.3);

    &.phase_change { background: $townsfolk; }
    &.death { background: $demon; }
    &.vote_result { background: $fabled; }
    &.nomination { background: $minion; }
    &.ability { background: $outsider; }
  }

  &__content {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  &__type {
    font-size: 0.7rem;
    opacity: 0.5;
  }

  &__detail {
    font-size: 0.85rem;
  }

  &__time {
    font-size: 0.6rem;
    opacity: 0.3;
    font-family: monospace;
  }
}
</style>
