/**
 * 游戏相关工具函数
 */

import { Team } from '../types/GameTypes';

/**
 * 根据玩家数量获取标准角色配置
 * @param {number} playerCount 玩家数量
 * @returns {object} 角色配置 {townsfolk, outsider, minion, demon}
 */
export function getStandardRoleConfiguration(playerCount) {
  // 基于官方规则的角色配置表
  const configurations = [
    null, // 0 players
    null, // 1 player
    null, // 2 players
    null, // 3 players
    null, // 4 players
    { townsfolk: 3, outsider: 0, minion: 1, demon: 1 }, // 5 players
    { townsfolk: 3, outsider: 1, minion: 1, demon: 1 }, // 6 players
    { townsfolk: 5, outsider: 0, minion: 1, demon: 1 }, // 7 players
    { townsfolk: 5, outsider: 1, minion: 1, demon: 1 }, // 8 players
    { townsfolk: 5, outsider: 2, minion: 1, demon: 1 }, // 9 players
    { townsfolk: 7, outsider: 0, minion: 2, demon: 1 }, // 10 players
    { townsfolk: 7, outsider: 1, minion: 2, demon: 1 }, // 11 players
    { townsfolk: 7, outsider: 2, minion: 2, demon: 1 }, // 12 players
    { townsfolk: 9, outsider: 0, minion: 3, demon: 1 }, // 13 players
    { townsfolk: 9, outsider: 1, minion: 3, demon: 1 }, // 14 players
    { townsfolk: 9, outsider: 2, minion: 3, demon: 1 }  // 15 players
  ];

  if (playerCount < 5 || playerCount > 15) {
    throw new Error(`Invalid player count: ${playerCount}. Must be between 5 and 15.`);
  }

  return configurations[playerCount];
}

/**
 * 验证角色组合是否符合玩家数量
 * @param {object} composition 角色组合
 * @param {number} playerCount 玩家数量
 * @returns {boolean}
 */
export function validateRoleComposition(composition, playerCount) {
  if (!composition) return false;
  
  const total = composition.townsfolk + composition.outsider + composition.minion + composition.demon;
  return total === playerCount;
}

/**
 * 检查角色是否为恶人
 * @param {Role} role 角色
 * @returns {boolean}
 */
export function isEvilRole(role) {
  return role.team === Team.MINION || role.team === Team.DEMON;
}

/**
 * 检查角色是否为好人
 * @param {Role} role 角色
 * @returns {boolean}
 */
export function isGoodRole(role) {
  return role.team === Team.TOWNSFOLK || role.team === Team.OUTSIDER;
}

/**
 * 获取角色的夜间行动优先级
 * @param {Role} role 角色
 * @param {boolean} isFirstNight 是否为第一夜
 * @returns {number}
 */
export function getRoleNightPriority(role, isFirstNight = false) {
  if (isFirstNight) {
    return role.firstNight || 0;
  }
  return role.otherNight || 0;
}

/**
 * 按夜间顺序排序角色
 * @param {Role[]} roles 角色列表
 * @param {boolean} isFirstNight 是否为第一夜
 * @returns {Role[]}
 */
export function sortRolesByNightOrder(roles, isFirstNight = false) {
  return roles
    .filter(role => getRoleNightPriority(role, isFirstNight) > 0)
    .sort((a, b) => {
      const priorityA = getRoleNightPriority(a, isFirstNight);
      const priorityB = getRoleNightPriority(b, isFirstNight);
      return priorityA - priorityB;
    });
}

/**
 * 计算多数票阈值
 * @param {number} alivePlayerCount 存活玩家数量
 * @returns {number}
 */
export function calculateMajorityThreshold(alivePlayerCount) {
  return Math.ceil(alivePlayerCount / 2);
}

/**
 * 生成唯一ID
 * @returns {string}
 */
export function generateId() {
  return Math.random().toString(36).substr(2, 9) + Date.now().toString(36);
}

/**
 * 深拷贝对象
 * @param {any} obj 要拷贝的对象
 * @returns {any}
 */
export function deepClone(obj) {
  if (obj === null || typeof obj !== 'object') return obj;
  if (obj instanceof Date) return new Date(obj.getTime());
  if (obj instanceof Array) return obj.map(item => deepClone(item));
  if (typeof obj === 'object') {
    const clonedObj = {};
    for (const key in obj) {
      if (Object.prototype.hasOwnProperty.call(obj, key)) {
        clonedObj[key] = deepClone(obj[key]);
      }
    }
    return clonedObj;
  }
}