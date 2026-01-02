<template>
  <div>
    <Card>
      <CardHeader>
        <CardTitle>Manage Admins</CardTitle>
      </CardHeader>
      <CardContent>
        <UserTable />
      </CardContent>
    </Card>
  </div>
</template>

<script lang="ts">
import UserTable from '@/components/UserTable.vue';
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
    UserTable,
  },
};
</script>

<style scoped></style>
