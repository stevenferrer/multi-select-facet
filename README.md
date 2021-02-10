# Multi-Select Facet Example
An example of multi-select facet using [Solr](https://lucene.apache.org/solr), [Vue](https://vuejs.org) and [Go](http://go.dev/). 

Blog post: [Multi-Select Facet with Solr, Vue and Go](https://sf9v.github.io/posts/multi-select-facet-solr-vue-go/)

![screenshot](./screenshot.png)

## Running the example

1. Run Solr using [podman](https://podman.io/).

```console
$ make start-solr
```

Or using `docker`.

```console
$ PODMAN=docker make start-solr
```

2. Run the API
```console
// build the api
$ go build -v -o api

// start the api with the initialization options
$ ./api -create-collection -init-schema -index-data -init-suggester
```

3. Run the web app (open a new terminal tab)
```console
// cd to web app folder
$ cd webapp

// install dependencies
$ yarn // or npm install

// start the web app
$ yarn serve // or npm run serve
```

4. Open http://localhost:8080 in your browser and start playing with it!


## Contributing
Feel free to improve this project by [make a pull-request](https://github.com/sf9v/multi-select-facet/pulls) or [opening an issue](https://github.com/sf9v/multi-select-facet/issues).

## License

MIT