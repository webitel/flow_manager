package ops_test

import (
	"testing"

	"github.com/webitel/flow_manager/internal/runtime/ops"
	"github.com/webitel/flow_manager/internal/runtime/tree"
)

func makeInput(args map[string]any, vars map[string]string, globalVar func(string) string) ops.OpInput {
	return ops.OpInput{
		Node:      &tree.Node{Args: args},
		Variables: vars,
		GlobalVar: globalVar,
	}
}

func TestDecodeArgs_SimpleStrings(t *testing.T) {
	type Args struct {
		Name   string `json:"name"`
		SetVar string `json:"setVar"`
	}

	in := makeInput(
		map[string]any{"name": "hello ${first}", "setVar": "result"},
		map[string]string{"first": "world"},
		nil,
	)
	var out Args
	if err := ops.DecodeArgs(in, &out); err != nil {
		t.Fatal(err)
	}
	if out.Name != "hello world" {
		t.Errorf("Name: want %q, got %q", "hello world", out.Name)
	}
	if out.SetVar != "result" {
		t.Errorf("SetVar: want %q, got %q", "result", out.SetVar)
	}
}

func TestDecodeArgs_GlobalVar(t *testing.T) {
	type Args struct {
		Name string `json:"name"`
	}
	globals := map[string]string{"calName": "Business Hours"}

	in := makeInput(
		map[string]any{"name": "$${calName}"},
		map[string]string{},
		func(k string) string { return globals[k] },
	)
	var out Args
	if err := ops.DecodeArgs(in, &out); err != nil {
		t.Fatal(err)
	}
	if out.Name != "Business Hours" {
		t.Errorf("Name: want %q, got %q", "Business Hours", out.Name)
	}
}

func TestDecodeArgs_PtrStringAndInt(t *testing.T) {
	type Args struct {
		Name *string `json:"name"`
		Id   *int    `json:"id"`
	}

	in := makeInput(
		map[string]any{"name": "Cal ${suffix}", "id": 42},
		map[string]string{"suffix": "A"},
		nil,
	)
	var out Args
	if err := ops.DecodeArgs(in, &out); err != nil {
		t.Fatal(err)
	}
	if out.Name == nil || *out.Name != "Cal A" {
		t.Errorf("Name: want %q, got %v", "Cal A", out.Name)
	}
	if out.Id == nil || *out.Id != 42 {
		t.Errorf("Id: want 42, got %v", out.Id)
	}
}

func TestDecodeArgs_Bool(t *testing.T) {
	type Args struct {
		Extended bool `json:"extended"`
	}
	in := makeInput(map[string]any{"extended": true}, nil, nil)
	var out Args
	if err := ops.DecodeArgs(in, &out); err != nil {
		t.Fatal(err)
	}
	if !out.Extended {
		t.Error("Extended should be true")
	}
}

func TestDecodeArgs_NestedStruct(t *testing.T) {
	type Inner struct {
		Value string `json:"value"`
	}
	type Args struct {
		Items []Inner `json:"items"`
	}

	in := makeInput(
		map[string]any{
			"items": []any{
				map[string]any{"value": "item-${n}"},
			},
		},
		map[string]string{"n": "1"},
		nil,
	)
	var out Args
	if err := ops.DecodeArgs(in, &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Items) != 1 || out.Items[0].Value != "item-1" {
		t.Errorf("Items: got %+v", out.Items)
	}
}

func TestDecodeArgs_SliceFromJSONString(t *testing.T) {
	type Args struct {
		Ids []int `json:"ids"`
	}

	in := makeInput(
		map[string]any{"ids": "${myIds}"},
		map[string]string{"myIds": "[1,2,3]"},
		nil,
	)
	var out Args
	if err := ops.DecodeArgs(in, &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Ids) != 3 || out.Ids[0] != 1 || out.Ids[2] != 3 {
		t.Errorf("Ids: got %v", out.Ids)
	}
}
