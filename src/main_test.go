package main

import (
	"testing"
	"time"
)

// Crea un coche con incidencia fija (A/B/C).
func genCarWithIssue(id int, issue IssueType) *Car {
	lag := randDecimal() // 0.0 .. 2.0

	var base float64
	switch issue {
	case MECH: // Categoría A: 5s
		base = 5
	case ELECTRIC: // Categoría B: 3s
		base = 3
	case BODY: // Categoría C: 1s
		base = 1
	}

	return &Car{
		id:       id,
		issue:    issue,
		duration: time.Duration((base + lag) * float64(time.Second)),
		curphase: 0,
	}
}

func noEvents(events chan Event) {
	for range events {
	}
}

func genSpecificCars(ncars, nA, nB, nC int) map[int]*Car {
	cars := make(map[int]*Car, ncars)
	id := 0
	for i := 0; i < nA; i++ {
		cars[id] = genCarWithIssue(id, MECH)
		id++
	}
	for i := 0; i < nB; i++ {
		cars[id] = genCarWithIssue(id, ELECTRIC)
		id++
	}
	for i := 0; i < nC; i++ {
		cars[id] = genCarWithIssue(id, BODY)
		id++
	}
	return cars
}

// Ejecuta una simulación con número de coches por categoría A/B/C.
// Devuelve la duración total de la simulación.
func runSimulation(nA, nB, nC int, numSlots, numMechs int, withLogs bool) time.Duration {
	numCars := nA + nB + nC

	numWorkersDoc := numSlots
	numWorkersRep := numMechs
	numWorkersClean := numSlots
	numWorkersDeliver := numSlots

	// Canales entre fases (prioridad alta/media/baja)
	docChans := initPhaseChans()
	repChans := initPhaseChans()
	cleanChans := initPhaseChans()
	deliverChans := initPhaseChans()
	var noExits [3]chan *Car // fase 4 no reenvía a nadie

	// canal para eventos
	events := make(chan Event)
	// canales para terminar a las goroutines
	stopDoc := make(chan struct{})
	stopRep := make(chan struct{})
	stopClean := make(chan struct{})
	stopDeliver := make(chan struct{})

	g := newGarage(numSlots)

	// Logger: para tests no quiero ruido en stdout
	// así que si withLogs==false simplemente silencio los eventos.
	if withLogs {
		go logger(events)
	} else {
		go noEvents(events)
	}

	// Lanzar workers de cada fase
	startPhase(g, numWorkersDoc, docChans, repChans, events, DOCPHASE, stopDoc)
	startPhase(g, numWorkersRep, repChans, cleanChans, events, REPAIRPHASE, stopRep)
	startPhase(g, numWorkersClean, cleanChans, deliverChans, events, CLEANPHASE, stopClean)
	startPhase(g, numWorkersDeliver, deliverChans, noExits, events, DELIVERYPHASE, stopDeliver)

	// ----- Generación de coches con distribución A/B/C -----
	cars := genSpecificCars(numCars, nA, nB, nC)

	start := time.Now()

	// Productor: mete todos los coches en la fase 1 respetando numSlots
	for i := 0; i < numCars; i++ {
		<-g.freeSlots // espera plaza libre
		c := getCarFromQ(cars)
		c.start = time.Now() // inicio del cronómetro de este coche
		g.signInCar(c)       // lo registramos en el mapa
		g.wg.Add(1)          // este coche hará todo el ciclo
		sendCar(docChans, c) // entra en la primera fase
	}

	// Esperar a que TODOS los coches terminen
	g.wg.Wait()
	freeThreats(stopDoc, stopRep, stopClean, stopDeliver)
	close(events)
	closeChans(docChans)
	closeChans(repChans)
	closeChans(cleanChans)
	closeChans(deliverChans)
	close(g.freeSlots)
	return time.Since(start)
}

// ===================== Tests de los 3 escenarios del enunciado =====================

// Test 1: A 10, B 10, C 10
func TestEscenario1_A10_B10_C10(t *testing.T) {
	const sims = 5
	numSlots := 10
	numMechs := 4

	var total time.Duration
	for i := 0; i < sims; i++ {
		dur := runSimulation(10, 10, 10, numSlots, numMechs, false)
		total += dur
	}

	media := total / sims
	t.Logf("Escenario 1 (A=10, B=10, C=10), %d simulaciones, tiempo medio: %v", sims, media)
}

// Test 2: A 20, B 5, C 5
func TestEscenario2_A20_B5_C5(t *testing.T) {
	const sims = 5
	numSlots := 10
	numMechs := 4

	var total time.Duration
	for i := 0; i < sims; i++ {
		dur := runSimulation(20, 5, 5, numSlots, numMechs, false)
		total += dur
	}

	media := total / sims
	t.Logf("Escenario 2 (A=20, B=5, C=5), %d simulaciones, tiempo medio: %v", sims, media)
}

// Test 3: A 5, B 5, C 20
func TestEscenario3_A5_B5_C20(t *testing.T) {
	const sims = 5
	numSlots := 10
	numMechs := 4

	var total time.Duration
	for i := 0; i < sims; i++ {
		dur := runSimulation(5, 5, 20, numSlots, numMechs, false)
		total += dur
	}

	media := total / sims
	t.Logf("Escenario 3 (A=5, B=5, C=20), %d simulaciones, tiempo medio: %v", sims, media)
}
