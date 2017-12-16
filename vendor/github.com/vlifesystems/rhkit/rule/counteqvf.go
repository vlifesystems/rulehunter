// Copyright (C) 2017 vLife Systems Ltd <http://vlifesystems.com>
// Licensed under an MIT licence.  Please see LICENSE.md for details.

package rule

import (
	"fmt"
	"strings"

	"github.com/lawrencewoodman/ddataset"
	"github.com/lawrencewoodman/dlit"
	"github.com/vlifesystems/rhkit/description"
)

// CountEQVF represents a rule determining if a count of the number
// of fields supplied containing a supplied string is equal to
// a value.
type CountEQVF struct {
	value  *dlit.Literal
	fields []string
	num    int64
}

func NewCountEQVF(value *dlit.Literal, fields []string, num int64) *CountEQVF {
	if len(fields) < 2 {
		panic("NewCountEQVF: Must contain at least two fields")
	}
	return &CountEQVF{value: value, fields: fields, num: num}
}

func (r *CountEQVF) String() string {
	return fmt.Sprintf("count(\"%s\", %s) == %d",
		r.value, strings.Join(r.fields, ", "), r.num)
}

func (r *CountEQVF) Fields() []string {
	return r.fields
}

func (r *CountEQVF) IsTrue(record ddataset.Record) (bool, error) {
	n := int64(0)
	for _, f := range r.fields {
		fieldValue, ok := record[f]
		if !ok {
			return false, InvalidRuleError{Rule: r}
		}
		if fieldValue.Err() != nil {
			return false, IncompatibleTypesRuleError{Rule: r}
		}
		if fieldValue.String() == r.value.String() {
			n++
		}
	}
	return n == r.num, nil
}

func generateCountEQVF(
	inputDescription *description.Description,
	generationDesc GenerationDescriber,
) []Rule {
	validFields := []string{}
	for _, f := range generationDesc.Fields() {
		fd := inputDescription.Fields[f]
		if fd.NumValues >= 2 && fd.NumValues <= 4 &&
			(fd.Kind == description.String || fd.Kind == description.Number) {
			validFields = append(validFields, f)
		}
	}

	allValues := []*dlit.Literal{}
	allValuesMap := map[string]bool{}
	for _, f := range validFields {
		for _, v := range inputDescription.Fields[f].Values {
			if v.Num >= 2 {
				if _, ok := allValuesMap[v.Value.String()]; !ok {
					allValues = append(allValues, v.Value)
					allValuesMap[v.Value.String()] = true
				}
			}
		}
	}

	isValueInField := func(v *dlit.Literal, field string) bool {
		for _, fv := range inputDescription.Fields[field].Values {
			if fv.Value.String() == v.String() {
				return true
			}
		}
		return false
	}

	isValueInAllFields := func(v *dlit.Literal, fields []string) bool {
		for _, f := range fields {
			if !isValueInField(v, f) {
				return false
			}
		}
		return true
	}

	possibleValues := []*dlit.Literal{}
	for _, v := range allValues {
		presentInNumFields := 0
		for _, f := range validFields {
			if isValueInField(v, f) {
				presentInNumFields++
			}
		}
		if presentInNumFields >= 2 {
			possibleValues = append(possibleValues, v)
		}
	}

	possibleFields := []string{}
	possibleFieldsMap := map[string]bool{}
	for _, v := range possibleValues {
		for _, f := range validFields {
			if _, ok := possibleFieldsMap[f]; !ok && isValueInField(v, f) {
				possibleFields = append(possibleFields, f)
				possibleFieldsMap[f] = true
			}
		}
	}

	if len(possibleFields) < 2 {
		return []Rule{}
	}

	rules := make([]Rule, 0)
	maxNumFields := 40.0 / len(possibleFields)
	for _, v := range possibleValues {
		for _, fields := range stringCombinations(possibleFields, 2, maxNumFields) {
			if isValueInAllFields(v, fields) {
				for n := int64(0); n <= int64(len(fields)); n++ {
					r := NewCountEQVF(v, fields, n)
					rules = append(rules, r)
				}
			}
		}
	}

	return rules
}

func getMaskStrings(mask string, values []string) []string {
	r := []string{}
	for j, b := range mask {
		if j >= len(values) {
			break
		}
		if b == '1' {
			v := values[j]
			r = append(r, v)
		}
	}
	return r
}

func stringCombinations(values []string, min, max int) [][]string {
	r := [][]string{}
	for i := 3; ; i++ {
		mask := makeMask(len(values), i)
		numOnBits := countNumOnBits(mask)
		if len(mask) > len(values) {
			break
		}
		if numOnBits >= min && numOnBits <= max && numOnBits <= len(values) {
			r = append(r, getMaskStrings(mask, values))
		}
	}
	return r
}
