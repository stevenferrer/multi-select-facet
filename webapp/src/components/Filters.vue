<template>
  <div>
    <div v-for="(facet, i) in facets" :key="i">
      <h1 class="title">{{ facet.name }}</h1>
      <div class="fields">
        <div v-for="(bucket, j) in facet.buckets.slice(0, 5)" :key="j">
          <b-checkbox
            v-model="facet.selected"
            :native-value="bucket.val"
            @input="onChecked"
          >
            {{ bucket.val }} ({{ bucket.productCount }})
          </b-checkbox>
        </div>

        <b-collapse
          v-if="facet.buckets.length > 5"
          :open="false"
          position="is-bottom"
          aria-id="contentIdForA11y1"
        >
          <a
            slot="trigger"
            slot-scope="props"
            aria-controls="contentIdForA11y1"
          >
            <b-icon
              :icon="!props.open ? 'chevron-down' : 'chevron-up'"
            ></b-icon>
            {{ !props.open ? "Show more" : "Show less" }}
          </a>
          <div v-for="(bucket, j) in facet.buckets.slice(5)" :key="j">
            <b-checkbox
              v-model="facet.selected"
              :native-value="bucket.val"
              @input="onChecked"
            >
              {{ bucket.val }} ({{ bucket.productCount }})
            </b-checkbox>
          </div>
        </b-collapse>
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
