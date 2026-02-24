<template>
  <div class="home-screen">
    <div class="home-screen__hero">
      <img
        class="home-screen__logo"
        src="/static/apple-icon.png"
        alt="Blood on the Clocktower"
      />
      <h1 class="home-screen__title">{{ $t('app.title') }}</h1>
      <p class="home-screen__subtitle">{{ $t('app.subtitle') }}</p>
    </div>

    <div class="home-screen__actions">
      <button
        class="button townsfolk home-screen__btn"
        @click="createRoom"
        :class="{ disabled: loading }"
      >
        <font-awesome-icon icon="plus-circle" />
        {{ $t('home.createRoom') }}
      </button>

      <button
        class="button home-screen__btn"
        @click="showJoinSheet = true"
      >
        <font-awesome-icon icon="link" />
        {{ $t('home.joinRoom') }}
      </button>
    </div>

    <!-- Join Room Bottom Sheet -->
    <JoinRoomSheet
      v-if="showJoinSheet"
      @close="showJoinSheet = false"
      @join="joinRoom"
    />

    <!-- Error toast -->
    <transition name="fade">
      <div class="home-screen__error" v-if="error">
        {{ error }}
      </div>
    </transition>
  </div>
</template>

<script>
import JoinRoomSheet from "./JoinRoomSheet";

export default {
  name: "HomeScreen",
  components: { JoinRoomSheet },
  data() {
    return {
      showJoinSheet: false,
      loading: false,
      error: ""
    };
  },
  methods: {
    async createRoom() {
      if (this.loading) return;
      this.loading = true;
      this.error = "";
      try {
        await this.$store.dispatch("createRoom");
      } catch (e) {
        this.error = e.message || this.$t('errors.failedToCreateRoom');
        setTimeout(() => { this.error = ""; }, 3000);
      } finally {
        this.loading = false;
      }
    },
    async joinRoom(roomCode) {
      this.showJoinSheet = false;
      this.loading = true;
      this.error = "";
      try {
        await this.$store.dispatch("joinRoom", roomCode);
      } catch (e) {
        this.error = e.message || this.$t('errors.failedToJoinRoom');
        setTimeout(() => { this.error = ""; }, 3000);
      } finally {
        this.loading = false;
      }
    }
  }
};
</script>

<style lang="scss" scoped>
@import "../vars";

.home-screen {
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 20px;
  text-align: center;

  &__hero {
    margin-bottom: 48px;
  }

  &__logo {
    width: 120px;
    height: 120px;
    border-radius: 24px;
    margin-bottom: 20px;
    box-shadow: 0 4px 20px rgba(0, 0, 0, 0.5);
  }

  &__title {
    font-size: 1.6rem;
    margin-bottom: 8px;
  }

  &__subtitle {
    font-size: 0.9rem;
    opacity: 0.6;
    font-family: Papyrus, serif;
  }

  &__actions {
    display: flex;
    flex-direction: column;
    gap: 12px;
    width: 100%;
    max-width: 280px;
  }

  &__btn {
    width: 100%;
    font-size: 0.95rem;
    padding: 4px 0;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 8px;

    svg {
      font-size: 0.85rem;
    }
  }

  &__error {
    position: fixed;
    bottom: 80px;
    left: 50%;
    transform: translateX(-50%);
    background: rgba($demon, 0.9);
    padding: 10px 20px;
    border-radius: 8px;
    font-size: 0.85rem;
    white-space: nowrap;
  }
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 200ms;
}
.fade-enter,
.fade-leave-to {
  opacity: 0;
}
</style>
