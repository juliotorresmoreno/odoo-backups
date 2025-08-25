package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type ConfigureRequest struct {
	DBName string `json:"dbname"`
	Size   int    `json:"size"`
}

func (h *handler) configureHandler(w http.ResponseWriter, r *http.Request) {
	var req ConfigureRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.DBName == "" || req.Size <= 0 {
		http.Error(w, "Invalid request: DBName and Size must be provided", http.StatusBadRequest)
		return
	}

	err := h.storage.CreateLocalPVC(context.TODO(), req.DBName, req.Size, "juliotorres-pc")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = h.storage.ExecuteWithPVC(context.TODO(), req.DBName)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to execute with PVC: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("PVC created for database %s with size %dGi", req.DBName, req.Size)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"message": fmt.Sprintf("PVC created for database %s with size %dGi", req.DBName, req.Size),
	})
}
