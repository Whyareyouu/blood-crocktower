package contracts

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// Копия генерируется и проверяется scripts/check-contracts.mjs из единого источника.
//
//go:embed protocol.schema.json
var protocolSchema []byte

var schemas = sync.OnceValues(func() (map[string]*jsonschema.Schema, error) {
	var document map[string]any
	if err := json.Unmarshal(protocolSchema, &document); err != nil {
		return nil, err
	}
	compiler := jsonschema.NewCompiler()
	compiler.AssertFormat()
	const location = "https://blood.local/protocol.schema.json"
	if err := compiler.AddResource(location, document); err != nil {
		return nil, err
	}
	result := make(map[string]*jsonschema.Schema)
	for name := range document["$defs"].(map[string]any) {
		schema, err := compiler.Compile(location + "#/$defs/" + name)
		if err != nil {
			return nil, err
		}
		result[name] = schema
	}
	return result, nil
})

func Validate(name string, data []byte) error {
	compiled, err := schemas()
	if err != nil {
		return err
	}
	schema, ok := compiled[name]
	if !ok {
		return fmt.Errorf("неизвестная схема %s", name)
	}
	value, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
	if err != nil {
		return err
	}
	return schema.Validate(value)
}
