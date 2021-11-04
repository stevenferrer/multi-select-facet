package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/pkg/errors"

	solr "github.com/stevenferrer/solr-go"
)

const (
	dataPath          = "data/products.json"
	defaultCollection = "multi-select-facet-demo"
)

func main() {
	collection := flag.String("collection", defaultCollection, "specify the name of collection")
	createCollection := flag.Bool("create-collection", false, "create solr collection")
	initSchema := flag.Bool("init-schema", false, "initialize solr schema")
	initSuggester := flag.Bool("init-suggester", false, "initialize suggester component")
	indexData := flag.Bool("index-data", false, "index the products data")
	flag.Parse()

	baseURL := "http://localhost:8983"
	solrClient := solr.NewJSONClient(baseURL)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if *createCollection {
		log.Println("creating collection...")
		err := solrClient.CreateCollection(ctx, solr.NewCollectionParams().
			Name(*collection).NumShards(1).ReplicationFactor(1))
		if err != nil {
			log.Fatal(err)
		}
	}

	if *initSchema {
		log.Print("initializing solr schema...")
		err := initSolrSchema(ctx, *collection, solrClient)
		if err != nil {
			log.Fatal(err)
		}
	}

	if *indexData {
		log.Println("indexing products...")
		err := indexProducts(ctx, *collection, solrClient)
		if err != nil {
			log.Fatal(err)
		}
	}

	if *initSuggester {
		log.Println("initializing suggester component...")
		err := initSuggestConfig(ctx, *collection, solrClient)
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
	srvr := &http.Server{
		Addr:           addr,
		Handler:        r,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	// Start the server
	go func() {
		log.Printf("Server listening on %s\n", addr)
		if err := srvr.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()

	// setup signal capturing
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	// wait for SIGINT (pkill -2)
	<-c

	if err := srvr.Shutdown(ctx); err != nil {
		log.Fatal(err)
	}
}

func initSolrSchema(ctx context.Context, collection string, solrClient solr.Client) (err error) {
	// Auto-suggest field type
	fieldTypes := []solr.FieldType{
		// approach #1
		// Refer to https://blog.griddynamics.com/implementing-autocomplete-with-solr/
		{
			Name:                 "text_suggest_1",
			Class:                "solr.TextField",
			PositionIncrementGap: "100",
			IndexAnalyzer: &solr.Analyzer{
				Tokenizer: &solr.Tokenizer{
					Class: "solr.StandardTokenizerFactory",
				},
				Filters: []solr.Filter{
					{
						Class: "solr.LowerCaseFilterFactory",
					},
					{
						Class:       "solr.EdgeNGramFilterFactory",
						MinGramSize: 1,
						MaxGramSize: 100,
					},
				},
			},
			QueryAnalyzer: &solr.Analyzer{
				Tokenizer: &solr.Tokenizer{
					Class: "solr.KeywordTokenizerFactory",
				},
				Filters: []solr.Filter{
					{
						Class: "solr.LowerCaseFilterFactory",
					},
				},
			},
		},

		// approach #2
		// Refer to https://blog.griddynamics.com/implement-autocomplete-search-for-large-e-commerce-catalogs/
		{
			Name:                 "text_suggest",
			Class:                "solr.TextField",
			PositionIncrementGap: "100",
			IndexAnalyzer: &solr.Analyzer{
				Tokenizer: &solr.Tokenizer{
					Class: "solr.WhitespaceTokenizerFactory",
				},
				Filters: []solr.Filter{
					{
						Class: "solr.LowerCaseFilterFactory",
					},
					{
						Class: "solr.ASCIIFoldingFilterFactory",
					},
					{
						Class:       "solr.EdgeNGramFilterFactory",
						MinGramSize: 1,
						MaxGramSize: 20,
					},
				},
			},
			QueryAnalyzer: &solr.Analyzer{
				Tokenizer: &solr.Tokenizer{
					Class: "solr.WhitespaceTokenizerFactory",
				},
				Filters: []solr.Filter{
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

	err = solrClient.AddFieldTypes(ctx, collection, fieldTypes...)
	if err != nil {
		return errors.Wrap(err, "add field types")
	}

	// define the fields
	fields := []solr.Field{
		{
			Name: "docType",
			Type: "string",
		},
		{
			Name: "name",
			Type: "text_general",
		},
		{
			Name: "category",
			Type: "text_gen_sort",
		},
		{
			Name: "brand",
			Type: "text_gen_sort",
		},
		{
			Name: "productType",
			Type: "string",
		},
		{
			Name:        "suggest",
			Type:        "text_suggest",
			Stored:      false,
			MultiValued: true,
		},
	}

	err = solrClient.AddFields(ctx, collection, fields...)
	if err != nil {
		return errors.Wrap(err, "add fields")
	}

	copyFields := []solr.CopyField{
		{
			Source: "name",
			Dest:   "suggest",
		},

		{
			Source: "name",
			Dest:   "_text_",
		},
		{
			Source: "category",
			Dest:   "_text_",
		},
		{
			Source: "brand",
			Dest:   "_text_",
		},
		{
			Source: "productType",
			Dest:   "_text_",
		},
		{
			Source: "*_s",
			Dest:   "_text_",
		},
	}

	err = solrClient.AddCopyFields(ctx, collection, copyFields...)
	if err != nil {
		return errors.Wrap(err, "add copy fields")
	}

	return nil
}

func initSuggestConfig(ctx context.Context, collection string, solrClient solr.Client) error {
	// suggester configs
	suggestComponent := solr.NewComponent(solr.SearchComponent).
		Name("suggest").Class("solr.SuggestComponent").
		Config(solr.M{
			"suggester": solr.M{
				"name":                     "default",
				"lookupImpl":               "AnalyzingInfixLookupFactory",
				"dictionaryImpl":           "DocumentDictionaryFactory",
				"field":                    "suggest",
				"suggestAnalyzerFieldType": "text_suggest",
			},
		})

	suggestHandler := solr.NewComponent(solr.RequestHandler).
		Name("/suggest").Class("solr.SearchHandler").
		Config(solr.M{
			"startup": "lazy",
			"defaults": solr.M{
				"suggest":            true,
				"suggest.count":      10,
				"suggest.dictionary": "default",
			},
			"components": []string{"suggest"},
		})

	err := solrClient.AddComponents(ctx, collection,
		suggestComponent, suggestHandler)
	if err != nil {
		return errors.Wrap(err, "add suggester configs")
	}

	return nil
}

func indexProducts(ctx context.Context, collection string,
	solrClient solr.Client) error {
	f, err := os.OpenFile(dataPath, os.O_RDONLY, 0644)
	if err != nil {
		return err
	}

	_, err = solrClient.Update(ctx, collection, solr.JSON, f)
	if err != nil {
		return err
	}

	// commit updates
	err = solrClient.Commit(ctx, collection)
	if err != nil {
		return err
	}

	return nil
}
