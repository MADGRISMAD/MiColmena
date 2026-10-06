// Package ratelimit limita los intentos de acceso para frenar la fuerza bruta y el registro en masa.
// Todo vive en memoria: sirve para una sola instancia de la API. Si se escala a varias,
// el estado tendría que moverse a un almacén compartido (Redis o PostgreSQL).
package ratelimit

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

const (
	// idleTTL es el tiempo sin actividad tras el cual se olvida una clave.
	idleTTL = 10 * time.Minute
	// sweepEvery es cada cuánto se revisan las claves inactivas.
	sweepEvery = time.Minute
	// maxKeys acota la memoria si alguien prueba con millones de claves distintas.
	maxKeys = 100_000
)

// Limiter es un cubo de fichas por clave (por ejemplo, por IP): permite `burst` intentos
// seguidos y recupera `perMinute` fichas cada minuto.
type Limiter struct {
	mu        sync.Mutex
	limit     rate.Limit
	burst     int
	entries   map[string]*bucket
	lastSweep time.Time
	now       func() time.Time
}

type bucket struct {
	limiter *rate.Limiter
	last    time.Time
}

// New crea un limitador que permite perMinute intentos por minuto y por clave.
func New(perMinute int) *Limiter {
	return newLimiter(perMinute, time.Now)
}

func newLimiter(perMinute int, now func() time.Time) *Limiter {
	if perMinute < 1 {
		perMinute = 1
	}
	return &Limiter{
		limit:     rate.Every(time.Minute / time.Duration(perMinute)),
		burst:     perMinute,
		entries:   make(map[string]*bucket),
		lastSweep: now(),
		now:       now,
	}
}

// Allow gasta una ficha de la clave. Si no quedan, devuelve false y cuánto falta para la siguiente.
func (l *Limiter) Allow(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	l.sweep(now)

	b, ok := l.entries[key]
	if !ok {
		b = &bucket{limiter: rate.NewLimiter(l.limit, l.burst)}
		l.entries[key] = b
	}
	b.last = now

	r := b.limiter.ReserveN(now, 1)
	if !r.OK() {
		return false, time.Minute
	}
	if d := r.DelayFrom(now); d > 0 {
		r.CancelAt(now) // no gastar la ficha de un intento rechazado
		return false, d
	}
	return true, 0
}

func (l *Limiter) sweep(now time.Time) {
	if now.Sub(l.lastSweep) < sweepEvery && len(l.entries) < maxKeys {
		return
	}
	l.lastSweep = now
	for key, b := range l.entries {
		if now.Sub(b.last) > idleTTL {
			delete(l.entries, key)
		}
	}
	// Si aun así hay demasiadas, se descartan claves al azar: es preferible a quedarse sin memoria.
	for key := range l.entries {
		if len(l.entries) < maxKeys {
			break
		}
		delete(l.entries, key)
	}
}

// Lockout bloquea una clave (por ejemplo, un email) tras varios fallos seguidos.
type Lockout struct {
	mu       sync.Mutex
	max      int
	duration time.Duration
	entries  map[string]*failures
	now      func() time.Time
	lastGC   time.Time
}

type failures struct {
	count       int
	last        time.Time
	lockedUntil time.Time
}

// NewLockout bloquea durante `duration` tras `max` fallos seguidos. Los fallos más viejos
// que `duration` dejan de contar.
func NewLockout(max int, duration time.Duration) *Lockout {
	return newLockout(max, duration, time.Now)
}

func newLockout(max int, duration time.Duration, now func() time.Time) *Lockout {
	if max < 1 {
		max = 1
	}
	return &Lockout{max: max, duration: duration, entries: make(map[string]*failures), now: now, lastGC: now()}
}

// Locked indica si la clave está bloqueada y cuánto falta para que se libere.
func (l *Lockout) Locked(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	f, ok := l.entries[key]
	if !ok {
		return false, 0
	}
	if remaining := f.lockedUntil.Sub(l.now()); remaining > 0 {
		return true, remaining
	}
	return false, 0
}

// Fail registra un fallo. Al llegar al máximo, bloquea la clave.
func (l *Lockout) Fail(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	l.gc(now)

	f, ok := l.entries[key]
	if !ok || now.Sub(f.last) > l.duration {
		f = &failures{}
		l.entries[key] = f
	}
	f.count++
	f.last = now
	if f.count >= l.max {
		f.lockedUntil = now.Add(l.duration)
		f.count = 0
	}
}

// Reset olvida los fallos de la clave (tras un acceso correcto).
func (l *Lockout) Reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.entries, key)
}

func (l *Lockout) gc(now time.Time) {
	if now.Sub(l.lastGC) < sweepEvery && len(l.entries) < maxKeys {
		return
	}
	l.lastGC = now
	for key, f := range l.entries {
		if now.Sub(f.last) > l.duration && !now.Before(f.lockedUntil) {
			delete(l.entries, key)
		}
	}
	for key := range l.entries {
		if len(l.entries) < maxKeys {
			break
		}
		delete(l.entries, key)
	}
}
