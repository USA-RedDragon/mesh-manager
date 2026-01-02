<template>
  <div>
    <Card>
      <CardHeader>
        <CardTitle>Tunnels</CardTitle>
      </CardHeader>
      <CardContent>
        <TunnelsTable />
      </CardContent>
    </Card>
  </div>
</template>

<script lang="ts">
import TunnelsTable from '@/components/TunnelsTable.vue';
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';

export default {
  components: {
    Card,
    CardContent,
    CardHeader,
    CardTitle,
    TunnelsTable,
  },
};
</script>

<style scoped></style>
