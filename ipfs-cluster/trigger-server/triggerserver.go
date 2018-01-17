package main

import (
    "log"
    "net/http"
    "os/exec"
    "time"
    "fmt"
    "io/ioutil"
    "math/rand"
    "os"
    "github.com/ipfs/go-ipfs-api"
    "encoding/json"
    "io"
    "bytes"
)

// TriggerConfig is server configuration JSON format
type TriggerConfig struct {
	Secret string
	Port int
	RunnerPath string
	Runner string
	RunnerArgs string
}

const configPath = "/etc/trigger-server.json"
var config TriggerConfig

// IPFS Shell
var sh = shell.NewShell("localhost:5001")

func run(rw http.ResponseWriter, req *http.Request) {
    if req.Method != "POST" {
        fmt.Fprintf(rw, `Kubernetes-IPFS Trigger Server
	https://github.com/ipfs/kubernetes-ipfs/
	`)
	return
    }
    if req.Header.Get("X-Auth") != config.Secret {
	    log.Println("User failed authentication")
	    rw.WriteHeader(http.StatusUnauthorized)
	    fmt.Fprintf(rw, "Invalid authentication\n")
	    return
    }
    // This command takes a while, so we'll hang out here while it runs
    cmd := exec.Command("bash", "-c", fmt.Sprintf("%s%s %s", config.RunnerPath, config.Runner, config.RunnerArgs))
    cmd.Dir = config.RunnerPath
    pipeReader, pipeWriter := io.Pipe()
    out := bytes.NewBufferString("")
    cmd.Stdout = pipeWriter
    cmd.Stderr = pipeWriter
    go writeOutput(rw, out, pipeReader)
    if err := cmd.Run(); err != nil {
	    log.Printf("Failed to start runner (%s/%s %s): %s", config.RunnerPath, config.Runner, config.RunnerArgs, err)
	    return
    }
    pipeWriter.Close()
    log.Println("Finished execution")
    // Add the output to IPFS
    addr, ipfsErr := sh.Add(bytes.NewReader(out.Bytes()))
    // Make it an IPFS link
    addr = "/ipfs/" + addr
    // Send the link to the client
    fmt.Fprintf(rw, addr)
    // Handle errors that may have occurred
    if ipfsErr != nil {
	    log.Println("IPFS Error adding output")
	    log.Println(ipfsErr)
    }
}

func writeOutput(res http.ResponseWriter, out *bytes.Buffer, pipeReader *io.PipeReader) {
	buffer := make([]byte, 4096)
	for {
		n, err := pipeReader.Read(buffer)
		if err != nil {
			pipeReader.Close()
			break
		}

		data := buffer[0:n]
		res.Write(data)
		out.Write(data)
		if f, ok := res.(http.Flusher); ok {
			f.Flush()
		}
		//reset buffer
		for i := 0; i < n; i++ {
			buffer[i] = 0
		}
	}
}

func init() {
    rand.Seed(time.Now().UnixNano())
    configFile, readErr := ioutil.ReadFile(configPath)
    if readErr != nil || len(configFile) < 1 {
        log.Printf("Missing configuration in %s\n", configPath)
	cwd, _ := os.Getwd()
	config.Secret = randLetters(30)
	config.Port = 8082
	config.RunnerPath = cwd
	config.RunnerArgs = ""
	configData, marshalErr := json.Marshal(config)
	if marshalErr != nil {
		log.Printf("Couldn't create config file: %s\n", marshalErr)
		os.Exit(1)
	}
	err := ioutil.WriteFile(configPath, configData, 0644)
	if err != nil {
		log.Printf("Couldn't write config file: %s\n", err)
		os.Exit(1)
	}
	log.Printf("Generated configuration in %s\n", configPath)
    } else {
	    json.Unmarshal(configFile, &config)
    }
}

func main() {
    log.Printf("Starting on port %d\n", config.Port)
    http.HandleFunc("/run", run)
    http.HandleFunc("/", func(rw http.ResponseWriter, req *http.Request) {
        fmt.Fprintf(rw, `Kubernetes-IPFS Trigger Server
	https://github.com/ipfs/kubernetes-ipfs/
	`)
    })
    log.Println(http.ListenAndServe(fmt.Sprintf(":%d", config.Port), nil))
}

var letters = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randLetters(n int) string {
    b := make([]byte, n)
    for i := range b {
        b[i] = letters[rand.Intn(len(letters))]
    }
    return string(b)
}
