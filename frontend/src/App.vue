<template>
  <div>
    <AppHeader />
    <div class="container mx-auto pt-2">
      <RouterView />
    </div>
    <AppFooter />
  </div>
</template>

<script lang="ts">
import { RouterView } from 'vue-router';
import AppFooter from './components/AppFooter.vue';
import AppHeader from './components/AppHeader.vue';
import API from '@/services/API';

import { mapStores } from 'pinia';
import { useUserStore, useSettingsStore } from '@/store';

import { getWebsocketURI } from '@/services/util';
import { Websocket, type WebsocketMessageHandler } from '@/services/ws';

export default {
  name: 'App',
  components: {
    RouterView,
    AppHeader,
    AppFooter,
  },
  data() {
    return {
      socket: new Websocket(getWebsocketURI() + '/events', this.onWebsocketMessage as WebsocketMessageHandler),
    };
  },
  created() {},
  mounted() {
    this.fetchData();
    this.socket.connect();
  },
  unmounted() {
    if (this.socket) {
      this.socket.close();
    }
  },
  methods: {
    onWebsocketMessage(event: MessageEvent): void {
      const data = JSON.parse(event.data);
      this.$EventBus.emit(data.type, data.data);
    },
    fetchData() {
      // GET /users/me
      API.get('/users/me')
        .then((res) => {
          this.userStore.id = res.data.id;
          this.userStore.username = res.data.username;
          this.userStore.created_at = res.data.created_at;
          this.userStore.loggedIn = true;
        })
        .catch((_err) => {
          this.userStore.loggedIn = false;
        });
    },
  },
  computed: {
    ...mapStores(useUserStore, useSettingsStore),
  },
};
</script>

<style scoped></style>
