package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

var port = os.Getenv("PORT")
var ctx = context.Background()
var values = []string{}
var rdb *redis.Client
var lastError = ""

var RedisHost = os.Getenv("REDIS_HOST")
var RedisPassword = os.Getenv("REDIS_PASSWORD")
var RedisPort = os.Getenv("REDIS_PORT")
var RedisSSL = os.Getenv("REDIS_SSL")

func GetValues() []string {
	if rdb == nil {
		return values
	}

	vals, err := rdb.SMembers(ctx, "values").Result()
	if err != nil {
		lastError = fmt.Sprintf("Error getting values: %s", err)
		log.Printf(lastError)
	}
	return vals
}

func AddValue(text string) {
	if rdb == nil {
		values = append(values, text)
		return
	}

	err := rdb.SAdd(ctx, "values", text).Err()
	if err != nil {
		lastError = fmt.Sprintf("Error adding value %q: %s", text, err)
		log.Printf(lastError)
	}
}

func DelValue(text string) {
	if rdb == nil {
		for i, t := range values {
			if t == text {
				values = append(values[:i], values[i+1:]...)
				break
			}
		}
		return
	}
	err := rdb.SRem(ctx, "values", text).Err()
	if err != nil {
		lastError = fmt.Sprintf("Error deleting value %q: %s", text, err)
		log.Printf(lastError)
	}
}

func GetPage(w http.ResponseWriter, r *http.Request) {
	log.Printf("%s\n", r.URL.String())
	changed := false

	for _, text := range r.URL.Query()["add"] {
		AddValue(text)
		changed = true
	}
	for _, text := range r.URL.Query()["del"] {
		DelValue(text)
		changed = true
	}

	if changed {
		// Redirect so we clear any query params from the shown URL
		http.Redirect(w, r, "/", 307)
		return
	}

	config := ""
	if RedisHost == "" {
		config = "No DB configured (using in-memory storage)<br>\n"
	} else {
		envs := os.Environ()
		sort.Strings(envs)
		for _, env := range envs {
			if strings.HasPrefix(env, "AZUR") || strings.HasPrefix(env, "RED") {
				if strings.Contains(env, "PASSWORD") {
					i := strings.Index(env, "=")
					env = env[:i+6] + "..."
				}
				config += env + "<br>\n"
			}
		}
	}

	errMsg := lastError
	if errMsg != "" {
		errMsg += "<br>\n"
		lastError = ""
	}

	str := fmt.Sprintf("Item Count: %d<br>\n", len(GetValues()))
	for _, text := range GetValues() {
		str += fmt.Sprintf(`<button id=xx onclick="window.location='?del=%s'">X</button> %s<br>`+"\n", url.QueryEscape(text), text)
	}

	fmt.Fprintf(w, `
<html>
 <title>Azure Container Apps Service Add-ons Redis Sample</title>
 <style>
  body { margin:20px ; background-color:#ebf5fb ; }
  #xx { font-size:10 ;  border:0 ; padding:0 ; background-color:floralwhite ; }
 </style>
 <body>
  <h1>Azure Container App Redis Sample</h1>
  <form action="/">
    Enter some text: <input type =text name=add id=add>&nbsp;
    <input type="submit" value="Add">
  </form>
  <b>Config:</b><br>
  <tt>%s</tt><br>
  <div style="color:red">%s</div>
  <button onclick="window.location='/'">Refresh</button>
  <hr>
  %s
  <script> document.getElementById("add").focus(); </script>
 </body>
</html>
`, config, errMsg, str)
}

func main() {
	if port == "" {
		port = "8080"
	}

	// Cheat for now - look for managed env var names too, if dev isn't set
	if RedisHost == "" {
		Prefix := "AZURE_"
		RedisHost = os.Getenv(Prefix + "REDIS_HOST")
		RedisPassword = os.Getenv(Prefix + "REDIS_PASSWORD")
		RedisPort = os.Getenv(Prefix + "REDIS_PORT")
		RedisSSL = os.Getenv(Prefix + "REDIS_SSL")
	}

	if RedisHost != "" {
		log.Printf("RedisHost: %s:%s", RedisHost, RedisPort)
		log.Printf("RedisPass: %s...", RedisPassword[:5])
		log.Printf("RedisSSL: %s...", RedisSSL)

		tlsConfig := (*tls.Config)(nil)
		if RedisSSL == "true" {
			tlsConfig = &tls.Config{
				MinVersion: tls.VersionTLS12,
			}
		}

		rdb = redis.NewClient(&redis.Options{
			Addr:      RedisHost + ":" + RedisPort,
			Password:  RedisPassword,
			DB:        0, // use default DB
			TLSConfig: tlsConfig,
		}).WithTimeout(time.Second * 10)
	}
	http.HandleFunc("/", GetPage)

	log.Printf("Listening on port %s\n", port)
	http.ListenAndServe(":"+port, nil)
}
