package chain

import (
	"bufio"
	"io"
)

// BuildChainFromScanners builds a markov chain from scanners providing
// tokens
func BuildChainFromScanners(tokenSources ...*bufio.Scanner) (MarkovChain, error) {
	tokChans := make([]chan string, 0, len(tokenSources))
	chainChan := make(chan MarkovChain)
	errorChan := make(chan error)
	for _, v := range tokenSources {
		localVal := v

		tokChan := make(chan string, 20)
		tokChans = append(tokChans, tokChan)
		go func() {
			for localVal.Scan() {
				if str := localVal.Text(); str != "" {
					tokChan <- str
				}
			}

			if e := localVal.Err(); e != nil && e != io.EOF {
				errorChan <- e
			}

			close(tokChan)
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
