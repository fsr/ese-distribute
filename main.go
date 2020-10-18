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
    "time"
    "github.com/segmentio/ksuid"
)

type State struct {
    M sync.Mutex
    Rooms map[string]int
    WaitingClients []string
    Clients map[string]*client
}

type client struct {
    ID         string
    LastAccess time.Time
    Link       string
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

func giveSlotToClient(url string) {
    var waitingClient *client = nil
    for waitingClient == nil {
        if len(state.WaitingClients) == 0 {
            return
        }

        uuid := state.WaitingClients[0]
        state.WaitingClients = state.WaitingClients[1:]
        waitingClient = state.Clients[uuid]
    }
    log.Printf("room %s reserved for %s", url, waitingClient.ID)
    waitingClient.Link = url
    state.Rooms[url] -= 1
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
    giveSlotToClient(url)
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

func handleRegisterRoom(w http.ResponseWriter, r *http.Request) {
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

func cleanUp() {
    for uuid, c := range state.Clients {
        now := time.Now()
        timeout, _ := time.ParseDuration("10s")
        if now.Sub(c.LastAccess) > timeout {
            log.Printf("Client %s too old", uuid)
            // delete from client list
            delete(state.Clients, uuid)
            // free slot for room
            if c.Link != "" {
                state.Rooms[c.Link] += 1
            }
        }
    }
}

func findRoom(uuid string) string {
    cleanUp()

    var roomBest string
    currentClient := state.Clients[uuid]
    // uuid registered?
    if currentClient == nil {
        log.Printf("uuid %s not registered", uuid)
        return "nouuid"
    }
    currentClient.LastAccess = time.Now()
    // check if there is a reserved room
    if currentClient.Link != "" {
        log.Printf("room %s given to %s", currentClient.Link, currentClient.ID)
        delete(state.Clients, uuid)
        return currentClient.Link
    }
    // search for room with most available spots
    freeCountBest := 0
    for room, freeCount := range(state.Rooms) {
        if freeCount < freeCountBest {
            continue
        }

        freeCountBest = freeCount
        roomBest = room
    }

    if freeCountBest == 0 {
        return "wait"
    }
    state.Rooms[roomBest] -= 1;

    log.Printf("room %s [%d] given to %s", roomBest, freeCountBest, currentClient.ID)
    delete(state.Clients, uuid)
    return roomBest


}

func handleRegisterClient(w http.ResponseWriter, r *http.Request) {
    state.M.Lock()
    defer state.M.Unlock()

    uuid := ksuid.New().String()
    log.Printf("uuid registered: %s", uuid)
    currentClient := client{ID: uuid, LastAccess: time.Now(), Link: ""}
    state.Clients[uuid] = &currentClient
    state.WaitingClients = append(state.WaitingClients, uuid)
    fmt.Fprintf(w, uuid)
}

func handlePoll(w http.ResponseWriter, r *http.Request) {
    state.M.Lock()
    defer state.M.Unlock()

    room := findRoom(r.FormValue("uuid"))

    fmt.Fprintf(w, "%s", room)

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
    state.Clients = make(map[string]*client)

    http.Handle("/", http.FileServer(http.Dir("static")))
    http.HandleFunc("/api/poll", handlePoll)
    http.HandleFunc("/api/free", handleFree)
    http.HandleFunc("/api/state", handleState)
    http.HandleFunc("/api/delete", handleDelete)
    http.HandleFunc("/api/register", handleRegisterRoom)
    http.HandleFunc("/api/register_client", handleRegisterClient)
    log.Printf("listening on %s", config.Addr)
    http.ListenAndServe(config.Addr, nil)
}
