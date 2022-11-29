package flow

import (
	"encoding/json"
	"fmt"

	"github.com/webitel/flow_manager/model"
)

const maxGotoLoop = 1000

type TreeKey int

type TreeVal struct {
	Val  interface{}
	Name string
	Root TreeKey
	Out  []TreeKey
}

type ConditionVal struct {
	Expression string
	Then       TreeKey
	Else       TreeKey
}

type SwitchVal struct {
	Variable string
	Cases    map[string]TreeKey
}

type LabeledTree struct {
	Nodes       map[TreeKey]*TreeVal
	Current     TreeKey
	Tags        map[string]TreeKey
	i           TreeKey
	gotoCounter int
}

func NewLabeledTree(apps model.Applications) LabeledTree {
	var m LabeledTree
	m.Nodes = make(map[TreeKey]*TreeVal)
	m.Tags = make(map[string]TreeKey)

	m.append(&TreeVal{
		Val:  nil,
		Out:  make([]TreeKey, 0, 0),
		Root: TreeKey(1),
	}, "")
	m.reverseFill(m.Current, apps)
	m.Current = m.i
	m.i = 0
	return m
}

func (m *LabeledTree) Goto(tag string) bool {
	if m.gotoCounter >= maxGotoLoop {
		m.Current = -1
		return false
	}
	key, ok := m.Tags[tag]
	if !ok {
		return false
	}
	m.gotoCounter++
	m.Current = key
	return true
}

func (m *LabeledTree) Next() (*TreeVal, bool) {
	// TODO for testing
	m.i++
	if m.i > 10000 {
		m.Current = -1
	}
	// TODO end

	v, ok := m.Nodes[m.Current]
	if !ok {
		return nil, false
	}

	if len(v.Out) > 0 {
		m.Current = v.Out[0]
		return v, ok
	} else {
		m.Current = -1
		return nil, false
	}
}

func (m *LabeledTree) append(v *TreeVal, tag string) TreeKey {
	m.i++
	m.Nodes[m.i] = v
	if len(v.Out) != 0 && v.Root == v.Out[0] {
		v.Out = m.getOut(v.Root)
	}

	if tag != "" {
		m.Tags[tag] = m.i
	}
	return m.i
}

func (m *LabeledTree) print() {
	d, _ := json.MarshalIndent(m, "", "    ")
	fmt.Println(string(d))
}

func (m *LabeledTree) getOut(key TreeKey) []TreeKey {
	if v, ok := m.Nodes[key]; ok {
		return v.Out
	}

	return nil
}

func (m *LabeledTree) reverseFill(from TreeKey, apps model.Applications) TreeKey {
	root := from

	for i := len(apps) - 1; i >= 0; i-- {
		r := parseReq(apps[i])

		switch r.Name {
		case "if":
			cnd := ConditionVal{}
			mVal := &TreeVal{
				Name: "if",
				Val:  &cnd,
				Out:  []TreeKey{from},
				Root: root,
			}

			from = m.append(mVal, r.Tag)
			if tmp, ok := r.args.(map[string]interface{}); ok {
				if th, ok := tmp["then"].([]interface{}); ok {
					cnd.Then = m.reverseFill(from, ArrInterfaceToArrayApplication(th))
				}

				if el, ok := tmp["else"].([]interface{}); ok {
					cnd.Else = m.reverseFill(from, ArrInterfaceToArrayApplication(el))
				}

				if ex, ok := tmp["expression"].(string); ok {
					cnd.Expression = parseExpression(ex)
				}
			}
			continue
		case "switch":
			cnd := SwitchVal{
				Cases: make(map[string]TreeKey),
			}
			mVal := &TreeVal{
				Name: "switch",
				Val:  &cnd,
				Out:  []TreeKey{from},
				Root: root,
			}
			from = m.append(mVal, r.Tag)
			if tmp, ok := r.args.(map[string]interface{}); ok {
				var cases map[string]interface{}

				if v, ok := tmp["variable"].(string); ok {
					cnd.Variable = v
				}

				cases, ok = tmp["case"].(map[string]interface{})
				for caseName, caseVal := range cases {
					if c, ok := caseVal.([]interface{}); ok {
						cnd.Cases[caseName] = m.reverseFill(from, ArrInterfaceToArrayApplication(c))
					}
				}
			}

			continue
		case "break":
			from = m.append(&TreeVal{
				Name: r.Name,
				Val:  r.args,
				Out:  []TreeKey{from},
				Root: root,
			}, r.Tag)
		default:
			from = m.append(&TreeVal{
				Name: r.Name,
				Val:  r.args,
				Out:  []TreeKey{from},
				Root: root,
			}, r.Tag)
		}
	}

	return from
}
