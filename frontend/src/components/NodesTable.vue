<script setup lang="ts">
import type { ColumnDef } from '@tanstack/vue-table'
import { onMounted, ref } from 'vue'
import DataTable from './datatable/DataTable.vue'
import API from '../services/API'

import { h } from 'vue'
import { Button as UiButton } from '@/components/ui/button'

interface Service {
  url: string
  protocol: string
  name: string
  shouldLink: boolean
}

interface NodeNoChildren {
  hostname: string
  ip: string
  services: Service[]
  etx?: number | null
}

interface Node {
  hostname: string
  ip: string
  services: Service[]
  children: NodeNoChildren[]
  etx?: number | null
}

const columns: ColumnDef<Node>[] = [
  {
    accessorKey: 'hostname',
    header: () => h('div', {  }, 'Name'),
    cell: ({ row }) => {
      const hostname = row.getValue('hostname')
      return h('a', {
        target: "_blank",
        href: `http://${hostname}.local.mesh`,
        class: 'text-primary underline underline-offset-2 font-medium'
      }, row.getValue('hostname'))
    },
  },
  {
    accessorKey: 'ip',
    header: () => h('div', {  }, 'IP'),
    cell: ({ row }) => {
      return h('p', { }, row.getValue('ip'))
    },
  },
  {
    accessorKey: 'etx',
    header: () => h('div', {  }, 'ETX'),
    cell: ({ row }) => {
      const value = row.getValue('etx') as number | null | undefined
      if (value === null || value === undefined) {
        return h('span', { }, '—')
      }
      return h('span', { }, String(value))
    },
  },
  {
    accessorKey: 'children',
    header: () => h('div', {  }, 'Devices'),
    cell: ({ row }) => {
      const devices = row.getValue('children') as NodeNoChildren[]
      const ret = []
      if (!devices || devices.length === 0) {
        return h('p', { }, '')
      }
      for (let i = 0; i < devices.length; i++) {
        const device = devices[i]
        ret.push(h('p', { }, device.hostname + ' (' + device.ip + ')'))
      }
      return h('div', { }, ret)
    },
  },
  {
    accessorKey: 'services',
    header: () => h('div', {  }, 'Services'),
    cell: ({ row }) => {
      const services = row.getValue('services') as Service[]
      const ret = []
      if (!services || services.length === 0) {
        return h('p', { }, '')
      }
      for (let i = 0; i < services.length; i++) {
        const service = services[i]
        ret.push(h('a', {
          target: "_blank",
          href: service.url,
          class: 'text-primary underline underline-offset-2 font-medium'
        }, service.name))
        if (i < services.length - 1) {
          ret.push(h('br', { }))
        }
      }
      return h('div', { }, ret)
    },
  },
]

const data = ref<Node[]>([])
const loading = ref(false)
const devicesCount = ref(0)
const nodesCount = ref(0)
const totalRecords = ref(0)
const limit = ref(10)
const search = ref('')

const props = defineProps<{
  babel?: boolean
}>()

function ipv4ToInt(ip: string): number | null {
  const parts = ip.split('.').map((p) => Number(p))
  if (parts.length !== 4 || parts.some((p) => Number.isNaN(p) || p < 0 || p > 255)) return null
  return (((parts[0] << 24) >>> 0) | ((parts[1] << 16) >>> 0) | ((parts[2] << 8) >>> 0) | (parts[3] >>> 0)) >>> 0
}

function parseCIDR(cidr: string): { prefix: number; length: number } | null {
  const [ipPart, lenPart] = cidr.split('/')
  const length = Number(lenPart)
  if (Number.isNaN(length) || length < 0 || length > 32) return null
  const ipInt = ipv4ToInt(ipPart)
  if (ipInt === null) return null
  return { prefix: ipInt, length }
}

function etxForIp(ip: string, etxMap: Record<string, number>): number | null {
  const ipInt = ipv4ToInt(ip)
  if (ipInt === null) return null

  let bestLen = -1
  let bestMetric: number | null = null

  for (const [cidr, metric] of Object.entries(etxMap)) {
    const parsed = parseCIDR(cidr)
    if (!parsed) continue

    const mask = parsed.length === 0 ? 0 : ((0xffffffff << (32 - parsed.length)) >>> 0)
    if ((ipInt & mask) === (parsed.prefix & mask)) {
      if (parsed.length > bestLen || (parsed.length === bestLen && bestMetric !== null && metric < bestMetric)) {
        bestLen = parsed.length
        bestMetric = metric
      }
    }
  }

  return bestMetric
}

async function fetchData(page = 1, pageSize = 10) {
  loading.value = true
  limit.value = pageSize
  const api = props.babel ? '/babel' : '/olsr'

  const params = [`page=${page}`, `limit=${pageSize}`]
  if (search.value) {
    params.push(`filter=${encodeURIComponent(search.value.trim())}`)
  }

  try {
    const countPromise = API.get(`${api}/hosts/count`)
    const hostsPromise = API.get(`${api}/hosts?${params.join('&')}`)
    const etxPromise = props.babel ? API.get(`${api}/etx`) : null

    const [countRes, hostsRes, etxRes] = await Promise.all([
      countPromise,
      hostsPromise,
      etxPromise ?? Promise.resolve(null),
    ])

    nodesCount.value = countRes.data.nodes
    devicesCount.value = countRes.data.total

    const etxMap: Record<string, number> = etxRes?.data?.etx ?? {}

    const nodes = hostsRes.data.nodes || []

    for (let i = 0; i < nodes.length; i++) {
      const node = nodes[i]

      node.etx = etxForIp(node.ip, etxMap)

      if (node.services != null) {
        for (let j = 0; j < node.services.length; j++) {
          const service = node.services[j]
          const url = new URL(service.url)
          url.hostname = url.hostname + '.local.mesh'
          service.url = url.toString()
          node.services[j] = service
        }
      }

      if (node.children != null) {
        for (let j = 0; j < node.children.length; j++) {
          const child = node.children[j]
          if (child.services != null) {
            for (let k = 0; k < child.services.length; k++) {
              const service = child.services[k]
              const url = new URL(service.url)
              url.hostname = url.hostname + '.local.mesh'
              service.url = url.toString()
              child.services[k] = service
            }
          }
          node.children[j] = child
        }
      }

      nodes[i] = node
    }

    data.value = nodes
    totalRecords.value = hostsRes.data.total
  } catch (_err) {
    // Errors are logged to console; UI will just omit ETX/rows on failure
    console.error(_err)
  } finally {
    loading.value = false
  }
}

function onSearch() {
  fetchData(1, limit.value)
}

function clearSearch() {
  search.value = ''
  fetchData(1, limit.value)
}

onMounted(() => {
  fetchData(1, limit.value)
})
</script>

<template>
  <div class="mx-auto space-y-3">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <p class="text-sm text-muted-foreground">Found {{ nodesCount }} nodes and {{ devicesCount }} total devices.</p>
      <div class="flex items-center gap-2">
        <label class="text-sm font-medium" for="nodes-search">Search</label>
        <input
          id="nodes-search"
          v-model="search"
          type="text"
          class="w-48 rounded-md border px-2 py-1 text-sm"
          placeholder="Hostname"
          @keyup.enter="onSearch"
        />
        <UiButton size="sm" variant="secondary" @click="onSearch">Apply</UiButton>
        <UiButton size="sm" variant="ghost" @click="clearSearch">Clear</UiButton>
      </div>
    </div>
    <DataTable
      :columns="columns"
      :data="data"
      pagination
      :rowCount="totalRecords"
      :pageCount="Math.ceil(totalRecords / limit) || 1"
      :fetchData="fetchData"
    />
  </div>
</template>
