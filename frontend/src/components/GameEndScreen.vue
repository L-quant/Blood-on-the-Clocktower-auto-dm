<!-- GameEndScreen 结算屏幕（胜方、胜因与复盘展示）
  [OUT] App.vue（屏幕路由，懒加载）
  [POS] 游戏结束画面，展示胜利方、胜因与结算复盘 -->
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
      <div class="game-end__recap" v-if="recap">
        <h2 class="game-end__recap-title">{{ $t('game.recapTitle') }}</h2>
        <p class="game-end__recap-text">{{ recap }}</p>
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
    ...mapState("game", ["winner", "winReason", "recap"])
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

  &__recap {
    text-align: left;
    padding: 18px 20px;
    border-radius: 12px;
    margin-bottom: 24px;
    background: rgba(255, 255, 255, 0.06);
    border: 1px solid rgba(255, 255, 255, 0.12);
  }

  &__recap-title {
    font-size: 0.95rem;
    margin: 0 0 10px;
    letter-spacing: 0.08em;
    text-transform: uppercase;
    opacity: 0.7;
  }

  &__recap-text {
    font-size: 0.9rem;
    line-height: 1.6;
    margin: 0;
    white-space: pre-wrap;
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
