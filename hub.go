package main

import (
	"code.google.com/p/goauth2/oauth"
	"github.com/google/go-github/github"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/yosssi/ace"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/marianitadn/Hub/model"
)

const PRODUCTION_ENV = "production"
const DEVELOPMENT_ENV = "development"

type Context struct {
	STORE 	*sessions.CookieStore
	URL 	string
	DB 		*mgo.Database
}

// App config
var ENV = Context{}

// GitHub credentials
var oauthCfg = &oauth.Config{}


func setEnv() {
	var MONGO_URL, SECRET, URL string

	if os.Getenv("GOENV") == PRODUCTION_ENV {
		// Get vars from ENV
		URL = ":" + os.Getenv("PORT")

		MONGO_URL = os.Getenv("MONGO_URL")
		SECRET = os.Getenv("SESSION_SECRET")

		oauthCfg.ClientId = os.Getenv("GH_ID")
		oauthCfg.ClientSecret = os.Getenv("GH_SECRET")
		oauthCfg.AuthURL = "https://github.com/login/oauth/authorize"
		oauthCfg.TokenURL = "https://github.com/login/oauth/access_token"
		oauthCfg.RedirectURL = "https://xphub.herokuapp.com/logged"

	} else {
		URL = "localhost:3000"

		MONGO_URL = "localhost/hub"
		SECRET = "big-secret-here"
	}

	// Try mongoDB connection
	mongo, err := mgo.Dial(MONGO_URL)
	if err != nil {
		log.Fatalln("Cannot reach mongoDB: ", err)
		os.Exit(1)
	}

	ENV.URL = URL
	ENV.STORE = sessions.NewCookieStore([]byte(SECRET))
	ENV.DB = mongo.DB("hub")
}

func main() {

	// Set Environment variables
	setEnv()

	mux := mux.NewRouter()
	mux.HandleFunc("/", index)
	mux.HandleFunc("/login", HandleLogin)
	mux.HandleFunc("/login/{user}", HandleDevLogin)
	mux.HandleFunc("/logged", HandleGitHubLoginResponse)
	mux.HandleFunc("/logout", HandleLogout)
	mux.HandleFunc("/{user}/profile", HandleUserProfile)

	mux.PathPrefix("/").Handler(http.FileServer(http.Dir("./static/")))
	mux.HandleFunc("/{*}", Handle404)

	log.Println("App started at:", ENV.URL)
	http.ListenAndServe(ENV.URL, mux)
}

func updateRepos(user string, access_token string) {
	// Using octokit
	t := &oauth.Transport{
		Token: &oauth.Token{AccessToken: access_token},
	}
	client := github.NewClient(t.Client())

	// List all repositories for the authenticated user
	repos, _, _ := client.Repositories.List("", nil)

	// Push public repos to db
	for _, repo := range repos {
		if (! *repo.Private) {

			collection := ENV.DB.C("repos")
			doc := model.Repo{
				ID:          int(*repo.ID),
				Name:        string(*repo.Name),
				User:        string(user),
				Owner:       string(*repo.Owner.Login),
				Description: string(*repo.Description),
				Fork:        bool(*repo.Fork),
				Stars:       int(*repo.StargazersCount),
				Watchers:    int(*repo.WatchersCount),
				Forks:       int(*repo.ForksCount),
			}
			_, err := collection.UpsertId(doc.ID, doc)
			if err != nil {
				log.Println("Could not upsert repo: ", err)
				os.Exit(1)
			}
		}
	}
}

func index(w http.ResponseWriter, r *http.Request) {
	// Get session info
	session, _ := ENV.STORE.Get(r, "session")
	log.Println(session.Values["user"])

	// Get template
	tpl, err := ace.Load("views/base", "views/index", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Title": "index page",
		"User": session.Values["user"],
		"Msgs": []string{"1", "2", "3"},
		"Map": map[string]int{
			"ceva": 0,
		},
	}

	if err := tpl.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func HandleLogin(w http.ResponseWriter, r *http.Request) {
	// Go to GitHub Authentication page
	url := oauthCfg.AuthURL + "?client_id=" + oauthCfg.ClientId
	url += "&redirect_uri=" + oauthCfg.RedirectURL

	// Redirect user to that page
	http.Redirect(w, r, url, http.StatusFound)
}

func HandleDevLogin(w http.ResponseWriter, r *http.Request) {
	// Get user from URL
	user := mux.Vars(r)["user"]

	// Save username to session
	session, _ := ENV.STORE.Get(r, "session")
    // Set some session values.
    session.Values["user"] = user
    // Save it
    session.Save(r, w)

	// Redirect user to index
	http.Redirect(w, r, "/", http.StatusFound)
}

func HandleLogout(w http.ResponseWriter, r *http.Request) {
	// Get user session and delete it
	session, _ := ENV.STORE.Get(r, "session")
	delete(session.Values, "user")
	_ = session.Save(r, w)

	// Redirect user to that page
	http.Redirect(w, r, "/", http.StatusFound)
}

func HandleUserProfile(w http.ResponseWriter, r *http.Request) {
	// Get session info
	session, _ := ENV.STORE.Get(r, "session")

	// Get user from db
	user := model.User{}
	collection := ENV.DB.C("users")
	err := collection.Find(bson.M{"username":session.Values["user"]}).One(&user)
	if err != nil {
		log.Fatalln("Cannot get user profile: ", err)
		os.Exit(1)
	}

	// Get all user repos
	repos := []model.Repo{}
	collection = ENV.DB.C("repos")
	err = collection.Find(bson.M{"user":session.Values["user"]}).All(&repos)
	if err != nil {
		log.Fatalln("Cannot get user repos: ", err)
		os.Exit(1)
	}

	// Get template
	tpl, err := ace.Load("views/base", "views/profile", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Title": "profile page",
		"User": user,
		"Repos": repos,
	}

	if err := tpl.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func HandleGitHubLoginResponse(w http.ResponseWriter, r *http.Request) {
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

	// Get current user info
	userinfo, _, _ := client.Users.Get("")

	// Add user to db
	collection := ENV.DB.C("users")
	doc := model.User{
		ID:        int(*userinfo.ID),
		Avatar:    string(*userinfo.AvatarURL),
		Username:  string(*userinfo.Login),
		Fullname:  string(*userinfo.Name),
		Followers: int(*userinfo.Followers),
		Following: int(*userinfo.Following),
	}
	_, err := collection.UpsertId(doc.ID, doc)
	if err != nil {
		log.Println("Could not add user: ", err)
		os.Exit(1)
	}

	// Save username to session
	session, _ := ENV.STORE.Get(r, "session")
    // Set some session values.
    session.Values["user"] = string(*userinfo.Login)
    // Save it
    session.Save(r, w)

    // Update user repos
	updateRepos(string(*userinfo.Login), access_token)

    // Redirect to index
	http.Redirect(w, r, "/", http.StatusFound)
}

func Handle404(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("404: Page not found. Go away!"))
}

