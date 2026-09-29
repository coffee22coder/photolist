package main

type Photo struct {
	ID     int
	UserID int
	Path   string // MD5, 32 hex-символа, без каталога и расширения
}

type StMem struct {
	items []*Photo
}

func NewStorage() *StMem {
	return &StMem{
		items: make([]*Photo, 0, 10),
	}
}

func (st *StMem) Add(p *Photo) error {
	st.items = append(st.items, p)
	return nil
}

func (st *StMem) GetPhotos(userID int) ([]*Photo, error) {
	userPhoto := make([]*Photo, 0)

	for _, p := range st.items {
		if p.UserID == userID {
			userPhoto = append(userPhoto, p)
		}
	}

	return userPhoto, nil
}
