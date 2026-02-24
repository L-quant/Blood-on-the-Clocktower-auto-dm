<template>
  <header class="top-bar">
    <div class="top-bar__left">
      <span
        v-if="roomId"
        class="top-bar__status"
        :class="statusClass"
        :title="statusTitle"
      >
        <span class="top-bar__dot"></span>
        <span class="top-bar__status-text">{{ statusText }}</span>
      </span>
    </div>
    <div class="top-bar__center">
      <span class="top-bar__phase" v-if="showPhase">
        {{ phaseText }}
      </span>
      <span class="top-bar__title" v-else>
        {{ $t('app.title') }}
      </span>
    </div>
    <div class="top-bar__right">
      <span
        class="top-bar__room-code"
        v-if="roomId"
        @click="copyRoomCode"
        :title="$t('lobby.copyLink')"
      >
        {{ shortRoomId }}
        <font-awesome-icon icon="copy" class="top-bar__copy-icon" />
      </span>
      <button
        class="top-bar__settings-btn"
        @click="$emit('toggle-settings')"
        v-if="showSettings"
      >
        <font-awesome-icon icon="cog" />
      </button>
    </div>
  </header>
</template>

<script>
import { mapState, mapGetters } from "vuex";

export default {
  name: "TopBar",
  computed: {
    ...mapState(["roomId", "connected", "reconnecting"]),
    ...mapState("game", ["phase", "dayCount"]),
    ...mapGetters("game", ["isPlaying", "isNight"]),
    statusClass() {
      if (this.reconnecting) return "reconnecting";
      if (this.connected) return "connected";
      return "disconnected";
    },
    statusText() {
      if (this.reconnecting) return this.$t("connection.reconnecting");
      if (this.connected) return this.$t("connection.connected");
      return this.$t("connection.disconnected");
    },
    statusTitle() {
      return this.statusText;
    },
    showPhase() {
      return this.isPlaying;
    },
    phaseText() {
      const phase = this.phase;
      if (phase === "day" || phase === "nomination" || phase === "voting") {
        return this.$t("game.dayN", { n: this.dayCount }) + "·" + this.$t("game.phases." + phase);
      }
      if (phase === "night" || phase === "first_night") {
        return this.$t("game.phases." + phase);
      }
      return this.$t("game.phases." + phase);
    },
    shortRoomId() {
      if (!this.roomId) return '';
      // Show first 8 chars of UUID for readability
      return this.roomId.length > 8 ? this.roomId.substr(0, 8) : this.roomId;
    },
    showSettings() {
      return !this.isPlaying;
    }
  },
  methods: {
    async copyRoomCode() {
      try {
        await navigator.clipboard.writeText(this.roomId);
      } catch (e) {
        const el = document.createElement("textarea");
        el.value = this.roomId;
        document.body.appendChild(el);
        el.select();
        document.execCommand("copy");
        document.body.removeChild(el);
      }
    }
  }
};
</script>

<style lang="scss" scoped>
@import "../vars";

.top-bar {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  height: 44px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 12px;
  background: rgba(0, 0, 0, 0.85);
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
  z-index: 100;
  backdrop-filter: blur(10px);
  -webkit-backdrop-filter: blur(10px);

  &__left,
  &__right {
    display: flex;
    align-items: center;
    min-width: 80px;
  }

  &__right {
    justify-content: flex-end;
  }

  &__center {
    flex: 1;
    text-align: center;
    font-family: PiratesBay, sans-serif;
    font-size: 1rem;
    letter-spacing: 1px;
  }

  &__status {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 0.7rem;
    opacity: 0.8;
  }

  &__dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: #666;
  }

  &__status-text {
    white-space: nowrap;
  }

  .connected .top-bar__dot {
    background: #4caf50;
  }
  .reconnecting .top-bar__dot {
    background: #ff9800;
    animation: pulse-dot 1s ease-in-out infinite;
  }
  .disconnected .top-bar__dot {
    background: #f44336;
  }

  &__phase {
    font-size: 0.9rem;
  }

  &__title {
    font-size: 0.85rem;
    opacity: 0.7;
  }

  &__room-code {
    font-family: monospace;
    font-size: 0.8rem;
    letter-spacing: 2px;
    padding: 2px 8px;
    border: 1px solid rgba(255, 255, 255, 0.2);
    border-radius: 4px;
    cursor: pointer;
    transition: all 200ms;
    display: flex;
    align-items: center;
    gap: 6px;

    &:hover {
      border-color: rgba(255, 255, 255, 0.5);
    }

    &:active {
      transform: scale(0.95);
    }
  }

  &__copy-icon {
    font-size: 0.7rem;
    opacity: 0.5;
  }

  &__settings-btn {
    background: none;
    border: none;
    color: white;
    font-size: 1rem;
    padding: 4px 8px;
    cursor: pointer;
    opacity: 0.7;
    margin-left: 8px;

    &:hover {
      opacity: 1;
    }
  }
}

@keyframes pulse-dot {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.3; }
}
</style>
