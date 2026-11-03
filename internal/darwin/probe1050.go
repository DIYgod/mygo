//go:build darwin

package darwin

import (
	"log"
	"os"

)

// Throwaway probe for #1050: never merged.

var ProbeVeto = os.Getenv("MYGO_PROBE_VETO") == "1"

func probeLog(format string, args ...any) {
	if os.Getenv("MYGO_PROBE") != "" {
		log.Printf("probe: "+format, args...)
	}
}

func probeEvent(ev id) string {
	if ev == 0 {
		return "nil"
	}
	typ := sendInt(ev, "type")
	s := ""
	switch typ {
	case 1, 2, 3, 4, 25, 26:
		s = " num=" + itoa(sendInt(ev, "eventNumber"))
	}
	status := ""
	for _, t := range theBackend.trays {
		if send(ev, "window") == send(t.button, "window") && send(ev, "window") != 0 {
			status = " (status window)"
		}
	}
	return "type=" + itoa(typ) + " win=" + itoa(int(send(ev, "window"))) + s + status
}

func itoa(n int) string {
	if n < 0 {
		return "-" + itoa(-n)
	}
	if n < 10 {
		return string(rune('0' + n))
	}
	return itoa(n/10) + string(rune('0'+n%10))
}

