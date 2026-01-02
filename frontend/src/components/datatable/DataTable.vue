<script setup lang="ts" generic="TData, TValue">
import type { ColumnDef, TableOptionsWithReactiveData } from '@tanstack/vue-table'
import {
  FlexRender,
  getCoreRowModel,
  useVueTable,
} from '@tanstack/vue-table'
import { toRefs, onMounted, ref } from 'vue'
import DataTablePagination from './DataTablePagination.vue'

import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'

const props = defineProps<{
  columns: ColumnDef<TData, TValue>[]
  data: TData[]
  pagination?: boolean
  rowCount?: number
  pageCount?: number
  fetchData: (_pageIndex: number, _pageSize: number) => Promise<void>
}>()

const { data, rowCount, pageCount } = toRefs(props)

const paginationState = ref({ pageIndex: 0, pageSize: 10 })

const options: TableOptionsWithReactiveData<TData> = {
  get data() { return data.value },
  get columns() { return props.columns },
  get pageCount() {
    const fallback = Math.ceil(((rowCount.value ?? 0) || 0) / (paginationState.value.pageSize || 1)) || 1
    return pageCount.value ?? fallback
  },
  get rowCount() { return rowCount.value ?? data.value.length },
  getCoreRowModel: getCoreRowModel(),
  manualPagination: props.pagination,
  get state() {
    return {
      pagination: paginationState.value,
    }
  },
  onPaginationChange: (updater) => {
    const nextState = typeof updater === 'function' ? updater(paginationState.value) : updater
    paginationState.value = nextState
    table.setOptions((prev) => ({
      ...prev,
      state: {
        ...prev.state,
        pagination: nextState,
      },
    }))
    props.fetchData(nextState.pageIndex + 1, nextState.pageSize)
  },
}

const table = useVueTable(options)

onMounted(() => {
  props.fetchData(paginationState.value.pageIndex + 1, paginationState.value.pageSize)
})
</script>

<template>
  <div>
    <div class="border rounded-md">
      <Table>
        <TableHeader>
          <TableRow v-for="headerGroup in table.getHeaderGroups()" :key="headerGroup.id">
            <TableHead v-for="header in headerGroup.headers" :key="header.id">
              <FlexRender
                v-if="!header.isPlaceholder" :render="header.column.columnDef.header"
                :props="header.getContext()"
              />
            </TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          <template v-if="table.getRowModel().rows?.length">
            <TableRow
              v-for="row in table.getRowModel().rows" :key="row.id"
              :data-state="row.getIsSelected() ? 'selected' : undefined"
            >
              <TableCell v-for="cell in row.getVisibleCells()" :key="cell.id">
                <FlexRender :render="cell.column.columnDef.cell" :props="cell.getContext()" />
              </TableCell>
            </TableRow>
          </template>
          <template v-else>
            <TableRow>
              <TableCell :colspan="columns.length" class="h-24 text-center">
                No results.
              </TableCell>
            </TableRow>
          </template>
        </TableBody>
      </Table>
    </div>
    <DataTablePagination v-if="props.pagination" :table="table" />
  </div>
</template>
