<template>
  <div class="solo-mode" v-if="showSoloMode">
    <div class="solo-card">
      <h3>{{ $t('solo.title') }}</h3>
      <p>{{ $t('solo.description') }}</p>

      <div class="solo-config">
        <label>
          {{ $t('solo.botCount') }}
          <select v-model.number="botCount">
            <option :value="4">{{ $t('solo.bots4') }}</option>
            <option :value="5">{{ $t('solo.bots5') }}</option>
            <option :value="6" selected>{{ $t('solo.bots6') }}</option>
            <option :value="7">{{ $t('solo.bots7') }}</option>
            <option :value="9">{{ $t('solo.bots9') }}</option>
          </select>
        </label>
        <label>
          {{ $t('solo.personality') }}
          <select v-model="personality">
            <option value="random">{{ $t('solo.random') }}</option>
            <option value="aggressive">{{ $t('solo.aggressive') }}</option>
            <option value="cautious">{{ $t('solo.cautious') }}</option>
            <option value="smart">{{ $t('solo.smart') }}</option>
          </select>
        </label>
      </div>

      <div class="solo-actions">
        <button class="btn-solo" @click="startSoloGame" :disabled="loading">
          {{ loading ? $t('solo.addingBots') : $t('solo.startGame') }}
        </button>
        <button class="btn-cancel" @click="$emit('close')">{{ $t('solo.cancel') }}</button>
      </div>

      <p v-if="error" class="error">{{ error }}</p>
    </div>
  </div>
</template>

<script>
export default {
  name: "SoloMode",
  props: {
    showSoloMode: {
      type: Boolean,
      default: false
    },
    roomId: {
      type: String,
      default: ""
    }
  },
  data() {
    return {
      botCount: 6,
      personality: "random",
      loading: false,
      error: ""
    };
  },
  methods: {
    async startSoloGame() {
      if (!this.roomId) {
        this.error = this.$t("solo.noRoom");
        return;
      }

      this.loading = true;
      this.error = "";

      try {
        const apiUrl = process.env.VUE_APP_API_URL || "http://localhost:8080";
        const token = localStorage.getItem("auth_token") || "";

        const response = await fetch(`${apiUrl}/v1/rooms/${this.roomId}/bots`, {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            "Authorization": `Bearer ${token}`
          },
          body: JSON.stringify({
            count: this.botCount,
            personality: this.personality
          })
        });

        if (!response.ok) {
          const text = await response.text();
          throw new Error(text || "Failed to add bots");
        }

        const data = await response.json();
        this.$emit("bots-added", data);
        this.$emit("close");
      } catch (err) {
        this.error = err.message;
      } finally {
        this.loading = false;
      }
    }
  }
};
</script>

<style scoped lang="scss">
.solo-mode {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background: rgba(0, 0, 0, 0.7);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 2000;
}

.solo-card {
  background: #1a1a2e;
  border-radius: 12px;
  padding: 24px;
  width: 360px;
  max-width: 90vw;
  color: #eee;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.5);

  h3 {
    margin: 0 0 8px;
    font-size: 20px;
    color: #f1c40f;
  }

  p {
    margin: 0 0 16px;
    color: #aaa;
    font-size: 14px;
  }
}

.solo-config {
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin-bottom: 20px;

  label {
    display: flex;
    justify-content: space-between;
    align-items: center;
    font-size: 14px;

    select {
      background: rgba(255, 255, 255, 0.1);
      border: 1px solid rgba(255, 255, 255, 0.2);
      border-radius: 4px;
      color: #eee;
      padding: 4px 8px;
      font-size: 13px;
    }
  }
}

.solo-actions {
  display: flex;
  gap: 10px;

  .btn-solo {
    flex: 1;
    background: #2ecc71;
    color: white;
    border: none;
    border-radius: 6px;
    padding: 10px;
    font-size: 15px;
    cursor: pointer;
    font-weight: bold;

    svg {
      margin-right: 6px;
    }

    &:hover:not(:disabled) {
      background: #27ae60;
    }

    &:disabled {
      opacity: 0.6;
      cursor: not-allowed;
    }
  }

  .btn-cancel {
    background: transparent;
    border: 1px solid rgba(255, 255, 255, 0.3);
    color: #aaa;
    border-radius: 6px;
    padding: 10px 16px;
    cursor: pointer;
    font-size: 14px;

    &:hover {
      border-color: rgba(255, 255, 255, 0.5);
      color: #eee;
    }
  }
}

.error {
  color: #e74c3c !important;
  margin-top: 12px !important;
}
</style>
