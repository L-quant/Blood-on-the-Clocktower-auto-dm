<!-- VoteOverlay 投票界面：提名信息、进度条、投票按钮
  [OUT] App.vue（全局覆盖层）
  [POS] 投票阶段的全屏交互界面 -->
<template>
  <transition name="vote-slide">
    <div class="vote-overlay" v-if="isActive" role="region" :aria-label="$t('vote.nomination')">
      <!-- Nomination banner -->
      <div class="vote-overlay__banner">
        <div class="vote-overlay__nomination">
          <span class="vote-overlay__nominator">
            {{ $t('vote.nominatorLabel', { n: nominatorSeat }) }}
          </span>
          <span class="vote-overlay__nominee">
            {{ $t('vote.nomineeLabel', { n: nomineeSeat }) }}
          </span>
        </div>
      </div>

      <!-- Defense phase -->
      <div class="vote-overlay__defense" v-if="subPhase === 'defense'">
        <p class="vote-overlay__defense-text">{{ $t('vote.defensePhase') }}</p>
        <button
          v-if="canEndDefense"
          class="vote-overlay__end-defense-btn"
          @click="endDefense"
        >
          {{ $t('vote.endDefense') }}
        </button>
        <p v-else class="vote-overlay__defense-waiting">{{ $t('vote.defenseWaiting') }}</p>
      </div>

      <!-- Voting phase -->
      <template v-if="subPhase === 'voting' || subPhase === 'resolved'">
        <!-- Vote progress -->
        <div class="vote-overlay__progress">
          <div class="vote-overlay__progress-info">
            <span>{{ $t('vote.currentVotes', { count: currentYesCount }) }}</span>
            <span>{{ $t('vote.requiredVotes', { count: requiredMajority }) }}</span>
          </div>
          <div class="vote-overlay__progress-bar">
            <div
              class="vote-overlay__progress-fill"
              :style="{ width: (voteProgress * 100) + '%' }"
              :class="{ full: voteProgress >= 1 }"
            ></div>
          </div>
        </div>

        <!-- Vote circle indicators -->
        <div class="vote-overlay__voters" v-if="votes.length">
          <span
            v-for="v in votes"
            :key="v.seatIndex"
            class="vote-overlay__voter"
            :class="{ yes: v.vote, no: !v.vote, current: v.seatIndex === currentVoterSeat }"
          >
            {{ $t('square.seat', { n: v.seatIndex }) }}{{ v.vote ? '👍' : '👎' }}
          </span>
        </div>

        <!-- Vote buttons (visible when in voting and I haven't voted) -->
        <div class="vote-overlay__my-turn" v-if="canVote">
          <p class="vote-overlay__turn-text pulse">{{ $t('vote.yourTurn') }}</p>
          <div class="vote-overlay__vote-buttons">
            <button
              class="vote-overlay__vote-btn yes"
              :aria-label="$t('vote.voteYes')"
              @click="castVote(true)"
            >
              <span class="vote-overlay__vote-icon" aria-hidden="true">👍</span>
              <span>{{ $t('vote.voteYes') }}</span>
            </button>
            <button
              class="vote-overlay__vote-btn no"
              :aria-label="$t('vote.voteNo')"
              @click="castVote(false)"
            >
              <span class="vote-overlay__vote-icon" aria-hidden="true">👎</span>
              <span>{{ $t('vote.voteNo') }}</span>
            </button>
          </div>
        </div>

        <!-- Waiting for others -->
        <div class="vote-overlay__waiting" v-else-if="!result">
          <p>{{ $t('vote.waiting') }}</p>
        </div>

        <!-- Vote result -->
        <div class="vote-overlay__result" v-if="result">
          <p
            class="vote-overlay__result-text"
            :class="result"
          >
            {{ result === 'executed' ? $t('vote.executed') : $t('vote.safe') }}
          </p>
        </div>
      </template>
    </div>
  </transition>
</template>

<script>
import { mapState, mapGetters } from "vuex";

export default {
  name: "VoteOverlay",
  computed: {
    ...mapState("vote", [
      "isActive", "subPhase", "nominator", "nominee", "votes",
      "currentVoterIndex", "requiredMajority", "currentYesCount",
      "myVote", "isVotePending", "result"
    ]),
    ...mapGetters("vote", ["voteProgress"]),
    nominatorSeat() {
      return this.nominator ? this.nominator.seatIndex : '?';
    },
    nomineeSeat() {
      return this.nominee ? this.nominee.seatIndex : '?';
    },
    currentVoterSeat() {
      return this.currentVoterIndex;
    },
    canVote() {
      return this.subPhase === 'voting' && this.myVote === null && !this.isVotePending;
    },
    canEndDefense() {
      const mySeat = this.$store.state.seatIndex;
      return mySeat === this.nominatorSeat || mySeat === this.nomineeSeat;
    }
  },
  methods: {
    castVote(vote) {
      this.$store.commit('vote/setVotePending', true);
      this.$store.dispatch("sendVote", vote);
    },
    endDefense() {
      this.$store.dispatch("sendEndDefense");
    }
  }
};
</script>

<style lang="scss" scoped>
@import "../vars";

.vote-overlay {
  position: fixed;
  left: 0;
  right: 0;
  bottom: 56px; // above bottom nav
  background: rgba(0, 0, 0, 0.9);
  border-top: 2px solid rgba($demon, 0.5);
  padding: 16px 20px;
  z-index: 110;
  backdrop-filter: blur(10px);

  &__banner {
    text-align: center;
    margin-bottom: 12px;
  }

  &__nomination {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 4px;
  }

  &__nominator {
    font-size: 0.8rem;
    opacity: 0.6;
  }

  &__nominee {
    font-size: 1.2rem;
    font-family: PiratesBay, sans-serif;
    color: $demon;
  }

  &__progress {
    margin-bottom: 12px;
  }

  &__progress-info {
    display: flex;
    justify-content: space-between;
    font-size: 0.7rem;
    opacity: 0.5;
    margin-bottom: 4px;
  }

  &__progress-bar {
    height: 6px;
    background: rgba(255, 255, 255, 0.1);
    border-radius: 3px;
    overflow: hidden;
  }

  &__progress-fill {
    height: 100%;
    background: $townsfolk;
    border-radius: 3px;
    transition: width 300ms ease;

    &.full {
      background: $demon;
    }
  }

  &__voters {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
    justify-content: center;
    margin-bottom: 12px;
  }

  &__voter {
    font-size: 0.65rem;
    padding: 2px 6px;
    border-radius: 4px;
    background: rgba(255, 255, 255, 0.08);

    &.yes {
      color: $townsfolk;
    }
    &.no {
      color: rgba(255, 255, 255, 0.4);
    }
    &.current {
      border: 1px solid $fabled;
    }
  }

  &__defense {
    text-align: center;
    padding: 8px 0;
  }

  &__defense-text {
    font-size: 0.9rem;
    color: $fabled;
    margin-bottom: 12px;
  }

  &__end-defense-btn {
    padding: 10px 24px;
    border: 2px solid $fabled;
    border-radius: 8px;
    background: none;
    color: white;
    font-size: 0.9rem;
    cursor: pointer;
    transition: all 200ms;

    &:active {
      background: rgba($fabled, 0.2);
      transform: scale(0.95);
    }
  }

  &__defense-waiting {
    font-size: 0.85rem;
    opacity: 0.5;
  }

  &__my-turn {
    text-align: center;
  }

  &__turn-text {
    font-size: 0.9rem;
    margin-bottom: 12px;
    color: $fabled;

    &.pulse {
      animation: vote-pulse 1s ease-in-out infinite;
    }
  }

  &__vote-buttons {
    display: flex;
    gap: 16px;
    justify-content: center;
  }

  &__vote-btn {
    flex: 1;
    max-width: 140px;
    padding: 16px;
    border: 3px solid;
    border-radius: 12px;
    background: none;
    color: white;
    font-size: 1rem;
    font-weight: bold;
    cursor: pointer;
    transition: all 200ms;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 6px;

    &.yes {
      border-color: $townsfolk;
      &:active {
        background: rgba($townsfolk, 0.2);
        transform: scale(0.95);
      }
    }

    &.no {
      border-color: rgba(255, 255, 255, 0.3);
      &:active {
        background: rgba(255, 255, 255, 0.1);
        transform: scale(0.95);
      }
    }
  }

  &__vote-icon {
    font-size: 1.8rem;
  }

  &__waiting {
    text-align: center;
    font-size: 0.85rem;
    opacity: 0.5;
  }

  &__result {
    text-align: center;
  }

  &__result-text {
    font-size: 1.3rem;
    font-family: PiratesBay, sans-serif;

    &.executed {
      color: $demon;
    }
    &.safe {
      color: $townsfolk;
    }
  }
}

.vote-slide-enter-active,
.vote-slide-leave-active {
  transition: transform 300ms ease-out, opacity 200ms;
}
.vote-slide-enter,
.vote-slide-leave-to {
  opacity: 0;
  transform: translateY(100%);
}

@keyframes vote-pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}
</style>
