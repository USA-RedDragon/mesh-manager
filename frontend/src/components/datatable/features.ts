import { rowPaginationFeature, rowSelectionFeature, tableFeatures } from '@tanstack/vue-table'

/** Table features shared by every DataTable in the app. */
export const dataTableFeatures = tableFeatures({ rowPaginationFeature, rowSelectionFeature })

/** Feature set type for column definitions passed to DataTable. */
export type DataTableFeatures = typeof dataTableFeatures
