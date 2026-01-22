/**
 * 视角过滤器
 * 根据玩家身份过滤游戏状态中的敏感信息
 */

/**
 * 信息类型枚举
 */
export const InformationType = {
  PUBLIC: 'public',        // 公开信息
  PRIVATE: 'private',      // 私密信息
  ROLE: 'role'            // 角色信息
};

/**
 * 视角过滤器类
 */
export default class PerspectiveFilter {
  constructor() {
    this.gameEndedOverride = false;
  }

  /**
   * 过滤完整游戏状态
   * @param {object} gameState - 完整游戏状态
   * @param {string} playerId - 玩家ID
   * @returns {object} 过滤后的游戏状态
   */
  filterGameState(gameState, playerId) {
    if (!gameState || !playerId) {
      console.warn('[PerspectiveFilter] Invalid parameters');
      return gameState;
    }

    // 如果游戏已结束，返回完整状态
    if (this.gameEndedOverride || gameState.isGameEnded) {
      return gameState;
    }

    // 创建过滤后的状态副本
    const filteredState = {
      ...gameState,
      players: this._filterPlayers(gameState.players, playerId, gameState.isGameEnded),
      publicInformation: this._extractPublicInformation(gameState),
      privateInformation: this._extractPrivateInformation(gameState, playerId)
    };

    return filteredState;
  }

  /**
   * 过滤玩家列表
   * @private
   * @param {Array} players - 玩家列表
   * @param {string} viewerId - 查看者ID
   * @param {boolean} isGameEnded - 游戏是否结束
   * @returns {Array} 过滤后的玩家列表
   */
  _filterPlayers(players, viewerId, isGameEnded = false) {
    if (!players || !Array.isArray(players)) {
      return [];
    }

    return players.map(player => 
      this.filterPlayer(player, viewerId, isGameEnded)
    );
  }

  /**
   * 过滤单个玩家信息
   * @param {object} player - 玩家对象
   * @param {string} viewerId - 查看者ID
   * @param {boolean} isGameEnded - 游戏是否结束
   * @returns {object} 过滤后的玩家对象
   */
  filterPlayer(player, viewerId, isGameEnded = false) {
    if (!player) {
      return null;
    }

    const isMyPlayer = player.id === viewerId;
    const shouldShowRole = isGameEnded || isMyPlayer;

    // 创建过滤后的玩家对象
    const filteredPlayer = {
      id: player.id,
      name: player.name,
      seatIndex: player.seatIndex || 0,
      
      // 公开信息
      isAlive: player.isAlive !== undefined ? player.isAlive : !player.isDead,
      isDead: player.isDead || false,
      isVoteless: player.isVoteless || false,
      pronouns: player.pronouns || '',
      
      // 角色信息（仅自己或游戏结束时可见）
      role: shouldShowRole ? player.role : {},
      
      // 角色卡显示状态
      roleCardVisible: shouldShowRole,
      
      // 提醒标记（公开信息）
      reminders: player.reminders || []
    };

    return filteredPlayer;
  }

  /**
   * 提取公开信息
   * @private
   * @param {object} gameState - 游戏状态
   * @returns {object} 公开信息
   */
  _extractPublicInformation(gameState) {
    const publicInfo = {
      // 死亡玩家列表（索引）
      deadPlayers: [],
      
      // 当前游戏阶段
      currentPhase: gameState.currentPhase || 'day',
      
      // 天数
      dayNumber: gameState.dayNumber || 1,
      
      // 投票结果
      votingResults: gameState.votingResults || [],
      
      // 提名信息
      nominations: gameState.nominations || []
    };

    // 收集死亡玩家索引
    if (gameState.players && Array.isArray(gameState.players)) {
      gameState.players.forEach((player, index) => {
        if (player.isDead) {
          publicInfo.deadPlayers.push(index);
        }
      });
    }

    return publicInfo;
  }

  /**
   * 提取私密信息（仅当前玩家）
   * @private
   * @param {object} gameState - 游戏状态
   * @param {string} playerId - 玩家ID
   * @returns {object} 私密信息
   */
  _extractPrivateInformation(gameState, playerId) {
    // 找到当前玩家
    const currentPlayer = gameState.players?.find(p => p.id === playerId);
    
    if (!currentPlayer) {
      return null;
    }

    const privateInfo = {
      // 我的角色
      myRole: currentPlayer.role || {},
      
      // 夜间信息
      nightInformation: this.filterNightInformation(
        gameState.nightInformation || [],
        playerId
      ),
      
      // 能力结果
      abilityResults: this._filterAbilityResults(
        gameState.abilityResults || [],
        playerId
      )
    };

    return privateInfo;
  }

  /**
   * 过滤夜间信息
   * @param {Array} nightInfo - 夜间信息列表
   * @param {string} playerId - 玩家ID
   * @param {boolean} isPlayerDead - 玩家是否死亡
   * @returns {Array} 过滤后的夜间信息
   */
  filterNightInformation(nightInfo, playerId, isPlayerDead = false) {
    if (!nightInfo || !Array.isArray(nightInfo)) {
      return [];
    }

    // 只返回与当前玩家相关的夜间信息
    let filtered = nightInfo.filter(info => {
      return info.targetPlayerId === playerId || 
             info.recipientPlayerId === playerId;
    });

    // 如果玩家已死亡，不再接收新的夜间信息
    // 但保留死亡前的信息
    if (isPlayerDead) {
      const deathTime = this._getPlayerDeathTime(playerId);
      if (deathTime) {
        filtered = filtered.filter(info => {
          return info.timestamp < deathTime;
        });
      }
    }

    return filtered;
  }

  /**
   * 获取玩家死亡时间
   * @private
   * @param {string} playerId - 玩家ID
   * @returns {number|null} 死亡时间戳
   */
  _getPlayerDeathTime(playerId) {
    // 这里应该从游戏状态中获取玩家的死亡时间
    // 简化实现：返回null表示不限制
    return null;
  }

  /**
   * 过滤能力结果
   * @private
   * @param {Array} abilityResults - 能力结果列表
   * @param {string} playerId - 玩家ID
   * @returns {Array} 过滤后的能力结果
   */
  _filterAbilityResults(abilityResults, playerId) {
    if (!abilityResults || !Array.isArray(abilityResults)) {
      return [];
    }

    // 只返回当前玩家的能力结果
    return abilityResults.filter(result => {
      return result.playerId === playerId;
    });
  }

  /**
   * 分类信息类型
   * @param {object} info - 游戏信息
   * @returns {string} 信息类型
   */
  classifyInformation(info) {
    if (!info) {
      return InformationType.PUBLIC;
    }

    // 角色信息
    if (info.type === 'role' || info.isRoleInfo) {
      return InformationType.ROLE;
    }

    // 私密信息（夜间信息、能力结果等）
    if (info.type === 'night' || 
        info.type === 'ability' || 
        info.isPrivate) {
      return InformationType.PRIVATE;
    }

    // 默认为公开信息
    return InformationType.PUBLIC;
  }

  /**
   * 检查信息是否对玩家可见
   * @param {object} info - 游戏信息
   * @param {string} playerId - 玩家ID
   * @returns {boolean}
   */
  isInformationVisibleTo(info, playerId) {
    const infoType = this.classifyInformation(info);

    switch (infoType) {
      case InformationType.PUBLIC:
        // 公开信息对所有人可见
        return true;

      case InformationType.PRIVATE:
        // 私密信息只对相关玩家可见
        return info.targetPlayerId === playerId || 
               info.recipientPlayerId === playerId ||
               info.playerId === playerId;

      case InformationType.ROLE:
        // 角色信息只对自己可见（除非游戏结束）
        return info.playerId === playerId || this.gameEndedOverride;

      default:
        return false;
    }
  }

  /**
   * 设置游戏结束覆盖（解除所有隐私保护）
   * @param {boolean} isEnded - 游戏是否结束
   */
  setGameEndedOverride(isEnded) {
    this.gameEndedOverride = isEnded;
    if (isEnded) {
      console.log('[PerspectiveFilter] Game ended - all roles revealed');
    }
  }

  /**
   * 重置过滤器
   */
  reset() {
    this.gameEndedOverride = false;
  }
}
