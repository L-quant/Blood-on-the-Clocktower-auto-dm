<!-- NightOverlay 夜晚行动界面：角色揭示→睡眠→唤醒→选择目标→等待→结果
  [OUT] App.vue（全局覆盖层）
  [POS] 夜晚阶段的全屏交互界面 -->
<template>
  <transition name="night-fade">
    <div class="night-overlay" v-if="showOverlay" role="dialog" :aria-label="$t('night.title')" aria-modal="true">

      <!-- Step 0: Role Reveal (first night, only shows role identity) -->
      <div v-if="step === 'role_reveal'" class="night-overlay__step woken">
        <div class="night-overlay__role-icon">
          <img v-if="myRoleIcon" :src="myRoleIcon" :alt="myRoleName" />
        </div>
        <h2 class="night-overlay__wake-title">{{ $t('night.yourRole') }}</h2>
        <p class="night-overlay__role-name">{{ myRoleName }}</p>
        <button class="button townsfolk night-overlay__continue" @click="enterNight">
          {{ $t('night.confirm') }}
        </button>
      </div>

      <!-- Step 0b: Team Reveal (evil team recognition) -->
      <div v-else-if="step === 'team_reveal'" class="night-overlay__step team-reveal">
        <div class="night-overlay__evil-icon">&#x1F608;</div>
        <h2 class="night-overlay__wake-title">{{ $t('teamReveal.title') }}</h2>

        <!-- 数据还未到达时，显示加载动画 -->
        <div v-if="!teamRecognition" class="night-overlay__team-loading">
          <div class="night-overlay__spinner"></div>
          <p class="night-overlay__waiting-text">{{ $t('night.waiting') }}</p>
        </div>

        <!-- Minion view: show demon identity -->
        <template v-else-if="!isDemonView">
          <p class="night-overlay__team-label">{{ $t('teamReveal.yourDemon') }}</p>
          <div class="night-overlay__team-members">
            <div class="night-overlay__team-member demon-member">
              <span class="night-overlay__member-seat">{{ demonSeatLabel }}</span>
            </div>
          </div>
          <p class="night-overlay__team-hint" v-if="fellowMinions.length">
            {{ $t('teamReveal.fellowMinions') }}:
            {{ fellowMinions.map(m => m.label).join(', ') }}
          </p>
        </template>

        <!-- Demon view: show minions + bluffs -->
        <template v-else>
          <p class="night-overlay__team-label">{{ $t('teamReveal.yourMinions') }}</p>
          <div class="night-overlay__team-members">
            <div
              v-for="m in minionSeatLabels"
              :key="m.id"
              class="night-overlay__team-member minion-member"
            >
              <span class="night-overlay__member-seat">{{ m.label }}</span>
            </div>
          </div>
          <div class="night-overlay__bluffs-section" v-if="bluffNames.length">
            <p class="night-overlay__team-label">{{ $t('teamReveal.bluffs') }}</p>
            <div class="night-overlay__bluff-list">
              <span
                v-for="(name, i) in bluffNames"
                :key="i"
                class="night-overlay__bluff-tag"
              >{{ name }}</span>
            </div>
          </div>
        </template>

        <button v-if="teamRecognition" class="button demon night-overlay__continue" @click="dismissTeamReveal">
          {{ $t('night.confirm') }}
        </button>
      </div>

      <!-- Step 1: Sleeping (waiting to be woken) -->
      <div v-else-if="step === 'sleeping'" class="night-overlay__step sleeping">
        <div class="night-overlay__moon">&#x1F319;</div>
        <h2 class="night-overlay__sleep-title">{{ $t('night.sleeping') }}</h2>
        <p class="night-overlay__sleep-hint">{{ $t('night.sleepingHint') }}</p>
        <div class="night-overlay__sleep-dots">
          <span class="dot"></span><span class="dot"></span><span class="dot"></span>
        </div>
      </div>

      <!-- Step 1: Woken up -->
      <div v-else-if="step === 'woken'" class="night-overlay__step woken">
        <div class="night-overlay__role-icon">
          <img
            v-if="roleIcon"
            :src="roleIcon"
            :alt="roleName"
          />
        </div>
        <h2 class="night-overlay__wake-title">{{ $t('night.wakeUp') }}</h2>
        <p class="night-overlay__role-name">{{ localizedRoleName }}</p>
        <p class="night-overlay__ability" v-if="actionType !== 'no_action'">{{ abilityText }}</p>
        <p class="night-overlay__ability" v-else>{{ $t('night.noAction') }}</p>
        <button
          class="button townsfolk night-overlay__continue"
          @click="goToSelecting"
        >
          {{ isAutoAction ? $t('night.confirm') : $t('night.selectTarget') }}
        </button>
      </div>

      <!-- Step 2: Selecting targets -->
      <div v-else-if="step === 'selecting'" class="night-overlay__step selecting">
        <h3 class="night-overlay__select-title">
          {{ actionType === 'select_two' ? $t('night.selectTwoTargets') : $t('night.selectTarget') }}
        </h3>
        <div class="night-overlay__targets">
          <button
            v-for="target in targets"
            :key="target.seatIndex || target"
            class="night-overlay__target"
            :class="{ selected: isSelected(target) }"
            @click="toggleTarget(target)"
          >
            <span class="night-overlay__target-seat">
              {{ $t('square.seat', { n: target.seatIndex || target }) }}
            </span>
          </button>
        </div>
        <div class="night-overlay__selected-info" v-if="selectedTargets.length">
          {{ $t('night.selectTarget') }}: {{ selectedTargets.map(t => t.seatIndex || t).join(', ') }}
        </div>
        <div class="night-overlay__select-actions">
          <button
            v-if="showSkipAction"
            class="button night-overlay__skip"
            @click="skipAction"
          >{{ $t('night.skip') }}</button>
          <button
            class="button townsfolk night-overlay__confirm"
            :class="{ disabled: !canSubmit }"
            @click="submitAction"
          >{{ $t('night.confirm') }}</button>
        </div>
      </div>

      <!-- Step 3: Waiting -->
      <div v-else-if="step === 'waiting'" class="night-overlay__step waiting">
        <div class="night-overlay__spinner"></div>
        <p class="night-overlay__waiting-text">{{ $t('night.waiting') }}</p>
        <p class="night-overlay__progress" v-if="progress.total > 0">
          {{ $t('night.progress', { current: progress.current, total: progress.total }) }}
        </p>
      </div>

      <!-- Step 4: Result -->
      <div v-else-if="step === 'result'" class="night-overlay__step result">
        <h3 class="night-overlay__result-title">{{ $t('night.result') }}</h3>
        <div class="night-overlay__result-content">
          <p>{{ result }}</p>
        </div>
        <button
          class="button townsfolk night-overlay__done"
          @click="dismiss"
        >{{ $t('night.confirm') }}</button>
      </div>
    </div>
  </transition>
</template>

<script>
import { mapState, mapGetters } from "vuex";

export default {
  name: "NightOverlay",
  computed: {
    ...mapState("night", [
      "step", "roleId", "roleName", "abilityText",
      "actionType", "targets", "selectedTargets",
      "result", "progress"
    ]),
    ...mapGetters("night", ["isActive", "canSubmit"]),
    /**
     * Show overlay when step is active (not idle/done).
     * Covers: role_reveal, sleeping, woken, selecting, waiting, result.
     */
    showOverlay() {
      return this.isActive && this.step !== 'selecting';
    },
    myRoleName() {
      const r = this.$store.state.players.myRole;
      return r ? r.roleName : '';
    },
    myRoleAbility() {
      const r = this.$store.state.players.myRole;
      return r ? r.ability : '';
    },
    myRoleIcon() {
      const r = this.$store.state.players.myRole;
      if (!r || !r.roleId) return '';
      try { return require(`../assets/icons/${r.roleId}.png`); } catch (_e) { return ''; }
    },
    isAutoAction() {
      return this.actionType === 'passive' || this.actionType === 'info' || this.actionType === 'no_action';
    },
    showSkipAction() {
      const myRole = this.$store.state.players.myRole;
      const isEvilTeam = !!myRole && myRole.team === 'evil';
      return !isEvilTeam;
    },
    localizedRoleName() {
      if (!this.roleId) return this.roleName || '';
      const key = 'roles.' + this.roleId;
      return this.$te(key) ? this.$t(key) : (this.roleName || '');
    },
    roleIcon() {
      if (!this.roleId) return '';
      try {
        return require(`../assets/icons/${this.roleId}.png`);
      } catch (_e) {
        return '';
      }
    },
    // --- Team Reveal computed ---
    teamRecognition() {
      return this.$store.state.night.teamRecognition;
    },
    isDemonView() {
      return this.$store.getters['night/isDemon'];
    },
    demonSeatLabel() {
      if (!this.teamRecognition || !this.teamRecognition.demonId) return '';
      const p = this.$store.state.players.players.find(
        pl => pl.id === this.teamRecognition.demonId
      );
      return p ? this.$t('lobby.seat', { n: p.seatIndex }) : this.teamRecognition.demonId;
    },
    minionSeatLabels() {
      if (!this.teamRecognition || !this.teamRecognition.minionIds) return [];
      return this.teamRecognition.minionIds.map(id => {
        const p = this.$store.state.players.players.find(pl => pl.id === id);
        return { id, label: p ? this.$t('lobby.seat', { n: p.seatIndex }) : id };
      });
    },
    fellowMinions() {
      if (!this.teamRecognition || !this.teamRecognition.minionIds) return [];
      const myUserId = this.$store.state.players.players.find(p => p.isMe);
      const myId = myUserId ? myUserId.id : '';
      return this.teamRecognition.minionIds
        .filter(id => id !== myId)
        .map(id => {
          const p = this.$store.state.players.players.find(pl => pl.id === id);
          return { id, label: p ? this.$t('lobby.seat', { n: p.seatIndex }) : id };
        });
    },
    bluffNames() {
      if (!this.teamRecognition || !this.teamRecognition.bluffs) return [];
      return this.teamRecognition.bluffs.map(b => {
        const key = 'roles.' + b;
        return this.$te(key) ? this.$t(key) : b;
      });
    }
  },
  methods: {
    goToSelecting() {
      if (this.actionType === 'no_action') {
        this.dismiss();
      } else if (this.actionType === 'passive' || this.actionType === 'info') {
        this.submitAction();
      } else {
        this.$store.commit("night/setStep", "selecting");
      }
    },
    isSelected(target) {
      const id = target.seatIndex || target;
      return this.selectedTargets.some(t => (t.seatIndex || t) === id);
    },
    toggleTarget(target) {
      if (this.isSelected(target)) {
        this.$store.commit("night/removeTarget", target);
      } else {
        this.$store.commit("night/selectTarget", target);
      }
    },
    enterNight() {
      const myTeam = this.$store.state.players.myRole && this.$store.state.players.myRole.team;
      console.log('[NightOverlay] enterNight called, myTeam:', myTeam, 'teamRecognition:', !!this.teamRecognition);
      // 邪恶阵营（team === 'evil'）先展示团队认知，再入睡
      // 用 myRole.team 判断，不依赖 team.recognition 事件是否到达
      if (myTeam === 'evil') {
        this.$store.commit("night/setStep", "team_reveal");
        return;
      }
      this.$store.commit("night/setStep", "sleeping");
      // pendingPrompt consumption is handled by the watcher on step
    },
    submitAction() {
      this.$store.dispatch("sendNightAction", {
        targets: this.selectedTargets
      });
    },
    skipAction() {
      this.$store.dispatch("sendNightAction", { targets: [] });
    },
    dismiss() {
      this.$store.commit("night/closePanel");
    },
    dismissTeamReveal() {
      this.$store.commit("night/setStep", "sleeping");
      // 若 phase.day 在 team_reveal 期间到达，延迟的重置在此补偿
      if (this.$store._pendingNightReset) {
        this.$store._pendingNightReset = false;
        this.$store.commit("night/reset");
      }
    }
  },
  watch: {
    step(newVal, oldVal) {
      console.log('[NightOverlay] step changed:', oldVal, '->', newVal, 'pendingPrompt:', !!this.$store.state.night.pendingPrompt);
      if (newVal === 'sleeping' && this.$store.state.night.pendingPrompt) {
        setTimeout(() => {
          console.log('[NightOverlay] consuming pendingPrompt after delay');
          this.$store.commit("night/consumePendingPrompt");
        }, 1500);
      }
    }
  }
};
</script>

<style lang="scss" scoped>
@import "../vars";

.night-overlay {
  position: fixed; top: 0; left: 0; right: 0; bottom: 0;
  background: rgba(0, 0, 20, 0.92); z-index: 120;
  display: flex; align-items: center; justify-content: center; padding: 20px;

  &__step { text-align: center; max-width: 360px; width: 100%; }

  // Woken step
  &__role-icon {
    width: 100px; height: 100px; margin: 0 auto 20px; border-radius: 50%;
    background: rgba(255, 255, 255, 0.1);
    display: flex; align-items: center; justify-content: center;
    animation: glow-pulse 2s ease-in-out infinite;
    img { width: 80%; height: 80%; object-fit: contain; }
  }
  &__wake-title { font-size: 1.8rem; margin-bottom: 8px; animation: fade-in-up 0.5s ease-out; }
  &__role-name { font-size: 1.1rem; opacity: 0.8; margin-bottom: 12px; }
  &__ability { font-size: 0.85rem; opacity: 0.6; line-height: 1.4; margin-bottom: 32px; font-family: Papyrus, serif; }
  &__continue { min-width: 200px; font-size: 1rem; padding: 6px 0; }

  // Selecting step
  &__select-title { margin-bottom: 20px; font-size: 1.1rem; }
  &__targets { display: flex; flex-wrap: wrap; gap: 8px; justify-content: center; margin-bottom: 16px; }
  &__target {
    width: 64px; height: 64px; border-radius: 50%;
    border: 2px solid rgba(255, 255, 255, 0.2); background: rgba(255, 255, 255, 0.05);
    color: white; cursor: pointer; transition: all 200ms;
    display: flex; align-items: center; justify-content: center;
    &:hover { border-color: rgba(255, 255, 255, 0.4); }
    &.selected { border-color: $townsfolk; background: rgba($townsfolk, 0.2); box-shadow: 0 0 12px rgba($townsfolk, 0.3); }
    &:active { transform: scale(0.9); }
  }
  &__target-seat { font-size: 0.75rem; font-weight: bold; }
  &__selected-info { font-size: 0.75rem; opacity: 0.5; margin-bottom: 16px; }
  &__select-actions {
    display: flex; gap: 12px; justify-content: center;
    .button { min-width: 120px; padding: 4px 0; }
  }

  // Waiting step
  &__spinner {
    width: 48px; height: 48px; border: 3px solid rgba(255, 255, 255, 0.1);
    border-top-color: $townsfolk; border-radius: 50%; margin: 0 auto 20px;
    animation: spin 1s linear infinite;
  }
  &__waiting-text { font-size: 1rem; opacity: 0.6; margin-bottom: 8px; }
  &__progress { font-size: 0.8rem; opacity: 0.4; }

  // Result step
  &__result-title { margin-bottom: 16px; }
  &__result-content {
    background: rgba(255, 255, 255, 0.08); border-radius: 12px;
    padding: 20px; margin-bottom: 24px; font-family: Papyrus, serif;
    font-size: 0.95rem; line-height: 1.5;
  }
  &__done { min-width: 200px; padding: 6px 0; }

  // Sleeping step
  &__moon { font-size: 4rem; margin-bottom: 20px; animation: float 3s ease-in-out infinite; }

  // Team Reveal step
  &__evil-icon { font-size: 3.5rem; margin-bottom: 12px; animation: glow-pulse 2s ease-in-out infinite; }
  &__team-label { font-size: 1rem; opacity: 0.7; margin: 16px 0 8px; font-family: Papyrus, serif; }
  &__team-members { display: flex; flex-wrap: wrap; gap: 10px; justify-content: center; margin-bottom: 8px; }
  &__team-member {
    padding: 8px 18px; border-radius: 8px; font-weight: bold; font-size: 1rem;
    &.demon-member { background: rgba($demon, 0.25); border: 1.5px solid $demon; color: lighten($demon, 20%); }
    &.minion-member { background: rgba($minion, 0.2); border: 1.5px solid $minion; color: lighten($minion, 15%); }
  }
  &__member-seat { letter-spacing: 0.5px; }
  &__team-hint { font-size: 0.85rem; opacity: 0.55; margin-top: 4px; }
  &__bluffs-section { margin-top: 12px; }
  &__bluff-list { display: flex; flex-wrap: wrap; gap: 8px; justify-content: center; }
  &__bluff-tag {
    background: rgba($outsider, 0.15); border: 1px solid rgba($outsider, 0.4);
    border-radius: 6px; padding: 4px 12px; font-size: 0.9rem; color: lighten($outsider, 10%);
  }
  &__sleep-title { font-size: 1.6rem; margin-bottom: 12px; opacity: 0.8; animation: fade-in-up 0.5s ease-out; }
  &__sleep-hint { font-size: 0.9rem; opacity: 0.5; margin-bottom: 24px; font-family: Papyrus, serif; }
  &__sleep-dots {
    display: flex; gap: 8px; justify-content: center;
    .dot {
      width: 8px; height: 8px; border-radius: 50%; background: rgba(255, 255, 255, 0.3);
      animation: dot-pulse 1.5s ease-in-out infinite;
      &:nth-child(2) { animation-delay: 0.3s; }
      &:nth-child(3) { animation-delay: 0.6s; }
    }
  }
}

.night-fade-enter-active, .night-fade-leave-active { transition: opacity 500ms; }
.night-fade-enter, .night-fade-leave-to { opacity: 0; }

@keyframes glow-pulse {
  0%, 100% { box-shadow: 0 0 20px rgba($townsfolk, 0.2); }
  50% { box-shadow: 0 0 40px rgba($townsfolk, 0.4); }
}
@keyframes fade-in-up {
  from { opacity: 0; transform: translateY(20px); }
  to { opacity: 1; transform: translateY(0); }
}
@keyframes spin { to { transform: rotate(360deg); } }
@keyframes float {
  0%, 100% { transform: translateY(0); }
  50% { transform: translateY(-10px); }
}
@keyframes dot-pulse {
  0%, 100% { opacity: 0.3; transform: scale(1); }
  50% { opacity: 1; transform: scale(1.4); }
}
</style>
