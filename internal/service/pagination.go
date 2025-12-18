package service

import (
	"net/http"
	"strconv"
)

type Pagination struct {
	Limit  int    `json:"limit" validate:"gte=1,lte=20"`  // больше или равно чем 1
	Offset int    `json:"offset" validate:"gte=0"`        // больше или равно чем 0
	Sort   string `json:"sort" validate:"oneof=asc desc"` // валидируем либо desc либо asc
	Search string `json:"search" validate:"max=100"`      // поиск по title и по content
}

func (p *Pagination) Parse(r *http.Request) (*Pagination, error) {
	qs := r.URL.Query()

	limit := qs.Get("limit")
	if limit != "" {
		l, _ := strconv.Atoi(limit)

		p.Limit = l
	}

	offset := qs.Get("offset")
	if offset != "" {
		o, _ := strconv.Atoi(offset)

		p.Offset = o
	}

	sort := qs.Get("sort")
	if sort != "" {
		p.Sort = sort
	}

	search := qs.Get("search")
	if search != "" {
		p.Search = search
	}

	return p, nil
}
