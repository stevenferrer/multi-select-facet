<template>
  <div>
    <div v-for="(facet, i) in facets" :key="i">
      <h1 class="title">{{ facet.name }}</h1>
      <div class="fields">
        <div v-for="(bucket, j) in facet.buckets" :key="j">
          <b-checkbox
            v-model="facet.selected"
            :native-value="bucket.val"
            @input="onChecked"
          >
            {{ bucket.val }} ({{ bucket.productCount }})
          </b-checkbox>
        </div>
      </div>

      <hr />
    </div>
  </div>
</template>

<script>
export default {
  props: {
    facets: {
      type: Array,
      default: () => [],
    },
    filtered: {
      type: Object,
      default: () => {},
    },
  },
  methods: {
    onChecked() {
      this.$emit("changed");
    },
  },
};
</script>
