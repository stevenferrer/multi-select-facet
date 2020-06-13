<template>
  <div class="card">
    <div class="card-image">
      <figure class="image is-1by1 zoom">
        <img src="https://bulma.io/images/placeholders/1280x960.png" />
      </figure>
    </div>
    <div class="card-content">
      <div>
        <div class="product-title">
          <span class="is-size-6">{{ data.name }}</span
          ><br />
        </div>

        <span class="has-text-grey">{{ data.brand }}</span>
        <br />
        <span class="is-size-6 has-text-success">PHP 1,501.30 </span>
        <br />
        <span>
          <b-rate v-model="rating" show-score disabled size="is-small">
          </b-rate>
        </span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.card {
  overflow: hidden;
  position: relative;
  border-radius: 0.5em;
}

.product-title {
  display: -webkit-box;
  height: 2em;
  -webkit-line-clamp: 3;
  line-height: 1em;
  -webkit-box-orient: vertical;
  overflow: hidden;
  text-overflow: ellipsis;
}
</style>

<script>
export default {
  props: {
    data: {
      type: Object,
    },
  },
  data() {
    return {
      rating: 4.3,
    };
  },
};
</script>
