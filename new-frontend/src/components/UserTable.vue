<script setup lang="ts">
import type { ColumnDef } from '@tanstack/vue-table'
import moment from 'moment'
import { h, onMounted, ref } from 'vue'
import { RouterLink } from 'vue-router'

import DataTable from './datatable/DataTable.vue'
import { Button } from '@/components/ui/button'
import API from '@/services/API'

interface User {
  id: number
  username: string
  created_at: string
}

const users = ref<User[]>([])
const totalRecords = ref(0)
const pageSize = ref(10)
const loading = ref(false)

const columns: ColumnDef<User>[] = [
  {
    accessorKey: 'id',
    header: () => h('span', 'ID'),
    cell: ({ row }) => h('span', row.getValue('id') as string),
  },
  {
    accessorKey: 'username',
    header: () => h('span', 'Username'),
    cell: ({ row }) => h('span', row.getValue('username') as string),
  },
  {
    accessorKey: 'created_at',
    header: () => h('span', 'Created'),
    cell: ({ row }) => {
      const created = row.getValue('created_at') as string
      return h('span', moment(created).fromNow())
    },
  },
  {
    id: 'actions',
    header: () => h('span', 'Actions'),
    cell: ({ row }) => h('div', { class: 'flex gap-2 justify-end' }, [
      h(Button, {
        variant: 'destructive',
        size: 'sm',
        onClick: () => deleteUser(row.original),
      }, () => 'Delete'),
    ]),
  },
]

async function fetchData(page = 1, limit = pageSize.value) {
  loading.value = true
  try {
    const res = await API.get(`/users?page=${page}&limit=${limit}`)
    const normalized = (res.data.users || []).map((user: User) => ({
      ...user,
      created_at: user.created_at,
    }))
    users.value = normalized
    totalRecords.value = res.data.total
    pageSize.value = limit
  } catch (err) {
    console.error(err)
  } finally {
    loading.value = false
  }
}

function deleteUser(user: User) {
  if (user.id === 1) {
    alert('The system account cannot be deleted.')
    return
  }
  if (!confirm(`Delete user ${user.username}?`)) {
    return
  }
  API.delete('/users/' + user.id)
    .then(() => {
      fetchData()
    })
    .catch((err) => {
      console.error(err)
      alert(`Error deleting user ${user.username}`)
    })
}

onMounted(() => {
  fetchData()
})
</script>

<template>
  <div class="space-y-3">
    <div class="flex justify-between items-center">
      <h3 class="text-lg font-semibold">Admins</h3>
      <RouterLink to="/admin/users/register">
        <Button size="sm">Enroll New Admin</Button>
      </RouterLink>
    </div>
    <DataTable
      :columns="columns"
      :data="users"
      pagination
      :rowCount="totalRecords"
      :pageCount="Math.ceil(totalRecords / pageSize) || 1"
      :fetchData="fetchData"
    />
  </div>
</template>
