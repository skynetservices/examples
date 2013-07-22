package main

import (
	"github.com/skynetservices/skynet2"
	"github.com/skynetservices/skynet2/log"
	"github.com/skynetservices/skynet2/service"
	"github.com/skynetservices/skynet2/stats"
	"github.com/skynetservices/zkmanager"
	"os"
	"strings"
	"time"
)

var led *LED = NewLED()
var registered bool

const (
	RED = iota
	GREEN
	BLUE
)

type LedReporter struct {
	blinkChan chan int
}

func (r *LedReporter) UpdateHostStats(host string, stats stats.Host) {}
func (r *LedReporter) MethodCalled(method string)                    {}
func (r *LedReporter) MethodCompleted(method string, duration int64, err error) {
	if err != nil {
		r.blinkChan <- RED
	} else {
		r.blinkChan <- GREEN
	}
}

func (r *LedReporter) watch() {
	var t *time.Ticker = time.NewTicker(1 * time.Second)

	for {
		select {
		case color := <-r.blinkChan:
			t.Stop()

			switch color {
			case RED:
				led.Red(true)
			case BLUE:
				led.Blue(true)
			case GREEN:
				led.Green(true)
			}

			t = time.NewTicker(250 * time.Millisecond)
		case <-t.C:
			if registered {
				led.Blue(true)
			} else {
				led.Off()
			}
		}
	}
}

func NewLedReporter() (r *LedReporter) {
	r = &LedReporter{}
	go r.watch()

	return
}

type PiDemoService struct {
}

func (s *PiDemoService) Registered(service *service.Service) {
	registered = true
	led.Blue(true)
}
func (s *PiDemoService) Unregistered(service *service.Service) {
	registered = false
	led.Blue(false)
}

func (s *PiDemoService) Started(service *service.Service) {}
func (s *PiDemoService) Stopped(service *service.Service) {
}

func NewPiDemoService() *PiDemoService {
	r := &PiDemoService{}
	return r
}

func (s *PiDemoService) Upcase(requestInfo *skynet.RequestInfo, in map[string]interface{}, out map[string]interface{}) (err error) {
	out["data"] = strings.ToUpper(in["data"].(string))
	return
}

func main() {
	log.SetLogLevel(log.DEBUG)
	stats.AddReporter(NewLedReporter())

	skynet.SetServiceManager(zkmanager.NewZookeeperServiceManager(os.Getenv("SKYNET_ZOOKEEPER"), 1*time.Second))

	testService := NewPiDemoService()

	config, _ := skynet.GetServiceConfig()

	if config.Name == "" {
		config.Name = "PiDemoService"
	}

	if config.Version == "unknown" {
		config.Version = "1"
	}

	if config.Region == "unknown" {
		config.Region = "Clearwater"
	}

	service := service.CreateService(testService, config)

	// handle panic so that we remove ourselves from the pool in case
	// of catastrophic failure
	defer func() {
		service.Shutdown()
		if err := recover(); err != nil {
			log.Panic("Unrecovered error occured: ", err)
		}
	}()

	waiter := service.Start()

	// waiting on the sync.WaitGroup returned by service.Start() will
	// wait for the service to finish running.
	waiter.Wait()
}