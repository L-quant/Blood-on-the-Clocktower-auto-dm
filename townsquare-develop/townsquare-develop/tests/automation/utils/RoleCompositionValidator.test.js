/**
 * 测试角色组合验证器
 */

import RoleCompositionValidator from '../../../src/automation/utils/RoleCompositionValidator';
import { Team } from '../../../src/automation/types/GameTypes';
import { getRoleComposition } from '../../../src/automation/data/ScriptDefinitions';

describe('RoleCompositionValidator', () => {
  describe('validate', () => {
    test('应该验证有效的角色组合', () => {
      const composition = { townsfolk: 3, outsider: 0, minion: 1, demon: 1 };
      const result = RoleCompositionValidator.validate(composition, 5);
      
      expect(result.valid).toBe(true);
      expect(result.errors).toHaveLength(0);
    });

    test('应该拒绝玩家数量小于5的组合', () => {
      const composition = { townsfolk: 2, outsider: 0, minion: 1, demon: 1 };
      const result = RoleCompositionValidator.validate(composition, 4);
      
      expect(result.valid).toBe(false);
      expect(result.errors).toContain('Invalid player count: 4. Must be between 5 and 15.');
    });

    test('应该拒绝玩家数量大于15的组合', () => {
      const composition = { townsfolk: 10, outsider: 2, minion: 3, demon: 1 };
      const result = RoleCompositionValidator.validate(composition, 16);
      
      expect(result.valid).toBe(false);
      expect(result.errors).toContain('Invalid player count: 16. Must be between 5 and 15.');
    });

    test('应该拒绝角色总数不匹配玩家数量的组合', () => {
      const composition = { townsfolk: 3, outsider: 0, minion: 1, demon: 1 };
      const result = RoleCompositionValidator.validate(composition, 6);
      
      expect(result.valid).toBe(false);
      expect(result.errors.some(e => e.includes('does not match player count'))).toBe(true);
    });

    test('应该拒绝没有恶魔的组合', () => {
      const composition = { townsfolk: 4, outsider: 0, minion: 1, demon: 0 };
      const result = RoleCompositionValidator.validate(composition, 5);
      
      expect(result.valid).toBe(false);
      expect(result.errors).toContain('Must have at least 1 Demon');
    });

    test('应该拒绝多个恶魔的组合', () => {
      const composition = { townsfolk: 2, outsider: 0, minion: 1, demon: 2 };
      const result = RoleCompositionValidator.validate(composition, 5);
      
      expect(result.valid).toBe(false);
      expect(result.errors).toContain('Cannot have more than 1 Demon in standard games');
    });

    test('应该拒绝爪牙数量过多的组合', () => {
      const composition = { townsfolk: 3, outsider: 0, minion: 5, demon: 1 };
      const result = RoleCompositionValidator.validate(composition, 9);
      
      expect(result.valid).toBe(false);
      expect(result.errors.some(e => e.includes('Too many Minions'))).toBe(true);
    });

    test('应该拒绝外来者数量过多的组合', () => {
      const composition = { townsfolk: 3, outsider: 5, minion: 1, demon: 1 };
      const result = RoleCompositionValidator.validate(composition, 10);
      
      expect(result.valid).toBe(false);
      expect(result.errors.some(e => e.includes('Too many Outsiders'))).toBe(true);
    });

    test('应该拒绝村民数量为负数的组合', () => {
      const composition = { townsfolk: -1, outsider: 3, minion: 2, demon: 1 };
      const result = RoleCompositionValidator.validate(composition, 5);
      
      expect(result.valid).toBe(false);
      expect(result.errors).toContain('Townsfolk count cannot be negative');
    });
  });

  describe('validateRoleList', () => {
    test('应该验证匹配组合的角色列表', () => {
      const roles = [
        { id: 'washerwoman', team: Team.TOWNSFOLK },
        { id: 'chef', team: Team.TOWNSFOLK },
        { id: 'empath', team: Team.TOWNSFOLK },
        { id: 'poisoner', team: Team.MINION },
        { id: 'imp', team: Team.DEMON }
      ];
      const composition = { townsfolk: 3, outsider: 0, minion: 1, demon: 1 };
      
      const result = RoleCompositionValidator.validateRoleList(roles, composition);
      
      expect(result.valid).toBe(true);
      expect(result.errors).toHaveLength(0);
    });

    test('应该拒绝村民数量不匹配的角色列表', () => {
      const roles = [
        { id: 'washerwoman', team: Team.TOWNSFOLK },
        { id: 'chef', team: Team.TOWNSFOLK },
        { id: 'poisoner', team: Team.MINION },
        { id: 'imp', team: Team.DEMON }
      ];
      const composition = { townsfolk: 3, outsider: 0, minion: 1, demon: 1 };
      
      const result = RoleCompositionValidator.validateRoleList(roles, composition);
      
      expect(result.valid).toBe(false);
      expect(result.errors.some(e => e.includes('townsfolk count mismatch'))).toBe(true);
    });

    test('应该检测重复的角色', () => {
      const roles = [
        { id: 'washerwoman', team: Team.TOWNSFOLK },
        { id: 'washerwoman', team: Team.TOWNSFOLK },
        { id: 'chef', team: Team.TOWNSFOLK },
        { id: 'poisoner', team: Team.MINION },
        { id: 'imp', team: Team.DEMON }
      ];
      const composition = { townsfolk: 3, outsider: 0, minion: 1, demon: 1 };
      
      const result = RoleCompositionValidator.validateRoleList(roles, composition);
      
      expect(result.valid).toBe(false);
      expect(result.errors).toContain('Duplicate roles detected');
    });
  });

  describe('applySpecialRules', () => {
    test('应该应用男爵的特殊规则', () => {
      const composition = { townsfolk: 7, outsider: 0, minion: 2, demon: 1 };
      const selectedRoles = [{ id: 'baron', team: Team.MINION }];
      
      const modified = RoleCompositionValidator.applySpecialRules(composition, selectedRoles);
      
      expect(modified.townsfolk).toBe(5);
      expect(modified.outsider).toBe(2);
    });

    test('应该应用教父的特殊规则', () => {
      const composition = { townsfolk: 7, outsider: 1, minion: 2, demon: 1 };
      const selectedRoles = [{ id: 'godfather', team: Team.MINION }];
      
      const modified = RoleCompositionValidator.applySpecialRules(composition, selectedRoles);
      
      // 教父会随机增加或减少1个外来者
      const totalChange = Math.abs(modified.townsfolk - composition.townsfolk) + 
                         Math.abs(modified.outsider - composition.outsider);
      expect(totalChange).toBe(2); // 一个增加1，一个减少1
    });

    test('没有特殊角色时应该返回原始组合', () => {
      const composition = { townsfolk: 7, outsider: 0, minion: 2, demon: 1 };
      const selectedRoles = [{ id: 'poisoner', team: Team.MINION }];
      
      const modified = RoleCompositionValidator.applySpecialRules(composition, selectedRoles);
      
      expect(modified).toEqual(composition);
    });
  });

  describe('canAchieveComposition', () => {
    test('应该识别可以通过男爵达成的组合', () => {
      const targetComposition = { townsfolk: 5, outsider: 2, minion: 2, demon: 1 };
      const baseComposition = { townsfolk: 7, outsider: 0, minion: 2, demon: 1 };
      const availableRoles = [{ id: 'baron', team: Team.MINION }];
      
      const result = RoleCompositionValidator.canAchieveComposition(
        targetComposition,
        baseComposition,
        availableRoles
      );
      
      expect(result.achievable).toBe(true);
      expect(result.requiredRoles).toHaveLength(1);
      expect(result.requiredRoles[0].id).toBe('baron');
    });

    test('应该识别可以通过教父达成的组合', () => {
      const targetComposition = { townsfolk: 6, outsider: 2, minion: 2, demon: 1 };
      const baseComposition = { townsfolk: 7, outsider: 1, minion: 2, demon: 1 };
      const availableRoles = [{ id: 'godfather', team: Team.MINION }];
      
      const result = RoleCompositionValidator.canAchieveComposition(
        targetComposition,
        baseComposition,
        availableRoles
      );
      
      expect(result.achievable).toBe(true);
      expect(result.requiredRoles).toHaveLength(1);
      expect(result.requiredRoles[0].id).toBe('godfather');
    });

    test('相同组合应该直接可达成', () => {
      const composition = { townsfolk: 7, outsider: 0, minion: 2, demon: 1 };
      const availableRoles = [];
      
      const result = RoleCompositionValidator.canAchieveComposition(
        composition,
        composition,
        availableRoles
      );
      
      expect(result.achievable).toBe(true);
      expect(result.requiredRoles).toHaveLength(0);
    });

    test('无法达成的组合应该返回false', () => {
      const targetComposition = { townsfolk: 3, outsider: 4, minion: 2, demon: 1 };
      const baseComposition = { townsfolk: 7, outsider: 0, minion: 2, demon: 1 };
      const availableRoles = [{ id: 'baron', team: Team.MINION }];
      
      const result = RoleCompositionValidator.canAchieveComposition(
        targetComposition,
        baseComposition,
        availableRoles
      );
      
      expect(result.achievable).toBe(false);
    });
  });

  describe('getCompositionStats', () => {
    test('应该计算正确的统计信息', () => {
      const composition = { townsfolk: 7, outsider: 0, minion: 2, demon: 1 };
      const stats = RoleCompositionValidator.getCompositionStats(composition);
      
      expect(stats.total).toBe(10);
      expect(stats.goodCount).toBe(7);
      expect(stats.evilCount).toBe(3);
      expect(stats.goodPercentage).toBe('70.0');
      expect(stats.evilPercentage).toBe('30.0');
    });

    test('应该计算各团队的百分比', () => {
      const composition = { townsfolk: 5, outsider: 2, minion: 1, demon: 1 };
      const stats = RoleCompositionValidator.getCompositionStats(composition);
      
      expect(stats.townsfolkPercentage).toBe('55.6');
      expect(stats.outsiderPercentage).toBe('22.2');
      expect(stats.minionPercentage).toBe('11.1');
      expect(stats.demonPercentage).toBe('11.1');
    });
  });

  describe('areCompositionsEqual', () => {
    test('相同的组合应该返回true', () => {
      const comp1 = { townsfolk: 7, outsider: 0, minion: 2, demon: 1 };
      const comp2 = { townsfolk: 7, outsider: 0, minion: 2, demon: 1 };
      
      expect(RoleCompositionValidator.areCompositionsEqual(comp1, comp2)).toBe(true);
    });

    test('不同的组合应该返回false', () => {
      const comp1 = { townsfolk: 7, outsider: 0, minion: 2, demon: 1 };
      const comp2 = { townsfolk: 5, outsider: 2, minion: 2, demon: 1 };
      
      expect(RoleCompositionValidator.areCompositionsEqual(comp1, comp2)).toBe(false);
    });
  });

  describe('compositionToString和compositionFromString', () => {
    test('应该正确转换为字符串', () => {
      const composition = { townsfolk: 7, outsider: 0, minion: 2, demon: 1 };
      const str = RoleCompositionValidator.compositionToString(composition);
      
      expect(str).toBe('7T/0O/2M/1D');
    });

    test('应该正确从字符串解析', () => {
      const str = '7T/0O/2M/1D';
      const composition = RoleCompositionValidator.compositionFromString(str);
      
      expect(composition).toEqual({
        townsfolk: 7,
        outsider: 0,
        minion: 2,
        demon: 1
      });
    });

    test('往返转换应该保持一致', () => {
      const original = { townsfolk: 9, outsider: 2, minion: 3, demon: 1 };
      const str = RoleCompositionValidator.compositionToString(original);
      const parsed = RoleCompositionValidator.compositionFromString(str);
      
      expect(parsed).toEqual(original);
    });

    test('无效字符串应该返回null', () => {
      const result = RoleCompositionValidator.compositionFromString('invalid');
      
      expect(result).toBeNull();
    });
  });

  describe('集成测试 - 所有有效玩家数量', () => {
    test('所有标准角色组合都应该通过验证', () => {
      for (let playerCount = 5; playerCount <= 15; playerCount++) {
        const composition = getRoleComposition(playerCount);
        const result = RoleCompositionValidator.validate(composition, playerCount);
        
        expect(result.valid).toBe(true);
        expect(result.errors).toHaveLength(0);
      }
    });
  });
});
