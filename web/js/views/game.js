// ============================================
// 游戏主界面 - 三栏布局 (通知 | 座位 | 聊天)
// ============================================

import { store } from '../store.js';
import { wsSendCommand, api, refreshState } from '../api.js';
import { toast, escapeHtml } from '../utils.js';
import { navigate } from '../app.js';
import { getRoleData, getRoleName, getRoleIcon } from '../roles.js';

let stateUnsub = null;
let msgUnsub = null;
let timerInterval = null;
let activeChatTab = 'public'; // 'public' | 'private' | 'evil_team'
let whisperTargetId = '';

// ---- Main Render ----

export function renderGame() {
  return `
    <div class="game-page phase-${store.getPhase()}" id="game-page">
      ${renderPhaseBanner()}
      <div class="game-layout-3col">
        <div class="notif-panel" id="notif-panel">
          <div class="panel-header">📢 广播通知</div>
          <div class="notif-list" id="notif-list">
            ${renderNotifications()}
          </div>
        </div>
        <div class="center-panel">
          <div class="townsquare-container">
            ${renderTownSquare()}
          </div>
          <div class="action-area" id="action-area">
            ${renderActionPanel()}
          </div>
        </div>
        <div class="chat-panel" id="chat-panel">
          ${renderChatPanelInner()}
        </div>
      </div>
      ${renderGameOverOverlay()}
    </div>
  `;
}

function renderPlayerOptions() {
  const players = store.getPlayerList();
  const myId = store.get('userId');
  return players
    .filter(p => p.user_id !== myId)
    .map(p => `<option value="${p.user_id}" ${whisperTargetId === p.user_id ? 'selected' : ''}>${p.seat_number}. ${p.name || '玩家'}</option>`)
    .join('');
}

function renderPhaseBanner() {
  const gs = store.get('gameState');
  if (!gs) return '<div class="phase-banner"><div class="phase-text">加载中...</div></div>';

  const phase = gs.phase || 'lobby';
  const subPhase = gs.sub_phase || '';
  const dayCount = gs.day_count || 0;
  const nightCount = gs.night_count || 0;

  const phaseInfo = getPhaseInfo(phase, subPhase, dayCount, nightCount);
  const myPlayer = store.getMyPlayer();
  const myRole = myPlayer && myPlayer.role ? getRoleData(myPlayer.role) : null;

  return `
    <div class="phase-banner">
      <div class="phase-left">
        <span class="phase-icon">${phaseInfo.icon}</span>
        <span class="phase-text">${phaseInfo.text}</span>
        <span class="phase-sub">${phaseInfo.sub}</span>
      </div>
      <div class="phase-right">
        <span class="phase-timer" id="phase-timer"></span>
        ${myRole ? `
          <div class="role-badge">
            <span class="role-icon">${myRole.icon}</span>
            <span class="role-name">${myRole.name}</span>
          </div>
        ` : ''}
      </div>
    </div>
  `;
}

function getPhaseInfo(phase, subPhase, dayCount, nightCount) {
  switch (phase) {
    case 'first_night':
      return { icon: '🌙', text: '第一个夜晚', sub: '' };
    case 'night':
      return { icon: '🌙', text: `第 ${nightCount} 夜`, sub: '' };
    case 'day':
      if (subPhase === 'discussion') return { icon: '☀️', text: `第 ${dayCount} 天`, sub: '讨论中' };
      return { icon: '☀️', text: `第 ${dayCount} 天`, sub: '' };
    case 'nomination':
      if (subPhase === 'defense') return { icon: '⚖️', text: '提名阶段', sub: '辩护中' };
      if (subPhase === 'voting') return { icon: '🗳️', text: '提名阶段', sub: '投票中' };
      return { icon: '📢', text: '提名阶段', sub: '可提名' };
    case 'ended':
      return { icon: '🏁', text: '游戏结束', sub: '' };
    default:
      return { icon: '⏳', text: '等待中', sub: phase };
  }
}

// ---- Town Square ----

function renderTownSquare() {
  const players = store.getPlayerList();
  if (players.length === 0) {
    return '<div class="townsquare"><div class="townsquare-center"><div class="day-count">等待中</div></div></div>';
  }

  const gs = store.get('gameState') || {};
  const myId = store.get('userId');
  const nomination = gs.nomination;
  const aliveCount = players.filter(p => p.alive).length;

  // Calculate positions in a circle
  const tokenHtml = players.map((p, i) => {
    const angle = (2 * Math.PI * i / players.length) - Math.PI / 2;
    const radius = 42; // % from center
    const x = 50 + radius * Math.cos(angle);
    const y = 50 + radius * Math.sin(angle);

    const isMe = p.user_id === myId;
    const isNominee = nomination && nomination.nominee === p.user_id;
    const isNominator = nomination && nomination.nominator === p.user_id;

    let voteClass = '';
    if (nomination && nomination.votes) {
      if (nomination.votes[p.user_id] === true) voteClass = 'voted-yes';
      else if (nomination.votes[p.user_id] === false) voteClass = 'voted-no';
    }

    const classes = [
      'player-token',
      p.alive ? 'alive' : 'dead',
      isMe ? 'is-me' : '',
      isNominee ? 'nominated' : '',
      isNominator ? 'nominator' : '',
      voteClass,
    ].filter(Boolean).join(' ');

    const myPlayer = store.getMyPlayer();
    const showRole = isMe && p.role;
    const roleData = showRole ? getRoleData(p.role) : null;

    return `
      <div class="${classes}" style="left:${x}%;top:${y}%" data-player-id="${p.user_id}" data-seat="${p.seat_number}">
        <div class="token-circle">
          ${roleData ? roleData.icon : '👤'}
          <span class="token-seat">${p.seat_number}</span>
          <span class="token-shroud">☠️</span>
          ${!p.alive && p.has_ghost_vote ? '<span class="ghost-vote-badge">👻</span>' : ''}
          <span class="token-vote-indicator">${voteClass === 'voted-yes' ? '✅' : voteClass === 'voted-no' ? '❌' : ''}</span>
        </div>
        <span class="token-name">${p.name || '玩家'}</span>
        ${showRole ? `<span class="token-role">${roleData.name}</span>` : ''}
      </div>
    `;
  }).join('');

  return `
    <div class="townsquare">
      ${tokenHtml}
      <div class="townsquare-center">
        <div class="day-count">${gs.phase === 'night' || gs.phase === 'first_night' ? '🌙' : '☀️'}</div>
        <div class="alive-count">存活: ${aliveCount}/${players.length}</div>
      </div>
    </div>
  `;
}

// ---- Action Panel ----

function renderActionPanel() {
  const gs = store.get('gameState');
  if (!gs) return '';

  const phase = gs.phase;
  const subPhase = gs.sub_phase || '';
  const myPlayer = store.getMyPlayer();
  if (!myPlayer) return '';

  // Night phase - show ability panel
  if ((phase === 'first_night' || phase === 'night') && store.get('nightActionPending')) {
    return renderNightActionPanel(gs, myPlayer);
  }

  // Night phase - waiting
  if (phase === 'first_night' || phase === 'night') {
    const nightResult = store.get('nightResult');
    return `
      <div class="action-panel">
        <h4>🌙 夜晚</h4>
        <div class="waiting-indicator">
          <div class="waiting-dots"><span></span><span></span><span></span></div>
          <div>夜晚进行中，请等待...</div>
        </div>
        ${nightResult ? `<div class="night-result">${nightResult}</div>` : ''}
      </div>
    `;
  }

  // Day phase - nomination controls
  if (phase === 'day' || phase === 'nomination') {
    return renderDayActionPanel(gs, myPlayer);
  }

  return '';
}

function renderNightActionPanel(gs, myPlayer) {
  const roleId = myPlayer.role;
  const roleData = getRoleData(roleId);
  const players = store.getPlayerList().filter(p => p.user_id !== myPlayer.user_id && p.alive);

  // Determine how many targets needed
  const needsTarget = ['poisoner', 'fortune_teller', 'monk', 'ravenkeeper', 'empath', 'butler', 'imp', 'washerwoman', 'librarian', 'investigator'].includes(roleId);
  const needsTwoTargets = ['fortune_teller', 'washerwoman', 'librarian', 'investigator'].includes(roleId);
  const noTarget = ['chef', 'soldier', 'mayor', 'virgin', 'saint', 'drunk', 'recluse', 'baron', 'scarlet_woman'].includes(roleId);

  if (noTarget) {
    // Auto-complete abilities that don't need targets
    return `
      <div class="action-panel">
        <h4>${roleData.icon} ${roleData.name} - 你的能力</h4>
        <div class="action-description">${roleData.desc}</div>
        <div class="waiting-indicator">
          <div class="waiting-dots"><span></span><span></span><span></span></div>
          <div>系统正在自动结算你的能力...</div>
        </div>
      </div>
    `;
  }

  const selectedTargets = store.get('selectedTargets') || [];

  return `
    <div class="action-panel">
      <h4>${roleData.icon} ${roleData.name} - 你的能力</h4>
      <div class="action-description">${roleData.desc}</div>
      <div style="font-size:0.8rem;color:var(--text-muted);margin-bottom:8px;">
        ${needsTwoTargets ? '请选择 2 名玩家' : '请选择 1 名玩家'}
      </div>
      <div class="action-targets" id="night-targets">
        ${players.map(p => `
          <button class="target-btn ${selectedTargets.includes(p.user_id) ? 'selected' : ''}"
                  data-target-id="${p.user_id}">
            ${p.seat_number}. ${p.name || '玩家'}
          </button>
        `).join('')}
        ${roleId === 'imp' ? `
          <button class="target-btn ${selectedTargets.includes(myPlayer.user_id) ? 'selected' : ''}"
                  data-target-id="${myPlayer.user_id}">
            ${myPlayer.seat_number}. ${myPlayer.name} (自己)
          </button>
        ` : ''}
      </div>
      <div class="action-submit">
        <button class="btn btn-primary" id="ability-submit-btn" ${selectedTargets.length === 0 ? 'disabled' : ''}>
          确认
        </button>
      </div>
    </div>
  `;
}

function renderDayActionPanel(gs, myPlayer) {
  const phase = gs.phase;
  const subPhase = gs.sub_phase || '';
  const nomination = gs.nomination;
  const players = store.getPlayerList();

  // Active nomination - voting phase
  if (nomination && !nomination.resolved && subPhase === 'voting') {
    const hasVoted = nomination.votes && nomination.votes[myPlayer.user_id] !== undefined;
    const canVote = myPlayer.alive || myPlayer.has_ghost_vote;
    const nominator = players.find(p => p.user_id === nomination.nominator);
    const nominee = players.find(p => p.user_id === nomination.nominee);

    return `
      <div class="nomination-panel">
        <h4>🗳️ 投票</h4>
        <div class="nomination-info">
          <span class="nom-player">${nominator ? nominator.name : '?'}</span>
          <span class="nom-arrow">➜ 提名 ➜</span>
          <span class="nom-player">${nominee ? nominee.name : '?'}</span>
        </div>
        <div class="vote-tally">
          <span class="yes-count">✅ 赞成: ${nomination.votes_for || 0}</span>
          <span class="no-count">❌ 反对: ${nomination.votes_against || 0}</span>
          <span class="threshold">需要: ${nomination.threshold || '?'}</span>
        </div>
        ${!hasVoted && canVote ? `
          <div class="vote-buttons">
            <button class="btn btn-vote-yes" id="vote-yes-btn">✅ 赞成</button>
            <button class="btn btn-vote-no" id="vote-no-btn">❌ 反对</button>
          </div>
        ` : hasVoted ? `
          <div style="color:var(--text-secondary);text-align:center;padding:8px;">你已投票</div>
        ` : `
          <div style="color:var(--text-muted);text-align:center;padding:8px;">你无法投票</div>
        `}
      </div>
    `;
  }

  // Active nomination - defense phase
  if (nomination && !nomination.resolved && subPhase === 'defense') {
    const nominee = players.find(p => p.user_id === nomination.nominee);
    const isNominee = nomination.nominee === myPlayer.user_id;

    return `
      <div class="nomination-panel">
        <h4>⚖️ 辩护阶段</h4>
        <div class="action-description">${nominee ? nominee.name : '?'} 正在进行辩护...</div>
        ${isNominee ? `
          <button class="btn btn-secondary" id="end-defense-btn">结束辩护</button>
        ` : ''}
      </div>
    `;
  }

  // Nomination open - can nominate
  if (myPlayer.alive && !myPlayer.has_nominated && (phase === 'nomination' || phase === 'day')) {
    const nominees = players.filter(p => !p.was_nominated && p.user_id !== myPlayer.user_id);

    return `
      <div class="action-panel">
        <h4>📢 提名</h4>
        <div class="action-description">选择一名玩家发起提名</div>
        <div class="action-targets" id="nominate-targets">
          ${nominees.map(p => `
            <button class="target-btn" data-nominate-id="${p.user_id}">
              ${p.seat_number}. ${p.name || '玩家'} ${!p.alive ? '(已死亡)' : ''}
            </button>
          `).join('')}
        </div>
        ${myPlayer.role === 'slayer' ? `
          <div style="margin-top:12px;padding-top:12px;border-top:1px solid var(--border-color);">
            <h4>🔫 杀手能力</h4>
            <div class="action-description">指定一名玩家，若其为恶魔则立即死亡（仅一次）</div>
            <div class="action-targets" id="slayer-targets">
              ${players.filter(p => p.alive && p.user_id !== myPlayer.user_id).map(p => `
                <button class="target-btn" data-slayer-target="${p.user_id}">
                  ${p.seat_number}. ${p.name || '玩家'}
                </button>
              `).join('')}
            </div>
          </div>
        ` : ''}
      </div>
    `;
  }

  // Default day panel
  return `
    <div class="action-panel">
      <h4>☀️ 白天</h4>
      <div class="action-description">
        ${!myPlayer.alive ? '你已死亡，仍可参与讨论' :
          myPlayer.has_nominated ? '你今天已经提名过了' :
          '等待提名阶段...'}
      </div>
    </div>
  `;
}

// ---- Notifications (Left Panel) ----

function renderNotifications() {
  const messages = store.get('messages') || [];
  return messages
    .filter(m => ['system', 'announcement', 'death'].includes(m.type))
    .map(msg => {
      switch (msg.type) {
        case 'announcement':
          return `<div class="notif-item announcement">${escapeHtml(msg.text)}</div>`;
        case 'death':
          return `<div class="notif-item death">${escapeHtml(msg.text)}</div>`;
        default:
          return `<div class="notif-item system">${escapeHtml(msg.text)}</div>`;
      }
    }).join('');
}

// ---- Chat Messages (Right Panel) ----

function renderChatMessages() {
  const messages = store.get('messages') || [];
  if (activeChatTab === 'public') {
    const filtered = messages.filter(m => m.type === 'public');
    if (filtered.length === 0) return '<div class="chat-empty">暂无公共消息</div>';
    return filtered
      .map(msg => `<div class="chat-msg public"><span class="chat-sender">[${msg.senderSeat || '?'}] ${escapeHtml(msg.sender)}</span>${escapeHtml(msg.text)}</div>`)
      .join('');
  } else if (activeChatTab === 'evil_team') {
    const filtered = messages.filter(m => m.type === 'evil_team');
    if (filtered.length === 0) return '<div class="chat-empty">坏人群聊 — 只有恶魔和爪牙能看到</div>';
    return filtered
      .map(msg => `<div class="chat-msg evil-team"><span class="chat-sender">😈 [${msg.senderSeat || '?'}] ${escapeHtml(msg.sender)}</span>${escapeHtml(msg.text)}</div>`)
      .join('');
  } else {
    // Private tab - show whisper messages
    const filtered = messages.filter(m => m.type === 'whisper');
    if (filtered.length === 0) return '<div class="chat-empty">暂无私聊消息</div>';
    return filtered
      .map(msg => {
        const dirClass = msg.fromMe ? 'whisper-sent' : 'whisper-received';
        const label = msg.fromMe ? `→ ${escapeHtml(msg.sender)}` : `← ${escapeHtml(msg.sender)}`;
        return `<div class="chat-msg whisper ${dirClass}"><span class="chat-sender">🔒 ${label}</span> ${escapeHtml(msg.text)}</div>`;
      })
      .join('');
  }
}

// ---- Game Over ----

function renderGameOverOverlay() {
  const gs = store.get('gameState');
  if (!gs || gs.phase !== 'ended') return '';

  const isGoodWin = gs.winner === 'good';
  const players = store.getPlayerList();

  return `
    <div class="game-over-overlay" id="game-over-overlay">
      <div class="game-over-content">
        <div class="game-over-title ${isGoodWin ? 'good-wins' : 'evil-wins'}">
          ${isGoodWin ? '☀️ 善良阵营获胜！' : '🩸 邪恶阵营获胜！'}
        </div>
        <div class="game-over-reason">${gs.win_reason || ''}</div>
        <div class="game-over-roles">
          ${players.map(p => `
            <div class="role-reveal">
              <div class="player-name">${p.name || '玩家'}</div>
              <div class="role-name ${p.team || ''}">${p.role ? getRoleName(p.role) : '?'}</div>
            </div>
          `).join('')}
        </div>
        ${renderAIDecisionLog(gs)}
        <button class="btn btn-primary btn-lg" id="game-over-back-btn">返回大厅</button>
      </div>
    </div>
  `;
}

function renderAIDecisionLog(gs) {
  const log = gs.ai_decision_log;
  if (!log || log.length === 0) return '';

  return `
    <div class="ai-decision-log">
      <h3 class="ai-log-title">📋 说书人决策记录</h3>
      <table class="ai-log-table">
        <thead>
          <tr>
            <th>夜晚</th>
            <th>玩家</th>
            <th>角色</th>
            <th>目标</th>
            <th>真实结果</th>
            <th>给出结果</th>
            <th>状态</th>
          </tr>
        </thead>
        <tbody>
          ${log.map(entry => {
            const statusTags = [];
            if (entry.is_poisoned) statusTags.push('<span class="ai-tag poisoned">中毒</span>');
            if (entry.is_drunk) statusTags.push('<span class="ai-tag drunk">酒鬼</span>');
            if (!entry.is_poisoned && !entry.is_drunk) statusTags.push('<span class="ai-tag normal">正常</span>');
            const modified = entry.true_result !== entry.given_result;
            return `
              <tr class="${modified ? 'ai-modified' : ''}">
                <td>${entry.night}</td>
                <td>${entry.player_name}</td>
                <td>${getRoleName(entry.role)}</td>
                <td>${entry.targets || '-'}</td>
                <td>${entry.true_result || '-'}</td>
                <td>${entry.given_result || '-'}</td>
                <td>${statusTags.join('')}</td>
              </tr>
            `;
          }).join('')}
        </tbody>
      </table>
    </div>
  `;
}

// ---- Mount/Unmount ----

export function mountGame() {
  const roomId = store.get('roomId');

  // Initial refresh
  refreshState();

  // Chat send
  function sendChat() {
    const chatInput = document.getElementById('chat-input');
    const msg = chatInput?.value.trim();
    if (!msg) return;
    chatInput.value = '';

    if (activeChatTab === 'private') {
      const target = whisperTargetId || document.getElementById('whisper-target')?.value;
      if (!target) {
        toast('请先选择私聊对象', 'warning');
        return;
      }
      wsSendCommand(roomId, 'whisper', { to_user_id: target, message: msg }).catch(err => {
        toast('发送失败', 'error');
      });
    } else if (activeChatTab === 'evil_team') {
      wsSendCommand(roomId, 'evil_team_chat', { message: msg }).catch(err => {
        toast('发送失败', 'error');
      });
    } else {
      wsSendCommand(roomId, 'public_chat', { message: msg }).catch(err => {
        toast('发送失败', 'error');
      });
    }
  }

  document.getElementById('chat-send-btn')?.addEventListener('click', sendChat);
  document.getElementById('chat-input')?.addEventListener('keydown', (e) => {
    if (e.key === 'Enter') sendChat();
  });

  // Chat tab switching
  document.getElementById('chat-tabs')?.addEventListener('click', (e) => {
    const tab = e.target.closest('[data-tab]');
    if (!tab) return;
    activeChatTab = tab.dataset.tab;
    // Re-render the chat panel
    const chatPanel = document.getElementById('chat-panel');
    if (chatPanel) {
      const temp = document.createElement('div');
      temp.innerHTML = renderChatPanelInner();
      chatPanel.innerHTML = temp.innerHTML;
      // Re-bind chat events
      bindChatEvents(roomId);
    }
  });

  // Whisper target change
  document.getElementById('whisper-target')?.addEventListener('change', (e) => {
    whisperTargetId = e.target.value;
  });

  // Delegated event listeners for dynamic content
  document.getElementById('app').addEventListener('click', handleGameClick);

  // Subscribe to state changes
  stateUnsub = store.subscribe('gameState', () => {
    updateGameUI();
  });

  // Subscribe to message changes
  msgUnsub = store.subscribe('messages', () => {
    updateNotifications();
    updateChatMessages();
  });

  // Timer update
  startTimer();
}

function renderChatPanelInner() {
  const myPlayer = store.getMyPlayer();
  const isEvil = myPlayer && myPlayer.team === 'evil';
  const chatPlaceholders = { public: '发送公共消息...', private: '发送私聊...', evil_team: '发送坏人群聊...' };

  return `
    <div class="chat-tabs" id="chat-tabs">
      <button class="chat-tab ${activeChatTab === 'public' ? 'active' : ''}" data-tab="public">💬 公共</button>
      <button class="chat-tab ${activeChatTab === 'private' ? 'active' : ''}" data-tab="private">🔒 私聊</button>
      ${isEvil ? `<button class="chat-tab evil-tab ${activeChatTab === 'evil_team' ? 'active' : ''}" data-tab="evil_team">😈 坏人</button>` : ''}
    </div>
    ${activeChatTab === 'private' ? `
    <div class="chat-target-bar">
      <select id="whisper-target" class="input whisper-select">
        <option value="">-- 选择玩家 --</option>
        ${renderPlayerOptions()}
      </select>
    </div>` : ''}
    <div class="chat-messages" id="chat-messages">
      ${renderChatMessages()}
    </div>
    <div class="chat-input-area">
      <input class="input" type="text" id="chat-input"
             placeholder="${chatPlaceholders[activeChatTab] || '发送消息...'}"
             maxlength="500">
      <button class="btn chat-send-btn" id="chat-send-btn">发送</button>
    </div>
  `;
}

function bindChatEvents(roomId) {
  function sendChat() {
    const chatInput = document.getElementById('chat-input');
    const msg = chatInput?.value.trim();
    if (!msg) return;
    chatInput.value = '';

    if (activeChatTab === 'private') {
      const target = whisperTargetId || document.getElementById('whisper-target')?.value;
      if (!target) {
        toast('请先选择私聊对象', 'warning');
        return;
      }
      wsSendCommand(roomId, 'whisper', { to_user_id: target, message: msg }).catch(() => toast('发送失败', 'error'));
    } else if (activeChatTab === 'evil_team') {
      wsSendCommand(roomId, 'evil_team_chat', { message: msg }).catch(() => toast('发送失败', 'error'));
    } else {
      wsSendCommand(roomId, 'public_chat', { message: msg }).catch(() => toast('发送失败', 'error'));
    }
  }

  document.getElementById('chat-send-btn')?.addEventListener('click', sendChat);
  document.getElementById('chat-input')?.addEventListener('keydown', (e) => { if (e.key === 'Enter') sendChat(); });
  document.getElementById('chat-tabs')?.addEventListener('click', (e) => {
    const tab = e.target.closest('[data-tab]');
    if (!tab) return;
    activeChatTab = tab.dataset.tab;
    const chatPanel = document.getElementById('chat-panel');
    if (chatPanel) {
      chatPanel.innerHTML = renderChatPanelInner();
      bindChatEvents(roomId);
    }
  });
  document.getElementById('whisper-target')?.addEventListener('change', (e) => { whisperTargetId = e.target.value; });
}

function handleGameClick(e) {
  const roomId = store.get('roomId');

  // Night action target selection
  const targetBtn = e.target.closest('[data-target-id]');
  if (targetBtn) {
    const targetId = targetBtn.dataset.targetId;
    const roleId = store.getMyPlayer()?.role;
    const needsTwoTargets = ['fortune_teller', 'washerwoman', 'librarian', 'investigator'].includes(roleId);
    let selected = [...(store.get('selectedTargets') || [])];

    if (selected.includes(targetId)) {
      selected = selected.filter(id => id !== targetId);
    } else {
      if (needsTwoTargets && selected.length >= 2) {
        selected.shift(); // Remove oldest
      } else if (!needsTwoTargets && selected.length >= 1) {
        selected = [];
      }
      selected.push(targetId);
    }

    store.set('selectedTargets', selected);
    updateActionPanel();
    return;
  }

  // Submit night ability
  if (e.target.id === 'ability-submit-btn' || e.target.closest('#ability-submit-btn')) {
    const targets = store.get('selectedTargets') || [];
    if (targets.length === 0) return;

    const data = targets.length === 1
      ? { target: targets[0] }
      : { targets: JSON.stringify(targets) };

    wsSendCommand(roomId, 'ability.use', data).then(() => {
      store.update({ nightActionPending: false, selectedTargets: [] });
      toast('能力已使用', 'success');
      updateActionPanel();
    }).catch(err => {
      toast(err.message || '使用能力失败', 'error');
    });
    return;
  }

  // Nominate
  const nominateBtn = e.target.closest('[data-nominate-id]');
  if (nominateBtn) {
    const nomineeId = nominateBtn.dataset.nominateId;
    wsSendCommand(roomId, 'nominate', { nominee: nomineeId }).then(() => {
      toast('提名已发起', 'success');
    }).catch(err => {
      toast(err.message || '提名失败', 'error');
    });
    return;
  }

  // Slayer shot
  const slayerBtn = e.target.closest('[data-slayer-target]');
  if (slayerBtn) {
    const targetId = slayerBtn.dataset.slayerTarget;
    if (confirm('确定要对该玩家使用杀手能力吗？')) {
      wsSendCommand(roomId, 'slayer_shot', { target: targetId }).then(() => {
        toast('🔫 开枪！', 'info');
      }).catch(err => {
        toast(err.message || '射击失败', 'error');
      });
    }
    return;
  }

  // Vote yes
  if (e.target.id === 'vote-yes-btn' || e.target.closest('#vote-yes-btn')) {
    wsSendCommand(roomId, 'vote', { vote: 'yes' }).then(() => {
      toast('已投赞成票', 'success');
    }).catch(err => {
      toast(err.message || '投票失败', 'error');
    });
    return;
  }

  // Vote no
  if (e.target.id === 'vote-no-btn' || e.target.closest('#vote-no-btn')) {
    wsSendCommand(roomId, 'vote', { vote: 'no' }).then(() => {
      toast('已投反对票', 'info');
    }).catch(err => {
      toast(err.message || '投票失败', 'error');
    });
    return;
  }

  // End defense
  if (e.target.id === 'end-defense-btn' || e.target.closest('#end-defense-btn')) {
    wsSendCommand(roomId, 'end_defense', {}).catch(err => {
      toast(err.message || '操作失败', 'error');
    });
    return;
  }

  // Game over back button
  if (e.target.id === 'game-over-back-btn' || e.target.closest('#game-over-back-btn')) {
    store.update({
      roomId: '',
      gameState: null,
      messages: [],
      nightActionPending: false,
      nightResult: '',
      selectedTargets: [],
    });
    navigate('home');
    return;
  }

  // Click on player token (for info or selection in nomination)
  const playerToken = e.target.closest('.player-token');
  if (playerToken) {
    const playerId = playerToken.dataset.playerId;
    // Could show player info popup here
  }
}

// ---- UI Update Functions ----

function updateGameUI() {
  const gamePage = document.getElementById('game-page');
  if (!gamePage) return;

  const gs = store.get('gameState');
  if (!gs) return;

  // Update phase class
  gamePage.className = `game-page phase-${gs.phase || 'lobby'}`;

  // Update phase banner
  const banner = gamePage.querySelector('.phase-banner');
  if (banner) {
    const temp = document.createElement('div');
    temp.innerHTML = renderPhaseBanner();
    banner.replaceWith(temp.firstElementChild);
  }

  // Update town square
  const tsContainer = gamePage.querySelector('.townsquare-container');
  if (tsContainer) {
    tsContainer.innerHTML = renderTownSquare();
  }

  // Update action panel
  updateActionPanel();

  // Check for game over
  if (gs.phase === 'ended') {
    const existing = document.getElementById('game-over-overlay');
    if (!existing) {
      const overlay = document.createElement('div');
      overlay.innerHTML = renderGameOverOverlay();
      if (overlay.firstElementChild) {
        document.body.appendChild(overlay.firstElementChild);
      }
    }
  }
}

function updateActionPanel() {
  const actionArea = document.getElementById('action-area');
  if (!actionArea) {
    // Fallback: try old side-panel approach
    const sidePanel = document.querySelector('.side-panel');
    if (!sidePanel) return;
    const existingAction = sidePanel.querySelector('.action-panel, .nomination-panel');
    const newHtml = renderActionPanel();
    if (existingAction) {
      const temp = document.createElement('div');
      temp.innerHTML = newHtml;
      if (temp.firstElementChild) existingAction.replaceWith(temp.firstElementChild);
    }
    return;
  }
  actionArea.innerHTML = renderActionPanel();
}

function updateNotifications() {
  const container = document.getElementById('notif-list');
  if (!container) return;
  container.innerHTML = renderNotifications();
  container.scrollTop = container.scrollHeight;
}

function updateChatMessages() {
  const container = document.getElementById('chat-messages');
  if (!container) return;
  container.innerHTML = renderChatMessages();
  container.scrollTop = container.scrollHeight;
}

function startTimer() {
  clearInterval(timerInterval);
  timerInterval = setInterval(() => {
    const gs = store.get('gameState');
    if (!gs || !gs.phase_ends_at) return;

    const timerEl = document.getElementById('phase-timer');
    if (!timerEl) return;

    const remaining = Math.max(0, Math.floor((gs.phase_ends_at - Date.now()) / 1000));
    if (remaining > 0) {
      const min = Math.floor(remaining / 60);
      const sec = remaining % 60;
      timerEl.textContent = `${min}:${sec.toString().padStart(2, '0')}`;
    } else {
      timerEl.textContent = '';
    }
  }, 1000);
}

export function unmountGame() {
  if (stateUnsub) { stateUnsub(); stateUnsub = null; }
  if (msgUnsub) { msgUnsub(); msgUnsub = null; }
  clearInterval(timerInterval);

  document.getElementById('app')?.removeEventListener('click', handleGameClick);

  const overlay = document.getElementById('game-over-overlay');
  if (overlay) overlay.remove();
}

// escapeHtml imported from utils.js
