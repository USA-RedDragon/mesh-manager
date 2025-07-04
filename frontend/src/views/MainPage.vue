<template>
  <div>
    <span style="display: flex; justify-content: space-evenly">
      <Card style="width: 48%;">
        <template #title>Daemon Status</template>
        <template #content>
          <h3 style="font-weight: bold;">Babel Daemon</h3>
          <p>
            <PVBadge v-if="babelRunning" value="✔️" severity="success"></PVBadge>
            <PVBadge v-else value="✖️" severity="danger"></PVBadge>
            {{ babelRunning ? 'Running':'Stopped' }}
          </p>
          <br />
          <h3 style="font-weight: bold;">MeshLink</h3>
          <p>
            <PVBadge v-if="meshLinkRunning" value="✔️" severity="success"></PVBadge>
            <PVBadge v-else value="✖️" severity="danger"></PVBadge>
            {{ meshLinkRunning ? 'Running':'Stopped' }}
          </p>
          <br />
          <h3 style="font-weight: bold;">OLSR Daemon</h3>
          <p>
            <PVBadge v-if="olsrdRunning" value="✔️" severity="success"></PVBadge>
            <PVBadge v-else value="✖️" severity="danger"></PVBadge>
            {{ olsrdRunning ? 'Running':'Stopped' }}
          </p>
          <br />
          <h3 style="font-weight: bold;">DNSMasq</h3>
          <p>
            <PVBadge v-if="dnsRunning" value="✔️" severity="success"></PVBadge>
            <PVBadge v-else value="✖️" severity="danger"></PVBadge>
            {{ dnsRunning ? 'Running':'Stopped' }}
          </p>
        </template>
      </Card>
      <Card style="width: 48%;">
        <template #title>Network Statistics</template>
        <template #content>
          <h3 style="font-weight: bold;">Tunnels Connected</h3>
          <p>{{ wireguardTunnelsConnected }}/{{ totalWireguardTunnels }}</p>
          <br />
          <h3 style="font-weight: bold;">Current Bandwidth</h3>
          <p><span style="font-weight: bold;">RX:</span> {{ prettyBytes(stats.total_rx_bytes_per_sec) }}/s</p>
          <p><span style="font-weight: bold;">TX:</span> {{ prettyBytes(stats.total_tx_bytes_per_sec) }}/s</p>
          <br />
          <h3 style="font-weight: bold;">Total Traffic Since Restart</h3>
          <p><span style="font-weight: bold;">RX:</span> {{ prettyBytes(stats.total_rx_mb) }}</p>
          <p><span style="font-weight: bold;">TX:</span> {{ prettyBytes(stats.total_tx_mb) }}</p>
        </template>
      </Card>
    </span>
  </div>
</template>

<script>
import Badge from 'primevue/badge';
import Card from 'primevue/card';

import prettyBytes from 'pretty-bytes';

import API from '@/services/API';

export default {
  components: {
    Card,
    PVBadge: Badge,
  },
  created() {
    this.fetchData();
  },
  mounted() {
    this.$EventBus.on('tunnel_disconnection', this.tunnelDisconnected);
    this.$EventBus.on('tunnel_connection', this.tunnelConnected);
    this.$EventBus.on('total_traffic', this.totalTraffic);
    this.$EventBus.on('total_bandwidth', this.totalBandwidth);
  },
  unmounted() {
    this.$EventBus.off('tunnel_disconnection', this.tunnelDisconnected);
    this.$EventBus.off('tunnel_connection', this.tunnelConnected);
    this.$EventBus.off('total_traffic', this.totalTraffic);
    this.$EventBus.off('total_bandwidth', this.totalBandwidth);
  },
  data: function() {
    return {
      babelRunning: true,
      olsrdRunning: true,
      dnsRunning: true,
      meshLinkRunning: true,
      wireguardTunnelsConnected: 0,
      totalWireguardTunnels: 0,
      stats: {},
    };
  },
  methods: {
    tunnelDisconnected(_) {
      this.wireguardTunnelsConnected--;
    },
    tunnelConnected(_) {
      this.wireguardTunnelsConnected++;
    },
    totalBandwidth(event) {
      if ('TX' in event) {
        this.stats.total_rx_bytes_per_sec = event.RX;
        this.stats.total_tx_bytes_per_sec = event.TX;
      }
    },
    totalTraffic(event) {
      // Truncate to 2 decimal places
      let rx = Math.round(event.RX * 100) / 100;
      // Convert to bytes
      rx = rx * 1024 * 1024;
      this.stats.total_rx_mb = rx;

      // Truncate to 2 decimal places
      let tx = Math.round(event.TX * 100) / 100;
      // Convert to bytes
      tx = tx * 1024 * 1024;
      this.stats.total_tx_mb = tx;
    },
    prettyBytes(bytes) {
      if (!bytes) {
        return '0 B';
      }
      return prettyBytes(bytes);
    },
    fetchData() {
      API.get('/olsr/running').then((res) => {
        this.olsrdRunning = res.data.running;
      });
      API.get('/babel/running').then((res) => {
        this.babelRunning = res.data.running;
      });
      API.get('/dns/running').then((res) => {
        this.dnsRunning = res.data.running;
      });
      API.get('/meshlink/running').then((res) => {
        this.meshLinkRunning = res.data.running;
      });
      API.get('/tunnels/wireguard/count/connected').then((res) => {
        this.wireguardTunnelsConnected = res.data.count;
      });
      API.get('/tunnels/wireguard/count').then((res) => {
        this.totalWireguardTunnels = res.data.count;
      });
      API.get('/stats').then((res) => {
        if (typeof res.data == 'string') {
          return;
        }
        if (!('stats' in res.data)) {
          return;
        }
        if (res.data.stats.total_rx_mb != 0) {
          // Truncate to 2 decimal places
          res.data.stats.total_rx_mb = Math.round(res.data.stats.total_rx_mb * 100) / 100;
          // Convert to bytes
          res.data.stats.total_rx_mb = res.data.stats.total_rx_mb * 1024 * 1024;
        }
        if (res.data.stats.total_tx_mb != 0) {
          // Truncate to 2 decimal places
          res.data.stats.total_tx_mb = Math.round(res.data.stats.total_tx_mb * 100) / 100;
          // Convert to bytes
          res.data.stats.total_tx_mb = res.data.stats.total_tx_mb * 1024 * 1024;
        }
        this.stats = res.data.stats;
      });
    },
  },
  computed: {},
};
</script>

<style scoped></style>
