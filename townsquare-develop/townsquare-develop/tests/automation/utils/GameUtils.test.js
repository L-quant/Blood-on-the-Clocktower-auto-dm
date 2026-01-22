/**
 * GameUtils 测试
 */

import {
  getStandardRoleConfiguration,
  validateRoleComposition,
  isEvilRole,
  isGoodRole,
  getRoleNightPriority,
  sortRolesByNightOrder,
  calculateMajorityThreshold,
  generateId,
  deepClone
} from '@/automation/utils/GameUtils';
import { Role, Team } from '@/automation/types/GameTypes';

describe('GameUtils', () => {
  describe('getStandardRoleConfiguration', () => {
    test('should return correct configuration for 5 players', () => {
      const config = getStandardRoleConfiguration(5);
      expect(config).toEqual({
        townsfolk: 3,
        outsider: 0,
        minion: 1,
        demon: 1
      });
    });

    test('should return correct configuration for 10 players', () => {
      const config = getStandardRoleConfiguration(10);
      expect(config).toEqual({
        townsfolk: 7,
        outsider: 0,
        minion: 2,
        demon: 1
      });
    });

    test('should throw error for invalid player count', () => {
      expect(() => getStandardRoleConfiguration(3)).toThrow();
      expect(() => getStandardRoleConfiguration(16)).toThrow();
    });
  });

  describe('validateRoleComposition', () => {
    test('should validate correct composition', () => {
      const composition = { townsfolk: 3, outsider: 0, minion: 1, demon: 1 };
      expect(validateRoleComposition(composition, 5)).toBe(true);
    });

    test('should reject incorrect composition', () => {
      const composition = { townsfolk: 3, outsider: 0, minion: 1, demon: 1 };
      expect(validateRoleComposition(composition, 6)).toBe(false);
    });

    test('should handle null composition', () => {
      expect(validateRoleComposition(null, 5)).toBe(false);
    });
  });

  describe('isEvilRole and isGoodRole', () => {
    test('should identify evil roles correctly', () => {
      const minionRole = new Role({
        id: 'poisoner',
        name: 'Poisoner',
        team: Team.MINION,
        ability: 'Test ability'
      });

      const demonRole = new Role({
        id: 'imp',
        name: 'Imp',
        team: Team.DEMON,
        ability: 'Test ability'
      });

      expect(isEvilRole(minionRole)).toBe(true);
      expect(isEvilRole(demonRole)).toBe(true);
      expect(isGoodRole(minionRole)).toBe(false);
      expect(isGoodRole(demonRole)).toBe(false);
    });

    test('should identify good roles correctly', () => {
      const townsfolkRole = new Role({
        id: 'washerwoman',
        name: 'Washerwoman',
        team: Team.TOWNSFOLK,
        ability: 'Test ability'
      });

      const outsiderRole = new Role({
        id: 'butler',
        name: 'Butler',
        team: Team.OUTSIDER,
        ability: 'Test ability'
      });

      expect(isGoodRole(townsfolkRole)).toBe(true);
      expect(isGoodRole(outsiderRole)).toBe(true);
      expect(isEvilRole(townsfolkRole)).toBe(false);
      expect(isEvilRole(outsiderRole)).toBe(false);
    });
  });

  describe('getRoleNightPriority', () => {
    test('should return first night priority when isFirstNight is true', () => {
      const role = new Role({
        id: 'test',
        name: 'Test',
        team: Team.TOWNSFOLK,
        ability: 'Test',
        firstNight: 10,
        otherNight: 20
      });

      expect(getRoleNightPriority(role, true)).toBe(10);
      expect(getRoleNightPriority(role, false)).toBe(20);
    });

    test('should return 0 for roles without night actions', () => {
      const role = new Role({
        id: 'test',
        name: 'Test',
        team: Team.TOWNSFOLK,
        ability: 'Test'
      });

      expect(getRoleNightPriority(role, true)).toBe(0);
      expect(getRoleNightPriority(role, false)).toBe(0);
    });
  });

  describe('sortRolesByNightOrder', () => {
    test('should sort roles by night order', () => {
      const roles = [
        new Role({
          id: 'role3',
          name: 'Role 3',
          team: Team.TOWNSFOLK,
          ability: 'Test',
          firstNight: 30
        }),
        new Role({
          id: 'role1',
          name: 'Role 1',
          team: Team.TOWNSFOLK,
          ability: 'Test',
          firstNight: 10
        }),
        new Role({
          id: 'role2',
          name: 'Role 2',
          team: Team.TOWNSFOLK,
          ability: 'Test',
          firstNight: 20
        })
      ];

      const sorted = sortRolesByNightOrder(roles, true);
      expect(sorted[0].id).toBe('role1');
      expect(sorted[1].id).toBe('role2');
      expect(sorted[2].id).toBe('role3');
    });

    test('should filter out roles without night actions', () => {
      const roles = [
        new Role({
          id: 'active',
          name: 'Active',
          team: Team.TOWNSFOLK,
          ability: 'Test',
          firstNight: 10
        }),
        new Role({
          id: 'passive',
          name: 'Passive',
          team: Team.TOWNSFOLK,
          ability: 'Test'
        })
      ];

      const sorted = sortRolesByNightOrder(roles, true);
      expect(sorted).toHaveLength(1);
      expect(sorted[0].id).toBe('active');
    });
  });

  describe('calculateMajorityThreshold', () => {
    test('should calculate correct majority threshold', () => {
      expect(calculateMajorityThreshold(5)).toBe(3);
      expect(calculateMajorityThreshold(6)).toBe(3);
      expect(calculateMajorityThreshold(7)).toBe(4);
      expect(calculateMajorityThreshold(8)).toBe(4);
    });
  });

  describe('generateId', () => {
    test('should generate unique IDs', () => {
      const id1 = generateId();
      const id2 = generateId();
      
      expect(id1).not.toBe(id2);
      expect(typeof id1).toBe('string');
      expect(id1.length).toBeGreaterThan(0);
    });
  });

  describe('deepClone', () => {
    test('should deep clone objects', () => {
      const original = {
        a: 1,
        b: {
          c: 2,
          d: [3, 4, { e: 5 }]
        }
      };

      const cloned = deepClone(original);
      
      expect(cloned).toEqual(original);
      expect(cloned).not.toBe(original);
      expect(cloned.b).not.toBe(original.b);
      expect(cloned.b.d).not.toBe(original.b.d);
    });

    test('should handle primitive values', () => {
      expect(deepClone(null)).toBe(null);
      expect(deepClone(undefined)).toBe(undefined);
      expect(deepClone(42)).toBe(42);
      expect(deepClone('string')).toBe('string');
      expect(deepClone(true)).toBe(true);
    });

    test('should handle dates', () => {
      const date = new Date();
      const cloned = deepClone(date);
      
      expect(cloned).toEqual(date);
      expect(cloned).not.toBe(date);
    });
  });
});