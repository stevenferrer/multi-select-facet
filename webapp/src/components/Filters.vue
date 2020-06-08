<template>
  <div>
    <div class="filter">
      <h1>Color Family</h1>
      <div class="fields">
        <div v-for="(v, i) in ['Black', 'Red', 'White', 'Purple']" :key="i">
          <b-checkbox>
            {{ v }}
          </b-checkbox>
        </div>
      </div>
    </div>

    <hr />

    <div class="filter">
      <h1>Service</h1>
      <div class="fields">
        <div
          v-for="(v, i) in [
            'Global Collection',
            'Cash On Delivery',
            'Filfulled By Lazada',
            'Free Shipping',
          ]"
          :key="i"
        >
          <b-checkbox>
            {{ v }}
          </b-checkbox>
        </div>
      </div>
    </div>
    <hr />

    <div class="filter">
      <h1>Location</h1>
      <div class="fields">
        <div v-for="(v, i) in ['NCR', 'Overseas']" :key="i">
          <b-checkbox>
            {{ v }}
          </b-checkbox>
        </div>
      </div>
    </div>
  </div>
</template>
