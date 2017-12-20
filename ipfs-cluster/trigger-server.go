package main

import (
    "log"
    "net/http"
    "os/exec"
    "fmt"
    "io/ioutil"
)

func run(rw http.ResponseWriter, req *http.Request) {
    if req.Method == "POST" {
        fmt.Fprintf(rw, "Beginning work...")
	out, err := exec.Command("bash", "./runner.sh", "5", "5").Output()
        io_err := ioutil.WriteFile("/tmp/output", out, 0644)
	fmt.Printf("Wrote output")
	if err != nil {
		log.Fatal(err)
	}
	if io_err != nil {
		log.Fatal(err)
	}
    } else {
        fmt.Fprintf(rw, "Hello")
    }
}

func main() {
    log.Println("Starting")
    http.HandleFunc("/run", run)
    log.Fatal(http.ListenAndServe(":8082", nil))
    log.Println("Listening")
}
