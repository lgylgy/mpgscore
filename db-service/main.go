package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

type Player struct {
	ID     string   `json:"id"`
	Name   string   `json:"name"`
	Team   string   `json:"team"`
	Grades []string `json:"grades"`
}

var controller *Controller

func main() {

	controller = NewController()
	err := controller.Connect("$MONGODB", "$MONGODB", "$MONGODB")
	if err != nil {
		log.Fatal(err)
	}
	defer controller.Close()

	registerHandlers()
}

func registerHandlers() {
	routes := mux.NewRouter()
	routes.Handle("/", http.RedirectHandler("/players", http.StatusFound))
	routes.Methods("GET").Path("/players").
		Handler(handler(listPlayers))
	routes.Methods("GET").Path("/players/{id}").
		Handler(handler(getPlayer))
	routes.Methods("GET").Path("/teams/{id}").
		Handler(handler(listTeamPlayers))
	routes.Methods("PUT").Path("/players/{id}").
		Handler(handler(updatePlayer))
	routes.Methods("POST").Path("/players").
		Handler(handler(createPlayer))

	log.Fatal(http.ListenAndServe(":8080", routes))
}

func createPlayer(w http.ResponseWriter, r *http.Request) (interface{}, *mpgError) {
	defer r.Body.Close()

	var player Player
	if err := json.NewDecoder(r.Body).Decode(&player); err != nil {
		return nil, mpgErrorf(err, "invalid request payload: %v", err)
	}
	_, err := controller.AddPlayer(&player)
	if err != nil {
		return nil, mpgErrorf(err, "could not save book: %v", err)
	}
	return player, nil
}

func listPlayers(w http.ResponseWriter, r *http.Request) (interface{}, *mpgError) {
	players, err := controller.ListPlayers()
	if err != nil {
		return nil, mpgErrorf(err, "could not list players: %v", err)
	}
	return players, nil
}

func listTeamPlayers(w http.ResponseWriter, r *http.Request) (interface{}, *mpgError) {
	params := mux.Vars(r)
	players, err := controller.ListTeamPlayers(params["id"])
	if err != nil {
		return nil, mpgErrorf(err, "could not list players: %v", err)
	}
	return players, nil
}

func getPlayer(w http.ResponseWriter, r *http.Request) (interface{}, *mpgError) {
	params := mux.Vars(r)
	player, err := controller.GetPlayer(params["id"])
	if err != nil {
		return nil, mpgErrorf(err, "could not find player: %v", err)
	}
	return player, nil
}

func updatePlayer(w http.ResponseWriter, r *http.Request) (interface{}, *mpgError) {
	var player Player
	if err := json.NewDecoder(r.Body).Decode(&player); err != nil {
		return nil, mpgErrorf(err, "invalid request payload: %v", err)
	}
	params := mux.Vars(r)
	player.ID = params["id"]
	updated, err := controller.UpdatePlayer(&player)
	if err != nil {
		return nil, mpgErrorf(err, "could not update player: %v", err)
	}
	return updated, nil
}

type handler func(http.ResponseWriter, *http.Request) (interface{}, *mpgError)

type mpgError struct {
	Error   error
	Message string
	Code    int
}

func (h handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	result, e := h(w, r)
	if e != nil {
		log.Printf("Handler error: status code: %d, message: %s, underlying err: %#v",
			e.Code, e.Message, e.Error)
		http.Error(w, e.Message, e.Code)
		return
	}
	resultJson, err := json.Marshal(result)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
	w.Write(resultJson)
}

func mpgErrorf(err error, format string, v ...interface{}) *mpgError {
	return &mpgError{
		Error:   err,
		Message: fmt.Sprintf(format, v...),
		Code:    http.StatusInternalServerError,
	}
}