<template>
  <div>
    <Card>
      <template #title>Mesh Map</template>
      <template #content>
        <p>
          This mesh map is updated every hour on the 30 minute mark.
          The map is <a href="https://github.com/USA-RedDragon/MeshMap" target="_blank">open source</a>
        </p>
        <iframe src="https://oklahoma-supernode.aredn.mcswain.cloud/meshmap/" title="Mesh Map"></iframe>
      </template>
    </Card>
  </div>
</template>

<script>
import Card from 'primevue/card';

export default {
  components: {
    Card,
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
