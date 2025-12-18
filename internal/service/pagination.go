package service

import (
	"net/http"
	"strconv"
)

type Pagination struct {
	Limit  int    `json:"limit" validate:"gte=1,lte=20"`  // больше или равно чем 1
	Offset int    `json:"offset" validate:"gte=0"`        // больше или равно чем 0
	Sort   string `json:"sort" validate:"oneof=asc desc"` // валидируем либо desc либо asc
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

	return p, nil
}
