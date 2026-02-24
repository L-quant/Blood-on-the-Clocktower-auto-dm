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
  closeAllModals(state) {
    Object.keys(state.modals).forEach(key => {
      if (typeof state.modals[key] === 'boolean') {
        state.modals[key] = false;
      } else {
        state.modals[key] = { ...state.modals[key], open: false };
      }
    });
  },
  toggleModal(state, modal) {
    if (typeof state.modals[modal] === 'boolean') {
      state.modals[modal] = !state.modals[modal];
    } else if (state.modals[modal]) {
      state.modals[modal].open = !state.modals[modal].open;
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

const getters = {
  isModalOpen: state => modal => {
    if (typeof state.modals[modal] === 'boolean') {
      return state.modals[modal];
    }
    return state.modals[modal] ? state.modals[modal].open : false;
  },
  anyModalOpen: state => {
    return Object.values(state.modals).some(m => {
      return typeof m === 'boolean' ? m : m.open;
    });
  }
};

export default {
  namespaced: true,
  state,
  mutations,
  getters
};
