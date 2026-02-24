/**
 * SoundManager - singleton audio manager.
 * Preloads game sounds and plays them on demand.
 * Respects grimoire.isMuted setting.
 */

class SoundManager {
  constructor() {
    this.sounds = {};
    this.muted = false;
    this.volume = 0.5;
    this._preloaded = false;
  }

  /**
   * Preload all game sounds. Call once on app init.
   */
  preload() {
    if (this._preloaded) return;

    const soundFiles = {
      // Reuse existing countdown sound
      countdown: require("../assets/sounds/countdown.mp3")
    };

    // Only load files that exist
    for (const [name, src] of Object.entries(soundFiles)) {
      try {
        const audio = new Audio(src);
        audio.preload = "auto";
        audio.volume = this.volume;
        this.sounds[name] = audio;
      } catch (e) {
        console.warn(`SoundManager: failed to load ${name}`, e);
      }
    }

    this._preloaded = true;
  }

  /**
   * Play a named sound.
   * @param {string} name - Sound name (e.g., 'bell', 'death')
   */
  play(name) {
    if (this.muted) return;
    const audio = this.sounds[name];
    if (!audio) return;
    // Clone to allow overlapping plays
    const clone = audio.cloneNode();
    clone.volume = this.volume;
    clone.play().catch(() => {
      // Autoplay may be blocked by browser
    });
  }

  /**
   * Set mute state.
   * @param {boolean} muted
   */
  setMuted(muted) {
    this.muted = muted;
  }

  /**
   * Set volume (0.0 - 1.0).
   * @param {number} vol
   */
  setVolume(vol) {
    this.volume = Math.max(0, Math.min(1, vol));
    for (const audio of Object.values(this.sounds)) {
      audio.volume = this.volume;
    }
  }
}

// Export singleton
const soundManager = new SoundManager();
export default soundManager;
