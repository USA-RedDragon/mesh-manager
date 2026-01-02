<script setup lang="ts">
import type { ColumnDef } from '@tanstack/vue-table'
import moment, { type Moment } from 'moment'
import prettyBytes from 'pretty-bytes'
import { computed, getCurrentInstance, h, onBeforeUnmount, onMounted, ref } from 'vue'
import { RouterLink } from 'vue-router'

import ClickToCopy from './ClickToCopy.vue'
import DataTable from './datatable/DataTable.vue'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import API from '@/services/API'
import type { TunnelConnectionEvent, TunnelDisconnectionEvent, TunnelStatsEvent } from '@/services/EventBus'

type ConnectionTime = string | Moment | 'Never'

interface Tunnel {
  id: number
  rx_bytes_per_sec: number
  tx_bytes_per_sec: number
  rx_bytes: number
  tx_bytes: number
  total_rx_mb: number
  total_tx_mb: number
  active: boolean
  connection_time: ConnectionTime
  editing: boolean
  enabled: boolean
  hostname: string
  ip: string
  wireguard: boolean
  wireguard_port?: number
  password?: string
  client?: boolean
  created_at: string
}

const props = defineProps<{
  admin?: boolean
}>()

const tunnels = ref<Tunnel[]>([])
const totalRecords = ref(0)
const pageSize = ref(10)
const loading = ref(false)
const search = ref('')

const instance = getCurrentInstance()
const bus = instance?.proxy?.$EventBus

function formatBytes(bytes: number) {
  if (!bytes) return '0 B'
  return prettyBytes(bytes)
}

function formatConnection(time: ConnectionTime) {
  if (!time || time === 'Never') return 'Never'
  const asMoment = typeof time === 'string' ? moment(time) : time
  if (!asMoment || !asMoment.isValid()) return 'Never'
  return asMoment.fromNow()
}

const columns = computed<ColumnDef<Tunnel>[]>(() => {
  const shared: ColumnDef<Tunnel>[] = [
    {
      accessorKey: 'enabled',
      header: () => h('span', 'Enabled'),
      cell: ({ row }) => {
        const value = row.original.enabled
        if (props.admin && row.original.editing) {
          return h('input', {
            type: 'checkbox',
            checked: value,
            onChange: (event: Event) => {
              const target = event.target as HTMLInputElement
              row.original.enabled = target.checked
            },
          })
        }
        return h(Badge, { variant: value ? 'default' : 'destructive' }, () => (value ? 'Yes' : 'No'))
      },
    },
    {
      accessorKey: 'active',
      header: () => h('span', 'Connected'),
      cell: ({ row }) => h('div', { class: 'flex items-center gap-2' }, [
        h(Badge, { variant: row.original.active ? 'default' : 'destructive' }, () => (row.original.active ? '✔' : '✖')),
        h('span', { class: 'text-sm text-muted-foreground' }, formatConnection(row.original.connection_time)),
      ]),
    },
    {
      accessorKey: 'hostname',
      header: () => h('span', 'Name'),
      cell: ({ row }) => {
        if (props.admin && row.original.editing) {
          return h('input', {
            class: 'w-full rounded-md border px-2 py-1 text-sm',
            value: row.original.hostname,
            onInput: (event: Event) => {
              const target = event.target as HTMLInputElement
              row.original.hostname = target.value
            },
          })
        }
        return h('span', row.original.hostname)
      },
    },
    {
      accessorKey: 'ip',
      header: () => h('span', 'IP'),
      cell: ({ row }) => {
        if (props.admin && row.original.editing) {
          return h('input', {
            class: 'w-full rounded-md border px-2 py-1 text-sm',
            value: row.original.ip,
            onInput: (event: Event) => {
              const target = event.target as HTMLInputElement
              row.original.ip = target.value
            },
          })
        }
        return h('span', row.original.ip)
      },
    },
    {
      accessorKey: 'wireguard_port',
      header: () => h('span', 'Wireguard Port'),
      cell: ({ row }) => (row.original.wireguard ? h('span', row.original.wireguard_port ?? '-') : h('span', '-')),
    },
  ]

  const adminOnly: ColumnDef<Tunnel>[] = [
    {
      accessorKey: 'password',
      header: () => h('span', 'Password'),
      cell: ({ row }) => {
        if (row.original.editing) {
          return h('input', {
            class: 'w-full rounded-md border px-2 py-1 text-sm',
            value: row.original.password,
            onInput: (event: Event) => {
              const target = event.target as HTMLInputElement
              row.original.password = target.value
            },
          })
        }
        if (row.original.client) {
          return h('span', 'Private')
        }
        return h(ClickToCopy, { copy: row.original.password || '' })
      },
    },
    {
      accessorKey: 'created_at',
      header: () => h('span', 'Created'),
      cell: ({ row }) => h('span', moment(row.original.created_at).fromNow()),
    },
    {
      id: 'actions',
      header: () => h('span', 'Actions'),
      cell: ({ row }) => {
        const editing = row.original.editing
        return h('div', { class: 'flex gap-2 justify-end' }, [
          editing
            ? h(Button, { size: 'sm', onClick: () => saveTunnel(row.original) }, () => 'Save')
            : h(Button, { size: 'sm', variant: 'secondary', onClick: () => startEdit(row.original) }, () => 'Edit'),
          editing
            ? h(Button, { size: 'sm', variant: 'ghost', onClick: () => cancelEdit(row.original) }, () => 'Cancel')
            : null,
          h(Button, { size: 'sm', variant: 'destructive', onClick: () => deleteTunnel(row.original) }, () => 'Delete'),
        ])
      },
    },
  ]

  const userOnly: ColumnDef<Tunnel>[] = [
    {
      accessorKey: 'rx_bytes_per_sec',
      header: () => h('span', 'Bandwidth'),
      cell: ({ row }) => h('div', { class: 'text-sm leading-tight' }, [
        h('p', [h('span', { class: 'font-semibold' }, 'RX: '), formatBytes(row.original.rx_bytes_per_sec), '/s']),
        h('p', [h('span', { class: 'font-semibold' }, 'TX: '), formatBytes(row.original.tx_bytes_per_sec), '/s']),
      ]),
    },
    {
      accessorKey: 'rx_bytes',
      header: () => h('span', 'Session Traffic'),
      cell: ({ row }) => h('div', { class: 'text-sm leading-tight' }, [
        h('p', [h('span', { class: 'font-semibold' }, 'RX: '), formatBytes(row.original.rx_bytes)]),
        h('p', [h('span', { class: 'font-semibold' }, 'TX: '), formatBytes(row.original.tx_bytes)]),
      ]),
    },
    {
      accessorKey: 'total_rx_mb',
      header: () => h('span', 'Total Traffic'),
      cell: ({ row }) => h('div', { class: 'text-sm leading-tight' }, [
        h('p', [h('span', { class: 'font-semibold' }, 'RX: '), formatBytes(row.original.total_rx_mb)]),
        h('p', [h('span', { class: 'font-semibold' }, 'TX: '), formatBytes(row.original.total_tx_mb)]),
      ]),
    },
  ]

  return props.admin ? [...shared, ...adminOnly] : [...shared, ...userOnly]
})

async function fetchData(page = 1, limit = pageSize.value) {
  loading.value = true
  try {
    const params = [`page=${page}`, `limit=${limit}`, `admin=${props.admin ?? false}`, 'type=wireguard']
    if (search.value) {
      params.push(`filter=${encodeURIComponent(search.value)}`)
    }
    const res = await API.get(`/tunnels?${params.join('&')}`)
    const normalized: Tunnel[] = (res.data.tunnels || []).map((tunnel: Tunnel) => {
      const connection = !tunnel.connection_time || tunnel.connection_time === '0001-01-01T00:00:00Z'
        ? 'Never'
        : tunnel.connection_time
      const totalRx = tunnel.total_rx_mb ? Math.round(tunnel.total_rx_mb * 100) / 100 * 1024 * 1024 : 0
      const totalTx = tunnel.total_tx_mb ? Math.round(tunnel.total_tx_mb * 100) / 100 * 1024 * 1024 : 0
      return {
        ...tunnel,
        editing: false,
        connection_time: connection,
        total_rx_mb: totalRx,
        total_tx_mb: totalTx,
      }
    })
    tunnels.value = normalized
    totalRecords.value = res.data.total
    pageSize.value = limit
  } catch (err) {
    console.error(err)
  } finally {
    loading.value = false
  }
}

function updateTunnelStats(event: TunnelStatsEvent) {
  const tunnel = tunnels.value.find((t) => t.id === event.id)
  if (!tunnel) return
  tunnel.rx_bytes_per_sec = event.rx_bytes_per_sec
  tunnel.tx_bytes_per_sec = event.tx_bytes_per_sec
  tunnel.rx_bytes = event.rx_bytes
  tunnel.tx_bytes = event.tx_bytes
  tunnel.total_rx_mb = event.total_rx_mb * 1024 * 1024
  tunnel.total_tx_mb = event.total_tx_mb * 1024 * 1024
}

function updateTunnelConnected(event: TunnelConnectionEvent) {
  const tunnel = tunnels.value.find((t) => t.id === event.id)
  if (!tunnel) return
  tunnel.active = true
  tunnel.connection_time = !event.connection_time || event.connection_time === '0001-01-01T00:00:00Z'
    ? 'Never'
    : moment(event.connection_time)
}

function updateTunnelDisconnected(event: TunnelDisconnectionEvent) {
  const tunnel = tunnels.value.find((t) => t.id === event.id)
  if (!tunnel) return
  tunnel.active = false
}

function startEdit(tunnel: Tunnel) {
  tunnel.editing = true
}

function cancelEdit(tunnel: Tunnel) {
  tunnel.editing = false
  fetchData()
}

async function saveTunnel(tunnel: Tunnel) {
  tunnel.editing = false
  try {
    await API.patch('/tunnels/', {
      id: tunnel.id,
      hostname: tunnel.hostname,
      password: tunnel.password,
      enabled: tunnel.enabled,
      wireguard: tunnel.wireguard,
      ip: tunnel.ip,
    })
    fetchData()
  } catch (err) {
    console.error(err)
    alert(`Error updating tunnel ${tunnel.hostname}`)
  }
}

async function deleteTunnel(tunnel: Tunnel) {
  if (!confirm(`Delete tunnel ${tunnel.hostname}?`)) return
  try {
    await API.delete('/tunnels/' + tunnel.id)
    fetchData()
  } catch (err) {
    console.error(err)
    alert(`Error deleting tunnel ${tunnel.hostname}`)
  }
}

function onSearch() {
  fetchData(1, pageSize.value)
}

function clearSearch() {
  search.value = ''
  fetchData(1, pageSize.value)
}

onMounted(() => {
  bus?.on('tunnel_stats', updateTunnelStats)
  bus?.on('tunnel_connection', updateTunnelConnected)
  bus?.on('tunnel_disconnection', updateTunnelDisconnected)
})

onBeforeUnmount(() => {
  bus?.off('tunnel_stats', updateTunnelStats)
  bus?.off('tunnel_connection', updateTunnelConnected)
  bus?.off('tunnel_disconnection', updateTunnelDisconnected)
})
</script>

<template>
  <div class="space-y-3">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div class="flex items-center gap-2">
        <label class="text-sm font-medium" for="tunnel-search">Search</label>
        <input
          id="tunnel-search"
          v-model="search"
          type="text"
          class="w-48 rounded-md border px-2 py-1 text-sm"
          placeholder="Hostname"
          @keyup.enter="onSearch"
        />
        <Button size="sm" variant="secondary" @click="onSearch">Apply</Button>
        <Button size="sm" variant="ghost" @click="clearSearch">Clear</Button>
      </div>
      <RouterLink v-if="admin" to="/admin/tunnels/create/wireguard">
        <Button size="sm">New Tunnel</Button>
      </RouterLink>
    </div>
    <DataTable
      :columns="columns"
      :data="tunnels"
      pagination
      :rowCount="totalRecords"
      :pageCount="Math.ceil(totalRecords / pageSize) || 1"
      :fetchData="fetchData"
    />
  </div>
</template>
