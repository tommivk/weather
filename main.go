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

var config Config

type Config struct {
	Language   string
	Units      string
	Favourites []Location
}

type Location struct {
	City        string
	Country     string
	Coordinates Coordinates
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

func fetchData(url string) ([]byte, error) {
	res, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("Failed to fetch data: %s", err)
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("Failed to read response: %s", err)
	}
	return body, nil
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
	return coordinates, nil
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
	url := fmt.Sprintf("https://api.openweathermap.org/data/2.5/weather?lat=%f&lon=%f&units=metric&lang=%s&appid=%s", coordinates.Lat, coordinates.Lon, config.Language, API_KEY)
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

func fetchFavourites() {
	if len(config.Favourites) == 0 {
		fmt.Println("No favourites added")
	}
	for i := 0; i < len(config.Favourites); i++ {
		weather, err := fetchWeather(config.Favourites[i].Coordinates)
		if err != nil {
			log.Fatal(nil)
		}
		printResult(weather)
	}
}

func readConfigFile() error {
	data, err := os.ReadFile("config.json")
	if os.IsNotExist(err) {
		fmt.Println("Config file missing")
		err = createNewConfigFile()
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	json.Unmarshal(data, &config)
	return nil
}

func createNewConfigFile() error {
	file, err := json.Marshal(Config{Units: "metric", Language: "en", Favourites: []Location{}})
	if err != nil {
		return err
	}
	err = saveConfig(file)
	if err != nil {
		return err
	}
	fmt.Println("New Config file created")
	return nil
}

func saveConfig(config []byte) error {
	err := os.WriteFile("config.json", config, 0666)
	if err != nil {
		return err
	}
	return nil
}

func addFavourite(city, country string) {
	for i := 0; i < len(config.Favourites); i++ {
		c := config.Favourites[i].City
		if strings.ToLower(c) == strings.ToLower(city) {
			fmt.Printf("%s already exists in favourites\n", city)
			return
		}
	}

	coordinates, err := fetchCoordinates(city, country)
	if err != nil {
		log.Fatal(err)
	}

	newFavourites := append(config.Favourites, Location{City: city, Country: country, Coordinates: coordinates})
	config.Favourites = newFavourites
	configBytes, err := json.Marshal(config)
	if err != nil {
		log.Fatal("Failed to marshal data: ", err)
	}
	err = saveConfig(configBytes)
	if err != nil {
		log.Fatal("Failed to save config file: ", err)
	}
	fmt.Printf("New location %s %s added\n", city, country)
}

func getWeatherByCity(city, country string) {
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

func main() {
	err := readConfigFile()
	if err != nil {
		log.Fatal(err)
	}

	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Println("Get weather by City")
		fmt.Print("City: ")
		var city string = ""
		var country string = ""

		if scanner.Scan() {
			input := strings.Fields(scanner.Text())
			city = input[0]
			if len(input) > 1 {
				country = input[1]
			}
			getWeatherByCity(city, country)
			continue
		}
		if err := scanner.Err(); err != nil {
			log.Fatal("Failed to scan input", scanner.Err())
		}
	}

}
