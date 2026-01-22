/**
 * 角色组合验证器
 * 验证角色组合是否符合游戏规则
 */

import { Team } from '../types/GameTypes';
import { getRoleComposition, hasSpecialRule, getSpecialRule } from '../data/ScriptDefinitions';

/**
 * 角色组合验证器类
 */
export default class RoleCompositionValidator {
  /**
   * 验证角色组合是否有效
   * @param {object} composition 角色组合 {townsfolk, outsider, minion, demon}
   * @param {number} playerCount 玩家数量
   * @returns {object} {valid: boolean, errors: string[]}
   */
  static validate(composition, playerCount) {
    const errors = [];

    // 检查玩家数量是否有效
    if (playerCount < 5 || playerCount > 15) {
      errors.push(`Invalid player count: ${playerCount}. Must be between 5 and 15.`);
      return { valid: false, errors };
    }

    // 检查角色总数是否匹配玩家数量
    const totalRoles = composition.townsfolk + composition.outsider + 
                       composition.minion + composition.demon;
    
    if (totalRoles !== playerCount) {
      errors.push(
        `Total roles (${totalRoles}) does not match player count (${playerCount})`
      );
    }

    // 检查必须有至少一个恶魔
    if (composition.demon < 1) {
      errors.push('Must have at least 1 Demon');
    }

    // 检查恶魔数量不能超过1（标准规则）
    if (composition.demon > 1) {
      errors.push('Cannot have more than 1 Demon in standard games');
    }

    // 检查爪牙数量是否合理
    const expectedComposition = getRoleComposition(playerCount);
    if (composition.minion > expectedComposition.minion + 1) {
      errors.push(`Too many Minions: ${composition.minion}`);
    }

    // 检查外来者数量是否合理
    if (composition.outsider > expectedComposition.outsider + 2) {
      errors.push(`Too many Outsiders: ${composition.outsider}`);
    }

    // 检查村民数量是否为正数
    if (composition.townsfolk < 0) {
      errors.push('Townsfolk count cannot be negative');
    }

    return {
      valid: errors.length === 0,
      errors
    };
  }

  /**
   * 验证角色列表是否符合组合要求
   * @param {Role[]} roles 角色列表
   * @param {object} composition 期望的角色组合
   * @returns {object} {valid: boolean, errors: string[]}
   */
  static validateRoleList(roles, composition) {
    const errors = [];

    // 统计各团队的角色数量
    const counts = {
      townsfolk: 0,
      outsider: 0,
      minion: 0,
      demon: 0
    };

    roles.forEach(role => {
      switch (role.team) {
        case Team.TOWNSFOLK:
          counts.townsfolk++;
          break;
        case Team.OUTSIDER:
          counts.outsider++;
          break;
        case Team.MINION:
          counts.minion++;
          break;
        case Team.DEMON:
          counts.demon++;
          break;
      }
    });

    // 检查每个团队的数量是否匹配
    Object.keys(composition).forEach(team => {
      if (counts[team] !== composition[team]) {
        errors.push(
          `${team} count mismatch: expected ${composition[team]}, got ${counts[team]}`
        );
      }
    });

    // 检查是否有重复角色
    const roleIds = roles.map(r => r.id);
    const uniqueRoleIds = new Set(roleIds);
    if (roleIds.length !== uniqueRoleIds.size) {
      errors.push('Duplicate roles detected');
    }

    return {
      valid: errors.length === 0,
      errors
    };
  }

  /**
   * 应用特殊角色规则修改组合
   * @param {object} composition 原始角色组合
   * @param {Role[]} selectedRoles 已选择的角色
   * @returns {object} 修改后的角色组合
   */
  static applySpecialRules(composition, selectedRoles) {
    let modifiedComposition = { ...composition };

    selectedRoles.forEach(role => {
      if (hasSpecialRule(role.id)) {
        const rule = getSpecialRule(role.id);
        
        if (rule.type === 'setup_modifier' && rule.modifyComposition) {
          modifiedComposition = rule.modifyComposition(modifiedComposition);
        }
      }
    });

    return modifiedComposition;
  }

  /**
   * 验证角色组合是否可以通过特殊规则调整达到
   * @param {object} targetComposition 目标组合
   * @param {object} baseComposition 基础组合
   * @param {Role[]} availableSpecialRoles 可用的特殊角色
   * @returns {object} {achievable: boolean, requiredRoles: Role[]}
   */
  static canAchieveComposition(targetComposition, baseComposition, availableSpecialRoles) {
    // 计算差异
    const diff = {
      townsfolk: targetComposition.townsfolk - baseComposition.townsfolk,
      outsider: targetComposition.outsider - baseComposition.outsider,
      minion: targetComposition.minion - baseComposition.minion,
      demon: targetComposition.demon - baseComposition.demon
    };

    // 尝试找到能够达成目标的特殊角色组合
    const requiredRoles = [];

    // 检查男爵（+2外来者，-2村民）
    if (diff.outsider === 2 && diff.townsfolk === -2) {
      const baron = availableSpecialRoles.find(r => r.id === 'baron');
      if (baron) {
        requiredRoles.push(baron);
        return { achievable: true, requiredRoles };
      }
    }

    // 检查教父（±1外来者，∓1村民）
    if (Math.abs(diff.outsider) === 1 && Math.abs(diff.townsfolk) === 1) {
      const godfather = availableSpecialRoles.find(r => r.id === 'godfather');
      if (godfather) {
        requiredRoles.push(godfather);
        return { achievable: true, requiredRoles };
      }
    }

    // 如果没有差异，直接可达成
    if (Object.values(diff).every(d => d === 0)) {
      return { achievable: true, requiredRoles: [] };
    }

    return { achievable: false, requiredRoles: [] };
  }

  /**
   * 获取角色组合的统计信息
   * @param {object} composition 角色组合
   * @returns {object} 统计信息
   */
  static getCompositionStats(composition) {
    const total = composition.townsfolk + composition.outsider + 
                  composition.minion + composition.demon;
    
    const goodCount = composition.townsfolk + composition.outsider;
    const evilCount = composition.minion + composition.demon;

    return {
      total,
      goodCount,
      evilCount,
      goodPercentage: (goodCount / total * 100).toFixed(1),
      evilPercentage: (evilCount / total * 100).toFixed(1),
      townsfolkPercentage: (composition.townsfolk / total * 100).toFixed(1),
      outsiderPercentage: (composition.outsider / total * 100).toFixed(1),
      minionPercentage: (composition.minion / total * 100).toFixed(1),
      demonPercentage: (composition.demon / total * 100).toFixed(1)
    };
  }

  /**
   * 比较两个角色组合
   * @param {object} comp1 组合1
   * @param {object} comp2 组合2
   * @returns {boolean} 是否相同
   */
  static areCompositionsEqual(comp1, comp2) {
    return comp1.townsfolk === comp2.townsfolk &&
           comp1.outsider === comp2.outsider &&
           comp1.minion === comp2.minion &&
           comp1.demon === comp2.demon;
  }

  /**
   * 生成角色组合的字符串表示
   * @param {object} composition 角色组合
   * @returns {string}
   */
  static compositionToString(composition) {
    return `${composition.townsfolk}T/${composition.outsider}O/${composition.minion}M/${composition.demon}D`;
  }

  /**
   * 从字符串解析角色组合
   * @param {string} str 字符串表示
   * @returns {object|null} 角色组合或null
   */
  static compositionFromString(str) {
    const match = str.match(/(\d+)T\/(\d+)O\/(\d+)M\/(\d+)D/);
    if (!match) return null;

    return {
      townsfolk: parseInt(match[1]),
      outsider: parseInt(match[2]),
      minion: parseInt(match[3]),
      demon: parseInt(match[4])
    };
  }
}