package chain

import (
	"fmt"
	"math/rand"
	"sync"
)

// MarkovChainLink wraps a the probabilities for a
// single 'link' in a Markov chain
type MarkovChainLink interface {
	// GetNextToken calculated a probabilistic next token
	GetNextToken(rand *rand.Rand) string

	// RetrieveNextTokenPossibilities retrieves all token possibilities from the specified token
	RetrieveNextTokenPossibilities() (nextTokens []string)

	// GetProbabilityOfToken calculates the probability of a given token following
	// the supplied key token, and a boolean indicating if the key token was present
	GetProbabilityOfToken(nextToken string) (nextTokenProbability float64, tokenPresent bool)
}

// MarkovChain wraps a set of links probabilities to make a full
// Markov chain
type MarkovChain interface {
	// CalculateNextToken calculates the next token based on the chain's probabilities
	// it returns the next token, and a boolean indicating if the key was present
	CalculateNextToken(token string, rand *rand.Rand) (nextToken string, keyPresent bool)

	// RetrieveMarkovLink retrieves all token possibilities from the specified
	// token, returns false if the token was not found
	RetrieveMarkovLink(token string) (link MarkovChainLink, keyPresent bool)
}

type singleTokenLink struct {
	Token                [1]string      `json:"token",xml:"token"`
	NextTokenOccurrences map[string]int `json:"next_token_occurrences",xml:"nextTokenOccurrences"`
	Total                int            `json:"total",xml:"total"`
}

func (l *singleTokenLink) String() string {
	return fmt.Sprintf("%v", *l)
}

type singleKeyChain struct {
	Links map[string]*singleTokenLink `json:"links"`
}

func (c *singleKeyChain) CalculateNextToken(token string, rand *rand.Rand) (nextToken string, keyPresent bool) {
	if link, ok := c.Links[token]; !ok {
		return "", false
	} else {
		return link.GetNextToken(rand), true
	}
}

func (c *singleKeyChain) RetrieveMarkovLink(token string) (link MarkovChainLink, keyPresent bool) {
	link, ok := c.Links[token]
	return link, ok
}

func buildChain(tokenChannel <-chan string) *singleKeyChain {
	links := make(map[string]*singleTokenLink)

	appendToChain := func(prev string, next string) {
		var link *singleTokenLink
		if extantLink, ok := links[prev]; !ok {
			link = &singleTokenLink{
				Token:                [1]string{prev},
				NextTokenOccurrences: make(map[string]int),
			}
		} else {
			link = extantLink
		}

		link.NextTokenOccurrences[next] = link.NextTokenOccurrences[next] + 1
		link.Total++
		links[prev] = link
	}

	lastVal := ""
	for val := range tokenChannel {
		appendToChain(lastVal, val)
		lastVal = val
	}
	appendToChain(lastVal, "")

	return &singleKeyChain{
		Links: links,
	}
}

// BuildSingleLinkChain builds a Markov chain from a series of keys provided
// by the tokenChannels and emits the result on the Markov chain channel when complete
func BuildSingleLinkChain(chainChannel chan<- MarkovChain, tokenChannels ...chan string) {
	chainSlice := make([]*singleKeyChain, 0, len(tokenChannels))
	wg := sync.WaitGroup{}
	chainTex := sync.Mutex{}
	for _, channel := range tokenChannels {
		channel := channel
		wg.Add(1)
		go func() {
			resultingChain := buildChain(channel)
			chainTex.Lock()
			chainSlice = append(chainSlice, resultingChain)
			chainTex.Unlock()
			wg.Done()
		}()
	}
	wg.Wait()

	chainChannel <- mergeChains(chainSlice...)
	close(chainChannel)
}

func mergeChains(chains ...*singleKeyChain) *singleKeyChain {
	mergedLinks := make(map[string]*singleTokenLink)

	for _, chain := range chains {
		for _, link := range chain.Links {
			key := link.Token[0]
			var mergedLink *singleTokenLink
			if extantLink, ok := mergedLinks[key]; !ok {
				mergedLink = &singleTokenLink{
					Token:                [1]string{key},
					NextTokenOccurrences: make(map[string]int),
				}
			} else {
				mergedLink = extantLink
			}

			mergedLink.Total += link.Total
			for k, v := range link.NextTokenOccurrences {
				mergedLink.NextTokenOccurrences[k] += v
			}

			mergedLinks[key] = mergedLink
		}
	}

	return &singleKeyChain{
		Links: mergedLinks,
	}
}

func (l *singleTokenLink) GetNextToken(rand *rand.Rand) string {
	goalSum := rand.Intn(l.Total)

	sum := 0
	// note that ranging over a map is a random operation,
	// so even if the goal sum is the same, the resulting
	// value may not be
	for k, v := range l.NextTokenOccurrences {
		sum += v
		if sum >= goalSum {
			return k
		}
	}

	// this should be impossible
	return ""
}

func (l *singleTokenLink) RetrieveNextTokenPossibilities() (nextTokens []string) {
	slice := make([]string, 0, len(l.NextTokenOccurrences))

	for k := range l.NextTokenOccurrences {
		slice = append(slice, k)
	}

	return slice
}

func (l *singleTokenLink) GetProbabilityOfToken(nextToken string) (nextTokenProbability float64, tokenPresent bool) {
	if occurrences, ok := l.NextTokenOccurrences[nextToken]; !ok {
		return 0.0, false
	} else {
		return float64(occurrences) / float64(l.Total), true
	}
}
