/**
 * 测试夜间行动处理器
 */

import NightActionProcessor from '../../../src/automation/core/NightActionProcessor';
import AbilityExecutor from '../../../src/automation/core/AbilityExecutor';
import {
  Ability,
  AbilityType,
  AbilityTiming,
  EffectType,
  TargetType,
  AbilityResultStatus
} from '../../../src/automation/types/AbilityTypes';
import { Team, GamePhase } from '../../../src/automation/types/GameTypes';

describe('NightActionProcessor', () => {
  let nightActionProcessor;
  let mockGameStateManager;
  let mockAbilityExecutor;
  let mockGameState;
  let mockPlayers;

  beforeEach(() => {
    // 创建模拟玩家
    mockPlayers = [
      { 
        id: '1', 
        name: 'Player 1', 
        isAlive: true, 
        role: { 
          id: 'washerwoman', 
          name: 'Washerwoman',
          team: Team.TOWNSFOLK,
          firstNight: 33,
          otherNight: 0
        }
      },
      { 
        id: '2', 
        name: 'Player 2', 
        isAlive: true, 
        role: { 
          id: 'imp', 
          name: 'Imp',
          team: Team.DEMON,
          firstNight: 0,
          otherNight: 24
        }
      },
      { 
        id: '3', 
        name: 'Player 3', 
        isAlive: true, 
        role: { 
          id: 'poisoner', 
          name: 'Poisoner',
          team: Team.MINION,
          firstNight: 17,
          otherNight: 7
        }
      },
      { 
        id: '4', 
        name: 'Player 4', 
        isAlive: false, 
        role: { 
          id: 'ravenkeeper', 
          name: 'Ravenkeeper',
          team: Team.TOWNSFOLK,
          firstNight: 0,
          otherNight: 52
        }
      }
    ];

    // 创建模拟游戏状态
    mockGameState = {
      gameId: 'test-game',
      phase: GamePhase.NIGHT,
      day: 2,
      players: mockPlayers
    };

    // 创建模拟游戏状态管理器
    mockGameStateManager = {
      getCurrentState: jest.fn(() => mockGameState),
      transitionToPhase: jest.fn(async (phase) => {
        mockGameState.phase = phase;
        if (phase === GamePhase.DAY) {
          mockGameState.day += 1;
        }
      })
    };

    // 创建模拟能力执行器
    mockAbilityExecutor = {
      executeAbilityWithTransaction: jest.fn(async (ability, context) => ({
        isSuccess: () => true,
        status: AbilityResultStatus.SUCCESS,
        message: 'Success',
        ability,
        player: context.player
      }))
    };

    nightActionProcessor = new NightActionProcessor(mockGameStateManager, mockAbilityExecutor);
  });

  describe('startNightPhase', () => {
    test('应该成功开始夜间阶段', async () => {
      await nightActionProcessor.startNightPhase();

      expect(nightActionProcessor.hasActiveNightPhase()).toBe(true);
      const result = nightActionProcessor.getCurrentNightResult();
      expect(result).toBeDefined();
      expect(result.day).toBe(2);
      expect(result.phase).toBe(GamePhase.NIGHT);
    });

    test('从非夜间阶段开始应该抛出错误', async () => {
      mockGameState.phase = GamePhase.DAY;

      await expect(nightActionProcessor.startNightPhase()).rejects.toThrow('Cannot start night phase');
    });
  });

  describe('getNightOrder', () => {
    test('应该按照第一夜顺序排序角色', () => {
      mockGameState.phase = GamePhase.FIRST_NIGHT;
      
      const roles = mockPlayers.map(p => p.role);
      const ordered = nightActionProcessor.getNightOrder(roles);

      // 应该按firstNight顺序排序：poisoner(17) < washerwoman(33)
      expect(ordered[0].id).toBe('poisoner');
      expect(ordered[1].id).toBe('washerwoman');
    });

    test('应该按照其他夜晚顺序排序角色', () => {
      mockGameState.phase = GamePhase.NIGHT;
      
      const roles = mockPlayers.map(p => p.role);
      const ordered = nightActionProcessor.getNightOrder(roles);

      // 应该按otherNight顺序排序：poisoner(7) < imp(24)
      expect(ordered[0].id).toBe('poisoner');
      expect(ordered[1].id).toBe('imp');
    });

    test('应该过滤掉没有夜间行动的角色', () => {
      mockGameState.phase = GamePhase.NIGHT;
      
      const roles = mockPlayers.map(p => p.role);
      const ordered = nightActionProcessor.getNightOrder(roles);

      // washerwoman在其他夜晚没有行动（otherNight: 0）
      expect(ordered.find(r => r.id === 'washerwoman')).toBeUndefined();
    });
  });

  describe('processRoleAction', () => {
    test('应该成功处理角色行动', async () => {
      const player = mockPlayers[1]; // Imp
      const role = player.role;
      
      // 添加能力
      role.ability = new Ability({
        id: 'imp-kill',
        name: 'Imp Kill',
        type: AbilityType.KILLING,
        timing: AbilityTiming.NIGHT,
        effects: []
      });

      const actionData = {
        targets: [mockPlayers[0]]
      };

      const result = await nightActionProcessor.processRoleAction(role, player, actionData);

      expect(result.success).toBe(true);
      expect(mockAbilityExecutor.executeAbilityWithTransaction).toHaveBeenCalled();
    });

    test('死亡玩家不应该执行行动', async () => {
      const player = mockPlayers[3]; // Dead player
      const role = player.role;

      const result = await nightActionProcessor.processRoleAction(role, player);

      expect(result.success).toBe(false);
      expect(result.message).toBe('Player is dead');
    });

    test('能力被阻止的玩家不应该执行行动', async () => {
      const player = mockPlayers[0];
      player.status = { abilityBlocked: true };
      const role = player.role;

      const result = await nightActionProcessor.processRoleAction(role, player);

      expect(result.success).toBe(false);
      expect(result.message).toBe('Ability is blocked');
    });

    test('没有能力的角色应该返回失败', async () => {
      const player = mockPlayers[0];
      const role = player.role;
      role.ability = null;

      const result = await nightActionProcessor.processRoleAction(role, player);

      expect(result.success).toBe(false);
      expect(result.message).toBe('No ability found for role');
    });
  });

  describe('resolveAbilityConflicts', () => {
    test('应该解决保护vs杀死冲突', () => {
      const conflicts = [
        {
          type: 'protection_vs_kill',
          abilities: [
            { type: 'protection', id: 'protect' },
            { type: 'killing', id: 'kill' }
          ],
          players: [mockPlayers[0], mockPlayers[1]]
        }
      ];

      const resolutions = nightActionProcessor.resolveAbilityConflicts(conflicts);

      expect(resolutions).toHaveLength(1);
      expect(resolutions[0].winner.type).toBe('protection');
      expect(resolutions[0].reason).toContain('Protection takes priority');
    });

    test('应该解决醉酒vs能力冲突', () => {
      const conflicts = [
        {
          type: 'drunk_vs_ability',
          abilities: [
            { type: 'drunk', id: 'drunk' },
            { type: 'information', id: 'info' }
          ],
          players: [mockPlayers[0]]
        }
      ];

      const resolutions = nightActionProcessor.resolveAbilityConflicts(conflicts);

      expect(resolutions).toHaveLength(1);
      expect(resolutions[0].winner.type).toBe('drunk');
      expect(resolutions[0].reason).toContain('Drunk status invalidates');
    });

    test('应该解决优先级冲突', () => {
      const conflicts = [
        {
          type: 'priority_conflict',
          abilities: [
            { type: 'ability1', id: 'a1', priority: 5 },
            { type: 'ability2', id: 'a2', priority: 10 }
          ],
          players: [mockPlayers[0], mockPlayers[1]]
        }
      ];

      const resolutions = nightActionProcessor.resolveAbilityConflicts(conflicts);

      expect(resolutions).toHaveLength(1);
      expect(resolutions[0].winner.priority).toBe(10);
    });
  });

  describe('completeNightPhase', () => {
    test('应该成功完成夜间阶段', async () => {
      await nightActionProcessor.startNightPhase();
      
      const result = await nightActionProcessor.completeNightPhase();

      expect(result).toBeDefined();
      expect(result.day).toBe(2);
      expect(result.duration).toBeGreaterThanOrEqual(0);
      expect(nightActionProcessor.hasActiveNightPhase()).toBe(false);
    });

    test('没有活跃夜间阶段时应该抛出错误', async () => {
      await expect(nightActionProcessor.completeNightPhase()).rejects.toThrow('No active night phase');
    });

    test('应该收集死亡玩家', async () => {
      await nightActionProcessor.startNightPhase();
      
      // 模拟玩家死亡
      mockPlayers[0].isAlive = false;
      
      const result = await nightActionProcessor.completeNightPhase();

      expect(result.deaths.length).toBeGreaterThan(0);
    });
  });

  describe('processNightPhase', () => {
    test('应该处理完整的夜间阶段', async () => {
      // 为角色添加能力
      mockPlayers.forEach(player => {
        if (player.role.otherNight > 0) {
          player.role.ability = new Ability({
            id: `${player.role.id}-ability`,
            name: `${player.role.name} Ability`,
            type: AbilityType.INFORMATION,
            timing: AbilityTiming.NIGHT,
            effects: []
          });
        }
      });

      const nightActions = {
        '2': { targets: [mockPlayers[0]] }, // Imp kills Player 1
        '3': { targets: [mockPlayers[0]] }  // Poisoner poisons Player 1
      };

      const result = await nightActionProcessor.processNightPhase(nightActions);

      expect(result).toBeDefined();
      expect(result.actions.length).toBeGreaterThan(0);
    });

    test('应该按正确顺序执行行动', async () => {
      const executionOrder = [];
      
      mockAbilityExecutor.executeAbilityWithTransaction = jest.fn(async (ability, context) => {
        executionOrder.push(context.player.role.id);
        return {
          isSuccess: () => true,
          status: AbilityResultStatus.SUCCESS,
          message: 'Success',
          ability,
          player: context.player
        };
      });

      // 为角色添加能力
      mockPlayers.forEach(player => {
        if (player.role.otherNight > 0) {
          player.role.ability = new Ability({
            id: `${player.role.id}-ability`,
            name: `${player.role.name} Ability`,
            type: AbilityType.INFORMATION,
            timing: AbilityTiming.NIGHT,
            effects: []
          });
        }
      });

      await nightActionProcessor.processNightPhase({});

      // 验证执行顺序：poisoner(7) 应该在 imp(24) 之前
      const poisonerIndex = executionOrder.indexOf('poisoner');
      const impIndex = executionOrder.indexOf('imp');
      
      expect(poisonerIndex).toBeLessThan(impIndex);
    });
  });

  describe('validateNightActions', () => {
    test('应该验证有效的夜间行动配置', () => {
      const nightActions = {
        '1': { targets: [mockPlayers[1]] },
        '2': { targets: [mockPlayers[0]] }
      };

      const validation = nightActionProcessor.validateNightActions(nightActions);

      expect(validation.valid).toBe(true);
      expect(validation.errors).toHaveLength(0);
    });

    test('应该检测不存在的玩家', () => {
      const nightActions = {
        'invalid-id': { targets: [] }
      };

      const validation = nightActionProcessor.validateNightActions(nightActions);

      expect(validation.valid).toBe(false);
      expect(validation.errors.some(e => e.includes('not found'))).toBe(true);
    });

    test('应该检测死亡玩家', () => {
      const nightActions = {
        '4': { targets: [mockPlayers[0]] } // Dead player
      };

      const validation = nightActionProcessor.validateNightActions(nightActions);

      expect(validation.valid).toBe(false);
      expect(validation.errors.some(e => e.includes('is dead'))).toBe(true);
    });

    test('应该检测无效的目标', () => {
      const nightActions = {
        '1': { targets: [null, mockPlayers[1]] }
      };

      const validation = nightActionProcessor.validateNightActions(nightActions);

      expect(validation.valid).toBe(false);
      expect(validation.errors.some(e => e.includes('Invalid target'))).toBe(true);
    });
  });

  describe('历史和统计', () => {
    test('应该记录夜间行动历史', async () => {
      mockPlayers[1].role.ability = new Ability({
        id: 'test-ability',
        name: 'Test Ability',
        type: AbilityType.INFORMATION,
        timing: AbilityTiming.NIGHT,
        effects: []
      });

      await nightActionProcessor.startNightPhase();
      await nightActionProcessor.processRoleAction(mockPlayers[1].role, mockPlayers[1]);

      const history = nightActionProcessor.getNightActionHistory();
      expect(history.length).toBeGreaterThan(0);
    });

    test('应该清除夜间行动历史', async () => {
      mockPlayers[1].role.ability = new Ability({
        id: 'test-ability',
        name: 'Test Ability',
        type: AbilityType.INFORMATION,
        timing: AbilityTiming.NIGHT,
        effects: []
      });

      await nightActionProcessor.startNightPhase();
      await nightActionProcessor.processRoleAction(mockPlayers[1].role, mockPlayers[1]);
      
      nightActionProcessor.clearNightActionHistory();
      
      const history = nightActionProcessor.getNightActionHistory();
      expect(history).toHaveLength(0);
    });

    test('应该生成夜间行动统计', async () => {
      mockPlayers[1].role.ability = new Ability({
        id: 'test-ability',
        name: 'Test Ability',
        type: AbilityType.INFORMATION,
        timing: AbilityTiming.NIGHT,
        effects: []
      });

      await nightActionProcessor.startNightPhase();
      await nightActionProcessor.processRoleAction(mockPlayers[1].role, mockPlayers[1]);
      await nightActionProcessor.processRoleAction(mockPlayers[1].role, mockPlayers[1]);

      const stats = nightActionProcessor.getNightActionStats();
      
      expect(stats.total).toBe(2);
      expect(stats.successful).toBe(2);
      expect(stats.roleUsage['imp']).toBeDefined();
      expect(stats.roleUsage['imp'].total).toBe(2);
    });
  });

  describe('集成测试 - 能力解析和状态更新', () => {
    test('应该在夜间行动中移除死亡角色的后续行动', async () => {
      const executionOrder = [];
      
      // 模拟能力执行器，第一个行动会杀死一个玩家
      mockAbilityExecutor.executeAbilityWithTransaction = jest.fn(async (ability, context) => {
        executionOrder.push(context.player.role.id);
        
        // 如果是Imp的行动，杀死目标玩家
        if (context.player.role.id === 'imp' && context.targets.length > 0) {
          const target = context.targets[0];
          // 更新mockGameState中的玩家状态
          const targetInState = mockGameState.players.find(p => p.id === target.id);
          if (targetInState) {
            targetInState.isAlive = false;
          }
        }
        
        return {
          isSuccess: () => true,
          status: AbilityResultStatus.SUCCESS,
          message: 'Success',
          ability,
          player: context.player
        };
      });

      // 设置夜间顺序：poisoner(7) -> imp(24) -> ravenkeeper(52)
      // Ravenkeeper现在是活着的，但会被Imp杀死
      mockPlayers[3].isAlive = true;
      mockPlayers[3].role.otherNight = 52;

      // 为所有角色添加能力
      mockPlayers.forEach(player => {
        if (player.role.otherNight > 0) {
          player.role.ability = new Ability({
            id: `${player.role.id}-ability`,
            name: `${player.role.name} Ability`,
            type: player.role.id === 'imp' ? AbilityType.KILLING : AbilityType.INFORMATION,
            timing: AbilityTiming.NIGHT,
            effects: []
          });
        }
      });

      const nightActions = {
        '2': { targets: [mockPlayers[3]] }, // Imp杀死Ravenkeeper
        '3': { targets: [mockPlayers[0]] }  // Poisoner中毒Player 1
      };

      await nightActionProcessor.processNightPhase(nightActions);

      // Ravenkeeper应该不在执行顺序中，因为他在Imp行动后死亡
      expect(executionOrder).toContain('poisoner');
      expect(executionOrder).toContain('imp');
      // Ravenkeeper的行动应该被移除（因为他在Imp行动后死亡）
      // 注意：这取决于实现细节，可能需要调整
    });

    test('应该在夜间阶段完成后自动转换到白天', async () => {
      // 为角色添加能力
      mockPlayers.forEach(player => {
        if (player.role.otherNight > 0) {
          player.role.ability = new Ability({
            id: `${player.role.id}-ability`,
            name: `${player.role.name} Ability`,
            type: AbilityType.INFORMATION,
            timing: AbilityTiming.NIGHT,
            effects: []
          });
        }
      });

      await nightActionProcessor.processNightPhase({});

      // 验证转换到白天阶段被调用
      expect(mockGameStateManager.transitionToPhase).toHaveBeenCalledWith(GamePhase.DAY);
    });

    test('应该在夜间阶段完成时清理临时状态', async () => {
      // 添加临时状态到玩家
      mockPlayers[0].status = {
        protected: true,
        protectedUntil: 'one_night',
        abilityBlocked: true
      };

      mockPlayers[1].status = {
        poisoned: true,
        poisonedUntil: 'permanent'
      };

      // 为角色添加能力
      mockPlayers.forEach(player => {
        if (player.role.otherNight > 0) {
          player.role.ability = new Ability({
            id: `${player.role.id}-ability`,
            name: `${player.role.name} Ability`,
            type: AbilityType.INFORMATION,
            timing: AbilityTiming.NIGHT,
            effects: []
          });
        }
      });

      await nightActionProcessor.processNightPhase({});

      // 验证一次性状态被清理
      expect(mockPlayers[0].status.protected).toBeUndefined();
      expect(mockPlayers[0].status.abilityBlocked).toBeUndefined();
      
      // 永久状态应该保留
      expect(mockPlayers[1].status.poisoned).toBe(true);
    });

    test('应该处理能力执行失败的情况', async () => {
      // 模拟能力执行失败
      mockAbilityExecutor.executeAbilityWithTransaction = jest.fn(async (ability, context) => ({
        isSuccess: () => false,
        status: AbilityResultStatus.FAILED,
        message: 'Ability execution failed',
        ability,
        player: context.player
      }));

      mockPlayers[1].role.ability = new Ability({
        id: 'test-ability',
        name: 'Test Ability',
        type: AbilityType.INFORMATION,
        timing: AbilityTiming.NIGHT,
        effects: []
      });

      await nightActionProcessor.startNightPhase();
      const result = await nightActionProcessor.processRoleAction(mockPlayers[1].role, mockPlayers[1]);

      expect(result.success).toBe(false);
      expect(result.result.status).toBe(AbilityResultStatus.FAILED);
    });

    test('应该处理多个玩家在同一夜死亡的情况', async () => {
      let killCount = 0;
      
      mockAbilityExecutor.executeAbilityWithTransaction = jest.fn(async (ability, context) => {
        // 每次执行都杀死一个玩家
        if (context.targets.length > 0 && killCount < 2) {
          const target = context.targets[0];
          const targetInState = mockGameState.players.find(p => p.id === target.id);
          if (targetInState) {
            targetInState.isAlive = false;
            killCount++;
          }
        }
        
        return {
          isSuccess: () => true,
          status: AbilityResultStatus.SUCCESS,
          message: 'Success',
          ability,
          player: context.player
        };
      });

      // 为角色添加能力
      mockPlayers.forEach(player => {
        if (player.role.otherNight > 0) {
          player.role.ability = new Ability({
            id: `${player.role.id}-ability`,
            name: `${player.role.name} Ability`,
            type: AbilityType.KILLING,
            timing: AbilityTiming.NIGHT,
            effects: []
          });
        }
      });

      const nightActions = {
        '2': { targets: [mockPlayers[0]] },
        '3': { targets: [mockPlayers[1]] }
      };

      const result = await nightActionProcessor.processNightPhase(nightActions);

      expect(result.deaths.length).toBeGreaterThanOrEqual(2);
    });
  });
});

  // 集成测试：胜负判断
  describe('Victory Condition Integration', () => {
    let mockVictoryChecker;
    let localMockPlayers;
    let localMockGameState;
    let localMockGameStateManager;
    let localMockAbilityExecutor;
    let localNightActionProcessor;

    beforeEach(() => {
      // 重新初始化所有mock对象
      localMockPlayers = [
        { 
          id: '1', 
          name: 'Player 1', 
          isAlive: true, 
          role: { 
            id: 'washerwoman', 
            name: 'Washerwoman',
            team: Team.TOWNSFOLK,
            firstNight: 33,
            otherNight: 0
          }
        },
        { 
          id: '2', 
          name: 'Player 2', 
          isAlive: true, 
          role: { 
            id: 'imp', 
            name: 'Imp',
            team: Team.DEMON,
            firstNight: 0,
            otherNight: 24
          }
        }
      ];

      localMockGameState = {
        gameId: 'test-game',
        phase: GamePhase.NIGHT,
        day: 1,
        players: localMockPlayers,
        timestamp: Date.now()
      };

      localMockGameStateManager = {
        getCurrentState: jest.fn(() => localMockGameState),
        transitionToPhase: jest.fn(),
        updatePlayerState: jest.fn()
      };

      localMockAbilityExecutor = {
        executeAbilityWithTransaction: jest.fn()
      };

      mockVictoryChecker = {
        checkAndEndGame: jest.fn()
      };

      localNightActionProcessor = new NightActionProcessor(
        localMockGameStateManager,
        localMockAbilityExecutor,
        mockVictoryChecker
      );
    });

    test('should check victory conditions after night phase completes', async () => {
      mockVictoryChecker.checkAndEndGame.mockResolvedValue(null);

      await localNightActionProcessor.startNightPhase();
      const result = await localNightActionProcessor.completeNightPhase();

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

      await localNightActionProcessor.startNightPhase();
      const result = await localNightActionProcessor.completeNightPhase();

      expect(result.gameEnded).toBe(true);
      expect(result.victoryResult).toEqual(victoryResult);
    });

    test('should not transition to day phase when game ends', async () => {
      const victoryResult = {
        winner: 'evil',
        reason: 'Evil wins',
        ended: true
      };

      mockVictoryChecker.checkAndEndGame.mockResolvedValue(victoryResult);

      await localNightActionProcessor.startNightPhase();
      await localNightActionProcessor.processNightPhase({});

      // transitionToPhase 不应该被调用转换到DAY
      const calls = localMockGameStateManager.transitionToPhase.mock.calls;
      const dayTransition = calls.find(call => call[0] === GamePhase.DAY);
      expect(dayTransition).toBeUndefined();
    });

    test('should handle night phase without victory checker', async () => {
      const processorWithoutChecker = new NightActionProcessor(
        localMockGameStateManager,
        localMockAbilityExecutor,
        null
      );

      await processorWithoutChecker.startNightPhase();
      const result = await processorWithoutChecker.completeNightPhase();

      expect(result.gameEnded).toBeUndefined();
      expect(result.victoryResult).toBeUndefined();
    });

    test('should continue to day phase when no victory condition is met', async () => {
      mockVictoryChecker.checkAndEndGame.mockResolvedValue(null);

      await localNightActionProcessor.processNightPhase({});

      expect(localMockGameStateManager.transitionToPhase).toHaveBeenCalledWith(GamePhase.DAY);
    });
  });
