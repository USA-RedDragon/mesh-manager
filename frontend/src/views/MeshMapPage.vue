<template>
  <div>
    <Card>
      <CardHeader>
        <CardTitle>Mesh Map</CardTitle>
      </CardHeader>
      <CardContent>
        <p>
          This mesh map is updated every hour on the 30 minute mark.
          The map is <a href="https://github.com/USA-RedDragon/MeshMap" target="_blank">open source</a>
        </p>
        <iframe src="https://oklahoma-supernode.aredn.mcswain.cloud/meshmap/" title="Mesh Map"></iframe>
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

export default {
  components: {
    Card,
    CardContent,
    CardHeader,
    CardTitle,
  },
  created() {
  },
  mounted() {
  },
  unmounted() {
  },
  data: function() {
    return {
    };
  },
  methods: {
  },
  computed: {},
};
</script>

<style scoped>
iframe {
  width: 100%;
  height: 100vh;
  border: none;
}
</style>
