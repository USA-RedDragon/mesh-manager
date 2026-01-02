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

      <details
        v-if="userStore.loggedIn"
        ref="adminMenu"
        class="admin-menu"
        :open="adminMenuOpen"
        @toggle="handleAdminToggle"
      >
        <summary :class="{ adminNavLink: true, 'router-link-active': $route.path.startsWith('/admin') }">Admin</summary>
        <div class="admin-menu-items">
          <RouterLink to="/admin/tunnels" @click="handleAdminNavigate">Tunnels</RouterLink>
          <RouterLink to="/admin/users" @click="handleAdminNavigate">Admin Users</RouterLink>
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
      adminMenuOpen: false,
    };
  },
  mounted() {
    this.adminMenuOpen = this.$route.path.startsWith('/admin');
    document.addEventListener('click', this.handleOutsideClick);
  },
  beforeUnmount() {
    document.removeEventListener('click', this.handleOutsideClick);
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
    handleAdminToggle(event) {
      this.adminMenuOpen = event.target?.open ?? false;
    },
    handleAdminNavigate() {
      this.adminMenuOpen = false;
    },
    handleOutsideClick(event) {
      const menu = this.$refs.adminMenu;
      if (!menu) return;
      if (menu.contains(event.target)) return;
      this.adminMenuOpen = false;
    },
  },
  computed: {
    ...mapStores(useUserStore),
  },
};
</script>

<style scoped>
header {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  padding: 0.75rem 1rem;
  margin: auto;
  width: 100%;
  background-color: var(--secondary);
}

header h1,
header nav {
  display: inline;
}

header h1 {
  margin: 0;
  font-size: 1.1rem;
  flex: 0 0 auto;
}

.button {
  flex: 0 0 auto;
  text-align: right;
}

header nav .router-link-active,
.adminNavLink.router-link-active {
  color: var(--secondary-foreground) !important;
  font-weight: bolder;
}

nav {
  display: flex;
  align-items: center;
  justify-content: center;
  flex: 1 1 300px;
  flex-wrap: wrap;
  gap: 0.75rem;
  text-align: center;
}

nav a {
  padding: 0.35rem 0.5rem;
  border-radius: 0.375rem;
}

.admin-menu {
  display: inline-flex;
  position: relative;
  margin-left: 0.25rem;
}

.admin-menu summary {
  list-style: none;
  cursor: pointer;
  padding: 0.35rem 0.5rem;
  border-radius: 0.375rem;
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

@media (max-width: 640px) {
  header {
    flex-direction: column;
    align-items: flex-start;
  }

  nav {
    width: 100%;
    justify-content: flex-start;
    gap: 0.5rem;
  }

  .button {
    width: 100%;
    text-align: left;
  }
}
</style>
