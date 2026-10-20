package store

import (
	"errors"
	"regexp"
	"strings"
)

// Category is a way to label a trip: Personal, Work, Medical, or one the owner adds (Charity, a side business, a club).
type Category struct {
	Key     string   `json:"key"` // the short name stored on each trip, such as "medical"
	Label   string   `json:"label"`
	Color   string   `json:"color"` // a hex colour like #dc2626, for the label on a trip; empty for the built-in look
	Rate    *float64 `json:"rate"`  // a mileage deduction per mile for this category; nil uses the general rate (work) or none
	Builtin bool     `json:"builtin"`
	Sort    int      `json:"sort"`
}

// MaxCategories is how many categories one car can have: the built-in ones and the owner's own.
const MaxCategories = 12

const categoriesSchema = `CREATE TABLE IF NOT EXISTS categories (
  key TEXT PRIMARY KEY, label TEXT NOT NULL, color TEXT NOT NULL DEFAULT '', rate REAL,
  builtin INTEGER NOT NULL DEFAULT 0, sort INTEGER NOT NULL DEFAULT 0)`

// The categories every car starts with. Personal and Work cannot be removed; Medical can be renamed but not removed either, so
// reports that mention it stay valid.
var seedCategories = []Category{
	{Key: "personal", Label: "Personal", Builtin: true, Sort: 1},
	{Key: "work", Label: "Work", Builtin: true, Sort: 2},
	{Key: "medical", Label: "Medical", Color: "#dc2626", Builtin: true, Sort: 3},
}

func (s *Store) ensureCategories() error {
	if _, err := s.DB.Exec(categoriesSchema); err != nil {
		return err
	}
	for _, c := range seedCategories {
		if _, err := s.DB.Exec(`INSERT OR IGNORE INTO categories(key,label,color,builtin,sort) VALUES(?,?,?,1,?)`, c.Key, c.Label, c.Color, c.Sort); err != nil {
			return err
		}
	}
	return nil
}

// ListCategories returns every category, the built-in ones first.
func (s *Store) ListCategories() ([]Category, error) {
	if err := s.ensureCategories(); err != nil {
		return nil, err
	}
	rows, err := s.DB.Query(`SELECT key,label,color,rate,builtin,sort FROM categories ORDER BY sort, label`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Category{}
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.Key, &c.Label, &c.Color, &c.Rate, &c.Builtin, &c.Sort); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// CategoryExists says whether a key is one of this car's categories.
func (s *Store) CategoryExists(key string) bool {
	if err := s.ensureCategories(); err != nil {
		return key == "work" || key == "personal"
	}
	var n int
	_ = s.DB.QueryRow(`SELECT COUNT(*) FROM categories WHERE key=?`, key).Scan(&n)
	return n > 0
}

var nonKey = regexp.MustCompile(`[^a-z0-9]+`)
var hexColor = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// CleanColor accepts a #rrggbb colour or nothing.
func CleanColor(c string) (string, error) {
	c = strings.TrimSpace(c)
	if c != "" && !hexColor.MatchString(c) {
		return "", errors.New("the colour must look like #dc2626")
	}
	return c, nil
}

// AddCategory makes a new category from a label and returns its key.
func (s *Store) AddCategory(label, color string, rate *float64) (string, error) {
	if err := s.ensureCategories(); err != nil {
		return "", err
	}
	label = strings.Join(strings.Fields(label), " ")
	if label == "" || len([]rune(label)) > 24 {
		return "", errors.New("give the category a short name (up to 24 characters)")
	}
	col, err := CleanColor(color)
	if err != nil {
		return "", err
	}
	var n int
	_ = s.DB.QueryRow(`SELECT COUNT(*) FROM categories`).Scan(&n)
	if n >= MaxCategories {
		return "", errors.New("that is the most categories a car can have")
	}
	key := strings.Trim(nonKey.ReplaceAllString(strings.ToLower(label), "-"), "-")
	if key == "" || key == "vacation" || key == "all" {
		key = "category"
	}
	base, i := key, 2
	for s.CategoryExists(key) {
		key = base + "-" + string(rune('0'+i%10))
		i++
		if i > 30 {
			return "", errors.New("that name is taken")
		}
	}
	_, err = s.DB.Exec(`INSERT INTO categories(key,label,color,rate,builtin,sort) VALUES(?,?,?,?,0,?)`, key, label, col, rate, 100+n)
	return key, err
}

// UpdateCategory changes a category's name, colour and rate. The key never changes.
func (s *Store) UpdateCategory(key, label, color string, rate *float64) error {
	if err := s.ensureCategories(); err != nil {
		return err
	}
	label = strings.Join(strings.Fields(label), " ")
	if label == "" || len([]rune(label)) > 24 {
		return errors.New("give the category a short name (up to 24 characters)")
	}
	col, err := CleanColor(color)
	if err != nil {
		return err
	}
	r, err := s.DB.Exec(`UPDATE categories SET label=?,color=?,rate=? WHERE key=?`, label, col, rate, key)
	if err != nil {
		return err
	}
	if n, _ := r.RowsAffected(); n == 0 {
		return errors.New("no such category")
	}
	return nil
}

// DeleteCategory removes one the owner added. Its trips become Personal, its rules go, and a mode that used it ends.
func (s *Store) DeleteCategory(key string) error {
	if err := s.ensureCategories(); err != nil {
		return err
	}
	var builtin bool
	if s.DB.QueryRow(`SELECT builtin FROM categories WHERE key=?`, key).Scan(&builtin) != nil {
		return errors.New("no such category")
	}
	if builtin {
		return errors.New("Personal, Work and Medical are built in and cannot be removed")
	}
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, q := range []string{
		`UPDATE trips SET category='personal', category_source='default' WHERE category=?`,
		`DELETE FROM rules WHERE category=?`,
		`DELETE FROM categories WHERE key=?`,
	} {
		if _, err := tx.Exec(q, key); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(timeOffSchema); err == nil {
		_, _ = tx.Exec(`DELETE FROM time_off WHERE kind=?`, key)
	}
	return tx.Commit()
}
