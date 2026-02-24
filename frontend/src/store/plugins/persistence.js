/**
 * Persistence plugin - saves selected state to localStorage
 * Handles: annotations (per room), settings, notes
 */

const STORAGE_KEYS = {
  SETTINGS: 'botc_settings',
  NOTES: 'botc_notes',
  ANNOTATIONS_PREFIX: 'botc_annotations_'
};

export default store => {
  // Load settings on init
  try {
    const saved = localStorage.getItem(STORAGE_KEYS.SETTINGS);
    if (saved) {
      const settings = JSON.parse(saved);
      Object.keys(settings).forEach(key => {
        store.commit('ui/updateSetting', { key, value: settings[key] });
      });
    }
  } catch (e) {
    // ignore
  }

  // Load notes on init
  try {
    const notes = localStorage.getItem(STORAGE_KEYS.NOTES);
    if (notes) {
      store.commit('ui/setNotes', notes);
    }
  } catch (e) {
    // ignore
  }

  // Subscribe to mutations to persist state
  store.subscribe(({ type, payload }) => {
    switch (type) {
      case 'ui/updateSetting':
        try {
          localStorage.setItem(
            STORAGE_KEYS.SETTINGS,
            JSON.stringify(store.state.ui.settings)
          );
        } catch (e) {
          // ignore
        }
        break;

      case 'ui/setNotes':
        try {
          localStorage.setItem(STORAGE_KEYS.NOTES, payload || '');
        } catch (e) {
          // ignore
        }
        break;

      case 'annotations/setAnnotation':
      case 'annotations/updateNote':
      case 'annotations/setGuessedRole':
      case 'annotations/clearAnnotation':
      case 'annotations/clearAll': {
        const roomId = store.state.roomId;
        if (roomId) {
          try {
            localStorage.setItem(
              STORAGE_KEYS.ANNOTATIONS_PREFIX + roomId,
              JSON.stringify(store.state.annotations.playerAnnotations)
            );
          } catch (e) {
            // ignore
          }
        }
        break;
      }

      case 'setRoomId': {
        // Load annotations for new room
        const roomId = payload;
        if (roomId) {
          try {
            const saved = localStorage.getItem(STORAGE_KEYS.ANNOTATIONS_PREFIX + roomId);
            if (saved) {
              store.commit('annotations/loadAnnotations', JSON.parse(saved));
            } else {
              store.commit('annotations/clearAll');
            }
          } catch (e) {
            store.commit('annotations/clearAll');
          }
        }
        break;
      }
    }
  });
};
