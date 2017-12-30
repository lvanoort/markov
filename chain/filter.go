package chain

import "strings"

// SourceFilter performs transforms on candidate tokens before they are fed into
// the Markov chain
type SourceFilter interface {
	// FilterToken accepts a candidate token and returns a slice of the resulting
	// tokens that should be passed on and potentially an error if one
	// occurs. A zero length slice means that the candidate token contains no
	// acceptable tokens
	FilterToken(candidate string) ([]string, error)
}

type filteredSource struct {
	src    TokenSource
	filter SourceFilter
	queue  []string
	index  int
}

func (s *filteredSource) NextToken() (string, error) {
	queueLen := len(s.queue)
	if s.index < queueLen {
		next := s.queue[s.index]
		s.index++
		return next, nil
	} else if queueLen != 0 {
		// nil-out queue when no longer needed so it can be
		// garbage collected
		s.queue = nil
		s.index = 0
	}

	for {
		candidate, readErr := s.src.NextToken()
		if readErr != nil {
			return "", readErr
		} else {
			tokens, tokenErr := s.filter.FilterToken(candidate)
			if tokenErr != nil {
				return "", tokenErr
			} else {
				count := len(tokens)
				if count == 1 {
					return tokens[0], nil
				} else if count > 1 {
					s.queue = tokens
					s.index = 1

					return tokens[0], nil
				}
			}
		}
	}
}

// MakeFilteredTokenSources applies a specified filter to number bunch of TokenSources
// and returns TokeSources with the filters applied
func MakeFilteredTokenSources(filter SourceFilter, sources ...TokenSource) []TokenSource {
	filteredSources := make([]TokenSource, 0, len(sources))
	for _, v := range sources {
		filteredSources = append(
			filteredSources,
			&filteredSource{
				src:    v,
				filter: filter,
			},
		)
	}
	return filteredSources
}

// ApplyFiltersToSource applies a series of filters in order to a TokenSource and returns
// a TokenSource with the filters applied
func ApplyFiltersToSource(source TokenSource, filters ...SourceFilter) TokenSource {
	for _, v := range filters {
		source = &filteredSource{
			src:    source,
			filter: v,
		}
	}
	return source
}

// SourceFilterFunc is adapter to so functions can be used as SourceFilters
type SourceFilterFunc func(candidate string) ([]string, error)

type funcFilter struct {
	filter SourceFilterFunc
}

func (f *funcFilter) FilterToken(candidate string) ([]string, error) {
	return f.filter(candidate)
}

// MakeFuncFilter converts a SourceFilterFunc into a SourceFilter
func MakeFuncFilter(filter SourceFilterFunc) SourceFilter {
	return &funcFilter{
		filter: filter,
	}
}

// LowercaseFilter filters a TokenSource by converting all candidate tokens
// to lowercase
func LowercaseFilter() SourceFilter {
	return MakeFuncFilter(func(candidate string) ([]string, error) {
		return []string{strings.ToLower(candidate)}, nil
	})
}

// TrimFilter filters a TokenSource by trimming leading and following
// tokens candidate tokens of whitespace
func TrimFilter() SourceFilter {
	return MakeFuncFilter(func(candidate string) ([]string, error) {
		return []string{strings.TrimSpace(candidate)}, nil
	})
}

// SubstitutionFilter filters a TokenSource by replacing candidate tokens
// with a substitution
func SubstitutionFilter(substitutions map[string]string) SourceFilter {
	return MakeFuncFilter(func(candidate string) ([]string, error) {
		substitution, hadSubstitution := substitutions[candidate]
		if hadSubstitution {
			return []string{substitution}, nil
		} else {
			return []string{candidate}, nil
		}
	})
}

// PrefixFilter filters a TokenSource by removing the specified prefix found
// in candidate tokens. If set to iterate, it will repeatedly attempt to
// trim the prefix until the string is empty or no more instances of the
// prefix are found
func PrefixFilter(prefix string, iterate bool) SourceFilter {
	return MakeFuncFilter(func(candidate string) ([]string, error) {
		if iterate {
			trimmed := candidate
			for trimmed != "" {
				next := strings.TrimPrefix(trimmed, prefix)
				if next == trimmed {
					return []string{trimmed}, nil
				} else {
					trimmed = next
				}
			}

			return []string{trimmed}, nil
		} else {
			return []string{strings.TrimPrefix(candidate, prefix)}, nil
		}
	})
}
