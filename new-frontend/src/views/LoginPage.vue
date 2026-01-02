<template>
  <div class="max-w-md mx-auto">
    <Card>
      <CardHeader>
        <CardTitle>Login</CardTitle>
      </CardHeader>
      <CardContent>
        <form class="space-y-4" @submit.prevent="handleLogin(!v$.$invalid)">
          <div class="space-y-1">
            <label for="username" class="text-sm font-medium">Username</label>
            <input
              id="username"
              v-model="v$.username.$model"
              type="text"
              class="w-full rounded-md border px-3 py-2 text-sm"
              :aria-invalid="v$.username.$invalid && submitted"
            />
            <p v-if="v$.username.$error && submitted" class="text-xs text-red-600">Username is required.</p>
          </div>
          <div class="space-y-1">
            <label for="password" class="text-sm font-medium">Password</label>
            <input
              id="password"
              v-model="v$.password.$model"
              type="password"
              class="w-full rounded-md border px-3 py-2 text-sm"
              :aria-invalid="v$.password.$invalid && submitted"
            />
            <p v-if="v$.password.$error && submitted" class="text-xs text-red-600">Password is required.</p>
          </div>
          <UiButton type="submit" class="w-full">Login</UiButton>
        </form>
      </CardContent>
    </Card>
  </div>
</template>

<script lang="ts">
import API from '@/services/API';

import { useVuelidate } from '@vuelidate/core';
import { required } from '@vuelidate/validators';

import { mapStores } from 'pinia';
import { useUserStore } from '@/store';

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
      username: '',
      password: '',
      submitted: false,
    };
  },
  validations() {
    return {
      username: {
        required,
      },
      password: {
        required,
      },
    };
  },
  methods: {
    handleLogin(isFormValid: boolean) {
      this.submitted = true;
      if (!isFormValid) {
        return;
      }

      API.post('/auth/login', {
        username: this.username.trim(),
        password: this.password.trim(),
      })
        .then(() => {
          API.get('/users/me').then((res) => {
            this.userStore.id = res.data.id;
            this.userStore.username = res.data.username;
            this.userStore.created_at = res.data.created_at;
            this.userStore.loggedIn = true;
            this.$router.push('/');
          });
        })
        .catch((err) => {
          console.error(err);
          const message = err?.response?.data?.error || 'Error logging in';
          alert(message);
        });
    },
  },
  computed: {
    ...mapStores(useUserStore),
  },
};
</script>

<style scoped></style>
