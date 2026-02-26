// Vuex 模块：提名与投票状态（提名者/被提名者/票数/结果/历史）
//
// [OUT] store/index.js（模块注册）
// [POS] 投票流程状态，驱动 VoteOverlay 交互

const state = () => ({
  isActive: false,
  subPhase: 'none', // 'none' | 'defense' | 'voting' | 'resolved'
  nominator: null, // { seatIndex }
  nominee: null, // { seatIndex }
  votes: [], // [{ seatIndex, vote: true/false }]
  currentVoterIndex: -1,
  requiredMajority: 0,
  currentYesCount: 0,
  myVote: null, // true | false | null
  isVotePending: false, // true while vote command is in-flight
  result: null, // 'executed' | 'safe' | null
  history: [] // past vote records
});

const mutations = {
  startNomination(state, { nominatorSeat, nomineeSeat, requiredMajority }) {
    state.isActive = true;
    state.subPhase = 'defense';
    state.nominator = { seatIndex: nominatorSeat };
    state.nominee = { seatIndex: nomineeSeat };
    state.votes = [];
    state.currentVoterIndex = -1;
    state.requiredMajority = requiredMajority || 0;
    state.currentYesCount = 0;
    state.myVote = null;
    state.isVotePending = false;
    state.result = null;
  },
  setSubPhase(state, subPhase) {
    state.subPhase = subPhase;
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
  setVotePending(state, val) {
    state.isVotePending = val;
  },
  setCurrentVoter(state, index) {
    state.currentVoterIndex = index;
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
    state.subPhase = 'none';
    state.nominator = null;
    state.nominee = null;
    state.votes = [];
    state.currentVoterIndex = -1;
    state.currentYesCount = 0;
    state.myVote = null;
    state.isVotePending = false;
    state.result = null;
  },
  reset(state) {
    state.isActive = false;
    state.subPhase = 'none';
    state.nominator = null;
    state.nominee = null;
    state.votes = [];
    state.currentVoterIndex = -1;
    state.requiredMajority = 0;
    state.currentYesCount = 0;
    state.myVote = null;
    state.isVotePending = false;
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
