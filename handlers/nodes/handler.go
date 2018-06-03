package nodes

import (
	"encoding/json"
	"kinetik-server/data"
	"kinetik-server/logger"
	"kinetik-server/models"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/docker/cli/cli/compose/types"
)

func GetNodes(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(data.GetDB().GetNodes())
}

func UpdateNode(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nodeIP := vars["id"]

	logger.StdLog.Println("Got a new node request")

	var nodeReport models.Node

	err := json.NewDecoder(r.Body).Decode(&nodeReport)

	if err == nil {

		previousState := data.GetDB().GetNode(nodeIP)
		if previousState != nil {
			nodeReport.Reservations = previousState.Reservations
		} else {
			nodeReport.Reservations = &types.Resource{}
		}

		data.GetDB().AddNode(nodeIP, &nodeReport)
		logger.StdLog.Println("Added node " + nodeIP)
	} else {
		logger.ErrLog.Println("Error while getting node : " + err.Error())
	}
}
