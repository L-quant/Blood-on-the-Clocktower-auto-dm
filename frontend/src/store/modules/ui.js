// Vuex 模块：UI 状态（屏幕路由、标签页、弹窗、设置）
//
// [OUT] store/index.js（模块注册）
// [POS] UI 控制中心，替代 Vue Router 管理画面切换

const state = () => ({
  activeTab: 'square', // 'square' | 'chat' | 'timeline' | 'me'
  screen: 'home', // 'home' | 'lobby' | 'game' | 'end'
  isMobile: false,
  modals: {
    roleDetail: { open: false, roleId: '' },
    playerAction: { open: false, seatIndex: -1 },
    reference: false,
    nightOrder: false,
    settings: false,
    joinRoom: false,
    confirm: { open: false, message: '', onConfirm: null }
  },
  settings: {
    soundEnabled: true,
    animationsEnabled: true,
    locale: 'zh',
    reducedMotion: false
  },
  notes: '' // player general notes
});

const mutations = {
  setActiveTab(state, tab) {
    state.activeTab = tab;
  },
  setScreen(state, screen) {
    state.screen = screen;
  },
  setIsMobile(state, isMobile) {
    state.isMobile = isMobile;
  },
  openModal(state, { modal, data }) {
    if (typeof state.modals[modal] === 'boolean') {
      state.modals[modal] = true;
    } else if (state.modals[modal]) {
      state.modals[modal] = { open: true, ...data };
    }
  },
  closeModal(state, modal) {
    if (typeof state.modals[modal] === 'boolean') {
      state.modals[modal] = false;
    } else if (state.modals[modal]) {
      state.modals[modal] = { ...state.modals[modal], open: false };
    }
  },
  updateSetting(state, { key, value }) {
    if (key in state.settings) {
      state.settings[key] = value;
    }
  },
  setNotes(state, notes) {
    state.notes = notes;
  },
  reset(state) {
    state.activeTab = 'square';
    state.notes = '';
    Object.keys(state.modals).forEach(key => {
      if (typeof state.modals[key] === 'boolean') {
        state.modals[key] = false;
      } else {
        state.modals[key] = { ...state.modals[key], open: false };
      }
    });
  }
};

export default {
  namespaced: true,
  state,
  mutations
};
