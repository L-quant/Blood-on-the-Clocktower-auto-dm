<template>
  <div class="alive-counter">
    <div class="alive-counter__bar">
      <span class="alive-counter__alive">
        <font-awesome-icon icon="user" />
        {{ $t('game.alive') }}: {{ aliveCount }}
      </span>
      <span class="alive-counter__separator">|</span>
      <span class="alive-counter__dead">
        <font-awesome-icon icon="skull" />
        {{ $t('game.dead') }}: {{ deadCount }}
      </span>
    </div>
  </div>
</template>

<script>
import { mapGetters } from "vuex";

export default {
  name: "AliveCounter",
  computed: {
    ...mapGetters("players", ["aliveCount", "playerCount"]),
    deadCount() {
      return this.playerCount - this.aliveCount;
    }
  }
};
</script>

<style lang="scss" scoped>
@import "../vars";

.alive-counter {
  padding: 0 16px;

  &__bar {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 12px;
    padding: 8px 16px;
    background: rgba(0, 0, 0, 0.5);
    border-radius: 20px;
    font-size: 0.75rem;
    opacity: 0.8;
  }

  &__alive {
    color: $townsfolk;
    display: flex;
    align-items: center;
    gap: 4px;
  }

  &__dead {
    color: $demon;
    display: flex;
    align-items: center;
    gap: 4px;
  }

  &__separator {
    opacity: 0.2;
  }
}
</style>
