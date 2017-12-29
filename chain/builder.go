package chain

import (
	"io"
)

// TokenSource provides a stream of tokens for building a Markov chain
type TokenSource interface {
	NextToken() (string, error)
}

// BuildChainFromSources builds a Markov chain from sources providing
// tokens
func BuildChainFromSources(tokenSources ...TokenSource) (MarkovChain, error) {
	tokChans := make([]chan string, 0, len(tokenSources))
	chainChan := make(chan MarkovChain)
	errorChan := make(chan error)
	for _, v := range tokenSources {
		localVal := v

		tokChan := make(chan string, 20)
		tokChans = append(tokChans, tokChan)
		go func() {
			for {
				token, tokenErr := localVal.NextToken()
				if tokenErr == io.EOF {
					close(tokChan)
					return
				} else if tokenErr != nil {
					errorChan <- tokenErr
				} else {
					tokChan <- token
				}
			}
		}()
	}

	go BuildSingleLinkChain(chainChan, tokChans...)

	select {
	case chain := <-chainChan:
		return chain, nil
	case e := <-errorChan:
		return nil, e
	}
}
