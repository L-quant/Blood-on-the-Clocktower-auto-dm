/**
 * 测试胜负判断器
 */

import VictoryConditionChecker from '../../../src/automation/core/VictoryConditionChecker';
import { GamePhase, Team } from '../../../src/automation/types/GameTypes';

describe('VictoryConditionChecker', () => {
  let victoryChecker;
  let mockGameStateManager;
  let mockGameState;
  let mockPlayers;

  beforeEach(() => {
    // 创建模拟玩家
    mockPlayers = [
      { 
        id: '1', 
        name: 'Player 1', 
        isAlive: true,
        isEvil: false,
        role: { id: 'washerwoman', name: 'Washerwoman', team: Team.TOWNSFOLK }
      },
      { 
        id: '2', 
        name: 'Player 2', 
        isAlive: true,
        isEvil: true,
        role: { id: 'imp', name: 'Imp', team: Team.DEMON }
      },
      { 
        id: '3', 
        name: 'Player 3', 
        isAlive: true,
        isEvil: true,
        role: { id: 'poisoner', name: 'Poisoner', team: Team.MINION }
      },
      { 
        id: '4', 
        name: 'Player 4', 
        isAlive: true,
        isEvil: false,
        role: { id: 'monk', name: 'Monk', team: Team.TOWNSFOLK }
      },
      { 
        id: '5', 
        name: 'Player 5', 
        isAlive: true,
        isEvil: false,
        role: { id: 'soldier', name: 'Soldier', team: Team.TOWNSFOLK }
      }
    ];

    // 创建模拟游戏状态
    mockGameState = {
      gameId: 'test-game',
      phase: GamePhase.DAY,
      day: 2,
      players: mockPlayers
    };

    // 创建模拟游戏状态管理器
    mockGameStateManager = {
      getCurrentState: jest.fn(() => mockGameState),
      transitionToPhase: jest.fn(async (phase) => {
        mockGameState.phase = phase;
      })
    };

    victoryChecker = new VictoryConditionChecker(mockGameStateManager);
  });

  describe('checkGoodVictory', () => {
    test('所有恶魔死亡时好人应该胜利', () => {
      // 杀死恶魔
      mockPlayers[1].isAlive = false; // Imp死亡

      const result = victoryChecker.checkGoodVictory(mockGameState);

      expect(result.victory).toBe(true);
      expect(result.reason).toContain('demons are dead');
    });

    test('恶魔存活时好人不应该胜利', () => {
      const result = victoryChecker.checkGoodVictory(mockGameState);

      expect(result.victory).toBe(false);
      expect(result.reason).toContain('Demons still alive');
    });
  });

  describe('checkEvilVictory', () => {
    test('只剩2个玩家时恶人应该胜利', () => {
      // 只留下2个玩家
      mockPlayers[0].isAlive = false;
      mockPlayers[2].isAlive = false;
      mockPlayers[3].isAlive = false;
      // 剩下: Player 2 (evil) 和 Player 5 (good)

      const result = victoryChecker.checkEvilVictory(mockGameState);

      expect(result.victory).toBe(true);
      expect(result.reason).toContain('Only 2 players remain');
    });

    test('恶人数量等于好人时恶人应该胜利', () => {
      // 杀死一些好人，使恶人和好人数量相等
      mockPlayers[0].isAlive = false; // Good
      mockPlayers[4].isAlive = false; // Good
      // 剩下: 1 good, 2 evil

      const result = victoryChecker.checkEvilVictory(mockGameState);

      expect(result.victory).toBe(true);
      expect(result.reason).toContain('equal or outnumber');
    });

    test('好人多于恶人时恶人不应该胜利', () => {
      const result = victoryChecker.checkEvilVictory(mockGameState);

      expect(result.victory).toBe(false);
      expect(result.reason).toContain('outnumber evil');
    });
  });

  describe('checkVictoryConditions', () => {
    test('应该检测好人胜利', () => {
      mockPlayers[1].isAlive = false; // 恶魔死亡

      const result = victoryChecker.checkVictoryConditions(mockGameState);

      expect(result.winner).toBe('good');
      expect(result.ended).toBe(true);
    });

    test('应该检测恶人胜利', () => {
      // 只剩2个玩家
      mockPlayers[0].isAlive = false;
      mockPlayers[2].isAlive = false;
      mockPlayers[3].isAlive = false;

      const result = victoryChecker.checkVictoryConditions(mockGameState);

      expect(result.winner).toBe('evil');
      expect(result.ended).toBe(true);
    });

    test('游戏未结束时应该返回null winner', () => {
      const result = victoryChecker.checkVictoryConditions(mockGameState);

      expect(result.winner).toBeNull();
      expect(result.ended).toBe(false);
    });

    test('应该使用当前游戏状态如果没有提供', () => {
      mockPlayers[1].isAlive = false;

      const result = victoryChecker.checkVictoryConditions();

      expect(result.winner).toBe('good');
      expect(mockGameStateManager.getCurrentState).toHaveBeenCalled();
    });
  });

  describe('isGameEnded', () => {
    test('游戏结束时应该返回true', () => {
      mockPlayers[1].isAlive = false; // 恶魔死亡

      const ended = victoryChecker.isGameEnded(mockGameState);

      expect(ended).toBe(true);
    });

    test('游戏未结束时应该返回false', () => {
      const ended = victoryChecker.isGameEnded(mockGameState);

      expect(ended).toBe(false);
    });
  });

  describe('generateGameReport', () => {
    test('应该生成完整的游戏报告', () => {
      const outcome = {
        winner: 'good',
        reason: 'All demons are dead'
      };

      const report = victoryChecker.generateGameReport(mockGameState, outcome);

      expect(report.gameId).toBe('test-game');
      expect(report.winner).toBe('good');
      expect(report.reason).toBe('All demons are dead');
      expect(report.duration.days).toBe(2);
      expect(report.players.total).toBe(5);
      expect(report.teams.good).toBeDefined();
      expect(report.teams.evil).toBeDefined();
      expect(report.playerDetails).toHaveLength(5);
    });

    test('应该包含玩家详细信息', () => {
      const outcome = { winner: 'good', reason: 'Test' };

      const report = victoryChecker.generateGameReport(mockGameState, outcome);

      expect(report.playerDetails[0]).toHaveProperty('id');
      expect(report.playerDetails[0]).toHaveProperty('name');
      expect(report.playerDetails[0]).toHaveProperty('role');
      expect(report.playerDetails[0]).toHaveProperty('team');
      expect(report.playerDetails[0]).toHaveProperty('isAlive');
    });

    test('应该正确统计队伍信息', () => {
      mockPlayers[0].isAlive = false; // Good死亡
      mockPlayers[1].isAlive = false; // Evil死亡

      const outcome = { winner: 'good', reason: 'Test' };
      const report = victoryChecker.generateGameReport(mockGameState, outcome);

      expect(report.teams.good.total).toBe(3);
      expect(report.teams.good.alive).toBe(2);
      expect(report.teams.good.dead).toBe(1);
      expect(report.teams.evil.total).toBe(2);
      expect(report.teams.evil.alive).toBe(1);
      expect(report.teams.evil.dead).toBe(1);
    });
  });

  describe('handleSpecialVictoryConditions', () => {
    test('应该处理圣徒被处决的特殊条件', () => {
      const conditions = [
        {
          type: 'saint_executed',
          details: { player: 'Player 1' }
        }
      ];

      const result = victoryChecker.handleSpecialVictoryConditions(conditions);

      expect(result.winner).toBe('evil');
      expect(result.reason).toContain('Saint was executed');
      expect(result.ended).toBe(true);
      expect(result.special).toBe(true);
    });

    test('应该处理无神论者被处决的特殊条件', () => {
      const conditions = [
        {
          type: 'atheist_executed',
          details: { player: 'Player 2' }
        }
      ];

      const result = victoryChecker.handleSpecialVictoryConditions(conditions);

      expect(result.winner).toBe('evil');
      expect(result.reason).toContain('Atheist was executed');
      expect(result.ended).toBe(true);
    });

    test('没有特殊条件时应该返回null winner', () => {
      const conditions = [];

      const result = victoryChecker.handleSpecialVictoryConditions(conditions);

      expect(result.winner).toBeNull();
      expect(result.ended).toBe(false);
    });
  });

  describe('checkAndEndGame', () => {
    test('游戏结束时应该转换到ENDED阶段', async () => {
      mockPlayers[1].isAlive = false; // 恶魔死亡

      const result = await victoryChecker.checkAndEndGame();

      expect(result).toBeDefined();
      expect(result.winner).toBe('good');
      expect(result.report).toBeDefined();
      expect(mockGameStateManager.transitionToPhase).toHaveBeenCalledWith(GamePhase.ENDED);
    });

    test('游戏未结束时应该返回null', async () => {
      const result = await victoryChecker.checkAndEndGame();

      expect(result).toBeNull();
      expect(mockGameStateManager.transitionToPhase).not.toHaveBeenCalled();
    });

    test('游戏已经结束时不应该再次检查', async () => {
      mockGameState.phase = GamePhase.ENDED;

      const result = await victoryChecker.checkAndEndGame();

      expect(result).toBeNull();
      expect(mockGameStateManager.transitionToPhase).not.toHaveBeenCalled();
    });

    test('设置阶段时不应该检查', async () => {
      mockGameState.phase = GamePhase.SETUP;

      const result = await victoryChecker.checkAndEndGame();

      expect(result).toBeNull();
    });
  });

  describe('getGameStatusSummary', () => {
    test('应该返回游戏状态摘要', () => {
      const summary = victoryChecker.getGameStatusSummary();

      expect(summary.phase).toBe(GamePhase.DAY);
      expect(summary.day).toBe(2);
      expect(summary.totalPlayers).toBe(5);
      expect(summary.alivePlayers).toBe(5);
      expect(summary.aliveGood).toBe(3);
      expect(summary.aliveEvil).toBe(2);
      expect(summary.isEnded).toBe(false);
    });

    test('应该反映玩家死亡', () => {
      mockPlayers[0].isAlive = false;
      mockPlayers[1].isAlive = false;

      const summary = victoryChecker.getGameStatusSummary();

      expect(summary.alivePlayers).toBe(3);
      expect(summary.aliveGood).toBe(2);
      expect(summary.aliveEvil).toBe(1);
    });
  });

  describe('validateVictoryConditions', () => {
    test('应该验证有效的游戏状态', () => {
      const validation = victoryChecker.validateVictoryConditions(mockGameState);

      expect(validation.valid).toBe(true);
      expect(validation.errors).toHaveLength(0);
    });

    test('应该检测没有玩家存活', () => {
      mockPlayers.forEach(p => p.isAlive = false);

      const validation = victoryChecker.validateVictoryConditions(mockGameState);

      expect(validation.valid).toBe(false);
      expect(validation.errors.some(e => e.includes('No players alive'))).toBe(true);
    });

    test('应该检测缺少角色分配', () => {
      mockPlayers[0].role = null;

      const validation = victoryChecker.validateVictoryConditions(mockGameState);

      expect(validation.valid).toBe(false);
      expect(validation.errors.some(e => e.includes('no role assigned'))).toBe(true);
    });
  });

  describe('predictGameOutcome', () => {
    test('所有恶魔死亡时应该预测好人必胜', () => {
      mockPlayers[1].isAlive = false;

      const prediction = victoryChecker.predictGameOutcome(mockGameState);

      expect(prediction.prediction).toBe('good');
      expect(prediction.confidence).toBe(1.0);
      expect(prediction.reasoning).toContain('demons are dead');
    });

    test('恶人等于好人时应该预测恶人必胜', () => {
      mockPlayers[0].isAlive = false;
      mockPlayers[4].isAlive = false;

      const prediction = victoryChecker.predictGameOutcome(mockGameState);

      expect(prediction.prediction).toBe('evil');
      expect(prediction.confidence).toBe(1.0);
    });

    test('好人远多于恶人时应该预测好人可能胜', () => {
      const prediction = victoryChecker.predictGameOutcome(mockGameState);

      expect(prediction.prediction).toBe('good');
      expect(prediction.confidence).toBeGreaterThan(0);
    });

    test('应该包含当前状态信息', () => {
      const prediction = victoryChecker.predictGameOutcome(mockGameState);

      expect(prediction.currentState).toBeDefined();
      expect(prediction.currentState.aliveGood).toBe(3);
      expect(prediction.currentState.aliveEvil).toBe(2);
    });
  });
});
