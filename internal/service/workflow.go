package service

import "git.neolidy.top/neo/storybook/internal/model"

// WorkflowService 统一管理状态流转规则。
type WorkflowService struct{}

func NewWorkflowService() *WorkflowService {
	return &WorkflowService{}
}

func (s *WorkflowService) CanStoryTransit(from, to string) bool {
	return canTransit(storyTransitions, from, to)
}

func (s *WorkflowService) CanTaskTransit(from, to string) bool {
	return canTransit(taskTransitions, from, to)
}

func (s *WorkflowService) CanSprintTransit(from, to string) bool {
	return canTransit(sprintTransitions, from, to)
}

func (s *WorkflowService) CanBugTransit(from, to string) bool {
	return canTransit(bugTransitions, from, to)
}

var storyTransitions = map[string]map[string]struct{}{
	model.StoryStatusPending: {
		model.StoryStatusBacklog: {}, // 审批通过 -> 待办
		model.StoryStatusReady:   {}, // 审批通过并准备就绪
	},
	model.StoryStatusBacklog: {
		model.StoryStatusReady:      {},
		model.StoryStatusInProgress: {},
	},
	model.StoryStatusReady: {
		model.StoryStatusBacklog:    {},
		model.StoryStatusInProgress: {},
	},
	model.StoryStatusInProgress: {
		model.StoryStatusReady: {},
		model.StoryStatusTest:  {},
		model.StoryStatusDone:  {},
	},
	model.StoryStatusTest: {
		model.StoryStatusInProgress: {},
		model.StoryStatusDone:       {},
	},
	model.StoryStatusDone: {
		model.StoryStatusTest: {},
	},
}

var taskTransitions = map[string]map[string]struct{}{
	model.TaskStatusTodo: {
		model.TaskStatusInProgress: {},
		model.TaskStatusBlocked:    {},
		model.TaskStatusDone:       {},
	},
	model.TaskStatusInProgress: {
		model.TaskStatusBlocked: {},
		model.TaskStatusDone:    {},
		model.TaskStatusTodo:    {},
	},
	model.TaskStatusBlocked: {
		model.TaskStatusInProgress: {},
		model.TaskStatusDone:       {},
	},
	model.TaskStatusDone: {
		model.TaskStatusInProgress: {},
	},
}

var sprintTransitions = map[string]map[string]struct{}{
	model.SprintStatusPlanned: {
		model.SprintStatusActive:    {},
		model.SprintStatusCompleted: {},
		model.SprintStatusCancelled: {},
	},
	model.SprintStatusActive: {
		model.SprintStatusCompleted: {},
		model.SprintStatusPlanned:   {},
		model.SprintStatusCancelled: {},
	},
	model.SprintStatusCompleted: {
		model.SprintStatusActive: {},
	},
	model.SprintStatusCancelled: {},
}

var bugTransitions = map[string]map[string]struct{}{
	model.BugStatusOpen: {
		model.BugStatusInProgress: {},
		model.BugStatusResolved:   {},
		model.BugStatusClosed:     {},
	},
	model.BugStatusInProgress: {
		model.BugStatusResolved: {},
		model.BugStatusOpen:     {},
	},
	model.BugStatusResolved: {
		model.BugStatusClosed:     {},
		model.BugStatusInProgress: {},
	},
	model.BugStatusClosed: {
		model.BugStatusInProgress: {},
	},
}

func canTransit(allowed map[string]map[string]struct{}, from, to string) bool {
	if from == to {
		return true
	}
	next, ok := allowed[from]
	if !ok {
		return false
	}
	_, ok = next[to]
	return ok
}

var Workflow = NewWorkflowService()
