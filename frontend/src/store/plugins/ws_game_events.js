// WebSocket 游戏事件处理：将后端 ProjectedEvent 映射到 Vuex mutations
//
// [IN]  websocket.js（WebSocketManager 调用）
// [OUT] store modules（通过 store.commit 更新状态）
// [POS] 处理 25+ 种后端事件类型
import apiService from "../../services/ApiService";
import i18n from "../../i18n";

// 已处理事件的 seq 去重集合，防止 WS 订阅追赶与广播重叠导致重复处理
const _processedSeqs = new Set();
const MAX_DEDUP_SIZE = 500;

/**
 * Process a ProjectedEvent from the backend.
 * @param {object} pe - ProjectedEvent { event_type, data, seq, server_ts, actor_user_id }
 * @param {object} store - Vuex store instance
 */
export function processGameEvent(pe, store) {
  if (!pe || !pe.event_type) return;
  // 按 seq 去重：同一事件只处理一次
  if (pe.seq && _processedSeqs.has(pe.seq)) return;
  if (pe.seq) {
    _processedSeqs.add(pe.seq);
    if (_processedSeqs.size > MAX_DEDUP_SIZE) {
      const arr = Array.from(_processedSeqs);
      arr.splice(0, arr.length - MAX_DEDUP_SIZE / 2).forEach(s => _processedSeqs.delete(s));
    }
  }
  const eventType = pe.event_type;
  let eventData = pe.data;
  if (typeof eventData === 'string') {
    try { eventData = JSON.parse(eventData); } catch (_e) { eventData = {}; }
  }
  if (!eventData) eventData = {};

  console.log('[DBG] processGameEvent:', eventType);

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
      store.commit('ui/setScreen', 'game');
      store.commit('night/clearNightInfoHistory');
      store.commit('night/clearGrimoireHistory');
      _processedSeqs.clear();
      break;
    case 'phase.first_night':
      store.commit('game/setPhase', 'first_night');
      store.commit('game/setDayCount', 0);
      store.commit('night/showRoleReveal');
      addPhaseTimeline('first_night', store);
      break;
    case 'phase.night':
      store.commit('game/setPhase', 'night');
      store.commit('night/setStep', 'sleeping');
      store.commit('players/resetNominationFlags');
      addPhaseTimeline('night', store);
      break;
    case 'phase.day':
      handlePhaseDay(eventData, store);
      break;
    case 'phase.nomination':
      store.commit('game/setPhase', 'nomination');
      addPhaseTimeline('nomination', store);
      break;
    case 'nomination.created':
      console.log('[DBG] nomination.created raw data:', JSON.stringify(eventData));
      handleNominationCreated(eventData, store);
      console.log('[DBG] after handleNominationCreated - voteOrder:', store.state.vote.voteOrder, 'currentVoter:', store.state.vote.currentVoterSeatIndex, 'subPhase:', store.state.vote.subPhase);
      break;
    case 'defense.progress':
      {
        const currentNominatorEnded = !!store.state.vote.nominatorEnded;
        const currentNomineeEnded = !!store.state.vote.nomineeEnded;
        const nominatorSeat = store.state.vote.nominator ? store.state.vote.nominator.seatIndex : -1;
        const nomineeSeat = store.state.vote.nominee ? store.state.vote.nominee.seatIndex : -1;

        let nominatorEnded = currentNominatorEnded;
        let nomineeEnded = currentNomineeEnded;

        // Prefer explicit booleans if backend provides them.
        if (eventData.nominator_ended !== undefined || eventData.nominee_ended !== undefined) {
          if (eventData.nominator_ended !== undefined) {
            nominatorEnded = eventData.nominator_ended === 'true';
          }
          if (eventData.nominee_ended !== undefined) {
            nomineeEnded = eventData.nominee_ended === 'true';
          }
        } else {
          // Current backend emits only user_id for defense.progress.
          const progressedUserId = eventData.user_id || '';
          const progressedPlayer = store.state.players.players.find(player => player.id === progressedUserId);
          const progressedSeat = progressedPlayer ? progressedPlayer.seatIndex : -1;

          if (progressedSeat > 0) {
            nominatorEnded = currentNominatorEnded || progressedSeat === nominatorSeat;
            nomineeEnded = currentNomineeEnded || progressedSeat === nomineeSeat;
          }
        }

        store.commit('vote/setDefenseProgress', {
          nominatorEnded,
          nomineeEnded
        });
      }
      break;
    case 'defense.ended':
      console.log('[DBG] defense.ended received, setting subPhase to voting');
      store.commit('vote/setSubPhase', 'voting');
      console.log('[DBG] after defense.ended - subPhase:', store.state.vote.subPhase, 'currentVoter:', store.state.vote.currentVoterSeatIndex, 'mySeat:', store.state.seatIndex);
      break;
    case 'vote.cast':
      console.log('[DBG] vote.cast received:', JSON.stringify(eventData));
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
      break;
    case 'night.action.prompt':
      handleNightActionPrompt(eventData, store);
      break;
    case 'night.action.completed':
      handleNightActionCompleted(eventData, store);
      break;
    case 'night.info':
      handleNightInfo(eventData, store);
      break;
    case 'team.recognition':
      handleTeamRecognition(eventData, store);
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
    case 'game.recap':
      store.commit('game/setRecap', eventData.summary || '');
      break;
    case 'timer.set': {
      const deadline = parseInt(eventData.deadline, 10) || 0;
      store.commit('game/setPhaseDeadline', deadline);
      break;
    }
    case 'red_herring.assigned':
      break;
    case 'reminder.added':
      handleReminderAdded(eventData, store);
      break;
    case 'ai.decision':
      break;
    case 'slayer.shot':
      handleSlayerShot(eventData, store);
      break;
    case 'poison.cleared':
      handlePoisonCleared(store);
      break;
    case 'poison.rollback':
      break;
    case 'player.poisoned':
      handlePlayerPoisoned(eventData, store);
      break;
    case 'player.protected':
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
  // 防御性校验：只接受属于自己的角色分配事件
  if (d.user_id && d.user_id !== apiService.userId) {
    console.warn('[handleRoleAssigned] ignoring event for other user:', d.user_id);
    return;
  }
  const roleId = d.role || '';
  const roleData = store.getters.rolesByKey.get(roleId);
  const localName = i18n.te('roles.' + roleId) ? i18n.t('roles.' + roleId) : (roleData ? roleData.name : roleId);
  const localAbility = i18n.te('roles.' + roleId + '_ability') ? i18n.t('roles.' + roleId + '_ability') : (roleData ? roleData.ability : '');
  store.commit('players/setMyRole', {
    roleId,
    roleName: localName,
    team: d.team || '',
    ability: localAbility,
    isPoisoned: !!d.is_poisoned,
    reminders: Array.isArray(d.reminders) ? d.reminders : []
  });
}

function handleBluffsAssigned(d, store) {
  // 防御性校验：bluffs 只应分配给恶魔玩家自己
  if (d.user_id && d.user_id !== apiService.userId) return;
  let bluffs = d.bluffs;
  if (typeof bluffs === 'string') {
    try { bluffs = JSON.parse(bluffs); } catch (_e) { bluffs = []; }
  }
  store.commit('players/setBluffs', bluffs || []);
}

function handlePhaseDay(d, store) {
  const newDayCount = store.state.game.dayCount + 1;
  store.commit('game/setPhase', 'day');
  store.commit('game/setDayCount', newDayCount);
  store.commit('players/resetNominationFlags');
  store.commit('vote/endVote');
  // 安全护栏：若正在展示团队认知界面，延迟重置夜晚状态
  if (store.state.night.step === 'team_reveal') {
    store._pendingNightReset = true;
  } else {
    store.commit('night/reset');
  }
  store.commit('timeline/addEvent', { type: 'phase_change', dayCount: newDayCount, data: { phase: 'day' } });

  const nightDeaths = parseNightDeaths(d);
  const announcement = buildMorningAnnouncement(nightDeaths);
  store.commit('chat/addPublicMessage', {
    seatIndex: -1,
    text: announcement,
    isSystem: true
  });
}

function parseNightDeaths(d) {
  if (!d || !d.night_deaths) return [];
  if (Array.isArray(d.night_deaths)) return d.night_deaths.map(n => parseInt(n, 10)).filter(n => n > 0);

  try {
    const parsed = JSON.parse(d.night_deaths);
    if (!Array.isArray(parsed)) return [];
    return parsed.map(n => parseInt(n, 10)).filter(n => n > 0);
  } catch (_e) {
    return [];
  }
}

function buildMorningAnnouncement(nightDeaths) {
  if (!nightDeaths.length) {
    return i18n.t('game.peacefulNightAnnouncement');
  }

  const seats = [...nightDeaths]
    .sort((a, b) => a - b)
    .map(seat => `${seat}号`)
    .join('、');

  return i18n.t('game.morningDeathAnnouncement', { seats });
}

function addPhaseTimeline(phase, store) {
  store.commit('timeline/addEvent', { type: 'phase_change', dayCount: store.state.game.dayCount, data: { phase } });
}

function handleNominationCreated(d, store) {
  const nominatorSeat = parseInt(d.nominator_seat, 10) || 0;
  const nomineeSeat = parseInt(d.nominee_seat, 10) || 0;
  const nominatorEnded = d.nominator_ended === 'true';
  const nomineeEnded = d.nominee_ended === 'true';
  const alivePlayers = store.state.players.players.filter(p => p.isAlive);
  const requiredMajority = Math.ceil(alivePlayers.length / 2);
  // Parse sequential vote order (seat numbers, clockwise from nominee+1)
  let voteOrder = [];
  if (d.vote_order) {
    try { voteOrder = JSON.parse(d.vote_order); } catch (_e) { voteOrder = []; }
  }
  store.commit('vote/startNomination', { nominatorSeat, nomineeSeat, requiredMajority, voteOrder, nominatorEnded, nomineeEnded });
  store.commit('players/updatePlayer', { seatIndex: nominatorSeat, property: 'hasNominatedToday', value: true });
  store.commit('players/updatePlayer', { seatIndex: nomineeSeat, property: 'isNominatedToday', value: true });
}

function handleVoteCast(pe, d, store) {
  const voterSeat = parseInt(d.voter_seat, 10) || 0;
  const voteValue = d.vote === 'yes';
  store.commit('vote/castVote', { seatIndex: voterSeat, vote: voteValue });
  // Note: castVote already advances currentVoterSeatIndex to the next voter
  if (pe.actor_user_id === apiService.userId) {
    store.commit('vote/setMyVote', voteValue);
    store.commit('vote/setVotePending', false);
  }
}

function handleNominationResolved(d, store) {
  let result;
  switch (d.result) {
    case 'on_the_block': result = 'on_the_block'; break;
    case 'tied': result = 'tied'; break;
    case 'executed': result = 'executed'; break; // legacy compat
    default: result = 'safe';
  }
  store.commit('vote/setSubPhase', 'resolved');
  store.commit('vote/setVotePending', false);
  store.commit('vote/setResult', result);

  // Auto-close VoteOverlay after 3 seconds
  setTimeout(() => {
    store.commit('vote/endVote');
  }, 3000);

  const yesCount = parseInt(d.votes_for, 10) || parseInt(d.yes_votes, 10) || store.state.vote.currentYesCount;

  // 优化播报：x提x，投票玩家xxxxx
  const nominatorSeat = store.state.vote.nominator ? store.state.vote.nominator.seatIndex : -1;
  const nomineeSeat = store.state.vote.nominee ? store.state.vote.nominee.seatIndex : -1;
  const voters = store.state.vote.votes
    .filter(v => v.vote)
    .map(v => `${v.seatIndex}号`)
    .join('、');
  
  const summary = i18n.t('vote.summary', {
    nominator: nominatorSeat > 0 ? nominatorSeat : '?',
    nominee: nomineeSeat > 0 ? nomineeSeat : '?',
    voters: voters || i18n.t('vote.noVoters'),
    count: yesCount
  });

  store.commit('chat/addPublicMessage', { 
    seatIndex: -1, 
    text: summary, 
    isSystem: true 
  });
}

function handleTimeExtended(d, store) {
  const deadline = parseInt(d.deadline, 10) || 0;
  const remaining = parseInt(d.extensions_remaining, 10) || 0;
  store.commit('game/setPhaseDeadline', deadline);
  store.commit('game/setExtensionsUsed', store.state.game.maxExtensions - remaining);
}

function handleNightActionPrompt(d, store) {
  console.log('[DBG] handleNightActionPrompt:', d.user_id, 'my:', apiService.userId, 'role:', d.role_id, 'action_type:', d.action_type);
  if (d.user_id !== apiService.userId) return;
  const nightRoleId = d.role_id || '';
  const nightRoleData = store.getters.rolesByKey.get(nightRoleId);
  const roleName = i18n.te('roles.' + nightRoleId) ? i18n.t('roles.' + nightRoleId) : (nightRoleData ? nightRoleData.name : nightRoleId);
  const abilityText = i18n.te('roles.' + nightRoleId + '_ability') ? i18n.t('roles.' + nightRoleId + '_ability') : (nightRoleData ? nightRoleData.ability : '');
  const actionType = d.action_type || 'passive';
  let targets = [];
  if (actionType === 'select_one' || actionType === 'select_two') {
    targets = store.state.players.players
      .filter(p => isNightActionTargetAllowed(nightRoleId, p))
      .sort((a, b) => a.seatIndex - b.seatIndex)
      .map(p => ({ seatIndex: p.seatIndex, id: p.id }));
  }
  store.commit('night/queuePrompt', { roleId: nightRoleId, roleName, abilityText, actionType, targets });
}

function isNightActionTargetAllowed(roleId, player) {
  if (!player) return false;
  if (roleId === 'imp') return true;
  if (roleId === 'poisoner') return !player.isMe;
  return !player.isMe && player.isAlive;
}

function handleNightActionCompleted(d, store) {
  if (d.user_id !== apiService.userId) return;
  // Skip auto-complete results during game start (e.g. imp first night no_action)
  // to avoid overriding role_reveal or idle state
  const step = store.state.night.step;
  if (step === 'idle' || step === 'role_reveal' || step === 'sleeping') return;
  // night.action.completed 现在只记录意图，不含 result。
  // 信息结果由后续 night.info 事件提供。
  // 只处理 timed_out 的特殊提示。
  const rawResult = d.result || '';
  if (rawResult === 'timed_out') {
    store.commit('night/setResult', i18n.t('night.timedOut'));
  } else {
    // 行动已提交，显示等待状态（等待 night.info）
    store.commit('night/setStep', 'waiting');
  }
}

function handlePlayerDied(d, store) {
  const diedUserId = d.user_id || '';
  const deadPlayer = store.state.players.players.find(p => p.id === diedUserId);
  if (deadPlayer) {
    store.commit('players/killPlayer', deadPlayer.seatIndex);
    if (d.cause !== 'slayer') {
      store.commit('timeline/addEvent', {
        type: 'death', dayCount: store.state.game.dayCount, data: { seatIndex: deadPlayer.seatIndex }
      });
    }
  }
}

function handleReminderAdded(d, store) {
  if (d.user_id !== apiService.userId) return;
  const current = store.state.players.myRole && Array.isArray(store.state.players.myRole.reminders)
    ? store.state.players.myRole.reminders
    : [];
  if (current.includes(d.reminder)) return;
  store.commit('players/updateMyRole', { reminders: [...current, d.reminder] });
}

function handlePlayerPoisoned(d, store) {
  if (d.user_id !== apiService.userId) return;
  store.commit('players/updateMyRole', { isPoisoned: true });
}

function handlePoisonCleared(store) {
  if (!store.state.players.myRole) return;
  store.commit('players/updateMyRole', { isPoisoned: false });
}

function handleSlayerShot(d, store) {
  store.commit('timeline/addEvent', {
    type: 'ability',
    dayCount: store.state.game.dayCount,
    data: {
      ability: 'slayer_shot',
      shooterSeat: parseInt(d.shooter_seat, 10) || 0,
      targetSeat: parseInt(d.target_seat, 10) || 0,
      result: d.result || 'no_effect'
    }
  });
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

function handleNightInfo(d, store) {
  if (d.user_id !== apiService.userId) return;
  const message = d.message || '';
  // 设置夜晚信息结果（如果当前还在夜晚面板中）
  store.commit('night/setResult', message);
  // 存储详细信息供 UI 展示
  let content = d.content;
  if (typeof content === 'string') {
    try { content = JSON.parse(content); } catch (_e) { /* keep as string */ }
  }
  const detail = {
    roleId: d.role_id || '',
    infoType: d.info_type || '',
    content,
    message
  };
  store.commit('night/setNightInfoDetail', detail);
  const nightNumber = store.state.game.dayCount + 1;
  if (detail.infoType === 'grimoire') {
    store.commit('night/setGrimoireEntry', {
      ...detail,
      nightNumber
    });
    return;
  }
  // 追加到历史记录（夜晚编号 = dayCount + 1，因为 night.info 在 phase.day 之前触发）
  store.commit('night/pushNightInfo', {
    ...detail,
    nightNumber
  });
}

function handleTeamRecognition(d, store) {
  if (d.user_id !== apiService.userId) return;
  let minionIds = d.minion_ids;
  if (typeof minionIds === 'string') {
    try { minionIds = JSON.parse(minionIds); } catch (_e) { minionIds = []; }
  }
  let bluffs = d.bluffs;
  if (typeof bluffs === 'string') {
    try { bluffs = JSON.parse(bluffs); } catch (_e) { bluffs = []; }
  }
  const data = {
    team: d.team || 'evil',
    demonId: d.demon_id || '',
    minionIds: minionIds || [],
    bluffs: bluffs || []
  };
  store.commit('night/setTeamRecognition', data);
  // 已在 team_reveal 或 sleeping 状态时，确保展示数据
  const curStep = store.state.night.step;
  if (curStep === 'sleeping' || curStep === 'idle') {
    store.commit('night/showTeamReveal');
  }
  // 如果是 team_reveal 状态（数据换到 team_reveal 后才到），无需额外操作，计算属性自动更新

  // 构建队友信息文本追加到夜晚查验历史
  const isDemon = data.bluffs && data.bluffs.length > 0;
  const players = store.state.players.players;
  let message = '';
  if (isDemon) {
    const mNames = data.minionIds.map(id => {
      const p = players.find(pl => pl.id === id);
      return p ? i18n.t('lobby.seat', { n: p.seatIndex }) : id;
    }).join(', ');
    const bNames = data.bluffs.map(b => {
      const key = 'roles.' + b;
      return i18n.te(key) ? i18n.t(key) : b;
    }).join(', ');
    message = i18n.t('teamReveal.demonSummary', { minions: mNames, bluffs: bNames });
  } else {
    const dp = players.find(pl => pl.id === data.demonId);
    const dName = dp ? i18n.t('lobby.seat', { n: dp.seatIndex }) : data.demonId;
    message = i18n.t('teamReveal.minionSummary', { demon: dName });
  }
  store.commit('night/pushNightInfo', {
    roleId: isDemon ? 'imp' : (d.role || 'minion'),
    infoType: 'team_recognition',
    content: data,
    message,
    nightNumber: 1
  });
}
