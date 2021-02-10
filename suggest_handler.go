package main

import (
	"encoding/json"
	"net/http"

	"github.com/sf9v/solr-go"
)

type suggestHandler struct {
	collection string
	solrClient solr.Client
}

type suggestion struct {
	Term string `json:"term"`
}

func (h *suggestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")

	if len(q) == 0 {
		return
	}

	dict := "default"
	suggestParams := solr.NewSuggesterParams("suggest").
		Build().Query(q).Dictionaries(dict)
	suggestResp, err := h.solrClient.Suggest(r.Context(), h.collection, suggestParams)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	suggest := *suggestResp.Suggest
	termBody := suggest[dict][q]

	suggestions := []suggestion{}
	for _, suggest := range termBody.Suggestions {
		suggestions = append(suggestions, suggestion{
			Term: suggest.Term,
		})
	}

	err = json.NewEncoder(w).Encode(solr.M{
		"numFound":    termBody.NumFound,
		"suggestions": suggestions,
	})
	if err != nil {
		http.Error(w, err.Error(), 500)
	}
}
