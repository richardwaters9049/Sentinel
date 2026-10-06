package readiness

import (
	"context"
	"fmt"
	"sort"
	"time"
)

type CheckFunc func(context.Context) error

type Result struct {
	Ready        bool              `json:"ready"`
	Dependencies map[string]string `json:"dependencies"`
}

type Checker struct {
	timeout time.Duration
	checks  map[string]CheckFunc
}

func New(timeout time.Duration, checks map[string]CheckFunc) *Checker {
	copied := make(map[string]CheckFunc, len(checks))
	for name, check := range checks {
		copied[name] = check
	}

	return &Checker{
		timeout: timeout,
		checks:  copied,
	}
}

func (c *Checker) Check(ctx context.Context) Result {
	result := Result{
		Ready:        true,
		Dependencies: make(map[string]string, len(c.checks)),
	}

	names := make([]string, 0, len(c.checks))
	for name := range c.checks {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		check := c.checks[name]
		if check == nil {
			result.Ready = false
			result.Dependencies[name] = "unavailable"
			continue
		}

		checkCtx, cancel := context.WithTimeout(ctx, c.timeout)
		err := check(checkCtx)
		cancel()

		if err != nil {
			result.Ready = false
			result.Dependencies[name] = "unavailable"
			continue
		}

		result.Dependencies[name] = "ready"
	}

	return result
}

func RequirePositiveTimeout(timeout time.Duration) error {
	if timeout <= 0 {
		return fmt.Errorf("readiness timeout must be greater than zero")
	}
	return nil
}
