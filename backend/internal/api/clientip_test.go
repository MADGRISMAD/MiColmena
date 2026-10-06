package api

import (
	"net/http/httptest"
	"testing"
)

func TestClientIP(t *testing.T) {
	tests := []struct {
		name       string
		remote     string
		xff        string
		trustProxy bool
		want       string
	}{
		{"directo", "203.0.113.9:5000", "", false, "203.0.113.9"},
		{"ignora X-Forwarded-For sin proxy de confianza", "203.0.113.9:5000", "1.2.3.4", false, "203.0.113.9"},
		{"proxy de confianza: usa el último valor", "10.0.0.5:5000", "9.9.9.9, 203.0.113.50", true, "203.0.113.50"},
		{"proxy de confianza: un solo valor", "10.0.0.5:5000", "203.0.113.50", true, "203.0.113.50"},
		{"proxy de confianza sin cabecera", "10.0.0.5:5000", "", true, "10.0.0.5"},
		{"proxy de confianza con basura en la cabecera", "10.0.0.5:5000", "no-es-una-ip", true, "10.0.0.5"},
		{"IPv6 se agrupa por /64", "[2001:db8:1:2:aaaa:bbbb:cccc:dddd]:5000", "", false, "2001:db8:1:2::/64"},
		{"dos IPv6 del mismo /64 comparten clave", "[2001:db8:1:2:1111:2222:3333:4444]:5000", "", false, "2001:db8:1:2::/64"},
		{"IPv4 mapeada en IPv6", "[::ffff:203.0.113.9]:5000", "", false, "203.0.113.9"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/", nil)
			r.RemoteAddr = tt.remote
			if tt.xff != "" {
				r.Header.Set("X-Forwarded-For", tt.xff)
			}
			if got := clientIP(r, tt.trustProxy); got != tt.want {
				t.Errorf("clientIP = %q, se esperaba %q", got, tt.want)
			}
		})
	}
}

func TestWaitText(t *testing.T) {
	for seconds, want := range map[int]string{1: "1 segundo", 45: "45 segundos", 89: "89 segundos", 90: "2 minutos", 900: "15 minutos"} {
		if got := waitText(seconds); got != want {
			t.Errorf("waitText(%d) = %q, se esperaba %q", seconds, got, want)
		}
	}
}
