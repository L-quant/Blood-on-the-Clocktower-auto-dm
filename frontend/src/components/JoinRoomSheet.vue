<template>
  <div class="sheet-overlay" @click.self="$emit('close')">
    <transition name="slide-up">
      <div class="join-sheet" v-if="visible">
        <div class="join-sheet__handle"></div>
        <h3 class="join-sheet__title">{{ $t('home.enterRoomCode') }}</h3>
        <div class="join-sheet__input-wrapper">
          <input
            ref="input"
            class="join-sheet__input"
            :class="{ error: hasError }"
            v-model="code"
            @input="onInput"
            @keydown.enter="submit"
            :placeholder="$t('home.enterRoomCode')"
            maxlength="36"
            autocomplete="off"
            spellcheck="false"
          />
          <p class="join-sheet__error" v-if="hasError">
            {{ errorMessage }}
          </p>
        </div>
        <button
          class="button townsfolk join-sheet__submit"
          :class="{ disabled: !isValid }"
          @click="submit"
        >
          {{ $t('home.joinRoom') }}
        </button>
      </div>
    </transition>
  </div>
</template>

<script>
export default {
  name: "JoinRoomSheet",
  data() {
    return {
      code: "",
      hasError: false,
      errorMessage: "",
      visible: false
    };
  },
  computed: {
    isValid() {
      return this.code.trim().length >= 4 && !this.hasError;
    }
  },
  methods: {
    onInput() {
      this.code = this.code.replace(/[^a-zA-Z0-9-]/g, "");
      this.hasError = false;
      this.errorMessage = "";
    },
    submit() {
      if (!this.isValid) return;
      this.$emit("join", this.code);
    }
  },
  mounted() {
    // Animate in
    this.$nextTick(() => {
      this.visible = true;
      this.$nextTick(() => {
        if (this.$refs.input) {
          this.$refs.input.focus();
        }
      });
    });
  }
};
</script>

<style lang="scss" scoped>
@import "../vars";

.sheet-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: flex-end;
  justify-content: center;
  z-index: 150;
}

.join-sheet {
  width: 100%;
  max-width: 420px;
  background: rgba(20, 20, 30, 0.95);
  border-top-left-radius: 20px;
  border-top-right-radius: 20px;
  padding: 16px 24px 40px;
  border-top: 2px solid rgba(255, 255, 255, 0.15);

  &__handle {
    width: 40px;
    height: 4px;
    background: rgba(255, 255, 255, 0.2);
    border-radius: 2px;
    margin: 0 auto 20px;
  }

  &__title {
    text-align: center;
    font-size: 1rem;
    margin-bottom: 20px;
  }

  &__input-wrapper {
    margin-bottom: 20px;
  }

  &__input {
    width: 100%;
    background: rgba(255, 255, 255, 0.08);
    border: 2px solid rgba(255, 255, 255, 0.2);
    border-radius: 12px;
    padding: 14px 16px;
    color: white;
    font-size: 0.85rem;
    font-family: monospace;
    text-align: center;
    letter-spacing: 1px;
    outline: none;
    transition: border-color 200ms;

    &:focus {
      border-color: $townsfolk;
    }

    &.error {
      border-color: $demon;
    }

    &::placeholder {
      color: rgba(255, 255, 255, 0.2);
      letter-spacing: 1px;
    }
  }

  &__error {
    color: $demon;
    font-size: 0.75rem;
    text-align: center;
    margin-top: 8px;
  }

  &__submit {
    width: 100%;
    font-size: 1rem;
    padding: 4px 0;
  }
}

.slide-up-enter-active,
.slide-up-leave-active {
  transition: transform 300ms ease-out;
}
.slide-up-enter,
.slide-up-leave-to {
  transform: translateY(100%);
}
</style>
