<template>
  <header>
    <h1>
      <RouterLink to="/">Node Console</RouterLink>
    </h1>
    <nav>
      <RouterLink to="/">Home</RouterLink>
      <RouterLink to="/olsr">OLSR</RouterLink>
      <RouterLink to="/babel">Babel</RouterLink>
      <RouterLink to="/tunnels">Tunnels</RouterLink>
      <RouterLink v-if="hasMeshmap" to="/meshmap">Mesh Map</RouterLink>

      <details v-if="userStore.loggedIn" class="admin-menu" :open="$route.path.startsWith('/admin')">
        <summary :class="{ adminNavLink: true, 'router-link-active': $route.path.startsWith('/admin') }">Admin</summary>
        <div class="admin-menu-items">
          <RouterLink to="/admin/tunnels">Tunnels</RouterLink>
          <RouterLink to="/admin/users">Admin Users</RouterLink>
        </div>
      </details>
      <RouterLink v-if="!userStore.loggedIn" to="/login"
        >Login</RouterLink
      >
      <a v-else href="#" @click="logout()">Logout</a>
    </nav>
    <ColorModeButton class="button" />
  </header>
</template>

<script lang="ts">
import API from '@/services/API';
import ColorModeButton from '@/components/ColorModeButton.vue';

import { mapStores } from 'pinia';
import { useUserStore } from '@/store';

export default {
  components: {
    ColorModeButton,
  },
  data: function() {
    return {
      hasMeshmap: true,
    };
  },
  methods: {
    logout() {
      API.get('/auth/logout')
        .then(() => {
          this.userStore.loggedIn = false;
          this.$router.push('/login');
        })
        .catch((err) => {
          console.error(err);
        });
    },
  },
  computed: {
    ...mapStores(useUserStore),
  },
};
</script>

<style scoped>
header {
  height: 3em;
  padding: 0.5em;
  margin: auto;
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  background-color: var(--secondary);
}

header h1,
header nav,
.button {
  font-size: 1rem;
  width: 33%;
}

.button {
  text-align: right;
}

header h1,
header nav {
  display: inline;
}

header nav .router-link-active,
.adminNavLink.router-link-active {
  color: var(--secondary-foreground) !important;
  font-weight: bolder;
}

nav {
  text-align: center;
}

nav a {
  padding: 0 1rem;
  border-left: 1px solid #444;
}

nav a:first-of-type {
  border: 0;
}

.admin-menu {
  display: inline-block;
  position: relative;
  margin-left: 1rem;
}

.admin-menu summary {
  list-style: none;
  cursor: pointer;
}

.admin-menu summary::-webkit-details-marker {
  display: none;
}

.admin-menu-items {
  position: absolute;
  right: 0;
  background: var(--card);
  border: 1px solid var(--border);
  border-radius: 0.5rem;
  padding: 0.5rem 0.75rem;
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  z-index: 10;
  min-width: 10rem;
}

.admin-menu[open] summary {
  color: var(--secondary-foreground);
  font-weight: 700;
}
</style>
