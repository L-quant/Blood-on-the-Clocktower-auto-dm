<!-- SkillsView 技能面板：提供白天公共技能宣称入口
  [IN]  store（角色、阶段、玩家列表）
  [OUT] GameScreen.vue（主布局右侧/移动端标签页）
  [POS] 白天技能交互面板，当前支持暗流涌动剧本的猎手宣称开枪 -->
<template>
  <div class="skills-view">
    <div class="skills-view__card">
      <div class="skills-view__header">
        <h3 class="skills-view__title">{{ $t('skills.title') }}</h3>
        <span class="skills-view__status" v-if="hasUsedSlayerSkill">
          {{ $t('skills.used') }}
        </span>
      </div>

      <p class="skills-view__hint" v-if="!isTroubleBrewing">
        {{ $t('skills.onlyTb') }}
      </p>
      <template v-else>
        <p class="skills-view__hint">
          {{ $t('skills.slayerHint') }}
        </p>

        <button
          class="skills-view__action"
          :class="{ disabled: !canUseSlayerAction || submitting }"
          @click="toggleTargetPicker"
        >
          {{ actionLabel }}
        </button>

        <p class="skills-view__subhint" v-if="!isDaytime">
          {{ $t('skills.dayOnly') }}
        </p>

        <div class="skills-view__targets" v-if="showTargetPicker && canUseSlayerAction">
          <span class="skills-view__target-title">{{ $t('skills.selectTarget') }}</span>
          <div class="skills-view__target-grid" v-if="targetPlayers.length > 0">
            <button
              v-for="player in targetPlayers"
              :key="player.id"
              class="skills-view__target"
              @click="useSlayerShot(player.seatIndex)"
            >
              {{ $t('square.seat', { n: player.seatIndex }) }}
            </button>
          </div>
          <p class="skills-view__subhint" v-else>
            {{ $t('skills.noTargets') }}
          </p>
        </div>

        <p class="skills-view__error" v-if="errorMessage">{{ errorMessage }}</p>
      </template>
    </div>
  </div>
</template>

<script>
import { mapState } from "vuex";

export default {
  name: "SkillsView",
  data() {
    return {
      showTargetPicker: false,
      submitting: false,
      errorMessage: ""
    };
  },
  computed: {
    ...mapState(["edition"]),
    ...mapState("players", ["myRole", "players"]),
    ...mapState("game", ["phase"]),
    isTroubleBrewing() {
      return !!this.edition && this.edition.id === "tb";
    },
    isDaytime() {
      return ["day", "nomination", "voting"].includes(this.phase);
    },
    isRealSlayer() {
      return !!this.myRole && this.myRole.roleId === "slayer";
    },
    hasUsedSlayerSkill() {
      const reminders = this.myRole && Array.isArray(this.myRole.reminders)
        ? this.myRole.reminders
        : [];
      return reminders.includes("slayer_claim_used") || reminders.includes("no_ability") || reminders.includes("无能力");
    },
    canUseSlayerAction() {
      return this.isTroubleBrewing && this.isDaytime && !this.submitting && !this.hasUsedSlayerSkill;
    },
    targetPlayers() {
      return [...this.players]
        .filter(player => player.isAlive && !player.isMe)
        .sort((left, right) => left.seatIndex - right.seatIndex);
    },
    actionLabel() {
      if (this.submitting) return this.$t('skills.submitting');
      if (this.hasUsedSlayerSkill) return this.$t('skills.used');
      return this.$t('skills.fireSlayer');
    }
  },
  methods: {
    toggleTargetPicker() {
      if (!this.canUseSlayerAction) return;
      this.errorMessage = "";
      this.showTargetPicker = !this.showTargetPicker;
    },
    async useSlayerShot(seatIndex) {
      if (!this.canUseSlayerAction) return;
      this.submitting = true;
      this.errorMessage = "";
      try {
        await this.$store.dispatch("useSlayerShot", seatIndex);
        this.showTargetPicker = false;
        this.$store.commit("ui/setActiveTab", "timeline");
      } catch (error) {
        this.errorMessage = error && error.message ? error.message : this.$t('skills.failed');
      } finally {
        this.submitting = false;
      }
    }
  }
};
</script>

<style lang="scss" scoped>
@import "../vars";

.skills-view {
  height: 100%;
  overflow-y: auto;
  padding: 16px;
  -webkit-overflow-scrolling: touch;

  &__card {
    background: rgba(0, 0, 0, 0.35);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 12px;
    padding: 16px;
  }

  &__header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    margin-bottom: 12px;
  }

  &__title {
    font-size: 1rem;
    text-align: left;
  }

  &__status {
    color: $outsider;
    font-size: 0.75rem;
    letter-spacing: 1px;
    text-transform: uppercase;
  }

  &__hint,
  &__subhint,
  &__error {
    margin: 0 0 12px;
    font-size: 0.82rem;
    line-height: 1.5;
  }

  &__hint,
  &__subhint {
    opacity: 0.7;
  }

  &__error {
    color: $demon;
  }

  &__action,
  &__target {
    border: 1px solid rgba(255, 255, 255, 0.15);
    border-radius: 10px;
    background: rgba(255, 255, 255, 0.06);
    color: white;
    cursor: pointer;
    transition: all 200ms;
  }

  &__action {
    width: 100%;
    padding: 12px 16px;
    font-size: 0.9rem;
    text-align: left;

    &.disabled {
      opacity: 0.35;
      cursor: not-allowed;
    }

    &:not(.disabled):hover {
      border-color: rgba($townsfolk, 0.6);
      background: rgba($townsfolk, 0.12);
    }
  }

  &__targets {
    margin-top: 16px;
  }

  &__target-title {
    display: block;
    margin-bottom: 10px;
    font-size: 0.72rem;
    letter-spacing: 1px;
    opacity: 0.5;
    text-transform: uppercase;
  }

  &__target-grid {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 8px;
  }

  &__target {
    padding: 10px 8px;
    font-size: 0.82rem;

    &:hover {
      border-color: rgba($demon, 0.7);
      background: rgba($demon, 0.14);
    }
  }
}
</style>