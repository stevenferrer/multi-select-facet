package main

import (
	"encoding/json"
	"net/http"

	"github.com/sf9v/solr-go"
	"github.com/sf9v/solr-go/suggester"
)

type suggestHandler struct {
	collection string
	solrClient solr.Client
}

func (h *suggestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")

	if len(q) == 0 {
		return
	}

	dict := "default"
	suggestResp, err := h.solrClient.Suggester().Suggest(r.Context(), h.collection,
		suggester.Params{Query: q, Dictionaries: []string{dict}})
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	suggest := *suggestResp.Suggest
	termBody := suggest[dict][q]

	suggestions := []struct {
		Term string `json:"term"`
	}{}
	for _, suggest := range termBody.Suggestions {
		suggestions = append(suggestions, struct {
			Term string `json:"term"`
		}{Term: suggest.Term})
	}

	resp := Map{
		"numFound":    termBody.NumFound,
		"suggestions": suggestions,
	}

	w.Header().Add("content-type", "application/json")

	err = json.NewEncoder(w).Encode(resp)
	if err != nil {
		http.Error(w, err.Error(), 500)
	}
}
