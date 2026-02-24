import Vue from "vue";
import Vuex from "vuex";

// Modules
import game from "./modules/game";
import players from "./modules/players";
import annotations from "./modules/annotations";
import chat from "./modules/chat";
import timeline from "./modules/timeline";
import night from "./modules/night";
import vote from "./modules/vote";
import ui from "./modules/ui";

// Plugins
import persistence from "./plugins/persistence";
import websocket from "./plugins/websocket";

// Services
import apiService from "../services/ApiService";

// Game data
import editionJSON from "../editions.json";
import rolesJSON from "../roles.json";
import fabledJSON from "../fabled.json";

Vue.use(Vuex);

// Build lookup maps
const editionsByKey = new Map(editionJSON.map(e => [e.id, e]));
const rolesByKey = new Map(rolesJSON.map(r => [r.id, r]));
const fabledByKey = new Map(fabledJSON.map(f => [f.id, f]));

const getRolesByEdition = (edition) => {
  if (!edition) return new Map();
  return new Map(
    rolesJSON
      .filter(r => r.edition === edition.id || (edition.roles && edition.roles.includes(r.id)))
      .sort((a, b) => b.team.localeCompare(a.team))
      .map(r => [r.id, r])
  );
};

export default new Vuex.Store({
  modules: {
    game,
    players,
    annotations,
    chat,
    timeline,
    night,
    vote,
    ui
  },

  state: {
    roomId: '',
    role: 'spectator', // 'player' | 'spectator'
    isRoomOwner: false,
    playerId: apiService.userId || '',
    seatIndex: -1, // -1 = not seated (backend uses 1-indexed seats)
    edition: editionsByKey.get('tb') || null,
    roles: getRolesByEdition(editionsByKey.get('tb')),
    fabled: fabledByKey,
    connected: false,
    reconnecting: false,
    latencyMs: 0,
    seatCount: 7 // configurable seat count for lobby (backend default is also 7)
  },

  getters: {
    isSeated: state => state.seatIndex >= 1,
    isPlayer: state => state.role === 'player',
    isSpectator: state => state.role === 'spectator',
    rolesByKey: () => rolesByKey,
    editionsByKey: () => editionsByKey,
    editionList: () => editionJSON,
    currentRoles: state => state.roles,
    seatLabel: () => (seatIndex) => `${seatIndex}号`
  },

  mutations: {
    setRoomId(state, roomId) {
      state.roomId = roomId;
    },
    setRole(state, role) {
      state.role = role;
    },
    setIsRoomOwner(state, isOwner) {
      state.isRoomOwner = isOwner;
    },
    setPlayerId(state, id) {
      state.playerId = id;
    },
    setSeatIndex(state, index) {
      state.seatIndex = index;
      state.role = index >= 1 ? 'player' : 'spectator';
    },
    setEdition(state, edition) {
      if (typeof edition === 'string') {
        state.edition = editionsByKey.get(edition) || state.edition;
      } else if (edition && edition.id) {
        state.edition = editionsByKey.get(edition.id) || edition;
      }
      state.roles = getRolesByEdition(state.edition);
    },
    setSeatCount(state, count) {
      state.seatCount = count;
    },
    setConnected(state, connected) {
      state.connected = connected;
    },
    setReconnecting(state, reconnecting) {
      state.reconnecting = reconnecting;
    },
    setLatency(state, ms) {
      state.latencyMs = ms;
    },
    resetRoom(state) {
      state.roomId = '';
      state.role = 'spectator';
      state.isRoomOwner = false;
      state.seatIndex = -1;
      state.connected = false;
      state.reconnecting = false;
      state.latencyMs = 0;
    },
    /**
     * Proxy mutation — no-op on state, exists solely so the WebSocket
     * plugin's store.subscribe() callback fires and sends the command.
     * Vuex 3 silently drops commits for undefined mutations AND skips
     * subscriber notifications, so this stub MUST exist.
     */
    // eslint-disable-next-line no-unused-vars
    sendCommand(state, payload) { /* handled by websocket plugin subscriber */ }
  },

  actions: {
    /**
     * Ensure we have a valid JWT token before any API calls.
     * Uses quick login (no username/password needed).
     */
    async ensureAuth({ commit }) {
      await apiService.ensureAuth();
      commit('setPlayerId', apiService.userId);
    },

    /**
     * Create a new game room via REST API (with JWT auth).
     */
    async createRoom({ commit, dispatch }) {
      await dispatch('ensureAuth');
      const data = await apiService.createRoom();
      commit('setRoomId', data.room_id);
      commit('setIsRoomOwner', true);
      commit('ui/setScreen', 'lobby');
      return data;
    },

    /**
     * Join an existing room via REST API (with JWT auth).
     */
    async joinRoom({ commit, dispatch }, roomId) {
      await dispatch('ensureAuth');
      await apiService.joinRoom(roomId);
      commit('setRoomId', roomId);
      commit('setIsRoomOwner', false);
      commit('ui/setScreen', 'lobby');
    },

    /**
     * Claim a seat in the lobby.
     * Backend command: "claim_seat" with { seat_number: "N" }
     */
    seatDown({ commit }, seatIndex) {
      commit('setSeatIndex', seatIndex);
      commit('players/markMe', seatIndex);
      commit('sendCommand', {
        type: 'claim_seat',
        data: { seat_number: String(seatIndex) }
      });
    },

    /**
     * Leave current seat.
     * Backend command: "leave"
     */
    leaveSeat({ commit }) {
      commit('setSeatIndex', -1);
      commit('sendCommand', {
        type: 'leave',
        data: {}
      });
    },

    /**
     * Leave the room entirely.
     * Backend command: "leave"
     */
    leaveRoom({ commit, dispatch }) {
      commit('sendCommand', { type: 'leave', data: {} });
      dispatch('resetAll');
    },

    /**
     * Start the game (room owner only).
     * Backend command: "start_game" with { edition }
     */
    startGame({ commit, state }) {
      if (!state.isRoomOwner) return;
      commit('sendCommand', {
        type: 'start_game',
        data: {
          edition: state.edition ? state.edition.id : 'tb'
        }
      });
    },

    /**
     * Update room settings (edition, max_players).
     * Backend command: "room_settings"
     */
    updateRoomSettings({ commit }, settings) {
      commit('sendCommand', {
        type: 'room_settings',
        data: settings
      });
    },

    /**
     * Send a chat message.
     * Backend commands: "public_chat", "whisper", "evil_team_chat"
     */
    sendChat({ commit, state, rootState }, { text, channel }) {
      if (!text) return;
      const msg = {
        seatIndex: state.seatIndex,
        text,
        isMe: true,
        timestamp: Date.now()
      };

      if (channel === 'whisper') {
        const target = rootState.chat.activeWhisperTarget;
        commit('chat/addWhisperMessage', { targetSeat: target, data: msg });
        // Backend whisper requires to_user_id — find the user ID for the target seat
        const targetPlayer = rootState.players.players.find(p => p.seatIndex === target);
        if (targetPlayer) {
          commit('sendCommand', {
            type: 'whisper',
            data: { to_user_id: targetPlayer.id, message: text }
          });
        }
      } else if (channel === 'evil') {
        commit('chat/addEvilMessage', msg);
        commit('sendCommand', {
          type: 'evil_team_chat',
          data: { message: text }
        });
      } else {
        commit('chat/addPublicMessage', msg);
        commit('sendCommand', {
          type: 'public_chat',
          data: { message: text }
        });
      }
    },

    /**
     * Send night action (ability use).
     * Backend command: "ability.use" with { targets }
     */
    sendNightAction({ commit }, { targets }) {
      commit('night/setStep', 'waiting');
      // Extract user IDs and JSON-stringify for backend map[string]string format
      const targetIds = (targets || []).map(t => t.id || String(t));
      commit('sendCommand', {
        type: 'ability.use',
        data: { targets: JSON.stringify(targetIds) }
      });
    },

    /**
     * Cast a vote on a nomination.
     * Backend command: "vote" with { vote: "yes"|"no" }
     */
    sendVote({ commit }, vote) {
      commit('vote/setMyVote', vote);
      commit('vote/setIsMyTurn', false);
      commit('sendCommand', {
        type: 'vote',
        data: { vote: vote ? 'yes' : 'no' }
      });
    },

    /**
     * Nominate a player.
     * Backend command: "nominate" with { nominee: user_id }
     */
    nominate({ commit, rootState }, nomineeSeat) {
      // Find the user_id for the nominee seat
      const nomineePlayer = rootState.players.players.find(p => p.seatIndex === nomineeSeat);
      if (!nomineePlayer) return;
      commit('sendCommand', {
        type: 'nominate',
        data: { nominee: nomineePlayer.id }
      });
    },

    resetAll({ commit }) {
      commit('resetRoom');
      commit('game/reset');
      commit('players/reset');
      commit('annotations/clearAll');
      commit('chat/reset');
      commit('timeline/clear');
      commit('night/reset');
      commit('vote/reset');
      commit('ui/reset');
      commit('ui/setScreen', 'home');
    }
  },

  plugins: [persistence, websocket]
});
