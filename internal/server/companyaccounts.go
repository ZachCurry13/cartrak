package server

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"

	"github.com/ZachCurry13/cartrak/internal/companylist"
)

const accountPrefix = "company:"

type account struct {
	Ref string `json:"ref"`
	companyPick
}

// accounts are the companies remembered by their account number, sorted by name.
func (s *Server) accounts() []account {
	kv, _ := s.S.KVWithPrefix(accountPrefix)
	out := []account{}
	for _, p := range kv {
		var a account
		if json.Unmarshal([]byte(p.V), &a.companyPick) == nil && a.Name != "" {
			a.Ref = strings.TrimPrefix(p.K, accountPrefix)
			out = append(out, a)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name) })
	return out
}

// byName is the remembered account whose name means the same company: the better guess for a line with no number or a new
// number. Only a single match counts, so two branches of one company are still asked about.
func byName(list []account, name string) *account {
	var hit *account
	for i := range list {
		if companylist.Same(name, list[i].Name) {
			if hit != nil && (hit.Lat != list[i].Lat || hit.Lon != list[i].Lon) {
				return nil
			}
			hit = &list[i]
		}
	}
	return hit
}

// listAccounts: GET /api/company-list/accounts. Admin only (addresses of customers).
func (s *Server) listAccounts(w http.ResponseWriter, r *http.Request) { ok(w, s.accounts()) }

// forgetAccount: DELETE /api/company-list/accounts/{ref}. Admin only.
func (s *Server) forgetAccount(w http.ResponseWriter, r *http.Request) {
	ref := r.PathValue("ref")
	if ref == "" || len(ref) > 20 || strings.Trim(ref, "0123456789") != "" {
		fail(w, 400, "give the account number")
		return
	}
	if err := s.S.DeleteKV(accountPrefix + ref); err != nil {
		fail(w, http.StatusInternalServerError, err.Error())
		return
	}
	ok(w, map[string]bool{"ok": true})
}
