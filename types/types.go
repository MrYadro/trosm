package types

import (
	"strconv"
)

type Poi struct {
	Name  string
	Color string
	Poit  string
}

type StopData struct {
	Name   string
	NameEn string
	Pois   []Poi
}

type RouteData struct {
	Ref   string
	Name  string
	From  string
	To    string
	Color string
	Stops []StopData
}

type RouteParams struct {
	Ref         string
	Operator    string
	Network     string
	PoiDistance int
}

type NaturalOrder []Poi

func (s NaturalOrder) Len() int {
	return len(s)
}

func (s NaturalOrder) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s NaturalOrder) Less(i, j int) bool {
	if len(s[i].Name) == len(s[j].Name) {
		intI, err := strconv.Atoi(s[i].Name)
		if err != nil {
			return false
		}
		intJ, err := strconv.Atoi(s[j].Name)
		if err != nil {
			return false
		}
		if intI < intJ {
			return true
		}
		return false
	}
	return len(s[i].Name) < len(s[j].Name)
}
