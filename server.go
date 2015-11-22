package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"github.com/drone/routes"

	//"golang.org/x/net/context"
)

type locationStruct struct {
	Address    string `json:"address"`
	City       string `json:"city"`
	Coordinate struct {
		Lat float64 `json:"lat"`
		Lng float64 `json:"lng"`
	} `json:"coordinate"`
	ID    bson.ObjectId `json:"id" bson:"_id,omitempty"`
	Name  string        `json:"name"`
	State string        `json:"state"`
	Zip   string        `json:"zip"`
}

type GoogleLocationStruct struct {
	Results []struct {
		AddressComponents []struct {
			LongName  string   `json:"long_name"`
			ShortName string   `json:"short_name"`
			Types     []string `json:"types"`
		} `json:"address_components"`
		FormattedAddress string `json:"formatted_address"`
		Geometry         struct {
			Location struct {
				Lat float64 `json:"lat"`
				Lng float64 `json:"lng"`
			} `json:"location"`
			LocationType string `json:"location_type"`
			Viewport     struct {
				Northeast struct {
					Lat float64 `json:"lat"`
					Lng float64 `json:"lng"`
				} `json:"northeast"`
				Southwest struct {
					Lat float64 `json:"lat"`
					Lng float64 `json:"lng"`
				} `json:"southwest"`
			} `json:"viewport"`
		} `json:"geometry"`
		PartialMatch bool     `json:"partial_match"`
		PlaceID      string   `json:"place_id"`
		Types        []string `json:"types"`
	} `json:"results"`
	Status string `json:"status"`
}
type UberResults struct {
	Prices []struct {
		CurrencyCode         string  `json:"currency_code"`
		DisplayName          string  `json:"display_name"`
		Distance             float64 `json:"distance"`
		Duration             int     `json:"duration"`
		Estimate             string  `json:"estimate"`
		HighEstimate         int     `json:"high_estimate"`
		LocalizedDisplayName string  `json:"localized_display_name"`
		LowEstimate          int     `json:"low_estimate"`
		Minimum              int     `json:"minimum"`
		ProductID            string  `json:"product_id"`
		SurgeMultiplier      int     `json:"surge_multiplier"`
	} `json:"prices"`
}

type UberPostRequest struct {
	LocationIds            []string `json:"location_ids"`
	StartingFromLocationID string   `json:"starting_from_location_id"`
}

type UberResponse struct {
	NextDestinationLocationID string        `json:"next_destination_location_id" bson:"next_destination_location_id"`
	StartingFromLocationID    string        `json:"starting_from_location_id" bson:"starting_from_location_id"`
	Status                    string        `json:"status"`
	TotalDistance             float64       `json:"total_distance" bson:"total_distance"`
	TotalUberCosts            int           `json:"total_uber_costs" bson:"total_uber_costs"`
	TotalUberDuration         int           `json:"total_uber_duration" bson:"total_uber_duration"`
	UberWaitTimeEta           int           `json:"uber_wait_time_eta" bson:"uber_wait_time_eta"`
	BestRouteLocationIds      []string      `json:"best_route_location_ids" bson:"best_route_location_ids"`
	ID                        bson.ObjectId `json:"id" bson:"_id,omitempty"`
}

type UberSandBoxRequestResponse struct {
	Driver          interface{} `json:"driver"`
	Eta             int         `json:"eta"`
	Location        interface{} `json:"location"`
	RequestID       string      `json:"request_id"`
	Status          string      `json:"status"`
	SurgeMultiplier int         `json:"surge_multiplier"`
	Vehicle         interface{} `json:"vehicle"`
}

type UberSandboxRequestIDJSON struct {
	EndLatitude    float64 `json:"end_latitude"`
	EndLongitude   float64 `json:"end_longitude"`
	ProductID      string  `json:"product_id"`
	StartLatitude  float64 `json:"start_latitude"`
	StartLongitude float64 `json:"start_longitude"`
}

const googleURLPrefix string = "http://maps.google.com/maps/api/geocode/json?address="
const googleURLPostfix string = "&sensor=false"
const mongoURL string = "mongodb://savioferns321:mongodb123@ds041633.mongolab.com:41633/savio_mongo"
const mongoDBName string = "savio_mongo"
const mongoCollectionName string = "addresses"
const uberRequestURL string = "https://api.uber.com/v1/estimates/price?start_latitude=[start_latitude]&start_longitude=[start_longitude]&end_latitude=[end_latitude]&end_longitude=[end_longitude]&server_token=O5w7yLR8AWiS3f3fmXz2ypcsW0l6m5VjiIQayHCW"
const startLatitude string = "[start_latitude]"
const startLongitude string = "[start_longitude]"
const endLatitude string = "[end_latitude]"
const endLongitude string = "[end_longitude]"
const serverToken string = "[server_token]"

func (location *GoogleLocationStruct) getGoogleLocation(requestURL string) {
	var buffer bytes.Buffer
	buffer.WriteString(googleURLPrefix)
	buffer.WriteString(requestURL)
	buffer.WriteString(googleURLPostfix)

	url := buffer.String()
	fmt.Println("Url is ", url)

	res, _ := http.Get(url)
	/*if err != nil {

		w.Write([]byte(`{    "error": "Unable to parse data from Google. Error at res, err := http.Get(url) -- line 75"}`))
		panic(err.Error())
	}*/

	body, _ := ioutil.ReadAll(res.Body)
	/*if err != nil {

		w.Write([]byte(`{    "error": "Unable to parse data from Google. body, err := ioutil.ReadAll(res.Body) -- line 84"}`))
		panic(err.Error())
	}*/

	_ = json.Unmarshal(body, &location)
	/*if err != nil {

		w.Write([]byte(`{    "error": "Unable to unmarshal Google data. body, 	err = json.Unmarshal(body, &googleLocation) -- line 94"}`))
		panic(err.Error())
	}*/
}

func getMongoCollection(collName string) (mgo.Collection, mgo.Session) {
	maxWait := time.Duration(20 * time.Second)
	session, err := mgo.DialWithTimeout("mongodb://savioferns321:mongodb123@ds041633.mongolab.com:41633/savio_mongo", maxWait)
	if err != nil {
		fmt.Println("Unable to connect to MongoDB")
		panic(err)
	}
	// Optional. Switch the session to a monotonic behavior.
	session.SetMode(mgo.Monotonic, true)

	mongoCollection := session.DB("savio_mongo").C(collName)
	return *mongoCollection, *session
}

func addLocation(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var t locationStruct
	err := decoder.Decode(&t)
	if err != nil {
		panic("Some error in decoding the JSON")
	}
	//TODO Hit Google's API and retrieve the co-ordinates, then save this struct to a MongoDB instance and retrieve the auto-generated ID.
	requestURL := strings.Join([]string{t.Address, t.City, t.State, t.Zip}, "+")
	requestURL = strings.Replace(requestURL, " ", "%20", -1)

	var googleLocation GoogleLocationStruct

	googleLocation.getGoogleLocation(requestURL)

	//TODO Prepare a response which will return the ID and the details of the JSON in JSON format.
	t.Coordinate.Lat = googleLocation.Results[0].Geometry.Location.Lat
	t.Coordinate.Lng = googleLocation.Results[0].Geometry.Location.Lng
	t.ID = bson.NewObjectId()
	//TODO Set the ID as the auto generated ID from MongoDB
	c, s := getMongoCollection("addresses")
	defer s.Close()

	err = c.Insert(bson.M{"_id": t.ID, "name": t.Name, "address": t.Address, "city": t.City, "state": t.State, "zip": t.Zip, "coordinate": bson.M{"lat": t.Coordinate.Lat, "lng": t.Coordinate.Lng}})
	if err != nil {
		fmt.Println("Error at line 124 ---- c := session.DB(tripplannerdb).C(addresses)")
		log.Fatal(err)
	}

	//TODO Store the output JSON into MongoDB with the ID as its key
	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	outputJSON, err := json.Marshal(t)
	if err != nil {

		w.Write([]byte(`{    "error": "Unable to marshal response. body, 	outputJSON, err := json.Marshal(t) -- line 110"}`))
		panic(err.Error())
	}
	w.Write(outputJSON)
}

func findLocation(w http.ResponseWriter, r *http.Request) {

	locationID := r.URL.Query().Get(":locationID")
	fmt.Println("Location ID is : ", locationID)
	var result locationStruct
	c, s := getMongoCollection("addresses")
	defer s.Close()
	err := c.Find(bson.M{"_id": bson.ObjectIdHex(locationID)}).One(&result)
	if err != nil {
		fmt.Println("err = c.Find(bson.M{\"id\": bson.M{\"$oid\": t}}).One(&result)")
		log.Fatal(err)
	}
	//Returning the result to user
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	outputJSON, err := json.Marshal(result)
	if err != nil {

		w.Write([]byte(`{    "error": "Unable to marshal response. body, 	outputJSON, err := json.Marshal(t) -- line 110"}`))
		panic(err.Error())
	}
	w.Write(outputJSON)

}

func updateLocation(w http.ResponseWriter, r *http.Request) {
	locationID := r.URL.Query().Get(":locationID")
	fmt.Println("Received location ID ", locationID)
	decoder := json.NewDecoder(r.Body)
	var t locationStruct
	err := decoder.Decode(&t)
	if err != nil {
		panic("Some error in decoding the JSON")
	}
	t.ID = bson.ObjectIdHex(locationID)

	//If location values are not null then update the co-ordinates from Google API
	var requestURL bytes.Buffer
	if len(t.Address) > 0 {
		requestURL.WriteString(t.Address + "+")
	}
	if len(t.City) > 0 {
		requestURL.WriteString(t.City + "+")
	}
	if len(t.State) > 0 {
		requestURL.WriteString(t.State + "+")
	}
	if len(t.Zip) > 0 {
		requestURL.WriteString(t.Zip)
	}

	if requestURL.Len() > 0 {
		//The Location needs to be changed via Google API
		requestStr := strings.Replace(requestURL.String(), " ", "%20", -1)
		var googleLocation GoogleLocationStruct
		googleLocation.getGoogleLocation(requestStr)

		//TODO Prepare a response which will return the ID and the details of the JSON in JSON format.
		t.Coordinate.Lat = googleLocation.Results[0].Geometry.Location.Lat
		t.Coordinate.Lng = googleLocation.Results[0].Geometry.Location.Lng
	}

	//Perform the update
	c, s := getMongoCollection("addresses")
	defer s.Close()
	err = c.Update(bson.M{"_id": bson.ObjectIdHex(locationID)}, t)
	if err != nil {
		fmt.Println("	Line 248 : err = c.Update(bson.M{id: bson.M{$oid: locationID}}, t)")
		log.Fatal(err)
	}

	//Prepare and write the response
	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	outputJSON, err := json.Marshal(t)
	if err != nil {

		w.Write([]byte(`{    "error": "Unable to marshal response. body, 	outputJSON, err := json.Marshal(t) -- line 110"}`))
		panic(err.Error())
	}
	w.Write(outputJSON)
	fmt.Println("Update done successfully!")

}

func deleteLocation(w http.ResponseWriter, r *http.Request) {

	locationID := r.URL.Query().Get(":locationID")
	fmt.Println("Location ID is : ", locationID)

	c, s := getMongoCollection("addresses")
	defer s.Close()
	err := c.Remove(bson.M{"_id": bson.ObjectIdHex(locationID)})
	if err != nil {
		fmt.Println("err = c.Find(bson.M{\"id\": bson.M{\"$oid\": t}}).One(&result)")
		log.Fatal(err)
	}

	//Returning the result to user
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	if err != nil {

		w.Write([]byte(`{    "error": "Unable to marshal response. body, 	outputJSON, err := json.Marshal(t) -- line 110"}`))
		panic(err.Error())
	}
	w.Write([]byte(`{"result": "Delete operation done successfully."}`))

}

func getUberCost(start locationStruct, end locationStruct) (int, int, float64, string) {

	uberURL := strings.Replace(uberRequestURL, startLatitude, strconv.FormatFloat(start.Coordinate.Lat, 'f', -1, 64), -1)
	uberURL = strings.Replace(uberURL, startLongitude, strconv.FormatFloat(start.Coordinate.Lng, 'f', -1, 64), -1)
	uberURL = strings.Replace(uberURL, endLatitude, strconv.FormatFloat(end.Coordinate.Lat, 'f', -1, 64), -1)
	uberURL = strings.Replace(uberURL, endLongitude, strconv.FormatFloat(end.Coordinate.Lng, 'f', -1, 64), -1)

	res, err := http.Get(uberURL)

	if err != nil {

		//w.Write([]byte(`{    "error": "Unable to parse data from Google. Error at res, err := http.Get(url) -- line 75"}`))
		fmt.Println("Unable to parse data from Google. Error at res, err := http.Get(url) -- line 75")
		panic(err.Error())
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {

		//w.Write([]byte(`{    "error": "Unable to parse data from Google. body, err := ioutil.ReadAll(res.Body) -- line 84"}`))
		fmt.Println("Unable to parse data from Google. Error at res, err := http.Get(url) -- line 84")
		panic(err.Error())
	}

	var uberResult UberResults
	_ = json.Unmarshal(body, &uberResult)

	return uberResult.Prices[0].LowEstimate, uberResult.Prices[0].Duration, uberResult.Prices[0].Distance, uberResult.Prices[0].ProductID
}

func planTrip(w http.ResponseWriter, r *http.Request) {
	//Decode the request and get all the location IDS
	decoder := json.NewDecoder(r.Body)
	var t UberPostRequest
	err := decoder.Decode(&t)
	if err != nil {
		panic("Some error in decoding the JSON")
	}

	c, s := getMongoCollection("addresses")
	defer s.Close()
	var startLocation locationStruct

	err = c.Find(bson.M{"_id": bson.ObjectIdHex(t.StartingFromLocationID)}).One(&startLocation)
	if err != nil {
		panic("Some error in querying for ID ")
	}
	tripStops := make([]locationStruct, len(t.LocationIds))
	optimumStops := make([]locationStruct, 0)
	for i := 0; i < len(t.LocationIds); i++ {
		var currentLocation locationStruct
		err = c.Find(bson.M{"_id": bson.ObjectIdHex(t.LocationIds[i])}).One(&currentLocation)
		tripStops[i] = currentLocation
	}

	var minCost int
	var minDur int
	var minDist float64
	currentStart := startLocation
	currentStart, tripStops, optimumStops, minCost, minDur, minDist = getCoordinates(currentStart, tripStops, optimumStops, minCost, minDur, minDist)

	fmt.Println("---------------------------------------------------\nFinal output is : ")
	fmt.Println("Start location is : ", startLocation.Name, "\n")
	printLocationNames(optimumStops)
	fmt.Println("Total cost : ", minCost)
	fmt.Println("Total duration : ", minDur)
	fmt.Println("Total distance : ", minDist)

	//Store the result in Mongo DB
	var tripPlan UberResponse
	tripPlan.ID = bson.NewObjectId()
	tripPlan.Status = "planning"
	tripPlan.StartingFromLocationID = t.StartingFromLocationID
	for i := 0; i < len(optimumStops); i++ {
		tripPlan.BestRouteLocationIds = append(tripPlan.BestRouteLocationIds, optimumStops[i].ID.Hex())
	}
	tripPlan.TotalDistance = minDist
	tripPlan.TotalUberCosts = minCost
	tripPlan.TotalUberDuration = minDur

	//Calculating the round trip distance
	var roundTripDistance float64
	var roundTripCost int
	var roundTripDur int

	roundTripCost, roundTripDur, roundTripDistance, _ = getUberCost(obtainLocation(tripPlan.BestRouteLocationIds[len(tripPlan.BestRouteLocationIds)-1]), obtainLocation(tripPlan.StartingFromLocationID))

	tripPlan.TotalDistance += roundTripDistance
	tripPlan.TotalUberCosts += roundTripCost
	tripPlan.TotalUberDuration += roundTripDur

	c, s = getMongoCollection("trips")
	err = c.Insert(bson.M{"_id": tripPlan.ID, "status": tripPlan.Status, "starting_from_location_id": tripPlan.StartingFromLocationID,
		"best_route_location_ids": tripPlan.BestRouteLocationIds, "total_uber_costs": tripPlan.TotalUberCosts,
		"total_uber_duration": tripPlan.TotalUberDuration, "total_distance": tripPlan.TotalDistance})
	if err != nil {
		panic("Error while inserting the trip entry!")
	}

	//Write the result to reponse
	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	outputJSON, err := json.Marshal(tripPlan)
	if err != nil {

		w.Write([]byte(`{    "error": "Unable to marshal response. body, 	outputJSON, err := json.Marshal(t) -- line 110"}`))
		panic(err.Error())
	}
	w.Write(outputJSON)
	fmt.Println("Operation completed successfully! ID : ", tripPlan.ID)

}

func getCoordinates(start locationStruct, input []locationStruct, output []locationStruct, totalCost int, totalDur int, totalDist float64) (locationStruct, []locationStruct, []locationStruct, int, int, float64) {

	if len(input) == 0 {
		return start, input, output, totalCost, totalDur, totalDist
	} else {
		// Find the nearest location from start location
		min := 0
		var currentCost int
		var currentDist float64
		var minCost int
		var minDur int
		var minDist float64
		minCost, _, minDist, _ = getUberCost(start, input[min])
		for i := 0; i < len(input); i++ {
			currentCost, _, _, _ = getUberCost(start, input[i])

			if currentCost < minCost && currentCost != -1 {
				min = i
				minCost, _, _, _ = getUberCost(start, input[i])
			} else if currentCost == minCost {
				currentCost, _, currentDist, _ = getUberCost(start, input[i])

				if currentDist < minDist && currentDist != -1 {
					min = i
					minCost, _, _, _ = getUberCost(start, input[i])
				}

			}

		}

		//Consider this location as the start location
		nearestLocation := input[min]
		minCost, minDur, minDist, _ = getUberCost(start, nearestLocation)
		totalCost += minCost
		totalDur += minDur
		totalDist += minDist

		//Remove it from input slice and append it to the output slice
		output = append(output, input[min])
		input = append(input[:min], input[min+1:]...)

		//Recursively call this function until the input slice is empty
		return getCoordinates(nearestLocation, input, output, totalCost, totalDur, totalDist)

	}

}

func getTripDetails(w http.ResponseWriter, r *http.Request) {
	tripID := r.URL.Query().Get(":tripID")
	fmt.Println(tripID)
	var result UberResponse
	c, s := getMongoCollection("trips")
	defer s.Close()
	err := c.Find(bson.M{"_id": bson.ObjectIdHex(tripID)}).One(&result)
	if err != nil {
		fmt.Println("err = c.Find(bson.M{\"id\": bson.M{\"$oid\":", tripID, "}}).One(&result)")
		log.Fatal(err)
	}

	fmt.Println(result)

	//Returning the result to user
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	outputJSON, err := json.Marshal(result)
	if err != nil {

		w.Write([]byte(`{    "error": "Unable to marshal response. body, 	outputJSON, err := json.Marshal(t) -- line 110"}`))
		panic(err.Error())
	}
	w.Write(outputJSON)

}

func printLocationNames(locations []locationStruct) {
	for i := 0; i < len(locations); i++ {
		fmt.Print(" ", locations[i].Name, " ")
	}
	fmt.Println("")
}

func requestTrip(w http.ResponseWriter, r *http.Request) {
	tripID := r.URL.Query().Get(":tripID")
	var result UberResponse
	var currrentStartLocation string
	c, s := getMongoCollection("trips")
	defer s.Close()
	err := c.Find(bson.M{"_id": bson.ObjectIdHex(tripID)}).One(&result)
	if err != nil {
		fmt.Println("err = c.Find(bson.M{\"id\": bson.M{\"$oid\":", tripID, "}}).One(&result)")
		log.Fatal(err)
	}

	if result.Status != "completed" {

		if result.Status == "planning" && len(result.NextDestinationLocationID) == 0 {
			//This is the first request
			result.Status = "requesting"
			currrentStartLocation = result.StartingFromLocationID
			//Set the next destination field
			result.NextDestinationLocationID = result.BestRouteLocationIds[0]
			//Populate the ETA field
			populateUberETA(&result, currrentStartLocation)
		} else if result.StartingFromLocationID == result.NextDestinationLocationID {
			result.Status = "completed"
			currrentStartLocation = result.StartingFromLocationID
		} else {
			//This is the subsequent request
			for p, v := range result.BestRouteLocationIds {
				if v == result.NextDestinationLocationID && p != len(result.BestRouteLocationIds)-1 {
					currrentStartLocation = result.NextDestinationLocationID
					result.NextDestinationLocationID = result.BestRouteLocationIds[p+1]
					break
				}
				if p == len(result.BestRouteLocationIds)-1 {
					currrentStartLocation = result.NextDestinationLocationID
					result.NextDestinationLocationID = result.StartingFromLocationID
				}

			}
			//Populate the ETA field
			populateUberETA(&result, currrentStartLocation)

		}

		//Update the trip in MongoDB
		err = c.Update(bson.M{"_id": bson.ObjectIdHex(tripID)}, result)
		if err != nil {
			fmt.Println("	Line 539 : err = c.Update(bson.M{\"_id\": bson.ObjectIdHex(tripID)}, t)")
			log.Fatal(err)
		}
	}
	//Returning the result to user
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	outputJSON, err := json.Marshal(result)
	if err != nil {
		w.Write([]byte(`{    "error": "Unable to marshal response. body, 	outputJSON, err := json.Marshal(t) -- line 548"}`))
		panic(err.Error())
	}
	w.Write(outputJSON)

}

func populateUberETA(inputTrip *UberResponse, startLocationID string) {
	apiurl := "https://sandbox-api.uber.com/v1/requests"

	startLocation := obtainLocation(startLocationID)
	endLocation := obtainLocation(inputTrip.NextDestinationLocationID)

	//Get the product ID
	productID := getProductID(startLocation, endLocation)
	fmt.Println("Product ID obtained is : ", productID)

	var requestIDJSON UberSandboxRequestIDJSON
	requestIDJSON.StartLatitude = startLocation.Coordinate.Lat
	requestIDJSON.StartLongitude = startLocation.Coordinate.Lng
	requestIDJSON.EndLatitude = endLocation.Coordinate.Lat
	requestIDJSON.EndLongitude = endLocation.Coordinate.Lng
	requestIDJSON.ProductID = productID

	jsonStr, err := json.Marshal(requestIDJSON)
	req, err := http.NewRequest("POST", apiurl, bytes.NewBuffer(jsonStr))
	if err != nil {
		fmt.Println("error: body, _ := ioutil.ReadAll(resp.Body) -- line 582")
		panic(err.Error())
	}

	req.Header.Set("Authorization", "Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzY29wZXMiOlsicmVxdWVzdCJdLCJzdWIiOiIzMGQ2ZWUwOC0xMjA0LTQ5MzQtOGExNy00YmY3YzdhZTdmODMiLCJpc3MiOiJ1YmVyLXVzMSIsImp0aSI6IjY2YzAxNjZjLTQwOWQtNDk4OC04YmNmLTFhYzU4MjI5Nzc0ZCIsImV4cCI6MTQ1MDE2NjM2MiwiaWF0IjoxNDQ3NTc0MzYyLCJ1YWN0IjoiSXlCeXJobHhUTXlNU1RXNnVRU25XNGRGd3dxNlFrIiwibmJmIjoxNDQ3NTc0MjcyLCJhdWQiOiJwbkdDSXFNVVN3NHNoa0VyeG9LaFA2ZFlhRERkTkQtdiJ9.b6bJYAInt3Zd_Qm-aHbhBQ-fLFPBKHzRvPCuiOBYsJ2gXmOAwsobTfF-MvNSls3eZOzHeClJ7J-OUijJ2wj3dALiWV2WvvfT3uUYfnMxRnKzBETzWya9vqkk10IsMZ-2N6bGoEkEhcLsPkL0pd7BShM0E3NnwGVTK6cgjWjAF5lE1lrs3b-uPreplGZVPg8GmDRcNGmWEpkdLEKW6hvmRRZPwJVlG9BFJU7aI0O6OChIHFRac1yLJwg5tGyw0495BRR8FeyDmaKzvBijH274X7AK0OcMnvIsqsVNmDhNXswYQlwNjQ9ksUGGefVuEZ_MTsYmBALzC9j1GMRrhx7_Ng")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("error: body, _ := ioutil.ReadAll(resp.Body) -- line 592")
		panic(err)
	}
	defer resp.Body.Close()
	var sandboxResponse UberSandBoxRequestResponse

	body, _ := ioutil.ReadAll(resp.Body)
	if err != nil {

		fmt.Println("error: body, _ := ioutil.ReadAll(resp.Body) -- line 599")
		panic(err.Error())
	}

	_ = json.Unmarshal(body, &sandboxResponse)
	if err != nil {

		fmt.Println("error: Unable to unmarshal SAndbox response data. -- line 610")
		panic(err.Error())
	}

	body, _ = ioutil.ReadAll(resp.Body)
	if err != nil {

		fmt.Println("error: body, _ := ioutil.ReadAll(resp.Body) -- line 599")
		panic(err.Error())
	}

	_ = json.Unmarshal(body, &sandboxResponse)
	if err != nil {

		fmt.Println("error: Unable to unmarshal SAndbox response data. -- line 610")
		panic(err.Error())

	}

	fmt.Println("Request ID : ", sandboxResponse.RequestID)
	fmt.Println("Coordinates : (", requestIDJSON.StartLatitude, ",", requestIDJSON.StartLongitude, "), (", requestIDJSON.EndLatitude, ",", requestIDJSON.EndLongitude, ")")
	fmt.Println("ETA : ", sandboxResponse.Eta)

	inputTrip.UberWaitTimeEta = sandboxResponse.Eta

}

func getProductID(startLocation locationStruct, endLocation locationStruct) string {

	_, _, _, productID := getUberCost(startLocation, endLocation)
	return productID
}

func obtainLocation(locationID string) locationStruct {
	var outputLocation locationStruct
	c, s := getMongoCollection("addresses")
	defer s.Close()
	err := c.Find(bson.M{"_id": bson.ObjectIdHex(locationID)}).One(&outputLocation)
	if err != nil {
		fmt.Println("err = c.Find(bson.M{\"id\": bson.M{\"$oid\": t}}).One(&result)")
		log.Fatal(err)
	}
	return outputLocation
}

func main() {
	mux := routes.New()
	mux.Post("/locations/", addLocation)
	mux.Get("/locations/:locationID", findLocation)
	mux.Put("/locations/:locationID", updateLocation)
	mux.Del("/locations/:locationID", deleteLocation)

	mux.Post("/trips/", planTrip)
	mux.Put("/trips/:tripID/request", requestTrip)
	mux.Get("/trips/:tripID", getTripDetails)

	http.Handle("/", mux)
	http.ListenAndServe(":8088", nil)
}
