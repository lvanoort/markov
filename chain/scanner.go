package chain

import (
	"bufio"
	"io"
)

type bufioScannerSource struct {
	src *bufio.Scanner
}

func (s *bufioScannerSource) NextToken() (string, error) {
	if s.src.Scan() {
		return s.src.Text(), nil
	} else {
		if e := s.src.Err(); e == nil {
			return "", io.EOF
		} else {
			return "", e
		}
	}
}

// BuildChainFromScanners is a convenience function for building a markov chain from scanners providing
// tokens
func BuildChainFromScanners(tokenSources ...*bufio.Scanner) (MarkovChain, error) {
	return BuildChainFromSources(SourcesFromScanners(tokenSources...)...)
}

// SourcesFromScanners converts scanners into token sources
func SourcesFromScanners(tokenSources ...*bufio.Scanner) []TokenSource {
	sources := make([]TokenSource, 0, len(tokenSources))
	for _, tok := range tokenSources {
		sources = append(sources, &bufioScannerSource{src: tok})
	}

	return sources
}
