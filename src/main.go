package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// ===================== Tipos base =====================

type IssueType string

const (
	MECH     IssueType = "Mecánica"
	ELECTRIC IssueType = "Eléctrica"
	BODY     IssueType = "Carrocería"
)
const (
	DOCPHASE      = 1
	REPAIRPHASE   = 2
	CLEANPHASE    = 3
	DELIVERYPHASE = 4
)

type Car struct {
	id       int
	issue    IssueType
	duration time.Duration
	curphase int
	start    time.Time
}

// Para mandar mensajes al hilo principal (logger)
type Event struct {
	elapsed time.Duration
	car     int
	phase   int
	status  string // "entra", "sale", "entregado", etc.
	issue   IssueType
}

// ===================== Garage compartido =====================

type Garage struct {
	mu   sync.RWMutex
	cars map[int]*Car

	freeSlots chan struct{}

	wg sync.WaitGroup
}

func (g *Garage) newGarage(numSlots int) *Garage {
	newg := &Garage{
		cars:      make(map[int]*Car),
		freeSlots: make(chan struct{}, numSlots),
	}

	// Inicializar plazas libres
	for i := 0; i < numSlots; i++ {
		newg.freeSlots <- struct{}{}
	}
	return newg
}

// ----- Métodos protegidos por RWMutex -----

func (g *Garage) signInCar(c *Car) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.cars[c.id] = c
}

func (g *Garage) updatePhase(id int, phase int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if c, ok := g.cars[id]; ok {
		c.curphase = phase
	}
}

func (g *Garage) delCar(id int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.cars, id)
}

/*func (g *Garage) getCar(id int) *Car {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.cars[id]
}*/

// ===================== Fases con pools de workers =====================

func genEvent(events chan<- Event, c *Car, phase int, sts string) {
	events <- Event{
		elapsed: time.Since(c.start),
		car:     c.id,
		phase:   phase,
		status:  sts,
		issue:   c.issue,
	}
}

func worker(g *Garage, entry <-chan *Car, exit chan<- *Car, events chan<- Event, phase int) {
	for c := range entry {
		g.updatePhase(c.id, phase)

		genEvent(events, c, phase, "entra")
		time.Sleep(c.duration)
		genEvent(events, c, phase, "sale")
		// Pasamos a reparación
		if phase == DELIVERYPHASE {
			g.delCar(c.id)

			// Liberar una plaza (podrá entrar otro Car en fase 1)
			g.freeSlots <- struct{}{}

			// Este Car ha terminado TODO el ciclo
			g.wg.Done()
		}
		if exit != nil {
			exit <- c
		}
	}
}

// funcion que genera workers para cada fase
func startPhase(g *Garage, nWorkers int, entry <-chan *Car, exit chan<- *Car, events chan<- Event, phase int) {
	for i := 0; i < nWorkers; i++ {
		go worker(g, entry, exit, events, phase)
	}
}

// ===================== Logger =====================

func logger(events <-chan Event) {
	for e := range events {
		secs := e.elapsed.Seconds()

		fmt.Printf("Tiempo: [%.2fs] Coche: %d Incidencia: %s Fase: %d Estado: %s\n",
			secs,
			e.car,
			e.issue,
			e.phase,
			e.status,
		)

	}
}

// ===================== Utilidades =====================

func randDecimal() float64 {
	rand.Seed(time.Now().UnixNano())
	n := rand.Intn(21) // 0..20
	return float64(n) / 10.0
}

func genCar(id int) *Car {
	// Aquí puedes meter prioridades, tiempos distintos por tipo, etc.
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
		start:    time.Now(),
	}
}

func genCars(n int) map[int]*Car {
	var cars map[int]*Car = make(map[int]*Car)

	for i := 0; i < n; i++ {
		cars[i] = genCar(i)
	}
	return cars
}

func getCar(cars map[int]*Car) *Car {

	for _, c := range cars {
		if c.issue == MECH {
			return c
		}
	}
	for _, c := range cars {
		if c.issue == ELECTRIC {
			return c
		}
	}
	for _, c := range cars {
		if c.issue == BODY {
			return c
		}
	}
	return nil
}

// ===================== Función main =====================

func main() {
	var g *Garage
	numCars := 20
	numSlots := 10
	numMechs := 4

	numWorkersDoc := numSlots
	numWorkersRep := numMechs // reparación limitada por mecánicos
	numWorkersLimpieza := numSlots
	numWorkersEntrega := numSlots

	// Canales entre fases
	docChan := make(chan *Car)
	repChan := make(chan *Car)
	cleanChan := make(chan *Car)
	deliverChan := make(chan *Car)

	// Canal para events hacia el logger
	events := make(chan Event)

	g = g.newGarage(numSlots)

	// Lanzar logger (único que imprime por stdout)
	go logger(events)

	// Lanzar workers de cada fase

	// Fase de documentacion:
	startPhase(g, numWorkersDoc, docChan, repChan, events, DOCPHASE)

	// Fase de reparacion
	startPhase(g, numWorkersRep, repChan, cleanChan, events, REPAIRPHASE)

	// Fase de limpieza
	startPhase(g, numWorkersLimpieza, cleanChan, deliverChan, events, CLEANPHASE)

	// Fase de entrega
	startPhase(g, numWorkersEntrega, deliverChan, nil, events, DELIVERYPHASE)

	everycars := genCars(numCars)
	// Generación de Cars (productor)
	for i := 0; i < numCars; i++ {
		// Esperar a tener una plaza libre
		<-g.freeSlots

		c := getCar(everycars)

		// Registrar el Car en el mapa global
		g.signInCar(c)

		// Este Car hará todo el ciclo -> lo contamos en el WaitGroup
		g.wg.Add(1)

		// Entra en la primera fase (documentación)
		docChan <- c

		// (opcional) para que no entren todos exacto a la vez:
		// time.Sleep(50 * time.Millisecond)
	}

	// IMPORTANTE:
	// Aquí podrías cerrar docChan cuando hayas mandado todos los Cars
	// y propagar los cierres hacia abajo (cuando todos los workers terminen).
	// Para la práctica básica no es estrictamente necesario, porque
	// cuando main termine, el programa se acaba y las goroutines se terminan.

	// Esperar a que TODOS los Cars terminen la fase de entrega
	g.wg.Wait()

	// Ahora podemos cerrar el canal de events para que el logger termine
	close(events)

	fmt.Println("Simulación completada.")
}
