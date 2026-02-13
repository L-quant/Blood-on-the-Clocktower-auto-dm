// ============================================
// 主应用入口 - 路由 + 初始化
// ============================================

import { store } from './store.js';
import { wsConnect } from './api.js';
export { toast } from './utils.js';
import { renderAuth, mountAuth } from './views/auth.js';
import { renderHome, mountHome } from './views/home.js';
import { renderRoom, mountRoom, unmountRoom } from './views/room.js';
import { renderGame, mountGame, unmountGame } from './views/game.js';

// Toast re-exported from utils.js

// ============================================
// 路由
// ============================================
let currentView = null;
let currentUnmount = null;

const views = {
  auth:  { render: renderAuth,  mount: mountAuth,  unmount: null },
  home:  { render: renderHome,  mount: mountHome,  unmount: null },
  room:  { render: renderRoom,  mount: mountRoom,  unmount: unmountRoom },
  game:  { render: renderGame,  mount: mountGame,  unmount: unmountGame },
};

export function navigate(viewName) {
  // Unmount current view
  if (currentUnmount) {
    currentUnmount();
    currentUnmount = null;
  }

  const view = views[viewName];
  if (!view) {
    console.error('Unknown view:', viewName);
    return;
  }

  // Update store
  store.set('currentView', viewName);
  currentView = viewName;

  // Update URL hash
  window.location.hash = viewName;

  // Render
  const app = document.getElementById('app');
  app.innerHTML = view.render();

  // Mount (bind events)
  if (view.mount) {
    view.mount();
  }
  currentUnmount = view.unmount || null;
}

// ============================================
// 初始化
// ============================================
function init() {
  // Check if already logged in
  if (store.isLoggedIn()) {
    // Connect WebSocket
    wsConnect();

    // Determine initial view
    const hash = window.location.hash.slice(1);
    const roomId = store.get('roomId');

    if (hash === 'game' && roomId) {
      navigate('game');
    } else if (hash === 'room' && roomId) {
      navigate('room');
    } else {
      navigate('home');
    }
  } else {
    navigate('auth');
  }

  // Handle browser back/forward
  window.addEventListener('hashchange', () => {
    const hash = window.location.hash.slice(1);
    if (hash && hash !== currentView && views[hash]) {
      // Auth guard
      if (hash !== 'auth' && !store.isLoggedIn()) {
        navigate('auth');
        return;
      }
      navigate(hash);
    }
  });
}

// Start the app
document.addEventListener('DOMContentLoaded', init);
