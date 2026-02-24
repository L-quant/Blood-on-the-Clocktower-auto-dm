<template>
  <transition name="slide-up">
    <div class="action-sheet-overlay" v-if="isOpen" @click.self="close">
      <div class="action-sheet">
        <div class="action-sheet__handle" @click="close"></div>

        <!-- Player header -->
        <div class="action-sheet__header">
          <div class="action-sheet__player-info">
            <span class="action-sheet__seat">{{ $t('square.seat', { n: seatIndex }) }}</span>
            <span class="action-sheet__status" :class="{ dead: !isAlive }">
              {{ isAlive ? '' : '☠️ ' + $t('square.ghost') }}
            </span>
          </div>
          <button class="action-sheet__close-btn" @click="close">
            <font-awesome-icon icon="times" />
          </button>
        </div>

        <!-- Current annotation display -->
        <div class="action-sheet__current" v-if="annotation">
          <span class="action-sheet__current-label">
            {{ annotation.guessedRoleId || annotation.guessedTeam || '' }}
          </span>
          <span class="action-sheet__current-note" v-if="annotation.note">
            {{ annotation.note }}
          </span>
        </div>

        <!-- Action mode: default -->
        <div v-if="mode === 'default'" class="action-sheet__actions">
          <button class="action-sheet__action" @click="mode = 'annotate'">
            <font-awesome-icon icon="user-edit" />
            <span>{{ $t('square.annotateRole') }}</span>
          </button>
          <button class="action-sheet__action" @click="mode = 'note'">
            <font-awesome-icon icon="clipboard" />
            <span>{{ $t('square.addNote') }}</span>
          </button>
          <button
            class="action-sheet__action nominate"
            v-if="canNominate"
            @click="nominate"
          >
            <font-awesome-icon icon="exclamation-triangle" />
            <span>{{ $t('square.nominatePlayer') }}</span>
          </button>
          <button
            class="action-sheet__action clear"
            v-if="annotation"
            @click="clearAnnotation"
          >
            <font-awesome-icon icon="times-circle" />
            <span>{{ $t('square.clearAnnotation') }}</span>
          </button>
        </div>

        <!-- Action mode: annotate role -->
        <div v-if="mode === 'annotate'" class="action-sheet__annotate">
          <button class="action-sheet__back" @click="mode = 'default'">
            ← {{ $t('annotation.cancel') }}
          </button>
          <RoleAnnotator
            :currentRoleId="annotation ? annotation.guessedRoleId : ''"
            :currentTeam="annotation ? annotation.guessedTeam : ''"
            @select="onRoleSelect"
          />
        </div>

        <!-- Action mode: note -->
        <div v-if="mode === 'note'" class="action-sheet__note">
          <button class="action-sheet__back" @click="mode = 'default'">
            ← {{ $t('annotation.cancel') }}
          </button>
          <textarea
            ref="noteInput"
            class="action-sheet__note-input"
            v-model="noteText"
            :placeholder="$t('annotation.notePlaceholder')"
            rows="3"
          ></textarea>
          <button class="button townsfolk action-sheet__save-btn" @click="saveNote">
            {{ $t('annotation.save') }}
          </button>
        </div>
      </div>
    </div>
  </transition>
</template>

<script>
import { mapState, mapGetters } from "vuex";
import RoleAnnotator from "./RoleAnnotator";

export default {
  name: "PlayerActionSheet",
  components: { RoleAnnotator },
  data() {
    return {
      mode: "default", // 'default' | 'annotate' | 'note'
      noteText: ""
    };
  },
  computed: {
    ...mapState("ui", {
      modal: state => state.modals.playerAction
    }),
    ...mapState("game", ["phase"]),
    ...mapState(["seatIndex"]),
    ...mapGetters("annotations", ["forSeat"]),
    ...mapGetters("players", ["me"]),
    isOpen() {
      return this.modal && this.modal.open;
    },
    targetSeatIndex() {
      return this.modal ? this.modal.seatIndex : -1;
    },
    seatIndexLabel() {
      return this.targetSeatIndex;
    },
    annotation() {
      return this.forSeat(this.targetSeatIndex);
    },
    targetPlayer() {
      const players = this.$store.state.players.players;
      return players.find(p => p.seatIndex === this.targetSeatIndex);
    },
    isAlive() {
      return this.targetPlayer ? this.targetPlayer.isAlive : true;
    },
    canNominate() {
      if (this.phase !== 'day') return false;
      if (!this.me) return false;
      if (!this.me.isAlive) return false;
      if (this.me.hasNominatedToday) return false;
      if (!this.targetPlayer) return false;
      if (this.targetPlayer.isNominatedToday) return false;
      return true;
    }
  },
  watch: {
    isOpen(val) {
      if (val) {
        this.mode = "default";
        this.noteText = (this.annotation && this.annotation.note) || "";
      }
    },
    mode(val) {
      if (val === "note") {
        this.$nextTick(() => {
          if (this.$refs.noteInput) {
            this.$refs.noteInput.focus();
          }
        });
      }
    }
  },
  methods: {
    close() {
      this.$store.commit("ui/closeModal", "playerAction");
    },
    onRoleSelect({ guessedRoleId, guessedTeam }) {
      this.$store.commit("annotations/setGuessedRole", {
        seatIndex: this.targetSeatIndex,
        guessedRoleId,
        guessedTeam
      });
      this.mode = "default";
    },
    saveNote() {
      this.$store.commit("annotations/updateNote", {
        seatIndex: this.targetSeatIndex,
        note: this.noteText
      });
      this.mode = "default";
    },
    clearAnnotation() {
      this.$store.commit("annotations/clearAnnotation", this.targetSeatIndex);
    },
    nominate() {
      this.$store.dispatch("nominate", this.targetSeatIndex);
      this.close();
    }
  }
};
</script>

<style lang="scss" scoped>
@import "../vars";

.action-sheet-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: flex-end;
  justify-content: center;
  z-index: 150;
}

.action-sheet {
  width: 100%;
  max-width: 420px;
  max-height: 80vh;
  background: rgba(20, 20, 30, 0.95);
  border-top-left-radius: 20px;
  border-top-right-radius: 20px;
  padding: 12px 20px 32px;
  border-top: 2px solid rgba(255, 255, 255, 0.15);
  overflow-y: auto;
  -webkit-overflow-scrolling: touch;

  &__handle {
    width: 40px;
    height: 4px;
    background: rgba(255, 255, 255, 0.2);
    border-radius: 2px;
    margin: 0 auto 12px;
    cursor: pointer;
  }

  &__header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 12px;
  }

  &__player-info {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  &__seat {
    font-size: 1.1rem;
    font-weight: bold;
    font-family: PiratesBay, sans-serif;
  }

  &__status {
    font-size: 0.75rem;
    opacity: 0.5;

    &.dead {
      color: $demon;
      opacity: 0.8;
    }
  }

  &__close-btn {
    background: none;
    border: none;
    color: white;
    font-size: 1.1rem;
    cursor: pointer;
    opacity: 0.5;
    padding: 4px;

    &:hover { opacity: 1; }
  }

  &__current {
    background: rgba(255, 255, 255, 0.05);
    border-radius: 8px;
    padding: 8px 12px;
    margin-bottom: 12px;
    font-size: 0.8rem;
  }

  &__current-label {
    opacity: 0.7;
  }

  &__current-note {
    display: block;
    margin-top: 4px;
    opacity: 0.5;
    font-style: italic;
  }

  &__actions {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  &__action {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 14px 16px;
    border: none;
    border-radius: 10px;
    background: rgba(255, 255, 255, 0.06);
    color: white;
    font-size: 0.9rem;
    cursor: pointer;
    transition: all 200ms;
    text-align: left;

    &:hover {
      background: rgba(255, 255, 255, 0.1);
    }

    &:active {
      transform: scale(0.98);
    }

    svg {
      width: 18px;
      opacity: 0.6;
    }

    &.nominate {
      color: $demon;
      svg { color: $demon; opacity: 0.8; }
    }

    &.clear {
      opacity: 0.5;
      font-size: 0.8rem;
    }
  }

  &__back {
    background: none;
    border: none;
    color: rgba(255, 255, 255, 0.5);
    font-size: 0.8rem;
    cursor: pointer;
    padding: 4px 0;
    margin-bottom: 8px;

    &:hover { color: white; }
  }

  &__note-input {
    width: 100%;
    background: rgba(255, 255, 255, 0.08);
    border: 1px solid rgba(255, 255, 255, 0.15);
    border-radius: 8px;
    padding: 12px;
    color: white;
    font-size: 0.85rem;
    resize: none;
    outline: none;
    font-family: inherit;
    margin-bottom: 12px;

    &:focus {
      border-color: $townsfolk;
    }

    &::placeholder {
      color: rgba(255, 255, 255, 0.2);
    }
  }

  &__save-btn {
    width: 100%;
    font-size: 0.9rem;
  }
}

.slide-up-enter-active,
.slide-up-leave-active {
  transition: opacity 200ms, transform 300ms ease-out;
  .action-sheet {
    transition: transform 300ms ease-out;
  }
}
.slide-up-enter,
.slide-up-leave-to {
  opacity: 0;
  .action-sheet {
    transform: translateY(100%);
  }
}
</style>
