package main

import (
	elastic "gopkg.in/olivere/elastic.v3"
	"fmt"
	"encoding/json"
	"net/http"
	"log"
	"strconv"
	"reflect"
	"github.com/pborman/uuid"
)

type Location struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type Post struct {
	User     string   `json:"user"`
	Message  string   `json:"message"`
	Location Location `json:"location"`
}

const (
	DISTANCE = "200km"
	INDEX = "around"
	TYPE = "post"
	// Needs to update
	//PROJECT_ID = "around-xxx"
	//BT_INSTANCE = "around-post"
	// Needs to update this URL if you deploy it to cloud.
	ES_URL = "http://35.193.103.196:9200"
)	

func main() {
	// Create a client
	client, err := elastic.NewClient(elastic.SetURL(ES_URL), elastic.SetSniff(false))
	if err != nil {
		panic(err)
		return
	}

	// Use the IndexExists service to check if a specified index exists.
	exists, err := client.IndexExists(INDEX).Do()
	if err != nil {
		panic(err)
	}
	if !exists {
		// Create a new index.
		mapping := `{
			"mappings":{
				"post":{
					"properties":{
						"location":{
							"type":"geo_point"
						}
					}
				}
			}
		}`
		_, err := client.CreateIndex(INDEX).Body(mapping).Do()
		if err != nil {
			// Handle error
			panic(err)
		}
	}

	fmt.Println("started-service")
	http.HandleFunc("/post", handlerPost)
	http.HandleFunc("/search", handlerSearch)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

//{
//	"user_name": "john",
//	"message": "test",
//	"location": {
//		"lat": 37,
//		"lon": -120
//	}
//}

func handlerPost(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Received one post request.");
	decoder := json.NewDecoder(r.Body) // string to Go's structure, create a constructor
	var p Post
	if err := decoder.Decode(&p); err != nil {
		panic(err)
		return
	}
	fmt.Fprintf(w, "Post received:%s\n", p.Message)
	id := uuid.New()
	// Save to ES.
	saveToES(&p, id)
}

func handlerSearch(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Received one request for search")

	lat, _ := strconv.ParseFloat(r.URL.Query().Get("lat"), 64)
	lon, _ := strconv.ParseFloat(r.URL.Query().Get("lon"), 64)
	// range is optional 
	ran := DISTANCE 
	if val := r.URL.Query().Get("range"); val != "" { 
		ran = val + "km" 
	}

	fmt.Printf( "Search received: %f %f %s\n", lat, lon, ran)

	// Create a client
	client, err := elastic.NewClient(elastic.SetURL(ES_URL), elastic.SetSniff(false))
	if err != nil {
		panic(err)
		return
	}  
	// Define geo distance query as specified in
	// https://www.elastic.co/guide/en/elasticsearch/reference/5.2/query-dsl-geo-distance-query.html
	q := elastic.NewGeoDistanceQuery("location")
	q = q.Distance(ran).Lat(lat).Lon(lon)

	// Some delay may range from seconds to minutes. So if you don't get enough results. Try it later.
	searchResult, err := client.Search().
		Index(INDEX).
		Query(q).
		Pretty(true).
		Do()

	if err != nil {
		// Handle error
		panic(err)
	}
	fmt.Printf("Query took %d milliseconds\n", searchResult.TookInMillis)
	fmt.Printf("Found a total of %d post\n", searchResult.TotalHits())

	var typ Post
	var ps []Post
	for _, item := range searchResult.Each(reflect.TypeOf(typ)) { // instance of . in Java
		p := item.(Post) // p = (Post) item . in Java
		fmt.Printf("Post by %s: %s at lat %v and lon %v\n", 
			p.User, p.Message, p.Location.Lat, p.Location.Lon)

		// TODO(student homework): Perform filtering based on keywords such as web spam etc.
		ps = append(ps, p)

	}
	
	js, err := json.Marshal(ps)
	if err != nil {
		panic(err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Write(js)

}

func saveToES(p *Post, id string) {
	es_client, err := elastic.NewClient(elastic.SetURL(ES_URL), elastic.SetSniff(false))
	if err != nil {
		panic(err)
		return
	}
	// Save it to index
	_, err = es_client.Index().
		Index(INDEX).
		Type(TYPE).
		Id(id).
		BodyJson(p).
		Refresh(true).
		Do()
	if err != nil {
		panic(err)
		return
	}

	fmt.Printf("Post is saved to Index: %s\n", p.Message)

}

// func handlerSearch(w http.ResponseWriter, r *http.Request) {
//       fmt.Println("Received one request for search")

//       lat, _ := strconv.ParseFloat(r.URL.Query().Get("lat"), 64)
//       lon, _ := strconv.ParseFloat(r.URL.Query().Get("lon"), 64)
//       // range is optional 
//       ran := DISTANCE 
//       if val := r.URL.Query().Get("range"); val != "" { 
//          ran = val + "km" 
//       }

//       fmt.Println("range is ", ran)

//       // Return a fake post
//       p := &Post{
//              User:"1111",
//              Message:"一生必去的100个地方",
//              Location: Location{
//                     Lat:lat,
//                     Lon:lon,
//              },
//       }

//       js, err := json.Marshal(p) // Marshal(go's structure) -> return json stucture - string representation
//       if err != nil {
//              panic(err)
//              return
//       }

//       w.Header().Set("Content-Type", "application/json")
//       w.Write(js)
// }