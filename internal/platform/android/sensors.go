package android

import (
	"sync"

	"github.com/rm4n0s/glutter/internal/platform"
)

// sensorBackend is the motion sensors service of Android (the sensor manager of the NDK, in package asensor) as the platform uses it:
// the service itself, and the way to end it with the app.
type sensorBackend interface {
	platform.Sensors
	Close()
}

// sensorState is the sensors service of the platform, made by the first call of Services.
type sensorState struct {
	once sync.Once
	mu   sync.Mutex
	svc  sensorBackend
}

// sensorService returns the sensors service, made on the first call: nil when there is no backend for the system the program runs
// on. Nothing is loaded until the app asks for a sensor.
func (p *Platform) sensorService() platform.Sensors {
	p.sensors.once.Do(func() {
		if svc := newSensorBackend(p); svc != nil {
			p.sensors.mu.Lock()
			p.sensors.svc = svc
			p.sensors.mu.Unlock()
		}
	})
	p.sensors.mu.Lock()
	defer p.sensors.mu.Unlock()
	if p.sensors.svc == nil {
		return nil
	}
	return p.sensors.svc
}

// closeSensors ends the sensors service: the sensors that are open are closed and the thread of the looper ends.
func (p *Platform) closeSensors() {
	p.sensors.mu.Lock()
	svc := p.sensors.svc
	p.sensors.svc = nil
	p.sensors.mu.Unlock()
	if svc != nil {
		svc.Close()
	}
}
