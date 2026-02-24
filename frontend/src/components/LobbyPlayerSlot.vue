<template>
  <div
    class="player-slot"
    :class="{
      occupied: isOccupied,
      'is-me': isMe,
      'is-owner': isOwner,
      empty: !isOccupied
    }"
    @click="handleClick"
  >
    <div class="player-slot__number">{{ $t('lobby.seat', { n: seatIndex }) }}</div>
    <div class="player-slot__icon">
      <span v-if="isMe" class="player-slot__star">&#11088;</span>
      <span v-else-if="isOccupied" class="player-slot__dot">&#9679;</span>
      <span v-else class="player-slot__empty">{{ $t('lobby.empty') }}</span>
    </div>
    <div class="player-slot__label" v-if="isMe">
      {{ $t('lobby.you') }}
      <span v-if="isOwner">({{ $t('lobby.owner') }})</span>
    </div>
  </div>
</template>

<script>
export default {
  name: "LobbyPlayerSlot",
  props: {
    seatIndex: { type: Number, required: true },
    isOccupied: { type: Boolean, default: false },
    isMe: { type: Boolean, default: false },
    isOwner: { type: Boolean, default: false }
  },
  methods: {
    handleClick() {
      this.$emit("click", this.seatIndex);
    }
  }
};
</script>

<style lang="scss" scoped>
@import "../vars";

.player-slot {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  width: 72px;
  height: 72px;
  border-radius: 12px;
  cursor: pointer;
  transition: all 200ms;
  gap: 2px;

  &.empty {
    border: 2px dashed rgba(255, 255, 255, 0.2);
    background: rgba(255, 255, 255, 0.03);

    &:hover {
      border-color: rgba(255, 255, 255, 0.4);
      background: rgba(255, 255, 255, 0.06);
    }

    &:active {
      transform: scale(0.95);
    }
  }

  &.occupied {
    border: 2px solid rgba(255, 255, 255, 0.3);
    background: rgba(255, 255, 255, 0.08);
  }

  &.is-me {
    border-color: $townsfolk;
    background: rgba($townsfolk, 0.1);
  }

  &__number {
    font-size: 0.65rem;
    opacity: 0.6;
    white-space: nowrap;
  }

  &__icon {
    font-size: 1.2rem;
    line-height: 1;
  }

  &__star {
    font-size: 1.1rem;
  }

  &__dot {
    color: rgba(255, 255, 255, 0.6);
  }

  &__empty {
    font-size: 0.65rem;
    opacity: 0.3;
  }

  &__label {
    font-size: 0.55rem;
    opacity: 0.7;
    white-space: nowrap;
  }
}
</style>
