<template>
  <section>
    <b-field>
      <b-autocomplete
        :data="data"
        placeholder="e.g. Apple IPhone 11"
        field="term"
        icon="search"
        :loading="isFetching"
        @typing="getAsyncData"
        @select="(option) => (selected = option)"
        clearable
      >
        <template slot-scope="props">
          {{ props.option.term }}
        </template>
      </b-autocomplete>
    </b-field>
  </section>
</template>

<script>
import debounce from "lodash/debounce";

export default {
  data() {
    return {
      data: [],
      selected: null,
      isFetching: false,
    };
  },
  methods: {
    getAsyncData: debounce(function(name) {
      if (!name.length) {
        this.data = [];
        return;
      }
      this.isFetching = true;

      this.$http
        .get(`http://localhost:8081/suggest?q=${encodeURIComponent(name)}`)
        .then(({ data }) => {
          this.data = [];
          data.suggestions.forEach((item) => this.data.push(item));
        })
        .catch((error) => {
          this.data = [];
          throw error;
        })
        .finally(() => {
          this.isFetching = false;
        });
    }, 500),
  },
};
</script>
