package main

import (
  "github.com/gorilla/mux"
  "gopkg.in/mgo.v2"
  "gopkg.in/mgo.v2/bson"
  "log"
  "net/http"
  "os"
)

const MONGO_URL = "localhost:27017"

type user struct {
  Id        bson.ObjectId  `bson:"_id"`
  Username  string         `bson:"username"`
}


func main() {

  sess, err := mgo.Dial(MONGO_URL)
  if err != nil {
    log.Fatalln("Cannot connect to mongo.")
    os.Exit(1)
  }
  defer sess.Close()

  collection := sess.DB("test").C("foo")
  doc := user{Id: bson.NewObjectId(), Username: "root"}
  err = collection.Insert(doc)
  if err != nil {
    log.Println("No insert.")
    os.Exit(1)
  }

  mux := mux.NewRouter()
  mux.HandleFunc("/", index)

  log.Println("Listetning...")
  http.ListenAndServe(":3000", mux)
}

func index(w http.ResponseWriter, r *http.Request) {
  w.Write([]byte("Hello World!"))
}
