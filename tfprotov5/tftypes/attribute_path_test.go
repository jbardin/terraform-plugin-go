package tftypes

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/google/go-cmp/cmp"
)

type attributePathStepperTestStruct struct {
	Name   string
	Colors attributePathStepperTestSlice
}

func (a attributePathStepperTestStruct) ApplyTerraform5AttributePathStep(step AttributePathStep) (interface{}, error) {
	attributeName, ok := step.(AttributeName)
	if !ok {
		return nil, fmt.Errorf("unsupported attribute path step type: %T", step)
	}
	switch attributeName {
	case "Name":
		return a.Name, nil
	case "Colors":
		return a.Colors, nil
	}
	return nil, fmt.Errorf("unsupported attribute path step attribute name: %q", attributeName)
}

type attributePathStepperTestSlice []string

func (a attributePathStepperTestSlice) ApplyTerraform5AttributePathStep(step AttributePathStep) (interface{}, error) {
	element, ok := step.(ElementKeyInt)
	if !ok {
		return nil, fmt.Errorf("unsupported attribute path step type: %T", step)
	}
	if element >= 0 && int(element) < len(a) {
		return a[element], nil
	}
	return nil, fmt.Errorf("unsupported attribute path step element key: %d", element)
}

func TestWalkAttributePath(t *testing.T) {
	t.Parallel()
	type testCase struct {
		value    interface{}
		path     *AttributePath
		expected interface{}
	}
	tests := map[string]testCase{
		"msi-root": {
			value: map[string]interface{}{
				"a": map[string]interface{}{
					"red":  true,
					"blue": 123,
				},
				"b": map[string]interface{}{
					"red":  false,
					"blue": 234,
				},
			},
			path: &AttributePath{
				steps: []AttributePathStep{
					AttributeName("a"),
				},
			},
			expected: map[string]interface{}{
				"red":  true,
				"blue": 123,
			},
		},
		"msi-full": {
			value: map[string]interface{}{
				"a": map[string]interface{}{
					"red":  true,
					"blue": 123,
				},
				"b": map[string]interface{}{
					"red":  false,
					"blue": 234,
				},
			},
			path: &AttributePath{
				steps: []AttributePathStep{
					AttributeName("a"),
					AttributeName("red"),
				},
			},
			expected: true,
		},
		"slice-interface-root": {
			value: []interface{}{
				map[string]interface{}{
					"a": true,
					"b": 123,
					"c": "hello",
				},
				map[string]interface{}{
					"a": false,
					"b": 1234,
					"c": []interface{}{
						"hello world",
						"happy terraforming",
					},
				},
			},
			path: &AttributePath{
				steps: []AttributePathStep{
					ElementKeyInt(1),
				},
			},
			expected: map[string]interface{}{
				"a": false,
				"b": 1234,
				"c": []interface{}{
					"hello world",
					"happy terraforming",
				},
			},
		},
		"slice-interface-full": {
			value: []interface{}{
				map[string]interface{}{
					"a": true,
					"b": 123,
					"c": "hello",
				},
				map[string]interface{}{
					"a": false,
					"b": 1234,
					"c": []interface{}{
						"hello world",
						"happy terraforming",
					},
				},
			},
			path: &AttributePath{
				steps: []AttributePathStep{
					ElementKeyInt(1),
					AttributeName("c"),
					ElementKeyInt(0),
				},
			},
			expected: "hello world",
		},
		"attributepathstepper": {
			value: []interface{}{
				attributePathStepperTestStruct{
					Name: "terraform",
					Colors: []string{
						"purple", "white",
					},
				},
				attributePathStepperTestStruct{
					Name: "nomad",
					Colors: []string{
						"green",
					},
				},
			},
			path: &AttributePath{
				steps: []AttributePathStep{
					ElementKeyInt(1),
					AttributeName("Colors"),
					ElementKeyInt(0),
				},
			},
			expected: "green",
		},
	}

	for name, test := range tests {
		name, test := name, test
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			result, remaining, err := WalkAttributePath(test.value, test.path)
			if err != nil {
				t.Fatalf("error walking attribute path, %v still remains in the path: %s", remaining, err)
			}
			if diff := cmp.Diff(test.expected, result, cmp.Comparer(numberComparer), ValueComparer()); diff != "" {
				t.Errorf("Unexpected results (-wanted +got): %s", diff)
			}
		})
	}
}

func TestAttributePathEqual(t *testing.T) {
	t.Parallel()
	type testCase struct {
		path1 *AttributePath
		path2 *AttributePath
		equal bool
	}

	tests := map[string]testCase{
		"empty": {
			path1: NewAttributePath(),
			path2: NewAttributePath(),
			equal: true,
		},
		"nil": {
			equal: true,
		},
		"empty-and-nil": {
			path1: NewAttributePath(),
			equal: true,
		},
		"an-different-types": {
			path1: NewAttributePath().WithAttributeName("testing"),
			path2: NewAttributePath().WithElementKeyString("testing"),
			equal: false,
		},
		"eks-different-types": {
			path1: NewAttributePath().WithElementKeyString("testing"),
			path2: NewAttributePath().WithAttributeName("testing"),
			equal: false,
		},
		"eki-different-types": {
			path1: NewAttributePath().WithElementKeyInt(1234),
			path2: NewAttributePath().WithAttributeName("testing"),
			equal: false,
		},
		"ekv-different-types": {
			path1: NewAttributePath().WithElementKeyValue(NewValue(String, "testing")),
			path2: NewAttributePath().WithAttributeName("testing"),
			equal: false,
		},
		"an": {
			path1: NewAttributePath().WithAttributeName("testing"),
			path2: NewAttributePath().WithAttributeName("testing"),
			equal: true,
		},
		"an-an": {
			path1: NewAttributePath().WithAttributeName("testing").WithAttributeName("testing2"),
			path2: NewAttributePath().WithAttributeName("testing").WithAttributeName("testing2"),
			equal: true,
		},
		"eks": {
			path1: NewAttributePath().WithElementKeyString("testing"),
			path2: NewAttributePath().WithElementKeyString("testing"),
			equal: true,
		},
		"eks-eks": {
			path1: NewAttributePath().WithElementKeyString("testing").WithElementKeyString("testing2"),
			path2: NewAttributePath().WithElementKeyString("testing").WithElementKeyString("testing2"),
			equal: true,
		},
		"eki": {
			path1: NewAttributePath().WithElementKeyInt(123),
			path2: NewAttributePath().WithElementKeyInt(123),
			equal: true,
		},
		"eki-eki": {
			path1: NewAttributePath().WithElementKeyInt(123).WithElementKeyInt(456),
			path2: NewAttributePath().WithElementKeyInt(123).WithElementKeyInt(456),
			equal: true,
		},
		"ekv": {
			path1: NewAttributePath().WithElementKeyValue(NewValue(List{
				ElementType: String,
			}, []Value{
				NewValue(String, "hello"),
				NewValue(String, "world"),
			})),
			path2: NewAttributePath().WithElementKeyValue(NewValue(List{
				ElementType: String,
			}, []Value{
				NewValue(String, "hello"),
				NewValue(String, "world"),
			})),
			equal: true,
		},
		"ekv-ekv": {
			path1: NewAttributePath().WithElementKeyValue(NewValue(List{
				ElementType: String,
			}, []Value{
				NewValue(String, "hello"),
				NewValue(String, "world"),
			})).WithElementKeyValue(NewValue(Bool, true)),
			path2: NewAttributePath().WithElementKeyValue(NewValue(List{
				ElementType: String,
			}, []Value{
				NewValue(String, "hello"),
				NewValue(String, "world"),
			})).WithElementKeyValue(NewValue(Bool, true)),
			equal: true,
		},
		"an-eks-eki-ekv": {
			path1: NewAttributePath().WithAttributeName("testing").WithElementKeyString("testing2").WithElementKeyInt(123).WithElementKeyValue(NewValue(String, "hello, world")),
			path2: NewAttributePath().WithAttributeName("testing").WithElementKeyString("testing2").WithElementKeyInt(123).WithElementKeyValue(NewValue(String, "hello, world")),
			equal: true,
		},
		"ekv-eki-eks-an": {
			path1: NewAttributePath().WithElementKeyValue(NewValue(Object{
				AttributeTypes: map[string]Type{
					"foo": Bool,
					"bar": Number,
				},
			}, map[string]Value{
				"foo": NewValue(Bool, true),
				"bar": NewValue(Number, big.NewFloat(1234)),
			})).WithElementKeyInt(123).WithElementKeyString("testing").WithAttributeName("othertesting"),
			path2: NewAttributePath().WithElementKeyValue(NewValue(Object{
				AttributeTypes: map[string]Type{
					"foo": Bool,
					"bar": Number,
				},
			}, map[string]Value{
				"foo": NewValue(Bool, true),
				"bar": NewValue(Number, big.NewFloat(1234)),
			})).WithElementKeyInt(123).WithElementKeyString("testing").WithAttributeName("othertesting"),
			equal: true,
		},
		"an-diff": {
			path1: NewAttributePath().WithAttributeName("testing"),
			path2: NewAttributePath().WithAttributeName("testing2"),
			equal: false,
		},
		"an-an-diff": {
			path1: NewAttributePath().WithAttributeName("testing").WithAttributeName("testing2"),
			path2: NewAttributePath().WithAttributeName("testing2").WithAttributeName("testing2"),
			equal: false,
		},
		"an-an-diff-2": {
			path1: NewAttributePath().WithAttributeName("testing").WithAttributeName("testing2"),
			path2: NewAttributePath().WithAttributeName("testing").WithAttributeName("testing3"),
			equal: false,
		},
		"eks-diff": {
			path1: NewAttributePath().WithElementKeyString("testing"),
			path2: NewAttributePath().WithElementKeyString("testing2"),
			equal: false,
		},
		"eks-eks-diff": {
			path1: NewAttributePath().WithElementKeyString("testing").WithElementKeyString("testing2"),
			path2: NewAttributePath().WithElementKeyString("testing2").WithElementKeyString("testing2"),
			equal: false,
		},
		"eks-eks-diff-2": {
			path1: NewAttributePath().WithElementKeyString("testing").WithElementKeyString("testing2"),
			path2: NewAttributePath().WithElementKeyString("testing").WithElementKeyString("testing3"),
			equal: false,
		},
		"eki-diff": {
			path1: NewAttributePath().WithElementKeyInt(123),
			path2: NewAttributePath().WithElementKeyInt(1234),
			equal: false,
		},
		"eki-eki-diff": {
			path1: NewAttributePath().WithElementKeyInt(123).WithElementKeyInt(456),
			path2: NewAttributePath().WithElementKeyInt(1234).WithElementKeyInt(456),
			equal: false,
		},
		"eki-eki-diff-2": {
			path1: NewAttributePath().WithElementKeyInt(123).WithElementKeyInt(456),
			path2: NewAttributePath().WithElementKeyInt(123).WithElementKeyInt(4567),
			equal: false,
		},
		"ekv-diff": {
			path1: NewAttributePath().WithElementKeyValue(NewValue(List{
				ElementType: String,
			}, []Value{
				NewValue(String, "hello"),
				NewValue(String, "world"),
			})),
			path2: NewAttributePath().WithElementKeyValue(NewValue(List{
				ElementType: String,
			}, []Value{
				NewValue(String, "hello"),
				NewValue(String, "fren"),
			})),
			equal: false,
		},
		"ekv-ekv-diff": {
			path1: NewAttributePath().WithElementKeyValue(NewValue(List{
				ElementType: String,
			}, []Value{
				NewValue(String, "hello"),
				NewValue(String, "world"),
			})).WithElementKeyValue(NewValue(Bool, true)),
			path2: NewAttributePath().WithElementKeyValue(NewValue(List{
				ElementType: String,
			}, []Value{
				NewValue(String, "hello"),
				NewValue(String, "fren"),
			})).WithElementKeyValue(NewValue(Bool, true)),
			equal: false,
		},
		"ekv-ekv-diff-2": {
			path1: NewAttributePath().WithElementKeyValue(NewValue(List{
				ElementType: String,
			}, []Value{
				NewValue(String, "hello"),
				NewValue(String, "world"),
			})).WithElementKeyValue(NewValue(Bool, true)),
			path2: NewAttributePath().WithElementKeyValue(NewValue(List{
				ElementType: String,
			}, []Value{
				NewValue(String, "hello"),
				NewValue(String, "world"),
			})).WithElementKeyValue(NewValue(Bool, false)),
			equal: false,
		},
		"an-eks-eki-ekv-diff": {
			path1: NewAttributePath().WithAttributeName("testing").WithElementKeyString("testing2").WithElementKeyInt(123).WithElementKeyValue(NewValue(String, "hello, world")),
			path2: NewAttributePath().WithAttributeName("testing2").WithElementKeyString("testing2").WithElementKeyInt(123).WithElementKeyValue(NewValue(String, "hello, world")),
			equal: false,
		},
		"an-eks-eki-ekv-diff-2": {
			path1: NewAttributePath().WithAttributeName("testing").WithElementKeyString("testing2").WithElementKeyInt(123).WithElementKeyValue(NewValue(String, "hello, world")),
			path2: NewAttributePath().WithAttributeName("testing").WithElementKeyString("testing3").WithElementKeyInt(123).WithElementKeyValue(NewValue(String, "hello, world")),
			equal: false,
		},
		"an-eks-eki-ekv-diff-3": {
			path1: NewAttributePath().WithAttributeName("testing").WithElementKeyString("testing2").WithElementKeyInt(123).WithElementKeyValue(NewValue(String, "hello, world")),
			path2: NewAttributePath().WithAttributeName("testing").WithElementKeyString("testing2").WithElementKeyInt(1234).WithElementKeyValue(NewValue(String, "hello, world")),
			equal: false,
		},
		"an-eks-eki-ekv-diff-4": {
			path1: NewAttributePath().WithAttributeName("testing").WithElementKeyString("testing2").WithElementKeyInt(123).WithElementKeyValue(NewValue(String, "hello, world")),
			path2: NewAttributePath().WithAttributeName("testing").WithElementKeyString("testing2").WithElementKeyInt(123).WithElementKeyValue(NewValue(String, "hello, friend")),
			equal: false,
		},
		"ekv-eki-eks-an-diff": {
			path1: NewAttributePath().WithElementKeyValue(NewValue(Object{
				AttributeTypes: map[string]Type{
					"foo": Bool,
					"bar": Number,
				},
			}, map[string]Value{
				"foo": NewValue(Bool, true),
				"bar": NewValue(Number, big.NewFloat(1234)),
			})).WithElementKeyInt(123).WithElementKeyString("testing").WithAttributeName("othertesting"),
			path2: NewAttributePath().WithElementKeyValue(NewValue(Object{
				AttributeTypes: map[string]Type{
					"foo": Bool,
					"bar": Number,
				},
			}, map[string]Value{
				"foo": NewValue(Bool, true),
				"bar": NewValue(Number, big.NewFloat(12345)),
			})).WithElementKeyInt(123).WithElementKeyString("testing").WithAttributeName("othertesting"),
			equal: false,
		},
		"ekv-eki-eks-an-diff-2": {
			path1: NewAttributePath().WithElementKeyValue(NewValue(Object{
				AttributeTypes: map[string]Type{
					"foo": Bool,
					"bar": Number,
				},
			}, map[string]Value{
				"foo": NewValue(Bool, true),
				"bar": NewValue(Number, big.NewFloat(1234)),
			})).WithElementKeyInt(123).WithElementKeyString("testing").WithAttributeName("othertesting"),
			path2: NewAttributePath().WithElementKeyValue(NewValue(Object{
				AttributeTypes: map[string]Type{
					"foo": Bool,
					"bar": Number,
				},
			}, map[string]Value{
				"foo": NewValue(Bool, true),
				"bar": NewValue(Number, big.NewFloat(1234)),
			})).WithElementKeyInt(1234).WithElementKeyString("testing").WithAttributeName("othertesting"),
			equal: false,
		},
		"ekv-eki-eks-an-diff-3": {
			path1: NewAttributePath().WithElementKeyValue(NewValue(Object{
				AttributeTypes: map[string]Type{
					"foo": Bool,
					"bar": Number,
				},
			}, map[string]Value{
				"foo": NewValue(Bool, true),
				"bar": NewValue(Number, big.NewFloat(1234)),
			})).WithElementKeyInt(123).WithElementKeyString("testing").WithAttributeName("othertesting"),
			path2: NewAttributePath().WithElementKeyValue(NewValue(Object{
				AttributeTypes: map[string]Type{
					"foo": Bool,
					"bar": Number,
				},
			}, map[string]Value{
				"foo": NewValue(Bool, true),
				"bar": NewValue(Number, big.NewFloat(1234)),
			})).WithElementKeyInt(123).WithElementKeyString("testing2").WithAttributeName("othertesting"),
			equal: false,
		},
		"ekv-eki-eks-an-diff-4": {
			path1: NewAttributePath().WithElementKeyValue(NewValue(Object{
				AttributeTypes: map[string]Type{
					"foo": Bool,
					"bar": Number,
				},
			}, map[string]Value{
				"foo": NewValue(Bool, true),
				"bar": NewValue(Number, big.NewFloat(1234)),
			})).WithElementKeyInt(123).WithElementKeyString("testing").WithAttributeName("othertesting"),
			path2: NewAttributePath().WithElementKeyValue(NewValue(Object{
				AttributeTypes: map[string]Type{
					"foo": Bool,
					"bar": Number,
				},
			}, map[string]Value{
				"foo": NewValue(Bool, true),
				"bar": NewValue(Number, big.NewFloat(1234)),
			})).WithElementKeyInt(123).WithElementKeyString("testing").WithAttributeName("othertesting2"),
			equal: false,
		},
	}

	for name, test := range tests {
		name, test := name, test
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			isEqual := test.path1.Equal(test.path2)
			if isEqual != test.equal {
				t.Fatalf("expected %v, got %v", test.equal, isEqual)
			}
			isEqual = test.path2.Equal(test.path1)
			if isEqual != test.equal {
				t.Fatalf("expected %v, got %v", test.equal, isEqual)
			}
		})
	}
}
