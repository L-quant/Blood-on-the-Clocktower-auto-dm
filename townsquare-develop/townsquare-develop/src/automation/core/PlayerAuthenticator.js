/**
 * 玩家身份验证器
 * 管理玩家身份令牌和座位认领
 */

/**
 * 生成随机令牌
 * @returns {string}
 */
function generateRandomToken() {
  return Math.random().toString(36).substring(2) + 
         Date.now().toString(36) +
         Math.random().toString(36).substring(2);
}

/**
 * 玩家身份验证器类
 */
export default class PlayerAuthenticator {
  constructor() {
    this.tokens = new Map(); // playerId -> PlayerToken
    this.seatBindings = new Map(); // seatIndex -> { playerId, token, claimedAt, lastActivity }
    this.tokenExpirationTime = 24 * 60 * 60 * 1000; // 24小时
    this.reconnectionTimeout = 5 * 60 * 1000; // 5分钟
  }

  /**
   * 生成玩家令牌
   * @param {string} playerId - 玩家ID
   * @param {number} seatIndex - 座位索引
   * @returns {object} PlayerToken
   */
  generatePlayerToken(playerId, seatIndex) {
    const now = Date.now();
    
    const token = {
      playerId,
      seatIndex,
      token: generateRandomToken(),
      issuedAt: now,
      expiresAt: now + this.tokenExpirationTime,
      lastActivity: now
    };

    // 存储令牌
    this.tokens.set(playerId, token);

    console.log(`[PlayerAuthenticator] Generated token for player ${playerId} at seat ${seatIndex}`);

    return token;
  }

  /**
   * 验证令牌
   * @param {object} tokenObj - 令牌对象
   * @returns {object} TokenValidationResult
   */
  validateToken(tokenObj) {
    if (!tokenObj || !tokenObj.token || !tokenObj.playerId) {
      return {
        isValid: false,
        reason: 'Invalid token format'
      };
    }

    // 查找存储的令牌
    const storedToken = this.tokens.get(tokenObj.playerId);

    if (!storedToken) {
      return {
        isValid: false,
        reason: 'Token not found'
      };
    }

    // 验证令牌字符串
    if (storedToken.token !== tokenObj.token) {
      return {
        isValid: false,
        reason: 'Token mismatch'
      };
    }

    // 检查是否过期
    if (Date.now() > storedToken.expiresAt) {
      this.tokens.delete(tokenObj.playerId);
      return {
        isValid: false,
        reason: 'Token expired'
      };
    }

    // 更新最后活动时间
    storedToken.lastActivity = Date.now();

    return {
      isValid: true,
      playerId: storedToken.playerId,
      seatIndex: storedToken.seatIndex
    };
  }

  /**
   * 认领座位
   * @param {string} playerId - 玩家ID
   * @param {number} seatIndex - 座位索引
   * @returns {Promise<object>} ClaimResult
   */
  async claimSeat(playerId, seatIndex) {
    // 检查座位是否已被占用
    if (this.seatBindings.has(seatIndex)) {
      const binding = this.seatBindings.get(seatIndex);
      
      // 如果是同一个玩家，返回现有令牌
      if (binding.playerId === playerId) {
        return {
          success: true,
          token: this.tokens.get(playerId)
        };
      }

      return {
        success: false,
        error: 'Seat already claimed by another player'
      };
    }

    // 检查玩家是否已经认领了其他座位
    for (const [existingSeatIndex, binding] of this.seatBindings.entries()) {
      if (binding.playerId === playerId) {
        // 释放旧座位
        this.releaseSeat(playerId);
        break;
      }
    }

    // 生成新令牌
    const token = this.generatePlayerToken(playerId, seatIndex);

    // 绑定座位
    this.seatBindings.set(seatIndex, {
      playerId,
      token: token.token,
      claimedAt: Date.now(),
      lastActivity: Date.now()
    });

    console.log(`[PlayerAuthenticator] Player ${playerId} claimed seat ${seatIndex}`);

    return {
      success: true,
      token
    };
  }

  /**
   * 释放座位
   * @param {string} playerId - 玩家ID
   */
  releaseSeat(playerId) {
    // 查找玩家的座位
    for (const [seatIndex, binding] of this.seatBindings.entries()) {
      if (binding.playerId === playerId) {
        this.seatBindings.delete(seatIndex);
        this.tokens.delete(playerId);
        console.log(`[PlayerAuthenticator] Released seat ${seatIndex} for player ${playerId}`);
        return;
      }
    }
  }

  /**
   * 获取玩家的座位
   * @param {string} playerId - 玩家ID
   * @returns {number} 座位索引，如果未找到返回-1
   */
  getPlayerSeat(playerId) {
    for (const [seatIndex, binding] of this.seatBindings.entries()) {
      if (binding.playerId === playerId) {
        return seatIndex;
      }
    }
    return -1;
  }

  /**
   * 检查座位是否已认领
   * @param {number} seatIndex - 座位索引
   * @returns {boolean}
   */
  isSeatClaimed(seatIndex) {
    return this.seatBindings.has(seatIndex);
  }

  /**
   * 获取座位的玩家ID
   * @param {number} seatIndex - 座位索引
   * @returns {string|null}
   */
  getPlayerIdBySeat(seatIndex) {
    const binding = this.seatBindings.get(seatIndex);
    return binding ? binding.playerId : null;
  }

  /**
   * 处理断线重连
   * @param {string} playerId - 玩家ID
   * @param {object} tokenObj - 令牌对象
   * @returns {Promise<object>} ReconnectionResult
   */
  async handleReconnection(playerId, tokenObj) {
    // 验证令牌
    const validation = this.validateToken(tokenObj);

    if (!validation.isValid) {
      return {
        success: false,
        error: `Token validation failed: ${validation.reason}`
      };
    }

    // 检查座位绑定是否仍然存在
    const seatIndex = validation.seatIndex;
    const binding = this.seatBindings.get(seatIndex);

    if (!binding || binding.playerId !== playerId) {
      return {
        success: false,
        error: 'Seat binding lost'
      };
    }

    // 更新最后活动时间
    binding.lastActivity = Date.now();

    console.log(`[PlayerAuthenticator] Player ${playerId} reconnected to seat ${seatIndex}`);

    return {
      success: true,
      seatIndex
    };
  }

  /**
   * 清理过期令牌
   */
  cleanupExpiredTokens() {
    const now = Date.now();
    let cleanedCount = 0;

    // 清理过期令牌
    for (const [playerId, token] of this.tokens.entries()) {
      if (now > token.expiresAt) {
        this.tokens.delete(playerId);
        cleanedCount++;
      }
    }

    // 清理长时间不活动的座位绑定
    for (const [seatIndex, binding] of this.seatBindings.entries()) {
      if (now - binding.lastActivity > this.reconnectionTimeout) {
        this.seatBindings.delete(seatIndex);
        this.tokens.delete(binding.playerId);
        console.log(`[PlayerAuthenticator] Released inactive seat ${seatIndex}`);
        cleanedCount++;
      }
    }

    if (cleanedCount > 0) {
      console.log(`[PlayerAuthenticator] Cleaned up ${cleanedCount} expired tokens/bindings`);
    }

    return cleanedCount;
  }

  /**
   * 获取所有座位绑定
   * @returns {Array}
   */
  getAllSeatBindings() {
    const bindings = [];
    for (const [seatIndex, binding] of this.seatBindings.entries()) {
      bindings.push({
        seatIndex,
        playerId: binding.playerId,
        claimedAt: binding.claimedAt,
        lastActivity: binding.lastActivity
      });
    }
    return bindings;
  }

  /**
   * 设置令牌过期时间
   * @param {number} milliseconds - 毫秒数
   */
  setTokenExpirationTime(milliseconds) {
    this.tokenExpirationTime = milliseconds;
  }

  /**
   * 设置重连超时时间
   * @param {number} milliseconds - 毫秒数
   */
  setReconnectionTimeout(milliseconds) {
    this.reconnectionTimeout = milliseconds;
  }

  /**
   * 重置验证器
   */
  reset() {
    this.tokens.clear();
    this.seatBindings.clear();
    console.log('[PlayerAuthenticator] Reset all tokens and bindings');
  }
}
