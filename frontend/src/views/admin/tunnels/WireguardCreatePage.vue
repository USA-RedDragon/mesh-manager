<template>
  <div class="max-w-2xl mx-auto">
    <Card>
      <CardHeader>
        <CardTitle>Create Wireguard Tunnel</CardTitle>
      </CardHeader>
      <CardContent>
        <form class="space-y-6" @submit.prevent="handleSubmit(!v$.$invalid)">
          <section class="space-y-3">
            <h3 class="text-sm font-semibold">Connection Type</h3>
            <div class="space-y-2">
              <label class="flex items-center gap-2 text-sm">
                <input
                  v-model="tunnelType"
                  type="radio"
                  name="tunnelType"
                  value="server"
                  class="h-4 w-4"
                />
                <span>Server - Provide a tunnel to another node</span>
              </label>
              <label class="flex items-center gap-2 text-sm">
                <input
                  v-model="tunnelType"
                  type="radio"
                  name="tunnelType"
                  value="client"
                  class="h-4 w-4"
                />
                <span>Client - Connect to another node's tunnel</span>
              </label>
            </div>
          </section>

          <section class="space-y-4">
            <h3 class="text-sm font-semibold">Connection Settings</h3>

            <div v-if="tunnelType === 'server'" class="space-y-1">
              <label for="hostname" class="text-sm font-medium">Hostname</label>
              <input
                id="hostname"
                v-model="v$.hostname.$model"
                type="text"
                class="w-full rounded-md border px-3 py-2 text-sm"
                :aria-invalid="v$.hostname.$invalid && submitted"
              />
              <p v-if="v$.hostname.$error && submitted" class="text-xs text-red-600">
                Hostname is required (3-63 chars, alphanumeric or -).
              </p>
            </div>

            <div v-if="tunnelType === 'client'" class="space-y-4">
              <div class="space-y-1">
                <label for="server" class="text-sm font-medium">Server Address</label>
                <input
                  id="server"
                  v-model="v$.server.$model"
                  type="text"
                  class="w-full rounded-md border px-3 py-2 text-sm"
                  :aria-invalid="v$.server.$invalid && submitted"
                />
                <p v-if="v$.server.$error && submitted" class="text-xs text-red-600">Server is required.</p>
              </div>

              <div class="space-y-1">
                <label for="network" class="text-sm font-medium">Network (hostname:port)</label>
                <input
                  id="network"
                  v-model="v$.network.$model"
                  type="text"
                  class="w-full rounded-md border px-3 py-2 text-sm"
                  :aria-invalid="v$.network.$invalid && submitted"
                />
                <p v-if="v$.network.$error && submitted" class="text-xs text-red-600">Network is required.</p>
              </div>

              <div class="space-y-1">
                <label for="password" class="text-sm font-medium">Key</label>
                <input
                  id="password"
                  v-model="v$.password.$model"
                  type="password"
                  class="w-full rounded-md border px-3 py-2 text-sm"
                  :aria-invalid="v$.password.$invalid && submitted"
                />
                <p v-if="v$.password.$error && submitted" class="text-xs text-red-600">Key is required.</p>
              </div>
            </div>
          </section>

          <UiButton type="submit" class="w-full">Submit</UiButton>
        </form>
      </CardContent>
    </Card>
  </div>
</template>

<script lang="ts">
import API from '@/services/API';

import { useVuelidate } from '@vuelidate/core';
import { requiredIf, minLength, maxLength } from '@vuelidate/validators';

import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import { Button as UiButton } from '@/components/ui/button';

export default {
  components: {
    Card,
    CardContent,
    CardHeader,
    CardTitle,
    UiButton,
  },
  setup: () => ({ v$: useVuelidate() }),
  data: function() {
    return {
      hostname: '',
      server: '',
      network: '',
      password: '',
      tunnelType: 'server',
      submitted: false,
    };
  },
  validations() {
    return {
      hostname: {
        required: requiredIf(this.tunnelType == 'server'),
        minLength: minLength(3),
        maxLength: maxLength(63),
      },
      password: {
        required: requiredIf(this.tunnelType == 'client'),
        minLength: minLength(44 * 3),
      },
      server: {
        required: requiredIf(this.tunnelType == 'client'),
        minLength: minLength(3),
      },
      network: {
        required: requiredIf(this.tunnelType == 'client'),
      },
    };
  },
  methods: {
    handleSubmit(isFormValid: boolean) {
      this.submitted = true;
      if (!isFormValid && this.v$.$errors.length > 0) {
        return;
      }

      if (this.tunnelType == 'client') {
        const networkParts = this.network.split(':');
        if (networkParts.length !== 2) {
          alert('Network must be in the format hostname:port');
          return;
        }

        const host = networkParts[0]!;
        const portStr = networkParts[1]!;

        if (host.length > 253) {
          alert('Network must be less than 254 characters');
          return;
        }

        if (!/^[A-Za-z0-9-.]+$/.test(host)) {
          alert("Network hostname must be alphanumeric, '.', or '-'");
          return;
        }

        const port = parseInt(portStr, 10);
        if (isNaN(port)) {
          alert('Network port must be a number');
          return;
        }
        if (port < 1 || port > 65535) {
          alert('Network port must be between 1 and 65535');
          return;
        }

        API.post('/tunnels', {
          hostname: this.server.trim(),
          password: this.password.trim(),
          ip: this.network.trim(),
          client: true,
          wireguard: true,
        })
          .then((res) => {
            alert(res.data.message || 'Tunnel created');
            this.$router.push('/admin/tunnels');
          })
          .catch((err) => {
            console.error(err);
            const message = err?.response?.data?.error || 'An unknown error occurred';
            alert(message);
          });
      } else if (this.tunnelType == 'server') {
        if (this.hostname.length > 63) {
          alert('Hostname must be less than 64 characters');
          return;
        }

        if (!/^[A-Za-z0-9-]+$/.test(this.hostname)) {
          alert('Hostname must be alphanumeric or -');
          return;
        }

        if (this.hostname.length < 3) {
          alert('Hostname must be at least 3 characters');
          return;
        }

        API.post('/tunnels', {
          hostname: this.hostname.trim(),
          client: false,
          wireguard: true,
        })
          .then((res) => {
            alert(res.data.message || 'Tunnel created');
            this.$router.push('/admin/tunnels');
          })
          .catch((err) => {
            console.error(err);
            const message = err?.response?.data?.error || 'An unknown error occurred';
            alert(message);
          });
      }
    },
  },
};
</script>

<style scoped></style>
