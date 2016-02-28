package main

import "encoding/json"
import "log"
import "net/http"
import "io/ioutil"
import "os"
import "strconv"
import "strings"

type Unmarshaller interface {
	Unmarshal([]byte, *map[string]interface{}) error
}
type mainUnmarshaller struct{}

func (unmarshaller mainUnmarshaller) Unmarshal(inp []byte, resp *map[string]interface{}) error {
	return json.Unmarshal(inp, resp)
}

type ResponseConverter interface {
	Convert(*http.Response) ([]byte, error)
}
type mainConverter struct{}

func (converter mainConverter) Convert(response *http.Response) ([]byte, error) {
	defer response.Body.Close()
	return ioutil.ReadAll(response.Body)
}

type HttpResponseFetcher interface {
	Fetch(url string) (*http.Response, error)
}
type mainFetcher struct{}

func (fetcher mainFetcher) Fetch(url string) (*http.Response, error) {
	log.Printf("Retrieving %q\n", url)
	return http.Get(url)
}

func GetBeerPage(fetcher HttpResponseFetcher, converter ResponseConverter, id int) string {
	url := "https://api.untappd.com/v4/beer/info/BID?client_id=CLIENTID&client_secret=CLIENTSECRET&compact=true"
	url = strings.Replace(url, "BID", strconv.Itoa(id), 1)
	url = strings.Replace(url, "CLIENTID", os.Getenv("CLIENTID"), 1)
	url = strings.Replace(url, "CLIENTSECRET", os.Getenv("CLIENTSECRET"), 1)

	response, err := fetcher.Fetch(url)

	if err != nil {
		log.Printf("%q\n", err)
	} else {
		contents, err := converter.Convert(response)
		if err != nil {
			log.Printf("%q\n", err)
		} else {
			return string(contents)
		}
	}

	return "Failed to retrieve " + strconv.Itoa(id)
}

func ConvertPageToName(page string, unmarshaller Unmarshaller) string {
	var mapper map[string]interface{}
	err := unmarshaller.Unmarshal([]byte(page), &mapper)
	if err != nil {
		log.Printf("%q\n", err)
		return "Failed to unmarshal"
	}

	response := mapper["response"].(map[string]interface{})
	beer := response["beer"].(map[string]interface{})
	brewery := beer["brewery"].(map[string]interface{})
	return brewery["brewery_name"].(string) + " - " + beer["beer_name"].(string)
}

func GetBeerName(id int) string {
	var fetcher HttpResponseFetcher = mainFetcher{}
	var converter ResponseConverter = mainConverter{}
	var unmarshaller Unmarshaller = mainUnmarshaller{}
	text := GetBeerPage(fetcher, converter, id)
	return ConvertPageToName(text, unmarshaller)
}