/*
	Copyright (C) 2016 vLife Systems Ltd <http://vlifesystems.com>
	This file is part of Rulehunter.

	Rulehunter is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	Rulehunter is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with Rulehunter; see the file COPYING.  If not, see
	<http://www.gnu.org/licenses/>.
*/

package rule

import (
	"github.com/lawrencewoodman/ddataset"
	"strconv"
)

// NEFVF represents a rule determening if fieldA != floatValue
type NEFVF struct {
	field string
	value float64
}

func NewNEFVF(field string, value float64) Rule {
	return &NEFVF{field: field, value: value}
}

func (r *NEFVF) String() string {
	return r.field + " != " + strconv.FormatFloat(r.value, 'f', -1, 64)
}

func (r *NEFVF) GetInNiParts() (bool, string, string) {
	return false, "", ""
}

func (r *NEFVF) IsTrue(record ddataset.Record) (bool, error) {
	lh, ok := record[r.field]
	if !ok {
		return false, InvalidRuleError{Rule: r}
	}

	lhFloat, lhIsFloat := lh.Float()
	if lhIsFloat {
		return lhFloat != r.value, nil
	}

	return false, IncompatibleTypesRuleError{Rule: r}
}