package scheme

import (
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	svg "github.com/ajstarks/svgo"
	overpass "github.com/serjvanilla/go-overpass"
)

type stopData struct {
	name   string
	nameEn string
	pois   []string
}

type routeData struct {
	ref   string
	name  string
	from  string
	to    string
	stops []stopData
}

type routeParams struct {
	ref         string
	operator    string
	network     string
	poiDistance int
}

type naturalOrder []string

func (s naturalOrder) Len() int {
	return len(s)
}

func (s naturalOrder) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s naturalOrder) Less(i, j int) bool {
	if len(s[i]) == len(s[j]) {
		intI, _ := strconv.Atoi(s[i])
		intJ, err := strconv.Atoi(s[j])
		if err != nil {
			strLen := len(s[i])
			for k := strLen - 1; k >= 0; k-- {
				if s[i][k] < s[j][k] {
					return true
				}
			}
		}
		if intI < intJ {
			return true
		}
		return false
	}
	return len(s[i]) < len(s[j])
}

func translateHeader(lang string) string {
	trans := make(map[string]string)
	trans["ru"] = "Схема маршрута"
	trans["en"] = "Scheme of route"
	trans["es"] = "Esquema de ruta"
	return trans[lang]
}

func removeDuplicates(xs *[]string) {
	found := make(map[string]bool)
	j := 0
	for i, x := range *xs {
		if !found[x] {
			found[x] = true
			(*xs)[j] = (*xs)[i]
			j++
		}
	}
	*xs = (*xs)[:j]
}

func getColorFromName(name string) string {
	wayColors := [10]string{"#49b45d", "#3473ba", "#f67536", "#0ebdf5", "#ffb81b", "#815aa1", "#d6473d", "#704233", "#909093", "#68a0bd"}
	outNum := 0
	strLen := utf8.RuneCountInString(name)
	for i := 0; i < strLen; i++ {
		color := fmt.Sprintf("%d", name[i])
		num, err := strconv.Atoi(color)
		if err != nil {
			num = 0
		}
		outNum += num
	}
	return wayColors[outNum%10]
}

func hsin(theta float64) float64 {
	return math.Pow(math.Sin(theta/2), 2)
}

func distance(lat1, lon1, lat2, lon2 float64) float64 {

	var la1, lo1, la2, lo2, r float64
	la1 = lat1 * math.Pi / 180
	lo1 = lon1 * math.Pi / 180
	la2 = lat2 * math.Pi / 180
	lo2 = lon2 * math.Pi / 180

	r = 6378100

	h := hsin(la2-la1) + math.Cos(la1)*math.Cos(la2)*hsin(lo2-lo1)

	return 2 * r * math.Asin(math.Sqrt(h))
}

func prepareData(route routeParams) []routeData {
	route.network = strings.Replace(route.network, "\"", "\\\"", -1)
	route.operator = strings.Replace(route.operator, "\"", "\\\"", -1)
	route.ref = strings.Replace(route.ref, "\"", "\\\"", -1)

	if route.poiDistance == 0 {
		route.poiDistance = 300
	}

	client := overpass.New()
	result, _ := client.Query(`[out:json];
	rel["network"="` + route.network + `"]["ref"="` + route.ref + `"]["operator"="` + route.operator + `"];
	node(r) -> .stops;
	(
		node(around.stops:500.0)["railway"="station"];
		node(around.stops:500.0)["public_transport"="stop_position"];
	);
	(
		._;
		rel(bn);
	);
	out body;`)
	fmt.Println(result.Timestamp)

	routesNum := 0
	stopsNum := 0

	var data []routeData
	for _, relation := range result.Relations {
		if relation.Tags["type"] == "route" && relation.Tags["ref"] == route.ref {
			data = append(data,
				routeData{name: relation.Tags["name"],
					from: relation.Tags["from"],
					to:   relation.Tags["to"],
					ref:  relation.Tags["ref"]})
			for _, member := range relation.Members {
				if member.Role == "stop" || member.Role == "stop_exit_only" || member.Role == "stop_entry_only" {
					data[routesNum].stops = append(data[routesNum].stops, stopData{name: member.Node.Tags["name"],
						nameEn: member.Node.Tags["name:en"]})
					stopID := member.Node.ID
					for _, mapNodes := range result.Nodes {
						if (mapNodes.Lat != 0 || mapNodes.Lon != 0) && mapNodes.Tags["public_transport"] == "stop_position" && stopID == mapNodes.ID {
							for _, potMapNodes := range result.Nodes {
								if potMapNodes.Lat != 0 || potMapNodes.Lon != 0 {
									len := distance(mapNodes.Lat, mapNodes.Lon, potMapNodes.Lat, potMapNodes.Lon)
									if len < float64(route.poiDistance) {
										if potMapNodes.Tags["railway"] != "" {
											data[routesNum].stops[stopsNum].pois = append(data[routesNum].stops[stopsNum].pois, "poezd")
										}
										for _, mapRelations := range result.Relations {
											if mapRelations.Tags["ref"] != route.ref && mapRelations.Tags["ref"] != "" {
												for _, potMapMembers := range mapRelations.Members {
													if potMapMembers.Type == "node" {
														if potMapMembers.Role == "stop" && potMapMembers.Node.ID == potMapNodes.ID {
															data[routesNum].stops[stopsNum].pois = append(data[routesNum].stops[stopsNum].pois, mapRelations.Tags["ref"])
														}
													}
												}
											}
										}
									}
								}
							}
							sort.Sort(naturalOrder(data[routesNum].stops[stopsNum].pois))
							removeDuplicates(&data[routesNum].stops[stopsNum].pois)
							stopsNum++
						}
					}
				}
			}
			stopsNum = 0
			routesNum++
		}
	}
	return data
}

func MTrans(w http.ResponseWriter, req *http.Request) {
	walkDistance, err := strconv.Atoi(req.FormValue("distance"))
	if err != nil {
		walkDistance = 0
	}
	transRef := req.FormValue("ref")
	lang := req.FormValue("lang")

	if lang == "" {
		lang = "ru"
	}

	routes := prepareData(routeParams{ref: transRef,
		network:     req.FormValue("network"),
		operator:    req.FormValue("operator"),
		poiDistance: walkDistance})

	themeColor := getColorFromName(transRef)

	w.Header().Set("Content-Type", "image/svg+xml")
	s := svg.New(w)

	var pageStart []int
	docLen := 0

	for _, route := range routes {
		pageStart = append(pageStart, docLen)
		docLen += 125 // header + route ref and from to size
		docLen += 522
		for _, stop := range route.stops {
			docLen += 67 // stop name size
			if stop.nameEn != "" {
				docLen += 67 // stop name:en size
			}
			docLen += 10                         // spacing between text and icons
			docLen += 70 * (len(stop.pois) / 15) // pois size
			docLen += 100
		}
		docLen += 125 // footer
	}

	s.Start(1920, docLen)

	for rt, route := range routes {
		s.Line(0, pageStart[rt]+0, 1920, pageStart[rt]+0, "stroke:black;")
		s.Text(100, pageStart[rt]+150, translateHeader(lang), "font-family:Fira Sans;font-weight:600;font-style: normal;text-anchor:start;font-size:50px;fill:black")
		if lang != "en" {
			s.Text(100, pageStart[rt]+216, "Scheme of route", "font-family:Fira Sans;font-style:normal;text-anchor:start;font-size:50px;fill:#514d48")
		}
		s.Rect(100, pageStart[rt]+271, 300, 200, "fill:"+themeColor)
		s.Text(250, pageStart[rt]+421, transRef, "font-family:Fira Sans;text-anchor:middle;font-size:150px;fill:white")
		if route.from != "" || route.to != "" {
			s.Text(450, pageStart[rt]+354, route.name, "font-family:Fira Sans;text-anchor:start;font-size:50px;fill:black")
			s.Text(450, pageStart[rt]+425, route.from+" - "+route.to, "font-family:Fira Sans;text-anchor:start;font-size:50px;fill:#514d48")
		} else {
			s.Text(450, pageStart[rt]+390, route.name, "font-family:Fira Sans;text-anchor:start;font-size:50px;fill:black")
		}
		var stopPos []int
		vertFix := 0
		for _, stop := range route.stops {
			stopPos = append(stopPos, vertFix)
			s.Text(205, pageStart[rt]+647+vertFix, stop.name, "font-family:Fira Sans;text-anchor:start;font-weight:600;font-size:50px;fill:black")
			vertFix += 67
			if stop.nameEn != "" {
				s.Text(205, pageStart[rt]+647+vertFix, stop.nameEn, "font-family:Fira Sans;text-anchor:start;font-size:50px;fill:#514d48")
				vertFix += 67
			}

			vertFix += 10

			horFix := 0
			stuffWidth := 0
			for po, poi := range stop.pois {
				strLen := utf8.RuneCountInString(poi)
				if strLen < 4 || poi == "poezd" {
					stuffWidth = 60
				} else {
					stuffWidth = strLen*18 + 20
				}

				s.Roundrect(200+horFix, pageStart[rt]+607+vertFix, stuffWidth, 60, 30, 30, "fill:"+getColorFromName(poi))
				if poi != "poezd" {
					s.Text(200+horFix+stuffWidth/2, pageStart[rt]+647+vertFix, poi, "font-family:Fira Sans;text-anchor:middle;font-size:30px;fill:white")
				} else {
					s.Image(215+horFix, pageStart[rt]+623+vertFix, 30, 30, "train.svg", "")
				}
				if strLen < 4 || poi == "poezd" {
					horFix += 70
				} else {
					horFix += strLen*18 + 20 + 10
				}
				if (po+1)%15 == 0 && po != 0 {
					vertFix += 70
					horFix = 0
				}
			}
			vertFix += 100
		}
		s.Line(130, pageStart[rt]+558, 130, pageStart[rt]+610+vertFix, "stroke-linecap:round;stroke:"+themeColor+";stroke-width:20")
		for stop := range route.stops {
			s.Circle(130, pageStart[rt]+630+stopPos[stop], 20, "fill:white;stroke:"+themeColor+";stroke-width:10")
		}
		s.Text(1820, pageStart[rt]+610+vertFix, "© OpenStreetMap contributors", "font-family:Fira Sans;text-anchor:end;font-size:20px;fill:black")
	}
	s.End()
}
