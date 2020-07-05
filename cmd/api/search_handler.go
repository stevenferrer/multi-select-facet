package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/stevenferrer/solr-go"
	"github.com/stevenferrer/solr-go/query"
)

// searchHandler is the search handler
type searchHandler struct {
	collection string
	solrClient solr.Client
}

// facetConfig is a facet configuration
type facetConfig struct {
	// field is the field name
	field,
	// facet is the facet name
	facet,
	// param is the url query param to be used for retrieving filter values
	urlParam string
	// isChild set to true to indicate that this is a child facet
	isChild bool
}

var facetConfigs = []facetConfig{
	// {
	// 	facet:    "Category",
	// 	field:    "category",
	// 	urlParam: "categories",
	// },
	{
		facet:    "Product Type",
		field:    "productType",
		urlParam: "productTypes",
	},
	{
		facet:    "Brand",
		field:    "brand",
		urlParam: "brands",
	},
	{
		facet:    "Color Family",
		field:    "colorFamily_s",
		urlParam: "colorFamilies",
		isChild:  true,
	},
	// {
	// 	facet:    "Sim Card Slots",
	// 	field:    "simCardSlots_s",
	// 	urlParam: "simCardSlots",
	// 	isChild:  true,
	// },
	{
		facet:    "Operating System",
		field:    "operatingSystem_s",
		urlParam: "operatingSystems",
		isChild:  true,
	},
	{
		facet:    "Storage Capacity",
		field:    "storageCapacity_s",
		urlParam: "storageCapacities",
		isChild:  true,
	},
}

func (h *searchHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	facetsMap := Map{}
	productFilters, skuFilters := []string{}, []string{}
	for _, fctCfg := range facetConfigs {
		tagVals := []string{}
		filterVals := strings.Split(r.URL.Query().Get(fctCfg.urlParam), ",")
		for _, val := range filterVals {
			if val == "" {
				continue
			}
			tagVals = append(tagVals, fctCfg.field+":"+val)
		}

		if !fctCfg.isChild {
			// product filters
			if len(tagVals) > 0 {
				productFilters = append(productFilters, fmt.Sprintf("{!tag=top}%s", strings.Join(tagVals, " OR ")))
			} else {
				productFilters = append(productFilters, fmt.Sprintf("{!tag=top}%s:*", fctCfg.field))
			}

			facetsMap[fctCfg.facet] = Map{
				"facet": Map{
					"productCount": "uniqueBlock(_root_)",
				},
				"field": fctCfg.field,
				"limit": -1,
				"type":  "terms",
			}

			continue
		}

		// sku filters
		if len(tagVals) > 0 {
			skuFilters = append(skuFilters, fmt.Sprintf("{!tag=%s}%s", fctCfg.field, strings.Join(tagVals, " OR ")))
		} else {
			skuFilters = append(skuFilters, fmt.Sprintf("{!tag=%s}%s:*", fctCfg.field, fctCfg.field))
		}

		facetsMap[fctCfg.facet] = Map{
			"domain": Map{
				"excludeTags": "top",
				"filter": []string{
					fmt.Sprintf("{!filters param=$skuFilters excludeTags=%s v=$sku}", fctCfg.field),
					"{!child of=$product filters=$filter v=$product}",
				},
			},
			"type":  "terms",
			"field": fctCfg.field,
			"limit": -1,
			"facet": Map{
				"productCount": "uniqueBlock(_root_)",
			},
		}
	}

	q := r.URL.Query().Get("q")
	if len(q) == 0 {
		q = "*"
	} else {
		q = fmt.Sprintf("%q", q)
	}

	productFilters = append(productFilters, fmt.Sprintf("{!tag=top}_text_:%s", q))

	query := Map{
		"query": "{!parent tag=top filters=$skuFilters which=$product score=total v=$sku}",
		"queries": Map{
			"product":    "docType:product",
			"sku":        "docType:sku",
			"skuFilters": skuFilters,
		},
		"filter": productFilters,
		"facet":  facetsMap,
	}

	queryResp, err := h.solrClient.Query().Query(r.Context(), h.collection, query)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// extract products and facets from query response
	resp, err := buildResp(queryResp)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	debug := r.URL.Query().Get("debug") == "true"
	if debug {
		resp["query"] = query
	}

	w.Header().Add("content-type", "application/json")
	err = json.NewEncoder(w).Encode(resp)
	if err != nil {
		http.Error(w, err.Error(), 500)
	}
}

func buildResp(queryResp *query.Response) (Map, error) {
	var facets = []Map{}
	for name, v := range queryResp.Facets {
		if name == "count" {
			continue
		}

		vv, ok := v.(Map)
		if !ok {
			return nil, errors.New("v is not M")
		}

		bucks, ok := vv["buckets"].([]Any)
		if !ok {
			return nil, errors.New("vv is not []Any")
		}

		buckets := []Map{}
		for _, bk := range bucks {
			buck, ok := bk.(Map)
			if !ok {
				return nil, errors.New("bucket is not M")
			}

			productCount, ok := buck["productCount"].(float64)
			if !ok {
				return nil, errors.New("productCount is not float64")
			}

			skuCount, ok := buck["count"].(float64)
			if !ok {
				return nil, errors.New("count is not float64")
			}

			val, ok := buck["val"].(string)
			if !ok {
				return nil, errors.New("val is not string")
			}

			buckets = append(buckets, Map{
				"val":          val,
				"skuCount":     int(skuCount),
				"productCount": int(productCount),
			})

		}

		var param string
		for _, fctCfg := range facetConfigs {
			if fctCfg.facet == name {
				param = fctCfg.urlParam
			}
		}

		facets = append(facets, Map{
			"name":    name,
			"param":   param,
			"buckets": buckets,
		})
	}

	var products = []Map{}
	for _, doc := range queryResp.Response.Docs {
		id, ok := doc["id"].(string)
		if !ok {
			return nil, errors.New("id not found or is not string")
		}

		name, ok := doc["name"].([]Any)
		if !ok {
			return nil, errors.New("name not found or is not []Any")
		}

		category, ok := doc["category"].([]Any)
		if !ok {
			return nil, errors.New("category not found or is not []Any")
		}

		brand, ok := doc["brand"].([]Any)
		if !ok {
			return nil, errors.New("brand not found or is not []Any")
		}

		productType, ok := doc["productType"].(string)
		if !ok {
			return nil, errors.New("productType not found or is not string")
		}

		products = append(products, Map{
			"id":          id,
			"name":        name[0].(string),
			"category":    category[0].(string),
			"brand":       brand[0].(string),
			"productType": productType,
		})
	}

	return Map{
		"products": products,
		"facets":   facets,
	}, nil
}
