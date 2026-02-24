// Annotations module - purely local, persisted to localStorage
// Keyed by seatIndex, stores player's personal guesses about other players' roles

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
