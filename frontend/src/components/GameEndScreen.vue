<template>
  <div class="game-end">
    <div class="game-end__content">
      <h1 class="game-end__title">{{ $t('game.gameOver') }}</h1>
      <div
        class="game-end__result"
        :class="winner"
      >
        <p class="game-end__winner-text">
          {{ winner === 'good' ? $t('game.goodWins') : $t('game.evilWins') }}
        </p>
        <p class="game-end__reason" v-if="winReason">{{ winReason }}</p>
      </div>
      <div class="game-end__actions">
        <button class="button townsfolk" @click="backToLobby">
          {{ $t('game.backToLobby') }}
        </button>
        <button class="button" @click="backToHome">
          {{ $t('game.newGame') }}
        </button>
      </div>
    </div>
  </div>
</template>

<script>
import { mapState } from "vuex";

export default {
  name: "GameEndScreen",
  computed: {
    ...mapState("game", ["winner", "winReason"])
  },
  methods: {
    backToLobby() {
      this.$store.commit("game/reset");
      this.$store.commit("ui/setScreen", "lobby");
    },
    backToHome() {
      this.$store.dispatch("resetAll");
    }
  }
};
</script>

<style lang="scss" scoped>
@import "../vars";

.game-end {
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;

  &__content {
    text-align: center;
    max-width: 400px;
  }

  &__title {
    font-size: 2rem;
    margin-bottom: 24px;
  }

  &__result {
    padding: 24px;
    border-radius: 12px;
    margin-bottom: 32px;

    &.good {
      background: rgba($townsfolk, 0.15);
      border: 2px solid $townsfolk;
    }
    &.evil {
      background: rgba($demon, 0.15);
      border: 2px solid $demon;
    }
  }

  &__winner-text {
    font-size: 1.3rem;
    font-family: PiratesBay, sans-serif;
    letter-spacing: 1px;
    margin: 0 0 8px;
  }

  &__reason {
    font-size: 0.85rem;
    opacity: 0.7;
    margin: 0;
  }

  &__actions {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }
}
</style>
