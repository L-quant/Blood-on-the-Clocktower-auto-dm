/**
 * SoundService - game audio management
 * Wraps existing SoundManager with a cleaner API
 */

class SoundService {
  constructor() {
    this.sounds = {};
    this.muted = false;
    this.volume = 0.5;
    this._preloaded = false;
  }

  preload() {
    if (this._preloaded) return;

    const soundFiles = {
      countdown: require("../assets/sounds/countdown.mp3")
    };

    for (const [name, src] of Object.entries(soundFiles)) {
      try {
        const audio = new Audio(src);
        audio.preload = "auto";
        audio.volume = this.volume;
        this.sounds[name] = audio;
      } catch (e) {
        // Sound loading failed
      }
    }

    this._preloaded = true;
  }

  play(name) {
    if (this.muted) return;
    const audio = this.sounds[name];
    if (!audio) return;
    const clone = audio.cloneNode();
    clone.volume = this.volume;
    clone.play().catch(() => {});
  }

  setMuted(muted) {
    this.muted = muted;
  }

  setVolume(vol) {
    this.volume = Math.max(0, Math.min(1, vol));
    for (const audio of Object.values(this.sounds)) {
      audio.volume = this.volume;
    }
  }
}

export const soundService = new SoundService();
export default soundService;
