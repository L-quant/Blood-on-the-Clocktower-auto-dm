/**
 * 测试脚本定义和角色组合配置
 */

import {
  ScriptType,
  ScriptDefinition,
  TroubleBrewingScript,
  BadMoonRisingScript,
  SectsAndVioletsScript,
  getScriptDefinition,
  getAllScripts,
  getRoleComposition,
  hasSpecialRule,
  getSpecialRule
} from '../../../src/automation/data/ScriptDefinitions';
import { Team } from '../../../src/automation/types/GameTypes';

describe('ScriptDefinitions', () => {
  describe('ScriptDefinition类', () => {
    test('应该正确创建脚本定义实例', () => {
      const script = new ScriptDefinition('test', 'Test Script', 'tb', ['washerwoman', 'imp']);
      
      expect(script.id).toBe('test');
      expect(script.name).toBe('Test Script');
      expect(script.edition).toBe('tb');
      expect(script.roles).toEqual(['washerwoman', 'imp']);
    });

    test('应该正确获取脚本中的角色', () => {
      const roles = TroubleBrewingScript.getRoles();
      
      expect(roles).toBeDefined();
      expect(roles.length).toBeGreaterThan(0);
      expect(roles.every(role => role.edition === 'tb')).toBe(true);
    });

    test('应该正确按团队获取角色', () => {
      const townsfolk = TroubleBrewingScript.getTownsfolk();
      const outsiders = TroubleBrewingScript.getOutsiders();
      const minions = TroubleBrewingScript.getMinions();
      const demons = TroubleBrewingScript.getDemons();
      
      expect(townsfolk.every(r => r.team === Team.TOWNSFOLK)).toBe(true);
      expect(outsiders.every(r => r.team === Team.OUTSIDER)).toBe(true);
      expect(minions.every(r => r.team === Team.MINION)).toBe(true);
      expect(demons.every(r => r.team === Team.DEMON)).toBe(true);
    });
  });

  describe('官方脚本定义', () => {
    test('Trouble Brewing脚本应该包含正确的角色', () => {
      expect(TroubleBrewingScript.id).toBe(ScriptType.TROUBLE_BREWING);
      expect(TroubleBrewingScript.name).toBe('Trouble Brewing');
      expect(TroubleBrewingScript.edition).toBe('tb');
      
      const roles = TroubleBrewingScript.getRoles();
      expect(roles.length).toBeGreaterThan(0);
      
      // 检查关键角色
      const roleIds = roles.map(r => r.id);
      expect(roleIds).toContain('washerwoman');
      expect(roleIds).toContain('imp');
      expect(roleIds).toContain('baron');
    });

    test('Bad Moon Rising脚本应该包含正确的角色', () => {
      expect(BadMoonRisingScript.id).toBe(ScriptType.BAD_MOON_RISING);
      expect(BadMoonRisingScript.name).toBe('Bad Moon Rising');
      expect(BadMoonRisingScript.edition).toBe('bmr');
      
      const roles = BadMoonRisingScript.getRoles();
      expect(roles.length).toBeGreaterThan(0);
      
      // 检查关键角色
      const roleIds = roles.map(r => r.id);
      expect(roleIds).toContain('godfather');
      expect(roleIds).toContain('zombuul');
    });

    test('Sects & Violets脚本应该包含正确的角色', () => {
      expect(SectsAndVioletsScript.id).toBe(ScriptType.SECTS_AND_VIOLETS);
      expect(SectsAndVioletsScript.name).toBe('Sects & Violets');
      expect(SectsAndVioletsScript.edition).toBe('snv');
      
      const roles = SectsAndVioletsScript.getRoles();
      expect(roles.length).toBeGreaterThan(0);
      
      // 检查关键角色
      const roleIds = roles.map(r => r.id);
      expect(roleIds).toContain('clockmaker');
      expect(roleIds).toContain('fanggu');
    });
  });

  describe('getScriptDefinition', () => {
    test('应该返回正确的脚本定义', () => {
      expect(getScriptDefinition(ScriptType.TROUBLE_BREWING)).toBe(TroubleBrewingScript);
      expect(getScriptDefinition(ScriptType.BAD_MOON_RISING)).toBe(BadMoonRisingScript);
      expect(getScriptDefinition(ScriptType.SECTS_AND_VIOLETS)).toBe(SectsAndVioletsScript);
    });

    test('未知脚本类型应该返回默认脚本', () => {
      expect(getScriptDefinition('unknown')).toBe(TroubleBrewingScript);
    });
  });

  describe('getAllScripts', () => {
    test('应该返回所有可用脚本', () => {
      const scripts = getAllScripts();
      
      expect(scripts).toHaveLength(3);
      expect(scripts).toContain(TroubleBrewingScript);
      expect(scripts).toContain(BadMoonRisingScript);
      expect(scripts).toContain(SectsAndVioletsScript);
    });
  });

  describe('getRoleComposition', () => {
    test('应该返回5人游戏的正确角色组合', () => {
      const composition = getRoleComposition(5);
      
      expect(composition).toEqual({
        townsfolk: 3,
        outsider: 0,
        minion: 1,
        demon: 1
      });
    });

    test('应该返回7人游戏的正确角色组合', () => {
      const composition = getRoleComposition(7);
      
      expect(composition).toEqual({
        townsfolk: 5,
        outsider: 0,
        minion: 1,
        demon: 1
      });
    });

    test('应该返回10人游戏的正确角色组合', () => {
      const composition = getRoleComposition(10);
      
      expect(composition).toEqual({
        townsfolk: 7,
        outsider: 0,
        minion: 2,
        demon: 1
      });
    });

    test('应该返回15人游戏的正确角色组合', () => {
      const composition = getRoleComposition(15);
      
      expect(composition).toEqual({
        townsfolk: 9,
        outsider: 2,
        minion: 3,
        demon: 1
      });
    });

    test('玩家数量小于5应该抛出错误', () => {
      expect(() => getRoleComposition(4)).toThrow('Invalid player count');
    });

    test('玩家数量大于15应该抛出错误', () => {
      expect(() => getRoleComposition(16)).toThrow('Invalid player count');
    });

    test('所有有效玩家数量的角色总数应该等于玩家数量', () => {
      for (let playerCount = 5; playerCount <= 15; playerCount++) {
        const composition = getRoleComposition(playerCount);
        const total = composition.townsfolk + composition.outsider + 
                     composition.minion + composition.demon;
        
        expect(total).toBe(playerCount);
      }
    });
  });

  describe('特殊角色规则', () => {
    test('应该正确识别特殊角色', () => {
      expect(hasSpecialRule('drunk')).toBe(true);
      expect(hasSpecialRule('baron')).toBe(true);
      expect(hasSpecialRule('godfather')).toBe(true);
      expect(hasSpecialRule('lunatic')).toBe(true);
      expect(hasSpecialRule('spy')).toBe(true);
      expect(hasSpecialRule('recluse')).toBe(true);
      
      expect(hasSpecialRule('washerwoman')).toBe(false);
      expect(hasSpecialRule('imp')).toBe(false);
    });

    test('应该返回正确的特殊规则', () => {
      const drunkRule = getSpecialRule('drunk');
      expect(drunkRule).toBeDefined();
      expect(drunkRule.type).toBe('disguise');
      expect(drunkRule.actualTeam).toBe(Team.OUTSIDER);
      expect(drunkRule.perceivedTeam).toBe(Team.TOWNSFOLK);
      
      const baronRule = getSpecialRule('baron');
      expect(baronRule).toBeDefined();
      expect(baronRule.type).toBe('setup_modifier');
      expect(baronRule.modifyComposition).toBeDefined();
    });

    test('男爵规则应该正确修改角色组合', () => {
      const baronRule = getSpecialRule('baron');
      const originalComposition = { townsfolk: 7, outsider: 0, minion: 2, demon: 1 };
      const modifiedComposition = baronRule.modifyComposition(originalComposition);
      
      expect(modifiedComposition.townsfolk).toBe(5);
      expect(modifiedComposition.outsider).toBe(2);
      expect(modifiedComposition.minion).toBe(2);
      expect(modifiedComposition.demon).toBe(1);
    });

    test('不存在的角色应该返回null', () => {
      expect(getSpecialRule('nonexistent')).toBeNull();
    });
  });
});
