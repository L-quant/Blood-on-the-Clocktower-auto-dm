// Vuex 模块：夜晚行动覆盖层状态（轮次、目标选择、进度）
//
// [OUT] store/index.js（模块注册）
// [POS] 夜晚阶段 UI 状态，驱动 NightOverlay 交互流程

const state = () => ({
  isMyTurn: false,
  step: 'idle', // 'idle' | 'role_reveal' | 'team_reveal' | 'sleeping' | 'woken' | 'selecting' | 'waiting' | 'result' | 'done'
  roleId: '',
  roleName: '',
  abilityText: '',
  actionType: '', // 'select_one' | 'select_two' | 'passive' | 'info'
  targets: [], // available targets
  selectedTargets: [],
  result: '',
  pendingPrompt: null, // queued prompt data when not yet sleeping
  nightInfoDetail: null, // { roleId, infoType, content, message } from night.info event
  teamRecognition: null, // { team, demonId, minionIds, bluffs } from team.recognition event
  nightInfoHistory: [], // accumulated night info records across all nights
  grimoireHistory: [], // spy grimoire snapshots grouped by night
  progress: {
    current: 0,
    total: 0
  }
});

const mutations = {
  setMyTurn(state, isMyTurn) {
    state.isMyTurn = isMyTurn;
  },
  setStep(state, step) {
    state.step = step;
  },
  openPanel(state, { roleId, roleName, abilityText, actionType }) {
    state.isMyTurn = true;
    state.step = 'woken';
    state.roleId = roleId || '';
    state.roleName = roleName || '';
    state.abilityText = abilityText || '';
    state.actionType = actionType || 'passive';
    state.selectedTargets = [];
    state.result = '';
  },
  setTargets(state, targets) {
    state.targets = targets;
  },
  selectTarget(state, target) {
    if (state.actionType === 'select_one') {
      state.selectedTargets = [target];
    } else if (state.actionType === 'select_two') {
      if (state.selectedTargets.length < 2) {
        state.selectedTargets.push(target);
      }
    }
  },
  removeTarget(state, target) {
    state.selectedTargets = state.selectedTargets.filter(t => t !== target);
  },
  setResult(state, result) {
    state.result = result;
    state.step = 'result';
  },
  setNightInfoDetail(state, detail) {
    state.nightInfoDetail = detail;
  },
  setTeamRecognition(state, data) {
    state.teamRecognition = data;
  },
  showTeamReveal(state) {
    if (state.teamRecognition) {
      state.step = 'team_reveal';
      state.isMyTurn = true;
    }
  },
  pushNightInfo(state, entry) {
    state.nightInfoHistory.push(entry);
  },
  setGrimoireEntry(state, entry) {
    const idx = state.grimoireHistory.findIndex(item => item.nightNumber === entry.nightNumber);
    if (idx >= 0) {
      state.grimoireHistory.splice(idx, 1, entry);
      return;
    }
    state.grimoireHistory.push(entry);
  },
  clearNightInfoHistory(state) {
    state.nightInfoHistory = [];
  },
  clearGrimoireHistory(state) {
    state.grimoireHistory = [];
  },
  setProgress(state, { current, total }) {
    state.progress.current = current;
    state.progress.total = total;
  },
  closePanel(state) {
    state.isMyTurn = false;
    state.step = 'sleeping';
    state.selectedTargets = [];
  },
  showRoleReveal(state) {
    console.log('[night/showRoleReveal] current step:', state.step);
    if (state.step === 'idle') {
      state.step = 'role_reveal';
    }
  },
  /** Queue a night prompt; only wake if already sleeping */
  queuePrompt(state, data) {
    console.log('[night/queuePrompt] current step:', state.step, 'data:', data.roleId, data.actionType);
    if (state.step === 'sleeping') {
      console.log('[night/queuePrompt] already sleeping → waking immediately');
      state.isMyTurn = true;
      state.step = 'woken';
      state.roleId = data.roleId || '';
      state.roleName = data.roleName || '';
      state.abilityText = data.abilityText || '';
      state.actionType = data.actionType || 'passive';
      state.targets = data.targets || [];
      state.selectedTargets = [];
      state.result = '';
      state.pendingPrompt = null;
    } else {
      console.log('[night/queuePrompt] step is', state.step, '→ storing as pendingPrompt');
      state.pendingPrompt = data;
    }
  },
  /** Consume queued prompt when entering sleeping */
  consumePendingPrompt(state) {
    console.log('[night/consumePendingPrompt] pendingPrompt:', !!state.pendingPrompt, 'step:', state.step);
    if (state.pendingPrompt && state.step === 'sleeping') {
      const p = state.pendingPrompt;
      state.pendingPrompt = null;
      state.isMyTurn = true;
      state.step = 'woken';
      state.roleId = p.roleId || '';
      state.roleName = p.roleName || '';
      state.abilityText = p.abilityText || '';
      state.actionType = p.actionType || 'passive';
      state.targets = p.targets || [];
      state.selectedTargets = [];
      state.result = '';
      console.log('[night/consumePendingPrompt] woke up as', p.roleId);
    }
  },
  reset(state) {
    state.isMyTurn = false;
    state.step = 'idle';
    state.roleId = '';
    state.roleName = '';
    state.abilityText = '';
    state.actionType = '';
    state.targets = [];
    state.selectedTargets = [];
    state.result = '';
    state.pendingPrompt = null;
    state.nightInfoDetail = null;
    state.teamRecognition = null;
    // nightInfoHistory / grimoireHistory 不在此清除，需跨夜保留；仅在游戏结束/重新开始时单独清除
    state.progress = { current: 0, total: 0 };
  }
};

const getters = {
  canSubmit: state => {
    if (state.actionType === 'passive' || state.actionType === 'info') return true;
    if (state.actionType === 'select_one') return state.selectedTargets.length === 1;
    if (state.actionType === 'select_two') return state.selectedTargets.length === 2;
    return false;
  },
  isActive: state => state.step !== 'idle' && state.step !== 'done',
  isDemon: (state) => {
    if (!state.teamRecognition) return false;
    return state.teamRecognition.demonId !== '' && state.teamRecognition.bluffs && state.teamRecognition.bluffs.length > 0;
  }
};

export default {
  namespaced: true,
  state,
  mutations,
  getters
};
