package tree

type SearchTree interface {
	Insert(key string, value any)
	Search(key string) (any, bool)
}
