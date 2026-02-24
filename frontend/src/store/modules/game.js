const state = () => ({
  phase: 'lobby', // 'lobby' | 'first_night' | 'night' | 'day' | 'nomination' | 'voting' | 'ended'
  dayCount: 0,
  winner: '', // 'good' | 'evil' | ''
  winReason: '',
  alivePlayers: 0,
  deadPlayers: 0
});

const mutations = {
  setPhase(state, phase) {
    state.phase = phase;
  },
  setDayCount(state, count) {
    state.dayCount = count;
  },
  setWinner(state, winner) {
    state.winner = winner;
  },
  setWinReason(state, reason) {
    state.winReason = reason;
  },
  setAliveCount(state, count) {
    state.alivePlayers = count;
  },
  setDeadCount(state, count) {
    state.deadPlayers = count;
  },
  updateCounts(state, { alive, dead }) {
    state.alivePlayers = alive;
    state.deadPlayers = dead;
  },
  reset(state) {
    state.phase = 'lobby';
    state.dayCount = 0;
    state.winner = '';
    state.winReason = '';
    state.alivePlayers = 0;
    state.deadPlayers = 0;
  }
};

const getters = {
  isNight: state => state.phase === 'night' || state.phase === 'first_night',
  isDay: state => state.phase === 'day',
  isLobby: state => state.phase === 'lobby',
  isEnded: state => state.phase === 'ended',
  isPlaying: state => !['lobby', 'ended'].includes(state.phase),
  phaseLabel: state => {
    const labels = {
      lobby: 'game.phases.lobby',
      first_night: 'game.phases.first_night',
      night: 'game.phases.night',
      day: 'game.phases.day',
      nomination: 'game.phases.nomination',
      voting: 'game.phases.voting',
      ended: 'game.phases.ended'
    };
    return labels[state.phase] || '';
  }
};

export default {
  namespaced: true,
  state,
  mutations,
  getters
};
