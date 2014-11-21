package main

import (
	"code.google.com/p/goauth2/oauth"
	"github.com/google/go-github/github"
	"github.com/gorilla/mux"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
)

const MONGO_URL = "localhost:27017"

type user struct {
	Id        bson.ObjectId  `bson:"_id"`
	Username  string         `bson:"username"`
}

var oauthCfg = &oauth.Config{

	ClientId: "cc2a22d7df2930f8fd18",
	ClientSecret: "b6147809adea6abb45ef5ee4cc6d212934a91aed",
 
	AuthURL: "https://github.com/login/oauth/authorize",
	TokenURL: "https://github.com/login/oauth/access_token",
	RedirectURL: "http://localhost:3000/logged",
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
	mux.HandleFunc("/login", login)
	mux.HandleFunc("/logged", logged)

	log.Println("Listetning...")
	http.ListenAndServe(":3000", mux)
}

func index(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello World!"))
}

func login(w http.ResponseWriter, r *http.Request) {
	// Go to GitHub Authentication page
	url := oauthCfg.AuthURL + "?client_id=" + oauthCfg.ClientId
	url += "&redirect_uri=" + oauthCfg.RedirectURL

	// Redirect user to that page
	http.Redirect(w, r, url, http.StatusFound)
}

func logged(w http.ResponseWriter, r *http.Request) {
	// Get the code from the response
	code := r.FormValue("code")

	// Build token url
	token_url := oauthCfg.TokenURL + "?client_id=" + oauthCfg.ClientId
	token_url += "&client_secret=" + oauthCfg.ClientSecret + "&code=" + code

	res, _ := http.Post(token_url, "application/json", nil)
	response, _ := ioutil.ReadAll(res.Body)

	// Parse response and get access token
	m, _ := url.ParseQuery(string(response))
	access_token := m["access_token"][0]
	log.Println(access_token)

	// Using octokit
	t := &oauth.Transport{
		Token: &oauth.Token{AccessToken: access_token},
	}

	client := github.NewClient(t.Client())

	// list all repositories for the authenticated user
	// repos, _, _ := client.Repositories.List()
	// log.Println(repos)

	// Get current user info
	user, _, _ := client.Users.Get("")
	log.Println(user)
	log.Println(string(*user.Login))
}
