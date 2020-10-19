DOCKER ?= docker
SOLR_INST ?="solr-multi-select-facet-demo" 
COLLECTION ?= "multi-select-facet-demo"

.PHONY: solr
solr: stop-solr
	$(DOCKER) run -d -p 8983:8983 --name $(SOLR_INST) solr:latest solr-precreate $(COLLECTION)

.PHONY: stop-solr
stop-solr:
	$(DOCKER) rm -f $(SOLR_INST) || true