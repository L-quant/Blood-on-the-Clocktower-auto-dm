/**
 * 测试能力解析器
 */

import AbilityResolver from '../../../src/automation/core/AbilityResolver';
import {
  Ability,
  AbilityContext,
  AbilityResult,
  AbilityType,
  AbilityTiming,
  EffectType,
  TargetType,
  AbilityResultStatus,
  EffectDuration,
  AbilityConditionType
} from '../../../src/automation/types/AbilityTypes';
import { Team, GamePhase } from '../../../src/automation/types/GameTypes';

describe('AbilityResolver', () => {
  let abilityResolver;
  let mockGameStateManager;
  let mockGameState;
  let mockPlayers;

  beforeEach(() => {
    // 创建模拟玩家
    mockPlayers = [
      { id: '1', name: 'Player 1', isAlive: true, role: { id: 'washerwoman', team: Team.TOWNSFOLK } },
      { id: '2', name: 'Player 2', isAlive: true, role: { id: 'chef', team: Team.TOWNSFOLK } },
      { id: '3', name: 'Player 3', isAlive: true, role: { id: 'imp', team: Team.DEMON } },
      { id: '4', name: 'Player 4', isAlive: true, role: { id: 'poisoner', team: Team.MINION } },
      { id: '5', name: 'Player 5', isAlive: false, role: { id: 'ravenkeeper', team: Team.TOWNSFOLK } }
    ];

    // 创建模拟游戏状态
    mockGameState = {
      gameId: 'test-game',
      phase: GamePhase.NIGHT,
      day: 1,
      players: mockPlayers,
      abilityHistory: []
    };

    // 创建模拟游戏状态管理器
    mockGameStateManager = {
      getCurrentState: jest.fn(() => mockGameState),
      updatePlayerState: jest.fn()
    };

    abilityResolver = new AbilityResolver(mockGameStateManager);
  });

  describe('parseAbility', () => {
    test('应该成功解析有效的能力', () => {
      const ability = new Ability({
        id: 'test-ability',
        name: 'Test Ability',
        description: 'A test ability',
        type: AbilityType.INFORMATION,
        timing: AbilityTiming.NIGHT,
        effects: [
          {
            type: EffectType.LEARN_INFO,
            target: TargetType.SINGLE_PLAYER,
            value: 'test-info'
          }
        ]
      });

      const context = new AbilityContext({
        gameState: mockGameState,
        player: mockPlayers[0],
        targets: [mockPlayers[1]],
        phase: GamePhase.NIGHT
      });

      const parsed = abilityResolver.parseAbility(ability, context);

      expect(parsed.canUse).toBe(true);
      expect(parsed.ability).toBe(ability);
      expect(parsed.context).toBe(context);
      expect(parsed.effects).toHaveLength(1);
    });

    test('能力为空时应该抛出错误', () => {
      const context = new AbilityContext({
        gameState: mockGameState,
        player: mockPlayers[0]
      });

      expect(() => {
        abilityResolver.parseAbility(null, context);
      }).toThrow('Ability is required');
    });

    test('上下文为空时应该抛出错误', () => {
      const ability = new Ability({
        id: 'test-ability',
        name: 'Test Ability',
        type: AbilityType.INFORMATION,
        timing: AbilityTiming.NIGHT,
        effects: []
      });

      expect(() => {
        abilityResolver.parseAbility(ability, null);
      }).toThrow('Context is required');
    });
  });

  describe('validateAbilityConditions', () => {
    test('应该验证玩家存活条件', () => {
      const ability = new Ability({
        id: 'test-ability',
        name: 'Test Ability',
        type: AbilityType.INFORMATION,
        timing: AbilityTiming.NIGHT,
        conditions: [
          { type: AbilityConditionType.PLAYER_ALIVE }
        ],
        effects: []
      });

      const aliveContext = new AbilityContext({
        gameState: mockGameState,
        player: mockPlayers[0]
      });

      const deadContext = new AbilityContext({
        gameState: mockGameState,
        player: mockPlayers[4]
      });

      expect(abilityResolver.validateAbilityConditions(ability, aliveContext)).toBe(true);
      expect(abilityResolver.validateAbilityConditions(ability, deadContext)).toBe(false);
    });

    test('能力或上下文为空时应该返回false', () => {
      expect(abilityResolver.validateAbilityConditions(null, null)).toBe(false);
    });
  });

  describe('executeAbility', () => {
    test('应该成功执行学习信息能力', async () => {
      const ability = new Ability({
        id: 'learn-info',
        name: 'Learn Info',
        type: AbilityType.INFORMATION,
        timing: AbilityTiming.NIGHT,
        effects: [
          {
            type: EffectType.LEARN_INFO,
            target: TargetType.SINGLE_PLAYER,
            value: 'test-information'
          }
        ]
      });

      const context = new AbilityContext({
        gameState: mockGameState,
        player: mockPlayers[0],
        targets: [mockPlayers[1]]
      });

      const parsed = abilityResolver.parseAbility(ability, context);
      const result = await abilityResolver.executeAbility(parsed);

      expect(result.isSuccess()).toBe(true);
      expect(result.status).toBe(AbilityResultStatus.SUCCESS);
      expect(result.effects).toHaveLength(1);
    });

    test('应该成功执行杀死能力', async () => {
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

      const parsed = abilityResolver.parseAbility(ability, context);
      const result = await abilityResolver.executeAbility(parsed);

      expect(result.isSuccess()).toBe(true);
      expect(mockPlayers[0].isAlive).toBe(false);
    });

    test('应该成功执行保护能力', async () => {
      const ability = new Ability({
        id: 'protect',
        name: 'Protect',
        type: AbilityType.PROTECTION,
        timing: AbilityTiming.NIGHT,
        effects: [
          {
            type: EffectType.PROTECT,
            target: TargetType.SINGLE_PLAYER,
            duration: EffectDuration.ONE_NIGHT
          }
        ]
      });

      const context = new AbilityContext({
        gameState: mockGameState,
        player: mockPlayers[0],
        targets: [mockPlayers[1]]
      });

      const parsed = abilityResolver.parseAbility(ability, context);
      const result = await abilityResolver.executeAbility(parsed);

      expect(result.isSuccess()).toBe(true);
      expect(mockPlayers[1].status.protected).toBe(true);
    });

    test('中毒的玩家使用能力应该失败', async () => {
      mockPlayers[0].status = { poisoned: true };

      const ability = new Ability({
        id: 'test-ability',
        name: 'Test Ability',
        type: AbilityType.INFORMATION,
        timing: AbilityTiming.NIGHT,
        effects: []
      });

      const context = new AbilityContext({
        gameState: mockGameState,
        player: mockPlayers[0]
      });

      const parsed = abilityResolver.parseAbility(ability, context);
      const result = await abilityResolver.executeAbility(parsed);

      expect(result.status).toBe(AbilityResultStatus.POISONED);
      expect(result.isBlocked()).toBe(true);
    });

    test('醉酒的玩家使用能力应该失败', async () => {
      mockPlayers[0].status = { drunk: true };

      const ability = new Ability({
        id: 'test-ability',
        name: 'Test Ability',
        type: AbilityType.INFORMATION,
        timing: AbilityTiming.NIGHT,
        effects: []
      });

      const context = new AbilityContext({
        gameState: mockGameState,
        player: mockPlayers[0]
      });

      const parsed = abilityResolver.parseAbility(ability, context);
      const result = await abilityResolver.executeAbility(parsed);

      expect(result.status).toBe(AbilityResultStatus.DRUNK);
      expect(result.isBlocked()).toBe(true);
    });

    test('不能使用的能力应该返回无效状态', async () => {
      const ability = new Ability({
        id: 'test-ability',
        name: 'Test Ability',
        type: AbilityType.INFORMATION,
        timing: AbilityTiming.NIGHT,
        conditions: [
          { type: AbilityConditionType.PLAYER_DEAD }
        ],
        effects: []
      });

      const context = new AbilityContext({
        gameState: mockGameState,
        player: mockPlayers[0] // 存活的玩家
      });

      const parsed = abilityResolver.parseAbility(ability, context);
      const result = await abilityResolver.executeAbility(parsed);

      expect(result.status).toBe(AbilityResultStatus.INVALID);
    });
  });

  describe('handleAbilitySideEffects', () => {
    test('应该记录能力使用历史', async () => {
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

      const parsed = abilityResolver.parseAbility(ability, context);
      const result = await abilityResolver.executeAbility(parsed);

      expect(mockGameState.abilityHistory.length).toBeGreaterThan(0);
      expect(mockGameState.abilityHistory[0].ability).toBe('test-ability');
      expect(mockGameState.abilityHistory[0].player).toBe('1');
    });
  });

  describe('目标解析', () => {
    test('应该正确解析SELF目标', () => {
      const ability = new Ability({
        id: 'self-ability',
        name: 'Self Ability',
        type: AbilityType.PASSIVE,
        timing: AbilityTiming.PASSIVE,
        effects: [
          {
            type: EffectType.PROTECT,
            target: TargetType.SELF
          }
        ]
      });

      const context = new AbilityContext({
        gameState: mockGameState,
        player: mockPlayers[0]
      });

      const parsed = abilityResolver.parseAbility(ability, context);

      expect(parsed.effects[0].resolvedTargets).toHaveLength(1);
      expect(parsed.effects[0].resolvedTargets[0]).toBe(mockPlayers[0]);
    });

    test('应该正确解析ALIVE_PLAYERS目标', () => {
      const ability = new Ability({
        id: 'all-alive',
        name: 'All Alive',
        type: AbilityType.INFORMATION,
        timing: AbilityTiming.NIGHT,
        effects: [
          {
            type: EffectType.LEARN_INFO,
            target: TargetType.ALIVE_PLAYERS
          }
        ]
      });

      const context = new AbilityContext({
        gameState: mockGameState,
        player: mockPlayers[0]
      });

      const parsed = abilityResolver.parseAbility(ability, context);

      expect(parsed.effects[0].resolvedTargets).toHaveLength(4); // 4个存活玩家
      expect(parsed.effects[0].resolvedTargets.every(p => p.isAlive)).toBe(true);
    });

    test('应该正确解析NEIGHBORS目标', () => {
      const ability = new Ability({
        id: 'neighbors',
        name: 'Neighbors',
        type: AbilityType.INFORMATION,
        timing: AbilityTiming.NIGHT,
        effects: [
          {
            type: EffectType.LEARN_INFO,
            target: TargetType.NEIGHBORS
          }
        ]
      });

      const context = new AbilityContext({
        gameState: mockGameState,
        player: mockPlayers[1] // Player 2
      });

      const parsed = abilityResolver.parseAbility(ability, context);

      expect(parsed.effects[0].resolvedTargets).toHaveLength(2);
    });
  });

  describe('特殊效果处理', () => {
    test('应该正确处理中毒效果', async () => {
      const ability = new Ability({
        id: 'poison',
        name: 'Poison',
        type: AbilityType.MANIPULATION,
        timing: AbilityTiming.NIGHT,
        effects: [
          {
            type: EffectType.POISON,
            target: TargetType.SINGLE_PLAYER,
            duration: EffectDuration.ONE_DAY
          }
        ]
      });

      const context = new AbilityContext({
        gameState: mockGameState,
        player: mockPlayers[3], // Poisoner
        targets: [mockPlayers[0]]
      });

      const parsed = abilityResolver.parseAbility(ability, context);
      const result = await abilityResolver.executeAbility(parsed);

      expect(result.isSuccess()).toBe(true);
      expect(mockPlayers[0].status.poisoned).toBe(true);
    });

    test('应该正确处理复活效果', async () => {
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
        targets: [mockPlayers[4]] // 死亡的玩家
      });

      const parsed = abilityResolver.parseAbility(ability, context);
      const result = await abilityResolver.executeAbility(parsed);

      expect(result.isSuccess()).toBe(true);
      expect(mockPlayers[4].isAlive).toBe(true);
    });

    test('保护状态应该阻止杀死', async () => {
      // 先保护玩家
      mockPlayers[0].status = { protected: true };

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

      const parsed = abilityResolver.parseAbility(ability, context);
      const result = await abilityResolver.executeAbility(parsed);

      expect(result.isSuccess()).toBe(true);
      expect(mockPlayers[0].isAlive).toBe(true); // 仍然存活
    });

    test('士兵应该免疫杀死', async () => {
      mockPlayers[0].role = { id: 'soldier', team: Team.TOWNSFOLK };

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

      const parsed = abilityResolver.parseAbility(ability, context);
      const result = await abilityResolver.executeAbility(parsed);

      expect(result.isSuccess()).toBe(true);
      expect(mockPlayers[0].isAlive).toBe(true); // 士兵免疫
    });
  });

  describe('错误处理', () => {
    test('执行失败的能力应该返回失败状态', async () => {
      const ability = new Ability({
        id: 'error-ability',
        name: 'Error Ability',
        type: AbilityType.MANIPULATION,
        timing: AbilityTiming.NIGHT,
        effects: [
          {
            type: 'unknown-effect-type', // 未知效果类型
            target: TargetType.SINGLE_PLAYER
          }
        ]
      });

      const context = new AbilityContext({
        gameState: mockGameState,
        player: mockPlayers[0],
        targets: [mockPlayers[1]]
      });

      const parsed = abilityResolver.parseAbility(ability, context);
      const result = await abilityResolver.executeAbility(parsed);

      expect(result.isSuccess()).toBe(true); // 仍然成功，但效果失败
      expect(result.effects[0].success).toBe(false);
    });
  });
});
