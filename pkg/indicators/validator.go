package indicators

import (
	"fmt"

	"github.com/CCAtAlvis/backgommon/pkg/interfaces"
)

// ValidateNoCycles checks for circular dependencies in an indicator
func ValidateNoCycles(indicator interfaces.Indicator) error {
	visited := make(map[string]bool)
	stack := make(map[string]bool)
	return validateNoCyclesRecursive(indicator, visited, stack)
}

func validateNoCyclesRecursive(indicator interfaces.Indicator, visited, stack map[string]bool) error {
	name := indicator.Name()

	// If this node is already in our DFS stack, we have a cycle
	if stack[name] {
		return fmt.Errorf("circular dependency detected involving %s", name)
	}

	// If we've already validated this node, skip it
	if visited[name] {
		return nil
	}

	// Add to DFS stack
	stack[name] = true
	visited[name] = true

	// Check all dependencies
	for _, dep := range indicator.Dependencies() {
		if err := validateNoCyclesRecursive(dep, visited, stack); err != nil {
			return fmt.Errorf("dependency chain: %s -> %v", name, err)
		}
	}

	// Remove from DFS stack (backtrack)
	stack[name] = false

	return nil
}

// ValidateIndicators checks for circular dependencies in multiple indicators
func ValidateIndicators(indicators []interfaces.Indicator) error {
	visited := make(map[string]bool)
	stack := make(map[string]bool)

	for _, ind := range indicators {
		if err := validateNoCyclesRecursive(ind, visited, stack); err != nil {
			return err
		}
	}

	return nil
}
