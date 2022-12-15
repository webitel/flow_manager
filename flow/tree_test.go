package flow

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/webitel/flow_manager/model"
)

var (
	s = []byte(`[
    {
        "log first": ""
    },
    {
        "log second for tag": "",
		"tag": "second tag"
    },
    {
        "if": {
            "expression": "1==1",
            "then": [
                {
                    "log then" : "then"
                },
				{
					"if": {
						"expression": "2==2",
						"then": [
							{
								"log second then" : "then"
							},
							{
								"log three then" : "then"
							},
							{
								"tag": "tag switch",
								"switch1": {
									"variable": "2",
									"case": {
										"c1": [
											{
												"log c1": "log c1"
											},
											{
												"goto": "tag switch"
											}
										],
										"c2": [
											{
												"log c2": "log c2"
											}
										]
									}
								}
							}
						],
						"else": [
							{
								"log second else": "else"
							}
						]
					}
				},
                {
                    "log then last" : "then"
                }
            ],
            "else": [
                {
                    "log else": "else"
                }
            ]
        }
    },

    {
        "hangup": ""
    },
    {
        "log end": "END"
    }
]`)
)

func TestLabeledTree(t *testing.T) {
	var apps model.Applications
	err := json.Unmarshal(s, &apps)
	if err != nil {
		panic(err.Error())
	}

	m := NewLabeledTree(apps)
	for {
		v, ok := m.Next()
		if !ok {
			fmt.Printf(" > %s \n", "EOF")
			break
		}

		fmt.Printf("> %v ", v.Name)

		if v.Name == "goto" {
			tag, _ := v.Val.args.(string)
			m.Goto(tag)
			continue
		}

		switch c := v.Val.args.(type) {
		case *ConditionVal:
			m.Current = c.Then
		case *SwitchVal:
			if k, ok := c.Cases["c1"]; ok {
				m.Current = k
			}
		}
	}
	m.print()
}

func BenchmarkLabeledTree(b *testing.B) {
	var apps model.Applications
	err := json.Unmarshal(s, &apps)
	if err != nil {
		panic(err.Error())
	}

	for i := 0; i < b.N; i++ {
		NewLabeledTree(apps)
	}
}

func BenchmarkOld(b *testing.B) {
	var apps model.Applications
	err := json.Unmarshal(s, &apps)
	if err != nil {
		panic(err.Error())
	}

	for i := 0; i < b.N; i++ {
		New(Config{
			Schema: apps,
		})
	}
}
