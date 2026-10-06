package ratelimit

import (
	"testing"
	"time"
)

type clock struct{ t time.Time }

func (c *clock) now() time.Time          { return c.t }
func (c *clock) advance(d time.Duration) { c.t = c.t.Add(d) }

func newClock() *clock { return &clock{t: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)} }

func TestLimiterBlocksAfterBurst(t *testing.T) {
	c := newClock()
	l := newLimiter(10, c.now)

	for i := 1; i <= 10; i++ {
		if ok, _ := l.Allow("1.2.3.4"); !ok {
			t.Fatalf("el intento %d debería pasar", i)
		}
	}
	ok, retry := l.Allow("1.2.3.4")
	if ok {
		t.Fatal("el intento 11 debería rechazarse")
	}
	if retry <= 0 || retry > 7*time.Second {
		t.Errorf("retry-after inesperado: %v (se esperan ~6 s)", retry)
	}
}

func TestLimiterIsPerKey(t *testing.T) {
	l := newLimiter(1, newClock().now)
	if ok, _ := l.Allow("a"); !ok {
		t.Fatal("a debería pasar")
	}
	if ok, _ := l.Allow("a"); ok {
		t.Fatal("a debería estar limitada")
	}
	if ok, _ := l.Allow("b"); !ok {
		t.Fatal("b no debería verse afectada por a")
	}
}

func TestLimiterRecovers(t *testing.T) {
	c := newClock()
	l := newLimiter(10, c.now)
	for i := 0; i < 10; i++ {
		l.Allow("k")
	}
	if ok, _ := l.Allow("k"); ok {
		t.Fatal("debería estar limitada")
	}
	c.advance(7 * time.Second) // una ficha cada 6 s
	if ok, _ := l.Allow("k"); !ok {
		t.Fatal("debería haber recuperado una ficha")
	}
}

func TestLimiterRejectedAttemptsDoNotExtendTheWait(t *testing.T) {
	c := newClock()
	l := newLimiter(10, c.now)
	for i := 0; i < 10; i++ {
		l.Allow("k")
	}
	_, first := l.Allow("k")
	for i := 0; i < 50; i++ {
		l.Allow("k") // insistir no debe alargar la espera
	}
	_, last := l.Allow("k")
	if last > first {
		t.Errorf("la espera creció de %v a %v por intentos rechazados", first, last)
	}
}

func TestLimiterForgetsIdleKeys(t *testing.T) {
	c := newClock()
	l := newLimiter(10, c.now)
	l.Allow("viejo")
	c.advance(idleTTL + 2*sweepEvery)
	l.Allow("nuevo")
	if _, ok := l.entries["viejo"]; ok {
		t.Error("la clave inactiva debería haberse eliminado")
	}
}

func TestLockout(t *testing.T) {
	c := newClock()
	l := newLockout(5, 15*time.Minute, c.now)

	for i := 0; i < 4; i++ {
		l.Fail("ana@x.com")
		if locked, _ := l.Locked("ana@x.com"); locked {
			t.Fatalf("no debería bloquearse tras %d fallos", i+1)
		}
	}
	l.Fail("ana@x.com")
	locked, remaining := l.Locked("ana@x.com")
	if !locked || remaining != 15*time.Minute {
		t.Fatalf("debería estar bloqueada 15 min, got locked=%v remaining=%v", locked, remaining)
	}
	if locked, _ := l.Locked("otra@x.com"); locked {
		t.Error("otra clave no debería verse afectada")
	}

	c.advance(15*time.Minute + time.Second)
	if locked, _ := l.Locked("ana@x.com"); locked {
		t.Error("el bloqueo debería haber expirado")
	}
}

func TestLockoutResetOnSuccess(t *testing.T) {
	l := newLockout(3, time.Minute, newClock().now)
	l.Fail("k")
	l.Fail("k")
	l.Reset("k")
	l.Fail("k")
	l.Fail("k")
	if locked, _ := l.Locked("k"); locked {
		t.Error("tras Reset, los fallos anteriores no deberían contar")
	}
}

func TestLockoutOldFailuresExpire(t *testing.T) {
	c := newClock()
	l := newLockout(3, time.Minute, c.now)
	l.Fail("k")
	l.Fail("k")
	c.advance(2 * time.Minute)
	l.Fail("k") // el contador vuelve a empezar
	if locked, _ := l.Locked("k"); locked {
		t.Error("fallos viejos no deberían sumar")
	}
}
