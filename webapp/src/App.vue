<template>
  <div class="container" id="app">
    <div class="columns">
      <div class="column is-narrow">
        <Filters />
      </div>
      <div class="column">
        <b-field>
          <b-input
            placeholder="Search..."
            type="search"
            icon-pack="fas"
            icon="search"
          >
          </b-input>
        </b-field>
        <div class="columns is-multiline is-mobile">
          <div
            v-for="i in 10"
            :key="i"
            class="column is-6-mobile is-4-tablet is-3-desktop is-one-fifth-widescreen is-2-fullhd"
          >
            <Product />
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import Filters from "./components/Filters";
import Product from "./components/Product";

export default {
  name: "App",
  components: {
    Filters,
    Product,
  },
};
</script>
