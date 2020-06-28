package main

import (
	"context"
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
	"github.com/pkg/errors"
	"github.com/stevenferrer/solr-go"
	solrindex "github.com/stevenferrer/solr-go/index"
	solrschema "github.com/stevenferrer/solr-go/schema"
)

// Any is a convenience type for interface{}
type Any = interface{}

// Map is a convenience type for map[string]interface{}
type Map = map[string]Any

const (
	dataPath = "products.json"
	coll     = "multi-select-facet-demo"
)

func main() {
	collection := flag.String("collection", coll, "specify the name of collection")
	initSchema := flag.Bool("initialize-schema", false, "initialize solr schema")
	index := flag.Bool("index-data", false, "index the data")
	flag.Parse()

	solrClient := solr.NewClient("localhost", 8983)

	ctx := context.Background()
	if *initSchema {
		log.Print("initializing solr schema...")
		err := initSolrSchema(ctx, *collection, solrClient.Schema())
		if err != nil {
			log.Fatal(err)
		}
	}

	if *index {
		log.Println("indexing products...")
		err := indexProducts(ctx, *collection, solrClient.Index())
		if err != nil {
			log.Fatal(err)
		}
	}

	r := chi.NewRouter()

	// basic middlewares
	r.Use(middleware.Logger)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// search handler
	r.Method(http.MethodGet, "/search", &searchHandler{
		collection: *collection,
		solrClient: solrClient,
	})

	r.Method(http.MethodGet, "/suggest", &suggestHandler{
		collection: *collection,
		solrClient: solrClient,
	})

	addr := ":8081"
	log.Printf("listening on %s\n", addr)
	err := http.ListenAndServe(addr, r)
	if err != nil {
		log.Fatal(err)
	}
}

func initSolrSchema(ctx context.Context, collection string,
	schemaClient solrschema.Client) (err error) {
	// auto-suggest field type
	fieldTypes := []solrschema.FieldType{

		// // approach #1
		// // see: https://blog.griddynamics.com/implementing-autocomplete-with-solr/
		// {
		// 	Name:                 "text_suggest",
		// 	Class:                "solr.TextField",
		// 	PositionIncrementGap: "100",
		// 	IndexAnalyzer: &solrschema.Analyzer{
		// 		Tokenizer: &solrschema.Tokenizer{
		// 			Class: "solr.StandardTokenizerFactory",
		// 		},
		// 		Filters: []solrschema.Filter{
		// 			{
		// 				Class: "solr.LowerCaseFilterFactory",
		// 			},
		// 			{
		// 				Class:       "solr.EdgeNGramFilterFactory",
		// 				MinGramSize: 1,
		// 				MaxGramSize: 100,
		// 			},
		// 		},
		// 	},
		// 	QueryAnalyzer: &solrschema.Analyzer{
		// 		Tokenizer: &solrschema.Tokenizer{
		// 			Class: "solr.KeywordTokenizerFactory",
		// 		},
		// 		Filters: []solrschema.Filter{
		// 			{
		// 				Class: "solr.LowerCaseFilterFactory",
		// 			},
		// 		},
		// 	},
		// },

		// approach #2
		// see: https://blog.griddynamics.com/implement-autocomplete-search-for-large-e-commerce-catalogs/
		{
			Name:   "text_suggest",
			Class:  "solr.TextField",
			Stored: true,
			IndexAnalyzer: &solrschema.Analyzer{
				Tokenizer: &solrschema.Tokenizer{
					Class: "solr.WhitespaceTokenizerFactory",
				},
				Filters: []solrschema.Filter{
					{
						Class: "solr.LowerCaseFilterFactory",
					},
					{
						Class: "solr.ASCIIFoldingFilterFactory",
					},
				},
			},
			QueryAnalyzer: &solrschema.Analyzer{
				Tokenizer: &solrschema.Tokenizer{
					Class: "solr.WhitespaceTokenizerFactory",
				},
				Filters: []solrschema.Filter{
					{
						Class: "solr.LowerCaseFilterFactory",
					},
					{
						Class: "solr.ASCIIFoldingFilterFactory",
					},
					{
						Class:    "solr.SynonymGraphFilterFactory",
						Synonyms: "synonyms.txt",
					},
				},
			},
		},
	}

	for _, fieldType := range fieldTypes {
		err = schemaClient.AddFieldType(ctx, collection, fieldType)
		if err != nil {
			return errors.Wrap(err, "add field type")
		}
	}

	// define the fields
	fields := []solrschema.Field{
		{
			Name:    "docType",
			Type:    "string",
			Indexed: true,
			Stored:  true,
		},
		{
			Name:    "name",
			Type:    "text_general",
			Indexed: true,
			Stored:  true,
		},
		{
			Name:    "category",
			Type:    "text_gen_sort",
			Indexed: true,
			Stored:  true,
		},
		{
			Name:    "brand",
			Type:    "text_gen_sort",
			Indexed: true,
			Stored:  true,
		},
		{
			Name:    "productType",
			Type:    "string",
			Indexed: true,
			Stored:  true,
		},
		{
			Name:        "suggest",
			Type:        "text_suggest",
			Stored:      true,
			MultiValued: true,
		},
	}

	for _, field := range fields {
		err = schemaClient.AddField(ctx, collection, field)
		if err != nil {
			return errors.Wrap(err, "add field")
		}
	}

	copyFields := []solrschema.CopyField{
		{
			Source: "name",
			Dest:   "suggest",
		},
		{
			Source: "category",
			Dest:   "suggest",
		},
		{
			Source: "brand",
			Dest:   "suggest",
		},
		{
			Source: "productType",
			Dest:   "suggest",
		},
		{
			Source: "*_s",
			Dest:   "suggest",
		},
	}

	for _, copyField := range copyFields {
		err = schemaClient.AddCopyField(ctx, collection, copyField)
		if err != nil {
			return errors.Wrap(err, "add copy field")
		}
	}

	return nil
}

func indexProducts(ctx context.Context, collection string,
	indexClient solrindex.JSONClient) error {
	b, err := ioutil.ReadFile(dataPath)
	if err != nil {
		return err
	}

	var products []Map
	err = json.Unmarshal(b, &products)
	if err != nil {
		return err
	}

	docs := solrindex.NewDocs()
	for _, product := range products {
		docs.AddDoc(product)
	}

	err = indexClient.AddDocuments(ctx, collection, docs)
	if err != nil {
		return err
	}

	// commit updates
	err = indexClient.Commit(ctx, collection)
	if err != nil {
		return err
	}

	return nil
}
