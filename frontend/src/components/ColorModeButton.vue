<template>
  <span>
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger as-child>
          <ShadButton variant="ghost" size="icon" @click="switchColorMode">
            <template v-if="colorMode === 'dark'">
              <Sun class="w-8 h-8" />
            </template>
            <template v-else-if="colorMode === 'light'">
              <SunMoon class="w-8 h-8" />
            </template>
            <template v-else>
              <Moon class="w-8 h-8" />
            </template>
          </ShadButton>
        </TooltipTrigger>
        <TooltipContent>
          <template v-if="colorMode === 'dark'">
            Switch to light mode
          </template>
          <template v-else-if="colorMode === 'light'">
            Switch to auto mode
          </template>
          <template v-else>
            Switch to dark mode
          </template>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  </span>
</template>

<script lang="ts">
import { Button as ShadButton } from '@/components/ui/button';

import {
  SunMoon,
  Sun,
  Moon,
} from 'lucide-vue-next';

import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip';

import { useColorMode } from '@vueuse/core';

export default {
  components: {
    ShadButton,
    SunMoon,
    Moon,
    Sun,
    Tooltip,
    TooltipContent,
    TooltipProvider,
    TooltipTrigger,
  },
  data: function() {
    return {
      colorMode: useColorMode({
        emitAuto: true,
      }),
    };
  },
  mounted() {
  },
  unmounted() {
  },
  methods: {
    switchColorMode() {
      switch (this.colorMode) {
        case 'dark':
          this.colorMode = 'light';
          break;
        case 'light':
          this.colorMode = 'auto';
          break;
        case 'auto':
          this.colorMode = 'dark';
          break;
      }
    },
  },
};
</script>

<style scoped>
</style>
