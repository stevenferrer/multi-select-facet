package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/stevenferrer/solr-go"
	. "github.com/stevenferrer/solr-go/types"
)

type facet struct {
	Name    string   `json:"name"`
	Buckets []bucket `json:"buckets"`
}

type bucket struct {
	Val          string `json:"val"`
	SkuCount     int    `json:"skuCount"`
	ProductCount int    `json:"productCount"`
}

type product struct {
	ID          string `json:"id"`
	Brand       string `json:"brand"`
	Category    string `json:"category"`
	ProductType string `json:"productType"`
}

type facetConfig struct {
	field, facet string
	vals         []string
}

type searchHandler struct {
	solrClient solr.Client
}

func (h *searchHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	categories := strings.Split(r.URL.Query().Get("categories"), ",")
	productTypes := strings.Split(r.URL.Query().Get("productTypes"), ",")
	brands := strings.Split(r.URL.Query().Get("brands"), ",")
	colors := strings.Split(r.URL.Query().Get("colors"), ",")
	sizes := strings.Split(r.URL.Query().Get("sizes"), ",")

	facetsMap := M{}

	catsFacetCfg := facetConfig{
		facet: "categories",
		field: "category",
		vals:  categories,
	}

	productTypsFacetCfg := facetConfig{
		facet: "productTypes",
		field: "productType",
		vals:  productTypes,
	}

	brandsFacetCfg := facetConfig{
		facet: "brands",
		field: "brand",
		vals:  brands,
	}

	filters := []string{}
	for _, facetCfg := range []facetConfig{catsFacetCfg, productTypsFacetCfg, brandsFacetCfg} {
		tagVals := []string{}
		for _, val := range facetCfg.vals {
			if val == "" {
				continue
			}
			tagVals = append(tagVals, facetCfg.field+":"+val)
		}

		if len(tagVals) == 0 {
			filters = append(filters, fmt.Sprintf("{!tag=top}%s:*", facetCfg.field))
		} else {
			filters = append(filters, fmt.Sprintf("{!tag=top}%s", strings.Join(tagVals, " OR ")))
		}

		facetsMap[facetCfg.facet] = M{
			"facet": M{
				"productCount": "uniqueBlock(_root_)",
			},
			"field": facetCfg.field,
			"limit": -1,
			"type":  "terms",
		}

	}

	colorsFctCfg := facetConfig{
		field: "color",
		facet: "colors",
		vals:  colors,
	}

	sizesFctCfg := facetConfig{
		field: "size",
		facet: "sizes",
		vals:  sizes,
	}

	childFqs := []string{}
	for _, fctCfg := range []facetConfig{colorsFctCfg, sizesFctCfg} {
		tagVals := []string{}
		for _, val := range fctCfg.vals {
			if val == "" {
				continue
			}
			tagVals = append(tagVals, fctCfg.field+":"+val)
		}

		if len(tagVals) == 0 {
			childFqs = append(childFqs, fmt.Sprintf("{!tag=%s}%s:*", fctCfg.field, fctCfg.field))
		} else {
			childFqs = append(childFqs, fmt.Sprintf("{!tag=%s}%s", fctCfg.field, strings.Join(tagVals, " OR ")))
		}

		facetsMap[fctCfg.facet] = M{
			"domain": M{
				"excludeTags": "top",
				"filter": []string{
					fmt.Sprintf("{!filters param=$child.fq excludeTags=%s v=$child.query}", fctCfg.field),
					"{!child of=$parent.fq filters=$fq v=$parent.fq}",
				},
			},
			"type":  "terms",
			"field": fctCfg.field,
			"limit": -1,
			"facet": M{
				"productCount": "uniqueBlock(_root_)",
			},
		}
	}

	query := M{
		"query": "{!parent tag=top filters=$child.fq which=$parent.fq score=total v=$child.query}",
		"queries": M{
			"parent.fq":   "docType:product",
			"child.query": "docType:sku",
			"child.fq":    childFqs,
		},
		"filter": filters,
		"fields": "*",
		"facet":  facetsMap,
	}

	// b, err := json.MarshalIndent(query, "", "  ")
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// fmt.Println(string(b))

	queryRespone, err := h.solrClient.Query().Query(r.Context(), collection, query)
	if err != nil {
		http.Error(w, err.Error(), 500)
	}

	var facets = []facet{}
	for k, v := range queryRespone.Facets {
		if k == "count" {
			continue
		}

		vv, ok := v.(map[string]interface{})
		if !ok {
			http.Error(w, "buckets is not map[string]interface{}", 500)
		}

		bucks, ok := vv["buckets"].([]interface{})
		if !ok {
			http.Error(w, "buckets not found", 500)
		}

		buckets := []bucket{}
		for _, bk := range bucks {
			buck, ok := bk.(map[string]interface{})
			if !ok {
				http.Error(w, "buck not map[string]interface{}", 500)
			}

			productCount, ok := buck["productCount"].(float64)
			if !ok {
				http.Error(w, "product count not found", 500)
			}

			skuCount, ok := buck["count"].(float64)
			if !ok {
				http.Error(w, "sku count not found", 500)
			}

			val, ok := buck["val"].(string)
			if !ok {
				http.Error(w, "val not found", 500)
			}

			buckets = append(buckets, bucket{
				Val:          val,
				SkuCount:     int(skuCount),
				ProductCount: int(productCount),
			})

		}

		facets = append(facets, facet{
			Name:    k,
			Buckets: buckets,
		})
	}

	var products = []product{}
	for _, doc := range queryRespone.Response.Docs {
		id, ok := doc["id"].(string)
		if !ok {
			http.Error(w, "id not found", 500)
		}

		brand, ok := doc["brand"].([]interface{})
		if !ok {
			http.Error(w, "brand not found", 500)
		}

		category, ok := doc["category"].([]interface{})
		if !ok {
			http.Error(w, "category not found", 500)
		}

		productType, ok := doc["productType"].(string)
		if !ok {
			http.Error(w, "productType not found", 500)
		}

		products = append(products, product{
			ID:          id,
			Brand:       brand[0].(string),
			Category:    category[0].(string),
			ProductType: productType,
		})
	}

	resp := M{
		"products": products,
		"facets":   facets,
	}

	w.Header().Add("content-type", "application/json")
	err = json.NewEncoder(w).Encode(resp)
	if err != nil {
		http.Error(w, http.StatusText(500), 500)
	}
}
