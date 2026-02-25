<!-- SquareView 城镇广场容器
  [IN]  PlayerCircle（玩家圆圈布局）
  [IN]  AliveCounter（存活计数）
  [IN]  PlayerActionSheet（玩家操作面板）
  [OUT] GameScreen.vue（主布局子组件）
  [POS] 游戏核心视图，展示玩家围坐的城镇广场 -->
<template>
  <div class="square-view">
    <!-- Player circle -->
    <div class="square-view__circle-container">
      <PlayerCircle
        @node-click="onNodeClick"
        @node-longpress="onNodeLongPress"
      />
    </div>

    <!-- Alive/dead counter -->
    <AliveCounter />

    <!-- Player action sheet -->
    <PlayerActionSheet />
  </div>
</template>

<script>
import PlayerCircle from "./PlayerCircle";
import AliveCounter from "./AliveCounter";
import PlayerActionSheet from "./PlayerActionSheet";

export default {
  name: "SquareView",
  components: { PlayerCircle, AliveCounter, PlayerActionSheet },
  methods: {
    onNodeClick(player) {
      if (player.isMe) {
        // Show own role card
        this.$store.commit("ui/setActiveTab", "me");
      } else {
        // Open action sheet for this player
        this.$store.commit("ui/openModal", {
          modal: "playerAction",
          data: { seatIndex: player.seatIndex }
        });
      }
    },
    onNodeLongPress(player) {
      if (!player.isMe) {
        // Quick whisper
        this.$store.commit("chat/setActiveChannel", "whisper");
        this.$store.commit("chat/setActiveWhisperTarget", player.seatIndex);
        this.$store.commit("ui/setActiveTab", "chat");
      }
    }
  }
};
</script>

<style lang="scss" scoped>
.square-view {
  height: 100%;
  display: flex;
  flex-direction: column;
  padding: 12px 0;

  &__circle-container {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    overflow: hidden;
  }
}
</style>
