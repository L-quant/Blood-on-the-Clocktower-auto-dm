/**
 * 座位管理器
 * 管理座位状态和玩家-座位映射关系
 */

/**
 * 座位管理器类
 */
export default class SeatManager {
  constructor(maxSeats = 20) {
    this.maxSeats = maxSeats;
    this.seats = new Map(); // seatIndex -> { playerId, claimedAt, isActive }
    this.playerSeats = new Map(); // playerId -> seatIndex
  }

  /**
   * 获取所有座位状态
   * @returns {Array} 座位状态列表
   */
  getAllSeats() {
    const seatStatuses = [];
    
    for (let i = 0; i < this.maxSeats; i++) {
      const seat = this.seats.get(i);
      
      seatStatuses.push({
        index: i,
        isClaimed: !!seat,
        playerId: seat ? seat.playerId : null,
        claimedAt: seat ? seat.claimedAt : null,
        isActive: seat ? seat.isActive : false
      });
    }

    return seatStatuses;
  }

  /**
   * 获取空座位列表
   * @returns {Array} 空座位索引列表
   */
  getAvailableSeats() {
    const available = [];
    
    for (let i = 0; i < this.maxSeats; i++) {
      if (!this.seats.has(i)) {
        available.push(i);
      }
    }

    return available;
  }

  /**
   * 绑定玩家到座位
   * @param {string} playerId - 玩家ID
   * @param {number} seatIndex - 座位索引
   * @returns {boolean} 是否绑定成功
   */
  bindPlayerToSeat(playerId, seatIndex) {
    // 验证座位索引
    if (seatIndex < 0 || seatIndex >= this.maxSeats) {
      console.error(`[SeatManager] Invalid seat index: ${seatIndex}`);
      return false;
    }

    // 检查座位是否已被占用
    if (this.seats.has(seatIndex)) {
      const existingSeat = this.seats.get(seatIndex);
      if (existingSeat.playerId !== playerId) {
        console.warn(`[SeatManager] Seat ${seatIndex} already claimed by ${existingSeat.playerId}`);
        return false;
      }
      // 如果是同一个玩家，更新活动状态
      existingSeat.isActive = true;
      existingSeat.claimedAt = Date.now();
      return true;
    }

    // 检查玩家是否已经占用其他座位
    if (this.playerSeats.has(playerId)) {
      const oldSeatIndex = this.playerSeats.get(playerId);
      console.log(`[SeatManager] Player ${playerId} moving from seat ${oldSeatIndex} to ${seatIndex}`);
      this.unbindPlayer(playerId);
    }

    // 绑定座位
    this.seats.set(seatIndex, {
      playerId,
      claimedAt: Date.now(),
      isActive: true
    });

    this.playerSeats.set(playerId, seatIndex);

    console.log(`[SeatManager] Bound player ${playerId} to seat ${seatIndex}`);
    return true;
  }

  /**
   * 解绑玩家
   * @param {string} playerId - 玩家ID
   * @returns {boolean} 是否解绑成功
   */
  unbindPlayer(playerId) {
    const seatIndex = this.playerSeats.get(playerId);
    
    if (seatIndex === undefined) {
      console.warn(`[SeatManager] Player ${playerId} not found`);
      return false;
    }

    this.seats.delete(seatIndex);
    this.playerSeats.delete(playerId);

    console.log(`[SeatManager] Unbound player ${playerId} from seat ${seatIndex}`);
    return true;
  }

  /**
   * 获取座位的玩家ID
   * @param {number} seatIndex - 座位索引
   * @returns {string|null}
   */
  getPlayerIdBySeat(seatIndex) {
    const seat = this.seats.get(seatIndex);
    return seat ? seat.playerId : null;
  }

  /**
   * 获取玩家的座位索引
   * @param {string} playerId - 玩家ID
   * @returns {number} 座位索引，如果未找到返回-1
   */
  getSeatByPlayerId(playerId) {
    return this.playerSeats.get(playerId) ?? -1;
  }

  /**
   * 检查玩家是否已就座
   * @param {string} playerId - 玩家ID
   * @returns {boolean}
   */
  isPlayerSeated(playerId) {
    return this.playerSeats.has(playerId);
  }

  /**
   * 检查座位是否已被占用
   * @param {number} seatIndex - 座位索引
   * @returns {boolean}
   */
  isSeatClaimed(seatIndex) {
    return this.seats.has(seatIndex);
  }

  /**
   * 获取已占用座位数量
   * @returns {number}
   */
  getClaimedSeatsCount() {
    return this.seats.size;
  }

  /**
   * 获取空座位数量
   * @returns {number}
   */
  getAvailableSeatsCount() {
    return this.maxSeats - this.seats.size;
  }

  /**
   * 检查是否还有空座位
   * @returns {boolean}
   */
  hasAvailableSeats() {
    return this.seats.size < this.maxSeats;
  }

  /**
   * 设置座位活动状态
   * @param {number} seatIndex - 座位索引
   * @param {boolean} isActive - 是否活动
   */
  setSeatActive(seatIndex, isActive) {
    const seat = this.seats.get(seatIndex);
    if (seat) {
      seat.isActive = isActive;
    }
  }

  /**
   * 获取所有已就座的玩家ID
   * @returns {Array}
   */
  getAllSeatedPlayers() {
    return Array.from(this.playerSeats.keys());
  }

  /**
   * 获取座位详细信息
   * @param {number} seatIndex - 座位索引
   * @returns {object|null}
   */
  getSeatInfo(seatIndex) {
    const seat = this.seats.get(seatIndex);
    
    if (!seat) {
      return null;
    }

    return {
      index: seatIndex,
      playerId: seat.playerId,
      claimedAt: seat.claimedAt,
      isActive: seat.isActive
    };
  }

  /**
   * 清理不活动的座位
   * @param {number} inactiveThreshold - 不活动阈值（毫秒）
   * @returns {number} 清理的座位数量
   */
  cleanupInactiveSeats(inactiveThreshold = 5 * 60 * 1000) {
    const now = Date.now();
    let cleanedCount = 0;

    for (const [seatIndex, seat] of this.seats.entries()) {
      if (!seat.isActive && (now - seat.claimedAt) > inactiveThreshold) {
        this.seats.delete(seatIndex);
        this.playerSeats.delete(seat.playerId);
        console.log(`[SeatManager] Cleaned up inactive seat ${seatIndex}`);
        cleanedCount++;
      }
    }

    return cleanedCount;
  }

  /**
   * 设置最大座位数
   * @param {number} maxSeats - 最大座位数
   */
  setMaxSeats(maxSeats) {
    if (maxSeats < this.seats.size) {
      console.warn(`[SeatManager] Cannot reduce max seats below current claimed seats`);
      return false;
    }

    this.maxSeats = maxSeats;
    return true;
  }

  /**
   * 重置座位管理器
   */
  reset() {
    this.seats.clear();
    this.playerSeats.clear();
    console.log('[SeatManager] Reset all seats');
  }

  /**
   * 导出座位状态
   * @returns {object}
   */
  exportState() {
    return {
      maxSeats: this.maxSeats,
      seats: Array.from(this.seats.entries()),
      playerSeats: Array.from(this.playerSeats.entries())
    };
  }

  /**
   * 导入座位状态
   * @param {object} state - 座位状态
   */
  importState(state) {
    if (!state) return;

    this.maxSeats = state.maxSeats || this.maxSeats;
    this.seats = new Map(state.seats || []);
    this.playerSeats = new Map(state.playerSeats || []);

    console.log('[SeatManager] Imported state');
  }
}
