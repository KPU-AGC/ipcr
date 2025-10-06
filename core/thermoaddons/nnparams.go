package thermoaddons

const Rcal = 1.987

var nnDH = map[string]float64{
	"AA": -7.9, "TT": -7.9, "AT": -7.2, "TA": -7.2,
	"CA": -8.5, "TG": -8.5, "GT": -8.4, "AC": -8.4,
	"CT": -7.8, "AG": -7.8, "GA": -8.2, "TC": -8.2,
	"CG": -10.6, "GC": -9.8, "GG": -8.0, "CC": -8.0,
}
var nnDS = map[string]float64{
	"AA": -22.2, "TT": -22.2, "AT": -20.4, "TA": -21.3,
	"CA": -22.7, "TG": -22.7, "GT": -22.4, "AC": -22.4,
	"CT": -21.0, "AG": -21.0, "GA": -22.2, "TC": -22.2,
	"CG": -27.2, "GC": -24.4, "GG": -19.9, "CC": -19.9,
}

const (
	initDH = 0.2
	initDS = -5.7
)
const symmetryDS = -1.4
