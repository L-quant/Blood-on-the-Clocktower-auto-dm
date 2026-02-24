const state = () => ({
  players: [],
  myRole: null, // { roleId, roleName, team, ability }
  bluffs: [null, null, null], // demon bluffs (only visible to demon)
  fabled: []
});

const createPlayer = (id, seatIndex, name) => ({
  id: id || '',
  name: name || '',
  seatIndex: seatIndex,
  isAlive: true,
  hasGhostVote: true,
  isNominatedToday: false,
  hasNominatedToday: false,
  isMe: false
});

const mutations = {
  setPlayers(state, players) {
    state.players = players;
  },
  addPlayer(state, { id, seatIndex }) {
    state.players.push(createPlayer(id, seatIndex));
  },
  removePlayer(state, seatIndex) {
    const idx = state.players.findIndex(p => p.seatIndex === seatIndex);
    if (idx >= 0) {
      state.players.splice(idx, 1);
    }
  },
  updatePlayer(state, { seatIndex, property, value }) {
    const player = state.players.find(p => p.seatIndex === seatIndex);
    if (player && property in player) {
      player[property] = value;
    }
  },
  seatPlayer(state, { id, seatIndex }) {
    const existing = state.players.find(p => p.seatIndex === seatIndex);
    if (existing) {
      existing.id = id;
    } else {
      state.players.push(createPlayer(id, seatIndex));
    }
  },
  unseatPlayer(state, seatIndex) {
    const idx = state.players.findIndex(p => p.seatIndex === seatIndex);
    if (idx >= 0) {
      state.players.splice(idx, 1);
    }
  },
  setMyRole(state, role) {
    state.myRole = role; // { roleId, roleName, team, ability }
  },
  setBluffs(state, bluffs) {
    state.bluffs = bluffs;
  },
  setFabled(state, fabled) {
    state.fabled = fabled;
  },
  markMe(state, seatIndex) {
    state.players.forEach(p => {
      p.isMe = p.seatIndex === seatIndex;
    });
  },
  killPlayer(state, seatIndex) {
    const player = state.players.find(p => p.seatIndex === seatIndex);
    if (player) {
      player.isAlive = false;
    }
  },
  resetNominationFlags(state) {
    state.players.forEach(p => {
      p.isNominatedToday = false;
      p.hasNominatedToday = false;
    });
  },
  reset(state) {
    state.players = [];
    state.myRole = null;
    state.bluffs = [null, null, null];
    state.fabled = [];
  }
};

const getters = {
  me: state => state.players.find(p => p.isMe),
  alive: state => state.players.filter(p => p.isAlive),
  dead: state => state.players.filter(p => !p.isAlive),
  bySeat: state => seatIndex => state.players.find(p => p.seatIndex === seatIndex),
  playerCount: state => state.players.length,
  aliveCount: state => state.players.filter(p => p.isAlive).length,
  sorted: state => [...state.players].sort((a, b) => a.seatIndex - b.seatIndex)
};

export default {
  namespaced: true,
  state,
  mutations,
  getters
};
