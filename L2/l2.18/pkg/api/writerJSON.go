package api

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// WriterJSON служит для формирования и направления ответа в формате JSON
func WriterJSON(w http.ResponseWriter, status int, data interface{}) {

	emergencyError := `{"fatal error":"%q"}`

	js, err := json.Marshal(data)
	if err != nil {
		// отправка при ошибке
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprintf(w, emergencyError, err)
		return
	}

	// нормальная отправка
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(js)
}
