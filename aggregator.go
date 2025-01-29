package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
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

	wordsToFind []string
	allWords    []Words
	result      Words
}

func (wf *WordsFinder) push(names *Words) {
	wf.mu.Lock()
	defer wf.mu.Unlock()
	// Safely push data into the array with mutex lock
	wf.allWords = append(wf.allWords, *names)
}

func (wf *WordsFinder) match(text string, linesOffset int, charsOffset int) {
	defer wf.wg.Done()

	names := Words{}

	// Start iterating with each word
	for _, name := range wf.wordsToFind {
		start := 0
		for {
			// Find the position of the word to begin
			position := strings.Index(strings.ToLower(text[start:]), strings.ToLower(name))
			if position == -1 {
				break
			}
			// The count of chars before the found word
			localPosition := start + position
			// Create position
			pos := Position{linesOffset, charsOffset + localPosition}
			// Save the position
			names[name] = append(names[name], pos)

			// Start from the new point
			start = localPosition + len(name)
		}
	}
	// Save the result to global array
	wf.push(&names)
}

func (wf *WordsFinder) aggregate() {
	aggregated := &Words{}
	// Collect result from every separate struct into the one
	for _, names := range wf.allWords {
		for name, positions := range names {
			for _, position := range positions {
				(*aggregated)[name] = append((*aggregated)[name], position)
			}
		}
	}
	wf.result = *aggregated
}

func (wf *WordsFinder) uploadFilters(filterFileName string) {
	// Upload data filters to the memory
	file, err := os.Open(filterFileName)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	fileContent, err := io.ReadAll(file)
	if err != nil {
		fmt.Println("Error reading file:", err)
		return
	}

	// Unmarshal the JSON data into the slice
	err = json.Unmarshal(fileContent, &wf.wordsToFind)
	if err != nil {
		fmt.Println("Error unmarshalling JSON:", err)
		return
	}
}

func (wf *WordsFinder) saveAsJson() {
	// Save the result to the json file
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
	// Upload words for lookup
	wf.uploadFilters(filterFileName)

	// Start tracking the main task
	start := time.Now()

	// Open file with the data
	readFile, err := os.Open(fileName)
	if err != nil {
		fmt.Println(err)
	}
	defer readFile.Close()

	// Init file scanning line by line
	fileScanner := bufio.NewScanner(readFile)
	fileScanner.Split(bufio.ScanLines)

	var linesCount = 1
	var charsCount = 0
	var text = ""

	for fileScanner.Scan() {
		text = fileScanner.Text()
		wf.wg.Add(1)
		// Start finding related data
		go wf.match(text, linesCount, charsCount)

		linesCount++
		charsCount += len(text)
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
