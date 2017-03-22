package helpers

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/MrYadro/trosm/types"
)

func TranslateHeader(lang string) string {
	trans := make(map[string]string)
	trans["ru"] = "Схема маршрута"
	trans["en"] = "Scheme of route"
	trans["es"] = "Esquema de ruta"
	trans["de"] = "Scheme der Route"
	trans["zh"] = "路线方案"
	trans["ko"] = "노선 구성표"
	if trans[lang] != "" {
		return trans[lang]
	}
	return "Scheme of route"
}

func RemoveDuplicates(xs *[]types.Poi) {
	found := make(map[string]bool)
	j := 0
	for i, x := range *xs {
		if !found[x.Name+"_"+x.Poit] {
			found[x.Name+"_"+x.Poit] = true
			(*xs)[j] = (*xs)[i]
			j++
		}
	}
	*xs = (*xs)[:j]
}

func GetColorFromName(name string) string {
	wayColors := [10]string{"#49b45d", "#3473ba", "#f67536", "#0ebdf5", "#ffb81b",
		"#815aa1", "#d6473d", "#704233", "#909093", "#68a0bd"}
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

func Distance(lat1, lon1, lat2, lon2 float64) float64 {

	var la1, lo1, la2, lo2, r float64
	la1 = lat1 * math.Pi / 180
	lo1 = lon1 * math.Pi / 180
	la2 = lat2 * math.Pi / 180
	lo2 = lon2 * math.Pi / 180

	r = 6378100

	h := hsin(la2-la1) + math.Cos(la1)*math.Cos(la2)*hsin(lo2-lo1)

	return 2 * r * math.Asin(math.Sqrt(h))
}

func ColorOsm(color string) string {
	colorMap := make(map[string]string)
	if strings.HasPrefix(color, "#") {
		return color
	}
	colorMap["black"] = "#000000"
	colorMap["gray"] = "#808080"
	colorMap["grey"] = colorMap["gray"]
	colorMap["maroon"] = "#800000"
	colorMap["olive"] = "#808000"
	colorMap["green"] = "#008000"
	colorMap["teal"] = "#008080"
	colorMap["navy"] = "#000080"
	colorMap["purple"] = "#800080"
	colorMap["white"] = "#FFFFFF"
	colorMap["silver"] = "#C0C0C0"
	colorMap["red"] = "#FF0000"
	colorMap["yellow"] = "#FFFF00"
	colorMap["lime"] = "#00FF00"
	colorMap["aqua"] = "#00FFFF"
	colorMap["blue"] = "#0000FF"
	colorMap["fuchsia"] = "#FF00FF"
	colorMap["magenta"] = colorMap["fuchsia"]
	if colorMap[color] == "" {
		return "#FF00FF"
	}
	return colorMap[color]
}
