/**
 * 投票管理器
 * 管理白天的讨论和投票流程
 */

import { GamePhase } from '../types/GameTypes';

/**
 * 投票管理器类
 */
export default class VotingManager {
  constructor(gameStateManager, victoryConditionChecker = null, options = {}) {
    this.gameStateManager = gameStateManager;
    this.victoryConditionChecker = victoryConditionChecker;
    this.currentDayPhase = null;
    this.nominations = [];
    this.votes = [];
    this.executedPlayer = null;
    this.privacyMode = options.privacyMode || false;
  }

  /**
   * 开始白天阶段
   * @param {object} nightResult 夜间结果
   * @returns {Promise<void>}
   */
  async startDayPhase(nightResult = {}) {
    const gameState = this.gameStateManager.getCurrentState();
    
    // 验证当前阶段
    if (gameState.phase !== GamePhase.DAY) {
      throw new Error(`Cannot start day phase from ${gameState.phase}`);
    }

    // 初始化白天阶段数据
    this.currentDayPhase = {
      day: gameState.day,
      nightResult,
      nominations: [],
      votes: [],
      executedPlayer: null,
      startTime: Date.now(),
      privacyMode: this.privacyMode
    };

    // 清空之前的提名和投票
    this.nominations = [];
    this.votes = [];
    this.executedPlayer = null;

    if (this.privacyMode) {
      console.log(`Starting day phase for day ${gameState.day} (Privacy Mode)`);
    } else {
      console.log(`Starting day phase for day ${gameState.day}`);
    }
    
    // 公布夜间死亡信息（但不透露角色）
    if (nightResult.deaths && nightResult.deaths.length > 0) {
      if (this.privacyMode) {
        console.log(`Night deaths: ${nightResult.deaths.map(d => d.name).join(', ')} (roles hidden)`);
      } else {
        console.log(`Night deaths: ${nightResult.deaths.map(d => d.name).join(', ')}`);
      }
    }
  }

  /**
   * 处理玩家提名
   * @param {Player} nominator 提名者
   * @param {Player} nominee 被提名者
   * @returns {Promise<void>}
   */
  async handleNomination(nominator, nominee) {
    if (!this.currentDayPhase) {
      throw new Error('No active day phase');
    }

    // 验证提名的合法性
    const validation = this.validateNomination(nominator, nominee);
    if (!validation.valid) {
      throw new Error(`Invalid nomination: ${validation.errors.join(', ')}`);
    }

    // 记录提名
    const nomination = {
      nominator: {
        id: nominator.id,
        name: nominator.name
      },
      nominee: {
        id: nominee.id,
        name: nominee.name
      },
      timestamp: Date.now()
    };

    this.nominations.push(nomination);
    this.currentDayPhase.nominations.push(nomination);

    console.log(`${nominator.name} nominated ${nominee.name}`);

    // 自动开始投票
    await this.conductVoting(nominee);
  }

  /**
   * 管理投票过程
   * @param {Player} nominee 被提名者
   * @returns {Promise<VotingResult>} 投票结果
   */
  async conductVoting(nominee) {
    if (!this.currentDayPhase) {
      throw new Error('No active day phase');
    }

    const gameState = this.gameStateManager.getCurrentState();
    const alivePlayers = gameState.players.filter(p => p.isAlive);

    // 收集所有玩家的投票
    const votesForNominee = [];
    
    for (const player of alivePlayers) {
      // 这里简化处理，实际应该等待玩家输入
      // 暂时使用随机投票模拟
      const vote = {
        voter: {
          id: player.id,
          name: player.name
        },
        nominee: {
          id: nominee.id,
          name: nominee.name
        },
        vote: false, // 默认不投票，实际应该由玩家决定
        timestamp: Date.now()
      };

      votesForNominee.push(vote);
    }

    this.votes.push(...votesForNominee);

    // 计算投票结果
    const result = this.calculateVotingResult(votesForNominee);
    result.nominee = nominee;

    console.log(`Voting for ${nominee.name}: ${result.votesFor} votes (need ${result.votesNeeded})`);

    return result;
  }

  /**
   * 计算投票结果
   * @param {Vote[]} votes 投票列表
   * @returns {VotingResult} 投票结果
   */
  calculateVotingResult(votes) {
    const gameState = this.gameStateManager.getCurrentState();
    const alivePlayers = gameState.players.filter(p => p.isAlive);
    
    // 计算赞成票数
    let votesFor = 0;
    
    votes.forEach(vote => {
      if (vote.vote) {
        const voter = gameState.players.find(p => p.id === vote.voter.id);
        
        // 检查投票修改器
        let voteCount = 1;
        if (voter.status && voter.status.voteModifier) {
          voteCount += voter.status.voteModifier;
        }
        
        votesFor += voteCount;
      }
    });

    // 计算所需票数（超过半数）
    const votesNeeded = Math.floor(alivePlayers.length / 2) + 1;

    // 判断是否通过
    const passed = votesFor >= votesNeeded;

    return {
      votesFor,
      votesAgainst: votes.length - votesFor,
      votesNeeded,
      totalVoters: votes.length,
      passed,
      votes
    };
  }

  /**
   * 执行处决
   * @param {Player} player 被处决的玩家
   * @returns {Promise<void>}
   */
  async executePlayer(player) {
    if (!this.currentDayPhase) {
      throw new Error('No active day phase');
    }

    // 更新玩家状态为死亡
    this.gameStateManager.updatePlayerState(player.id, { isAlive: false });

    // 记录处决
    this.executedPlayer = player;
    this.currentDayPhase.executedPlayer = player;

    if (this.privacyMode) {
      console.log(`${player.name} has been executed (role hidden)`);
    } else {
      console.log(`${player.name} has been executed`);
    }

    // 检查特殊胜利条件（例如圣徒被处决）
    if (this.victoryConditionChecker) {
      const specialConditions = this._checkSpecialExecutionConditions(player);
      if (specialConditions.length > 0) {
        const specialVictory = this.victoryConditionChecker.handleSpecialVictoryConditions(specialConditions);
        if (specialVictory.ended) {
          // 游戏因特殊条件结束
          await this.gameStateManager.transitionToPhase(GamePhase.ENDED);
          this.currentDayPhase.specialVictory = specialVictory;
        }
      }
    }
  }

  /**
   * 完成白天阶段
   * @returns {Promise<object>} 白天结果
   */
  async completeDayPhase() {
    if (!this.currentDayPhase) {
      throw new Error('No active day phase');
    }

    this.currentDayPhase.endTime = Date.now();
    this.currentDayPhase.duration = this.currentDayPhase.endTime - this.currentDayPhase.startTime;

    // 检查胜负条件
    if (this.victoryConditionChecker) {
      const victoryResult = await this.victoryConditionChecker.checkAndEndGame();
      if (victoryResult) {
        this.currentDayPhase.gameEnded = true;
        this.currentDayPhase.victoryResult = victoryResult;
      }
    }

    const result = { ...this.currentDayPhase };
    this.currentDayPhase = null;

    console.log(`Day phase completed. Executed: ${result.executedPlayer ? result.executedPlayer.name : 'None'}`);

    return result;
  }

  /**
   * 转换到夜间阶段
   * @returns {Promise<void>}
   */
  async transitionToNightPhase() {
    const gameState = this.gameStateManager.getCurrentState();
    
    // 验证当前阶段
    if (gameState.phase !== GamePhase.DAY) {
      throw new Error(`Cannot transition to night from ${gameState.phase}`);
    }

    // 检查游戏是否已结束
    if (gameState.phase === GamePhase.ENDED) {
      console.log('Game has ended, skipping night phase transition');
      return;
    }

    try {
      // 转换到夜间阶段
      await this.gameStateManager.transitionToPhase(GamePhase.NIGHT);
      
      console.log(`Transitioned to night phase (Day ${gameState.day})`);
    } catch (error) {
      console.error('Error transitioning to night phase:', error);
      throw error;
    }
  }

  /**
   * 处理完整的白天阶段
   * @param {object} dayConfig 白天配置 {nominations: [{nominator, nominee}], autoExecute: boolean}
   * @returns {Promise<object>} 白天结果
   */
  async processDayPhase(dayConfig = {}) {
    const nightResult = dayConfig.nightResult || {};
    await this.startDayPhase(nightResult);

    const nominations = dayConfig.nominations || [];
    let executionOccurred = false;

    // 处理所有提名
    for (const nomination of nominations) {
      try {
        // 手动处理提名，不自动调用conductVoting
        const validation = this.validateNomination(nomination.nominator, nomination.nominee);
        if (!validation.valid) {
          console.error(`Invalid nomination: ${validation.errors.join(', ')}`);
          continue;
        }

        // 记录提名
        const nominationRecord = {
          nominator: {
            id: nomination.nominator.id,
            name: nomination.nominator.name
          },
          nominee: {
            id: nomination.nominee.id,
            name: nomination.nominee.name
          },
          timestamp: Date.now()
        };

        this.nominations.push(nominationRecord);
        this.currentDayPhase.nominations.push(nominationRecord);

        // 进行投票
        const votingResult = await this.conductVoting(nomination.nominee);
        
        // 检查投票是否通过
        if (votingResult.passed) {
          // 如果配置了自动处决，则执行处决
          if (dayConfig.autoExecute !== false) {
            await this.executePlayer(nomination.nominee);
            executionOccurred = true;
            break; // 一天只能处决一个玩家
          }
        }
      } catch (error) {
        console.error(`Error processing nomination: ${error.message}`);
        // 继续处理下一个提名
      }
    }

    // 完成白天阶段
    const dayResult = await this.completeDayPhase();

    // 如果游戏已结束，不转换到夜间阶段
    if (!dayResult.gameEnded) {
      // 自动转换到夜间阶段
      await this.transitionToNightPhase();
    }

    return dayResult;
  }

  /**
   * 处理投票并决定是否处决
   * @param {Player} nominee 被提名者
   * @param {Array} playerVotes 玩家投票 [{playerId, vote: boolean}]
   * @returns {Promise<object>} {executed: boolean, votingResult: VotingResult}
   */
  async processVotingAndExecution(nominee, playerVotes = []) {
    if (!this.currentDayPhase) {
      throw new Error('No active day phase');
    }

    const gameState = this.gameStateManager.getCurrentState();
    const alivePlayers = gameState.players.filter(p => p.isAlive);

    // 收集投票
    const votes = [];
    
    for (const player of alivePlayers) {
      const playerVote = playerVotes.find(v => v.playerId === player.id);
      const vote = {
        voter: {
          id: player.id,
          name: player.name
        },
        nominee: {
          id: nominee.id,
          name: nominee.name
        },
        vote: playerVote ? playerVote.vote : false,
        timestamp: Date.now()
      };

      votes.push(vote);
    }

    this.votes.push(...votes);

    // 计算投票结果
    const votingResult = this.calculateVotingResult(votes);
    votingResult.nominee = nominee;

    console.log(`Voting for ${nominee.name}: ${votingResult.votesFor} votes (need ${votingResult.votesNeeded})`);

    // 如果投票通过，执行处决
    let executed = false;
    if (votingResult.passed) {
      await this.executePlayer(nominee);
      executed = true;
    }

    return {
      executed,
      votingResult
    };
  }

  /**
   * 验证提名的合法性
   * @param {Player} nominator 提名者
   * @param {Player} nominee 被提名者
   * @returns {object} {valid: boolean, errors: string[]}
   */
  validateNomination(nominator, nominee) {
    const errors = [];
    const gameState = this.gameStateManager.getCurrentState();

    // 检查提名者是否存在
    if (!nominator || !nominator.id) {
      errors.push('Nominator is required');
      return { valid: false, errors };
    }

    // 检查被提名者是否存在
    if (!nominee || !nominee.id) {
      errors.push('Nominee is required');
      return { valid: false, errors };
    }

    // 检查提名者是否存活
    const nominatorInGame = gameState.players.find(p => p.id === nominator.id);
    if (!nominatorInGame) {
      errors.push('Nominator not found in game');
    } else if (!nominatorInGame.isAlive) {
      errors.push('Nominator is dead');
    }

    // 检查被提名者是否存活
    const nomineeInGame = gameState.players.find(p => p.id === nominee.id);
    if (!nomineeInGame) {
      errors.push('Nominee not found in game');
    } else if (!nomineeInGame.isAlive) {
      errors.push('Nominee is dead');
    }

    // 检查是否已经提名过该玩家
    const alreadyNominated = this.nominations.some(n => n.nominee.id === nominee.id);
    if (alreadyNominated) {
      errors.push('Player has already been nominated today');
    }

    // 检查提名者是否已经提名过
    const nominatorUsed = this.nominations.some(n => n.nominator.id === nominator.id);
    if (nominatorUsed) {
      errors.push('Nominator has already used their nomination today');
    }

    return {
      valid: errors.length === 0,
      errors
    };
  }

  /**
   * 获取提名历史
   * @returns {Array} 提名历史
   */
  getNominationHistory() {
    return [...this.nominations];
  }

  /**
   * 获取投票历史
   * @returns {Array} 投票历史
   */
  getVotingHistory() {
    return [...this.votes];
  }

  /**
   * 获取当前白天阶段信息
   * @returns {object|null} 当前白天阶段
   */
  getCurrentDayPhase() {
    return this.currentDayPhase ? { ...this.currentDayPhase } : null;
  }

  /**
   * 检查是否有活跃的白天阶段
   * @returns {boolean}
   */
  hasActiveDayPhase() {
    return this.currentDayPhase !== null;
  }

  /**
   * 获取投票统计
   * @returns {object} 统计信息
   */
  getVotingStats() {
    const totalNominations = this.nominations.length;
    const totalVotes = this.votes.length;
    const executionCount = this.executedPlayer ? 1 : 0;

    const nominationsByPlayer = {};
    this.nominations.forEach(nomination => {
      const nominatorId = nomination.nominator.id;
      if (!nominationsByPlayer[nominatorId]) {
        nominationsByPlayer[nominatorId] = 0;
      }
      nominationsByPlayer[nominatorId]++;
    });

    return {
      totalNominations,
      totalVotes,
      executionCount,
      nominationsByPlayer
    };
  }

  /**
   * 清除投票历史
   */
  clearVotingHistory() {
    this.nominations = [];
    this.votes = [];
    this.executedPlayer = null;
  }

  /**
   * 检查特殊处决条件
   * @private
   * @param {Player} player 被处决的玩家
   * @returns {Array} 特殊条件列表
   */
  _checkSpecialExecutionConditions(player) {
    const conditions = [];

    // 检查是否是圣徒（Saint）
    if (player.role && player.role.id === 'saint') {
      conditions.push({
        type: 'saint_executed',
        details: {
          player: {
            id: player.id,
            name: player.name
          },
          role: player.role.name
        }
      });
    }

    // 检查是否是无神论者（Atheist）
    if (player.role && player.role.id === 'atheist') {
      conditions.push({
        type: 'atheist_executed',
        details: {
          player: {
            id: player.id,
            name: player.name
          },
          role: player.role.name
        }
      });
    }

    return conditions;
  }
}
