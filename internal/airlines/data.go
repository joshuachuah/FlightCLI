// Package airlines provides an embedded ICAO/IATA airline code dataset
// derived from the OpenFlights airlines database.
//
// Data source: OpenFlights Airlines Database
// https://github.com/jpatokal/openflights/blob/master/data/airlines.dat
// License: Open Database License (ODbL) — see NOTICE.txt
// Generated: 2026-05-02
package airlines

import "strings"

// Airline holds metadata for a single airline.
type Airline struct {
	Name     string // Airline name
	IATA     string // 2-character IATA code (e.g. "UA")
	ICAO     string // 3-character ICAO code (e.g. "UAL")
	Callsign string // Radio callsign
	Country  string // Country of origin
}

// ByICAO returns airline metadata for a 3-letter ICAO code, or nil.
func ByICAO(icao string) *Airline {
	icao = strings.ToUpper(strings.TrimSpace(icao))
	if a, ok := icaoToAirline[icao]; ok {
		return &a
	}
	return nil
}

// ByIATA returns airline metadata for a 2-character IATA code, or nil.
func ByIATA(iata string) *Airline {
	iata = strings.ToUpper(strings.TrimSpace(iata))
	if a, ok := iataToAirline[iata]; ok {
		return &a
	}
	return nil
}

// IATACode returns the IATA code for a given ICAO code, or "" if not found.
func IATACode(icao string) string {
	if a := ByICAO(icao); a != nil {
		return a.IATA
	}
	return ""
}

// ICAOCode returns the ICAO code for a given IATA code, or "" if not found.
// If multiple airlines share the same IATA code, the first is returned.
func ICAOCode(iata string) string {
	if a := ByIATA(iata); a != nil {
		return a.ICAO
	}
	return ""
}

// IsICAOCode returns true if the 3-letter prefix looks like a valid ICAO
// airline designator present in our dataset.
func IsICAOCode(prefix string) bool {
	prefix = strings.ToUpper(strings.TrimSpace(prefix))
	_, ok := icaoToAirline[prefix]
	return ok
}
