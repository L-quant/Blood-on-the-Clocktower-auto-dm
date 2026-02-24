<template>
  <transition name="night-fade">
    <div class="night-overlay" v-if="isActive" role="dialog" :aria-label="$t('night.title')" aria-modal="true">
      <!-- Step 1: Woken up -->
      <div v-if="step === 'woken'" class="night-overlay__step woken">
        <div class="night-overlay__role-icon">
          <img
            v-if="roleIcon"
            :src="roleIcon"
            :alt="roleName"
          />
        </div>
        <h2 class="night-overlay__wake-title">{{ $t('night.wakeUp') }}</h2>
        <p class="night-overlay__role-name">{{ localizedRoleName }}</p>
        <p class="night-overlay__ability">{{ abilityText }}</p>
        <button
          class="button townsfolk night-overlay__continue"
          @click="goToSelecting"
        >
          {{ actionType === 'passive' || actionType === 'info' ? $t('night.confirm') : $t('night.selectTarget') }}
        </button>
      </div>

      <!-- Step 2: Selecting targets -->
      <div v-if="step === 'selecting'" class="night-overlay__step selecting">
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
      <div v-if="step === 'waiting'" class="night-overlay__step waiting">
        <div class="night-overlay__spinner"></div>
        <p class="night-overlay__waiting-text">{{ $t('night.waiting') }}</p>
        <p class="night-overlay__progress" v-if="progress.total > 0">
          {{ $t('night.progress', { current: progress.current, total: progress.total }) }}
        </p>
      </div>

      <!-- Step 4: Result -->
      <div v-if="step === 'result'" class="night-overlay__step result">
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
    localizedRoleName() {
      if (!this.roleId) return this.roleName || '';
      const key = 'roles.' + this.roleId;
      return this.$te(key) ? this.$t(key) : (this.roleName || '');
    },
    roleIcon() {
      if (!this.roleId) return '';
      try {
        return require(`../assets/icons/${this.roleId}.png`);
      } catch (e) {
        return '';
      }
    }
  },
  methods: {
    goToSelecting() {
      if (this.actionType === 'passive' || this.actionType === 'info') {
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
    }
  }
};
</script>

<style lang="scss" scoped>
@import "../vars";

.night-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 20, 0.92);
  z-index: 120;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;

  &__step {
    text-align: center;
    max-width: 360px;
    width: 100%;
  }

  // Woken step
  &__role-icon {
    width: 100px;
    height: 100px;
    margin: 0 auto 20px;
    border-radius: 50%;
    background: rgba(255, 255, 255, 0.1);
    display: flex;
    align-items: center;
    justify-content: center;
    animation: glow-pulse 2s ease-in-out infinite;

    img {
      width: 80%;
      height: 80%;
      object-fit: contain;
    }
  }

  &__wake-title {
    font-size: 1.8rem;
    margin-bottom: 8px;
    animation: fade-in-up 0.5s ease-out;
  }

  &__role-name {
    font-size: 1.1rem;
    opacity: 0.8;
    margin-bottom: 12px;
  }

  &__ability {
    font-size: 0.85rem;
    opacity: 0.6;
    line-height: 1.4;
    margin-bottom: 32px;
    font-family: Papyrus, serif;
  }

  &__continue {
    min-width: 200px;
    font-size: 1rem;
    padding: 6px 0;
  }

  // Selecting step
  &__select-title {
    margin-bottom: 20px;
    font-size: 1.1rem;
  }

  &__targets {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
    justify-content: center;
    margin-bottom: 16px;
  }

  &__target {
    width: 64px;
    height: 64px;
    border-radius: 50%;
    border: 2px solid rgba(255, 255, 255, 0.2);
    background: rgba(255, 255, 255, 0.05);
    color: white;
    cursor: pointer;
    transition: all 200ms;
    display: flex;
    align-items: center;
    justify-content: center;

    &:hover {
      border-color: rgba(255, 255, 255, 0.4);
    }

    &.selected {
      border-color: $townsfolk;
      background: rgba($townsfolk, 0.2);
      box-shadow: 0 0 12px rgba($townsfolk, 0.3);
    }

    &:active {
      transform: scale(0.9);
    }
  }

  &__target-seat {
    font-size: 0.75rem;
    font-weight: bold;
  }

  &__selected-info {
    font-size: 0.75rem;
    opacity: 0.5;
    margin-bottom: 16px;
  }

  &__select-actions {
    display: flex;
    gap: 12px;
    justify-content: center;

    .button {
      min-width: 120px;
      padding: 4px 0;
    }
  }

  // Waiting step
  &__spinner {
    width: 48px;
    height: 48px;
    border: 3px solid rgba(255, 255, 255, 0.1);
    border-top-color: $townsfolk;
    border-radius: 50%;
    margin: 0 auto 20px;
    animation: spin 1s linear infinite;
  }

  &__waiting-text {
    font-size: 1rem;
    opacity: 0.6;
    margin-bottom: 8px;
  }

  &__progress {
    font-size: 0.8rem;
    opacity: 0.4;
  }

  // Result step
  &__result-title {
    margin-bottom: 16px;
  }

  &__result-content {
    background: rgba(255, 255, 255, 0.08);
    border-radius: 12px;
    padding: 20px;
    margin-bottom: 24px;
    font-family: Papyrus, serif;
    font-size: 0.95rem;
    line-height: 1.5;
  }

  &__done {
    min-width: 200px;
    padding: 6px 0;
  }
}

.night-fade-enter-active,
.night-fade-leave-active {
  transition: opacity 500ms;
}
.night-fade-enter,
.night-fade-leave-to {
  opacity: 0;
}

@keyframes glow-pulse {
  0%, 100% { box-shadow: 0 0 20px rgba($townsfolk, 0.2); }
  50% { box-shadow: 0 0 40px rgba($townsfolk, 0.4); }
}

@keyframes fade-in-up {
  from {
    opacity: 0;
    transform: translateY(20px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@keyframes spin {
  to { transform: rotate(360deg); }
}
</style>
