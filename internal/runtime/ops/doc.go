//go:generate go run github.com/webitel/flow_manager/cmd/docgen

package ops

// Documenter is an optional interface ops may implement to expose their
// documentation. The generator (cmd/docgen) collects these and writes doc.yaml.
type Documenter interface {
	Doc() OpDoc
}

// OpDoc is the schema-level doc for one op. Maps 1-to-1 to a top-level entry
// in bin/doc.yaml.
type OpDoc struct {
	Description string             `yaml:"description"`
	AvailableIn []string           `yaml:"available_in"`
	Visual      bool               `yaml:"visual,omitempty"`
	Args        map[string]ArgDoc  `yaml:"args,omitempty"`
	Notes       []string           `yaml:"notes,omitempty"`
	Examples    map[string]Example `yaml:"examples,omitempty"`
}

// ArgDoc documents one argument of an op.
type ArgDoc struct {
	Type        string `yaml:"type"`
	Required    bool   `yaml:"required,omitempty"`
	Description string `yaml:"description"`
	Default     any    `yaml:"default,omitempty"`
}

// Example is one named usage example.
type Example struct {
	Description string `yaml:"description"`
	Schema      string `yaml:"schema"`
}
