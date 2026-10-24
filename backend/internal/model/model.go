package model

import (
	"time"
)

type Accounts struct {
	Id          int64
	Email       string
	Password    string
	displayName string
	CreatedAt   time.Time
}

type Membership struct {
	Id        int64
	TeamId    int64
	AccountId int64
	role      string
	joinedAt  time.Time
	leftAt    time.Time
}

type Teams struct {
	Id         int64
	ParentTeam int64
	name       string
	CreatedAt  time.Time
}

type Projects struct {
	Id           int64
	MembershipId int64
	TaskId       int64
	TeamId       int64
}

type Tasks struct {
	Id          int64
	CreatedBy   int64
	TeamId      int64
	ProjectId   int64
	ParentId    int64
	Status      int64
	Title       string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Status struct {
	Id          int64
	ProjectId   int64
	title       string
	description string
	color       string
}

type TaskAssignees struct {
	Id           int64
	MembershipId int64
	TaskId       int64
	TeamId       int64
}
