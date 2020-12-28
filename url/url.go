package url

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
)

var (
	apiKey string
)

func init() {
	apiKey = os.Getenv("CUTTLY_API_KEY")
}

type urlResponse struct {
	URL struct {
		ShortLink string `json:"shortLink"`
	} `json:"url"`
}

// GetShortenedAlias - Returns a shortened uri alias
func GetShortenedAlias(uri string) (string, error) {
	resp, err := http.Get(fmt.Sprintf("https://cutt.ly/api/api.php?key=%s&short=%s", apiKey, url.QueryEscape(uri)))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var u urlResponse
	json.Unmarshal(body, &u)
	log.Println(u)

	return u.URL.ShortLink, nil
}
