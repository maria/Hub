package main

import (
	"html/template"
    "io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"

	"code.google.com/p/goauth2/oauth"
	"github.com/gin-gonic/gin"
	"github.com/google/go-github/github"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"gopkg.in/mgo.v2"
)

const MONGO_URL = "localhost/hub"

type user struct {
	ID         int     `bson:"_id"`
	Username   string  `bson:"username"`
	Fullname   string  `bson:"fullname"`
	Followers  int     `bson:"followers"`
	Following  int     `bson:"following"`
}

var oauthCfg = &oauth.Config{

	ClientId: "cc2a22d7df2930f8fd18",
	ClientSecret: "b6147809adea6abb45ef5ee4cc6d212934a91aed",

	AuthURL: "https://github.com/login/oauth/authorize",
	TokenURL: "https://github.com/login/oauth/access_token",
	RedirectURL: "http://localhost:3000/logged",
}

var store = sessions.NewCookieStore([]byte("big-secret-here"))


func main() {

	app := gin.Default()

	html := template.Must(template.ParseGlob("views/*.ace"))
	app.SetHTMLTemplate(html)

	app.GET("/", index)
	//app.GET("/login", HandleLogin)
	//app.GET("/login/:user", HandleDevLogin)
	//app.POST("/logged", HandleGitHubLoginResponse)
	//app.DELETE("/logout", HandleLogout)
	//app.GET("/{*}", Handle404)

	address := ":3000"
	if os.Getenv("PORT") != "" {
		address = os.Getenv("HOST") + ":" + os.Getenv("PORT")
	}

	app.Run(address)
}

func index(c *gin.Context) {

	data := map[string]interface{}{
		"Title": "index page",
		"User": "maria",
		"Msgs": []string{"1", "2", "3"},
		"Map": map[string]int{
			"ceva": 0,
		},
	}

	c.HTML(201, "index.ace", data)
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
	session, _ := store.Get(r, "session")
    // Set some session values.
    session.Values["user"] = user
    // Save it
    session.Save(r, w)

	// Redirect user to index
	http.Redirect(w, r, "/", http.StatusFound)
}

func HandleLogout(w http.ResponseWriter, r *http.Request) {
	// Get user session and delete it
	session, _ := store.Get(r, "session")
	delete(session.Values, "user")
	_ = session.Save(r, w)

	// Redirect user to that page
	http.Redirect(w, r, "/", http.StatusFound)
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

	// list all repositories for the authenticated user
	// repos, _, _ := client.Repositories.List()
	// log.Println(repos)

	// Get current user info
	userinfo, _, _ := client.Users.Get("")

	// Connect to mongo
	mongourl := MONGO_URL
	if os.Getenv("OPENSHIFT_MONGODB_DB_URL") != "" {
		mongourl = os.Getenv("OPENSHIFT_MONGODB_DB_URL")
	}
	mongo, err := mgo.Dial(mongourl)
	if err != nil {
		log.Fatalln("Cannot connect to mongo: %s", err)
		os.Exit(1)
	}
	defer mongo.Close()

	// Add user to db
	collection := mongo.DB("hub").C("users")
	doc := user{
		ID:        int(*userinfo.ID),
		Username:  string(*userinfo.Login),
		Fullname:  string(*userinfo.Name),
		Followers: int(*userinfo.Followers),
		Following: int(*userinfo.Following),
	}
	_, err = collection.UpsertId(doc.ID, doc)
	if err != nil {
		log.Println("Could not add user: %s", err)
		os.Exit(1)
	}

	// Save username to session
	session, _ := store.Get(r, "session")
    // Set some session values.
    session.Values["user"] = string(*userinfo.Login)
    // Save it
    session.Save(r, w)

    // Redirect to index
	http.Redirect(w, r, "/", http.StatusFound)
}

func Handle404(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("404: Page not found. Go away!"))
}

