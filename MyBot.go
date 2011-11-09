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
	rand					rand.Rand
	exploreDir		Direction
}

func NewBot(s *State) Bot {
	me := &GarboAnt{
		ants: make(map[Location]*Ant),
	}
	me.exploreHeat = &me.exploreHeat1;
	me.exploreNext = &me.exploreHeat2;
	return me
}

func (me *GarboAnt) SearchMap(s *State, source Location, isFinalState func(current Location) bool) (Direction, bool) {
	type Node struct {
		current Location
		startDir Direction
	}

	// BFS
	visited := make(map[Location]bool)
	queue := new(list.List)

	nextStep := func(current Location, oldNode *Node) {
		dirs := []Direction{West, South, North, East}
		for _, i := range dirs {
			loc := s.Map.Move(current, dirs[i])
			if s.Map.SafeDestination(loc) && !visited[loc] {
				startDir := dirs[i];
				if oldNode != nil {
					startDir = oldNode.startDir
				}
				queue.PushBack(&Node{current: loc, startDir: startDir})
			}
		}		
	}

	visited[source] = true
	nextStep(source, nil)

	for queue.Len() != 0 {
		curEl := queue.Front()
		queue.Remove(curEl)
		current := curEl.Value.(*Node)
		if isFinalState(current.current) {
			return current.startDir, true
		}
		if visited[current.current] {
			continue;
		}
		visited[current.current] = true
		nextStep(current.current, current)
	}

	return North, false
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
			switch s.Map.Item(loc) {
			case UNKNOWN: me.exploreHeat[loc] = 9999999
			case WATER: me.exploreHeat[loc] = 0
			}
		}
	}
	
	// Run the diffusion
	for steps := 0; steps < 10; steps++ {
		for row := 0; row < s.Map.Rows; row++ {
			for col := 0; col < s.Map.Cols; col++ {
				loc := s.Map.FromRowCol(row, col)
				next := float32(0.0)
				if (s.Map.Item(loc) != WATER) {
					for dir := Direction(0); dir < 4; dir++ {
						loc2 := s.Map.Move(loc, dir)
						next += me.exploreHeat[loc2]
					}
				}
				me.exploreNext[loc] = next * 0.18
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
	
	for _, ant := range me.ants {
		// If we are hunting food, but it has disappeared, switch back to exploring
		if ant.state == STATE_HUNT_FOOD && s.Map.Item(ant.closestFood) != FOOD {
			ant.state = STATE_EXPLORE;
		}
		// Idle or exploring ants will hunt for food if they find any
		if ant.state == STATE_IDLE || ant.state == STATE_EXPLORE {
			closest := 999999999
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
			})
		}
	}

	// Hunting for food now, as we may have switched other ants into this state
	for _, ant := range me.ants {
		if ant.state == STATE_HUNT_FOOD {
			// Move towards the food
			finalState := func(current Location) bool {
				return current == ant.closestFood
			}
			targetDir, valid := me.SearchMap(s, ant.loc, finalState)
			if valid {
				safeMove(ant.loc, targetDir)
			} else {
				ant.state = STATE_EXPLORE
			}			
		}
	}

	for _, ant := range me.ants {
		if ant.state == STATE_EXPLORE {
			// Find the best exploreHeat
			bestDir := me.exploreDir
			bestHeat := float32(0.0)
			for dir := Direction(0); dir < 4; dir++ {
				target := s.Map.Move(ant.loc, dir)
				if (me.exploreHeat[target] > bestHeat) {
					bestHeat = me.exploreHeat[target]
					bestDir = dir
				}
			}
			safeMove(ant.loc, bestDir)
		}
	}

	me.exploreDir = (me.exploreDir + 1) % 4

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
