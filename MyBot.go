package main

import (
	"container/list"
	"fmt"
	"log"
	"os"
	"rand"
	"time"
)

type AntState int8
const (
	STATE_EXPLORE = iota
	STATE_HUNT_FOOD
)

type Ant struct {
	loc						Location
	target				Location
	closestFood 	Location
	exploreTarget Location
	state 				AntState
	seenThisTurn 	bool
	
	// BFS move state
	moves					*list.List
	moveTarget		Location
}

func (ant *Ant) printMoves() {
	debugDir := ""
	for e := ant.moves.Front(); e != nil; e = e.Next() {
		debugDir += fmt.Sprintf("%s,", e.Value.(Direction))
	}	
	log.Println(debugDir)	
}

const MAX_SIZE = 200
const AREAS = 12

type GarboAnt struct {
	state 				*State
	
	exploreHeat1	[MAX_SIZE*MAX_SIZE]float32
	exploreHeat2	[MAX_SIZE*MAX_SIZE]float32
	exploreHeat   *[MAX_SIZE*MAX_SIZE]float32
	exploreNext		*[MAX_SIZE*MAX_SIZE]float32
	ants					map[Location]*Ant
	foodHunted		map[Location]*Ant
	knownHills		map[Location]bool
	knownWater		map[Location]bool
	rand					rand.Rand
	
	antCountArea	[AREAS*AREAS]int
	areaOffset		int
}

func NewBot(s *State) Bot {
	me := &GarboAnt{
		ants: make(map[Location]*Ant),
		knownHills: make(map[Location]bool),
		knownWater: make(map[Location]bool),
		foodHunted: make(map[Location]*Ant),
		state: s,
	}
	me.exploreHeat = &me.exploreHeat1;
	me.exploreNext = &me.exploreHeat2;
	return me
}

func (me *GarboAnt) locToArea(loc Location) int {
	row, col := me.state.Map.FromLocation(loc)
	areaRow := int(float64(row) / float64(me.state.Map.Rows) * AREAS)
	areaCol := int(float64(col) / float64(me.state.Map.Cols) * AREAS)
	return areaRow * AREAS + areaCol
}

func (me *GarboAnt) areaToLoc(loc int) Location {
	areaRow := loc / AREAS
	areaCol := loc % AREAS
	row := int(float64(areaRow) / float64(AREAS) * float64(me.state.Map.Rows))
	col := int(float64(areaCol) / float64(AREAS) * float64(me.state.Map.Cols))
	return me.state.Map.FromRowCol(row, col)
}

func (me *GarboAnt) SearchMap(s *State, source Location, final Location) (*list.List, bool) {
	if (source == final) {
		return nil, false
	}

	type Node struct {
		current Location
		wave int
	}

	// BFS
	visited := make(map[Location]int)
	queue := new(list.List)

	dirs := []Direction{West, South, North, East}
	invDir := []Direction{East, North, South, West}
	nextStep := func(current Location, oldNode *Node) {
		for _, dir := range dirs {
			loc := s.Map.Move(current, dir)
			_, alreadyVisited := visited[loc]
			_, waterExists := me.knownWater[loc]
			_, antExists := false, false //me.ants[loc]
			if !alreadyVisited && !waterExists && !antExists {
				wave := 1
				if oldNode != nil {
					wave = oldNode.wave + 1
				}
				queue.PushBack(&Node{current: loc, wave: wave, })
			}
		}		
	}

	buildPath := func(end Location) *list.List {
		path := new(list.List)
		for end != source {
			bestWave := 9999999;
			bestDir := North
			bestLoc := Location(0)
			for _, i := range dirs {
				loc := s.Map.Move(end, dirs[i])
				_, exists := visited[loc]
				if exists && visited[loc] < bestWave {
					bestWave = visited[loc]
					bestDir = invDir[i]
					bestLoc = loc
				}
			}
			path.PushFront(bestDir)
			end = bestLoc
		}
		return path
	}

	visited[source] = 0
	nextStep(source, nil)

	for queue.Len() != 0 {
		curEl := queue.Front()
		queue.Remove(curEl)
		current := curEl.Value.(*Node)
		if current.current == final {
			// Walk back from here, constructing the path
			return buildPath(current.current), true
		}
		if _, exists := visited[current.current]; exists {
			continue;
		}
		visited[current.current] = current.wave
		nextStep(current.current, current)
	}

	return nil, false
}

//DoTurn is where you should do your bot's actual work.
func (me *GarboAnt) DoTurn(s *State) os.Error {
	startTime := time.Nanoseconds();
	
	// Mark all ants as not seen so far
	for _, ant := range me.ants {
		ant.seenThisTurn = false
	}
	
	// Check to see which ants are alive and in the place we thought they should be
	for loc, ant := range s.Map.Ants {
		if ant != MY_ANT {
			continue
		}
	 	_, exists := me.ants[loc]
		if !exists {
			me.ants[loc] = &Ant{ state: STATE_EXPLORE,}
			me.ants[loc].loc = loc
			me.antCountArea[me.locToArea(loc)]++;
		} else {
			if me.ants[loc].loc != loc {
				log.Println("Ant state corrupted")
			}			
		}
		me.ants[loc].seenThisTurn = true		
	}
	
	for loc, ant := range me.ants {
		// Remove killed ants from state
		if !ant.seenThisTurn {
			me.ants[loc] = nil, false
			me.antCountArea[me.locToArea(loc)]--;
			log.Println("Ant killed at ", loc)
		}
	}

	// Anything that can't be seen is highest priority
	for row := 0; row < s.Map.Rows; row++ {
		for col := 0; col < s.Map.Cols; col++ {
			loc := s.Map.FromRowCol(row, col)
			item := s.Map.Item(loc)

			// Track hills
			if item.IsEnemyHill() {
				me.knownHills[loc] = true
			} else {
				_, isHill := me.knownHills[loc]
				if isHill {
					// Handle hills being killed
					log.Println("Hill killed!")
					me.knownHills[loc] = false, false
				}
			}
			
			// Track water
			if item == WATER {
				me.knownWater[loc] = true
			}
			if item != FOOD {
				me.foodHunted[loc] = nil, false
			} else {
				hunter := me.foodHunted[loc]
				if hunter != nil && !hunter.seenThisTurn {
					me.foodHunted[loc] = nil, false
				}
			}
		}
	}
/*
	str := ""
	for row := 0; row < AREAS; row++ {
	    for col := 0; col < AREAS; col++ {
	        str += fmt.Sprintf( "%d,", me.antCountArea[row*AREAS+col] )
	    }
	    str += "\n"
	}
	log.Println(str)
*/
	// Track all the safe moves made
	movesMade := []*Ant{}
	safeMove := func(loc Location, dir Direction) bool {
		target := s.Map.Move(loc, dir)
		if s.Map.SafeDestination(target) {
			me.ants[loc].target = target
			movesMade = append(movesMade, me.ants[loc])

			s.IssueOrderLoc(loc, dir)
			return true
		}
		return false
	}

	rebuildPath := func(ant *Ant, target Location) bool {
		// Rebuild the path
		moves, valid := me.SearchMap(s, ant.loc, target)
		if valid {
			ant.moves = moves
			ant.moveTarget = target
		} else {
			return false
		}
		return true
	}

	nextBFSMove := func(ant *Ant, target Location, retargetNow bool) bool {
		if ant.moves == nil || (ant.moveTarget != target && retargetNow) {
			if (!rebuildPath(ant, target)) {
				return false
			}
		}

		// Move along the path, if we get stuck, re-path
		front := ant.moves.Front()
		dir := front.Value.(Direction)
		// Check explicitly for water - means we need to rebuild path
		if me.knownWater[s.Map.Move(ant.loc, dir)] {
			if (!rebuildPath(ant, target)) {
				return false
			}
			front = ant.moves.Front()
			dir = front.Value.(Direction)
		}

		success := safeMove(ant.loc, dir)
		if (success) {
			ant.moves.Remove(front)
			if (ant.moves.Len() == 0) {
				ant.moves = nil
			}
			return true
		}
		
		// Otherwise, if we fail to move, it's because another ant is in the way
		return false
	}

	for _, ant := range me.ants {
		// If we are hunting food, but it has disappeared, switch back to exploring
		if ant.state == STATE_HUNT_FOOD && s.Map.Item(ant.closestFood) != FOOD {
			ant.state = STATE_EXPLORE
			me.foodHunted[ant.loc] = nil, false
		}
		// Exploring ants will hunt for food if they find any
		if ant.state == STATE_EXPLORE {
			bestLen := 9999999
			s.Map.DoInRad(ant.loc, s.ViewRadius2, func(row, col int) {
				loc := s.Map.FromRowCol(row, col)
				if s.Map.Food[loc] {
					moves, valid := me.SearchMap(s, ant.loc, loc)
					if valid && moves.Len() < bestLen {
						bestLen = moves.Len()
						ant.moves = moves
						ant.moveTarget = loc
						ant.closestFood = loc
						ant.state = STATE_HUNT_FOOD
					}
				}
			})
		}
	}
	
	findNewTarget := func(ant *Ant) {
		wrap := AREAS*AREAS
		me.areaOffset = (me.areaOffset + 3) % 8
		
		// Go to the section with the fewest ants
/*		bestArea := 0
		bestCount := 99999
		areaDirs := [8]int{-1,AREAS,1,-AREAS,AREAS-1,AREAS+1,-AREAS-1,-AREAS+1}
		me.areaOffset = (me.areaOffset + 3) % 8
		for i := me.areaOffset; i < me.areaOffset + 8; i++ {
			actual := me.locToArea(ant.loc) + areaDirs[i % 8]
			if actual < 0 {
				actual += wrap
			} else if actual >= wrap {
				actual -= wrap
			}
			if me.antCountArea[actual] < bestCount {
				bestCount = me.antCountArea[actual]
				bestArea = actual
			}
		}*/
		
/*		me.areaOffset = (me.areaOffset + 47) % wrap
		for i := 0; i < wrap; i++ {
			actual := (i + me.areaOffset) % wrap
			if me.antCountArea[actual] < bestCount {
				bestCount = me.antCountArea[actual]
				bestArea = actual
			}
		}*/
		
		bestArea := rand.Intn(wrap)
		
		// Now, path there
		ant.exploreTarget = me.areaToLoc(bestArea)
		row, col := s.Map.FromLocation(ant.exploreTarget)
		w := s.Map.Cols / AREAS
		h := s.Map.Rows / AREAS
		row = row + rand.Intn(w) - (w / 2)
		col = col + rand.Intn(h) - (h / 2)
		ant.exploreTarget = s.Map.FromRowCol(row, col)
		for me.knownWater[ant.exploreTarget] {
			ant.exploreTarget = s.Map.Move(ant.exploreTarget, North)
			ant.exploreTarget = s.Map.Move(ant.exploreTarget, North)
			ant.exploreTarget = s.Map.Move(ant.exploreTarget, East)
		}

		ant.moveTarget = ant.exploreTarget
	}
	
	tryAnyMove := func (ant *Ant) {
		for dir := Direction(0); dir < 4; dir++ {
			if (safeMove(ant.loc, dir)) {
				return
			}
		}
	}
	
	for _, ant := range me.ants {
		if ant.state == STATE_EXPLORE {
			if ant.moves == nil {
				findNewTarget(ant)
			}
			if !nextBFSMove(ant, ant.moveTarget, false) {
				findNewTarget(ant)
				tryAnyMove(ant)
			}
		}
	}

	// Hunting for food now, as we may have switched other ants into this state
	for _, ant := range me.ants {
		if ant.state == STATE_HUNT_FOOD {
			// Move towards the food
			if !nextBFSMove(ant, ant.closestFood, true) {
				tryAnyMove(ant)
			}
		}
	}

	// Go through all the moves, and update the ant states
	for _, ant := range movesMade {
		me.antCountArea[me.locToArea(ant.loc)]--;
		me.antCountArea[me.locToArea(ant.target)]++;				
		
		me.ants[ant.target] = me.ants[ant.loc]
		me.ants[ant.loc] = nil, false
		ant.loc = ant.target
	}

	log.Println(fmt.Sprintf( "Finished turn in %d ms", (time.Nanoseconds() - startTime) / 1000000.0))
	//returning an error will halt the whole program!
	return nil
}
