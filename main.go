package main

import (
    "encoding/json"
    "fmt"
    "io/ioutil"
    "log"
    "net/http"
    "sync"
    "os"
)

type State struct {
    M sync.Mutex
    Rooms map[string]int
}

type Config struct {
    Key string
    Addr string
    StateFile string
}

var state State
var config Config

func persistState() {
    data, _ := json.Marshal(state.Rooms)
    err := ioutil.WriteFile(config.StateFile, data, 0644)
    if err != nil {
        log.Printf("error writing state: %v", err)
    }
}


func handleFree(w http.ResponseWriter, r *http.Request) {
    state.M.Lock()
    defer state.M.Unlock()

    keys, present := r.URL.Query()["key"]
    if !present || len(keys) != 1 || keys[0] != config.Key {
        w.WriteHeader(http.StatusUnauthorized)
        return
    }

    url := r.FormValue("url")
    if url == "" {
        w.WriteHeader(http.StatusInternalServerError)
        return
    }

    state.Rooms[url] += 1;
    log.Printf("free %s", r.FormValue("url"))
    persistState()
}

func handlePoll(w http.ResponseWriter, r *http.Request) {
    state.M.Lock()
    defer state.M.Unlock()

    // search for room with most available spots
    var roomBest string
    freeCountBest := 0
    for room, freeCount := range(state.Rooms) {
        if freeCount < freeCountBest {
            continue
        }

        freeCountBest = freeCount
        roomBest = room
    }

    if freeCountBest == 0 {
        fmt.Fprintf(w, "wait")
        return
    }

    state.Rooms[roomBest] -= 1;
    fmt.Fprintf(w, "%s", roomBest)
    persistState()
}

func handleState(w http.ResponseWriter, r *http.Request) {
    state.M.Lock()
    defer state.M.Unlock()

    data, _ := json.Marshal(state.Rooms)
    w.Write(data)
}

func main() {
    configFile := "config.json"
    if len(os.Args) > 1 {
        configFile = os.Args[1]
    }

    // load config
    data, err := ioutil.ReadFile(configFile)
    if err != nil {
        panic(err)
    }
    json.Unmarshal(data, &config)

    // load state
    data, err = ioutil.ReadFile(config.StateFile)
    if err != nil && !os.IsNotExist(err) {
        panic(err)
    }

    if data != nil {
        json.Unmarshal(data, &state.Rooms)
    } else {
        state.Rooms = make(map[string]int)
    }

    http.Handle("/", http.FileServer(http.Dir("static")))
    http.HandleFunc("/api/poll", handlePoll)
    http.HandleFunc("/api/free", handleFree)
    http.HandleFunc("/api/state", handleState)
    log.Printf("listening on %s", config.Addr)
    http.ListenAndServe(config.Addr, nil)
}
