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

func RemoveDuplicates(xs *[]string) {
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
	for i := 0; i < len(name); i++ {
		color := fmt.Sprintf("%x", name[i])
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

func Mos(w http.ResponseWriter, req *http.Request) {
	var transRef = req.FormValue("ref")
	var transNet = req.FormValue("network")
	var transOp = req.FormValue("operator")
	var walkDistance = req.FormValue("distance")
	themeColor := getColorFromName(transRef)

	transNet = strings.Replace(transNet, "\"", "\\\"", -1)
	transOp = strings.Replace(transOp, "\"", "\\\"", -1)
	transRef = strings.Replace(transRef, "\"", "\\\"", -1)

	if walkDistance == "" {
		walkDistance = "500"
	}

	var routesNum int
	var stopsNum int

	w.Header().Set("Content-Type", "image/svg+xml")
	s := svg.New(w)

	client := overpass.New()
	result, _ := client.Query(`[out:json];
	rel["network"="` + transNet + `"]["ref"="` + transRef + `"]["operator"="` + transOp + `"];
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
	var stops []int
	for _, relation := range result.Relations {
		if relation.Tags["type"] == "route" && relation.Tags["ref"] == transRef {
			stops = append(stops, 0)
			fmt.Println(relation.Tags["name"]+": "+relation.Tags["from"]+" -> "+relation.Tags["to"], relation.ID)
			for _, member := range relation.Members {
				if member.Role == "stop" || member.Role == "stop_exit_only" || member.Role == "stop_entry_only" {
					fmt.Println(member.Node.Tags["name"])
					stopsNum++
				}
			}
			stops[routesNum] = stopsNum
			stopsNum = 0
			routesNum++
		}
	}
	docLen := 0
	for i := range stops {
		docLen += (125*2 + 443) + stops[i]*260
	}

	s.Start(1920, docLen)
	for i := range stops {
		prevStops := 0
		if i > 0 {
			prevStops += stops[i-1]
		}
		posFix := (125*2+443)*i + prevStops*260
		s.Text(100, posFix+150, "Схема маршрута", "font-family:Fira Sans;font-weight:600;font-style: normal;text-anchor:start;font-size:50px;fill:black")
		s.Text(100, posFix+216, "Scheme of route", "font-family:Fira Sans;font-style:normal;text-anchor:start;font-size:50px;fill:#514d48")
		s.Rect(100, posFix+271, 300, 200, "fill:"+themeColor)
		s.Text(250, posFix+421, transRef, "font-family:Fira Sans;text-anchor:middle;font-size:150px;fill:white")
		s.Line(130, posFix+558, 130, posFix+580+stops[i]*260, "stroke-linecap:round;stroke:"+themeColor+";stroke-width:20")
		s.Text(1820, posFix+580+stops[i]*260, "© OpenStreetMap contributors", "font-family:Fira Sans;text-anchor:end;font-size:20px;fill:black")
	}

	stopsNum = 0
	routesNum = 0
	stopName := ""
	stopNameEn := ""

	for _, relation := range result.Relations {
		prevStops := 0
		if relation.Tags["type"] == "route" && relation.Tags["ref"] == transRef {
			if routesNum > 0 {
				prevStops += stops[routesNum-1]
			}
			posFix := (125*2+443)*routesNum + prevStops*260
			s.Text(450, posFix+354, relation.Tags["name"], "font-family:Fira Sans;text-anchor:start;font-size:50px;fill:black")
			s.Text(450, posFix+425, relation.Tags["from"]+" - "+relation.Tags["to"], "font-family:Fira Sans;text-anchor:start;font-size:50px;fill:#514d48")
			for _, member := range relation.Members {
				if member.Role == "stop" || member.Role == "stop_exit_only" || member.Role == "stop_entry_only" {
					stopName = member.Node.Tags["name"]
					stopNameEn = member.Node.Tags["name:en"]
					stopID := member.Node.ID
					s.Circle(130, posFix+630+260*(stopsNum-prevStops), 20, "fill:white;stroke:"+themeColor+";stroke-width:10")
					s.Text(205, posFix+647+260*(stopsNum-prevStops), stopName, "font-family:Fira Sans;text-anchor:start;font-weight:600;font-size:50px;fill:black")
					s.Text(205, posFix+713+260*(stopsNum-prevStops), stopNameEn, "font-family:Fira Sans;text-anchor:start;font-size:50px;fill:#514d48")

					stuffAround := []string{}
					maxDist, _ := strconv.Atoi(walkDistance)

					for _, mapNodes := range result.Nodes {
						if (mapNodes.Lat != 0 || mapNodes.Lon != 0) && mapNodes.Tags["public_transport"] == "stop_position" && stopID == mapNodes.ID {
							for _, potMapNodes := range result.Nodes {
								if potMapNodes.Lat != 0 || potMapNodes.Lon != 0 {
									len := distance(mapNodes.Lat, mapNodes.Lon, potMapNodes.Lat, potMapNodes.Lon)
									if len < float64(maxDist) {
										if potMapNodes.Tags["railway"] != "" {
											stuffAround = append(stuffAround, "poezd")
										}
										for _, mapRelations := range result.Relations {
											if mapRelations.Tags["ref"] != transRef && mapRelations.Tags["ref"] != "" {
												for _, potMapMembers := range mapRelations.Members {
													if potMapMembers.Role == "stop" && potMapMembers.Node.ID == potMapNodes.ID {
														stuffAround = append(stuffAround, mapRelations.Tags["ref"])
													}
												}
											}
										}
									}
								}
							}
						}
					}

					sort.Strings(stuffAround)
					RemoveDuplicates(&stuffAround)
					horFix := 0
					stuffWidth := 0
					for _, stuff := range stuffAround {
						// stuff = "длинныйй"
						strLen := utf8.RuneCountInString(stuff)
						if strLen < 5 || stuff == "poezd" {
							stuffWidth = 60
						} else {
							stuffWidth = (strLen-1)*18 + 50
						}
						s.Roundrect(200+horFix, posFix+745+260*(stopsNum-prevStops), stuffWidth, 60, 30, 30, "fill:"+getColorFromName(stuff))
						if stuff != "poezd" {
							s.Text(200+horFix+stuffWidth/2, posFix+786+260*(stopsNum-prevStops), stuff, "font-family:Fira Sans;text-anchor:middle;font-size:30px;fill:white")
						} else {
							s.Image(215+horFix, posFix+760+260*(stopsNum-prevStops), 30, 30, "train.svg", "")
						}
						if len(stuff) < 5 || stuff == "poezd" {
							horFix += 70
						} else {
							horFix += (strLen-1)*18 + 50 + 10
						}
					}

					s.Line(0, posFix, 1920, posFix, "stroke:black;")

					stopsNum++
				}
			}
			routesNum++
		}
	}
	s.End()
}
