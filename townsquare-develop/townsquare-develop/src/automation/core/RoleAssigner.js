/**
 * 角色分配器
 * 负责自动分配角色给玩家
 */

import {
  getScriptDefinition,
  getRoleComposition,
  hasSpecialRule,
  getSpecialRule
} from '../data/ScriptDefinitions';
import RoleCompositionValidator from '../utils/RoleCompositionValidator';
import { Team } from '../types/GameTypes';

/**
 * 角色分配器类
 */
export default class RoleAssigner {
  constructor() {
    this.maxRetries = 3;
  }

  /**
   * 生成角色组合
   * @param {number} playerCount 玩家数量
   * @param {string} scriptType 脚本类型
   * @returns {object} 角色组合 {townsfolk, outsider, minion, demon}
   */
  generateRoleComposition(playerCount, scriptType) {
    // 验证玩家数量
    if (playerCount < 5 || playerCount > 15) {
      throw new Error(`Invalid player count: ${playerCount}. Must be between 5 and 15.`);
    }

    // 获取基础角色组合
    const baseComposition = getRoleComposition(playerCount);
    
    return { ...baseComposition };
  }

  /**
   * 分配角色给玩家
   * @param {Player[]} players 玩家列表
   * @param {string} scriptType 脚本类型
   * @param {object} options 选项 {privacyMode: boolean}
   * @returns {Promise<RoleAssignment[]>} 角色分配结果
   */
  async assignRolesToPlayers(players, scriptType, options = {}) {
    if (!players || players.length === 0) {
      throw new Error('Player list cannot be empty');
    }

    const playerCount = players.length;
    let attempts = 0;
    let lastError = null;

    // 重试机制
    while (attempts < this.maxRetries) {
      try {
        attempts++;
        
        // 生成角色组合
        const composition = this.generateRoleComposition(playerCount, scriptType);
        
        // 获取脚本定义
        const script = getScriptDefinition(scriptType);
        
        // 选择角色（不包含会修改组合的特殊角色）
        const selectedRoles = this._selectRoles(composition, script);
        
        // 验证角色列表
        const validation = RoleCompositionValidator.validateRoleList(
          selectedRoles,
          composition
        );
        
        if (!validation.valid) {
          throw new Error(`Role validation failed: ${validation.errors.join(', ')}`);
        }
        
        // 分配角色给玩家
        const assignments = this._assignRolesToPlayers(players, selectedRoles);
        
        // 如果是隐私模式，不在分配结果中暴露其他玩家的角色
        if (options.privacyMode) {
          console.log('Privacy mode enabled: role assignments will be filtered per player');
        }
        
        return assignments;
        
      } catch (error) {
        lastError = error;
        console.warn(`Role assignment attempt ${attempts} failed:`, error.message);
        
        if (attempts >= this.maxRetries) {
          throw new Error(
            `Failed to assign roles after ${this.maxRetries} attempts. Last error: ${lastError.message}`
          );
        }
      }
    }
  }

  /**
   * 从脚本中选择角色
   * @private
   * @param {object} composition 角色组合
   * @param {ScriptDefinition} script 脚本定义
   * @returns {Role[]} 选择的角色列表
   */
  _selectRoles(composition, script) {
    const selectedRoles = [];
    
    // 选择恶魔
    const demons = script.getDemons();
    if (demons.length === 0) {
      throw new Error('No demons available in script');
    }
    const selectedDemon = this._randomSelect(demons, 1)[0];
    selectedRoles.push(selectedDemon);
    
    // 选择爪牙
    const minions = script.getMinions();
    if (minions.length < composition.minion) {
      throw new Error(`Not enough minions in script. Need ${composition.minion}, have ${minions.length}`);
    }
    const selectedMinions = this._randomSelect(minions, composition.minion);
    selectedRoles.push(...selectedMinions);
    
    // 选择外来者
    const outsiders = script.getOutsiders();
    if (outsiders.length < composition.outsider) {
      throw new Error(`Not enough outsiders in script. Need ${composition.outsider}, have ${outsiders.length}`);
    }
    const selectedOutsiders = this._randomSelect(outsiders, composition.outsider);
    selectedRoles.push(...selectedOutsiders);
    
    // 选择村民
    const townsfolk = script.getTownsfolk();
    if (townsfolk.length < composition.townsfolk) {
      throw new Error(`Not enough townsfolk in script. Need ${composition.townsfolk}, have ${townsfolk.length}`);
    }
    const selectedTownsfolk = this._randomSelect(townsfolk, composition.townsfolk);
    selectedRoles.push(...selectedTownsfolk);
    
    return selectedRoles;
  }

  /**
   * 将角色分配给玩家
   * @private
   * @param {Player[]} players 玩家列表
   * @param {Role[]} roles 角色列表
   * @returns {RoleAssignment[]} 角色分配结果
   */
  _assignRolesToPlayers(players, roles) {
    if (players.length !== roles.length) {
      throw new Error(`Player count (${players.length}) does not match role count (${roles.length})`);
    }
    
    // 打乱角色顺序
    const shuffledRoles = this._shuffle([...roles]);
    
    // 创建分配结果
    const assignments = players.map((player, index) => {
      const role = shuffledRoles[index];
      
      return {
        playerId: player.id,
        playerName: player.name,
        role: role,
        actualRole: role,
        perceivedRole: role,
        isEvil: this._isEvilTeam(role.team),
        specialRuleApplied: false
      };
    });
    
    // 处理特殊角色分配
    this._handleSpecialRoleAssignments(assignments, roles);
    
    return assignments;
  }

  /**
   * 处理特殊角色分配逻辑
   * @private
   * @param {RoleAssignment[]} assignments 角色分配列表
   * @param {Role[]} allRoles 所有角色列表
   */
  _handleSpecialRoleAssignments(assignments, allRoles) {
    assignments.forEach(assignment => {
      const roleId = assignment.role.id;
      
      if (hasSpecialRule(roleId)) {
        const rule = getSpecialRule(roleId);
        
        // 处理伪装类型的特殊角色（酒鬼、疯子）
        if (rule.type === 'disguise' && rule.handler) {
          let availableRoles = [];
          
          if (roleId === 'drunk') {
            // 酒鬼认为自己是村民
            availableRoles = allRoles.filter(r => r.team === Team.TOWNSFOLK);
          } else if (roleId === 'lunatic') {
            // 疯子认为自己是恶魔
            availableRoles = allRoles.filter(r => r.team === Team.DEMON);
          }
          
          if (availableRoles.length > 0) {
            const result = rule.handler(assignment, availableRoles);
            assignment.perceivedRole = allRoles.find(r => r.id === result.perceivedRole) || assignment.role;
            assignment.specialRuleApplied = true;
          }
        }
      }
    });
  }

  /**
   * 判断团队是否为恶人
   * @private
   * @param {string} team 团队类型
   * @returns {boolean}
   */
  _isEvilTeam(team) {
    return team === Team.MINION || team === Team.DEMON;
  }

  /**
   * 从数组中随机选择指定数量的元素
   * @private
   * @param {Array} array 源数组
   * @param {number} count 选择数量
   * @returns {Array} 选择的元素
   */
  _randomSelect(array, count) {
    if (count > array.length) {
      throw new Error(`Cannot select ${count} items from array of length ${array.length}`);
    }
    
    const shuffled = this._shuffle([...array]);
    return shuffled.slice(0, count);
  }

  /**
   * 打乱数组顺序（Fisher-Yates算法）
   * @private
   * @param {Array} array 要打乱的数组
   * @returns {Array} 打乱后的数组
   */
  _shuffle(array) {
    for (let i = array.length - 1; i > 0; i--) {
      const j = Math.floor(Math.random() * (i + 1));
      [array[i], array[j]] = [array[j], array[i]];
    }
    return array;
  }

  /**
   * 验证角色组合的合法性
   * @param {object} composition 角色组合
   * @param {number} playerCount 玩家数量
   * @returns {boolean} 是否合法
   */
  validateRoleComposition(composition, playerCount) {
    const result = RoleCompositionValidator.validate(composition, playerCount);
    return result.valid;
  }

  /**
   * 获取角色分配的摘要信息
   * @param {RoleAssignment[]} assignments 角色分配列表
   * @param {boolean} hideDetails 是否隐藏详细信息（用于隐私模式）
   * @returns {object} 摘要信息
   */
  getAssignmentSummary(assignments, hideDetails = false) {
    const summary = {
      totalPlayers: assignments.length,
      townsfolk: 0,
      outsider: 0,
      minion: 0,
      demon: 0,
      goodPlayers: 0,
      evilPlayers: 0,
      specialRolesApplied: 0
    };
    
    assignments.forEach(assignment => {
      const team = assignment.role.team;
      
      switch (team) {
        case Team.TOWNSFOLK:
          summary.townsfolk++;
          summary.goodPlayers++;
          break;
        case Team.OUTSIDER:
          summary.outsider++;
          summary.goodPlayers++;
          break;
        case Team.MINION:
          summary.minion++;
          summary.evilPlayers++;
          break;
        case Team.DEMON:
          summary.demon++;
          summary.evilPlayers++;
          break;
      }
      
      if (assignment.specialRuleApplied) {
        summary.specialRolesApplied++;
      }
    });
    
    // 如果启用隐私模式，不返回详细的角色信息
    if (!hideDetails) {
      summary.roles = assignments.map(a => ({
        playerId: a.playerId,
        playerName: a.playerName,
        role: a.role.name,
        team: a.role.team
      }));
    }
    
    return summary;
  }
}
