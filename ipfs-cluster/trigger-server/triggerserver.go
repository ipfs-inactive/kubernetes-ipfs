package main

import (
	"bytes"
	"fmt"
	"github.com/ipfs/go-ipfs-api"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"time"
)

const CONFIG_PATH = "/etc/trigger-server"

// HTTP API listen port
const PORT = 8082

var RUNNER_PATH = os.Getenv("GOPATH") + "src/github.com/ipfs/kubernetes-ipfs/ipfs-cluster/"

// Runner takes two args
const RUNNER_NUM_NODES = "5"
const RUNNER_NUM_PINS = "5"

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
			cmd := exec.Command("bash", "runner.sh", RUNNER_NUM_NODES, RUNNER_NUM_PINS)
			cmd.Dir = RUNNER_PATH
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
	secret_bytes, readErr := ioutil.ReadFile(CONFIG_PATH)
	if readErr != nil {
		log.Printf("Missing secret configuration in %s. We'll generate one for you...\n", CONFIG_PATH)
		rand.Seed(time.Now().UnixNano())
		secret_bytes = randLetters(30)
		ioErr := ioutil.WriteFile(CONFIG_PATH, secret_bytes, 0644)
		if ioErr != nil {
			log.Fatal("Unable to write new secret file; check permissions and try again")
		}
	}
	secret = string(secret_bytes)
	log.Printf("Starting with secret as %s on port %d\n", secret, PORT)
	http.HandleFunc("/run", run)
	http.HandleFunc("/", func(rw http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(rw, `Kubernetes-IPFS Trigger Server
	https://github.com/ipfs/kubernetes-ipfs/
	`)
	})
	log.Println(http.ListenAndServe(":"+strconv.FormatInt(PORT, 10), nil))
}

var letters = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randLetters(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return b
}
