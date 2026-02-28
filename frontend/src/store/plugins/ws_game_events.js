// WebSocket 游戏事件处理：将后端 ProjectedEvent 映射到 Vuex mutations
//
// [IN]  websocket.js（WebSocketManager 调用）
// [OUT] store modules（通过 store.commit 更新状态）
// [POS] 处理 25+ 种后端事件类型
import apiService from "../../services/ApiService";
import i18n from "../../i18n";

/**
 * Process a ProjectedEvent from the backend.
 * @param {object} pe - ProjectedEvent { event_type, data, seq, server_ts, actor_user_id }
 * @param {object} store - Vuex store instance
 */
export function processGameEvent(pe, store) {
  if (!pe || !pe.event_type) return;
  const eventType = pe.event_type;
  let eventData = pe.data;
  if (typeof eventData === 'string') {
    try { eventData = JSON.parse(eventData); } catch (_e) { eventData = {}; }
  }
  if (!eventData) eventData = {};

  switch (eventType) {
    case 'player.joined':
      handlePlayerJoined(pe, eventData, store);
      break;
    case 'player.left':
      handlePlayerLeft(pe, eventData, store);
      break;
    case 'seat.claimed':
      handleSeatClaimed(pe, eventData, store);
      break;
    case 'role.assigned':
      handleRoleAssigned(eventData, store);
      break;
    case 'bluffs.assigned':
      handleBluffsAssigned(eventData, store);
      break;
    case 'room.settings.changed':
      if (eventData.edition) store.commit('setEdition', eventData.edition);
      if (eventData.max_players) store.commit('setSeatCount', parseInt(eventData.max_players, 10) || 8);
      break;
    case 'game.started':
      store.commit('game/setPhase', 'first_night');
      store.commit('ui/setScreen', 'game');
      break;
    case 'phase.first_night':
      store.commit('game/setPhase', 'first_night');
      store.commit('game/setDayCount', 0);
      addPhaseTimeline('first_night', store);
      break;
    case 'phase.night':
      store.commit('game/setPhase', 'night');
      store.commit('players/resetNominationFlags');
      addPhaseTimeline('night', store);
      break;
    case 'phase.day':
      handlePhaseDay(store);
      break;
    case 'phase.nomination':
      store.commit('game/setPhase', 'nomination');
      addPhaseTimeline('nomination', store);
      break;
    case 'nomination.created':
      handleNominationCreated(eventData, store);
      break;
    case 'defense.ended':
      store.commit('vote/setSubPhase', 'voting');
      break;
    case 'vote.cast':
      handleVoteCast(pe, eventData, store);
      break;
    case 'nomination.resolved':
      handleNominationResolved(eventData, store);
      break;
    case 'execution.resolved':
      break;
    case 'time.extended':
      handleTimeExtended(eventData, store);
      break;
    case 'night.action.queued':
      handleNightActionQueued(eventData, store);
      break;
    case 'night.action.completed':
      handleNightActionCompleted(eventData, store);
      break;
    case 'player.died':
    case 'player.executed':
      handlePlayerDied(eventData, store);
      break;
    case 'public.chat':
      handlePublicChat(pe, eventData, store);
      break;
    case 'whisper.sent':
      handleWhisperSent(pe, eventData, store);
      break;
    case 'evil_team.chat':
      handleEvilChat(pe, eventData, store);
      break;
    case 'game.ended':
      store.commit('game/setPhase', 'ended');
      store.commit('game/setWinner', eventData.winner || '');
      store.commit('game/setWinReason', eventData.reason || '');
      store.commit('ui/setScreen', 'end');
      break;
    case 'timer.set': {
      const deadline = parseInt(eventData.deadline, 10) || 0;
      store.commit('game/setPhaseDeadline', deadline);
      break;
    }
    case 'red_herring.assigned':
    case 'reminder.added':
    case 'ai.decision':
    case 'slayer.shot':
    case 'poison.cleared':
    case 'action.requested':
      break;
    default:
      break;
  }
}

function handlePlayerJoined(pe, d, store) {
  const seatNum = parseInt(d.seat_number, 10) || 0;
  const actorId = pe.actor_user_id || d.user_id || '';
  if (seatNum > 0 && actorId) {
    store.commit('players/seatPlayer', { id: actorId, seatIndex: seatNum });
    if (actorId === apiService.userId) store.commit('setSeatIndex', seatNum);
  }
}

function handlePlayerLeft(pe, d, store) {
  const leftActorId = pe.actor_user_id || '';
  if (leftActorId) {
    const leftPlayer = store.state.players.players.find(p => p.id === leftActorId);
    if (leftPlayer) store.commit('players/removePlayer', leftPlayer.seatIndex);
    if (leftActorId === apiService.userId) store.commit('setSeatIndex', -1);
  } else {
    store.commit('players/removePlayer', parseInt(d.seat_number, 10) || 0);
  }
}

function handleSeatClaimed(pe, d, store) {
  const seatNum = parseInt(d.seat_number, 10) || 0;
  const actorId = pe.actor_user_id || '';
  if (!actorId) return;
  const oldEntry = store.state.players.players.find(p => p.id === actorId);
  if (oldEntry && oldEntry.seatIndex !== seatNum) store.commit('players/removePlayer', oldEntry.seatIndex);
  store.commit('players/seatPlayer', { id: actorId, seatIndex: seatNum });
  if (actorId === apiService.userId) store.commit('setSeatIndex', seatNum);
}

function handleRoleAssigned(d, store) {
  const roleId = d.role || '';
  const roleData = store.getters.rolesByKey.get(roleId);
  const localName = i18n.te('roles.' + roleId) ? i18n.t('roles.' + roleId) : (roleData ? roleData.name : roleId);
  const localAbility = i18n.te('roles.' + roleId + '_ability') ? i18n.t('roles.' + roleId + '_ability') : (roleData ? roleData.ability : '');
  store.commit('players/setMyRole', { roleId, roleName: localName, team: d.team || '', ability: localAbility });
}

function handleBluffsAssigned(d, store) {
  let bluffs = d.bluffs;
  if (typeof bluffs === 'string') {
    try { bluffs = JSON.parse(bluffs); } catch (_e) { bluffs = []; }
  }
  store.commit('players/setBluffs', bluffs || []);
}

function handlePhaseDay(store) {
  const newDayCount = store.state.game.dayCount + 1;
  store.commit('game/setPhase', 'day');
  store.commit('game/setDayCount', newDayCount);
  store.commit('players/resetNominationFlags');
  store.commit('vote/endVote');
  store.commit('timeline/addEvent', { type: 'phase_change', dayCount: newDayCount, data: { phase: 'day' } });
}

function addPhaseTimeline(phase, store) {
  store.commit('timeline/addEvent', { type: 'phase_change', dayCount: store.state.game.dayCount, data: { phase } });
}

function handleNominationCreated(d, store) {
  const nominatorSeat = parseInt(d.nominator_seat, 10) || 0;
  const nomineeSeat = parseInt(d.nominee_seat, 10) || 0;
  const alivePlayers = store.state.players.players.filter(p => p.isAlive);
  const requiredMajority = Math.ceil(alivePlayers.length / 2);
  store.commit('vote/startNomination', { nominatorSeat, nomineeSeat, requiredMajority });
  store.commit('players/updatePlayer', { seatIndex: nominatorSeat, property: 'hasNominatedToday', value: true });
  store.commit('players/updatePlayer', { seatIndex: nomineeSeat, property: 'isNominatedToday', value: true });
}

function handleVoteCast(pe, d, store) {
  const voterSeat = parseInt(d.voter_seat, 10) || 0;
  const voteValue = d.vote === 'yes';
  store.commit('vote/castVote', { seatIndex: voterSeat, vote: voteValue });
  store.commit('vote/setCurrentVoter', voterSeat);
  if (pe.actor_user_id === apiService.userId) {
    store.commit('vote/setMyVote', voteValue);
    store.commit('vote/setVotePending', false);
  }
}

function handleNominationResolved(d, store) {
  const result = d.result === 'executed' ? 'executed' : 'safe';
  store.commit('vote/setSubPhase', 'resolved');
  store.commit('vote/setVotePending', false);
  store.commit('vote/setResult', result);
  const yesCount = parseInt(d.votes_for, 10) || parseInt(d.yes_votes, 10) || store.state.vote.currentYesCount;
  store.commit('timeline/addEvent', {
    type: 'vote_result', dayCount: store.state.game.dayCount,
    data: { nomineeSeat: store.state.vote.nominee ? store.state.vote.nominee.seatIndex : -1, yesCount, result }
  });
}

function handleTimeExtended(d, store) {
  const deadline = parseInt(d.deadline, 10) || 0;
  const remaining = parseInt(d.extensions_remaining, 10) || 0;
  store.commit('game/setPhaseDeadline', deadline);
  store.commit('game/setExtensionsUsed', store.state.game.maxExtensions - remaining);
}

function handleNightActionQueued(d, store) {
  if (d.user_id !== apiService.userId) return;
  const nightRoleId = d.role_id || '';
  const nightRoleData = store.getters.rolesByKey.get(nightRoleId);
  const roleName = i18n.te('roles.' + nightRoleId) ? i18n.t('roles.' + nightRoleId) : (nightRoleData ? nightRoleData.name : nightRoleId);
  const abilityText = i18n.te('roles.' + nightRoleId + '_ability') ? i18n.t('roles.' + nightRoleId + '_ability') : (nightRoleData ? nightRoleData.ability : '');
  const actionType = d.action_type || 'passive';
  store.commit('night/openPanel', { roleId: nightRoleId, roleName, abilityText, actionType });
  if (actionType === 'select_one' || actionType === 'select_two') {
    const targets = store.state.players.players
      .filter(p => !p.isMe && p.isAlive)
      .map(p => ({ seatIndex: p.seatIndex, id: p.id }));
    store.commit('night/setTargets', targets);
  }
}

function handleNightActionCompleted(d, store) {
  if (d.user_id !== apiService.userId) return;
  const rawResult = d.result || '';
  store.commit('night/setResult', rawResult === 'timed_out' ? i18n.t('night.timedOut') : rawResult);
}

function handlePlayerDied(d, store) {
  const diedUserId = d.user_id || '';
  const deadPlayer = store.state.players.players.find(p => p.id === diedUserId);
  if (deadPlayer) {
    store.commit('players/killPlayer', deadPlayer.seatIndex);
    store.commit('timeline/addEvent', {
      type: 'death', dayCount: store.state.game.dayCount, data: { seatIndex: deadPlayer.seatIndex }
    });
  }
}

function handlePublicChat(pe, d, store) {
  if (pe.actor_user_id === apiService.userId) return;
  store.commit('chat/addPublicMessage', { seatIndex: parseInt(d.sender_seat, 10) || -1, text: d.message || '', isSystem: false });
}

function handleWhisperSent(pe, d, store) {
  if (pe.actor_user_id === apiService.userId) return;
  const senderSeat = parseInt(d.sender_seat, 10) || -1;
  store.commit('chat/addWhisperMessage', { targetSeat: senderSeat, data: { seatIndex: senderSeat, text: d.message || '' } });
}

function handleEvilChat(pe, d, store) {
  if (pe.actor_user_id === apiService.userId) return;
  store.commit('chat/addEvilMessage', { seatIndex: parseInt(d.sender_seat, 10) || -1, text: d.message || '' });
}
