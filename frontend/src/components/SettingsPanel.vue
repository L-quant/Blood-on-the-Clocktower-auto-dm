<template>
  <div class="settings-overlay" @click.self="$emit('close')">
    <div class="settings-panel">
      <div class="settings-panel__header">
        <h3>{{ $t('settings.title') }}</h3>
        <button class="settings-panel__close" @click="$emit('close')">
          <font-awesome-icon icon="times" />
        </button>
      </div>
      <div class="settings-panel__body">
        <label class="settings-panel__row">
          <span>{{ $t('settings.sound') }}</span>
          <input
            type="checkbox"
            :checked="settings.soundEnabled"
            @change="toggle('soundEnabled')"
          />
        </label>
        <label class="settings-panel__row">
          <span>{{ $t('settings.animations') }}</span>
          <input
            type="checkbox"
            :checked="settings.animationsEnabled"
            @change="toggle('animationsEnabled')"
          />
        </label>
        <label class="settings-panel__row">
          <span>{{ $t('settings.language') }}</span>
          <select
            :value="settings.locale"
            @change="setLocale($event.target.value)"
          >
            <option value="zh">中文</option>
            <option value="en">English</option>
          </select>
        </label>
      </div>
    </div>
  </div>
</template>

<script>
import { mapState } from "vuex";
import soundService from "../services/SoundService";

export default {
  name: "SettingsPanel",
  computed: {
    ...mapState("ui", ["settings"])
  },
  methods: {
    toggle(key) {
      const newVal = !this.settings[key];
      this.$store.commit("ui/updateSetting", { key, value: newVal });
      if (key === "soundEnabled") {
        soundService.setMuted(!newVal);
      }
    },
    setLocale(locale) {
      this.$store.commit("ui/updateSetting", { key: "locale", value: locale });
      this.$i18n.locale = locale;
    }
  }
};
</script>

<style lang="scss" scoped>
.settings-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.6);
  display: flex;
  align-items: flex-start;
  justify-content: center;
  padding-top: 60px;
  z-index: 150;
}

.settings-panel {
  background: rgba(20, 20, 30, 0.95);
  border: 2px solid rgba(255, 255, 255, 0.15);
  border-radius: 12px;
  width: 90%;
  max-width: 360px;
  overflow: hidden;

  &__header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 16px 20px;
    border-bottom: 1px solid rgba(255, 255, 255, 0.1);

    h3 {
      text-align: left;
      font-size: 1rem;
    }
  }

  &__close {
    background: none;
    border: none;
    color: white;
    font-size: 1.1rem;
    cursor: pointer;
    opacity: 0.7;
    padding: 4px;

    &:hover {
      opacity: 1;
    }
  }

  &__body {
    padding: 12px 20px;
  }

  &__row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 12px 0;
    border-bottom: 1px solid rgba(255, 255, 255, 0.05);
    cursor: pointer;

    &:last-child {
      border-bottom: none;
    }

    span {
      font-size: 0.9rem;
    }

    input[type="checkbox"] {
      width: 18px;
      height: 18px;
      cursor: pointer;
    }

    select {
      background: rgba(255, 255, 255, 0.1);
      color: white;
      border: 1px solid rgba(255, 255, 255, 0.2);
      border-radius: 4px;
      padding: 4px 8px;
      font-size: 0.85rem;
      cursor: pointer;
    }
  }
}
</style>
