package model

import "fmt"

type Match struct {
	Meta     Meta     `json:"meta"`
	Sticky   Sticky   `json:"sticky"`
	Prematch Prematch `json:"prematch"`
	Auto     Auto     `json:"auto"`
	Teleop   Teleop   `json:"teleop"`
	Endgame  Endgame  `json:"endgame"`
}

type Fouls struct {
	Minor      uint64 `json:"minor"`
	Major      uint64 `json:"major"`
	YellowCard uint64 `json:"yellowCard"`
	RedCard    uint64 `json:"redCard"`
}

type Sticky struct {
	Fouls        Fouls  `json:"fouls"`
	Coopertition bool   `json:"coopertition"`
	Comment      string `json:"comment"`
}

type Position uint64

const (
	One   Position = 1
	Two   Position = 2
	Three Position = 3
)

type Prematch struct {
	StartingPosition Position `json:"startingPosition"`
	DriverStation    Position `json:"driverStation"`
}

func (match Match) StartingPositionRadio(position Position) Radio[Position] {
	return Radio[Position]{
		Checked: match.Prematch.StartingPosition == position,
		Name:    "startingPosition",
		Value:   position,
		Post:    match.Hash(),
	}
}

func (match Match) DriverStationRadio(position Position) Radio[Position] {
	return Radio[Position]{
		Checked: match.Prematch.DriverStation == position,
		Name:    "driverStation",
		Value:   position,
		Post:    match.Hash(),
	}
}

type Auto struct {
	CrossedLine bool   `json:"crossedLine"`
	L4          uint64 `json:"l4"`
	L3          uint64 `json:"l3"`
	L2          uint64 `json:"l2"`
	L1          uint64 `json:"l1"`
	Processor   uint64 `json:"processor"`
	Removed     uint64 `json:"removed"`
	RobotNet    uint64 `json:"robotNet"`
}

type Teleop struct {
	L4        uint64 `json:"l4"`
	L3        uint64 `json:"l3"`
	L2        uint64 `json:"l2"`
	L1        uint64 `json:"l1"`
	Processor uint64 `json:"processor"`
	Removed   uint64 `json:"removed"`
	RobotNet  uint64 `json:"robotNet"`
	HumanNet  uint64 `json:"humanNet"`
}

type Barge string

const (
	None    Barge = "None"
	Park    Barge = "Park"
	Shallow Barge = "Shallow"
	Deep    Barge = "Deep"
)

var BargeMap = map[string]Barge{
	"None":    None,
	"Park":    Park,
	"Shallow": Shallow,
	"Deep":    Deep,
}

type Endgame struct {
	Barge Barge `json:"barge"`
}

func (match Match) BargeRadio(barge Barge) Radio[Barge] {
	return Radio[Barge]{
		Checked: match.Endgame.Barge == barge,
		Name:    "barge",
		Value:   barge,
		Post:    match.Hash(),
	}
}

func (match Match) Hash() string {
	return fmt.Sprintf("%x", match.Meta.Hash())
}
