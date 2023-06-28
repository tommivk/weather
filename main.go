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
	"text/tabwriter"

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

func fetchLocationData(city, country string) (GeoResponse, error) {
	var GEO_URL = fmt.Sprintf("http://api.openweathermap.org/geo/1.0/direct?q=%s,%s&limit=1&appid=%s", city, country, API_KEY)

	data, err := fetchData(GEO_URL)
	if err != nil {
		return GeoResponse{}, err
	}

	var result []GeoResponse
	json.Unmarshal(data, &result)
	if len(result) == 0 {
		return GeoResponse{}, errors.New("Location not found")
	}

	return result[0], nil
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
	fmt.Printf("\nWeather in %s, %s: \n\n", result.City, result.Country)
	fmt.Printf("%s \nTemperature: %f ℃ \nFeels like: %f ℃ \n\n", strings.Title(result.Description), result.Temperature, result.FeelsLike)
	fmt.Println("--------------------------------------------------------")
}

func fetchFavourites() error {
	if len(config.Favourites) == 0 {
		fmt.Println("No favourites added")
	}
	for i := 0; i < len(config.Favourites); i++ {
		weather, err := fetchWeather(config.Favourites[i].Coordinates)
		if err != nil {
			return err
		}
		printResult(weather)
	}
	return nil
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

func listFavourites() {
	favourites := config.Favourites
	if len(favourites) == 0 {
		fmt.Println("No favourites added")
	}
	fmt.Println("\n------Favourites------")
	for i := 0; i < len(favourites); i++ {
		fmt.Printf("\n%s, %s", favourites[i].City, favourites[i].Country)
	}
	fmt.Print("\n\n----------------------\n\n")
}

func removeFavourite(city string) error {
	favourites := config.Favourites
	var index int = -1
	for i := 0; i < len(favourites); i++ {
		if strings.ToLower(favourites[i].City) == city {
			index = i
			break
		}
	}
	if index == -1 {
		return fmt.Errorf("City %s does not exist in favourites", city)
	}
	config.Favourites = append(config.Favourites[:index], config.Favourites[index+1:]...)
	configBytes, err := json.Marshal(config)
	if err != nil {
		return err
	}
	err = saveConfig(configBytes)
	if err != nil {
		return err
	}
	fmt.Printf("City %s successfully removed from favourites\n", city)
	return nil
}

func addFavourite(city, country string) error {
	for i := 0; i < len(config.Favourites); i++ {
		c := config.Favourites[i].City
		if strings.ToLower(c) == strings.ToLower(city) {
			return fmt.Errorf("%s already exists in favourites", city)
		}
	}

	locationData, err := fetchLocationData(city, country)
	if err != nil {
		return err
	}
	coordinates := Coordinates{Lat: locationData.Lat, Lon: locationData.Lon}
	newFavourites := append(config.Favourites, Location{City: locationData.Name, Country: locationData.Country, Coordinates: coordinates})
	config.Favourites = newFavourites
	configBytes, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("Failed to marshal data: %s", err)
	}
	err = saveConfig(configBytes)
	if err != nil {
		return fmt.Errorf("Failed to save config file: %s", err)
	}
	fmt.Printf("New location %s, %s added to favourites\n", locationData.Name, locationData.Country)
	return nil
}

func getWeatherByCity(city, country string) error {
	data, err := fetchLocationData(city, country)
	if err != nil {
		return err
	}
	coordinates := Coordinates{Lat: data.Lat, Lon: data.Lon}
	res, err := fetchWeather(coordinates)
	if err != nil {
		return err
	}
	printResult(res)
	return nil
}

func printCommands() {
	w := tabwriter.NewWriter(os.Stdout, 1, 1, 1, ' ', 0)
	fmt.Fprint(w, "\n-------Commands----------------------------------\n\n")
	fmt.Fprintln(w, "w\t<City>\t[<Country>]\t|\tGet weather by city")
	fmt.Fprintln(w, "f\t\t\t|\tGet weather for all of the cities in your favourites")
	fmt.Fprintln(w, "list\t\t\t|\tList favourites")
	fmt.Fprintln(w, "fav\t<City>\t[<Country>]\t|\tAdd city to favourites")
	fmt.Fprintln(w, "remove\t<City>\t\t|\tRemove city from favourites")
	fmt.Fprintln(w, "help\t\t\t|\tList available commands")
	fmt.Fprintln(w, "\n---------------------------------------------")
}

func main() {
	err := readConfigFile()
	if err != nil {
		log.Fatal(err)
	}

	printCommands()

	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("\nCommand: ")

		if scanner.Scan() {
			input := strings.Fields(scanner.Text())
			if len(input) == 0 {
				continue
			}

			command := strings.ToLower(input[0])

			var city, country string = "", ""

			if len(input) > 1 {
				city = strings.ToLower(input[1])
			}
			if len(input) > 2 {
				country = strings.ToLower(input[2])
			}
			fmt.Println()

			switch command {
			case "w":
				if len(input) < 2 {
					fmt.Println("Missing city parameter")
					continue
				}
				err := getWeatherByCity(city, country)
				if err != nil {
					fmt.Println(err)
				}
			case "f":
				err := fetchFavourites()
				if err != nil {
					fmt.Println(err)
				}
			case "list":
				listFavourites()
			case "fav":
				if len(input) < 2 {
					fmt.Println("Missing city parameter")
					continue
				}
				err := addFavourite(city, country)
				if err != nil {
					fmt.Println(err)
				}
			case "remove":
				err := removeFavourite(city)
				if err != nil {
					fmt.Println(err)
				}
			case "help":
				printCommands()
			default:
				fmt.Println("Unknown command")
			}

		}
		if err := scanner.Err(); err != nil {
			log.Fatal("Failed to scan input", scanner.Err())
		}
	}

}
