package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/somepgs/url-shortener/internal/service"
)

type URLHandler struct {
	service *service.UrlService
	baseURL string
}

type ShortenRequest struct {
	URL string `json:"url"`
}

type ShortenResponse struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

func NewURLHandler(service *service.UrlService, baseURL string) *URLHandler {
	return &URLHandler{service: service, baseURL: baseURL}
}

func (h *URLHandler) Shorten(w http.ResponseWriter, r *http.Request) {
	var req ShortenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request")
		return
	}
	if req.URL == "" {
		writeError(w, http.StatusBadRequest, "URL is required")
		return
	}

	u, err := h.service.ShortenURL(r.Context(), req.URL)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	resp := ShortenResponse{
		ShortURL:    h.baseURL + "/" + u.ShortCode,
		OriginalURL: u.OriginalURL,
	}
	writeJSON(w, http.StatusCreated, resp)
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

type errorResponse struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	err := json.NewEncoder(w).Encode(v)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, errorResponse{Error: msg})
}
