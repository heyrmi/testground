// Package fake generates the small amount of person-shaped data the
// challenges need, from a seeded stream.
//
// It exists instead of a fake-data dependency because the Go dependency budget
// is ten and this is twenty lines. It draws in a fixed order on purpose: the
// values a released challenge shows for a given seed are part of its
// stability contract, so the sequence here must not be rearranged.
package fake

import (
	"fmt"
	"math/rand/v2"
	"strings"
)

// Person is one generated record.
type Person struct {
	Name   string
	Email  string
	Status string
	Amount string
}

// NewPerson draws one record. The five draws happen in the order first name,
// last name, status, whole amount, fractional amount, and changing that order
// changes every page generated from a seed.
func NewPerson(r *rand.Rand, index int) Person {
	first := FirstNames[r.IntN(len(FirstNames))]
	last := LastNames[r.IntN(len(LastNames))]

	return Person{
		Name:   first + " " + last,
		Email:  fmt.Sprintf("%s.%s%d@example.test", strings.ToLower(first), strings.ToLower(last), index),
		Status: Statuses[r.IntN(len(Statuses))],
		Amount: fmt.Sprintf("%d.%02d", r.IntN(9000)+10, r.IntN(100)),
	}
}

// The corpora are small and deliberately varied, so generated rows read like
// records rather than like lorem ipsum, and so tests exercise non-ASCII names.
var (
	FirstNames = []string{
		"Ama", "Bilal", "Chandra", "Dilnoza", "Eero", "Fatima", "Gustavo", "Hana",
		"Ines", "Jarrah", "Kenji", "Lucia", "Mateo", "Nadia", "Oskar", "Priya",
		"Quentin", "Rahel", "Sonia", "Tomas", "Ugo", "Vera", "Wen", "Yusuf", "Zofia",
	}
	LastNames = []string{
		"Adeyemi", "Bergström", "Chowdhury", "Delacroix", "Eskildsen", "Ferreira",
		"Gruber", "Halvorsen", "Iqbal", "Jankowski", "Kaminski", "Lindqvist",
		"Moretti", "Nakamura", "Okonkwo", "Petrov", "Quiroga", "Rahman", "Silva",
		"Takahashi", "Ustinov", "Vasquez", "Watanabe", "Yilmaz", "Zeeman",
	}
	Statuses = []string{"active", "pending", "suspended", "closed"}
)
