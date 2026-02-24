<template>
  <transition name="fade">
    <div class="confirm-overlay" v-if="isOpen" @click.self="cancel">
      <div class="confirm-dialog">
        <p class="confirm-dialog__message">{{ message }}</p>
        <div class="confirm-dialog__actions">
          <button class="button" @click="cancel">
            {{ $t('confirm.no') }}
          </button>
          <button class="button demon" @click="confirm">
            {{ $t('confirm.yes') }}
          </button>
        </div>
      </div>
    </div>
  </transition>
</template>

<script>
import { mapState } from "vuex";

export default {
  name: "ConfirmDialog",
  computed: {
    ...mapState("ui", {
      confirmModal: state => state.modals.confirm
    }),
    isOpen() {
      return this.confirmModal && this.confirmModal.open;
    },
    message() {
      return this.confirmModal ? this.confirmModal.message : "";
    }
  },
  methods: {
    confirm() {
      if (this.confirmModal && this.confirmModal.onConfirm) {
        this.confirmModal.onConfirm();
      }
      this.$store.commit("ui/closeModal", "confirm");
    },
    cancel() {
      this.$store.commit("ui/closeModal", "confirm");
    }
  }
};
</script>

<style lang="scss" scoped>
.confirm-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.7);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 200;
}

.confirm-dialog {
  background: rgba(20, 20, 30, 0.95);
  border: 2px solid rgba(255, 255, 255, 0.15);
  border-radius: 12px;
  padding: 24px;
  max-width: 320px;
  width: 90%;
  text-align: center;

  &__message {
    margin: 0 0 20px;
    font-size: 0.95rem;
    line-height: 1.5;
  }

  &__actions {
    display: flex;
    gap: 12px;
    justify-content: center;

    .button {
      min-width: 80px;
      padding: 0 16px;
      font-size: 0.85rem;
    }
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
