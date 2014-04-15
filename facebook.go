package dmv

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/go-martini/martini"
)

var (
	fbProfileURL = "https://graph.facebook.com/me"
)

// Facebook stores the access and refresh tokens along with the users
// profile.
type Facebook struct {
	Errors       []error
	AccessToken  string
	RefreshToken string
	Profile      FacebookProfile
}

// FacebookProfile stores information about the user from facebook.
type FacebookProfile struct {
	ID         string `json:"id"`
	Username   string `json:"username"`
	Name       string `json:"name"`
	LastName   string `json:"last_name"`
	FirstName  string `json:"first_name"`
	MiddleName string `json:"middle_name"`
	Gender     string `json:"gender"`
	Link       string `json:"link"`
	Email      string `json:"email"`
}

// AuthFacebook authenticates users using Facebook and OAuth2.0. After
// handling a callback request, a request is made to get the users
// facebook profile and a Facebook struct will be mapped to the
// current request context.
//
// This function should be called twice in each application, once
// on the login handler, and once on the callback handler.
//
//
//     package main
//
//     import (
//         "github.com/go-martini/martini"
//         "github.com/martini-contrib/sessions"
//         "net/http"
//     )
//
//     func main() {
//         fbOpts := &dmv.OAuth2.0Options{
//             ClientID: "oauth_id",
//             ClientSecret: "oauth_secret",
//             RedirectURL: "http://host:port/auth/callback/facebook",
//         }
//
//         m := martini.Classic()
//         store := sessions.NewCookieStore([]byte("secret123"))
//         m.Use(sessions.Sessions("my_session", store))
//
//         m.Get("/", func(s sessions.Session) string {
//             return "hi" + s.Get("userID")
//         })
//         m.Get("/auth/facebook", dmv.AuthFacebook(fbOpts))
//         m.Get("/auth/callback/facebook", dmv.AuthFacebook(fbOpts), func(fb *dmv.Facebook, req *http.Request, w http.ResponseWriter) {
//             // Handle any errors.
//             if len(fb.Errors) > 0 {
//                 http.Error(w, "Oauth failure", http.StatusInternalServerError)
//                 return
//             }
//             // Do something in a database to create or find the user by the facebook profile id.
//             user := findOrCreateByFacebookID(fb.Profile.ID)
//             s.Set("userID", user.ID)
//             http.Redirect(w, req, "/", http.StatusFound)
//         })
//     }
func AuthFacebook(opts *OAuth2Options) martini.Handler {
	opts.AuthURL = "https://www.facebook.com/dialog/oauth"
	opts.TokenURL = "https://graph.facebook.com/oauth/access_token"

	return func(r *http.Request, w http.ResponseWriter, c martini.Context) {
		transport := makeTransport(opts, r)
		cbPath := ""
		if u, err := url.Parse(transport.Config.RedirectURL); err == nil {
			cbPath = u.Path
		}
		if r.URL.Path != cbPath {
			http.Redirect(w, r, transport.Config.AuthCodeURL(""), http.StatusFound)
			return
		}
		fb := &Facebook{}
		defer c.Map(fb)
		code := r.FormValue("code")
		tk, err := transport.Exchange(code)
		if err != nil {
			fb.Errors = append(fb.Errors, err)
			return
		}
		fb.AccessToken = tk.AccessToken
		fb.RefreshToken = tk.RefreshToken
		resp, err := transport.Client().Get(fbProfileURL)
		if err != nil {
			fb.Errors = append(fb.Errors, err)
			return
		}
		defer resp.Body.Close()
		profile := &FacebookProfile{}
		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fb.Errors = append(fb.Errors, err)
			return
		}
		if err := json.Unmarshal(data, profile); err != nil {
			fb.Errors = append(fb.Errors, err)
			return
		}
		fb.Profile = *profile
		return
	}
}
