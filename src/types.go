package main

import (
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
	elapsed time.Duration // tiempo desde que entró por primera vez en la fase 1
	car     int           // id del coche
	phase   int           // fase en la que ocurre el evento
	status  string        // "entra", "sale"
	issue   IssueType     // tipo de incidencia
}

// ===================== Garage compartido =====================

type Garage struct {
	mu   sync.RWMutex
	cars map[int]*Car

	freeSlots chan struct{} // uso tokens de estos para determinar las plazas libres; es mas eficiente

	wg sync.WaitGroup
}
