<template>
  <div class="lobby-screen">
    <!-- Room code display -->
    <div class="lobby-screen__room-info">
      <div class="lobby-screen__room-code">
        <span class="lobby-screen__room-label">{{ $t('lobby.roomCode') }}:</span>
        <span class="lobby-screen__room-value" :title="roomId">{{ shortRoomId }}</span>
      </div>
      <button
        class="lobby-screen__copy-btn"
        @click="copyRoomId"
      >
        <font-awesome-icon icon="copy" />
        {{ copyLabel }}
      </button>
    </div>

    <!-- Status message -->
    <div class="lobby-screen__status">
      <p v-if="!isSeated" class="lobby-screen__hint pulse">
        {{ $t('lobby.clickToSeat') }}
      </p>
      <p v-else-if="!isRoomOwner" class="lobby-screen__hint pulse">
        {{ $t('lobby.waitingForStart') }}
      </p>
    </div>

    <!-- Seat grid -->
    <LobbyPlayerGrid
      @seat-down="seatDown"
      @leave-seat="confirmLeaveSeat"
    />

    <!-- Spectator count -->
    <div class="lobby-screen__spectators" v-if="spectatorCount > 0">
      {{ $t('lobby.spectators', { count: spectatorCount }) }}
    </div>

    <!-- Room Owner: Game config -->
    <div class="lobby-screen__config" v-if="isRoomOwner">
      <!-- Seat count adjuster -->
      <div class="lobby-screen__seat-count">
        <span class="lobby-screen__divider">{{ $t('lobby.seatCount') }}</span>
        <div class="lobby-screen__counter">
          <button
            class="lobby-screen__counter-btn"
            :class="{ disabled: seatCount <= 5 }"
            @click="adjustSeatCount(-1)"
          >-</button>
          <span class="lobby-screen__counter-value">{{ seatCount }}</span>
          <button
            class="lobby-screen__counter-btn"
            :class="{ disabled: seatCount >= 15 }"
            @click="adjustSeatCount(1)"
          >+</button>
        </div>
      </div>

      <div class="lobby-screen__divider">{{ $t('lobby.edition') }}</div>
      <EditionPicker @change="onEditionChange" />

      <div class="lobby-screen__team-preview" v-if="teamPreview">
        <span class="lobby-screen__divider">{{ $t('lobby.expectedTeams') }}</span>
        <p class="lobby-screen__teams">{{ teamPreview }}</p>
      </div>

      <button
        class="button townsfolk lobby-screen__start-btn"
        :class="{ disabled: !canStart }"
        @click="startGame"
      >
        <font-awesome-icon icon="dice" />
        {{ $t('lobby.startGame') }}
      </button>
      <p class="lobby-screen__min-players" v-if="!canStart">
        {{ $t('lobby.minPlayers') }}
      </p>
    </div>

    <!-- Bottom action -->
    <div class="lobby-screen__bottom-actions">
      <button
        v-if="isSeated"
        class="button lobby-screen__action-btn"
        @click="confirmLeaveSeat"
      >
        {{ $t('lobby.leaveSeat') }}
      </button>
      <button
        class="button lobby-screen__action-btn"
        @click="confirmLeaveRoom"
      >
        {{ $t('lobby.leaveRoom') }}
      </button>
    </div>
  </div>
</template>

<script>
import { mapState, mapGetters } from "vuex";
import LobbyPlayerGrid from "./LobbyPlayerGrid";
import EditionPicker from "./EditionPicker";

export default {
  name: "LobbyScreen",
  components: { LobbyPlayerGrid, EditionPicker },
  data() {
    return {
      spectatorCount: 0,
      copyLabel: this.$t('lobby.copyLink'),
      copyTimer: null
    };
  },
  computed: {
    ...mapState(["roomId", "isRoomOwner", "seatIndex", "edition", "seatCount"]),
    ...mapGetters(["isSeated"]),
    ...mapGetters("players", ["playerCount"]),
    shortRoomId() {
      if (!this.roomId) return '';
      return this.roomId.length > 8 ? this.roomId.substr(0, 8) + '...' : this.roomId;
    },
    canStart() {
      return this.playerCount >= 5;
    },
    teamPreview() {
      if (!this.edition) return '';
      const count = this.playerCount || this.seatCount;
      const configs = {
        5: { t: 3, o: 0, m: 1, d: 1 },
        6: { t: 3, o: 1, m: 1, d: 1 },
        7: { t: 5, o: 0, m: 1, d: 1 },
        8: { t: 5, o: 1, m: 1, d: 1 },
        9: { t: 5, o: 2, m: 1, d: 1 },
        10: { t: 7, o: 0, m: 2, d: 1 },
        11: { t: 7, o: 1, m: 2, d: 1 },
        12: { t: 7, o: 2, m: 2, d: 1 },
        13: { t: 9, o: 0, m: 3, d: 1 },
        14: { t: 9, o: 1, m: 3, d: 1 },
        15: { t: 9, o: 2, m: 3, d: 1 }
      };
      const cfg = configs[count];
      if (!cfg) return '';
      return `${cfg.t}${this.$t('teams.townsfolk')} ${cfg.o}${this.$t('teams.outsider')} ${cfg.m}${this.$t('teams.minion')} ${cfg.d}${this.$t('teams.demon')}`;
    }
  },
  methods: {
    seatDown(seatIndex) {
      if (this.isSeated) return;
      this.$store.dispatch("seatDown", seatIndex);
    },
    confirmLeaveSeat() {
      this.$store.commit("ui/openModal", {
        modal: "confirm",
        data: {
          message: this.$t("confirm.leaveSeat"),
          onConfirm: () => {
            this.$store.dispatch("leaveSeat");
          }
        }
      });
    },
    confirmLeaveRoom() {
      this.$store.commit("ui/openModal", {
        modal: "confirm",
        data: {
          message: this.$t("confirm.leaveRoom"),
          onConfirm: () => {
            this.$store.dispatch("leaveRoom");
          }
        }
      });
    },
    startGame() {
      if (!this.canStart) return;
      this.$store.dispatch("startGame");
    },
    onEditionChange(ed) {
      // Send to backend
      this.$store.dispatch("updateRoomSettings", { edition: ed.id });
    },
    adjustSeatCount(delta) {
      const newCount = this.seatCount + delta;
      if (newCount < 5 || newCount > 15) return;
      this.$store.commit("setSeatCount", newCount);
      // Send to backend
      this.$store.dispatch("updateRoomSettings", { max_players: String(newCount) });
    },
    async copyRoomId() {
      const text = this.roomId;
      try {
        await navigator.clipboard.writeText(text);
      } catch (e) {
        const el = document.createElement("textarea");
        el.value = text;
        document.body.appendChild(el);
        el.select();
        document.execCommand("copy");
        document.body.removeChild(el);
      }
      // Show feedback
      this.copyLabel = this.$t('toast.copied');
      clearTimeout(this.copyTimer);
      this.copyTimer = setTimeout(() => {
        this.copyLabel = this.$t('lobby.copyLink');
      }, 2000);
    }
  },
  beforeDestroy() {
    clearTimeout(this.copyTimer);
  }
};
</script>

<style lang="scss" scoped>
@import "../vars";

.lobby-screen {
  padding: 20px 16px;
  max-width: 480px;
  margin: 0 auto;
  display: flex;
  flex-direction: column;
  gap: 20px;
  min-height: 100%;

  &__room-info {
    text-align: center;
  }

  &__room-code {
    margin-bottom: 8px;
  }

  &__room-label {
    font-size: 0.8rem;
    opacity: 0.5;
  }

  &__room-value {
    font-family: monospace;
    font-size: 1.4rem;
    letter-spacing: 4px;
    margin-left: 8px;
    font-weight: bold;
  }

  &__copy-btn {
    background: none;
    border: 1px solid rgba(255, 255, 255, 0.2);
    border-radius: 6px;
    color: white;
    padding: 6px 12px;
    font-size: 0.75rem;
    cursor: pointer;
    transition: all 200ms;
    display: inline-flex;
    align-items: center;
    gap: 6px;

    &:hover {
      border-color: rgba(255, 255, 255, 0.4);
    }

    &:active {
      transform: scale(0.95);
    }
  }

  &__status {
    text-align: center;
  }

  &__hint {
    font-size: 0.85rem;
    opacity: 0.6;
    margin: 0;

    &.pulse {
      animation: lobby-pulse 2s ease-in-out infinite;
    }
  }

  &__spectators {
    text-align: center;
    font-size: 0.75rem;
    opacity: 0.4;
  }

  &__config {
    display: flex;
    flex-direction: column;
    gap: 16px;
    align-items: center;
  }

  &__divider {
    font-size: 0.75rem;
    opacity: 0.4;
    text-transform: uppercase;
    letter-spacing: 2px;
    text-align: center;
  }

  &__seat-count {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 8px;
  }

  &__counter {
    display: flex;
    align-items: center;
    gap: 16px;
  }

  &__counter-btn {
    width: 36px;
    height: 36px;
    border: 2px solid rgba(255, 255, 255, 0.3);
    border-radius: 50%;
    background: rgba(255, 255, 255, 0.05);
    color: white;
    font-size: 1.2rem;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all 200ms;

    &:hover:not(.disabled) {
      border-color: $townsfolk;
      background: rgba($townsfolk, 0.1);
    }

    &:active:not(.disabled) {
      transform: scale(0.9);
    }

    &.disabled {
      opacity: 0.3;
      cursor: default;
    }
  }

  &__counter-value {
    font-size: 1.4rem;
    font-weight: bold;
    min-width: 30px;
    text-align: center;
  }

  &__team-preview {
    text-align: center;
  }

  &__teams {
    font-size: 0.8rem;
    opacity: 0.7;
    margin: 4px 0 0;
  }

  &__start-btn {
    width: 100%;
    max-width: 280px;
    font-size: 1rem;
    padding: 6px 0;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
  }

  &__min-players {
    font-size: 0.7rem;
    opacity: 0.4;
    text-align: center;
    margin: 0;
  }

  &__bottom-actions {
    display: flex;
    flex-direction: column;
    gap: 8px;
    align-items: center;
    padding-top: 12px;
    margin-top: auto;
  }

  &__action-btn {
    width: 100%;
    max-width: 280px;
    font-size: 0.85rem;
  }
}

@keyframes lobby-pulse {
  0%, 100% { opacity: 0.6; }
  50% { opacity: 0.3; }
}
</style>
