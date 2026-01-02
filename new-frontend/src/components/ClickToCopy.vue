<template>
  <UiButton
    variant="ghost"
    size="sm"
    class="click-to-copy"
    :class="{ 'click-to-copy--copied': copied }"
    @click="copyToClipboard"
  >
    <slot>{{ text }}</slot>
  </UiButton>
</template>

<script lang="ts">
import { Button as UiButton } from '@/components/ui/button';

export default {
  props: {
    text: {
      type: String,
      default: 'Click to copy',
    },
    copy: {
      type: String,
      default: '',
    },
  },
  components: {
    UiButton,
  },
  data: function() {
    return {
      copied: false,
    };
  },
  methods: {
    copyToClipboard() {
      const setCopied = () => {
        this.copied = true;
        setTimeout(() => {
          this.copied = false;
        }, 1000);
      };

      if ('navigator' in window && 'clipboard' in window.navigator) {
        navigator.clipboard.writeText(this.copy).then(setCopied);
        return;
      }

      const el = document.createElement('textarea');
      el.value = this.copy;
      document.body.appendChild(el);
      el.select();
      document.execCommand('copy');
      document.body.removeChild(el);
      setCopied();
    },
  },
};
</script>

<style scoped>
.click-to-copy {
  cursor: pointer;
  user-select: none;
}

.click-to-copy--copied {
  color: green;
}
</style>
