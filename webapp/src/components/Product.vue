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
          <span class="is-size-6">My Product</span>
        </div>
        <br />
        <span class="is-size-6 has-text-success">PHP 50.00 </span>
        <br />
        <span>
          <b-rate v-model="rating" show-score disabled size="is-small">
          </b-rate>
        </span>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  data() {
    return {
      rating: 4.3,
    };
  },
};
</script>
