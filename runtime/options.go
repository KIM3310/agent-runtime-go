package runtime

import "time"

// Option configures a Runner.
type Option func(*Runner)

// WithTools registers a list of tools.
func WithTools(tools []Tool) Option {
	return func(r *Runner) {
		for _, t := range tools {
			r.tools[t.Name] = t
		}
	}
}

// WithTool registers a single tool.
func WithTool(tool Tool) Option {
	return func(r *Runner) {
		r.tools[tool.Name] = tool
	}
}

// WithMaxSteps sets the max agent iteration limit.
func WithMaxSteps(n int) Option {
	return func(r *Runner) {
		r.maxSteps = n
	}
}

// WithTimeout sets the overall run timeout.
func WithTimeout(d time.Duration) Option {
	return func(r *Runner) {
		r.timeout = d
	}
}

// WithRetryPolicy overrides the default retry policy.
func WithRetryPolicy(p RetryPolicy) Option {
	return func(r *Runner) {
		r.retryPolicy = p
	}
}

// WithLogger overrides the default no-op logger.
func WithLogger(logger Logger) Option {
	return func(r *Runner) {
		r.logger = logger
	}
}
