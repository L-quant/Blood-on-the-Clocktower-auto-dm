// Vuex 模块：玩家列表、座位、角色、鬼牌、传奇角色管理
//
// [OUT] store/index.js（模块注册）
// [POS] 玩家数据中心，管理所有玩家的状态与角色信息

const state = () => ({
  players: [],
  myRole: null, // { roleId, roleName, team, ability, isPoisoned, reminders }
  bluffs: [null, null, null], // demon bluffs (only visible to demon)
  fabled: []
});

const sortPlayersBySeat = (players) => [...players].sort((a, b) => a.seatIndex - b.seatIndex);

const createPlayer = (id, seatIndex, name) => ({
  id: id || '',
  name: name || '',
  seatIndex: seatIndex,
  isAlive: true,
  hasGhostVote: true,
  isNominatedToday: false,
  hasNominatedToday: false,
  isPoisoned: false,
  reminders: [],
  isMe: false
});

const mutations = {
  setPlayers(state, players) {
    state.players = sortPlayersBySeat(players);
  },
  addPlayer(state, { id, seatIndex }) {
    state.players.push(createPlayer(id, seatIndex));
    state.players = sortPlayersBySeat(state.players);
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
    state.players = sortPlayersBySeat(state.players);
  },
  unseatPlayer(state, seatIndex) {
    const idx = state.players.findIndex(p => p.seatIndex === seatIndex);
    if (idx >= 0) {
      state.players.splice(idx, 1);
    }
  },
  setMyRole(state, role) {
    const reminders = Array.isArray(role && role.reminders) ? [...role.reminders] : [];
    state.myRole = {
      isPoisoned: false,
      ...role,
      reminders
    };
  },
  updateMyRole(state, patch) {
    if (!state.myRole) {
      state.myRole = {
        roleId: '',
        roleName: '',
        team: '',
        ability: '',
        isPoisoned: false,
        reminders: []
      };
    }
    state.myRole = {
      ...state.myRole,
      ...patch
    };
    if (Array.isArray(state.myRole.reminders)) {
      state.myRole.reminders = [...state.myRole.reminders];
    }
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
