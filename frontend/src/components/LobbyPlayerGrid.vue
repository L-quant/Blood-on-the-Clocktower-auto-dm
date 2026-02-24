<template>
  <div class="player-grid">
    <div class="player-grid__header">
      {{ $t('lobby.seatedPlayers', { count: seatedCount, total: seatCount }) }}
    </div>
    <div class="player-grid__slots">
      <LobbyPlayerSlot
        v-for="seat in seatCount"
        :key="seat"
        :seatIndex="seat"
        :isOccupied="isSeatOccupied(seat)"
        :isMe="isSeatMe(seat)"
        :isOwner="isSeatOwner(seat)"
        @click="onSlotClick"
      />
    </div>
  </div>
</template>

<script>
import { mapState, mapGetters } from "vuex";
import LobbyPlayerSlot from "./LobbyPlayerSlot";

export default {
  name: "LobbyPlayerGrid",
  components: { LobbyPlayerSlot },
  computed: {
    ...mapState(["seatCount", "seatIndex", "isRoomOwner"]),
    ...mapState("players", ["players"]),
    ...mapGetters("players", ["playerCount"]),
    seatedCount() {
      return this.playerCount;
    }
  },
  methods: {
    isSeatOccupied(seat) {
      return this.players.some(p => p.seatIndex === seat);
    },
    isSeatMe(seat) {
      return this.seatIndex === seat;
    },
    isSeatOwner(seat) {
      // The first player seated is typically the room owner
      // But in our system, room owner is tracked separately
      return this.isRoomOwner && this.seatIndex === seat;
    },
    onSlotClick(seatIndex) {
      if (this.isSeatMe(seatIndex)) {
        // Clicking own seat - option to leave
        this.$emit("leave-seat");
      } else if (!this.isSeatOccupied(seatIndex)) {
        // Clicking empty seat - sit down
        this.$emit("seat-down", seatIndex);
      }
    }
  }
};
</script>

<style lang="scss" scoped>
.player-grid {
  &__header {
    text-align: center;
    font-size: 0.8rem;
    opacity: 0.6;
    margin-bottom: 12px;
    letter-spacing: 1px;
  }

  &__slots {
    display: flex;
    flex-wrap: wrap;
    justify-content: center;
    gap: 8px;
    max-width: 340px;
    margin: 0 auto;
  }
}
</style>
