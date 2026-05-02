package airlines

import "testing"

func TestEmbeddedDatasetInvariants(t *testing.T) {
	for icao, airline := range icaoToAirline {
		if len(icao) != 3 {
			t.Errorf("ICAO key %q has length %d, want 3", icao, len(icao))
		}
		for _, c := range icao {
			if c < 'A' || c > 'Z' {
				t.Errorf("ICAO key %q contains non-A-Z rune %q", icao, string(c))
			}
		}
		if airline.ICAO != icao {
			t.Errorf("ICAO key %q points to airline ICAO %q", icao, airline.ICAO)
		}
		if airline.Name == "" || airline.IATA == "" || airline.ICAO == "" || airline.Country == "" {
			t.Errorf("airline for ICAO %q has empty required fields: %#v", icao, airline)
		}
		if len(airline.IATA) < 1 || len(airline.IATA) > 2 {
			t.Errorf("airline for ICAO %q has IATA %q length %d, want 1-2", icao, airline.IATA, len(airline.IATA))
		}
		for _, c := range airline.IATA {
			if (c < 'A' || c > 'Z') && (c < '0' || c > '9') {
				t.Errorf("airline for ICAO %q has non-alphanumeric IATA %q", icao, airline.IATA)
			}
		}
	}

	for iata, airline := range iataToAirline {
		if airline.IATA != iata {
			t.Errorf("IATA key %q points to airline IATA %q", iata, airline.IATA)
		}
		if ByICAO(airline.ICAO) == nil {
			t.Errorf("IATA key %q points to missing ICAO %q", iata, airline.ICAO)
		}
	}
}

func TestEmbeddedDatasetIncludesLegacyCoverage(t *testing.T) {
	legacyICAOCodes := []string{
		"AAL", "UAL", "DAL", "SWA", "JBU", "ASA", "HAL", "SKW", "RPA",
		"ENY", "AAY", "FFT", "NKS", "ASH", "BAW", "ACA", "TSC", "AFR",
		"DLH", "KLM", "SAS", "AIC", "IGO", "CPA", "SIA", "ANA", "JAL",
		"KAL", "THA", "MAS", "GIA", "ETH", "QFA", "ANZ", "UAE", "ETD",
		"QTR", "THY", "AEE", "RYR", "EZY", "DLA", "EWG", "VIR", "TAP",
		"IBE", "AZA", "SWR", "AUA", "FIN", "CSN", "CCA", "CSZ", "CXA",
		"CES", "CRK", "AMX", "AVA", "LAN", "TAM",
	}

	for _, icao := range legacyICAOCodes {
		if ByICAO(icao) == nil {
			t.Errorf("expected legacy ICAO code %q to be present", icao)
		}
	}

	if got := IATACode("ENY"); got != "MQ" {
		t.Fatalf("expected ENY to map to MQ, got %q", got)
	}
	if got := IATACode("SCX"); got != "SY" {
		t.Fatalf("expected SCX to map to SY, got %q", got)
	}
}
