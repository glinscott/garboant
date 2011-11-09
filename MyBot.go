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
	STATE_IDLE = iota
	STATE_HUNT_FOOD
	STATE_EXPLORE
)

type Ant struct {
	loc						Location
	target				Location
	closestFood 	Location
	state 				AntState
	seenThisTurn 	bool
	
	// BFS move state
	moves					*list.List
	moveTarget		Location
}

const (
	MAX_SIZE = 200
)

type GarboAnt struct {
	exploreHeat1	[MAX_SIZE*MAX_SIZE]float32
	exploreHeat2	[MAX_SIZE*MAX_SIZE]float32
	exploreHeat   *[MAX_SIZE*MAX_SIZE]float32
	exploreNext		*[MAX_SIZE*MAX_SIZE]float32
	ants					map[Location]*Ant
	knownHills		map[Location]bool
	rand					rand.Rand
}

func NewBot(s *State) Bot {
	me := &GarboAnt{
		ants: make(map[Location]*Ant),
		knownHills: make(map[Location]bool),
	}
	me.exploreHeat = &me.exploreHeat1;
	me.exploreNext = &me.exploreHeat2;
	return me
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
		for _, i := range dirs {
			loc := s.Map.Move(current, dirs[i])
			_, exists := visited[loc]
			if s.Map.SafeDestination(loc) && !exists {
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
			me.ants[loc] = &Ant{ state: STATE_EXPLORE, }
			me.ants[loc].loc = loc
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

			switch item {
			case UNKNOWN: me.exploreHeat[loc] = 9999
			case WATER: me.exploreHeat[loc] = 0
			case MY_HILL: me.exploreHeat[loc] = 0
			case MY_OCCUPIED_HILL: me.exploreHeat[loc] = 0
			}
		}
	}
	
	// Run the diffusion
	for steps := 0; steps < 20; steps++ {
		for row := 0; row < s.Map.Rows; row++ {
			for col := 0; col < s.Map.Cols; col++ {
				loc := s.Map.FromRowCol(row, col)
				next := float32(0.0)
				if s.Map.Item(loc) != WATER {
					for dir := Direction(0); dir < 4; dir++ {
						loc2 := s.Map.Move(loc, dir)
						if s.Map.Item(loc2) != WATER {
							next += me.exploreHeat[loc2] * 0.22
						}
					}
				}
				me.exploreNext[loc] = next
			}
		}
		me.exploreNext, me.exploreHeat = me.exploreHeat, me.exploreNext
	}
/*
	str := ""
	for row := 0; row < s.Map.Rows; row++ {
	    for col := 0; col < s.Map.Cols; col++ {
	        str += fmt.Sprintf( "%.1f,", me.exploreHeat[s.Map.FromRowCol(row, col)] / 1000 )
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

	nextBFSMove := func(ant *Ant, target Location, retargetNow bool) bool {
		if ant.moves == nil || (ant.moveTarget != target && retargetNow) {
			// Rebuild the path
			moves, valid := me.SearchMap(s, ant.loc, target)
			if valid {
				ant.moves = moves
				ant.moveTarget = target
			} else {
				log.Println("Unable to find path!")
				return false
			}			
		}

		// Move along the move path
		front := ant.moves.Front()
		ant.moves.Remove(front)
		if (ant.moves.Len() == 0) {
			ant.moves = nil
		}
		return safeMove(ant.loc, front.Value.(Direction))
	}

	for _, ant := range me.ants {
		// If we are hunting food, but it has disappeared, switch back to exploring
		if ant.state == STATE_HUNT_FOOD && s.Map.Item(ant.closestFood) != FOOD {
			ant.state = STATE_EXPLORE;
		}
		// Idle or exploring ants will hunt for food if they find any
		if ant.state == STATE_IDLE || ant.state == STATE_EXPLORE {
			closest := 999999999
			bestHeat := float32(0.0)
			heatLoc := ant.loc
			
			fRow, fCol := s.Map.FromLocation(ant.loc)
			s.Map.DoInRad(ant.loc, s.ViewRadius2, func(row, col int) {
				loc := s.Map.FromRowCol(row, col)
				if s.Map.Food[loc] {
					distance := (fRow-row)*(fRow-row)+(fCol-col)*(fCol-col);
					if distance < closest {
						closest = distance
						ant.closestFood = loc
						ant.state = STATE_HUNT_FOOD
					}
				}
				if me.exploreHeat[loc] > bestHeat {
					bestHeat = me.exploreHeat[loc]
					heatLoc = loc
				}
			})
			
			if (ant.state == STATE_EXPLORE) {
				nextBFSMove(ant, heatLoc, false)
			}
		}
	}

	// Hunting for food now, as we may have switched other ants into this state
	for _, ant := range me.ants {
		if ant.state == STATE_HUNT_FOOD {
			// Move towards the food
			nextBFSMove(ant, ant.closestFood, true);
		}
	}

	// Go through all the moves, and update the ant states
	for _, ant := range movesMade {
		me.ants[ant.target] = me.ants[ant.loc]
		me.ants[ant.loc] = nil, false
		ant.loc = ant.target
	}
	
	log.Println(fmt.Sprintf( "Finished turn in %d ms", (time.Nanoseconds() - startTime) / 1000000.0))
	//returning an error will halt the whole program!
	return nil
}
