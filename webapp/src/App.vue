<template>
  <div class="container" id="app">
    <div class="columns">
      <div class="column is-narrow">
        <Filters :facets="facets" @changed="onChanged" />
      </div>
      <div class="column">
        <div class="columns is-multiline is-mobile">
          <div
            v-for="(product, i) in products"
            :key="i"
            class="column is-6-mobile is-4-tablet is-3-desktop is-one-fifth-widescreen is-2-fullhd"
          >
            <Product :data="product" />
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import Filters from "./components/Filters";
import Product from "./components/Product";

async function search(filters) {
  if (!filters) {
    filters = [];
  }

  const url = new URL("http://localhost:8081/search");
  filters.forEach((filter) => {
    url.searchParams.append(filter.name, filter.selected.join(","));
  });

  let response = await fetch(url, {
    method: "GET",
    mode: "cors",
    headers: {
      Accept: "application/json",
      "Content-Type": "application/json;charset=UTF-8",
    },
  });

  if (response && response.ok) {
    const { products, facets } = await response.json();
    return {
      products,
      facets: facets
        .map((facet) => {
          let selected = [];
          let filter = filters.find((filter) => filter.name === facet.name);
          if (filter) {
            selected = filter.selected;
          }

          // sort bucket values
          facet.buckets = facet.buckets.sort((a, b) =>
            a.val.localeCompare(b.val)
          );

          return {
            ...facet,
            selected,
          };
        })
        .sort((a, b) => a.name.localeCompare(b.name)),
    };
  }

  return {
    products: null,
    facets: null,
  };
}

export default {
  name: "App",
  components: {
    Filters,
    Product,
  },
  data() {
    return {
      products: null,
      facets: null,
    };
  },
  async mounted() {
    const { products, facets } = await search();
    this.products = products;
    this.facets = facets;
  },
  methods: {
    async onChanged() {
      const { facets: oldFacet } = this;
      const filters = oldFacet
        .filter(({ selected }) => selected.length > 0)
        .map(({ name, selected }) => {
          return { name, selected };
        });

      const { products, facets } = await search(filters);
      this.products = products;
      this.facets = facets;
    },
  },
};
</script>
