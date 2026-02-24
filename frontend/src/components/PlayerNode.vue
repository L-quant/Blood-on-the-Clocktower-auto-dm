<template>
  <div
    class="player-node"
    :class="nodeClasses"
    :style="nodeStyle"
    role="button"
    :tabindex="0"
    :aria-label="$t('square.seat', { n: player.seatIndex }) + (player.isMe ? ' (' + $t('square.me') + ')' : '') + (!player.isAlive ? ' (' + $t('square.ghost') + ')' : '')"
    @click="handleClick"
    @keydown.enter="handleClick"
    @touchstart.passive="onTouchStart"
    @touchend.passive="onTouchEnd"
  >
    <!-- Token circle -->
    <div class="player-node__token">
      <!-- Role icon (only self or annotated) -->
      <img
        v-if="showRoleIcon"
        class="player-node__icon"
        :src="roleIconSrc"
        :alt="roleLabel"
      />
      <!-- Unknown marker -->
      <span v-else class="player-node__unknown">?</span>

      <!-- Death shroud -->
      <img
        v-if="!player.isAlive"
        class="player-node__shroud"
        src="../assets/shroud.png"
        alt="dead"
      />

      <!-- Ghost vote indicator -->
      <span
        v-if="!player.isAlive && player.hasGhostVote"
        class="player-node__ghost-vote"
        :title="$t('square.ghostVote')"
      >&#x1F44B;</span>
    </div>

    <!-- Seat label -->
    <span class="player-node__seat">
      {{ isMe ? $t('square.me') : '' }}
      {{ $t('square.seat', { n: player.seatIndex }) }}
    </span>

    <!-- Annotation note indicator -->
    <span
      v-if="hasNote"
      class="player-node__note-dot"
      :title="annotation.note"
    ></span>

    <!-- Nomination pulse -->
    <div
      v-if="isNominated"
      class="player-node__nomination-pulse"
    ></div>
  </div>
</template>

<script>
import { mapState, mapGetters } from "vuex";

export default {
  name: "PlayerNode",
  props: {
    player: { type: Object, required: true },
    angle: { type: Number, default: 0 }, // angle in degrees for circle positioning
    radius: { type: Number, default: 120 } // circle radius in px
  },
  data() {
    return {
      touchTimer: null
    };
  },
  computed: {
    ...mapState("players", ["myRole"]),
    ...mapGetters("annotations", ["forSeat"]),
    ...mapGetters("vote", ["isNominated"]),
    isMe() {
      return this.player.isMe;
    },
    annotation() {
      return this.forSeat(this.player.seatIndex);
    },
    hasNote() {
      return this.annotation && this.annotation.note;
    },
    showRoleIcon() {
      if (this.isMe && this.myRole) return true;
      if (this.annotation && this.annotation.guessedRoleId) return true;
      return false;
    },
    roleIconSrc() {
      if (this.isMe && this.myRole) {
        try {
          return require(`../assets/icons/${this.myRole.roleId}.png`);
        } catch (e) {
          return '';
        }
      }
      if (this.annotation && this.annotation.guessedRoleId) {
        try {
          return require(`../assets/icons/${this.annotation.guessedRoleId}.png`);
        } catch (e) {
          return '';
        }
      }
      return '';
    },
    roleLabel() {
      if (this.isMe && this.myRole) {
        const key = 'roles.' + this.myRole.roleId;
        return this.$te(key) ? this.$t(key) : this.myRole.roleName;
      }
      if (this.annotation && this.annotation.guessedRoleId) {
        const key = 'roles.' + this.annotation.guessedRoleId;
        return this.$te(key) ? this.$t(key) : this.annotation.guessedRoleId;
      }
      return '?';
    },
    teamColor() {
      if (this.isMe && this.myRole) return this.myRole.team;
      if (this.annotation && this.annotation.guessedTeam) return this.annotation.guessedTeam;
      return '';
    },
    isAnnotated() {
      return !this.isMe && this.annotation && (this.annotation.guessedRoleId || this.annotation.guessedTeam);
    },
    nodeClasses() {
      return {
        'is-me': this.isMe,
        'is-dead': !this.player.isAlive,
        'is-annotated': this.isAnnotated,
        'is-nominated': this.isNominated(this.player.seatIndex),
        [`team-${this.teamColor}`]: !!this.teamColor
      };
    },
    nodeStyle() {
      // Position on circle using angle and radius
      const rad = (this.angle - 90) * (Math.PI / 180);
      const x = Math.cos(rad) * this.radius;
      const y = Math.sin(rad) * this.radius;
      return {
        transform: `translate(${x}px, ${y}px)`
      };
    }
  },
  methods: {
    handleClick() {
      this.$emit("click", this.player);
    },
    onTouchStart() {
      this.touchTimer = setTimeout(() => {
        this.$emit("longpress", this.player);
      }, 500);
    },
    onTouchEnd() {
      clearTimeout(this.touchTimer);
    }
  }
};
</script>

<style lang="scss" scoped>
@import "../vars";

.player-node {
  position: absolute;
  display: flex;
  flex-direction: column;
  align-items: center;
  cursor: pointer;
  transition: transform 0.5s ease;
  user-select: none;
  -webkit-tap-highlight-color: transparent;

  &__token {
    width: 54px;
    height: 54px;
    border-radius: 50%;
    background: rgba(0, 0, 0, 0.6);
    border: 3px solid rgba(255, 255, 255, 0.2);
    display: flex;
    align-items: center;
    justify-content: center;
    overflow: hidden;
    transition: all 200ms;
  }

  &__icon {
    width: 80%;
    height: 80%;
    object-fit: contain;
    pointer-events: none;
  }

  &__unknown {
    font-size: 1.4rem;
    opacity: 0.3;
    font-weight: bold;
  }

  &__shroud {
    position: absolute;
    width: 36px;
    height: auto;
    top: -4px;
    opacity: 0.8;
    pointer-events: none;
    z-index: 2;
  }

  &__ghost-vote {
    position: absolute;
    bottom: -2px;
    right: -2px;
    font-size: 0.7rem;
    z-index: 3;
  }

  &__seat {
    margin-top: 4px;
    font-size: 0.55rem;
    opacity: 0.7;
    white-space: nowrap;
    text-align: center;
    line-height: 1.1;
  }

  &__note-dot {
    position: absolute;
    top: 0;
    right: 0;
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: $fabled;
    z-index: 3;
  }

  &__nomination-pulse {
    position: absolute;
    top: -3px;
    left: -3px;
    right: -3px;
    bottom: -3px;
    border-radius: 50%;
    border: 3px solid $demon;
    animation: nomination-pulse 1s ease-in-out infinite;
    pointer-events: none;
  }

  // States
  &.is-me .player-node__token {
    border-width: 3px;
    border-style: solid;
  }

  &.is-annotated .player-node__token {
    border-width: 3px;
    border-style: dashed;
    opacity: 0.85;
  }

  &.is-dead {
    opacity: 0.5;
    .player-node__token {
      filter: grayscale(50%);
    }
  }

  &.is-nominated .player-node__token {
    border-color: $demon;
  }

  // Team colors
  &.team-townsfolk .player-node__token { border-color: $townsfolk; }
  &.team-outsider .player-node__token { border-color: $outsider; }
  &.team-minion .player-node__token { border-color: $minion; }
  &.team-demon .player-node__token { border-color: $demon; }
  &.team-traveler .player-node__token { border-color: $traveler; }

  &:active {
    .player-node__token {
      transform: scale(0.9);
    }
  }
}

@keyframes nomination-pulse {
  0%, 100% { opacity: 1; transform: scale(1); }
  50% { opacity: 0.5; transform: scale(1.1); }
}
</style>
