SOLR_INST ?="solr-multi-select-facet-demo" 
COLLECTION ?= "multi-select-demo"

.PHONY: solr
solr: stop-solr
	docker run -d -p 8983:8983 --name $(SOLR_INST) solr:latest solr-precreate $(COLLECTION)

.PHONY: stop-solr
stop-solr:
	docker rm -f $(SOLR_INST) || true