<template>
  <transition name="phase-fade" @after-leave="onTransitionEnd">
    <div
      class="phase-transition"
      v-if="showing"
      :class="phaseClass"
    >
      <div class="phase-transition__content">
        <h1 class="phase-transition__title">{{ phaseTitle }}</h1>
        <p class="phase-transition__subtitle" v-if="phaseSubtitle">{{ phaseSubtitle }}</p>
      </div>
    </div>
  </transition>
</template>

<script>
import { mapState } from "vuex";

export default {
  name: "PhaseTransition",
  data() {
    return {
      showing: false,
      currentPhase: '',
      dismissTimer: null
    };
  },
  computed: {
    ...mapState("game", ["phase", "dayCount"]),
    phaseClass() {
      if (['night', 'first_night'].includes(this.currentPhase)) return 'night';
      if (this.currentPhase === 'day') return 'day';
      if (this.currentPhase === 'nomination') return 'nomination';
      if (this.currentPhase === 'voting') return 'voting';
      if (this.currentPhase === 'ended') return 'ended';
      return '';
    },
    phaseTitle() {
      return this.$t('game.phases.' + this.currentPhase) || '';
    },
    phaseSubtitle() {
      if (this.currentPhase === 'day') {
        return this.$t('game.dayN', { n: this.dayCount });
      }
      if (this.currentPhase === 'night' && this.dayCount > 0) {
        return this.$t('game.nightN', { n: this.dayCount });
      }
      return '';
    }
  },
  watch: {
    phase(newPhase, oldPhase) {
      if (newPhase && newPhase !== oldPhase && newPhase !== 'lobby') {
        this.showTransition(newPhase);
      }
    }
  },
  methods: {
    showTransition(phase) {
      clearTimeout(this.dismissTimer);
      this.currentPhase = phase;
      this.showing = true;
      this.dismissTimer = setTimeout(() => {
        this.showing = false;
      }, 2000);
    },
    onTransitionEnd() {
      // Cleanup
    }
  },
  beforeDestroy() {
    clearTimeout(this.dismissTimer);
  }
};
</script>

<style lang="scss" scoped>
@import "../vars";

.phase-transition {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 200;
  pointer-events: none;

  &.night {
    background: radial-gradient(
      ellipse at center,
      rgba(0, 15, 40, 0.95) 0%,
      rgba(0, 0, 0, 0.98) 100%
    );
  }

  &.day {
    background: radial-gradient(
      ellipse at center,
      rgba(60, 40, 10, 0.9) 0%,
      rgba(20, 10, 0, 0.95) 100%
    );
  }

  &.nomination {
    background: rgba(0, 0, 0, 0.85);
  }

  &.voting {
    background: rgba(30, 0, 0, 0.85);
  }

  &.ended {
    background: rgba(0, 0, 0, 0.9);
  }

  &__content {
    text-align: center;
    animation: phase-zoom-in 0.6s ease-out;
  }

  &__title {
    font-size: 2.5rem;
    letter-spacing: 3px;
    text-shadow: 0 0 30px rgba(255, 255, 255, 0.3);
  }

  &__subtitle {
    font-size: 1rem;
    opacity: 0.6;
    margin-top: 8px;
    font-family: Papyrus, serif;
  }
}

.phase-fade-enter-active {
  transition: opacity 500ms ease-in;
}
.phase-fade-leave-active {
  transition: opacity 800ms ease-out;
}
.phase-fade-enter,
.phase-fade-leave-to {
  opacity: 0;
}

@keyframes phase-zoom-in {
  from {
    opacity: 0;
    transform: scale(0.8);
  }
  to {
    opacity: 1;
    transform: scale(1);
  }
}
</style>
