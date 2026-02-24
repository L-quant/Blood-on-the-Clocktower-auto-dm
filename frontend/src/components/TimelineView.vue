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
        { id: 'votes' }
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
        const typeMap = { phases: 'phase_change', deaths: 'death', votes: 'vote_result' };
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
        return this.$t('game.phases.' + (event.data.phase || ''));
      }
      if (event.type === 'death' && event.data) {
        return this.$t('timeline.playerDied', { n: event.data.seatIndex });
      }
      if (event.type === 'vote_result' && event.data) {
        return this.$t('timeline.voteResult', {
          result: event.data.result === 'executed' ? this.$t('vote.executed') : this.$t('vote.safe')
        });
      }
      return '';
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
