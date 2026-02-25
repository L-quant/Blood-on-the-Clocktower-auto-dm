// Vuex 模块：玩家个人猜测标注（按房间 localStorage 持久化）
//
// [OUT] store/index.js（模块注册）
// [POS] 个人笔记功能，记录对其他玩家角色的猜测

const state = () => ({
  playerAnnotations: {}
  // Example:
  // { 3: { guessedRoleId: 'washerwoman', guessedTeam: 'townsfolk', note: '...', updatedAt: 1708400000000 } }
});

const mutations = {
  setAnnotation(state, { seatIndex, annotation }) {
    state.playerAnnotations = {
      ...state.playerAnnotations,
      [seatIndex]: {
        ...annotation,
        updatedAt: Date.now()
      }
    };
  },
  updateNote(state, { seatIndex, note }) {
    const existing = state.playerAnnotations[seatIndex] || {};
    state.playerAnnotations = {
      ...state.playerAnnotations,
      [seatIndex]: {
        ...existing,
        note,
        updatedAt: Date.now()
      }
    };
  },
  setGuessedRole(state, { seatIndex, guessedRoleId, guessedTeam }) {
    const existing = state.playerAnnotations[seatIndex] || {};
    state.playerAnnotations = {
      ...state.playerAnnotations,
      [seatIndex]: {
        ...existing,
        guessedRoleId: guessedRoleId || existing.guessedRoleId,
        guessedTeam: guessedTeam || existing.guessedTeam,
        updatedAt: Date.now()
      }
    };
  },
  clearAnnotation(state, seatIndex) {
    const copy = { ...state.playerAnnotations };
    delete copy[seatIndex];
    state.playerAnnotations = copy;
  },
  clearAll(state) {
    state.playerAnnotations = {};
  },
  loadAnnotations(state, annotations) {
    state.playerAnnotations = annotations || {};
  }
};

const getters = {
  forSeat: state => seatIndex => state.playerAnnotations[seatIndex] || null,
  hasAnnotation: state => seatIndex => !!state.playerAnnotations[seatIndex],
  allAnnotations: state => state.playerAnnotations
};

export default {
  namespaced: true,
  state,
  mutations,
  getters
};
