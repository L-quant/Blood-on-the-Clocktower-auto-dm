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
        <p class="vote-overlay__countdown" v-if="countdown > 0">{{ countdown }}s</p>
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
            <span class="vote-overlay__countdown" v-if="countdown > 0 && subPhase === 'voting'">{{ countdown }}s</span>
          </div>
          <div class="vote-overlay__progress-bar">
            <div
              class="vote-overlay__progress-fill"
              :style="{ width: (voteProgress * 100) + '%' }"
              :class="{ full: voteProgress >= 1 }"
            ></div>
          </div>
        </div>

        <!-- Vote circle indicators (sequential order) -->
        <div class="vote-overlay__voters" v-if="voteOrder.length">
          <span
            v-for="seat in voteOrder"
            :key="seat"
            class="vote-overlay__voter"
            :class="voterClass(seat)"
          >
            {{ $t('square.seat', { n: seat }) }}
            <template v-if="hasVoted(seat)">{{ getVote(seat) ? '👍' : '👎' }}</template>
            <template v-else-if="seat === currentVoterSeatIndex">⏳</template>
            <template v-else>·</template>
          </span>
        </div>

        <!-- Vote buttons (visible when in voting and I haven't voted) -->
        <div class="vote-overlay__my-turn" v-if="canVote">
          <p class="vote-overlay__turn-text pulse">{{ $t('vote.yourTurn') }}</p>
          <div class="vote-overlay__vote-buttons">
            <button
              class="vote-overlay__vote-btn yes"
              :aria-label="voteYesLabel"
              @click="castVote(true)"
            >
              <span class="vote-overlay__vote-icon" aria-hidden="true">👍</span>
              <span>{{ voteYesLabel }}</span>
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
          <p v-if="currentVoterSeatIndex > 0">{{ $t('vote.waitingForVoter', { n: currentVoterSeatIndex }) }}</p>
          <p v-else>{{ $t('vote.waiting') }}</p>
          <p class="vote-overlay__debug-info">
            [调试] 你的座位: {{ mySeatDebug }} | 当前投票者: {{ currentVoterSeatIndex }} | 阶段: {{ subPhase }}
          </p>
        </div>

        <!-- Vote result -->
        <div class="vote-overlay__result" v-if="result">
          <p
            class="vote-overlay__result-text"
            :class="resultClass"
          >
            {{ resultLabel }}
          </p>
          <button class="vote-overlay__close-btn" @click="closeOverlay">
            {{ $t('vote.close') }}
          </button>
        </div>
      </template>
    </div>
  </transition>
</template>

<script>
import { mapState, mapGetters } from "vuex";

export default {
  name: "VoteOverlay",
  data() {
    return {
      countdown: 0,
      countdownTimer: null
    };
  },
  computed: {
    ...mapState("vote", [
      "isActive", "subPhase", "nominator", "nominee", "votes",
      "voteOrder", "currentVoterSeatIndex", "requiredMajority", "currentYesCount",
      "myVote", "isVotePending", "result"
    ]),
    ...mapState("game", ["phaseDeadline"]),
    ...mapGetters("vote", ["voteProgress"]),
    nominatorSeat() {
      return this.nominator ? this.nominator.seatIndex : '?';
    },
    nomineeSeat() {
      return this.nominee ? this.nominee.seatIndex : '?';
    },
    currentVoterSeat() {
      return this.currentVoterSeatIndex;
    },
    mySeatDebug() {
      return this.$store.state.seatIndex;
    },
    mePlayer() {
      return this.$store.state.players.players.find(player => player.isMe) || null;
    },
    isDeadVoter() {
      return !!this.mePlayer && !this.mePlayer.isAlive;
    },
    voteYesLabel() {
      return this.isDeadVoter ? this.$t('vote.voteGhost') : this.$t('vote.voteYes');
    },
    canVote() {
      const mySeat = Number(this.$store.state.seatIndex);
      const currentVoter = Number(this.currentVoterSeatIndex);
      const result = this.subPhase === 'voting' && this.myVote === null && !this.isVotePending && mySeat === currentVoter;
      console.log('[DBG] canVote:', result, '| subPhase:', this.subPhase, '| myVote:', this.myVote, '| isVotePending:', this.isVotePending, '| mySeat:', mySeat, '(type:', typeof mySeat, ') | currentVoter:', currentVoter, '(type:', typeof currentVoter, ')');
      return result;
    },
    canEndDefense() {
      const mySeat = this.$store.state.seatIndex;
      return mySeat === this.nominatorSeat || mySeat === this.nomineeSeat;
    },
    resultClass() {
      if (this.result === 'on_the_block' || this.result === 'executed') return 'executed';
      if (this.result === 'tied') return 'tied';
      return 'safe';
    },
    resultLabel() {
      switch (this.result) {
        case 'on_the_block': return this.$t('vote.onTheBlock');
        case 'tied': return this.$t('vote.tied');
        case 'executed': return this.$t('vote.executed');
        default: return this.$t('vote.safe');
      }
    }
  },
  watch: {
    phaseDeadline(deadline) {
      this.startCountdown(deadline);
    },
    subPhase(val) {
      if (val === 'resolved') {
        this.stopCountdown();
      }
    },
    isActive(val) {
      if (!val) {
        this.stopCountdown();
      }
    }
  },
  beforeDestroy() {
    this.stopCountdown();
  },
  methods: {
    castVote(vote) {
      this.$store.commit('vote/setVotePending', true);
      this.$store.dispatch("sendVote", vote);
    },
    endDefense() {
      this.$store.dispatch("sendEndDefense");
    },
    closeOverlay() {
      this.$store.commit('vote/endVote');
    },
    voterClass(seat) {
      const v = this.votes.find(v => v.seatIndex === seat);
      if (v) return v.vote ? 'yes' : 'no';
      if (seat === this.currentVoterSeatIndex) return 'current';
      return 'pending';
    },
    hasVoted(seat) {
      return this.votes.some(v => v.seatIndex === seat);
    },
    getVote(seat) {
      const v = this.votes.find(v => v.seatIndex === seat);
      return v ? v.vote : false;
    },
    startCountdown(deadline) {
      this.stopCountdown();
      if (!deadline || deadline <= 0) return;
      const tick = () => {
        const remaining = Math.max(0, Math.ceil((deadline - Date.now()) / 1000));
        this.countdown = remaining;
        if (remaining <= 0) {
          this.stopCountdown();
        }
      };
      tick();
      this.countdownTimer = setInterval(tick, 1000);
    },
    stopCountdown() {
      if (this.countdownTimer) {
        clearInterval(this.countdownTimer);
        this.countdownTimer = null;
      }
      this.countdown = 0;
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
      animation: vote-pulse 1s ease-in-out infinite;
    }
    &.pending {
      opacity: 0.4;
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

  &__debug-info {
    font-size: 0.65rem;
    opacity: 0.4;
    margin-top: 8px;
    font-family: monospace;
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
    &.tied {
      color: $fabled;
    }
  }

  &__close-btn {
    margin-top: 8px;
    padding: 6px 20px;
    border: 1px solid rgba(255, 255, 255, 0.3);
    border-radius: 6px;
    background: none;
    color: white;
    font-size: 0.8rem;
    cursor: pointer;
    transition: all 200ms;

    &:active {
      background: rgba(255, 255, 255, 0.1);
      transform: scale(0.95);
    }
  }

  &__countdown {
    font-size: 1.1rem;
    font-family: PiratesBay, sans-serif;
    color: $fabled;
    font-weight: bold;
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
