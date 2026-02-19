<template>
  <div class="game-status" v-if="isGameActive">
    <div class="status-bar">
      <div class="status-phase" :class="phaseClass">
        <font-awesome-icon :icon="phaseIcon" />
        {{ phaseText }}
      </div>
      <div class="status-info">
        <span class="status-day" v-if="dayCount > 0">
          {{ $t('game.day') }} {{ dayCount }}
        </span>
        <span class="status-alive">
          <font-awesome-icon icon="heartbeat" />
          {{ aliveCount }}
        </span>
      </div>
    </div>
    <transition name="fade">
      <div v-if="winner" class="status-winner" :class="winner">
        <span class="winner-text">
          {{ winner === 'good' ? $t('game.goodWins') : $t('game.evilWins') }}
        </span>
        <span class="winner-reason">{{ winReason }}</span>
      </div>
    </transition>
  </div>
</template>

<script>
import { mapState } from "vuex";

export default {
  name: "GameStatus",
  computed: {
    ...mapState(["grimoire"]),
    ...mapState("players", ["players"]),
    ...mapState("session", {
      sessionPhase: state => state.phase,
      dayCount: state => state.dayCount || 0,
      winner: state => state.winner,
      winReason: state => state.winReason
    }),
    isGameActive() {
      return this.players.length >= 5 && this.sessionPhase && this.sessionPhase !== "lobby";
    },
    aliveCount() {
      return this.players.filter(p => p.isDead !== true).length;
    },
    phaseClass() {
      const phase = this.sessionPhase || "";
      if (phase.includes("night")) return "night";
      if (phase.includes("nomination") || phase.includes("voting")) return "nomination";
      if (phase === "ended") return "ended";
      return "day";
    },
    phaseIcon() {
      const phase = this.sessionPhase || "";
      if (phase.includes("night")) return "cloud-moon";
      if (phase.includes("nomination")) return "hand-point-right";
      if (phase === "ended") return "skull";
      return "heartbeat";
    },
    phaseText() {
      const phase = this.sessionPhase || "";
      switch (phase) {
        case "first_night": return this.$t("game.firstNight");
        case "night": return this.$t("game.night");
        case "day": return this.$t("game.day");
        case "nomination": return this.$t("game.nomination");
        case "voting": return this.$t("game.voting");
        case "ended": return this.$t("game.gameOver");
        default: return phase;
      }
    }
  }
};
</script>

<style scoped lang="scss">
.game-status {
  position: fixed;
  top: 0;
  left: 50%;
  transform: translateX(-50%);
  z-index: 100;
  pointer-events: none;
}

.status-bar {
  display: flex;
  align-items: center;
  gap: 16px;
  background: rgba(0, 0, 0, 0.7);
  backdrop-filter: blur(10px);
  border-radius: 0 0 12px 12px;
  padding: 6px 20px;
  color: #eee;
  font-size: 14px;
}

.status-phase {
  font-weight: bold;
  padding: 2px 10px;
  border-radius: 4px;

  svg {
    margin-right: 4px;
  }

  &.night {
    color: #9b59b6;
  }
  &.day {
    color: #f1c40f;
  }
  &.nomination {
    color: #e74c3c;
  }
  &.ended {
    color: #95a5a6;
  }
}

.status-info {
  display: flex;
  gap: 12px;

  .status-alive {
    svg {
      color: #e74c3c;
      margin-right: 2px;
    }
  }
}

.status-winner {
  text-align: center;
  padding: 12px 24px;
  border-radius: 0 0 12px 12px;
  font-size: 16px;
  pointer-events: auto;

  &.good {
    background: rgba(46, 204, 113, 0.9);
    color: white;
  }
  &.evil {
    background: rgba(231, 76, 60, 0.9);
    color: white;
  }

  .winner-text {
    font-weight: bold;
    display: block;
    font-size: 20px;
  }
  .winner-reason {
    font-size: 13px;
    opacity: 0.9;
  }
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.5s;
}
.fade-enter,
.fade-leave-to {
  opacity: 0;
}
</style>
