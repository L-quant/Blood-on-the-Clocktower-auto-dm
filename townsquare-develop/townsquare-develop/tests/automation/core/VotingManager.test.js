/**
 * 测试投票管理器
 */

import VotingManager from '../../../src/automation/core/VotingManager';
import { GamePhase, Team } from '../../../src/automation/types/GameTypes';

describe('VotingManager', () => {
  let votingManager;
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
        role: { id: 'washerwoman', name: 'Washerwoman', team: Team.TOWNSFOLK }
      },
      { 
        id: '2', 
        name: 'Player 2', 
        isAlive: true,
        role: { id: 'imp', name: 'Imp', team: Team.DEMON }
      },
      { 
        id: '3', 
        name: 'Player 3', 
        isAlive: true,
        role: { id: 'poisoner', name: 'Poisoner', team: Team.MINION }
      },
      { 
        id: '4', 
        name: 'Player 4', 
        isAlive: true,
        role: { id: 'monk', name: 'Monk', team: Team.TOWNSFOLK }
      },
      { 
        id: '5', 
        name: 'Player 5', 
        isAlive: false,
        role: { id: 'ravenkeeper', name: 'Ravenkeeper', team: Team.TOWNSFOLK }
      }
    ];

    // 创建模拟游戏状态
    mockGameState = {
      gameId: 'test-game',
      phase: GamePhase.DAY,
      day: 1,
      players: mockPlayers
    };

    // 创建模拟游戏状态管理器
    mockGameStateManager = {
      getCurrentState: jest.fn(() => mockGameState),
      updatePlayerState: jest.fn((playerId, updates) => {
        const player = mockGameState.players.find(p => p.id === playerId);
        if (player) {
          Object.assign(player, updates);
        }
      })
    };

    votingManager = new VotingManager(mockGameStateManager);
  });

  describe('startDayPhase', () => {
    test('应该成功开始白天阶段', async () => {
      const nightResult = {
        deaths: [{ id: '5', name: 'Player 5' }]
      };

      await votingManager.startDayPhase(nightResult);

      expect(votingManager.hasActiveDayPhase()).toBe(true);
      const dayPhase = votingManager.getCurrentDayPhase();
      expect(dayPhase).toBeDefined();
      expect(dayPhase.day).toBe(1);
      expect(dayPhase.nightResult).toEqual(nightResult);
    });

    test('从非白天阶段开始应该抛出错误', async () => {
      mockGameState.phase = GamePhase.NIGHT;

      await expect(votingManager.startDayPhase()).rejects.toThrow('Cannot start day phase');
    });

    test('应该清空之前的提名和投票', async () => {
      // 先添加一些提名和投票
      votingManager.nominations = [{ nominator: {}, nominee: {} }];
      votingManager.votes = [{ voter: {}, vote: true }];

      await votingManager.startDayPhase();

      expect(votingManager.getNominationHistory()).toHaveLength(0);
      expect(votingManager.getVotingHistory()).toHaveLength(0);
    });
  });

  describe('handleNomination', () => {
    beforeEach(async () => {
      await votingManager.startDayPhase();
    });

    test('应该成功处理有效的提名', async () => {
      const nominator = mockPlayers[0];
      const nominee = mockPlayers[1];

      await votingManager.handleNomination(nominator, nominee);

      const nominations = votingManager.getNominationHistory();
      expect(nominations).toHaveLength(1);
      expect(nominations[0].nominator.id).toBe(nominator.id);
      expect(nominations[0].nominee.id).toBe(nominee.id);
    });

    test('死亡玩家不能提名', async () => {
      const nominator = mockPlayers[4]; // Dead player
      const nominee = mockPlayers[1];

      await expect(votingManager.handleNomination(nominator, nominee))
        .rejects.toThrow('Nominator is dead');
    });

    test('不能提名死亡玩家', async () => {
      const nominator = mockPlayers[0];
      const nominee = mockPlayers[4]; // Dead player

      await expect(votingManager.handleNomination(nominator, nominee))
        .rejects.toThrow('Nominee is dead');
    });

    test('不能重复提名同一玩家', async () => {
      const nominator1 = mockPlayers[0];
      const nominator2 = mockPlayers[2];
      const nominee = mockPlayers[1];

      await votingManager.handleNomination(nominator1, nominee);

      await expect(votingManager.handleNomination(nominator2, nominee))
        .rejects.toThrow('already been nominated');
    });

    test('同一玩家不能提名两次', async () => {
      const nominator = mockPlayers[0];
      const nominee1 = mockPlayers[1];
      const nominee2 = mockPlayers[2];

      await votingManager.handleNomination(nominator, nominee1);

      await expect(votingManager.handleNomination(nominator, nominee2))
        .rejects.toThrow('already used their nomination');
    });
  });

  describe('conductVoting', () => {
    beforeEach(async () => {
      await votingManager.startDayPhase();
    });

    test('应该收集所有存活玩家的投票', async () => {
      const nominee = mockPlayers[1];

      const result = await votingManager.conductVoting(nominee);

      expect(result).toBeDefined();
      expect(result.totalVoters).toBe(4); // 4个存活玩家
      expect(result.votes).toHaveLength(4);
    });

    test('应该返回投票结果', async () => {
      const nominee = mockPlayers[1];

      const result = await votingManager.conductVoting(nominee);

      expect(result.votesFor).toBeDefined();
      expect(result.votesAgainst).toBeDefined();
      expect(result.votesNeeded).toBeDefined();
      expect(result.passed).toBeDefined();
    });

    test('没有活跃白天阶段时应该抛出错误', async () => {
      votingManager.currentDayPhase = null;

      await expect(votingManager.conductVoting(mockPlayers[1]))
        .rejects.toThrow('No active day phase');
    });
  });

  describe('calculateVotingResult', () => {
    test('应该正确计算投票结果', () => {
      const votes = [
        { voter: { id: '1' }, vote: true },
        { voter: { id: '2' }, vote: true },
        { voter: { id: '3' }, vote: false },
        { voter: { id: '4' }, vote: true }
      ];

      const result = votingManager.calculateVotingResult(votes);

      expect(result.votesFor).toBe(3);
      expect(result.votesAgainst).toBe(1);
      expect(result.totalVoters).toBe(4);
      expect(result.votesNeeded).toBe(3); // 超过半数：floor(4/2) + 1 = 3
      expect(result.passed).toBe(true);
    });

    test('应该处理投票修改器', () => {
      // 给玩家1添加投票修改器（+1票）
      mockPlayers[0].status = { voteModifier: 1 };

      const votes = [
        { voter: { id: '1' }, vote: true }, // 2票
        { voter: { id: '2' }, vote: false },
        { voter: { id: '3' }, vote: false },
        { voter: { id: '4' }, vote: false }
      ];

      const result = votingManager.calculateVotingResult(votes);

      expect(result.votesFor).toBe(2); // 1个玩家投票，但有+1修改器
      expect(result.votesNeeded).toBe(3);
      expect(result.passed).toBe(false);
    });

    test('票数不足时应该不通过', () => {
      const votes = [
        { voter: { id: '1' }, vote: true },
        { voter: { id: '2' }, vote: false },
        { voter: { id: '3' }, vote: false },
        { voter: { id: '4' }, vote: false }
      ];

      const result = votingManager.calculateVotingResult(votes);

      expect(result.votesFor).toBe(1);
      expect(result.votesNeeded).toBe(3);
      expect(result.passed).toBe(false);
    });
  });

  describe('executePlayer', () => {
    beforeEach(async () => {
      await votingManager.startDayPhase();
    });

    test('应该成功处决玩家', async () => {
      const player = mockPlayers[1];

      await votingManager.executePlayer(player);

      expect(mockGameStateManager.updatePlayerState).toHaveBeenCalledWith(
        player.id,
        { isAlive: false }
      );
      expect(votingManager.executedPlayer).toEqual(player);
    });

    test('没有活跃白天阶段时应该抛出错误', async () => {
      votingManager.currentDayPhase = null;

      await expect(votingManager.executePlayer(mockPlayers[1]))
        .rejects.toThrow('No active day phase');
    });
  });

  describe('completeDayPhase', () => {
    test('应该成功完成白天阶段', async () => {
      await votingManager.startDayPhase();

      const result = await votingManager.completeDayPhase();

      expect(result).toBeDefined();
      expect(result.day).toBe(1);
      expect(result.duration).toBeGreaterThanOrEqual(0);
      expect(votingManager.hasActiveDayPhase()).toBe(false);
    });

    test('没有活跃白天阶段时应该抛出错误', async () => {
      await expect(votingManager.completeDayPhase())
        .rejects.toThrow('No active day phase');
    });

    test('应该包含处决信息', async () => {
      await votingManager.startDayPhase();
      await votingManager.executePlayer(mockPlayers[1]);

      const result = await votingManager.completeDayPhase();

      expect(result.executedPlayer).toEqual(mockPlayers[1]);
    });
  });

  describe('validateNomination', () => {
    test('应该验证有效的提名', () => {
      const nominator = mockPlayers[0];
      const nominee = mockPlayers[1];

      const validation = votingManager.validateNomination(nominator, nominee);

      expect(validation.valid).toBe(true);
      expect(validation.errors).toHaveLength(0);
    });

    test('应该检测缺少提名者', () => {
      const validation = votingManager.validateNomination(null, mockPlayers[1]);

      expect(validation.valid).toBe(false);
      expect(validation.errors.some(e => e.includes('Nominator is required'))).toBe(true);
    });

    test('应该检测缺少被提名者', () => {
      const validation = votingManager.validateNomination(mockPlayers[0], null);

      expect(validation.valid).toBe(false);
      expect(validation.errors.some(e => e.includes('Nominee is required'))).toBe(true);
    });

    test('应该检测死亡的提名者', () => {
      const validation = votingManager.validateNomination(mockPlayers[4], mockPlayers[1]);

      expect(validation.valid).toBe(false);
      expect(validation.errors.some(e => e.includes('Nominator is dead'))).toBe(true);
    });

    test('应该检测死亡的被提名者', () => {
      const validation = votingManager.validateNomination(mockPlayers[0], mockPlayers[4]);

      expect(validation.valid).toBe(false);
      expect(validation.errors.some(e => e.includes('Nominee is dead'))).toBe(true);
    });
  });

  describe('历史和统计', () => {
    beforeEach(async () => {
      await votingManager.startDayPhase();
    });

    test('应该记录提名历史', async () => {
      await votingManager.handleNomination(mockPlayers[0], mockPlayers[1]);

      const history = votingManager.getNominationHistory();
      expect(history).toHaveLength(1);
      expect(history[0].nominator.id).toBe(mockPlayers[0].id);
    });

    test('应该记录投票历史', async () => {
      await votingManager.handleNomination(mockPlayers[0], mockPlayers[1]);

      const history = votingManager.getVotingHistory();
      expect(history.length).toBeGreaterThan(0);
    });

    test('应该生成投票统计', async () => {
      await votingManager.handleNomination(mockPlayers[0], mockPlayers[1]);
      await votingManager.executePlayer(mockPlayers[1]);

      const stats = votingManager.getVotingStats();

      expect(stats.totalNominations).toBe(1);
      expect(stats.executionCount).toBe(1);
      expect(stats.nominationsByPlayer[mockPlayers[0].id]).toBe(1);
    });

    test('应该清除投票历史', () => {
      votingManager.nominations = [{ nominator: {}, nominee: {} }];
      votingManager.votes = [{ voter: {}, vote: true }];
      votingManager.executedPlayer = mockPlayers[1];

      votingManager.clearVotingHistory();

      expect(votingManager.getNominationHistory()).toHaveLength(0);
      expect(votingManager.getVotingHistory()).toHaveLength(0);
      expect(votingManager.executedPlayer).toBeNull();
    });
  });

  describe('阶段转换', () => {
    test('应该成功转换到夜间阶段', async () => {
      mockGameStateManager.transitionToPhase = jest.fn(async (phase) => {
        mockGameState.phase = phase;
      });

      await votingManager.transitionToNightPhase();

      expect(mockGameStateManager.transitionToPhase).toHaveBeenCalledWith(GamePhase.NIGHT);
    });

    test('从非白天阶段转换应该抛出错误', async () => {
      mockGameState.phase = GamePhase.NIGHT;

      await expect(votingManager.transitionToNightPhase())
        .rejects.toThrow('Cannot transition to night');
    });
  });

  describe('processDayPhase', () => {
    beforeEach(() => {
      mockGameStateManager.transitionToPhase = jest.fn(async (phase) => {
        mockGameState.phase = phase;
      });
    });

    test('应该处理完整的白天阶段', async () => {
      const dayConfig = {
        nightResult: { deaths: [] },
        nominations: [
          { nominator: mockPlayers[0], nominee: mockPlayers[1] }
        ],
        autoExecute: false
      };

      const result = await votingManager.processDayPhase(dayConfig);

      expect(result).toBeDefined();
      expect(result.day).toBe(1);
      expect(mockGameStateManager.transitionToPhase).toHaveBeenCalledWith(GamePhase.NIGHT);
    });

    test('应该在投票通过时自动处决', async () => {
      // 模拟投票通过
      votingManager.calculateVotingResult = jest.fn(() => ({
        votesFor: 3,
        votesAgainst: 1,
        votesNeeded: 3,
        totalVoters: 4,
        passed: true,
        votes: []
      }));

      const dayConfig = {
        nominations: [
          { nominator: mockPlayers[0], nominee: mockPlayers[1] }
        ],
        autoExecute: true
      };

      const result = await votingManager.processDayPhase(dayConfig);

      expect(result.executedPlayer).toBeDefined();
      expect(result.executedPlayer.id).toBe(mockPlayers[1].id);
    });

    test('应该在无处决时转换到夜间', async () => {
      const dayConfig = {
        nominations: [],
        autoExecute: false
      };

      await votingManager.processDayPhase(dayConfig);

      expect(mockGameStateManager.transitionToPhase).toHaveBeenCalledWith(GamePhase.NIGHT);
    });
  });

  describe('processVotingAndExecution', () => {
    beforeEach(async () => {
      await votingManager.startDayPhase();
    });

    test('应该处理投票并根据结果决定是否处决', async () => {
      const nominee = mockPlayers[1];
      const playerVotes = [
        { playerId: '1', vote: true },
        { playerId: '2', vote: true },
        { playerId: '3', vote: true },
        { playerId: '4', vote: false }
      ];

      const result = await votingManager.processVotingAndExecution(nominee, playerVotes);

      expect(result.executed).toBe(true);
      expect(result.votingResult.passed).toBe(true);
      expect(result.votingResult.votesFor).toBe(3);
    });

    test('投票未通过时不应该处决', async () => {
      const nominee = mockPlayers[1];
      const playerVotes = [
        { playerId: '1', vote: true },
        { playerId: '2', vote: false },
        { playerId: '3', vote: false },
        { playerId: '4', vote: false }
      ];

      const result = await votingManager.processVotingAndExecution(nominee, playerVotes);

      expect(result.executed).toBe(false);
      expect(result.votingResult.passed).toBe(false);
      expect(result.votingResult.votesFor).toBe(1);
    });

    test('没有活跃白天阶段时应该抛出错误', async () => {
      votingManager.currentDayPhase = null;

      await expect(votingManager.processVotingAndExecution(mockPlayers[1], []))
        .rejects.toThrow('No active day phase');
    });
  });
});

  // 集成测试：胜负判断
  describe('Victory Condition Integration', () => {
    let mockVictoryChecker;
    let localMockPlayers;
    let localMockGameState;
    let localMockGameStateManager;
    let localVotingManager;

    beforeEach(() => {
      // 重新初始化所有mock对象
      localMockPlayers = [
        { 
          id: '1', 
          name: 'Player 1', 
          isAlive: true,
          role: { id: 'washerwoman', name: 'Washerwoman', team: Team.TOWNSFOLK }
        },
        { 
          id: '2', 
          name: 'Player 2', 
          isAlive: true,
          role: { id: 'imp', name: 'Imp', team: Team.DEMON }
        },
        { 
          id: '3', 
          name: 'Player 3', 
          isAlive: true,
          role: { id: 'poisoner', name: 'Poisoner', team: Team.MINION }
        },
        { 
          id: '4', 
          name: 'Player 4', 
          isAlive: true,
          role: { id: 'monk', name: 'Monk', team: Team.TOWNSFOLK }
        }
      ];

      localMockGameState = {
        gameId: 'test-game',
        phase: GamePhase.DAY,
        day: 1,
        players: localMockPlayers
      };

      localMockGameStateManager = {
        getCurrentState: jest.fn(() => localMockGameState),
        updatePlayerState: jest.fn((playerId, updates) => {
          const player = localMockGameState.players.find(p => p.id === playerId);
          if (player) {
            Object.assign(player, updates);
          }
        }),
        transitionToPhase: jest.fn()
      };

      mockVictoryChecker = {
        checkAndEndGame: jest.fn(),
        handleSpecialVictoryConditions: jest.fn()
      };

      localVotingManager = new VotingManager(localMockGameStateManager, mockVictoryChecker);
    });

    test('should check victory conditions after day phase completes', async () => {
      mockVictoryChecker.checkAndEndGame.mockResolvedValue(null);

      await localVotingManager.startDayPhase({});
      const result = await localVotingManager.completeDayPhase();

      expect(mockVictoryChecker.checkAndEndGame).toHaveBeenCalled();
      expect(result.gameEnded).toBeUndefined();
    });

    test('should mark game as ended when victory condition is met', async () => {
      const victoryResult = {
        winner: 'good',
        reason: 'All demons are dead',
        ended: true,
        report: { gameId: 'test-game' }
      };

      mockVictoryChecker.checkAndEndGame.mockResolvedValue(victoryResult);

      await localVotingManager.startDayPhase({});
      const result = await localVotingManager.completeDayPhase();

      expect(result.gameEnded).toBe(true);
      expect(result.victoryResult).toEqual(victoryResult);
    });

    test('should not transition to night phase when game ends', async () => {
      const victoryResult = {
        winner: 'evil',
        reason: 'Evil wins',
        ended: true
      };

      mockVictoryChecker.checkAndEndGame.mockResolvedValue(victoryResult);

      await localVotingManager.processDayPhase({
        nightResult: {},
        nominations: [],
        autoExecute: false
      });

      // transitionToPhase 不应该被调用转换到NIGHT
      const calls = localMockGameStateManager.transitionToPhase.mock.calls;
      const nightTransition = calls.find(call => call[0] === GamePhase.NIGHT);
      expect(nightTransition).toBeUndefined();
    });

    test('should check special victory conditions when executing player', async () => {
      const saintPlayer = {
        id: '1',
        name: 'Saint Player',
        isAlive: true,
        role: { id: 'saint', name: 'Saint' }
      };

      localMockGameState.players = [saintPlayer];
      mockVictoryChecker.handleSpecialVictoryConditions.mockReturnValue({
        winner: 'evil',
        reason: 'Saint was executed',
        ended: true,
        special: true
      });

      await localVotingManager.startDayPhase({});
      await localVotingManager.executePlayer(saintPlayer);

      expect(mockVictoryChecker.handleSpecialVictoryConditions).toHaveBeenCalled();
      expect(localMockGameStateManager.transitionToPhase).toHaveBeenCalledWith(GamePhase.ENDED);
    });

    test('should handle atheist execution special condition', async () => {
      const atheistPlayer = {
        id: '2',
        name: 'Atheist Player',
        isAlive: true,
        role: { id: 'atheist', name: 'Atheist' }
      };

      localMockGameState.players = [atheistPlayer];
      mockVictoryChecker.handleSpecialVictoryConditions.mockReturnValue({
        winner: 'evil',
        reason: 'Atheist was executed',
        ended: true,
        special: true
      });

      await localVotingManager.startDayPhase({});
      await localVotingManager.executePlayer(atheistPlayer);

      expect(mockVictoryChecker.handleSpecialVictoryConditions).toHaveBeenCalled();
      const calls = mockVictoryChecker.handleSpecialVictoryConditions.mock.calls[0][0];
      expect(calls[0].type).toBe('atheist_executed');
    });

    test('should not check special conditions for normal roles', async () => {
      const normalPlayer = {
        id: '3',
        name: 'Normal Player',
        isAlive: true,
        role: { id: 'washerwoman', name: 'Washerwoman' }
      };

      localMockGameState.players = [normalPlayer];

      await localVotingManager.startDayPhase({});
      await localVotingManager.executePlayer(normalPlayer);

      // 普通角色不应该触发特殊胜利条件检查
      expect(mockVictoryChecker.handleSpecialVictoryConditions).not.toHaveBeenCalled();
      expect(localMockGameStateManager.transitionToPhase).not.toHaveBeenCalledWith(GamePhase.ENDED);
    });

    test('should handle day phase without victory checker', async () => {
      const managerWithoutChecker = new VotingManager(localMockGameStateManager, null);

      await managerWithoutChecker.startDayPhase({});
      const result = await managerWithoutChecker.completeDayPhase();

      expect(result.gameEnded).toBeUndefined();
      expect(result.victoryResult).toBeUndefined();
    });

    test('should continue to night phase when no victory condition is met', async () => {
      mockVictoryChecker.checkAndEndGame.mockResolvedValue(null);

      await localVotingManager.processDayPhase({
        nightResult: {},
        nominations: [],
        autoExecute: false
      });

      expect(localMockGameStateManager.transitionToPhase).toHaveBeenCalledWith(GamePhase.NIGHT);
    });
  });
