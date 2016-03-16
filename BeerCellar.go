package main

import "bufio"
import "errors"
import "flag"
import "fmt"
import "log"
import "os"
import "strconv"
import "strings"
import "time"

// BeerCellar the overall beer cellar
type BeerCellar struct {
	version       string
	name          string
	syncTime      string
	untappdKey    string
	untappdSecret string
	bcellar       []Cellar
}

// Sync syncs with untappd
func (cellar *BeerCellar) Sync(fetcher httpResponseFetcher, converter responseConverter) {
	drunk := GetRecentDrinks(fetcher, converter, cellar.syncTime)
	for _, val := range drunk {
		log.Printf("Removing %v from cellar\n", val)
		cellar.RemoveBeer(val)
	}

	cellar.syncTime = time.Now().Format("02/01/06")
}

// RemoveBeer removes a beer from the cellar
func (cellar *BeerCellar) RemoveBeer(id int) {
	cellarIndex := -1
	cellarCost := -1
	for i, c := range cellar.bcellar {
		cost := c.GetRemoveCost(id)
		if cost >= 0 {
			if cellarIndex < 0 || cost < cellarCost {
				cellarIndex = i
				cellarCost = cost
			}
		}
	}

	if cellarIndex >= 0 {
		log.Printf("Removing %v from %v\n", id, cellarIndex)
		cellar.bcellar[cellarIndex].Remove(id)
	}
}

// CountBeers returns the number of beers of a given id in the cellar
func (cellar *BeerCellar) CountBeers(id int) int {
	sum := 0
	for _, v := range cellar.bcellar {
		sum += v.CountBeersInCellar(id)
	}
	return sum
}

// SetUntappd sets the untappd key, secret pair
func (cellar *BeerCellar) SetUntappd(key string, secret string) {
	cellar.untappdKey = key
	cellar.untappdSecret = secret
}

// GetUntappd Gets the untappd key,secret pair
func (cellar BeerCellar) GetUntappd() (string, string) {
	return cellar.untappdKey, cellar.untappdSecret
}

// PrintCellar prints out the cellar
func (cellar BeerCellar) PrintCellar(printer Printer) {
	for i, v := range cellar.bcellar {
		if i > 0 {
			printer.Println("--------------")
		}
		v.PrintCellar(printer)
	}
}

// GetNumberOfCellars gets the number of cellar boxes in the cellar
func (cellar BeerCellar) GetNumberOfCellars() int {
	return len(cellar.bcellar)
}

// Save stores a cellar to disk
func (cellar BeerCellar) Save() {
	for _, v := range cellar.bcellar {
		v.Save()
	}

	//Also save the metadata
	fileName := cellar.name + ".metadata"
	f, err := os.Create(fileName)
	if err != nil {
		log.Printf("Error saving metadata file: %v\n", fileName)
		return
	}

	defer f.Close()
	fmt.Fprintf(f, "%v~%v~%v~%v~%v\n", cellar.version, cellar.name, cellar.syncTime, cellar.untappdKey, cellar.untappdSecret)
}

// Size gets the size of the cellar
func (cellar BeerCellar) Size() int {
	size := 0
	for _, v := range cellar.bcellar {
		size += v.Size()
	}
	return size
}

// AddBeerToCellar Adds a beer to the cellar
func (cellar BeerCellar) AddBeerToCellar(beer Beer) Cellar {
	bestCellar := -1
	bestScore := -1

	for i, v := range cellar.bcellar {
		insertCount := v.ComputeInsertCost(beer)

		log.Printf("Adding beer to cellar %v: %v\n", i, insertCount)

		if insertCount >= 0 && (insertCount < bestScore || bestScore < 0) {
			bestScore = insertCount
			bestCellar = i
		}
	}

	cellar.bcellar[bestCellar].AddBeer(beer)
	return cellar.bcellar[bestCellar]
}

// GetEmptyCellarCount Gets the number of empty cellars
func (cellar BeerCellar) GetEmptyCellarCount() int {
	count := 0
	for _, v := range cellar.bcellar {
		if v.Size() == 0 {
			count++
		}
	}
	return count
}

// LoadBeerCellar loads a set of beer cellar files
func LoadBeerCellar(name string) (*BeerCellar, error) {

	bc := BeerCellar{
		version: "0.1",
		name:    name,
		bcellar: make([]Cellar, 0),
	}

	for i := 1; i < 9; i++ {
		tcellar := BuildCellar(name + strconv.Itoa(i) + ".cellar")
		if tcellar != nil {
			bc.bcellar = append(bc.bcellar, *tcellar)
		}
	}

	// Load in the metadata
	fileName := name + ".metadata"
	file, err := os.Open(fileName)
	if err != nil {
		log.Printf("Error opening file: %v - %v\n", fileName, err)
		return &bc, errors.New("Cannot open metadata file")
	}

	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		elems := strings.Split(line, "~")
		if len(elems) == 5 {
			bc.version = elems[0]
			bc.name = elems[1]
			bc.syncTime = elems[2]
			bc.untappdKey = elems[3]
			bc.untappdSecret = elems[4]
		}
	}

	return &bc, nil
}

// NewBeerCellar creates new beer cellar
func NewBeerCellar(name string) *BeerCellar {
	bc := BeerCellar{
		version:  "0.1",
		name:     name,
		syncTime: time.Now().Format("02/01/06"),
		bcellar:  make([]Cellar, 0),
	}

	for i := 1; i < 9; i++ {
		bc.bcellar = append(bc.bcellar, NewCellar(name+strconv.Itoa(i)+".cellar"))
	}

	return &bc
}

// LoadOrNewBeerCellar loads or creates a new BeerCellar
func LoadOrNewBeerCellar(name string) (*BeerCellar, error) {
	if _, err := os.Stat(name + ".metadata"); err == nil {
		return LoadBeerCellar(name)
	}

	return NewBeerCellar(name), nil
}

// GetVersion gets the version of the cellar code
func (cellar *BeerCellar) GetVersion() string {
	return cellar.version
}

// AddBeerByDays adds beers by days to the cellar.
func (cellar *BeerCellar) AddBeerByDays(id string, date string, size string, days string, count string) {
	startDate, _ := time.Parse("02/01/06", date)
	countVal, _ := strconv.Atoi(count)
	daysVal, _ := strconv.Atoi(days)
	for i := 0; i < countVal; i++ {
		cellar.AddBeer(id, startDate.Format("02/01/06"), size)
		startDate = startDate.AddDate(0, 0, daysVal)
	}
}

// AddBeer adds the beer to the cellar
func (cellar *BeerCellar) AddBeer(id string, date string, size string) *Cellar {
	idNum, _ := strconv.Atoi(id)
	if idNum >= 0 {
		cellarBox := cellar.AddBeerToCellar(Beer{id: idNum, drinkDate: date, size: size})
		return &cellarBox
	}

	return nil
}

// Min returns the min of the parameters
func Min(a int, b int) int {
	if a > b {
		return b
	}

	return a
}

// ListBeers lists the cellared beers of a given type
func (cellar *BeerCellar) ListBeers(num int, btype string) []Beer {
	log.Printf("Cellar looks like %v\n", cellar.bcellar)
	retList := MergeCellars(btype, cellar.bcellar...)
	return retList[:Min(len(retList), num)]
}

// PrintBeers prints out the beers of a given type
func (cellar *BeerCellar) PrintBeers(numBombers int, numSmall int) {
	bombers := cellar.ListBeers(numBombers, "bomber")
	smalls := cellar.ListBeers(numSmall, "small")

	fmt.Printf("Bombers\n")
	fmt.Printf("-------\n")
	for _, v := range bombers {
		fmt.Printf("%v\n", v)
	}

	fmt.Printf("Smalls\n")
	fmt.Printf("-------\n")
	for _, v := range smalls {
		fmt.Printf("%v\n", v)
	}

}

func runSearch(command string, flags *flag.FlagSet, search string) {
	if command == "search" {
		if flags.Parsed() {
			matches := Search(search)
			for _, match := range matches {
				fmt.Printf("%v: %v\n", match.name, match.id)
			}
		} else {
			flags.PrintDefaults()
		}
	}
}

func runVersion(command string, cellar *BeerCellar) {
	if command == "version" {
		fmt.Printf("BeerCellar: %q\n", cellar.GetVersion())
	}
}

func runAddBeer(command string, flags *flag.FlagSet, id string, date string, size string, days string, count string, cellar *BeerCellar) {
	if command == "add" {
		if flags.Parsed() {
			if days != "" {
				cellar.AddBeerByDays(id, date, size, days, count)
				cellar.PrintCellar(&StdOutPrint{})
			} else {
				box := cellar.AddBeer(id, date, size)
				print := &StdOutPrint{}
				box.PrintCellar(print)
			}
		} else {
			flags.PrintDefaults()
		}
	}
}

func runPrintCellar(command string, cellar *BeerCellar) {
	if command == "print" {
		cellar.PrintCellar(&StdOutPrint{})
	}
}

func runSync(command string, cellar *BeerCellar) {
	if command == "sync" {
		cellar.Sync(mainFetcher{}, mainConverter{})
	}
}

func runRemoveBeer(command string, flags *flag.FlagSet, id int, cellar *BeerCellar) {
	if command == "remove" {
		if flags.Parsed() {
			cellar.RemoveBeer(id)
			cellar.PrintCellar(&StdOutPrint{})
		}
	}
}

func runListBeers(command string, flags *flag.FlagSet, numBombers int, numSmall int, cellar *BeerCellar) {
	if command == "list" {
		if flags.Parsed() {
			cellar.PrintBeers(numBombers, numSmall)
		} else {
			flags.PrintDefaults()
		}
	}
}

func runSaveUntappd(command string, flags *flag.FlagSet, key string, secret string, cellar *BeerCellar) {
	if command == "untappd" {
		if flags.Parsed() {
			cellar.SetUntappd(key, secret)
			untappdKey = key
			untappdSecret = secret
		} else {
			flags.PrintDefaults()
		}
	} else {
		if cellar.untappdKey == "" {
			//Set from environment variables
			untappdKey = os.Getenv("CLIENTID")
			untappdSecret = os.Getenv("CLIENTSECRET")
		} else {
			untappdKey = cellar.untappdKey
			untappdSecret = cellar.untappdSecret
		}
	}
}

func main() {
	var beerid string
	var drinkDate string
	var size string
	var days string
	var count string
	addBeerFlags := flag.NewFlagSet("add", flag.ContinueOnError)
	addBeerFlags.StringVar(&beerid, "id", "-1", "ID of the beer")
	addBeerFlags.StringVar(&drinkDate, "date", "", "Date to be drunk by")
	addBeerFlags.StringVar(&size, "size", "", "Size of bottle")
	addBeerFlags.StringVar(&days, "days", "", "Number of separate days")
	addBeerFlags.StringVar(&count, "count", "", "Number of bottles")

	var numBombers int
	var numSmall int
	listBeerFlags := flag.NewFlagSet("list", flag.ContinueOnError)
	listBeerFlags.IntVar(&numBombers, "bombers", 0, "Number of bombers to list from the cellar")
	listBeerFlags.IntVar(&numSmall, "small", 0, "Number of small beers to list from the cellar")

	var key string
	var secret string
	saveUntappdFlags := flag.NewFlagSet("untappd", flag.ContinueOnError)
	saveUntappdFlags.StringVar(&key, "key", "", "Key for untappd")
	saveUntappdFlags.StringVar(&secret, "secret", "", "Secret for untappd")

	var search string
	searchFlags := flag.NewFlagSet("search", flag.ContinueOnError)
	searchFlags.StringVar(&search, "string", "", "String to search for")

	var removeID int
	removeFlags := flag.NewFlagSet("remove", flag.ContinueOnError)
	removeFlags.IntVar(&removeID, "id", 0, "The ID of the beer to be removed")

	cellarName := "prod"

	addBeerFlags.Parse(os.Args[2:])
	listBeerFlags.Parse(os.Args[2:])
	saveUntappdFlags.Parse(os.Args[2:])
	searchFlags.Parse(os.Args[2:])
	cellar, _ := LoadOrNewBeerCellar(cellarName)
	LoadCache("prod_cache")

	runSaveUntappd(os.Args[1], saveUntappdFlags, key, secret, cellar)
	runVersion(os.Args[1], cellar)
	runAddBeer(os.Args[1], addBeerFlags, beerid, drinkDate, size, days, count, cellar)
	runPrintCellar(os.Args[1], cellar)
	runListBeers(os.Args[1], listBeerFlags, numBombers, numSmall, cellar)
	runSearch(os.Args[1], searchFlags, search)
	runRemoveBeer(os.Args[1], removeFlags, removeID, cellar)
	runSync(os.Args[1], cellar)
	cellar.Save()
	SaveCache("prod_cache")
}
