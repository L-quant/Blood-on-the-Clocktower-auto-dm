/**
 * 测试能力执行管理器
 */

import AbilityExecutor from '../../../src/automation/core/AbilityExecutor';
import AbilityResolver from '../../../src/automation/core/AbilityResolver';
import {
  Ability,
  AbilityContext,
  AbilityType,
  AbilityTiming,
  EffectType,
  TargetType,
  AbilityResultStatus
} from '../../../src/automation/types/AbilityTypes';
import { Team, GamePhase } from '../../../src/automation/types/GameTypes';

describe('AbilityExecutor', () => {
  let abilityExecutor;
  let mockGameStateManager;
  let mockGameState;
  let mockPlayers;

  beforeEach(() => {
    // 创建模拟玩家
    mockPlayers = [
      { id: '1', name: 'Player 1', isAlive: true, role: { id: 'washerwoman', team: Team.TOWNSFOLK } },
      { id: '2', name: 'Player 2', isAlive: true, role: { id: 'chef', team: Team.TOWNSFOLK } },
      { id: '3', name: 'Player 3', isAlive: true, role: { id: 'imp', team: Team.DEMON } },
      { id: '4', name: 'Player 4', isAlive: false, role: { id: 'ravenkeeper', team: Team.TOWNSFOLK } }
    ];

    // 创建模拟游戏状态
    mockGameState = {
      gameId: 'test-game',
      phase: GamePhase.NIGHT,
      day: 1,
      players: mockPlayers,
      deadPlayers: [mockPlayers[3]],
      abilityHistory: []
    };

    // 创建模拟游戏状态管理器
    mockGameStateManager = {
      getCurrentState: jest.fn(() => mockGameState),
      updatePlayerState: jest.fn(),
      notifyStateChange: jest.fn()
    };

    abilityExecutor = new AbilityExecutor(mockGameStateManager);
  });

  describe('executeAbilityWithTransaction', () => {
    test('应该成功执行能力并应用状态变更', async () => {
      const ability = new Ability({
        id: 'kill',
        name: 'Kill',
        type: AbilityType.KILLING,
        timing: AbilityTiming.NIGHT,
        effects: [
          {
            type: EffectType.KILL,
            target: TargetType.SINGLE_PLAYER
          }
        ]
      });

      const context = new AbilityContext({
        gameState: mockGameState,
        player: mockPlayers[2], // Imp
        targets: [mockPlayers[0]]
      });

      const result = await abilityExecutor.executeAbilityWithTransaction(ability, context);

      expect(result.isSuccess()).toBe(true);
      expect(mockPlayers[0].isAlive).toBe(false);
      expect(mockGameState.deadPlayers).toContainEqual(mockPlayers[0]);
    });

    test('失败的能力执行应该回滚状态', async () => {
      const originalAliveStatus = mockPlayers[0].isAlive;

      // 创建一个会失败的能力（中毒的玩家）
      mockPlayers[2].status = { poisoned: true };

      const ability = new Ability({
        id: 'kill',
        name: 'Kill',
        type: AbilityType.KILLING,
        timing: AbilityTiming.NIGHT,
        effects: [
          {
            type: EffectType.KILL,
            target: TargetType.SINGLE_PLAYER
          }
        ]
      });

      const context = new AbilityContext({
        gameState: mockGameState,
        player: mockPlayers[2],
        targets: [mockPlayers[0]]
      });

      const result = await abilityExecutor.executeAbilityWithTransaction(ability, context);

      expect(result.status).toBe(AbilityResultStatus.POISONED);
      expect(mockPlayers[0].isAlive).toBe(originalAliveStatus); // 状态未改变
    });

    test('应该记录执行历史', async () => {
      const ability = new Ability({
        id: 'test-ability',
        name: 'Test Ability',
        type: AbilityType.INFORMATION,
        timing: AbilityTiming.NIGHT,
        effects: []
      });

      const context = new AbilityContext({
        gameState: mockGameState,
        player: mockPlayers[0],
        targets: []
      });

      await abilityExecutor.executeAbilityWithTransaction(ability, context);

      const history = abilityExecutor.getExecutionHistory();
      expect(history.length).toBeGreaterThan(0);
      expect(history[history.length - 1].ability).toBe('test-ability');
      expect(history[history.length - 1].player).toBe('1');
    });
  });

  describe('executeBatch', () => {
    test('应该批量执行多个能力', async () => {
      const executions = [
        {
          ability: new Ability({
            id: 'protect',
            name: 'Protect',
            type: AbilityType.PROTECTION,
            timing: AbilityTiming.NIGHT,
            effects: [
              {
                type: EffectType.PROTECT,
                target: TargetType.SINGLE_PLAYER
              }
            ]
          }),
          context: new AbilityContext({
            gameState: mockGameState,
            player: mockPlayers[0],
            targets: [mockPlayers[1]]
          })
        },
        {
          ability: new Ability({
            id: 'poison',
            name: 'Poison',
            type: AbilityType.MANIPULATION,
            timing: AbilityTiming.NIGHT,
            effects: [
              {
                type: EffectType.POISON,
                target: TargetType.SINGLE_PLAYER
              }
            ]
          }),
          context: new AbilityContext({
            gameState: mockGameState,
            player: mockPlayers[2],
            targets: [mockPlayers[0]]
          })
        }
      ];

      const results = await abilityExecutor.executeBatch(executions);

      expect(results).toHaveLength(2);
      expect(results[0].isSuccess()).toBe(true);
      expect(results[1].isSuccess()).toBe(true);
      expect(mockPlayers[1].status.protected).toBe(true);
      expect(mockPlayers[0].status.poisoned).toBe(true);
    });
  });

  describe('状态变更应用', () => {
    test('应该正确应用玩家死亡状态', async () => {
      const ability = new Ability({
        id: 'kill',
        name: 'Kill',
        type: AbilityType.KILLING,
        timing: AbilityTiming.NIGHT,
        effects: [
          {
            type: EffectType.KILL,
            target: TargetType.SINGLE_PLAYER
          }
        ]
      });

      const context = new AbilityContext({
        gameState: mockGameState,
        player: mockPlayers[2],
        targets: [mockPlayers[0]]
      });

      await abilityExecutor.executeAbilityWithTransaction(ability, context);

      expect(mockPlayers[0].isAlive).toBe(false);
      expect(mockGameState.deadPlayers).toContainEqual(mockPlayers[0]);
    });

    test('应该正确应用玩家复活状态', async () => {
      const ability = new Ability({
        id: 'resurrect',
        name: 'Resurrect',
        type: AbilityType.MANIPULATION,
        timing: AbilityTiming.NIGHT,
        effects: [
          {
            type: EffectType.RESURRECT,
            target: TargetType.SINGLE_PLAYER
          }
        ]
      });

      const context = new AbilityContext({
        gameState: mockGameState,
        player: mockPlayers[0],
        targets: [mockPlayers[3]] // 死亡的玩家
      });

      await abilityExecutor.executeAbilityWithTransaction(ability, context);

      expect(mockPlayers[3].isAlive).toBe(true);
      expect(mockGameState.deadPlayers).not.toContainEqual(mockPlayers[3]);
    });

    test('应该正确应用中毒状态', async () => {
      const ability = new Ability({
        id: 'poison',
        name: 'Poison',
        type: AbilityType.MANIPULATION,
        timing: AbilityTiming.NIGHT,
        effects: [
          {
            type: EffectType.POISON,
            target: TargetType.SINGLE_PLAYER
          }
        ]
      });

      const context = new AbilityContext({
        gameState: mockGameState,
        player: mockPlayers[2],
        targets: [mockPlayers[0]]
      });

      await abilityExecutor.executeAbilityWithTransaction(ability, context);

      expect(mockPlayers[0].status.poisoned).toBe(true);
    });

    test('应该正确应用保护状态', async () => {
      const ability = new Ability({
        id: 'protect',
        name: 'Protect',
        type: AbilityType.PROTECTION,
        timing: AbilityTiming.NIGHT,
        effects: [
          {
            type: EffectType.PROTECT,
            target: TargetType.SINGLE_PLAYER
          }
        ]
      });

      const context = new AbilityContext({
        gameState: mockGameState,
        player: mockPlayers[0],
        targets: [mockPlayers[1]]
      });

      await abilityExecutor.executeAbilityWithTransaction(ability, context);

      expect(mockPlayers[1].status.protected).toBe(true);
    });
  });

  describe('validatePreconditions', () => {
    test('应该验证有效的前置条件', () => {
      const ability = new Ability({
        id: 'test-ability',
        name: 'Test Ability',
        type: AbilityType.INFORMATION,
        timing: AbilityTiming.NIGHT,
        effects: []
      });

      const context = new AbilityContext({
        gameState: mockGameState,
        player: mockPlayers[0],
        targets: [mockPlayers[1]]
      });

      const validation = abilityExecutor.validatePreconditions(ability, context);

      expect(validation.valid).toBe(true);
      expect(validation.errors).toHaveLength(0);
    });

    test('应该检测缺少玩家的情况', () => {
      const ability = new Ability({
        id: 'test-ability',
        name: 'Test Ability',
        type: AbilityType.INFORMATION,
        timing: AbilityTiming.NIGHT,
        effects: []
      });

      const context = new AbilityContext({
        gameState: mockGameState,
        player: null,
        targets: []
      });

      const validation = abilityExecutor.validatePreconditions(ability, context);

      expect(validation.valid).toBe(false);
      expect(validation.errors).toContain('Player is required');
    });

    test('应该检测无效的目标', () => {
      const ability = new Ability({
        id: 'test-ability',
        name: 'Test Ability',
        type: AbilityType.INFORMATION,
        timing: AbilityTiming.NIGHT,
        effects: []
      });

      const context = new AbilityContext({
        gameState: mockGameState,
        player: mockPlayers[0],
        targets: [null, { id: '2' }]
      });

      const validation = abilityExecutor.validatePreconditions(ability, context);

      expect(validation.valid).toBe(false);
      expect(validation.errors.some(e => e.includes('Invalid target'))).toBe(true);
    });
  });

  describe('executeWithRetry', () => {
    test('应该在失败后重试', async () => {
      let attemptCount = 0;
      
      // 模拟前两次失败，第三次成功
      const originalExecute = abilityExecutor.executeAbilityWithTransaction.bind(abilityExecutor);
      abilityExecutor.executeAbilityWithTransaction = jest.fn(async (ability, context) => {
        attemptCount++;
        if (attemptCount < 3) {
          throw new Error('Simulated failure');
        }
        return originalExecute(ability, context);
      });

      const ability = new Ability({
        id: 'test-ability',
        name: 'Test Ability',
        type: AbilityType.INFORMATION,
        timing: AbilityTiming.NIGHT,
        effects: []
      });

      const context = new AbilityContext({
        gameState: mockGameState,
        player: mockPlayers[0],
        targets: []
      });

      const result = await abilityExecutor.executeWithRetry(ability, context, 3);

      expect(attemptCount).toBe(3);
      expect(result.isSuccess()).toBe(true);
    });

    test('应该在所有重试失败后返回失败结果', async () => {
      // 模拟所有尝试都失败
      abilityExecutor.executeAbilityWithTransaction = jest.fn(async () => {
        throw new Error('Persistent failure');
      });

      const ability = new Ability({
        id: 'test-ability',
        name: 'Test Ability',
        type: AbilityType.INFORMATION,
        timing: AbilityTiming.NIGHT,
        effects: []
      });

      const context = new AbilityContext({
        gameState: mockGameState,
        player: mockPlayers[0],
        targets: []
      });

      const result = await abilityExecutor.executeWithRetry(ability, context, 3);

      expect(result.status).toBe(AbilityResultStatus.FAILED);
      expect(result.message).toContain('Failed after 3 attempts');
    });
  });

  describe('执行历史管理', () => {
    test('应该获取执行历史', async () => {
      const ability = new Ability({
        id: 'test-ability',
        name: 'Test Ability',
        type: AbilityType.INFORMATION,
        timing: AbilityTiming.NIGHT,
        effects: []
      });

      const context = new AbilityContext({
        gameState: mockGameState,
        player: mockPlayers[0],
        targets: []
      });

      await abilityExecutor.executeAbilityWithTransaction(ability, context);

      const history = abilityExecutor.getExecutionHistory();
      expect(history.length).toBeGreaterThan(0);
    });

    test('应该按玩家过滤执行历史', async () => {
      const ability = new Ability({
        id: 'test-ability',
        name: 'Test Ability',
        type: AbilityType.INFORMATION,
        timing: AbilityTiming.NIGHT,
        effects: []
      });

      await abilityExecutor.executeAbilityWithTransaction(ability, new AbilityContext({
        gameState: mockGameState,
        player: mockPlayers[0],
        targets: []
      }));

      await abilityExecutor.executeAbilityWithTransaction(ability, new AbilityContext({
        gameState: mockGameState,
        player: mockPlayers[1],
        targets: []
      }));

      const history = abilityExecutor.getExecutionHistory({ playerId: '1' });
      expect(history.every(r => r.player === '1')).toBe(true);
    });

    test('应该获取最近的执行记录', async () => {
      const ability = new Ability({
        id: 'test-ability',
        name: 'Test Ability',
        type: AbilityType.INFORMATION,
        timing: AbilityTiming.NIGHT,
        effects: []
      });

      const context = new AbilityContext({
        gameState: mockGameState,
        player: mockPlayers[0],
        targets: []
      });

      // 执行多次
      for (let i = 0; i < 5; i++) {
        await abilityExecutor.executeAbilityWithTransaction(ability, context);
      }

      const recent = abilityExecutor.getRecentExecutions(3);
      expect(recent).toHaveLength(3);
    });

    test('应该清除执行历史', async () => {
      const ability = new Ability({
        id: 'test-ability',
        name: 'Test Ability',
        type: AbilityType.INFORMATION,
        timing: AbilityTiming.NIGHT,
        effects: []
      });

      const context = new AbilityContext({
        gameState: mockGameState,
        player: mockPlayers[0],
        targets: []
      });

      await abilityExecutor.executeAbilityWithTransaction(ability, context);
      
      abilityExecutor.clearExecutionHistory();
      
      const history = abilityExecutor.getExecutionHistory();
      expect(history).toHaveLength(0);
    });
  });

  describe('getExecutionStats', () => {
    test('应该生成正确的统计信息', async () => {
      const ability1 = new Ability({
        id: 'ability-1',
        name: 'Ability 1',
        type: AbilityType.INFORMATION,
        timing: AbilityTiming.NIGHT,
        effects: []
      });

      const ability2 = new Ability({
        id: 'ability-2',
        name: 'Ability 2',
        type: AbilityType.INFORMATION,
        timing: AbilityTiming.NIGHT,
        effects: []
      });

      const context = new AbilityContext({
        gameState: mockGameState,
        player: mockPlayers[0],
        targets: []
      });

      // 执行多次
      await abilityExecutor.executeAbilityWithTransaction(ability1, context);
      await abilityExecutor.executeAbilityWithTransaction(ability1, context);
      await abilityExecutor.executeAbilityWithTransaction(ability2, context);

      const stats = abilityExecutor.getExecutionStats();

      expect(stats.total).toBe(3);
      expect(stats.successful).toBe(3);
      expect(stats.failed).toBe(0);
      expect(stats.abilityUsage['ability-1'].total).toBe(2);
      expect(stats.abilityUsage['ability-2'].total).toBe(1);
    });
  });
});
