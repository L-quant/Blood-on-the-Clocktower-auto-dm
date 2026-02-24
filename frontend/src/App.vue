<template>
  <div
    id="app"
    :class="{
      night: isNight,
      'no-animations': !settings.animationsEnabled
    }"
  >
    <!-- Night backdrop -->
    <div class="backdrop" :class="{ visible: isNight }"></div>

    <!-- Top bar -->
    <TopBar @toggle-settings="toggleSettings" />

    <!-- Screen router -->
    <main class="app-content">
      <transition name="screen-fade" mode="out-in">
        <HomeScreen
          v-if="screen === 'home'"
          key="home"
        />
        <LobbyScreen
          v-else-if="screen === 'lobby'"
          key="lobby"
        />
        <GameScreen
          v-else-if="screen === 'game'"
          key="game"
        />
        <GameEndScreen
          v-else-if="screen === 'end'"
          key="end"
        />
      </transition>
    </main>

    <!-- Bottom navigation (game only) -->
    <BottomNav />

    <!-- Game overlays -->
    <NightOverlay />
    <VoteOverlay />
    <PhaseTransition />

    <!-- Global overlays -->
    <ConfirmDialog />
    <SettingsPanel v-if="showSettings" @close="showSettings = false" />

    <!-- Version -->
    <span id="version" v-if="screen === 'home'">v{{ version }}</span>
  </div>
</template>

<script>
import { mapState, mapGetters } from "vuex";
const { version } = require("../package.json");
import TopBar from "./components/TopBar";
import BottomNav from "./components/BottomNav";
import HomeScreen from "./components/HomeScreen";
import LobbyScreen from "./components/LobbyScreen";
import ConfirmDialog from "./components/ConfirmDialog";
import NightOverlay from "./components/NightOverlay";
import VoteOverlay from "./components/VoteOverlay";
import PhaseTransition from "./components/PhaseTransition";
import soundService from "./services/SoundService";

// Lazy-loaded screens
const GameScreen = () => import("./components/GameScreen");
const GameEndScreen = () => import("./components/GameEndScreen");
const SettingsPanel = () => import("./components/SettingsPanel");

export default {
  components: {
    TopBar,
    BottomNav,
    HomeScreen,
    LobbyScreen,
    GameScreen,
    GameEndScreen,
    ConfirmDialog,
    NightOverlay,
    VoteOverlay,
    PhaseTransition,
    SettingsPanel
  },
  data() {
    return {
      version,
      showSettings: false
    };
  },
  computed: {
    ...mapState("ui", {
      screen: "screen",
      settings: "settings"
    }),
    ...mapGetters("game", ["isNight"])
  },
  methods: {
    toggleSettings() {
      this.showSettings = !this.showSettings;
    },
    detectMobile() {
      this.$store.commit("ui/setIsMobile", window.innerWidth <= 768);
    },
    handleHash() {
      const hash = window.location.hash.substr(1);
      if (hash) {
        this.$store.dispatch("joinRoom", hash).catch(() => {});
      }
    }
  },
  created() {
    // Pre-auth so token is ready when user clicks Create/Join
    this.$store.dispatch("ensureAuth").then(() => {
      // Handle hash-based room join after auth is ready
      this.handleHash();
    }).catch(() => {});
    this.detectMobile();
    window.addEventListener("resize", this.detectMobile);
  },
  mounted() {
    soundService.preload();
    soundService.setMuted(!this.settings.soundEnabled);
  },
  beforeDestroy() {
    window.removeEventListener("resize", this.detectMobile);
  }
};
</script>

<style lang="scss">
@import "vars";

@font-face {
  font-family: "Papyrus";
  src: url("assets/fonts/papyrus.eot");
  src: url("assets/fonts/papyrus.eot?#iefix") format("embedded-opentype"),
    url("assets/fonts/papyrus.woff2") format("woff2"),
    url("assets/fonts/papyrus.woff") format("woff"),
    url("assets/fonts/papyrus.ttf") format("truetype"),
    url("assets/fonts/papyrus.svg#PapyrusW01") format("svg");
}

@font-face {
  font-family: PiratesBay;
  src: url("assets/fonts/piratesbay.ttf");
  font-display: swap;
}

html,
body {
  font-size: 1.2em;
  line-height: 1.4;
  background: url("assets/background.jpg") center center;
  background-size: cover;
  color: white;
  height: 100%;
  font-family: "Roboto Condensed", sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  padding: 0;
  margin: 0;
  overflow: hidden;
}

@import "media";

* {
  box-sizing: border-box;
  position: relative;
}

a {
  color: $townsfolk;
  &:hover {
    color: $demon;
  }
}

h1, h2, h3, h4, h5 {
  margin: 0;
  text-align: center;
  font-family: PiratesBay, sans-serif;
  letter-spacing: 1px;
  font-weight: normal;
}

ul {
  list-style-type: none;
  margin: 0;
  padding: 0;
}

#app {
  height: 100%;
  display: flex;
  flex-direction: column;

  &.no-animations *,
  &.no-animations *:after,
  &.no-animations *:before {
    transition: none !important;
    animation: none !important;
  }
}

.app-content {
  flex: 1;
  overflow-y: auto;
  overflow-x: hidden;
  padding-top: 44px; // TopBar height
  -webkit-overflow-scrolling: touch;
}

// Show bottom nav padding when in game
.app-content:has(+ .bottom-nav) {
  padding-bottom: 56px;
}

#version {
  position: fixed;
  right: 10px;
  bottom: 10px;
  font-size: 60%;
  opacity: 0.5;
  z-index: 1;
}

// Night backdrop
#app > .backdrop {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  pointer-events: none;
  background: linear-gradient(
    180deg,
    rgba(0, 0, 0, 1) 0%,
    rgba(1, 22, 46, 1) 50%,
    rgba(0, 39, 70, 1) 100%
  );
  opacity: 0;
  transition: opacity 1s ease-in-out;
  z-index: 0;

  &.visible {
    opacity: 0.5;
  }

  &:after {
    content: " ";
    display: block;
    width: 100%;
    padding-right: 2000px;
    height: 100%;
    background: url("assets/clouds.png") repeat;
    background-size: 2000px auto;
    animation: move-clouds 120s linear infinite;
    opacity: 0.3;
  }
}

@keyframes move-clouds {
  from { transform: translate3d(-2000px, 0, 0); }
  to { transform: translate3d(0, 0, 0); }
}

// Screen transitions
.screen-fade-enter-active,
.screen-fade-leave-active {
  transition: opacity 300ms, transform 300ms;
}
.screen-fade-enter {
  opacity: 0;
  transform: translateY(10px);
}
.screen-fade-leave-to {
  opacity: 0;
  transform: translateY(-10px);
}

// Buttons (global)
.button-group {
  display: flex;
  align-items: center;
  justify-content: center;
  .button {
    margin: 5px 0;
    border-radius: 0;
    &:first-child {
      border-top-left-radius: 15px;
      border-bottom-left-radius: 15px;
    }
    &:last-child {
      border-top-right-radius: 15px;
      border-bottom-right-radius: 15px;
    }
  }
}
.button {
  padding: 0;
  border: solid 0.125em transparent;
  border-radius: 15px;
  box-shadow: inset 0 1px 1px #9c9c9c, 0 0 10px #000;
  background: radial-gradient(
        at 0 -15%,
        rgba(#fff, 0.07) 70%,
        rgba(#fff, 0) 71%
      )
      0 0/ 80% 90% no-repeat content-box,
    linear-gradient(#4e4e4e, #040404) content-box,
    linear-gradient(#292929, #010101) border-box;
  color: white;
  font-weight: bold;
  text-shadow: 1px 1px rgba(0, 0, 0, 0.5);
  line-height: 170%;
  margin: 5px auto;
  cursor: pointer;
  transition: all 200ms;
  white-space: nowrap;
  &:hover {
    color: red;
  }
  &:active {
    transform: scale(0.95);
  }
  &.disabled {
    color: gray;
    cursor: default;
    opacity: 0.75;
    &:active {
      transform: none;
    }
  }
  &:before,
  &:after {
    content: " ";
    display: inline-block;
    width: 10px;
    height: 10px;
  }
  &.townsfolk {
    background: radial-gradient(
          at 0 -15%,
          rgba(255, 255, 255, 0.07) 70%,
          rgba(255, 255, 255, 0) 71%
        )
        0 0/80% 90% no-repeat content-box,
      linear-gradient(#0031ad, rgba(5, 0, 0, 0.22)) content-box,
      linear-gradient(#292929, #001142) border-box;
    box-shadow: inset 0 1px 1px #002c9c, 0 0 10px #000;
    &:hover:not(.disabled) {
      color: #008cf7;
    }
  }
  &.demon {
    background: radial-gradient(
          at 0 -15%,
          rgba(255, 255, 255, 0.07) 70%,
          rgba(255, 255, 255, 0) 71%
        )
        0 0/80% 90% no-repeat content-box,
      linear-gradient(#ad0000, rgba(5, 0, 0, 0.22)) content-box,
      linear-gradient(#292929, #420000) border-box;
    box-shadow: inset 0 1px 1px #9c0000, 0 0 10px #000;
  }
}

// Skeleton loading
.skeleton {
  background: linear-gradient(
    90deg,
    rgba(255, 255, 255, 0.05) 25%,
    rgba(255, 255, 255, 0.1) 50%,
    rgba(255, 255, 255, 0.05) 75%
  );
  background-size: 200% 100%;
  animation: skeleton-shimmer 1.5s infinite;
  border-radius: 4px;
}

@keyframes skeleton-shimmer {
  0% { background-position: 200% 0; }
  100% { background-position: -200% 0; }
}
</style>
