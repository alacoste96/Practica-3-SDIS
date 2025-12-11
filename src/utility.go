package main

import (
	"math/rand"
	"time"
)

func newGarage(numSlots int) *Garage {
	g := &Garage{
		cars:      make(map[int]*Car),
		freeSlots: make(chan struct{}, numSlots),
	}
	for i := 0; i < numSlots; i++ {
		g.freeSlots <- struct{}{}
	}
	return g
}

func freeThreats(docChan, repChan, cleanChan, deliverChan chan struct{}) {
	docChan <- struct{}{}
	repChan <- struct{}{}
	cleanChan <- struct{}{}
	deliverChan <- struct{}{}
	close(docChan)
	close(repChan)
	close(cleanChan)
	close(deliverChan)
}

func getCarFromQ(cars map[int]*Car) *Car {
	var chosen *Car

	for id, c := range cars {
		if c.issue == MECH {
			chosen = c
			delete(cars, id)
			return chosen
		}
	}
	for id, c := range cars {
		if c.issue == ELECTRIC {
			chosen = c
			delete(cars, id)
			return chosen
		}
	}
	for id, c := range cars {
		if c.issue == BODY {
			chosen = c
			delete(cars, id)
			return chosen
		}
	}
	return nil
}

func genCars(ncars int) map[int]*Car {
	var i int

	cars := make(map[int]*Car, ncars)
	for i = 0; i < ncars; i++ {
		cars[i] = genCar(i)
	}
	return cars
}

// ===================== Utilidades =====================

func randDecimal() float64 {
	n := rand.Intn(21) // 0..20
	return float64(n) / 10.0
}

// genera un coche con prio aleatoria
func genCar(id int) *Car {
	var issue IssueType
	var duration time.Duration
	var lag float64
	var prio int = rand.Intn(3)

	lag = randDecimal()
	switch prio {
	case 0: // mecánica
		issue = MECH
		duration = time.Duration((5 + lag) * float64(time.Second))
	case 1: // eléctrica
		issue = ELECTRIC
		duration = time.Duration((3 + lag) * float64(time.Second))
	case 2: // carroceria
		issue = BODY
		duration = time.Duration((1 + lag) * float64(time.Second))
	}

	return &Car{
		id:       id,
		issue:    issue,
		duration: duration,
		curphase: 0,
	}
}

// obtiene un coche a traves de un canal de comunicacion entre hilos, con prioridades
func getCar(chans [3]chan *Car, stop <-chan struct{}) *Car {
	var c *Car

	select {
	case c = <-chans[0]: // pillo el de alta prio
	default:
		select {
		case c = <-chans[1]: // pillo el de prio media
		default:
			select { // pillo el de baja prio o espero por el primero que llegue
			case c = <-chans[0]:
			case c = <-chans[1]:
			case c = <-chans[2]:
			case <-stop:
				return nil
			}
		}
	}
	return c
}

// envía un coche a traves de un canal en funcion de su prioridad
func sendCar(chans [3]chan *Car, c *Car) {
	switch c.issue {
	case MECH:
		chans[0] <- c // prio alta
	case ELECTRIC:
		chans[1] <- c // prio media
	case BODY:
		chans[2] <- c // prio baja
	}
}

// inicializa los canales de una fase (1 por cada prioridad)
// indice 0 = prio alta. indice 1 = prio media. indice 2 = prio baja
func initPhaseChans() [3]chan *Car {
	var chans [3]chan *Car

	for i := range chans {
		chans[i] = make(chan *Car)
	}
	return chans
}

func closeChans(chans [3]chan *Car) {
	for i := range chans {
		close(chans[i])
	}
}

// genera un evento que se envía al logger para que este escriba por stdout
func genEvent(events chan<- Event, c *Car, sts string) {
	events <- Event{
		elapsed: time.Since(c.start),
		car:     c.id,
		phase:   c.curphase,
		status:  sts,
		issue:   c.issue,
	}
}
