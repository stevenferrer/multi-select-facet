package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/stevenferrer/solr-go"
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
	facets := []solr.Faceter{}
	productFilters, skuFilters := []string{}, []string{}
	for _, fc := range facetConfigs {
		filterVals := strings.Split(r.URL.Query().Get(fc.urlParam), ",")
		tagVals := []string{}
		for _, val := range filterVals {
			if val == "" {
				continue
			}
			tagVals = append(tagVals, fc.field+":"+val)
		}

		if !fc.isChild {
			// product filters
			if len(tagVals) > 0 {
				productFilters = append(productFilters, solr.NewStandardQueryParser().
					Tag("top").Query("'"+strings.Join(tagVals, " OR ")+"'").BuildParser())
			} else {
				productFilters = append(productFilters, solr.NewStandardQueryParser().
					Tag("top").Query(fc.field+":*").BuildParser())
			}

			termsFacet := solr.NewTermsFacet(fc.facet).Field(fc.field).
				Limit(-1).AddToFacet("productCount", "uniqueBlock(_root_)")
			facets = append(facets, termsFacet)

			continue
		}

		// sku filters
		if len(tagVals) > 0 {
			skuFilters = append(skuFilters, solr.NewStandardQueryParser().Tag(fc.field).
				Query("'"+strings.Join(tagVals, " OR ")+"'").BuildParser())
		} else {
			skuFilters = append(skuFilters, solr.NewStandardQueryParser().Tag(fc.field).
				Query(fc.field+":*").BuildParser())
		}

		termsFacet := solr.NewTermsFacet(fc.facet).
			Field(fc.field).Limit(-1).
			AddToDomain("excludeTags", "top").
			AddToDomain("filter", []string{
				solr.NewFiltersQueryParser().Param("$skuFilters").
					ExcludeTags(fc.field).Query("$sku").BuildParser(),
				solr.NewChildrenQueryParser().Of("$product").Filters("$filter").
					Query("$product").BuildParser(),
			}).
			AddToFacet("productCount", "uniqueBlock(_root_)")

		facets = append(facets, termsFacet)
	}

	queryStr := r.URL.Query().Get("q")
	if len(queryStr) == 0 {
		queryStr = "*"
	} else {
		queryStr = fmt.Sprintf("%q", queryStr)
	}

	productFilters = append(productFilters, solr.NewStandardQueryParser().
		Tag("top").Query("_text_:"+queryStr).BuildParser())

	queryParser := solr.NewParentQueryParser().Tag("top").Filters("$skuFilters").
		Which("$product").Score("total").Query("$sku")
	query := solr.NewQuery(queryParser.BuildParser()).
		Filters(productFilters...).
		Facets(facets...).
		Queries(solr.M{
			"product":    "docType:product",
			"sku":        "docType:sku",
			"skuFilters": skuFilters,
		})

	queryResp, err := h.solrClient.Query(r.Context(), h.collection, query)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// extract products and facets from query response
	resp, err := buildResponse(queryResp)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	debug := r.URL.Query().Get("debug") == "true"
	if debug {
		resp["query"] = query
	}

	err = json.NewEncoder(w).Encode(resp)
	if err != nil {
		http.Error(w, err.Error(), 500)
	}
}

func buildResponse(response *solr.QueryResponse) (solr.M, error) {
	var facets = []solr.M{}
	for name, v := range response.Facets {
		if name == "count" {
			continue
		}

		vv, ok := v.(map[string]interface{})
		if !ok {
			return nil, errors.New("v is not map[string]interface{}")
		}

		bucks, ok := vv["buckets"].([]interface{})
		if !ok {
			return nil, errors.New("vv is not []interface{}")
		}

		buckets := []solr.M{}
		for _, bk := range bucks {
			buck, ok := bk.(map[string]interface{})
			if !ok {
				return nil, errors.New("bucket is not map[string]interface{}")
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

			buckets = append(buckets, solr.M{
				"val":          val,
				"skuCount":     int(skuCount),
				"productCount": int(productCount),
			})

		}

		var param string
		for _, fc := range facetConfigs {
			if fc.facet == name {
				param = fc.urlParam
			}
		}

		facets = append(facets, solr.M{
			"name":    name,
			"param":   param,
			"buckets": buckets,
		})
	}

	var products = []solr.M{}
	for _, doc := range response.Response.Documents {
		id, ok := doc["id"].(string)
		if !ok {
			return nil, errors.New("id not found or is not string")
		}

		name, ok := doc["name"].([]interface{})
		if !ok {
			return nil, errors.New("name not found or is not []interface{}")
		}

		category, ok := doc["category"].([]interface{})
		if !ok {
			return nil, errors.New("category not found or is not []interface{}")
		}

		brand, ok := doc["brand"].([]interface{})
		if !ok {
			return nil, errors.New("brand not found or is not []interface{}")
		}

		productType, ok := doc["productType"].(string)
		if !ok {
			return nil, errors.New("productType not found or is not string")
		}

		products = append(products, solr.M{
			"id":          id,
			"name":        name[0].(string),
			"category":    category[0].(string),
			"brand":       brand[0].(string),
			"productType": productType,
		})
	}

	return solr.M{
		"products": products,
		"facets":   facets,
	}, nil
}
