package services

import (
	"context"
	"fmt"
	"log"
)

// AgentState represents the state passed between agents
type AgentState struct {
	SessionID    string                 `json:"session_id"`
	CandidateID  string                 `json:"candidate_id"`
	Messages     []map[string]interface{} `json:"messages"`
	Context      map[string]interface{} `json:"context"`
	CurrentAgent string                 `json:"current_agent"`
	IsComplete   bool                   `json:"is_complete"`
}

// Agent defines the interface for all agents
type Agent interface {
	Name() string
	Execute(ctx context.Context, state *AgentState) (*AgentState, error)
}

// AgentGraph orchestrates multi-agent workflow
type AgentGraph struct {
	agents map[string]Agent
	edges  map[string][]string
}

// NewAgentGraph creates a new agent orchestration graph
func NewAgentGraph() *AgentGraph {
	return &AgentGraph{
		agents: make(map[string]Agent),
		edges:  make(map[string][]string),
	}
}

// AddAgent registers an agent
func (g *AgentGraph) AddAgent(agent Agent) {
	g.agents[agent.Name()] = agent
	if g.edges[agent.Name()] == nil {
		g.edges[agent.Name()] = []string{}
	}
}

// AddEdge creates a transition from one agent to another
func (g *AgentGraph) AddEdge(from, to string) {
	if g.edges[from] == nil {
		g.edges[from] = []string{}
	}
	g.edges[from] = append(g.edges[from], to)
}

// Execute runs the agent graph starting from the given agent
func (g *AgentGraph) Execute(ctx context.Context, startAgent string, state *AgentState) (*AgentState, error) {
	currentAgent := startAgent
	visited := make(map[string]bool)
	maxIterations := 20

	for i := 0; i < maxIterations; i++ {
		if currentAgent == "" || state.IsComplete {
			break
		}

		if visited[currentAgent] && i > 0 {
			log.Printf("Warning: possible infinite loop at agent %s", currentAgent)
			break
		}
		visited[currentAgent] = true

		agent, exists := g.agents[currentAgent]
		if !exists {
			return nil, fmt.Errorf("agent %s not found", currentAgent)
		}

		log.Printf("Executing agent: %s", currentAgent)
		state.CurrentAgent = currentAgent

		newState, err := agent.Execute(ctx, state)
		if err != nil {
			return nil, fmt.Errorf("agent %s failed: %w", currentAgent, err)
		}

		state = newState

		// Determine next agent
		nextAgents := g.edges[currentAgent]
		if len(nextAgents) == 0 {
			break
		}
		currentAgent = nextAgents[0]
	}

	return state, nil
}
