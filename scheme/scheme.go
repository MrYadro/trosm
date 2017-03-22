package scheme

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/MrYadro/trosm/helpers"
	"github.com/MrYadro/trosm/types"
	svg "github.com/ajstarks/svgo"
	overpass "github.com/serjvanilla/go-overpass"
)

func isPoi(node overpass.Node) (bool, types.Poi) {
	if node.Tags["railway"] == "station" {
		if node.Tags["station"] == "" {
			return true, types.Poi{
				Name:  "poi",
				Color: node.Tags["colour"],
				Poit:  "train"}
		}
		if node.Tags["station"] == "subway" {
			return true, types.Poi{
				Name:  "poi",
				Color: node.Tags["colour"],
				Poit:  "metro"}
		}
	}
	return false, types.Poi{}
}

func prepareData(route types.RouteParams) []types.RouteData {
	route.Network = strings.Replace(route.Network, "\"", "\\\"", -1)
	opNoEscaped := route.Operator
	route.Operator = strings.Replace(route.Operator, "\"", "\\\"", -1)
	route.Ref = strings.Replace(route.Ref, "\"", "\\\"", -1)

	if route.PoiDistance == 0 {
		route.PoiDistance = 300
	}

	client := overpass.New()
	result, _ := client.Query(`[out:json];
	rel["network"="` + route.Network + `"]["ref"="` + route.Ref +
		`"]["operator"="` + route.Operator + `"];
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

	var data []types.RouteData
	for _, relation := range result.Relations {
		if relation.Tags["type"] == "route" &&
			relation.Tags["ref"] == route.Ref &&
			(relation.Tags["network"] == route.Network ||
				relation.Tags["operator"] == opNoEscaped) {
			data = append(data,
				types.RouteData{
					Name:  relation.Tags["name"],
					From:  relation.Tags["from"],
					To:    relation.Tags["to"],
					Ref:   relation.Tags["ref"],
					Color: relation.Tags["colour"]})
			for _, member := range relation.Members {
				if member.Role == "stop" || member.Role == "stop_exit_only" || member.Role == "stop_entry_only" {
					data[routesNum].Stops = append(data[routesNum].Stops, types.StopData{Name: member.Node.Tags["name"],
						NameEn: member.Node.Tags["name:en"]})
					stopID := member.Node.ID
					for _, mapNodes := range result.Nodes {
						if (mapNodes.Lat != 0 || mapNodes.Lon != 0) && mapNodes.Tags["public_transport"] == "stop_position" && stopID == mapNodes.ID {
							for _, potMapNodes := range result.Nodes {
								if potMapNodes.Lat != 0 || potMapNodes.Lon != 0 {
									len := helpers.Distance(mapNodes.Lat, mapNodes.Lon, potMapNodes.Lat, potMapNodes.Lon)
									if len < float64(route.PoiDistance) {
										potPoi, tempPoi := isPoi(*potMapNodes)
										if potPoi {
											data[routesNum].Stops[stopsNum].Pois = append(data[routesNum].Stops[stopsNum].Pois, tempPoi)
										} else {
											for _, mapRelations := range result.Relations {
												if mapRelations.Tags["ref"] != route.Ref && mapRelations.Tags["ref"] != "" {
													for _, potMapMembers := range mapRelations.Members {
														if potMapMembers.Type == "node" {
															if potMapMembers.Role == "stop" && potMapMembers.Node.ID == potMapNodes.ID {
																data[routesNum].Stops[stopsNum].Pois = append(data[routesNum].Stops[stopsNum].Pois, types.Poi{Name: mapRelations.Tags["ref"],
																	Color: mapRelations.Tags["colour"],
																	Poit:  "stop"})
															}
														}
													}
												}
											}
										}
									}
								}
							}
							sort.Sort(types.NaturalOrder(data[routesNum].Stops[stopsNum].Pois))
							helpers.RemoveDuplicates(&data[routesNum].Stops[stopsNum].Pois)
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

// MTrans generates MOSTRANS lookalike route maps
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

	routes := prepareData(types.RouteParams{Ref: transRef,
		Network:     req.FormValue("network"),
		Operator:    req.FormValue("operator"),
		PoiDistance: walkDistance})

	themeColor := helpers.GetColorFromName(transRef)

	w.Header().Set("Content-Type", "image/svg+xml")
	s := svg.New(w)

	var pageStart []int
	docLen := 0

	for _, route := range routes {
		pageStart = append(pageStart, docLen)
		docLen += 125 // header + route ref and from to size
		docLen += 522
		for _, stop := range route.Stops {
			docLen += 67 // stop name size
			if stop.NameEn != "" {
				docLen += 67 // stop name:en size
			}
			docLen += 10 // spacing between text and icons
			if len(stop.Pois)%15 != 0 {
				docLen += 70 * (len(stop.Pois) / 15) // pois size
			}
			docLen += 100
		}
		docLen += 125 // footer
	}

	s.Start(1920, docLen)

	for rt, route := range routes {
		if route.Color != "" {
			themeColor = route.Color
		}
		s.Line(0, pageStart[rt], 1920, pageStart[rt]+0, "stroke:black;")                                                                 // line for cutting
		s.Text(100, pageStart[rt]+150, helpers.TranslateHeader(lang), "font-family:Fira Sans;font-weight:600;font-size:50px;fill:black") // header
		if lang != "en" {
			s.Text(100, pageStart[rt]+216, "Scheme of route", "font-family:Fira Sans;font-size:50px;fill:#514d48") // header sub text
		}
		s.Rect(100, pageStart[rt]+271, 300, 200, "fill:"+themeColor)                                                    // route ref background
		s.Text(250, pageStart[rt]+421, transRef, "font-family:Fira Sans;text-anchor:middle;font-size:150px;fill:white") // route ref
		if route.From != "" || route.To != "" {
			s.Text(450, pageStart[rt]+354, route.Name, "font-family:Fira Sans;font-size:50px;fill:black")                  //route name
			s.Text(450, pageStart[rt]+425, route.From+" - "+route.To, "font-family:Fira Sans;font-size:50px;fill:#514d48") //route from - to
		} else {
			s.Text(450, pageStart[rt]+390, route.Name, "font-family:Fira Sans;font-size:50px;fill:black") //route name if no from to
		}
		var stopPos []int
		vertFix := 0
		for _, stop := range route.Stops {
			stopPos = append(stopPos, vertFix)
			s.Text(205, pageStart[rt]+647+vertFix, stop.Name, "font-family:Fira Sans;font-weight:600;font-size:50px;fill:black") // stop name
			vertFix += 67
			if stop.NameEn != "" {
				s.Text(205, pageStart[rt]+647+vertFix, stop.NameEn, "font-family:Fira Sans;font-size:50px;fill:#514d48") // stop name in english if present
				vertFix += 67
			}

			vertFix += 10

			horFix := 0
			var stuffWidth int
			for po, poi := range stop.Pois {
				strLen := utf8.RuneCountInString(poi.Name)
				if strLen < 4 || poi.Name == "poi" {
					stuffWidth = 60
				} else {
					stuffWidth = strLen*18 + 20
				}

				if poi.Color != "" {
					s.Roundrect(200+horFix, pageStart[rt]+607+vertFix, stuffWidth, 60, 30, 30, "fill:"+helpers.ColorOsm(poi.Color))
				} else {
					s.Roundrect(200+horFix, pageStart[rt]+607+vertFix, stuffWidth, 60, 30, 30, "fill:"+helpers.GetColorFromName(poi.Name))
				}
				if poi.Name != "poi" {
					s.Text(200+horFix+stuffWidth/2, pageStart[rt]+647+vertFix, poi.Name, "font-family:Fira Sans;text-anchor:middle;font-size:30px;fill:white") // number of route
				} else {
					s.Image(215+horFix, pageStart[rt]+623+vertFix, 30, 30, poi.Poit+".svg", "") // icon of PT
				}
				if strLen < 4 || poi.Name == "poi" {
					horFix += 70
				} else {
					horFix += strLen*18 + 20 + 10
				}
				if (po+1)%15 == 0 && po != 0 && len(stop.Pois) != 15 {
					vertFix += 70
					horFix = 0
				}
			}
			vertFix += 100
		}
		s.Line(130, pageStart[rt]+558, 130, pageStart[rt]+610+vertFix, "stroke-linecap:round;stroke:"+themeColor+";stroke-width:20") // route "line"
		for stop := range route.Stops {
			if stop == 0 || stop == len(route.Stops)-1 {
				s.Circle(130, pageStart[rt]+630+stopPos[stop], 20, "fill:white;stroke:"+themeColor+";stroke-width:10") // first and last stop
			} else {
				s.Circle(130, pageStart[rt]+630+stopPos[stop], 25, "fill:"+themeColor+";stroke:white;stroke-width:10") // other stops
			}
		}
		s.Text(1820, pageStart[rt]+610+vertFix, "© OpenStreetMap contributors", "font-family:Fira Sans;text-anchor:end;font-size:20px;fill:black") // ©
	}
	s.End()
}
