package cobra

import (
	"fmt"
	"time"

	"github.com/spf13/pflag"

	"github.com/guilhermelinosp/hellnet-cli-template/internal/cli"
)

// flagBridge adapts an abstraction Flag into pflag's Value contract. Cobra
// performs lexical parsing; this bridge channels every write straight into
// the template-owned storage so runtime values never fork between layers.
type flagBridge struct{ f *cli.Flag }

func newFlagBridge(f *cli.Flag) pflag.Value { return flagBridge{f} }

// Set converts textual input; errors surface through normal usage handling.
func (b flagBridge) Set(s string) error { return b.f.Apply(s) }

// String renders the effective value for help/default annotations.
func (b flagBridge) String() string {
	switch v := b.f.Effective().(type) {
	case string:
		return v
	case bool:
		return fmt.Sprintf("%t", v)
	case int:
		return fmt.Sprintf("%d", v)
	case float64:
		return fmt.Sprintf("%g", v)
	case time.Duration:
		return v.String()
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", v)
	}
}

// Type maps abstraction kinds to the four pflag lexer behaviors:
// "string"/"int"/"float64"/"duration" require a following value, while
// "bool" is satisfied by presence alone (--json works without =true).
func (b flagBridge) Type() string {
	switch b.f.Type {
	case cli.FlagBool:
		return "bool"
	case cli.FlagInt:
		return "int"
	case cli.FlagFloat64:
		return "float64"
	case cli.FlagDuration:
		return "duration"
	default:
		return "string"
	}
}

// boolNoOptDefault lets boolean flags activate without an explicit value.
func boolNoOptDefault(t cli.FlagType) string {
	if t == cli.FlagBool {
		return "true"
	}
	return ""
}
