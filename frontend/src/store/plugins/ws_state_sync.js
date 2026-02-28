// WebSocket 状态同步：REST 拉取完整房间状态后同步到 Vuex
//
// [IN]  websocket.js（WebSocketManager 调用）
// [OUT] store modules（通过 store.commit 更新状态）
// [POS] 初次连接或重连后同步完整游戏状态
import apiService from "../../services/ApiService";

/**
 * Sync full room state from REST API response into Vuex store.
 * @param {object} state - Backend room state object
 * @param {object} store - Vuex store instance
 */
export function syncRoomState(state, store) {
  if (!state) return;

  if (state.players) {
    syncPlayers(state.players, store);
  }

  if (state.phase) {
    store.commit('game/setPhase', state.phase);
    if (state.phase === 'lobby') {
      store.commit('ui/setScreen', 'lobby');
    } else if (state.phase === 'ended') {
      store.commit('ui/setScreen', 'end');
      if (state.winner) store.commit('game/setWinner', state.winner);
    } else {
      store.commit('ui/setScreen', 'game');
    }
  }

  if (state.day_count !== undefined) {
    store.commit('game/setDayCount', state.day_count);
  }
  if (state.edition) {
    store.commit('setEdition', state.edition);
  }
  if (state.max_players) {
    store.commit('setSeatCount', state.max_players);
  }

  syncOwnRole(state.players, store);
}

function syncPlayers(playersMap, store) {
  const playersList = [];
  const entries = typeof playersMap === 'object' && !Array.isArray(playersMap)
    ? Object.entries(playersMap) : [];

  let mySeatIndex = -1;
  entries.forEach(([userId, p]) => {
    const seatIndex = p.seat_number || 0;
    const isMe = userId === apiService.userId;
    playersList.push({
      id: userId,
      name: p.name || '',
      seatIndex,
      isAlive: p.alive !== undefined ? p.alive : true,
      hasGhostVote: p.has_ghost_vote !== undefined ? p.has_ghost_vote : true,
      isNominatedToday: p.was_nominated || false,
      hasNominatedToday: p.has_nominated || false,
      isMe
    });
    if (isMe && seatIndex > 0) mySeatIndex = seatIndex;
  });

  if (playersList.length > 0) store.commit('players/setPlayers', playersList);
  if (mySeatIndex > 0) store.commit('setSeatIndex', mySeatIndex);
}

function syncOwnRole(playersMap, store) {
  if (!playersMap) return;
  const meData = playersMap[apiService.userId];
  if (meData && meData.role && !store.state.players.myRole) {
    const roleData = store.getters.rolesByKey.get(meData.role);
    store.commit('players/setMyRole', {
      roleId: meData.role,
      roleName: roleData ? roleData.name : meData.role,
      team: meData.team || '',
      ability: roleData ? roleData.ability : ''
    });
  }
}
