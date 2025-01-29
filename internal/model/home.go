package model

import (
	"crypto/sha256"
	"encoding/binary"
)

type MatchType string

const (
	Practice      MatchType = "Practice"
	Qualification           = "Qualification"
	Playoff                 = "Playoff"
)

var MatchTypeMap = map[string]MatchType{
	"Practice":      Practice,
	"Qualification": Qualification,
	"Playoff":       Playoff,
}

type Meta struct {
	ScouterName string    `json:"scouterName"`
	MatchNumber uint64    `json:"matchNumber"`
	TeamNumber  uint64    `json:"teamNumber"`
	MatchType   MatchType `json:"matchType"`
}

func (meta *Meta) Hash() []byte {
	hasher := sha256.New()
	hasher.Write([]byte(meta.ScouterName))
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, meta.MatchNumber)
	hasher.Write(b)
	b = make([]byte, 8)
	binary.LittleEndian.PutUint64(b, meta.TeamNumber)
	hasher.Write(b)
	hasher.Write([]byte(meta.MatchType))
	return hasher.Sum(nil)
}

type Page[T any] struct {
	Title string
	Data  T
}

type MatchForm struct {
	Meta  Meta
	Error string
}

type Index struct {
	Form    MatchForm
	Matches []Match
}

type IsSelected[T any] struct {
	Selected bool
	Value    T
}

type Radio[T any] struct {
	Checked bool
	Name    string
	Value   T
	Post    string
}

func (matchForm MatchForm) IsSelected(matchType MatchType) IsSelected[MatchType] {
	return IsSelected[MatchType]{
		Selected: matchForm.Meta.MatchType == matchType,
		Value:    matchType,
	}
}
