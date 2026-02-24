const state = () => ({
  isMyTurn: false,
  step: 'idle', // 'idle' | 'woken' | 'selecting' | 'waiting' | 'result' | 'done'
  roleId: '',
  roleName: '',
  abilityText: '',
  actionType: '', // 'select_one' | 'select_two' | 'passive' | 'info'
  targets: [], // available targets
  selectedTargets: [],
  result: '',
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
  setProgress(state, { current, total }) {
    state.progress.current = current;
    state.progress.total = total;
  },
  closePanel(state) {
    state.isMyTurn = false;
    state.step = 'idle';
    state.selectedTargets = [];
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
  isActive: state => state.step !== 'idle' && state.step !== 'done'
};

export default {
  namespaced: true,
  state,
  mutations,
  getters
};
