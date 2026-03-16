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

    <div class="square-view__night-targeting" v-if="isNightSelecting">
      <p class="square-view__night-hint">
        {{ $t('night.selectionHint', { action: actionVerb }) }}
      </p>
      <p class="square-view__night-progress">
        {{ $t('night.selectedCount', { count: selectedCount, required: requiredCount }) }}
      </p>
      <div class="square-view__night-actions">
        <button
          class="square-view__night-btn"
          :class="{ disabled: !canSubmitNightSelection }"
          @click="submitNightSelection"
        >
          {{ $t('night.confirm') }}
        </button>
        <button
          v-if="showNightSkipAction"
          class="square-view__night-btn square-view__night-btn--skip"
          @click="skipNightSelection"
        >
          {{ $t('night.skip') }}
        </button>
      </div>
    </div>

    <div class="square-view__phase-action" v-if="canAdvanceToNight">
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
    ...mapState("players", ["myRole"]),
    ...mapState("night", ["step", "targets", "selectedTargets", "roleId", "actionType"]),
    extensionsRemaining() {
      return this.maxExtensions - this.extensionsUsed;
    },
    isNightSelecting() {
      return this.step === 'selecting';
    },
    requiredCount() {
      if (this.actionType === 'select_two') return 2;
      if (this.actionType === 'select_one') return 1;
      return 0;
    },
    selectedCount() {
      return Array.isArray(this.selectedTargets) ? this.selectedTargets.length : 0;
    },
    canSubmitNightSelection() {
      return this.requiredCount > 0 && this.selectedCount === this.requiredCount;
    },
    showNightSkipAction() {
      const isEvilTeam = !!this.myRole && this.myRole.team === 'evil';
      return !isEvilTeam;
    },
    actionVerb() {
      const roleVerbMap = {
        imp: '击杀',
        poisoner: '投毒',
        monk: '保护',
        ravenkeeper: '查验',
        fortuneteller: '查验',
        washerwoman: '查验',
        librarian: '查验',
        investigator: '查验',
        butler: '指定'
      };
      const mapped = roleVerbMap[this.roleId] || (this.actionType === 'select_two' ? '查验' : '选择');
      return mapped;
    },
    canAdvanceToNight() {
      const isDay = this.phase === 'day' || this.phase === 'nomination';
      const noActiveVote = this.voteSubPhase === 'none' || this.voteSubPhase === 'resolved';
      return isDay && noActiveVote && this.$store.getters.isRoomOwner;
    }
  },
  methods: {
    isNightTargetSelectable(player) {
      if (!this.isNightSelecting || !Array.isArray(this.targets)) return false;
      return this.targets.some(target => target && target.seatIndex === player.seatIndex);
    },
    getNightSelectedTargetBySeat(seatIndex) {
      if (!Array.isArray(this.selectedTargets)) return null;
      return this.selectedTargets.find(target => target && target.seatIndex === seatIndex) || null;
    },
    submitNightSelection() {
      if (!this.canSubmitNightSelection) return;
      this.$store.dispatch('sendNightAction', { targets: this.selectedTargets });
    },
    skipNightSelection() {
      this.$store.dispatch('sendNightAction', { targets: [] });
    },
    onNodeClick(player) {
      if (this.isNightSelecting) {
        if (!this.isNightTargetSelectable(player)) return;
        const selectedTarget = this.getNightSelectedTargetBySeat(player.seatIndex);
        if (selectedTarget) {
          this.$store.commit('night/removeTarget', selectedTarget);
          return;
        }
        const target = this.targets.find(item => item && item.seatIndex === player.seatIndex);
        if (!target) return;
        this.$store.commit('night/selectTarget', target);
        return;
      }
      // Open action sheet for any player (including self — allows self-nomination)
      this.$store.commit("ui/openModal", {
        modal: "playerAction",
        data: { seatIndex: player.seatIndex }
      });
    },
    onNodeLongPress(player) {
      if (this.isNightSelecting) return;
      if (!player.isMe) {
        this.$store.commit("chat/setActiveChannel", "whisper");
        this.$store.commit("chat/setActiveWhisperTarget", player.seatIndex);
        this.$store.commit("ui/setActiveTab", "chat");
      }
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

  &__night-targeting {
    margin: 8px 12px 0;
    padding: 10px 12px;
    border-radius: 10px;
    border: 1px solid rgba($townsfolk, 0.45);
    background: rgba(0, 0, 0, 0.45);
    text-align: center;
  }

  &__night-hint {
    margin: 0;
    font-size: 0.82rem;
  }

  &__night-progress {
    margin: 6px 0 0;
    font-size: 0.72rem;
    opacity: 0.75;
  }

  &__night-actions {
    margin-top: 8px;
    display: flex;
    gap: 8px;
    justify-content: center;
  }

  &__night-btn {
    min-width: 88px;
    padding: 6px 10px;
    border-radius: 8px;
    border: 1px solid rgba($townsfolk, 0.55);
    color: white;
    background: rgba($townsfolk, 0.14);
    cursor: pointer;
    transition: all 160ms ease;

    &.disabled {
      opacity: 0.35;
      cursor: not-allowed;
    }
  }

  &__night-btn--skip {
    border-color: rgba(255, 255, 255, 0.25);
    background: rgba(255, 255, 255, 0.06);
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
