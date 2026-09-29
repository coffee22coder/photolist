package main

import "context"

type StMem struct {
	items []*Photo
}

func NewStorage() *StMem {
	return &StMem{
		items: make([]*Photo, 0, 10),
	}
}

func (st *StMem) Add(ctx context.Context, p *Photo) error {
	st.items = append(st.items, p)
	return nil
}

func (st *StMem) GetPhotos(ctx context.Context, userID int) ([]*Photo, error) {
	userPhoto := make([]*Photo, 0)

	for _, p := range st.items {
		if p.UserID == userID {
			userPhoto = append(userPhoto, p)
		}
	}

	return userPhoto, nil
}
