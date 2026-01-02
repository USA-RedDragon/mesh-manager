<template>
  <div>
    <Card>
      <CardHeader>
        <CardTitle>Babel Mesh</CardTitle>
      </CardHeader>
      <CardContent>
        <NodesTable babel />
      </CardContent>
    </Card>
  </div>
</template>

<script lang="ts">
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import NodesTable from '../components/NodesTable.vue';

export default {
  components: {
    Card,
    CardContent,
    CardHeader,
    CardTitle,
    NodesTable,
  },
  created() {},
  mounted() {},
  unmounted() {},
  data: function() {
    return {};
  },
  methods: {},
  computed: {},
};
</script>

<style scoped></style>
