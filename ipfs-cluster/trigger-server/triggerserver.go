package main

import (
    "log"
    "net/http"
    "os/exec"
    "time"
    "fmt"
    "io/ioutil"
    "math/rand"
    "strconv"
    "os"
    "github.com/ipfs/go-ipfs-api"
    "bytes"
)

const configPath = "/etc/trigger-server"
// port is HTTP API listen port
const port = 8082
var runnerPath = os.Getenv("GOPATH") + "src/github.com/ipfs/kubernetes-ipfs/ipfs-cluster/"
// Runner takes two args
const runnerNumNodes = "5"
const runnerNumPins = "5"

// Secret is set by main
var secret = ""

// IPFS Shell
var sh = shell.NewShell("localhost:5001")

func run(rw http.ResponseWriter, req *http.Request) {
    if req.Method == "POST" {
	if req.Header.Get("X-Auth") != secret {
		log.Println("User failed authentication")
		fmt.Fprintf(rw, "Invalid authentication\n")
	} else {
		// This command takes a while, so we'll hang out here while it runs
		cmd := exec.Command("bash", "runner.sh", runnerNumNodes, runnerNumPins)
		cmd.Dir = runnerPath
		out, err := cmd.CombinedOutput()
		// Add the output to IPFS
		addr, ipfsErr := sh.Add(bytes.NewReader(out))
		// Make it an IPFS link
		addr = "/ipfs/" + addr
		// Send the link to the client
		fmt.Fprintf(rw, addr)
		// Handle errors that may have occurred
		if err != nil {
			log.Println("Error from runner")
			log.Println(err)
		}
		if ipfsErr != nil {
			log.Println("IPFS Error adding output")
			log.Println(ipfsErr)
		}
	}
    } else {
	// For non-POST requests, tell the user what we are.
        fmt.Fprintf(rw, `Kubernetes-IPFS Trigger Server
	https://github.com/ipfs/kubernetes-ipfs/
	`)
    }
}

func main() {
    secretBytes, readErr := ioutil.ReadFile(configPath)
    if readErr != nil {
        log.Printf("Missing secret configuration in %s. We'll generate one for you...\n", configPath)
	rand.Seed(time.Now().UnixNano())
	secretBytes = randLetters(30)
	ioErr := ioutil.WriteFile(configPath, secretBytes, 0644)
	if ioErr != nil {
		log.Fatal("Unable to write new secret file; check permissions and try again")
	}
    }
    secret = string(secretBytes)
    log.Printf("Starting with secret as %s on port %d\n", secret, port)
    http.HandleFunc("/run", run)
    http.HandleFunc("/", func(rw http.ResponseWriter, req *http.Request) {
        fmt.Fprintf(rw, `Kubernetes-IPFS Trigger Server
	https://github.com/ipfs/kubernetes-ipfs/
	`)
    })
    log.Println(http.ListenAndServe(":" + strconv.FormatInt(port, 10), nil))
}

var letters = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randLetters(n int) []byte {
    b := make([]byte, n)
    for i := range b {
        b[i] = letters[rand.Intn(len(letters))]
    }
    return b
}
