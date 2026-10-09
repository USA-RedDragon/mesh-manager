<script setup lang="ts" generic="TData extends RowData">
import type { ColumnDef, PaginationState, RowData, Updater } from '@tanstack/vue-table'
import { FlexRender, useTable } from '@tanstack/vue-table'
import { toRefs, onMounted, ref } from 'vue'
import DataTablePagination from './DataTablePagination.vue'
import { dataTableFeatures, type DataTableFeatures } from './features'

import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'

const props = defineProps<{
  columns: ColumnDef<DataTableFeatures, TData>[]
  data: TData[]
  pagination?: boolean
  rowCount?: number
  pageCount?: number
  fetchData: (_pageIndex: number, _pageSize: number) => Promise<void>
}>()

const { data, rowCount, pageCount } = toRefs(props)

const paginationState = ref<PaginationState>({ pageIndex: 0, pageSize: 10 })

const table = useTable({
  features: dataTableFeatures,
  get data() { return data.value },
  get columns() { return props.columns },
  get pageCount() {
    const fallback = Math.ceil(((rowCount.value ?? 0) || 0) / (paginationState.value.pageSize || 1)) || 1
    return pageCount.value ?? fallback
  },
  get rowCount() { return rowCount.value ?? data.value.length },
  manualPagination: props.pagination,
  get state() {
    return { pagination: paginationState.value }
  },
  onPaginationChange: (updater: Updater<PaginationState>) => {
    const nextState = typeof updater === 'function' ? updater(paginationState.value) : updater
    paginationState.value = nextState
    props.fetchData(nextState.pageIndex + 1, nextState.pageSize)
  },
})

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
              <FlexRender v-if="!header.isPlaceholder" :header="header" />
            </TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          <template v-if="table.getRowModel().rows?.length">
            <TableRow
              v-for="row in table.getRowModel().rows" :key="row.id"
              :data-state="row.getIsSelected() ? 'selected' : undefined"
            >
              <TableCell v-for="cell in row.getAllCells()" :key="cell.id">
                <FlexRender :cell="cell" />
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
