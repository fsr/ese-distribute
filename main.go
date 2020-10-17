package main

import (
    "encoding/json"
    "fmt"
    "io/ioutil"
    "log"
    "net/http"
    "sync"
    "os"
    "strconv"
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

    count, err := strconv.Atoi(r.FormValue("count"))
    if err != nil || count <= 0 {
        w.WriteHeader(http.StatusInternalServerError)
        return
    }

    state.Rooms[url] += count;
    log.Printf("free %s", r.FormValue("url"))
    data, _ := json.Marshal(state.Rooms)
    w.Write(data)
    persistState()
}

func handleDelete(w http.ResponseWriter, r *http.Request) {
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

    delete(state.Rooms, url)

    log.Printf("delete %s", r.FormValue("url"))
    data, _ := json.Marshal(state.Rooms)
    w.Write(data)
    persistState()
}

func handleRegister(w http.ResponseWriter, r *http.Request) {
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

    count, err := strconv.Atoi(r.FormValue("count"))
    if err != nil || count <= 0 {
        w.WriteHeader(http.StatusInternalServerError)
        return
    }

    _, exists := state.Rooms[url]
    if exists {
        w.WriteHeader(http.StatusConflict)
        return
    }
    state.Rooms[url] = count;

    log.Printf("register %s", r.FormValue("url"))
    data, _ := json.Marshal(state.Rooms)
    w.Write(data)
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

    log.Printf("offer %s", roomBest)
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
    http.HandleFunc("/api/delete", handleDelete)
    http.HandleFunc("/api/register", handleRegister)
    log.Printf("listening on %s", config.Addr)
    http.ListenAndServe(config.Addr, nil)
}
