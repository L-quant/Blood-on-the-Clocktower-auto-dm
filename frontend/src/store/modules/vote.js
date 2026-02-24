const state = () => ({
  isActive: false,
  nominator: null, // { seatIndex }
  nominee: null, // { seatIndex }
  votes: [], // [{ seatIndex, vote: true/false }]
  currentVoterIndex: -1,
  requiredMajority: 0,
  currentYesCount: 0,
  myVote: null, // true | false | null
  isMyTurn: false,
  countdown: 0, // seconds, 0 = no timer
  result: null, // 'executed' | 'safe' | null
  history: [] // past vote records
});

const mutations = {
  startNomination(state, { nominatorSeat, nomineeSeat, requiredMajority }) {
    state.isActive = true;
    state.nominator = { seatIndex: nominatorSeat };
    state.nominee = { seatIndex: nomineeSeat };
    state.votes = [];
    state.currentVoterIndex = -1;
    state.requiredMajority = requiredMajority || 0;
    state.currentYesCount = 0;
    state.myVote = null;
    state.isMyTurn = false;
    state.result = null;
  },
  castVote(state, { seatIndex, vote }) {
    const existing = state.votes.findIndex(v => v.seatIndex === seatIndex);
    if (existing >= 0) {
      state.votes[existing].vote = vote;
    } else {
      state.votes.push({ seatIndex, vote });
    }
    if (vote) {
      state.currentYesCount = state.votes.filter(v => v.vote).length;
    }
  },
  setMyVote(state, vote) {
    state.myVote = vote;
  },
  setCurrentVoter(state, index) {
    state.currentVoterIndex = index;
  },
  setIsMyTurn(state, isMyTurn) {
    state.isMyTurn = isMyTurn;
  },
  setCountdown(state, seconds) {
    state.countdown = seconds;
  },
  setResult(state, result) {
    state.result = result;
    // Add to history
    state.history.push({
      nominatorSeat: state.nominator ? state.nominator.seatIndex : -1,
      nomineeSeat: state.nominee ? state.nominee.seatIndex : -1,
      votes: [...state.votes],
      yesCount: state.currentYesCount,
      requiredMajority: state.requiredMajority,
      result: result,
      timestamp: Date.now()
    });
  },
  endVote(state) {
    state.isActive = false;
    state.nominator = null;
    state.nominee = null;
    state.votes = [];
    state.currentVoterIndex = -1;
    state.currentYesCount = 0;
    state.myVote = null;
    state.isMyTurn = false;
    state.countdown = 0;
    state.result = null;
  },
  clearHistory(state) {
    state.history = [];
  },
  reset(state) {
    state.isActive = false;
    state.nominator = null;
    state.nominee = null;
    state.votes = [];
    state.currentVoterIndex = -1;
    state.requiredMajority = 0;
    state.currentYesCount = 0;
    state.myVote = null;
    state.isMyTurn = false;
    state.countdown = 0;
    state.result = null;
    state.history = [];
  }
};

const getters = {
  voteProgress: state => {
    if (!state.requiredMajority) return 0;
    return Math.min(1, state.currentYesCount / state.requiredMajority);
  },
  isNominated: state => seatIndex => {
    return state.nominee && state.nominee.seatIndex === seatIndex;
  }
};

export default {
  namespaced: true,
  state,
  mutations,
  getters
};
