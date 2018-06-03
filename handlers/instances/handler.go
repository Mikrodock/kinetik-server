package instances

import (
	"encoding/json"
	"kinetik-server/data"
	"kinetik-server/models/internals"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

func GetInstances(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(data.GetDB().GetInstances())
}

func DeleteInstance(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id := params["id"]
	idInt, err := strconv.Atoi(id)
	err = data.GetDB().DeleteNode(idInt)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(internals.ErrorMessage{
			Message: err.Error(),
		})
	} else {
		w.WriteHeader(http.StatusOK)
	}
}

func UpdateMetrics(w http.ResponseWriter, r *http.Request) {

}
