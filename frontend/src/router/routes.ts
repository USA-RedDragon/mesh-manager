export default [
  {
    path: '/',
    name: 'Main',
    component: () => import('../views/MainPage.vue'),
  },
  {
    path: '/login',
    name: 'Login',
    component: () => import('../views/LoginPage.vue'),
  },
  {
    path: '/tunnels',
    name: 'Tunnels',
    component: () => import('../views/TunnelsPage.vue'),
  },
  {
    path: '/meshmap',
    name: 'MeshMap',
    component: () => import('../views/MeshMapPage.vue'),
  },
  {
    path: '/olsr',
    name: 'OLSR',
    component: () => import('../views/OLSRPage.vue'),
    alias: '/nodes',
  },
  {
    path: '/babel',
    name: 'Babel',
    component: () => import('../views/BabelPage.vue'),
  },
  {
    path: '/admin/users',
    name: 'AdminUsers',
    component: () => import('../views/admin/UsersPage.vue'),
  },
  {
    path: '/admin/users/register',
    name: 'AdminUserRegister',
    component: () => import('../views/admin/RegisterPage.vue'),
  },
  {
    path: '/admin/tunnels',
    name: 'AdminTunnels',
    component: () => import('../views/admin/TunnelsPage.vue'),
  },
  {
    path: '/admin/tunnels/create/wireguard',
    name: 'AdminTunnelsWireguardCreate',
    component: () => import('../views/admin/tunnels/WireguardCreatePage.vue'),
  },
]
