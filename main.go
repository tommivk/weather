package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	_ "github.com/joho/godotenv/autoload"
)

var API_KEY = os.Getenv("API_KEY")

func fetchData(url string) ([]byte, error) {
	res, err := http.Get(url)
	if err != nil {
		return nil, errors.New("Failed to fetch")
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, errors.New("Failed to read response")
	}
	return body, nil
}

type GeoResponse struct {
	Name    string
	Lat     float32
	Lon     float32
	Country string
}

type Coordinates struct {
	Lat float32
	Lon float32
}

func fetchCoordinates(city, country string) (Coordinates, error) {
	var GEO_URL = fmt.Sprintf("http://api.openweathermap.org/geo/1.0/direct?q=%s,%s&limit=1&appid=%s", city, country, API_KEY)

	data, err := fetchData(GEO_URL)
	if err != nil {
		return Coordinates{}, err
	}

	var result []GeoResponse
	json.Unmarshal(data, &result)
	if len(result) == 0 {
		return Coordinates{}, errors.New("Location not found")
	}

	coordinates := Coordinates{Lat: result[0].Lat, Lon: result[0].Lon}
	return coordinates, err
}

type WeatherDetails struct {
	Description string
}

type Temperatures struct {
	Temp      float32
	FeelsLike float32 `json:"feels_like"`
}

type LocationDetails struct {
	Country string
}

type WeatherResponse struct {
	WeatherDetails  []WeatherDetails `json:"weather"`
	Temperatures    Temperatures     `json:"main"`
	City            string           `json:"name"`
	LocationDetails LocationDetails  `json:"sys"`
}

type WeatherResult struct {
	Temperature float32
	FeelsLike   float32
	Description string
	Country     string
	City        string
}

func fetchWeather(coordinates Coordinates) (WeatherResult, error) {
	url := fmt.Sprintf("https://api.openweathermap.org/data/2.5/weather?lat=%f&lon=%f&units=metric&lang=fi&appid=%s", coordinates.Lat, coordinates.Lon, API_KEY)
	data, err := fetchData(url)
	if err != nil {
		return WeatherResult{}, err
	}

	var result WeatherResponse
	json.Unmarshal(data, &result)

	if len(result.WeatherDetails) == 0 {
		return WeatherResult{}, errors.New("Not found")
	}

	res := WeatherResult{
		Temperature: result.Temperatures.Temp,
		FeelsLike:   result.Temperatures.FeelsLike,
		Description: result.WeatherDetails[0].Description,
		Country:     result.LocationDetails.Country,
		City:        result.City,
	}

	return res, nil
}

func printResult(result WeatherResult) {
	fmt.Println("--------------------------------------------------------")
	fmt.Printf("\nWeather in %s, %s: \n\n", result.City, result.Country)
	fmt.Printf("%s \nTemperature: %f ℃ \nFeels like: %f ℃ \n\n", strings.Title(result.Description), result.Temperature, result.FeelsLike)
	fmt.Println("--------------------------------------------------------")
}

func main() {
	var city string
	var country string

	scanner := bufio.NewScanner(os.Stdin)
	for {
		if scanner.Scan() {
			input := strings.Fields(scanner.Text())
			if len(input) != 2 {
				fmt.Println("Invalid input")
				continue
			}
			city = input[0]
			country = input[1]
			break
		}
		log.Fatal("Failed to scan input")
	}

	coordinates, err := fetchCoordinates(city, country)
	if err != nil {
		log.Fatal(err)
	}

	res, err := fetchWeather(coordinates)
	if err != nil {
		log.Fatal(err)
	}
	printResult(res)
}
