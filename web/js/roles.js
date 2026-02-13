// ============================================
// 角色数据定义
// ============================================

export const ROLE_DATA = {
  // --- 村民 (Townsfolk) ---
  washerwoman:  { id: 'washerwoman',  name: '洗衣妇',     type: 'townsfolk', team: 'good', icon: '👗', desc: '首夜获知2名玩家中的1名是某个特定村民' },
  librarian:    { id: 'librarian',    name: '图书管理员', type: 'townsfolk', team: 'good', icon: '📚', desc: '首夜获知2名玩家中的1名是某个特定外来者（或无外来者）' },
  investigator: { id: 'investigator', name: '调查员',     type: 'townsfolk', team: 'good', icon: '🔍', desc: '首夜获知2名玩家中的1名是某个特定爪牙' },
  chef:         { id: 'chef',         name: '厨师',       type: 'townsfolk', team: 'good', icon: '🍳', desc: '首夜获知有多少对邪恶玩家相邻而坐' },
  empath:       { id: 'empath',       name: '共情者',     type: 'townsfolk', team: 'good', icon: '💞', desc: '每夜获知你两侧存活邻居中有几个邪恶玩家' },
  fortune_teller:{ id: 'fortune_teller', name: '占卜师',  type: 'townsfolk', team: 'good', icon: '🔮', desc: '每夜选2名玩家，获知其中是否有恶魔' },
  undertaker:   { id: 'undertaker',   name: '掘墓人',     type: 'townsfolk', team: 'good', icon: '⚰️', desc: '如果白天有人被处决，当夜获知其真实角色' },
  monk:         { id: 'monk',         name: '僧侣',       type: 'townsfolk', team: 'good', icon: '🙏', desc: '每夜（首夜除外）选一名其他玩家保护其免受恶魔攻击' },
  ravenkeeper:  { id: 'ravenkeeper',  name: '守鸦人',     type: 'townsfolk', team: 'good', icon: '🐦‍⬛', desc: '若你在夜间死亡，可立刻选一名玩家查看其角色' },
  virgin:       { id: 'virgin',       name: '贞洁者',     type: 'townsfolk', team: 'good', icon: '🌸', desc: '如果村民提名你，该提名者立即死亡（仅一次）' },
  slayer:       { id: 'slayer',       name: '杀手',       type: 'townsfolk', team: 'good', icon: '🔫', desc: '白天使用一次：指定一名玩家，若其为恶魔则立即死亡' },
  soldier:      { id: 'soldier',      name: '士兵',       type: 'townsfolk', team: 'good', icon: '🛡️', desc: '恶魔无法杀死你' },
  mayor:        { id: 'mayor',        name: '市长',       type: 'townsfolk', team: 'good', icon: '🎩', desc: '若存活至最后3人且无人被处决，善良阵营获胜' },

  // --- 外来者 (Outsider) ---
  butler:  { id: 'butler',  name: '管家',   type: 'outsider', team: 'good', icon: '🎭', desc: '每夜选一名玩家为主人，投票时只能在主人投票后才能投票' },
  drunk:   { id: 'drunk',   name: '酒鬼',   type: 'outsider', team: 'good', icon: '🍺', desc: '你以为自己是某个村民角色，但实际上你是酒鬼，能力无效' },
  recluse: { id: 'recluse', name: '隐士',   type: 'outsider', team: 'good', icon: '🏚️', desc: '你可能被识别为邪恶角色' },
  saint:   { id: 'saint',   name: '圣徒',   type: 'outsider', team: 'good', icon: '😇', desc: '如果你被处决，邪恶阵营获胜' },

  // --- 爪牙 (Minion) ---
  poisoner:      { id: 'poisoner',      name: '投毒者',     type: 'minion', team: 'evil', icon: '☠️', desc: '每夜选一名玩家中毒，其能力失效且信息可能为假' },
  spy:           { id: 'spy',           name: '间谍',       type: 'minion', team: 'evil', icon: '🕵️', desc: '每夜可查看所有角色信息，且可被视为善良角色' },
  scarlet_woman: { id: 'scarlet_woman', name: '红衣女郎',   type: 'minion', team: 'evil', icon: '💃', desc: '当恶魔死亡且存活≥5人时，你成为新的恶魔' },
  baron:         { id: 'baron',         name: '男爵',       type: 'minion', team: 'evil', icon: '🎖️', desc: '游戏中额外增加2名外来者' },

  // --- 恶魔 (Demon) ---
  imp: { id: 'imp', name: '小恶魔', type: 'demon', team: 'evil', icon: '👿', desc: '每夜选一名玩家杀死；选自己则将恶魔身份转移给一名爪牙' },
};

// Player count -> role composition
export const COMPOSITION = {
  5:  { townsfolk: 3, outsider: 0, minion: 1, demon: 1 },
  6:  { townsfolk: 3, outsider: 1, minion: 1, demon: 1 },
  7:  { townsfolk: 5, outsider: 0, minion: 1, demon: 1 },
  8:  { townsfolk: 5, outsider: 1, minion: 1, demon: 1 },
  9:  { townsfolk: 5, outsider: 2, minion: 1, demon: 1 },
  10: { townsfolk: 7, outsider: 0, minion: 2, demon: 1 },
  11: { townsfolk: 7, outsider: 1, minion: 2, demon: 1 },
  12: { townsfolk: 7, outsider: 2, minion: 2, demon: 1 },
  13: { townsfolk: 9, outsider: 0, minion: 3, demon: 1 },
  14: { townsfolk: 9, outsider: 1, minion: 3, demon: 1 },
  15: { townsfolk: 9, outsider: 2, minion: 3, demon: 1 },
};

export const EDITIONS = [
  { id: 'tb', name: '暗流涌动', nameEn: 'Trouble Brewing', icon: '🩸', desc: '入门剧本，适合新手' },
];

export function getRoleData(roleId) {
  return ROLE_DATA[roleId] || { id: roleId, name: roleId, type: 'unknown', team: 'unknown', icon: '❓', desc: '' };
}

export function getRoleName(roleId) {
  return getRoleData(roleId).name;
}

export function getRoleIcon(roleId) {
  return getRoleData(roleId).icon;
}

export function getComposition(playerCount) {
  return COMPOSITION[playerCount] || COMPOSITION[5];
}
