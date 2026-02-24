<template>
  <div
    class="player-circle"
    ref="circle"
    :style="circleStyle"
  >
    <PlayerNode
      v-for="(player, index) in sortedPlayers"
      :key="player.seatIndex"
      :player="player"
      :angle="getAngle(index)"
      :radius="radius"
      @click="onNodeClick"
      @longpress="onNodeLongPress"
    />
  </div>
</template>

<script>
import { mapGetters, mapState } from "vuex";
import PlayerNode from "./PlayerNode";

export default {
  name: "PlayerCircle",
  components: { PlayerNode },
  data() {
    return {
      containerWidth: 300,
      containerHeight: 300
    };
  },
  computed: {
    ...mapGetters("players", ["sorted", "playerCount"]),
    ...mapState("ui", ["isMobile"]),
    sortedPlayers() {
      return this.sorted;
    },
    radius() {
      // Dynamic radius based on container and player count
      const minDim = Math.min(this.containerWidth, this.containerHeight);
      const base = (minDim / 2) - 50; // padding for nodes
      // Scale down slightly for more players
      return Math.max(80, Math.min(base, 160));
    },
    circleStyle() {
      return {
        width: (this.radius * 2 + 100) + 'px',
        height: (this.radius * 2 + 100) + 'px'
      };
    }
  },
  methods: {
    getAngle(index) {
      if (this.playerCount === 0) return 0;
      return (360 / this.playerCount) * index;
    },
    onNodeClick(player) {
      this.$emit("node-click", player);
    },
    onNodeLongPress(player) {
      this.$emit("node-longpress", player);
    },
    updateDimensions() {
      if (this.$refs.circle && this.$refs.circle.parentElement) {
        const parent = this.$refs.circle.parentElement;
        this.containerWidth = parent.clientWidth;
        this.containerHeight = parent.clientHeight - 100; // minus padding for alive counter
      }
    }
  },
  mounted() {
    this.updateDimensions();
    window.addEventListener("resize", this.updateDimensions);
  },
  beforeDestroy() {
    window.removeEventListener("resize", this.updateDimensions);
  }
};
</script>

<style lang="scss" scoped>
.player-circle {
  display: flex;
  align-items: center;
  justify-content: center;
  margin: 0 auto;
  flex-shrink: 0;
}
</style>
