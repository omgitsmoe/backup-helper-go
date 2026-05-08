package pathstore

type PathId int
type Store struct {
	m map[string]PathId
	// NOTE: not bearable without fast lookup table
	l []string
}

func NewStore() Store {
	return Store{
		m: make(map[string]PathId),
	}
}

func (s *Store) Store(p string) PathId {
	if id, exist := s.m[p]; exist {
		return id
	}

	id := PathId(len(s.l))
	s.m[p] = id
	s.l = append(s.l, p)

	return id
}

func (s *Store) Lookup(searchId PathId) string {
	return s.l[searchId]
}
