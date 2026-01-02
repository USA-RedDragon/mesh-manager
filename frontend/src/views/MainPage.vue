<template>
  <div>
    <div class="info">
      <Card>
        <CardHeader>
          <CardTitle>Node Info</CardTitle>
        </CardHeader>
        <CardContent>
          <h3 style="font-weight: bold;">Hostname</h3>
          <p>{{ hostname }}</p>
          <br />
          <h3 style="font-weight: bold;">IP</h3>
          <p>{{ nodeIP }}</p>
          <br />
          <h3 style="font-weight: bold;">Version</h3>
          <p>{{ version }}</p>
          <span v-if="!!gridsquare">
            <br />
            <h3 style="font-weight: bold;">Location</h3>
            <p>{{ gridsquare }}</p>
          </span>
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <CardTitle>Daemon Status</CardTitle>
        </CardHeader>
        <CardContent>
          <table>
            <tbody>
              <tr>
                <td style="width: 70%;">
                  <p style="font-weight: bold;">babeld</p>
                </td>
                <td>
                  <StatusBadge :status="babelRunning" />
                </td>
              </tr>
              <tr>
                <td>
                  <p style="font-weight: bold;">meshlink</p>
                </td>
                <td>
                  <StatusBadge :status="meshLinkRunning" />
                </td>
              </tr>
              <tr>
                <td>
                  <p style="font-weight: bold;">olsrd</p>
                </td>
                <td>
                  <StatusBadge :status="olsrdRunning" />
                </td>
              </tr>
              <tr>
                <td>
                  <p style="font-weight: bold;">dnsmasq</p>
                </td>
                <td>
                  <StatusBadge :status="dnsRunning" />
                </td>
              </tr>
            </tbody>
          </table>
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <CardTitle>Tunnels</CardTitle>
        </CardHeader>
        <CardContent>
          <h3 style="font-weight: bold;">Total</h3>
          <p>{{ wireguardTunnelsConnected }}/{{ totalWireguardTunnels }} connected</p>
          <br />
          <h3 style="font-weight: bold;">Client</h3>
          <p>{{ wireguardClientTunnelsConnected }}/{{ totalClientWireguardTunnels }} connected</p>
          <br />
          <h3 style="font-weight: bold;">Server</h3>
          <p>{{ wireguardServerTunnelsConnected }}/{{ totalServerWireguardTunnels }} connected</p>
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <CardTitle>Network Stats</CardTitle>
        </CardHeader>
        <CardContent>
          <h3 style="font-weight: bold;">Current Bandwidth</h3>
          <p><span style="font-weight: bold;">RX:</span> {{ prettyBytes(stats.total_rx_bytes_per_sec) }}/s</p>
          <p><span style="font-weight: bold;">TX:</span> {{ prettyBytes(stats.total_tx_bytes_per_sec) }}/s</p>
          <br />
          <h3 style="font-weight: bold;">Total Traffic Since Restart</h3>
          <p><span style="font-weight: bold;">RX:</span> {{ prettyBytes(stats.total_rx_mb) }}</p>
          <p><span style="font-weight: bold;">TX:</span> {{ prettyBytes(stats.total_tx_mb) }}</p>
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <CardTitle>OLSR Mesh Stats</CardTitle>
        </CardHeader>
        <CardContent>
          <h3 style="font-weight: bold;">Nodes</h3>
          <p>{{ olsrNodes }}</p>
          <br />
          <h3 style="font-weight: bold;">Devices</h3>
          <p>{{ olsrDevices }}</p>
          <br />
          <h3 style="font-weight: bold;">Services</h3>
          <p>{{ olsrServices }}</p>
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <CardTitle>Babel Mesh Stats</CardTitle>
        </CardHeader>
        <CardContent>
          <h3 style="font-weight: bold;">Nodes</h3>
          <p>{{ babelNodes }}</p>
          <br />
          <h3 style="font-weight: bold;">Devices</h3>
          <p>{{ babelDevices }}</p>
          <br />
          <h3 style="font-weight: bold;">Services</h3>
          <p>{{ babelServices }}</p>
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <CardTitle>Load Average</CardTitle>
        </CardHeader>
        <CardContent>
          <h3 style="font-weight: bold;">1 minute</h3>
          <p>{{ loadavg.one_min }}%</p>
          <br />
          <h3 style="font-weight: bold;">5 minutes</h3>
          <p>{{ loadavg.five_min }}%</p>
          <br />
          <h3 style="font-weight: bold;">15 minutes</h3>
          <p>{{ loadavg.fifteen_min }}%</p>
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <CardTitle>Uptime</CardTitle>
        </CardHeader>
        <CardContent>
          <p>{{ uptime }}</p>
        </CardContent>
      </Card>
    </div>
  </div>
</template>

<script lang="ts">
import StatusBadge from '@/components/StatusBadge.vue';
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';

import prettyBytes from 'pretty-bytes';

import API from '@/services/API';
import type {
  TotalBandwidthEvent,
  TotalTrafficEvent,
  TunnelConnectionEvent,
  TunnelDisconnectionEvent,
} from '@/services/EventBus';

export default {
  components: {
    Card,
    CardContent,
    CardHeader,
    CardTitle,
    StatusBadge,
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
      wireguardClientTunnelsConnected: 0,
      totalClientWireguardTunnels: 0,
      wireguardServerTunnelsConnected: 0,
      totalServerWireguardTunnels: 0,
      hostname: '',
      olsrNodes: 0,
      olsrDevices: 0,
      olsrServices: 0,
      babelNodes: 0,
      babelDevices: 0,
      babelServices: 0,
      loadavg: {
        one_min: 0,
        five_min: 0,
        fifteen_min: 0,
      },
      version: '',
      nodeIP: '',
      uptime: '',
      gridsquare: '',
      stats: {
        total_tx_bytes_per_sec: 0,
        total_rx_bytes_per_sec: 0,
        total_tx_mb: 0,
        total_rx_mb: 0,
      },
    };
  },
  methods: {
    tunnelDisconnected(event: TunnelDisconnectionEvent) {
      this.wireguardTunnelsConnected--;
      if (event.client) {
        this.wireguardClientTunnelsConnected--;
      } else {
        this.wireguardServerTunnelsConnected--;
      }
      if (this.wireguardTunnelsConnected < 0) {
        this.wireguardTunnelsConnected = 0;
      }
    },
    tunnelConnected(event: TunnelConnectionEvent) {
      this.wireguardTunnelsConnected++;
      if (event.client) {
        this.wireguardClientTunnelsConnected++;
      } else {
        this.wireguardServerTunnelsConnected++;
      }
    },
    totalBandwidth(event: TotalBandwidthEvent) {
      if ('TX' in event) {
        this.stats.total_rx_bytes_per_sec = event.RX;
        this.stats.total_tx_bytes_per_sec = event.TX;
      }
    },
    totalTraffic(event: TotalTrafficEvent) {
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
    prettyBytes(bytes: number) {
      if (!bytes) {
        return '0 B';
      }
      return prettyBytes(bytes);
    },
    fetchData() {
      API.get('/version').then((response) => {
        const longVersion = response.data;
        this.version = longVersion.split(' ')[0];
      });
      API.get(`/olsr/hosts/count`).then((res) => {
        this.olsrNodes = res.data.nodes;
        this.olsrDevices = res.data.total;
        this.olsrServices = res.data.services;
      });
      API.get(`/babel/hosts/count`).then((res) => {
        this.babelNodes = res.data.nodes;
        this.babelDevices = res.data.total;
        this.babelServices = res.data.services;
      });
      API.get('/node-ip').then((response) => {
        this.nodeIP = response.data.nodeIP;
      });
      API.get('/gridsquare').then((response) => {
        this.gridsquare = response.data.gridsquare;
      });
      API.get('/loadavg').then((res) => {
        this.loadavg.one_min = res.data.loadavg.one_min.toFixed(0);
        this.loadavg.five_min = res.data.loadavg.five_min.toFixed(0);
        this.loadavg.fifteen_min = res.data.loadavg.fifteen_min.toFixed(0);
      });
      API.get('/uptime').then((res) => {
        this.uptime = res.data.uptime;
      });
      API.get('/hostname').then((res) => {
        this.hostname = res.data.hostname;
      });
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
      API.get('/tunnels/wireguard/client/count/connected').then((res) => {
        this.wireguardClientTunnelsConnected = res.data.count;
      });
      API.get('/tunnels/wireguard/client/count').then((res) => {
        this.totalClientWireguardTunnels = res.data.count;
      });
      API.get('/tunnels/wireguard/server/count/connected').then((res) => {
        this.wireguardServerTunnelsConnected = res.data.count;
      });
      API.get('/tunnels/wireguard/server/count').then((res) => {
        this.totalServerWireguardTunnels = res.data.count;
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

<style scoped>
.info {
  -webkit-column-count: 4;
  -moz-column-count: 4;
  column-count: 4;
}

.info > div {
  break-inside: avoid;
}

.info > div:not(:first-child) {
  margin-top: 1em;
}

@media (max-width: 2100px) {
  .info {
    -moz-column-count: 4;
    -webkit-column-count: 4;
    column-count: 4;
  }
}
@media (max-width: 1200px) {
  .info {
    -moz-column-count: 3;
    -webkit-column-count: 3;
    column-count: 3;
  }
}
@media (max-width: 600px) {
  .info {
    -moz-column-count: 2;
    -webkit-column-count: 2;
    column-count: 2;
  }
}
</style>
