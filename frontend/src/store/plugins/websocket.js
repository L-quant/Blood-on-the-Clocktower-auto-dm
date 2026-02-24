/**
 * WebSocket plugin for Vuex
 * Bridges WebSocket events to store mutations.
 *
 * Backend protocol:
 *   - Connect: ws://host/ws?token=<jwt>
 *   - Subscribe: { type: "subscribe", payload: { room_id, last_seq } }
 *   - Command:   { type: "command",   payload: { command_id, room_id, type, data } }
 *   - Ping:      { type: "ping" }
 *
 * Backend events arrive as:
 *   { type: "event", payload: { event_type, data, seq, server_ts } }
 */

import apiService from "../../services/ApiService";

const WS_URL = process.env.VUE_APP_WS_URL || 'ws://localhost:8080/ws';

class WebSocketManager {
  constructor(store) {
    this._store = store;
    this._socket = null;
    this._pingTimer = null;
    this._reconnectTimer = null;
    this._reconnectDelay = 1000;
    this._maxReconnectDelay = 30000;
    this._lastSeq = 0;
    this._pingInterval = 25000; // must be < server's 30s read deadline
    this._roomId = null;
  }

  connect(roomId) {
    this.disconnect();

    const token = apiService.token;
    if (!token) {
      return;
    }

    this._roomId = roomId;
    const wsUrl = `${WS_URL}?token=${encodeURIComponent(token)}`;

    this._socket = new WebSocket(wsUrl);
    this._reconnectDelay = 1000;

    this._socket.onopen = () => {
      this._store.commit('setConnected', true);
      this._store.commit('setReconnecting', false);

      // Subscribe to room events
      this._send('subscribe', {
        room_id: roomId,
        last_seq: this._lastSeq
      });

      this._startPing();
    };

    this._socket.onmessage = (event) => {
      this._handleMessage(event.data);
    };

    this._socket.onclose = (event) => {
      this._socket = null;
      this._stopPing();
      this._store.commit('setConnected', false);

      // Reconnect unless intentionally closed (code 1000)
      if (event.code !== 1000 && this._roomId) {
        this._store.commit('setReconnecting', true);
        this._scheduleReconnect();
      }
    };

    this._socket.onerror = () => {
      // Error will be followed by onclose
    };
  }

  disconnect() {
    this._stopPing();
    clearTimeout(this._reconnectTimer);
    this._roomId = null;
    this._store.commit('setReconnecting', false);

    if (this._socket) {
      this._socket.close(1000);
      this._socket = null;
    }

    this._store.commit('setConnected', false);
  }

  /**
   * Send a game command via WebSocket.
   * Backend expects: { type: "command", payload: { command_id, room_id, type, data } }
   */
  send(command, data) {
    if (this._socket && this._socket.readyState === WebSocket.OPEN) {
      this._socket.send(JSON.stringify({
        type: 'command',
        request_id: Math.random().toString(36).substr(2),
        payload: {
          command_id: Math.random().toString(36).substr(2),
          room_id: this._roomId,
          type: command,
          data: data
        }
      }));
    }
  }

  _send(type, payload) {
    if (this._socket && this._socket.readyState === WebSocket.OPEN) {
      this._socket.send(JSON.stringify({
        type: type,
        request_id: Math.random().toString(36).substr(2),
        payload: payload
      }));
    }
  }

  _handleMessage(raw) {
    let parsed;
    try {
      parsed = JSON.parse(raw);
    } catch (e) {
      return;
    }

    const type = parsed.type;

    switch (type) {
      case 'subscribed':
        this._store.commit('setReconnecting', false);
        // After subscribing, send a "join" command so backend creates the player
        // (will be rejected if already joined — that's fine)
        this.send('join', { name: this._store.state.playerId ? 'Player' : 'Player' });
        // Fetch full room state to ensure we have accurate player list + settings
        // Use a delay to let our join event process first
        setTimeout(() => this._fetchRoomState(), 500);
        break;

      case 'pong':
        this._handlePong(parsed.payload);
        break;

      case 'error':
        // Server error received
        break;

      case 'command_result':
        // Could handle success/failure of commands
        break;

      case 'event': {
        // Backend sends: { type: "event", payload: ProjectedEvent }
        // ProjectedEvent: { event_type, data, seq, server_ts, room_id }
        let pe;
        if (typeof parsed.payload === 'string') {
          try { pe = JSON.parse(parsed.payload); } catch (e) { return; }
        } else {
          pe = parsed.payload;
        }
        if (pe && pe.seq) {
          this._lastSeq = Math.max(this._lastSeq, pe.seq);
        }
        this._processGameEvent(pe);
        break;
      }

      default:
        break;
    }
  }

  /**
   * Process a ProjectedEvent from the backend.
   * ProjectedEvent format: { event_type: string, data: object|string, seq: number, server_ts: number }
   */
  _processGameEvent(pe) {
    if (!pe || !pe.event_type) return;

    const eventType = pe.event_type;
    // data may be a JSON string or already parsed object
    let eventData = pe.data;
    if (typeof eventData === 'string') {
      try { eventData = JSON.parse(eventData); } catch (e) { eventData = {}; }
    }
    if (!eventData) eventData = {};

    const store = this._store;

    switch (eventType) {
      // ── Player & Seat events ──
      case 'player.joined': {
        const seatNum = parseInt(eventData.seat_number, 10) || 0;
        const actorId = pe.actor_user_id || eventData.user_id || '';
        if (seatNum > 0 && actorId) {
          store.commit('players/seatPlayer', {
            id: actorId,
            seatIndex: seatNum
          });
          if (actorId === apiService.userId) {
            store.commit('setSeatIndex', seatNum);
          }
        }
        break;
      }

      case 'player.left': {
        const leftActorId = pe.actor_user_id || '';
        if (leftActorId) {
          // Remove by id to be precise
          const leftPlayer = store.state.players.players.find(p => p.id === leftActorId);
          if (leftPlayer) {
            store.commit('players/removePlayer', leftPlayer.seatIndex);
          }
          if (leftActorId === apiService.userId) {
            store.commit('setSeatIndex', -1);
          }
        } else {
          store.commit('players/removePlayer',
            parseInt(eventData.seat_number, 10) || 0
          );
        }
        break;
      }

      case 'seat.claimed': {
        const seatNum = parseInt(eventData.seat_number, 10) || 0;
        const actorId = pe.actor_user_id || '';
        if (actorId) {
          // Remove player from old seat first (if they had one)
          const oldEntry = store.state.players.players.find(p => p.id === actorId);
          if (oldEntry && oldEntry.seatIndex !== seatNum) {
            store.commit('players/removePlayer', oldEntry.seatIndex);
          }
          store.commit('players/seatPlayer', {
            id: actorId,
            seatIndex: seatNum
          });
          if (actorId === apiService.userId) {
            store.commit('setSeatIndex', seatNum);
          }
        }
        break;
      }

      // ── Role assignment (only sent for own role via projection) ──
      case 'role.assigned': {
        const roleId = eventData.role || '';
        // Look up role metadata from roles.json for display name & ability
        const roleData = store.getters.rolesByKey.get(roleId);
        store.commit('players/setMyRole', {
          roleId: roleId,
          roleName: roleData ? roleData.name : roleId,
          team: eventData.team || '',
          ability: roleData ? roleData.ability : ''
        });
        break;
      }

      case 'bluffs.assigned': {
        let bluffs = eventData.bluffs;
        if (typeof bluffs === 'string') {
          try { bluffs = JSON.parse(bluffs); } catch (e) { bluffs = []; }
        }
        store.commit('players/setBluffs', bluffs || []);
        break;
      }

      // ── Room settings ──
      case 'room.settings.changed':
        if (eventData.edition) {
          store.commit('setEdition', eventData.edition);
        }
        if (eventData.max_players) {
          store.commit('setSeatCount', parseInt(eventData.max_players, 10) || 8);
        }
        break;

      // ── Game lifecycle ──
      case 'game.started':
        store.commit('game/setPhase', 'first_night');
        store.commit('ui/setScreen', 'game');
        break;

      // ── Phase transitions (backend sends separate event types per phase) ──
      case 'phase.first_night':
        store.commit('game/setPhase', 'first_night');
        store.commit('game/setDayCount', 0);
        store.commit('timeline/addEvent', {
          type: 'phase_change',
          dayCount: 0,
          data: { phase: 'first_night' }
        });
        break;

      case 'phase.night':
        store.commit('game/setPhase', 'night');
        store.commit('players/resetNominationFlags');
        store.commit('timeline/addEvent', {
          type: 'phase_change',
          dayCount: store.state.game.dayCount,
          data: { phase: 'night' }
        });
        break;

      case 'phase.day': {
        const newDayCount = store.state.game.dayCount + 1;
        store.commit('game/setPhase', 'day');
        store.commit('game/setDayCount', newDayCount);
        store.commit('players/resetNominationFlags');
        store.commit('vote/endVote');
        store.commit('timeline/addEvent', {
          type: 'phase_change',
          dayCount: newDayCount,
          data: { phase: 'day' }
        });
        break;
      }

      case 'phase.nomination':
        store.commit('game/setPhase', 'nomination');
        store.commit('timeline/addEvent', {
          type: 'phase_change',
          dayCount: store.state.game.dayCount,
          data: { phase: 'nomination' }
        });
        break;

      // ── Nomination & Voting ──
      case 'nomination.created': {
        const nominatorSeat = parseInt(eventData.nominator_seat, 10) || 0;
        const nomineeSeat = parseInt(eventData.nominee_seat, 10) || 0;
        store.commit('vote/startNomination', {
          nominatorSeat,
          nomineeSeat,
          requiredMajority: 0
        });
        store.commit('players/updatePlayer', {
          seatIndex: nominatorSeat,
          property: 'hasNominatedToday',
          value: true
        });
        store.commit('players/updatePlayer', {
          seatIndex: nomineeSeat,
          property: 'isNominatedToday',
          value: true
        });
        break;
      }

      case 'defense.ended':
        // Voting phase now starts; currently no frontend action needed
        break;

      case 'vote.cast': {
        const voterSeat = parseInt(eventData.voter_seat, 10) || 0;
        const voteValue = eventData.vote === 'yes';
        store.commit('vote/castVote', {
          seatIndex: voterSeat,
          vote: voteValue
        });
        store.commit('vote/setCurrentVoter', voterSeat);
        break;
      }

      case 'nomination.resolved': {
        const result = eventData.result === 'executed' ? 'executed' : 'safe';
        store.commit('vote/setResult', result);
        store.commit('timeline/addEvent', {
          type: 'vote_result',
          dayCount: store.state.game.dayCount,
          data: {
            nomineeSeat: store.state.vote.nominee ? store.state.vote.nominee.seatIndex : -1,
            yesCount: parseInt(eventData.votes_for, 10) || store.state.vote.currentYesCount,
            result
          }
        });
        break;
      }

      case 'execution.resolved':
        // Already handled by nomination.resolved + player.died
        break;

      // ── Night actions ──
      case 'night.action.queued':
        // Check if this is my action
        if (eventData.user_id === apiService.userId) {
          const nightRoleId = eventData.role_id || '';
          const nightRoleData = store.getters.rolesByKey.get(nightRoleId);
          const roleName = nightRoleData ? nightRoleData.name : nightRoleId;
          const abilityText = nightRoleData ? nightRoleData.ability : '';

          // Determine action type from role data
          let actionType = 'passive';
          if (nightRoleData) {
            // Roles that select targets at night
            const selectOneRoles = [
              'poisoner', 'monk', 'fortuneteller', 'imp', 'spy',
              'washerwoman', 'librarian', 'investigator', 'empath',
              'undertaker', 'ravenkeeper', 'innkeeper', 'gambler',
              'exorcist', 'courtier', 'gossip', 'chambermaid',
              'dreamer', 'snakecharmer', 'flowergirl', 'seamstress',
              'mathematician', 'juggler', 'sage', 'oracle',
              'pukka', 'shabaloth', 'po', 'fanggu',
              'vigormortis', 'nodashii', 'vortox',
              'zombuul', 'witch', 'cerenovus', 'pithag',
              'assassin', 'godfather', 'devilsadvocate',
              'lycanthrope', 'lunatic', 'grandmother',
              'bountyhunter', 'pixie', 'choirboy', 'nightwatchman',
              'balloonist'
            ];
            const selectTwoRoles = ['fortuneteller'];
            if (selectTwoRoles.includes(nightRoleId)) {
              actionType = 'select_two';
            } else if (selectOneRoles.includes(nightRoleId)) {
              actionType = 'select_one';
            } else {
              actionType = 'passive';
            }
          }

          store.commit('night/openPanel', {
            roleId: nightRoleId,
            roleName: roleName,
            abilityText: abilityText,
            actionType: actionType
          });

          // Populate available targets (all other players)
          const allPlayers = store.state.players.players;
          const targets = allPlayers
            .filter(p => !p.isMe && p.isAlive)
            .map(p => ({ seatIndex: p.seatIndex, id: p.id }));
          store.commit('night/setTargets', targets);
        }
        break;

      case 'night.action.completed':
        if (eventData.user_id === apiService.userId) {
          store.commit('night/setResult', eventData.result || '');
        }
        break;

      // ── Deaths ──
      case 'player.died': {
        // Backend sends user_id, we need to find seat
        const diedUserId = eventData.user_id || '';
        const players = store.state.players.players;
        const deadPlayer = players.find(p => p.id === diedUserId);
        if (deadPlayer) {
          store.commit('players/killPlayer', deadPlayer.seatIndex);
          store.commit('timeline/addEvent', {
            type: 'death',
            dayCount: store.state.game.dayCount,
            data: { seatIndex: deadPlayer.seatIndex }
          });
        }
        break;
      }

      // ── Chat ── FIX-8: Skip own messages to prevent duplication (sendChat already adds locally)
      case 'public.chat':
        if (pe.actor_user_id !== apiService.userId) {
          store.commit('chat/addPublicMessage', {
            seatIndex: parseInt(eventData.sender_seat, 10) || -1,
            text: eventData.message || '',
            isSystem: false
          });
        }
        break;

      case 'whisper.sent': {
        // FIX-8: Skip own whispers (already added locally by sendChat)
        if (pe.actor_user_id !== apiService.userId) {
          const senderSeat = parseInt(eventData.sender_seat, 10) || -1;
          store.commit('chat/addWhisperMessage', {
            targetSeat: senderSeat,
            data: {
              seatIndex: senderSeat,
              text: eventData.message || ''
            }
          });
        }
        break;
      }

      case 'evil_team.chat':
        // FIX-8: Skip own evil team messages (already added locally by sendChat)
        if (pe.actor_user_id !== apiService.userId) {
          store.commit('chat/addEvilMessage', {
            seatIndex: parseInt(eventData.sender_seat, 10) || -1,
            text: eventData.message || ''
          });
        }
        break;

      // ── Game end ──
      case 'game.ended':
        store.commit('game/setPhase', 'ended');
        store.commit('game/setWinner', eventData.winner || '');
        store.commit('game/setWinReason', eventData.reason || '');
        store.commit('ui/setScreen', 'end');
        break;

      // ── Ignored / internal events ──
      case 'red_herring.assigned':
      case 'reminder.added':
      case 'ai.decision':
      case 'slayer.shot':
      case 'poison.cleared':
        break;

      default:
        break;
    }
  }

  _fetchRoomState() {
    if (!this._roomId) return;
    apiService.getRoomState(this._roomId).then(state => {
      this._syncRoomState(state);
    }).catch(() => {
      // Ignore - state will be built from events
    });
  }

  _syncRoomState(state) {
    if (!state) return;

    if (state.players) {
      const playersList = [];
      // Backend state.players is a map of userId -> Player
      const playersMap = state.players;
      const entries = typeof playersMap === 'object' && !Array.isArray(playersMap)
        ? Object.entries(playersMap)
        : [];

      let mySeatIndex = -1;

      entries.forEach(([userId, p]) => {
        const seatIndex = p.seat_number || 0;
        const isMe = userId === apiService.userId;
        playersList.push({
          id: userId,
          name: p.name || '',
          seatIndex: seatIndex,
          isAlive: p.alive !== undefined ? p.alive : true,
          hasGhostVote: p.has_ghost_vote !== undefined ? p.has_ghost_vote : true,
          isNominatedToday: p.was_nominated || false,
          hasNominatedToday: p.has_nominated || false,
          isMe: isMe
        });
        if (isMe && seatIndex > 0) {
          mySeatIndex = seatIndex;
        }
      });

      if (playersList.length > 0) {
        this._store.commit('players/setPlayers', playersList);
      }

      // Update root seatIndex for the current user
      if (mySeatIndex > 0) {
        this._store.commit('setSeatIndex', mySeatIndex);
      }
    }

    if (state.phase) {
      this._store.commit('game/setPhase', state.phase);
      // Switch to appropriate screen based on phase
      if (state.phase === 'lobby') {
        this._store.commit('ui/setScreen', 'lobby');
      } else if (state.phase === 'ended') {
        this._store.commit('ui/setScreen', 'end');
        if (state.winner) {
          this._store.commit('game/setWinner', state.winner);
        }
      } else {
        this._store.commit('ui/setScreen', 'game');
      }
    }

    if (state.day_count !== undefined) {
      this._store.commit('game/setDayCount', state.day_count);
    }

    if (state.edition) {
      this._store.commit('setEdition', state.edition);
    }

    if (state.max_players) {
      this._store.commit('setSeatCount', state.max_players);
    }

    // Sync own role from projected state (role field is only visible for self)
    if (state.players) {
      const meData = state.players[apiService.userId];
      if (meData && meData.role && !this._store.state.players.myRole) {
        const roleData = this._store.getters.rolesByKey.get(meData.role);
        this._store.commit('players/setMyRole', {
          roleId: meData.role,
          roleName: roleData ? roleData.name : meData.role,
          team: meData.team || '',
          ability: roleData ? roleData.ability : ''
        });
      }
    }
  }

  _startPing() {
    this._stopPing();
    this._pingTimer = setInterval(() => {
      this._send('ping', { timestamp: Date.now() });
    }, this._pingInterval);
  }

  _stopPing() {
    if (this._pingTimer) {
      clearInterval(this._pingTimer);
      this._pingTimer = null;
    }
  }

  _handlePong(payload) {
    if (payload && payload.timestamp) {
      const latency = Date.now() - payload.timestamp;
      this._store.commit('setLatency', latency);
    }
  }

  _scheduleReconnect() {
    this._reconnectTimer = setTimeout(() => {
      if (!this._store.state.connected && this._roomId) {
        this.connect(this._roomId);
      }
    }, this._reconnectDelay);

    // Exponential backoff
    this._reconnectDelay = Math.min(
      this._reconnectDelay * 2,
      this._maxReconnectDelay
    );
  }
}

export default store => {
  const ws = new WebSocketManager(store);

  // Expose WebSocket manager on store for direct access
  store.$ws = ws;

  store.subscribe(({ type, payload }) => {
    switch (type) {
      case 'setRoomId':
        if (payload) {
          ws.connect(payload);
        } else {
          ws.disconnect();
        }
        break;

      case 'disconnect':
        ws.disconnect();
        break;

      // Player actions that send to server
      case 'sendCommand':
        if (payload && payload.type) {
          ws.send(payload.type, payload.data || {});
        }
        break;
    }
  });
};
