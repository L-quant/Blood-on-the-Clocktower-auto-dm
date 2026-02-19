class LiveSession {
  constructor(store) {
    // 使用环境变量或默认本地后端
    this._wss = process.env.VUE_APP_WS_URL || "ws://localhost:8080/ws";
    this._useLocalBackend = true; // 使用本地后端
    // this._wss = "wss://live.clocktower.online:8080/"; // 原始 townsquare 服务器
    // this._useLocalBackend = false;
    this._socket = null;
    this._isSpectator = true;
    this._gamestate = [];
    this._store = store;
    this._pingInterval = 30 * 1000; // 30 seconds between pings
    this._pingTimer = null;
    this._reconnectTimer = null;
    this._players = {}; // map of players connected to a session
    this._pings = {}; // map of player IDs to ping
    this._lastSeq = 0; // 用于事件溯源的序列号
    // reconnect to previous session
    if (this._store.state.session.sessionId) {
      this.connect(this._store.state.session.sessionId);
    }
  }

  /**
   * Open a new session for the passed channel.
   * @param channel
   * @private
   */
  _open(channel) {
    this.disconnect();
    
    let wsUrl;
    if (this._useLocalBackend) {
      // 连接到本地后端
      wsUrl = `${this._wss}?player_id=${this._store.state.session.playerId}&room_id=${channel}`;
    } else {
      // 原始 townsquare 连接方式
      wsUrl = this._wss + channel + "/" + 
        (this._isSpectator ? this._store.state.session.playerId : "host");
    }
    
    this._socket = new WebSocket(wsUrl);
    this._socket.addEventListener("message", this._handleMessage.bind(this));
    this._socket.onopen = this._onOpen.bind(this);
    this._socket.onclose = err => {
      this._socket = null;
      clearInterval(this._pingTimer);
      this._pingTimer = null;
      if (err.code !== 1000) {
        // connection interrupted, reconnect after 3 seconds
        this._store.commit("session/setReconnecting", true);
        this._reconnectTimer = setTimeout(
          () => this.connect(channel),
          3 * 1000
        );
      } else {
        this._store.commit("session/setSessionId", "");
        if (err.reason) alert(err.reason);
      }
    };
  }

  /**
   * Send a message through the socket.
   * @param command
   * @param params
   * @private
   */
  _send(command, params) {
    if (this._socket && this._socket.readyState === 1) {
      if (this._useLocalBackend) {
        // 本地后端格式
        this._socket.send(JSON.stringify({
          type: "command",
          request_id: Math.random().toString(36).substr(2),
          payload: {
            command_id: Math.random().toString(36).substr(2),
            type: command,
            data: params
          }
        }));
      } else {
        // 原始 townsquare 格式
        this._socket.send(JSON.stringify([command, params]));
      }
    }
  }

  /**
   * Send a message directly to a single playerId, if provided.
   * Otherwise broadcast it.
   * @param playerId player ID or "host", optional
   * @param command
   * @param params
   * @private
   */
  _sendDirect(playerId, command, params) {
    if (this._useLocalBackend) {
      // 本地后端：所有消息都走统一格式
      this._send(command, { target: playerId, ...params });
    } else {
      // 原始 townsquare 格式
      if (playerId) {
        this._send("direct", { [playerId]: [command, params] });
      } else {
        this._send(command, params);
      }
    }
  }

  /**
   * Open event handler for socket.
   * @private
   */
  _onOpen() {
    if (this._useLocalBackend) {
      // 本地后端：发送订阅消息
      if (this._socket && this._socket.readyState === 1) {
        this._socket.send(JSON.stringify({
          type: "subscribe",
          request_id: Math.random().toString(36).substr(2),
          payload: {
            room_id: this._store.state.session.sessionId,
            last_seq: this._lastSeq
          }
        }));
      }
      this._ping();
      return;
    }
    
    // 原始逻辑
    if (this._isSpectator) {
      this._sendDirect(
        "host",
        "getGamestate",
        this._store.state.session.playerId
      );
    } else {
      this.sendGamestate();
    }
    this._ping();
  }

  /**
   * Send a ping message with player ID and ST flag.
   * @private
   */
  _ping() {
    this._handlePing();
    this._send("ping", [
      this._isSpectator
        ? this._store.state.session.playerId
        : Object.keys(this._players).length,
      "latency"
    ]);
    clearTimeout(this._pingTimer);
    this._pingTimer = setTimeout(this._ping.bind(this), this._pingInterval);
  }

  /**
   * Handle an incoming socket message.
   * @param data
   * @private
   */
  _handleMessage({ data }) {
    let command, params;
    try {
      const parsed = JSON.parse(data);
      
      // 处理本地后端消息格式
      if (this._useLocalBackend) {
        // 本地后端消息格式: { type, seq, payload, ... }
        if (parsed.type) {
          command = parsed.type;
          params = parsed.payload || parsed.data || parsed;
          
          // 更新序列号用于断线重连
          if (parsed.seq) {
            this._lastSeq = Math.max(this._lastSeq, parsed.seq);
          }
          
          // 处理后端事件类型映射
          this._handleBackendEvent(command, params);
          return;
        }
      }
      
      // 原始 townsquare 格式: [command, params]
      [command, params] = parsed;
    } catch (err) {
      console.log("unsupported socket message", data);
      return;
    }
    this._processCommand(command, params);
  }

  /**
   * 处理本地后端事件
   * @param eventType
   * @param payload
   * @private
   */
  _handleBackendEvent(eventType, payload) {
    console.log("[Backend Event]", eventType, payload);
    
    switch (eventType) {
      case "subscribed":
        // 订阅成功
        console.log("Subscribed to room:", payload.room_id);
        this._store.commit("session/setReconnecting", false);
        break;
        
      case "event":
      case "events": {
        // 处理游戏事件
        const events = Array.isArray(payload) ? payload : [payload];
        events.forEach(event => this._processBackendGameEvent(event));
        break;
      }
        
      case "pong":
        this._handlePing(payload);
        break;
        
      case "error":
        console.error("Backend error:", payload);
        break;
        
      case "room_state":
        // 房间状态同步
        this._syncRoomState(payload);
        break;
        
      default:
        // 尝试作为游戏事件处理
        this._processBackendGameEvent({ type: eventType, ...payload });
    }
  }

  /**
   * 处理后端游戏事件
   * @param event
   * @private
   */
  _processBackendGameEvent(event) {
    const { type, data } = event;
    
    switch (type) {
      case "player.joined":
        this._store.commit("players/add", data.name || "");
        break;
        
      case "player.left":
        // 找到并移除玩家
        break;
        
      case "player.seated":
        // 玩家就座
        break;
        
      case "role.assigned":
        // 角色分配（只有自己能收到自己的）
        break;
        
      case "game.started":
        // 游戏开始
        break;
        
      case "phase.changed":
        // 阶段变化
        if (data.phase === "night" || data.phase === "first_night") {
          this._store.commit("toggleNight", true);
        } else {
          this._store.commit("toggleNight", false);
        }
        this._store.commit("session/setPhase", data.phase || "");
        if (data.day_count !== undefined) {
          this._store.commit("session/setDayCount", data.day_count);
        }
        break;
        
      case "nomination.started":
        this._store.commit("session/nomination", { 
          nomination: [data.nominator, data.nominee] 
        });
        break;
        
      case "vote.cast":
        // 投票
        this._store.commit("session/vote", [data.player_index, data.vote]);
        break;
        
      case "player.died": {
        // 玩家死亡
        const player = this._store.state.players.players[data.player_index];
        if (player) {
          this._store.commit("players/update", {
            player,
            property: "isDead",
            value: true
          });
        }
        break;
      }
        
      case "public.chat":
        this._store.commit("session/addChatMessage", {
          from: data.sender === "auto-dm" ? "auto-dm" : "other",
          displayName: data.sender_name || data.sender || "Unknown",
          text: data.message || data.text || "",
          timestamp: Date.now()
        });
        break;

      case "whisper.received":
        this._store.commit("session/addChatMessage", {
          from: "system",
          displayName: "Whisper from " + (data.sender_name || data.sender),
          text: data.message || data.text || "",
          timestamp: Date.now()
        });
        break;

      case "game.ended":
        this._store.commit("session/setPhase", "ended");
        this._store.commit("session/setWinner", data.winner || "");
        this._store.commit("session/setWinReason", data.reason || "");
        break;
        
      default:
        console.log("[Unknown Event]", type, data);
    }
  }

  /**
   * 同步房间状态
   * @param state
   * @private
   */
  _syncRoomState(state) {
    if (!state) return;
    
    // 同步玩家
    if (state.players) {
      const currentPlayers = this._store.state.players.players;
      
      // 调整玩家数量
      while (currentPlayers.length < state.players.length) {
        this._store.commit("players/add", "");
      }
      while (currentPlayers.length > state.players.length) {
        this._store.commit("players/remove", currentPlayers.length - 1);
      }
      
      // 更新每个玩家
      state.players.forEach((p, index) => {
        const player = currentPlayers[index];
        if (p.name && player.name !== p.name) {
          this._store.commit("players/update", {
            player,
            property: "name",
            value: p.name
          });
        }
        if (p.isDead !== undefined && player.isDead !== p.isDead) {
          this._store.commit("players/update", {
            player,
            property: "isDead",
            value: p.isDead
          });
        }
      });
    }
    
    // 同步游戏阶段
    if (state.phase) {
      this._store.commit("toggleNight", state.phase === "night");
    }
  }

  /**
   * 处理原始 townsquare 命令
   * @param command
   * @param params
   * @private
   */
  _processCommand(command, params) {
    switch (command) {
      case "getGamestate":
        this.sendGamestate(params);
        break;
      case "edition":
        this._updateEdition(params);
        break;
      case "fabled":
        this._updateFabled(params);
        break;
      case "gs":
        this._updateGamestate(params);
        break;
      case "player":
        this._updatePlayer(params);
        break;
      case "claim":
        this._updateSeat(params);
        break;
      case "ping":
        this._handlePing(params);
        break;
      case "nomination":
        if (!this._isSpectator) return;
        if (!params) {
          // create vote history record
          this._store.commit(
            "session/addHistory",
            this._store.state.players.players
          );
        }
        this._store.commit("session/nomination", { nomination: params });
        break;
      case "swap":
        if (!this._isSpectator) return;
        this._store.commit("players/swap", params);
        break;
      case "move":
        if (!this._isSpectator) return;
        this._store.commit("players/move", params);
        break;
      case "remove":
        if (!this._isSpectator) return;
        this._store.commit("players/remove", params);
        break;
      case "marked":
        if (!this._isSpectator) return;
        this._store.commit("session/setMarkedPlayer", params);
        break;
      case "isNight":
        if (!this._isSpectator) return;
        this._store.commit("toggleNight", params);
        break;
      case "isVoteHistoryAllowed":
        if (!this._isSpectator) return;
        this._store.commit("session/setVoteHistoryAllowed", params);
        this._store.commit("session/clearVoteHistory");
        break;
      case "votingSpeed":
        if (!this._isSpectator) return;
        this._store.commit("session/setVotingSpeed", params);
        break;
      case "clearVoteHistory":
        if (!this._isSpectator) return;
        this._store.commit("session/clearVoteHistory");
        break;
      case "isVoteInProgress":
        if (!this._isSpectator) return;
        this._store.commit("session/setVoteInProgress", params);
        break;
      case "vote":
        this._handleVote(params);
        break;
      case "lock":
        this._handleLock(params);
        break;
      case "bye":
        this._handleBye(params);
        break;
      case "pronouns":
        this._updatePlayerPronouns(params);
        break;
    }
  }

  /**
   * Connect to a new live session, either as host or spectator.
   * Set a unique playerId if there isn't one yet.
   * @param channel
   */
  connect(channel) {
    if (!this._store.state.session.playerId) {
      this._store.commit(
        "session/setPlayerId",
        Math.random()
          .toString(36)
          .substr(2)
      );
    }
    this._pings = {};
    this._store.commit("session/setPlayerCount", 0);
    this._store.commit("session/setPing", 0);
    this._isSpectator = this._store.state.session.isSpectator;
    this._open(channel);
  }

  /**
   * Close the current session, if any.
   */
  disconnect() {
    this._pings = {};
    this._store.commit("session/setPlayerCount", 0);
    this._store.commit("session/setPing", 0);
    this._store.commit("session/setReconnecting", false);
    clearTimeout(this._reconnectTimer);
    if (this._socket) {
      if (this._isSpectator) {
        this._sendDirect("host", "bye", this._store.state.session.playerId);
      }
      this._socket.close(1000);
      this._socket = null;
    }
  }

  /**
   * Publish the current gamestate.
   * Optional param to reduce traffic. (send only player data)
   * @param playerId
   * @param isLightweight
   */
  sendGamestate(playerId = "", isLightweight = false) {
    if (this._isSpectator) return;
    this._gamestate = this._store.state.players.players.map(player => ({
      name: player.name,
      id: player.id,
      isDead: player.isDead,
      isVoteless: player.isVoteless,
      pronouns: player.pronouns,
      ...(player.role && player.role.team === "traveler"
        ? { roleId: player.role.id }
        : {})
    }));
    if (isLightweight) {
      this._sendDirect(playerId, "gs", {
        gamestate: this._gamestate,
        isLightweight
      });
    } else {
      const { session, grimoire } = this._store.state;
      const { fabled } = this._store.state.players;
      this.sendEdition(playerId);
      this._sendDirect(playerId, "gs", {
        gamestate: this._gamestate,
        isNight: grimoire.isNight,
        isVoteHistoryAllowed: session.isVoteHistoryAllowed,
        nomination: session.nomination,
        votingSpeed: session.votingSpeed,
        lockedVote: session.lockedVote,
        isVoteInProgress: session.isVoteInProgress,
        markedPlayer: session.markedPlayer,
        fabled: fabled.map(f => (f.isCustom ? f : { id: f.id })),
        ...(session.nomination ? { votes: session.votes } : {})
      });
    }
  }

  /**
   * Update the gamestate based on incoming data.
   * @param data
   * @private
   */
  _updateGamestate(data) {
    if (!this._isSpectator) return;
    const {
      gamestate,
      isLightweight,
      isNight,
      isVoteHistoryAllowed,
      nomination,
      votingSpeed,
      votes,
      lockedVote,
      isVoteInProgress,
      markedPlayer,
      fabled
    } = data;
    const players = this._store.state.players.players;
    // adjust number of players
    if (players.length < gamestate.length) {
      for (let x = players.length; x < gamestate.length; x++) {
        this._store.commit("players/add", gamestate[x].name);
      }
    } else if (players.length > gamestate.length) {
      for (let x = players.length; x > gamestate.length; x--) {
        this._store.commit("players/remove", x - 1);
      }
    }
    // update status for each player
    gamestate.forEach((state, x) => {
      const player = players[x];
      const { roleId } = state;
      // update relevant properties
      ["name", "id", "isDead", "isVoteless", "pronouns"].forEach(property => {
        const value = state[property];
        if (player[property] !== value) {
          this._store.commit("players/update", { player, property, value });
        }
      });
      // roles are special, because of travelers
      if (roleId && player.role.id !== roleId) {
        const role =
          this._store.state.roles.get(roleId) ||
          this._store.getters.rolesJSONbyId.get(roleId);
        if (role) {
          this._store.commit("players/update", {
            player,
            property: "role",
            value: role
          });
        }
      } else if (!roleId && player.role.team === "traveler") {
        this._store.commit("players/update", {
          player,
          property: "role",
          value: {}
        });
      }
    });
    if (!isLightweight) {
      this._store.commit("toggleNight", !!isNight);
      this._store.commit("session/setVoteHistoryAllowed", isVoteHistoryAllowed);
      this._store.commit("session/nomination", {
        nomination,
        votes,
        votingSpeed,
        lockedVote,
        isVoteInProgress
      });
      this._store.commit("session/setMarkedPlayer", markedPlayer);
      this._store.commit("players/setFabled", {
        fabled: fabled.map(f => this._store.state.fabled.get(f.id) || f)
      });
    }
  }

  /**
   * Publish an edition update. ST only
   * @param playerId
   */
  sendEdition(playerId = "") {
    if (this._isSpectator) return;
    const { edition } = this._store.state;
    let roles;
    if (!edition.isOfficial) {
      roles = this._store.getters.customRolesStripped;
    }
    this._sendDirect(playerId, "edition", {
      edition: edition.isOfficial ? { id: edition.id } : edition,
      ...(roles ? { roles } : {})
    });
  }

  /**
   * Update edition and roles for custom editions.
   * @param edition
   * @param roles
   * @private
   */
  _updateEdition({ edition, roles }) {
    if (!this._isSpectator) return;
    this._store.commit("setEdition", edition);
    if (roles) {
      this._store.commit("setCustomRoles", roles);
      if (this._store.state.roles.size !== roles.length) {
        const missing = [];
        roles.forEach(({ id }) => {
          if (!this._store.state.roles.get(id)) {
            missing.push(id);
          }
        });
        alert(
          `This session contains custom characters that can't be found. ` +
            `Please load them before joining! ` +
            `Missing roles: ${missing.join(", ")}`
        );
        this.disconnect();
        this._store.commit("toggleModal", "edition");
      }
    }
  }

  /**
   * Publish a fabled update. ST only
   */
  sendFabled() {
    if (this._isSpectator) return;
    const { fabled } = this._store.state.players;
    this._send(
      "fabled",
      fabled.map(f => (f.isCustom ? f : { id: f.id }))
    );
  }

  /**
   * Update fabled roles.
   * @param fabled
   * @private
   */
  _updateFabled(fabled) {
    if (!this._isSpectator) return;
    this._store.commit("players/setFabled", {
      fabled: fabled.map(f => this._store.state.fabled.get(f.id) || f)
    });
  }

  /**
   * Publish a player update.
   * @param player
   * @param property
   * @param value
   */
  sendPlayer({ player, property, value }) {
    if (this._isSpectator || property === "reminders") return;
    const index = this._store.state.players.players.indexOf(player);
    if (property === "role") {
      if (value.team && value.team === "traveler") {
        // update local gamestate to remember this player as a traveler
        this._gamestate[index].roleId = value.id;
        this._send("player", {
          index,
          property,
          value: value.id
        });
      } else if (this._gamestate[index].roleId) {
        // player was previously a traveler
        delete this._gamestate[index].roleId;
        this._send("player", { index, property, value: "" });
      }
    } else {
      this._send("player", { index, property, value });
    }
  }

  /**
   * Update a player based on incoming data. Player only.
   * @param index
   * @param property
   * @param value
   * @private
   */
  _updatePlayer({ index, property, value }) {
    if (!this._isSpectator) return;
    const player = this._store.state.players.players[index];
    if (!player) return;
    // special case where a player stops being a traveler
    if (property === "role") {
      if (!value && player.role.team === "traveler") {
        // reset to an unknown role
        this._store.commit("players/update", {
          player,
          property: "role",
          value: {}
        });
      } else {
        // load role, first from session, the global, then fail gracefully
        const role =
          this._store.state.roles.get(value) ||
          this._store.getters.rolesJSONbyId.get(value) ||
          {};
        this._store.commit("players/update", {
          player,
          property: "role",
          value: role
        });
      }
    } else {
      // just update the player otherwise
      this._store.commit("players/update", { player, property, value });
    }
  }

  /**
   * Publish a player pronouns update
   * @param player
   * @param value
   * @param isFromSockets
   */
  sendPlayerPronouns({ player, value, isFromSockets }) {
    //send pronoun only for the seated player or storyteller
    //Do not re-send pronoun data for an update that was recieved from the sockets layer
    if (
      isFromSockets ||
      (this._isSpectator && this._store.state.session.playerId !== player.id)
    )
      return;
    const index = this._store.state.players.players.indexOf(player);
    this._send("pronouns", [index, value]);
  }

  /**
   * Update a pronouns based on incoming data.
   * @param index
   * @param value
   * @private
   */
  _updatePlayerPronouns([index, value]) {
    const player = this._store.state.players.players[index];

    this._store.commit("players/update", {
      player,
      property: "pronouns",
      value,
      isFromSockets: true
    });
  }

  /**
   * Handle a ping message by another player / storyteller
   * @param playerIdOrCount
   * @param latency
   * @private
   */
  _handlePing([playerIdOrCount = 0, latency] = []) {
    const now = new Date().getTime();
    if (!this._isSpectator) {
      // remove players that haven't sent a ping in twice the timespan
      for (let player in this._players) {
        if (now - this._players[player] > this._pingInterval * 2) {
          delete this._players[player];
          delete this._pings[player];
        }
      }
      // remove claimed seats from players that are no longer connected
      this._store.state.players.players.forEach(player => {
        if (player.id && !this._players[player.id]) {
          this._store.commit("players/update", {
            player,
            property: "id",
            value: ""
          });
        }
      });
      // store new player data
      if (playerIdOrCount) {
        this._players[playerIdOrCount] = now;
        const ping = parseInt(latency, 10);
        if (ping && ping > 0 && ping < 30 * 1000) {
          // ping to Players
          this._pings[playerIdOrCount] = ping;
          const pings = Object.values(this._pings);
          this._store.commit(
            "session/setPing",
            Math.round(pings.reduce((a, b) => a + b, 0) / pings.length)
          );
        }
      }
    } else if (latency) {
      // ping to ST
      this._store.commit("session/setPing", parseInt(latency, 10));
    }
    // update player count
    if (!this._isSpectator || playerIdOrCount) {
      this._store.commit(
        "session/setPlayerCount",
        this._isSpectator ? playerIdOrCount : Object.keys(this._players).length
      );
    }
  }

  /**
   * Handle a player leaving the sessions. ST only
   * @param playerId
   * @private
   */
  _handleBye(playerId) {
    if (this._isSpectator) return;
    delete this._players[playerId];
    this._store.commit(
      "session/setPlayerCount",
      Object.keys(this._players).length
    );
  }

  /**
   * Claim a seat, needs to be confirmed by the Storyteller.
   * Seats already occupied can't be claimed.
   * @param seat either -1 to vacate or the index of the seat claimed
   */
  claimSeat(seat) {
    if (!this._isSpectator) return;
    const players = this._store.state.players.players;
    if (players.length > seat && (seat < 0 || !players[seat].id)) {
      this._send("claim", [seat, this._store.state.session.playerId]);
    }
  }

  /**
   * Update a player id associated with that seat.
   * @param index seat index or -1
   * @param value playerId to add / remove
   * @private
   */
  _updateSeat([index, value]) {
    if (this._isSpectator) return;
    const property = "id";
    const players = this._store.state.players.players;
    // remove previous seat
    const oldIndex = players.findIndex(({ id }) => id === value);
    if (oldIndex >= 0 && oldIndex !== index) {
      this._store.commit("players/update", {
        player: players[oldIndex],
        property,
        value: ""
      });
    }
    // add playerId to new seat
    if (index >= 0) {
      const player = players[index];
      if (!player) return;
      this._store.commit("players/update", { player, property, value });
    }
    // update player session list as if this was a ping
    this._handlePing([true, value, 0]);
  }

  /**
   * Distribute player roles to all seated players in a direct message.
   * This will be split server side so that each player only receives their own (sub)message.
   */
  distributeRoles() {
    if (this._isSpectator) return;
    const message = {};
    this._store.state.players.players.forEach((player, index) => {
      if (player.id && player.role) {
        message[player.id] = [
          "player",
          { index, property: "role", value: player.role.id }
        ];
      }
    });
    if (Object.keys(message).length) {
      this._send("direct", message);
    }
  }

  /**
   * A player nomination. ST only
   * This also syncs the voting speed to the players.
   * Payload can be an object with {nomination} property or just the nomination itself, or undefined.
   * @param payload [nominator, nominee]|{nomination}
   */
  nomination(payload) {
    if (this._isSpectator) return;
    const nomination = payload ? payload.nomination || payload : payload;
    const players = this._store.state.players.players;
    if (
      !nomination ||
      (players.length > nomination[0] && players.length > nomination[1])
    ) {
      this.setVotingSpeed(this._store.state.session.votingSpeed);
      this._send("nomination", nomination);
    }
  }

  /**
   * Set the isVoteInProgress status. ST only
   */
  setVoteInProgress() {
    if (this._isSpectator) return;
    this._send("isVoteInProgress", this._store.state.session.isVoteInProgress);
  }

  /**
   * Send the isNight status. ST only
   */
  setIsNight() {
    if (this._isSpectator) return;
    this._send("isNight", this._store.state.grimoire.isNight);
  }

  /**
   * Send the isVoteHistoryAllowed state. ST only
   */
  setVoteHistoryAllowed() {
    if (this._isSpectator) return;
    this._send(
      "isVoteHistoryAllowed",
      this._store.state.session.isVoteHistoryAllowed
    );
  }

  /**
   * Send the voting speed. ST only
   * @param votingSpeed voting speed in seconds, minimum 1
   */
  setVotingSpeed(votingSpeed) {
    if (this._isSpectator) return;
    if (votingSpeed) {
      this._send("votingSpeed", votingSpeed);
    }
  }

  /**
   * Set which player is on the block. ST only
   * @param playerIndex, player id or -1 for empty
   */
  setMarked(playerIndex) {
    if (this._isSpectator) return;
    this._send("marked", playerIndex);
  }

  /**
   * Clear the vote history for everyone. ST only
   */
  clearVoteHistory() {
    if (this._isSpectator) return;
    this._send("clearVoteHistory");
  }

  /**
   * Send a vote. Player or ST
   * @param index Seat of the player
   * @param sync Flag whether to sync this vote with others or not
   */
  vote([index]) {
    const player = this._store.state.players.players[index];
    if (
      this._store.state.session.playerId === player.id ||
      !this._isSpectator
    ) {
      // send vote only if it is your own vote or you are the storyteller
      this._send("vote", [
        index,
        this._store.state.session.votes[index],
        !this._isSpectator
      ]);
    }
  }

  /**
   * Handle an incoming vote, but only if it is from ST or unlocked.
   * @param index
   * @param vote
   * @param fromST
   */
  _handleVote([index, vote, fromST]) {
    const { session, players } = this._store.state;
    const playerCount = players.players.length;
    const indexAdjusted =
      (index - 1 + playerCount - session.nomination[1]) % playerCount;
    if (fromST || indexAdjusted >= session.lockedVote - 1) {
      this._store.commit("session/vote", [index, vote]);
    }
  }

  /**
   * Lock a vote. ST only
   */
  lockVote() {
    if (this._isSpectator) return;
    const { lockedVote, votes, nomination } = this._store.state.session;
    const { players } = this._store.state.players;
    const index = (nomination[1] + lockedVote - 1) % players.length;
    this._send("lock", [this._store.state.session.lockedVote, votes[index]]);
  }

  /**
   * Update vote lock and the locked vote, if it differs. Player only
   * @param lock
   * @param vote
   * @private
   */
  _handleLock([lock, vote]) {
    if (!this._isSpectator) return;
    this._store.commit("session/lockVote", lock);
    if (lock > 1) {
      const { lockedVote, nomination } = this._store.state.session;
      const { players } = this._store.state.players;
      const index = (nomination[1] + lockedVote - 1) % players.length;
      if (this._store.state.session.votes[index] !== vote) {
        this._store.commit("session/vote", [index, vote]);
      }
    }
  }

  /**
   * Swap two player seats. ST only
   * @param payload
   */
  swapPlayer(payload) {
    if (this._isSpectator) return;
    this._send("swap", payload);
  }

  /**
   * Move a player to another seat. ST only
   * @param payload
   */
  movePlayer(payload) {
    if (this._isSpectator) return;
    this._send("move", payload);
  }

  /**
   * Remove a player. ST only
   * @param payload
   */
  removePlayer(payload) {
    if (this._isSpectator) return;
    this._send("remove", payload);
  }

  /**
   * Send a chat message.
   * @param text
   */
  sendChat(text) {
    if (!text) return;
    this._send("chat", { message: text });
  }
}

export default store => {
  // setup
  const session = new LiveSession(store);

  // listen to mutations
  store.subscribe(({ type, payload }, state) => {
    switch (type) {
      case "session/setSessionId":
        if (state.session.sessionId) {
          session.connect(state.session.sessionId);
        } else {
          window.location.hash = "";
          session.disconnect();
        }
        break;
      case "session/claimSeat":
        session.claimSeat(payload);
        break;
      case "session/distributeRoles":
        if (payload) {
          session.distributeRoles();
        }
        break;
      case "session/nomination":
      case "session/setNomination":
        session.nomination(payload);
        break;
      case "session/setVoteInProgress":
        session.setVoteInProgress(payload);
        break;
      case "session/voteSync":
        session.vote(payload);
        break;
      case "session/lockVote":
        session.lockVote();
        break;
      case "session/setVotingSpeed":
        session.setVotingSpeed(payload);
        break;
      case "session/clearVoteHistory":
        session.clearVoteHistory();
        break;
      case "session/setVoteHistoryAllowed":
        session.setVoteHistoryAllowed();
        break;
      case "toggleNight":
        session.setIsNight();
        break;
      case "setEdition":
        session.sendEdition();
        break;
      case "players/setFabled":
        session.sendFabled();
        break;
      case "session/setMarkedPlayer":
        session.setMarked(payload);
        break;
      case "players/swap":
        session.swapPlayer(payload);
        break;
      case "players/move":
        session.movePlayer(payload);
        break;
      case "players/remove":
        session.removePlayer(payload);
        break;
      case "players/set":
      case "players/clear":
      case "players/add":
        session.sendGamestate("", true);
        break;
      case "players/update":
        if (payload.property === "pronouns") {
          session.sendPlayerPronouns(payload);
        } else {
          session.sendPlayer(payload);
        }
        break;
      case "session/sendChat":
        session.sendChat(payload);
        break;
    }
  });

  // check for session Id in hash
  const sessionId = window.location.hash.substr(1);
  if (sessionId) {
    store.commit("session/setSpectator", true);
    store.commit("session/setSessionId", sessionId);
    store.commit("toggleGrimoire", false);
  }
};
