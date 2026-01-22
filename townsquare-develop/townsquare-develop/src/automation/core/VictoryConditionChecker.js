/**
 * 胜负判断器
 * 检查和判断游戏的胜负条件
 */

import { Team, GamePhase } from '../types/GameTypes';

/**
 * 胜负判断器类
 */
export default class VictoryConditionChecker {
  constructor(gameStateManager, rolePrivacySystem = null) {
    this.gameStateManager = gameStateManager;
    this.rolePrivacySystem = rolePrivacySystem;
  }

  /**
   * 检查胜利条件
   * @param {GameState} gameState 游戏状态（可选，默认使用当前状态）
   * @returns {object} 胜利结果 {winner: 'good'|'evil'|null, reason: string, ended: boolean}
   */
  checkVictoryConditions(gameState = null) {
    const state = gameState || this.gameStateManager.getCurrentState();

    // 检查好人胜利条件
    const goodVictory = this.checkGoodVictory(state);
    if (goodVictory.victory) {
      return {
        winner: 'good',
        reason: goodVictory.reason,
        ended: true,
        details: goodVictory.details
      };
    }

    // 检查恶人胜利条件
    const evilVictory = this.checkEvilVictory(state);
    if (evilVictory.victory) {
      return {
        winner: 'evil',
        reason: evilVictory.reason,
        ended: true,
        details: evilVictory.details
      };
    }

    // 游戏继续
    return {
      winner: null,
      reason: 'Game continues',
      ended: false
    };
  }

  /**
   * 检查好人胜利条件
   * @param {GameState} gameState 游戏状态
   * @returns {object} {victory: boolean, reason: string, details: object}
   */
  checkGoodVictory(gameState) {
    const alivePlayers = gameState.players.filter(p => p.isAlive);
    const aliveEvil = alivePlayers.filter(p => p.isEvil);
    const aliveDemons = alivePlayers.filter(p => p.role && p.role.team === Team.DEMON);

    // 条件1：所有恶魔都死亡
    if (aliveDemons.length === 0) {
      return {
        victory: true,
        reason: 'All demons are dead',
        details: {
          aliveGood: alivePlayers.filter(p => !p.isEvil).length,
          deadDemons: gameState.players.filter(p => 
            !p.isAlive && p.role && p.role.team === Team.DEMON
          ).length
        }
      };
    }

    return {
      victory: false,
      reason: 'Demons still alive'
    };
  }

  /**
   * 检查恶人胜利条件
   * @param {GameState} gameState 游戏状态
   * @returns {object} {victory: boolean, reason: string, details: object}
   */
  checkEvilVictory(gameState) {
    const alivePlayers = gameState.players.filter(p => p.isAlive);
    const aliveGood = alivePlayers.filter(p => !p.isEvil);
    const aliveEvil = alivePlayers.filter(p => p.isEvil);

    // 条件1：只剩下2个存活玩家（1个恶人和1个好人）
    if (alivePlayers.length === 2 && aliveEvil.length >= 1 && aliveGood.length >= 1) {
      return {
        victory: true,
        reason: 'Only 2 players remain (final two)',
        details: {
          aliveGood: aliveGood.length,
          aliveEvil: aliveEvil.length,
          totalAlive: alivePlayers.length
        }
      };
    }

    // 条件2：恶人数量等于或超过好人数量
    if (aliveEvil.length >= aliveGood.length && aliveEvil.length > 0) {
      return {
        victory: true,
        reason: 'Evil players equal or outnumber good players',
        details: {
          aliveGood: aliveGood.length,
          aliveEvil: aliveEvil.length
        }
      };
    }

    return {
      victory: false,
      reason: 'Good players still outnumber evil'
    };
  }

  /**
   * 验证游戏是否结束
   * @param {GameState} gameState 游戏状态（可选）
   * @returns {boolean}
   */
  isGameEnded(gameState = null) {
    const result = this.checkVictoryConditions(gameState);
    return result.ended;
  }

  /**
   * 生成游戏结果报告
   * @param {GameState} gameState 游戏状态
   * @param {object} outcome 游戏结果
   * @returns {object} 游戏报告
   */
  generateGameReport(gameState, outcome) {
    const alivePlayers = gameState.players.filter(p => p.isAlive);
    const deadPlayers = gameState.players.filter(p => !p.isAlive);

    const report = {
      gameId: gameState.gameId,
      winner: outcome.winner,
      reason: outcome.reason,
      duration: {
        days: gameState.day,
        phase: gameState.phase
      },
      players: {
        total: gameState.players.length,
        alive: alivePlayers.length,
        dead: deadPlayers.length
      },
      teams: {
        good: {
          total: gameState.players.filter(p => !p.isEvil).length,
          alive: alivePlayers.filter(p => !p.isEvil).length,
          dead: deadPlayers.filter(p => !p.isEvil).length
        },
        evil: {
          total: gameState.players.filter(p => p.isEvil).length,
          alive: alivePlayers.filter(p => p.isEvil).length,
          dead: deadPlayers.filter(p => p.isEvil).length
        }
      },
      playerDetails: gameState.players.map(p => ({
        id: p.id,
        name: p.name,
        role: p.role ? p.role.name : 'Unknown',
        team: p.isEvil ? 'evil' : 'good',
        isAlive: p.isAlive
      })),
      timestamp: Date.now()
    };

    return report;
  }

  /**
   * 处理特殊胜利条件
   * @param {Array} conditions 特殊条件列表
   * @returns {object} 胜利结果
   */
  handleSpecialVictoryConditions(conditions) {
    // 这里可以处理特殊角色的胜利条件
    // 例如：圣徒（Saint）被处决时好人输
    // 例如：小丑（Jester）的特殊胜利条件

    for (const condition of conditions) {
      if (condition.type === 'saint_executed') {
        return {
          winner: 'evil',
          reason: 'Saint was executed',
          ended: true,
          special: true,
          details: condition.details
        };
      }

      if (condition.type === 'atheist_executed') {
        return {
          winner: 'evil',
          reason: 'Atheist was executed',
          ended: true,
          special: true,
          details: condition.details
        };
      }
    }

    return {
      winner: null,
      reason: 'No special victory conditions met',
      ended: false
    };
  }

  /**
   * 检查并结束游戏（如果满足胜利条件）
   * @returns {Promise<object|null>} 如果游戏结束返回结果，否则返回null
   */
  async checkAndEndGame() {
    const gameState = this.gameStateManager.getCurrentState();

    // 只在游戏进行中检查
    if (gameState.phase === GamePhase.ENDED || gameState.phase === GamePhase.SETUP) {
      return null;
    }

    const victoryResult = this.checkVictoryConditions(gameState);

    if (victoryResult.ended) {
      // 转换到游戏结束阶段
      await this.gameStateManager.transitionToPhase(GamePhase.ENDED);

      // 如果启用了隐私保护，游戏结束时解除隐私保护
      if (this.rolePrivacySystem && this.rolePrivacySystem.isPrivacyEnabled()) {
        this.rolePrivacySystem.disablePrivacyProtection();
        console.log('Privacy protection disabled - all roles are now visible');
      }

      // 生成游戏报告
      const report = this.generateGameReport(gameState, victoryResult);

      console.log(`Game ended! Winner: ${victoryResult.winner}. Reason: ${victoryResult.reason}`);

      return {
        ...victoryResult,
        report,
        rolesRevealed: true
      };
    }

    return null;
  }

  /**
   * 获取当前游戏状态摘要
   * @returns {object} 游戏状态摘要
   */
  getGameStatusSummary() {
    const gameState = this.gameStateManager.getCurrentState();
    const alivePlayers = gameState.players.filter(p => p.isAlive);
    const aliveGood = alivePlayers.filter(p => !p.isEvil);
    const aliveEvil = alivePlayers.filter(p => p.isEvil);

    return {
      phase: gameState.phase,
      day: gameState.day,
      totalPlayers: gameState.players.length,
      alivePlayers: alivePlayers.length,
      aliveGood: aliveGood.length,
      aliveEvil: aliveEvil.length,
      isEnded: this.isGameEnded(gameState)
    };
  }

  /**
   * 验证胜利条件的一致性
   * @param {GameState} gameState 游戏状态
   * @returns {object} {valid: boolean, errors: string[]}
   */
  validateVictoryConditions(gameState) {
    const errors = [];

    const alivePlayers = gameState.players.filter(p => p.isAlive);
    const aliveGood = alivePlayers.filter(p => !p.isEvil);
    const aliveEvil = alivePlayers.filter(p => p.isEvil);

    // 检查是否有玩家存活
    if (alivePlayers.length === 0) {
      errors.push('No players alive');
    }

    // 检查是否有好人和恶人
    if (aliveGood.length === 0 && aliveEvil.length === 0) {
      errors.push('No good or evil players alive');
    }

    // 检查角色分配
    gameState.players.forEach(player => {
      if (!player.role) {
        errors.push(`Player ${player.id} has no role assigned`);
      }
    });

    return {
      valid: errors.length === 0,
      errors
    };
  }

  /**
   * 预测游戏结果
   * @param {GameState} gameState 游戏状态
   * @returns {object} 预测结果
   */
  predictGameOutcome(gameState) {
    const alivePlayers = gameState.players.filter(p => p.isAlive);
    const aliveGood = alivePlayers.filter(p => !p.isEvil);
    const aliveEvil = alivePlayers.filter(p => p.isEvil);
    const aliveDemons = alivePlayers.filter(p => p.role && p.role.team === Team.DEMON);

    let prediction = 'uncertain';
    let confidence = 0;
    let reasoning = '';

    // 如果恶魔都死了，好人必胜
    if (aliveDemons.length === 0) {
      prediction = 'good';
      confidence = 1.0;
      reasoning = 'All demons are dead';
    }
    // 如果恶人数量等于或超过好人，恶人必胜
    else if (aliveEvil.length >= aliveGood.length) {
      prediction = 'evil';
      confidence = 1.0;
      reasoning = 'Evil equals or outnumbers good';
    }
    // 如果好人远多于恶人（超过2倍）
    else if (aliveGood.length > aliveEvil.length * 2) {
      prediction = 'good';
      confidence = 0.7;
      reasoning = 'Good significantly outnumbers evil';
    }
    // 如果好人多于恶人但不到2倍
    else if (aliveGood.length > aliveEvil.length) {
      prediction = 'good';
      confidence = 0.5;
      reasoning = 'Good outnumbers evil';
    }
    // 如果恶人接近好人数量
    else if (aliveEvil.length >= aliveGood.length * 0.8) {
      prediction = 'evil';
      confidence = 0.6;
      reasoning = 'Evil is close to good in numbers';
    }

    return {
      prediction,
      confidence,
      reasoning,
      currentState: {
        aliveGood: aliveGood.length,
        aliveEvil: aliveEvil.length,
        aliveDemons: aliveDemons.length
      }
    };
  }
}
