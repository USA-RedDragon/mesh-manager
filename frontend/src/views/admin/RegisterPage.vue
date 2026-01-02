<template>
  <div class="max-w-lg mx-auto">
    <Card>
      <CardHeader>
        <CardTitle>Register</CardTitle>
      </CardHeader>
      <CardContent>
        <form class="space-y-4" @submit.prevent="handleRegister(!v$.$invalid)">
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

          <div class="space-y-1">
            <label for="confirmPassword" class="text-sm font-medium">Confirm Password</label>
            <input
              id="confirmPassword"
              v-model="v$.confirmPassword.$model"
              type="password"
              class="w-full rounded-md border px-3 py-2 text-sm"
              :aria-invalid="v$.confirmPassword.$invalid && submitted"
            />
            <p v-if="v$.confirmPassword.$error && submitted" class="text-xs text-red-600">Passwords must match.</p>
          </div>

          <UiButton type="submit" class="w-full">Register</UiButton>
        </form>
      </CardContent>
    </Card>
  </div>
</template>

<script lang="ts">
import API from '@/services/API';

import { useVuelidate } from '@vuelidate/core';
import { required, sameAs } from '@vuelidate/validators';

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
      confirmPassword: '',
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
      confirmPassword: {
        required,
        sameAs: sameAs(this.password),
      },
    };
  },
  methods: {
    handleRegister(isFormValid: boolean) {
      this.submitted = true;
      if (!isFormValid) {
        return;
      }

      if (this.confirmPassword != this.password) {
        alert('Passwords do not match');
        return;
      }
      API.post('/users', {
        username: this.username.trim(),
        password: this.password.trim(),
      })
        .then((res) => {
          alert(res.data.message || 'User created');
          this.$router.push('/admin/users');
        })
        .catch((err) => {
          console.error(err);
          const message = err?.response?.data?.error || 'An unknown error occurred';
          alert(message);
        });
    },
  },
};
</script>

<style scoped></style>
