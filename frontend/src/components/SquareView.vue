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

    <!-- Phase action: extend discussion / advance to night -->
    <div class="square-view__phase-action" v-if="canExtendTime || canAdvanceToNight">
      <button
        v-if="canExtendTime"
        class="square-view__extend-btn"
        @click="extendTime"
      >
        {{ $t('game.extendTime') }}
        <span class="square-view__extend-count">
          {{ $t('game.extensionsRemaining', { count: extensionsRemaining }) }}
        </span>
      </button>
      <button
        v-if="canAdvanceToNight"
        class="square-view__advance-btn"
        @click="advanceToNight"
      >
        {{ $t('game.advanceToNight') }}
      </button>
    </div>

    <!-- Player action sheet -->
    <PlayerActionSheet />
  </div>
</template>

<script>
import { mapState } from "vuex";
import PlayerCircle from "./PlayerCircle";
import AliveCounter from "./AliveCounter";
import PlayerActionSheet from "./PlayerActionSheet";

export default {
  name: "SquareView",
  components: { PlayerCircle, AliveCounter, PlayerActionSheet },
  computed: {
    ...mapState("game", ["phase", "extensionsUsed", "maxExtensions"]),
    ...mapState("vote", { voteSubPhase: "subPhase" }),
    extensionsRemaining() {
      return this.maxExtensions - this.extensionsUsed;
    },
    canExtendTime() {
      return this.phase === 'day' && this.extensionsRemaining > 0;
    },
    canAdvanceToNight() {
      const isDay = this.phase === 'day' || this.phase === 'nomination';
      const noActiveVote = this.voteSubPhase === 'none' || this.voteSubPhase === 'resolved';
      return isDay && noActiveVote && this.$store.state.isRoomOwner;
    }
  },
  methods: {
    onNodeClick(player) {
      if (player.isMe) {
        this.$store.commit("ui/setActiveTab", "me");
      } else {
        this.$store.commit("ui/openModal", {
          modal: "playerAction",
          data: { seatIndex: player.seatIndex }
        });
      }
    },
    onNodeLongPress(player) {
      if (!player.isMe) {
        this.$store.commit("chat/setActiveChannel", "whisper");
        this.$store.commit("chat/setActiveWhisperTarget", player.seatIndex);
        this.$store.commit("ui/setActiveTab", "chat");
      }
    },
    extendTime() {
      this.$store.commit("sendCommand", { type: "extend_time", data: {} });
    },
    advanceToNight() {
      this.$store.dispatch("advancePhase", "night");
    }
  }
};
</script>

<style lang="scss" scoped>
@import "../vars";

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

  &__phase-action {
    text-align: center;
    padding: 8px 16px;
  }

  &__extend-btn {
    padding: 8px 20px;
    border: 2px solid $townsfolk;
    border-radius: 8px;
    background: rgba($townsfolk, 0.1);
    color: white;
    font-size: 0.85rem;
    cursor: pointer;
    transition: all 200ms;
    margin-right: 8px;

    &:active {
      background: rgba($townsfolk, 0.25);
      transform: scale(0.95);
    }
  }

  &__extend-count {
    font-size: 0.75rem;
    opacity: 0.7;
    margin-left: 4px;
  }

  &__advance-btn {
    padding: 10px 24px;
    border: 2px solid $fabled;
    border-radius: 8px;
    background: rgba($fabled, 0.1);
    color: white;
    font-size: 0.9rem;
    cursor: pointer;
    transition: all 200ms;

    &:active {
      background: rgba($fabled, 0.25);
      transform: scale(0.95);
    }
  }
}
</style>
