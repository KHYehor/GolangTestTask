package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"time"
)

type Position struct {
	LineOffset int `json:"lineOffset"`
	CharOffset int `json:"charOffset"`
}

type Words map[string][]Position

type WordsFinder struct {
	mu sync.Mutex
	wg sync.WaitGroup

	dataFileName   string
	filterFileName string

	wordsToFind []string
	allWords    []Words
	result      Words
}

func (wf *WordsFinder) push(names *Words) {
	wf.mu.Lock()
	defer wf.mu.Unlock()
	wf.allWords = append(wf.allWords, *names)
}

func (wf *WordsFinder) match(text string, lineOffset int, charsOffset int) {
	defer wf.wg.Done()

	names := Words{}

	for _, name := range wf.wordsToFind {
		start := 0
		for {
			position := strings.Index(strings.ToLower(text[start:]), strings.ToLower(name))
			if position == -1 {
				break
			}
			absolutePosition := start + position
			pos := Position{lineOffset, charsOffset + absolutePosition}
			names[name] = append(names[name], pos)

			start = absolutePosition + len(name)
		}
	}
	wf.push(&names)
}

func (wf *WordsFinder) aggregate() {
	aggregated := &Words{}
	for _, names := range wf.allWords {
		for name, positions := range names {
			for _, position := range positions {
				(*aggregated)[name] = append((*aggregated)[name], position)
			}
		}
	}
	wf.result = *aggregated
}

func (wf *WordsFinder) uploadFilters() {
	file, err := os.Open(wf.filterFileName)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	fileContent, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Println("Error reading file:", err)
		return
	}

	// Unmarshal the JSON data into the slice
	err = json.Unmarshal(fileContent, &wf.wordsToFind)
	if err != nil {
		fmt.Println("Error unmarshaling JSON:", err)
		return
	}
}

func (wf *WordsFinder) saveAsJson() {
	file, err := os.Create("./data/result.json")
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}
	defer file.Close()

	jsonData, err := json.MarshalIndent(wf.result, "", "  ")
	if err != nil {
		fmt.Println("Error marshaling data:", err)
		return
	}

	_, err = file.Write(jsonData)
	if err != nil {
		fmt.Println("Error writing to file:", err)
		return
	}
}

func (wf *WordsFinder) process(fileName string, filterFileName string) {
	wf.dataFileName = fileName
	wf.filterFileName = filterFileName

	// Upload words for lookup
	wf.uploadFilters()

	// Start tracking the main task
	start := time.Now()

	// Open file with the data
	readFile, err := os.Open(wf.dataFileName)
	if err != nil {
		fmt.Println(err)
	}
	defer readFile.Close()

	// Init file scanning line by line
	fileScanner := bufio.NewScanner(readFile)
	fileScanner.Split(bufio.ScanLines)

	var linesCount = 0
	var charsCount = 0

	var text = ""
	for fileScanner.Scan() {
		text = fileScanner.Text()
		wf.wg.Add(1)
		// Start finding related data
		go wf.match(text, linesCount, charsCount)

		linesCount++
		// +1 char because of newline char "\n"
		charsCount += len(text) + 1
		text = ""
	}

	// Wait until all goroutines finish processing
	wf.wg.Wait()

	// Compose all data into one structure
	wf.aggregate()

	// Stop tracking the main task
	end := time.Now()
	fmt.Println(wf.result)
	fmt.Printf("All data are found in %v\n", end.Sub(start))

	// Save result to the json file
	wf.saveAsJson()
}
