package server

import (
	"net/http"
	"strings"
)

// listCategories: GET /api/categories. The ways trips can be labelled on this car.
func (s *Server) listCategories(w http.ResponseWriter, r *http.Request) {
	cs, err := s.S.ListCategories()
	if err != nil {
		fail(w, 500, "db error")
		return
	}
	ok(w, cs)
}

type categoryIn struct {
	Label     string   `json:"label"`
	Color     string   `json:"color"`
	Rate      *float64 `json:"rate"`
	ClearRate bool     `json:"clearRate"`
}

func (in *categoryIn) check() string {
	if in.Rate != nil && (*in.Rate < 0 || *in.Rate > 10) {
		return "the rate per mile must be between 0 and 10"
	}
	return ""
}

// addCategory: POST /api/categories {label, color, rate}. Admin only.
func (s *Server) addCategory(w http.ResponseWriter, r *http.Request) {
	var in categoryIn
	if decode(r, &in) != nil {
		fail(w, 400, "bad request")
		return
	}
	if msg := in.check(); msg != "" {
		fail(w, 400, msg)
		return
	}
	key, err := s.S.AddCategory(in.Label, in.Color, in.Rate)
	if err != nil {
		fail(w, 400, err.Error())
		return
	}
	ok(w, map[string]string{"key": key})
}

// patchCategory: PATCH /api/categories/{key}. Admin only.
func (s *Server) patchCategory(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	var in categoryIn
	if decode(r, &in) != nil {
		fail(w, 400, "bad request")
		return
	}
	if msg := in.check(); msg != "" {
		fail(w, 400, msg)
		return
	}
	cs, _ := s.S.ListCategories()
	for _, c := range cs {
		if c.Key != key {
			continue
		}
		rate := c.Rate
		if in.ClearRate {
			rate = nil
		} else if in.Rate != nil {
			rate = in.Rate
		}
		label := strings.TrimSpace(in.Label)
		if label == "" {
			label = c.Label
		}
		if err := s.S.UpdateCategory(key, label, in.Color, rate); err != nil {
			fail(w, 400, err.Error())
			return
		}
		ok(w, "saved")
		return
	}
	fail(w, 404, "no such category")
}

// deleteCategory: DELETE /api/categories/{key}. Trips with it become Personal. Admin only.
func (s *Server) deleteCategory(w http.ResponseWriter, r *http.Request) {
	if err := s.S.DeleteCategory(r.PathValue("key")); err != nil {
		fail(w, 400, err.Error())
		return
	}
	if s.D != nil {
		_ = s.D.Reclassify()
	}
	ok(w, "removed")
}
