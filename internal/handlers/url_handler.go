package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/somepgs/url-shortener/internal/service"
)

type URLHandler struct {
	service *service.UrlService
}

type ShortenRequest struct {
	URL string `json:"url"`
}

type ShortenResponse struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

func NewURLHandler(service *service.UrlService) *URLHandler {
	return &URLHandler{service: service}
}

func (h *URLHandler) Shorten(w http.ResponseWriter, r *http.Request) {
	var req ShortenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.URL == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}

	url, err := h.service.ShortenURL(r.Context(), req.URL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := ShortenResponse{
		ShortURL:    "http://localhost:8080/" + url.ShortCode,
		OriginalURL: url.OriginalURL,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *URLHandler) Redirect(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	shortCode := vars["code"]

	originalURL, err := h.service.GetAndRedirect(r.Context(), shortCode)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	http.Redirect(w, r, originalURL, http.StatusTemporaryRedirect)
}
