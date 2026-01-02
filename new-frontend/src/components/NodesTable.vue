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
}

interface Node {
  hostname: string
  ip: string
  services: Service[]
  children: NodeNoChildren[]
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

async function fetchData(page=1, pageSize=10) {
  loading.value = true;
  limit.value = pageSize;
  const api = props.babel ? '/babel' : '/olsr';

  API.get(`${api}/hosts/count`)
    .then((res) => {
      nodesCount.value = res.data.nodes;
      devicesCount.value = res.data.total;
    })
    .catch((err) => {
      console.error(err);
    });

  const params = [`page=${page}`, `limit=${pageSize}`];
  if (search.value) {
    params.push(`filter=${encodeURIComponent(search.value.trim())}`);
  }

  API.get(`${api}/hosts?${params.join('&')}`)
    .then((res) => {
      if (!res.data.nodes) {
        res.data.nodes = [];
      }

      // Iterate through each node's services and each node's child's services
      // and make them a new URL()
      for (let i = 0; i < res.data.nodes.length; i++) {
        const node = res.data.nodes[i];
        if (node.services != null) {
          for (let j = 0; j < node.services.length; j++) {
            const service = node.services[j];
            service.url = new URL(service.url);
            service.url.hostname = service.url.hostname + '.local.mesh';
            node.services[j] = service;
          }
        }
        if (node.children != null) {
          for (let j = 0; j < node.children.length; j++) {
            const child = node.children[j];
            if (child.services != null) {
              for (let k = 0; k < child.services.length; k++) {
                const service = child.services[k];
                service.url = new URL(service.url);
                service.url.hostname = service.url.hostname + '.local.mesh';
                child.services[k] = service;
              }
            }
            node.children[j] = child;
          }
        }
        res.data.nodes[i] = node;
      }

      data.value = res.data.nodes;
      totalRecords.value = res.data.total;
      loading.value = false;
    })
    .catch((err) => {
      console.error(err);
    });
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
