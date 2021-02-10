PODMAN ?= podman
SOLR ?="multi-select-facet-demo" 

.PHONY: build-api
build-api: 
	go build -v -o api

.PHONY: start-solr
start-solr: stop-solr
	$(PODMAN) run -d -p 8983:8983 --name $(SOLR) solr:8 solr -c -f
	$(PODMAN) exec -it $(SOLR) bash -c 'sleep 5; wait-for-solr.sh --max-attempts 10 --wait-seconds 5'

.PHONY: stop-solr
stop-solr:
	$(PODMAN) rm -f $(SOLR) || true
