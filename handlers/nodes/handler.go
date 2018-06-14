package nodes

import (
	"encoding/json"
	"io/ioutil"
	"kinetik-server/control"
	"kinetik-server/data"
	"kinetik-server/logger"
	"kinetik-server/models"
	"kinetik-server/rand"
	"net/http"
	"os"

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

func StartDocker(w http.ResponseWriter, r *http.Request) {
	body, _ := ioutil.ReadAll(r.Body)
	err := control.StartDocker(string(body))
	if err != nil {
		http.Error(w, err.Error(), 500)
	} else {
		w.WriteHeader(200)
	}
}

func CreateNode(w http.ResponseWriter, r *http.Request) {

	cfg, err := control.CreateKlerk(&control.KlerkCreationOption{
		Name:              "klerk-" + rand.String(4),
		Region:            "ams3",
		Size:              "s-1vcpu-1gb",
		Token:             os.Getenv("DO_TOKEN"),
		SSHKeyFingerprint: control.ComputePublicFingerprint("/root/.ssh/id_rsa"),
	})

	if err != nil {
		http.Error(w, err.Error(), 500)
	} else {
		w.WriteHeader(201)
		json.NewEncoder(w).Encode(cfg)
	}
}
