package tagmatcher

import (
	"strings"

	"github.com/juju/errors"
)

type MatcherType int

const (
	MatcherTypeExact MatcherType = iota
	MatcherTypeFuzzy
	MatcherTypeExactThenFuzzy
)

type TagMatcher struct {
	DefMatcherType MatcherType
}

func New() *TagMatcher {
	return &TagMatcher{
		DefMatcherType: MatcherTypeExactThenFuzzy,
	}
}

func (m *TagMatcher) Filter(tags []TagPather, pattern string) ([]TagPather, error) {
	if len(pattern) > 30 {
		return nil, errors.Errorf("pattern is too long")
	}

	pats := strings.Fields(pattern)

	results := []*Result{}

	for _, p := range pats {
		mType := m.DefMatcherType

		// Handle matcher-type prefix, if any
		if strings.HasPrefix(p, "=~") || strings.HasPrefix(p, "~=") {
			p = p[2:]
			mType = MatcherTypeExactThenFuzzy
		} else if strings.HasPrefix(p, "=") {
			p = p[1:]
			mType = MatcherTypeExact
		} else if strings.HasPrefix(p, "~") {
			p = p[1:]
			mType = MatcherTypeFuzzy
		}

		var m Matcher

		switch mType {
		case MatcherTypeExact:
			m = &MatcherExact{}
		}

		results = append(results, m.Filter(tags, p)...)
	}

	return CombineResults(results, tags), nil
}
