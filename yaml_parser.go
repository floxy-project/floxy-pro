package floxy

import (
	"errors"
	"fmt"
	"time"

	"gopkg.in/yaml.v3"
)

// ParseWorkflowYAML parses a YAML document describing handlers and flows
// and builds WorkflowDefinition(s) using the existing Builder DSL.
//
// Notes/assumptions:
// - The YAML may contain multiple flows; we return a map keyed by flow name.
// - Handlers are defined globally and referenced by steps via the `handler` field.
// - We keep handler -> exec mapping for floxyctl to execute external commands.
// - Supported step kinds: task (default), parallel (with auto-join), condition (with else branch).
// - No nested flows (fork/join) beyond `parallel` and `condition` are required at this time.
// - DQL is not supported here.
//
// version: workflow version to assign to created definitions (default recommended: 1).
func ParseWorkflowYAML(data []byte, version int) (defs map[string]*WorkflowDefinition, handlersExec map[string]string, err error) {
	var doc YamlRoot
	if err = yaml.Unmarshal(data, &doc); err != nil {
		return nil, nil, fmt.Errorf("yaml parse error: %w", err)
	}

	// Build handlers exec map
	handlersExec = make(map[string]string, len(doc.Handlers))
	for i, h := range doc.Handlers {
		if h.Name == "" {
			return nil, nil, fmt.Errorf("handlers[%d]: missing name", i)
		}
		if h.Exec == "" {
			return nil, nil, fmt.Errorf("handlers[%d]: missing exec for handler %q", i, h.Name)
		}
		handlersExec[h.Name] = h.Exec
	}

	defs = make(map[string]*WorkflowDefinition)
	for i, f := range doc.Flows {
		if f.Name == "" {
			return nil, nil, fmt.Errorf("flows[%d]: missing name", i)
		}

		b := NewBuilder(f.Name, version)

		if len(f.Steps) == 0 {
			return nil, nil, fmt.Errorf("flow %q: steps are required", f.Name)
		}

		// build steps in sequence order
		if err := buildStepsIntoBuilder(b, f.Steps, handlersExec); err != nil {
			return nil, nil, fmt.Errorf("flow %q: %w", f.Name, err)
		}

		def, buildErr := b.Build()
		if buildErr != nil {
			return nil, nil, buildErr
		}
		defs[f.Name] = def
	}

	return defs, handlersExec, nil
}

// --- YAML model ---

type YamlRoot struct {
	TLS      *TLSConfig    `yaml:"tls"`
	Handlers []YamlHandler `yaml:"handlers"`
	Flows    []yamlFlow    `yaml:"flows"`
}

type TLSConfig struct {
	SkipVerify bool   `yaml:"skip_verify"`
	CertFile   string `yaml:"cert_file"`
	KeyFile    string `yaml:"key_file"`
	CAFile     string `yaml:"ca_file"`
}

type YamlHandler struct {
	Name string     `yaml:"name"`
	Exec string     `yaml:"exec"`
	TLS  *TLSConfig `yaml:"tls"`
}

type yamlFlow struct {
	Name  string     `yaml:"name"`
	Steps []YamlStep `yaml:"steps"`
}

// YamlStep supports 3 shapes:
// 1) task (default):
//    - name: step_name
//      handler: handler_name
//      on_failure: refund_handler   # optional
//      max_retries: 3               # optional
//
// 2) parallel block:
//    - type: parallel
//      name: parallel_name
//      tasks:
//        - name: task1
//          handler: handler1
//        - name: task2
//          handler: handler2
//      # join step will be auto-created as "parallel_name_join"
//
// 3) condition block:
//    - type: condition
//      name: check_something
//      expr: "input.value > 10"    # or `condition:`
//      else:
//        - name: fallback_task
//          handler: handler_fallback
//
// Shorthand form is also supported for a task step: a plain string equals both name and handler.
// Example:
//   - reserve_stock  # becomes name=reserve_stock, handler=reserve_stock
//
// Additionally, to match the user-provided sample (which mixes scalar and mapping),
// YamlStep.UnmarshalYAML tries to interpret a scalar as task name/handler.

type YamlStep struct {
	// common
	Type string `yaml:"type"`
	Name string `yaml:"name"`

	// task
	Handler    string         `yaml:"handler"`
	OnFailure  string         `yaml:"on_failure"`
	MaxRetries *int           `yaml:"max_retries"`
	NoIdem     *bool          `yaml:"no_idempotent"`
	Delay      *int64         `yaml:"delay"`       // milliseconds
	RetryDelay *int64         `yaml:"retry_delay"` // milliseconds
	RetryStr   string         `yaml:"retry_strategy"`
	Timeout    *int64         `yaml:"timeout"` // milliseconds
	Metadata   map[string]any `yaml:"metadata"`

	// parallel
	Tasks []YamlTask `yaml:"tasks"`

	// condition
	Expr string     `yaml:"expr"`
	Cond string     `yaml:"condition"` // alias for expr
	Else []YamlStep `yaml:"else"`
}

type YamlTask struct {
	Name       string         `yaml:"name"`
	Handler    string         `yaml:"handler"`
	MaxRetries *int           `yaml:"max_retries"`
	NoIdem     *bool          `yaml:"no_idempotent"`
	Delay      *int64         `yaml:"delay"`       // ms
	RetryDelay *int64         `yaml:"retry_delay"` // ms
	RetryStr   string         `yaml:"retry_strategy"`
	Timeout    *int64         `yaml:"timeout"` // ms
	Metadata   map[string]any `yaml:"metadata"`
}

func (s *YamlStep) UnmarshalYAML(value *yaml.Node) error {
	// Support scalar shorthand: "- step_name"
	if value.Kind == yaml.ScalarNode {
		var name string
		if err := value.Decode(&name); err != nil {
			return err
		}
		s.Type = "task"
		s.Name = name
		s.Handler = name
		return nil
	}
	// Otherwise decode as mapping
	type alias YamlStep
	var a alias
	if err := value.Decode(&a); err != nil {
		return err
	}
	*s = YamlStep(a)
	if s.Type == "" {
		// default kind
		s.Type = "task"
	}
	// normalize expr/condition synonyms
	if s.Expr == "" && s.Cond != "" {
		s.Expr = s.Cond
	}
	return nil
}

// --- build helpers ---

func buildStepsIntoBuilder(b *Builder, steps []YamlStep, handlersExec map[string]string) error {
	for idx := range steps {
		st := steps[idx]
		switch st.Type {
		case "task":
			if st.Name == "" {
				return fmt.Errorf("steps[%d]: task requires name", idx)
			}
			if st.Handler == "" {
				st.Handler = st.Name
			}
			b.Step(st.Name, st.Handler)
			// fill options directly on the step in builder
			step := b.steps[st.Name]
			applyTaskOptions(step, st, handlersExec)
			if st.OnFailure != "" {
				// Use provided string as both step name and handler for compensation
				b.OnFailure(st.OnFailure, st.OnFailure)
				// attach exec metadata for compensation handler if known
				if comp, ok := b.steps[st.OnFailure]; ok {
					if exec := handlersExec[st.OnFailure]; exec != "" {
						if comp.Metadata == nil {
							comp.Metadata = make(map[string]any)
						}
						comp.Metadata["exec"] = exec
					}
				}
			}

		case "parallel":
			if st.Name == "" {
				return fmt.Errorf("steps[%d]: parallel requires name", idx)
			}
			if len(st.Tasks) < 2 {
				return fmt.Errorf("parallel %q must contain at least 2 tasks", st.Name)
			}
			var defs []*StepDefinition
			for i := range st.Tasks {
				t := st.Tasks[i]
				if t.Name == "" {
					return fmt.Errorf("parallel %q: tasks[%d] missing name", st.Name, i)
				}
				if t.Handler == "" {
					t.Handler = t.Name
				}
				step := NewTask(t.Name, t.Handler)
				applyYamlTaskOptions(step, t, handlersExec)
				defs = append(defs, step)
			}
			b.Parallel(st.Name, defs...)

		case "condition":
			if st.Name == "" {
				return fmt.Errorf("steps[%d]: condition requires name", idx)
			}
			if st.Expr == "" {
				return fmt.Errorf("condition %q: expr/condition is required", st.Name)
			}
			// Build else branch, if any
			var elseFn func(*Builder)
			if len(st.Else) > 0 {
				elseSteps := st.Else // capture
				elseFn = func(eb *Builder) {
					_ = buildStepsIntoBuilder(eb, elseSteps, handlersExec)
				}
			}
			b.Condition(st.Name, st.Expr, elseFn)

		default:
			return fmt.Errorf("steps[%d]: unsupported type %q", idx, st.Type)
		}
	}
	return nil
}

func applyTaskOptions(step *StepDefinition, st YamlStep, handlersExec map[string]string) {
	if step.Metadata == nil {
		step.Metadata = make(map[string]any)
	}
	// Attach exec command if known for the handler
	if st.Handler != "" {
		if exec := handlersExec[st.Handler]; exec != "" {
			step.Metadata["exec"] = exec
		}
	}
	// Optional flags
	if st.MaxRetries != nil {
		step.MaxRetries = *st.MaxRetries
	}
	if st.NoIdem != nil {
		step.NoIdempotent = *st.NoIdem
	}
	if st.Delay != nil {
		step.Delay = millisecondsToDuration(*st.Delay)
	}
	if st.RetryDelay != nil {
		step.RetryDelay = millisecondsToDuration(*st.RetryDelay)
	}
	switch st.RetryStr {
	case "exponential":
		step.RetryStrategy = RetryStrategyExponential
	case "linear":
		step.RetryStrategy = RetryStrategyLinear
	case "", "fixed":
		step.RetryStrategy = RetryStrategyFixed
	default:
		// ignore unknown, keep default
	}
	if st.Timeout != nil {
		step.Timeout = millisecondsToDuration(*st.Timeout)
	}
	// Merge metadata
	for k, v := range st.Metadata {
		step.Metadata[k] = v
	}
}

func applyYamlTaskOptions(step *StepDefinition, t YamlTask, handlersExec map[string]string) {
	if step.Metadata == nil {
		step.Metadata = make(map[string]any)
	}
	if t.Handler != "" {
		if exec := handlersExec[t.Handler]; exec != "" {
			step.Metadata["exec"] = exec
		}
	}
	if t.MaxRetries != nil {
		step.MaxRetries = *t.MaxRetries
	}
	if t.NoIdem != nil {
		step.NoIdempotent = *t.NoIdem
	}
	if t.Delay != nil {
		step.Delay = millisecondsToDuration(*t.Delay)
	}
	if t.RetryDelay != nil {
		step.RetryDelay = millisecondsToDuration(*t.RetryDelay)
	}
	switch t.RetryStr {
	case "exponential":
		step.RetryStrategy = RetryStrategyExponential
	case "linear":
		step.RetryStrategy = RetryStrategyLinear
	case "", "fixed":
		step.RetryStrategy = RetryStrategyFixed
	default:
		// ignore
	}
	if t.Timeout != nil {
		step.Timeout = millisecondsToDuration(*t.Timeout)
	}
	for k, v := range t.Metadata {
		step.Metadata[k] = v
	}
}

func millisecondsToDuration(ms int64) time.Duration {
	// Convert milliseconds to time.Duration
	return time.Duration(ms) * time.Millisecond
}

// Validate a minimal doc without flows
func ValidateYAMLDocument(data []byte) error {
	var r YamlRoot
	if err := yaml.Unmarshal(data, &r); err != nil {
		return err
	}
	if len(r.Flows) == 0 {
		return errors.New("no flows defined")
	}
	return nil
}
