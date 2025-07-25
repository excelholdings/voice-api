package classifier

import (
	"bufio"
	"encoding/json"
	"github.com/flyflow-devs/flyflow/internal/logger"
	"github.com/n3integration/classifier/knn"
	"github.com/n3integration/classifier/naive"
	"log"
	"os"
	"strings"
)

type Classifier struct {
	classifier *naive.Classifier
	classifier2 *knn.Classifier
}


func NewClassifier() *Classifier {
	n, k := trainClassifier()
	return &Classifier{
		classifier: n,
		classifier2: k,
	}
}

func (c *Classifier) GetKNNWord(text string) (string, error) {
	return c.classifier2.ClassifyString(text)
}

func (c *Classifier) GetProbabilities(text string) (map[string]float64, string) {
	return c.classifier.Probabilities(text)
}

func (c *Classifier) GetFillerWord(userMessage string, whitelist []string, previousFillerWord string) string {
	// Split the userMessage into words
	words := strings.Fields(userMessage)

	// Check if the number of words is less than or equal to 4
	if len(words) <= 4 {
		return ""
	}

	probabilities, result := c.classifier.Probabilities(userMessage)
	if len(whitelist) == 0 {
		return result
	} else {
		best := 0.0
		toReturn := ""
		for word, prob := range probabilities {
			if contains(word, whitelist) && prob > best && word != previousFillerWord {
				toReturn = word
				best = prob
			}
		}
		return toReturn
	}

}


func contains(word string, list []string) bool {
	// Iterate over the words to check if any matches the given word
	for _, w := range list {
		if w == word {
			return true
		}
	}
	return false
}


type TrainingData struct {
	User     string `json:"user"`
	Assistant string `json:"assistant"`
	FillerWord string `json:"filler_word"`
}

func trainClassifier() (*naive.Classifier, *knn.Classifier) {
	classifier := naive.New()
	classifier2 := knn.New()

	file, err := os.Open("internal/classifier/data/filler_words.jsonl")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		var example TrainingData
		err := json.Unmarshal([]byte(line), &example)
		if err != nil {
			log.Printf("Error unmarshaling JSON: %v", err)
			continue
		}

		// Train the classifier with the user input and filler word
		if err := classifier.TrainString(example.User, example.FillerWord); err != nil {
			logger.S.Fatal(err)
		}
		if err := classifier2.TrainString(example.User, example.FillerWord); err != nil {
			logger.S.Fatal(err)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	return classifier, classifier2
}
