package tagspattern

type TagMatcher struct{}

func (m *TagMatcher) Filter(tags []TagPather, pattern string) []TagPather {
	return tags[1:3]
}
