package main

import (
  "github.com/gorilla/mux"
  "log"
  "net/http"
)


func main() {
  mux := mux.NewRouter()
  mux.HandleFunc("/", index)

  log.Println("Listetning...")
  http.ListenAndServe(":3000", mux)
}

func index(w http.ResponseWriter, r *http.Request) {
  w.Write([]byte("Hello World!"))
}
