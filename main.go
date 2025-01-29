package main

func main() {
	wf := &WordsFinder{}

	wf.process("./data/data.txt", "./data/filter-words.json")
}
