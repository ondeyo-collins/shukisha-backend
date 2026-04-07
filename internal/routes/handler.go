package routes

import (
	"encoding/json"
	"net/http"
	"strconv"
)

type RouteHandler struct {
	service *RouteService
}

func NewRouteHandler(service *RouteService) *RouteHandler {
	return &RouteHandler{service: service}
}

func (h *RouteHandler) SearchRoutes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query().Get("query")
	routes, err := h.service.SearchRoutes(r.Context(), query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(routes)
}

func (h *RouteHandler) GetRouteDetails(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Simple path param extraction for standard net/http
	// In a real app, use a router like chi or mux
	routeID := r.URL.Path[len("/v1/routes/"):]
	if routeID == "" {
		http.Error(w, "Route ID required", http.StatusBadRequest)
		return
	}

	route, err := h.service.GetRouteWithStops(r.Context(), routeID)
	if err != nil {
		http.Error(w, "Route not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(route)
}

func (h *RouteHandler) SearchNearestStops(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	lat, _ := strconv.ParseFloat(r.URL.Query().Get("lat"), 64)
	lng, _ := strconv.ParseFloat(r.URL.Query().Get("lng"), 64)
	radius, _ := strconv.Atoi(r.URL.Query().Get("radius"))
	if radius == 0 {
		radius = 1000 // Default 1km
	}

	stops, err := h.service.FindNearestStops(r.Context(), lat, lng, radius)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(stops)
}
