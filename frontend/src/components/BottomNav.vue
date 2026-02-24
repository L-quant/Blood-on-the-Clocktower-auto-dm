<template>
  <nav class="bottom-nav" v-if="visible" role="tablist" :aria-label="$t('nav.square')">
    <button
      v-for="tab in tabs"
      :key="tab.id"
      class="bottom-nav__tab"
      :class="{ active: activeTab === tab.id }"
      role="tab"
      :aria-selected="activeTab === tab.id"
      :aria-label="$t(tab.label)"
      @click="setTab(tab.id)"
    >
      <font-awesome-icon :icon="tab.icon" class="bottom-nav__icon" />
      <span class="bottom-nav__label">{{ $t(tab.label) }}</span>
      <span
        class="bottom-nav__badge"
        v-if="tab.badge && tab.badge > 0"
      >{{ tab.badge > 99 ? '99+' : tab.badge }}</span>
    </button>
  </nav>
</template>

<script>
import { mapState, mapGetters } from "vuex";

export default {
  name: "BottomNav",
  computed: {
    ...mapState("ui", ["activeTab"]),
    ...mapState("ui", { uiScreen: "screen" }),
    ...mapGetters("game", ["isPlaying"]),
    ...mapGetters("chat", ["totalUnread"]),
    visible() {
      return this.uiScreen === "game";
    },
    tabs() {
      return [
        { id: "square", icon: "users", label: "nav.square", badge: 0 },
        { id: "chat", icon: "clipboard", label: "nav.chat", badge: this.totalUnread },
        { id: "timeline", icon: "book-open", label: "nav.timeline", badge: 0 },
        { id: "me", icon: "user", label: "nav.me", badge: 0 }
      ];
    }
  },
  methods: {
    setTab(tab) {
      this.$store.commit("ui/setActiveTab", tab);
    }
  }
};
</script>

<style lang="scss" scoped>
@import "../vars";

.bottom-nav {
  position: fixed;
  bottom: 0;
  left: 0;
  right: 0;
  height: 56px;
  display: flex;
  align-items: center;
  justify-content: space-around;
  background: rgba(0, 0, 0, 0.9);
  border-top: 1px solid rgba(255, 255, 255, 0.1);
  z-index: 100;
  padding-bottom: env(safe-area-inset-bottom, 0);
  backdrop-filter: blur(10px);
  -webkit-backdrop-filter: blur(10px);

  &__tab {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 2px;
    padding: 6px 0;
    background: none;
    border: none;
    color: rgba(255, 255, 255, 0.4);
    cursor: pointer;
    transition: all 200ms;

    &.active {
      color: white;
    }

    &:active {
      transform: scale(0.9);
    }
  }

  &__icon {
    font-size: 1.1rem;
  }

  &__label {
    font-size: 0.6rem;
    letter-spacing: 0.5px;
  }

  &__badge {
    position: absolute;
    top: 4px;
    right: calc(50% - 16px);
    min-width: 16px;
    height: 16px;
    border-radius: 8px;
    background: $demon;
    color: white;
    font-size: 0.55rem;
    font-weight: bold;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 0 4px;
  }
}
</style>
