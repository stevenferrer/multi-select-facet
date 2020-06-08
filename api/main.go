package main

import (
	"context"
	"log"

	"github.com/davecgh/go-spew/spew"
	"github.com/stevenferrer/solr-go"
	. "github.com/stevenferrer/solr-go/types"
)

type Facet struct {
	Name    string
	Buckets []Bucket
}

type Bucket struct {
	Val          string
	SkuCount     int
	ProductCount int
}

type Product struct {
	ID    string
	Title string
}

func main() {
	sample()
}

func sample() {
	client := solr.NewClient("localhost", 8983)

	resp, err := client.Query().Query(context.Background(), "merka-products", M{
		"queries": M{
			"child.query": "docType:sku",
			"child.fq": []string{
				"{!tag=color}color_s:Black OR color_s:Red",
				"{!tag=size}size_s:Large OR size_s:Medium",
			},
			"parent.fq": "docType:product",
		},
		"filter": []string{
			"{!tag=top df=category v=Demo}",
		},
		"fields": "id,title,score",
		"query":  "{!parent tag=top filters=$child.fq which=$parent.fq score=total v=$child.query}",
		"facet": M{
			"colors": M{
				"domain": M{
					"excludeTags": "top",
					"filter": []string{
						"{!filters param=$child.fq excludeTags=color v=$child.query}",
						"{!child of=$parent.fq filters=$fq v=$parent.fq}",
					},
				},
				"type":  "terms",
				"field": "color_s",
				"limit": -1,
				"facet": M{
					"productCount": "uniqueBlock(_root_)",
				},
			},
			"sizes": M{
				"domain": M{
					"excludeTags": "top",
					"filter": []string{
						"{!filters param=$child.fq excludeTags=size v=$child.query}",
						"{!child of=$parent.fq filters=$fq v=$parent.fq}",
					},
				},
				"type":  "terms",
				"field": "size_s",
				"limit": -1,
				"facet": M{
					"productCount": "uniqueBlock(_root_)",
				},
			},
			"categories": M{
				"type":  "terms",
				"field": "category",
				"limit": -1,
				"facet": M{
					"productCount": "uniqueBlock(_root_)",
				},
			},
			"brands": M{
				"type":  "terms",
				"field": "brand",
				"limit": -1,
				"facet": M{
					"productCount": "uniqueBlock(_root_)",
				},
			},
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	var facets = []Facet{}
	for k, v := range resp.Facets {
		if k == "count" {
			continue
		}

		vv, ok := v.(map[string]interface{})
		if !ok {
			log.Fatal("buckets is not map[string]interface{}")
		}

		bucks, ok := vv["buckets"].([]interface{})
		if !ok {
			log.Fatal("buckets not found")
		}

		buckets := []Bucket{}
		for _, bk := range bucks {
			buck, ok := bk.(map[string]interface{})
			if !ok {
				log.Fatal("bk not map[string]interface{}")
			}

			productCount, ok := buck["productCount"].(float64)
			if !ok {
				log.Fatal("product count not found")
			}

			skuCount, ok := buck["count"].(float64)
			if !ok {
				log.Fatal("sku count not found")
			}

			val, ok := buck["val"].(string)
			if !ok {
				log.Fatal("val not found")
			}

			buckets = append(buckets, Bucket{
				Val:          val,
				SkuCount:     int(skuCount),
				ProductCount: int(productCount),
			})

		}

		facets = append(facets, Facet{
			Name:    k,
			Buckets: buckets,
		})
	}

	var products = []Product{}
	for _, doc := range resp.Response.Docs {
		id, ok := doc["id"].(string)
		if !ok {
			log.Fatal("id not found")
		}

		title, ok := doc["title"].([]interface{})
		if !ok {
			log.Fatal("title note found")
		}

		products = append(products, Product{
			ID:    id,
			Title: title[0].(string),
		})
	}

	spew.Dump(facets)
	spew.Dump(products)
}
