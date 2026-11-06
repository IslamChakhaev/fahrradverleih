package handler

import "net/http"

type bikeRequest struct {
	Name string `json:"name"`
}

func (h *Handler) listBikes(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.store.Bikes())
}

func (h *Handler) createBike(w http.ResponseWriter, r *http.Request) {
	var req bikeRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, err)
		return
	}
	bike, err := h.store.AddBike(req.Name)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, bike)
}

func (h *Handler) updateBike(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	var req bikeRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, err)
		return
	}
	bike, err := h.store.UpdateBike(id, req.Name)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, bike)
}
