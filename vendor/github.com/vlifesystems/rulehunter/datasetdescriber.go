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

package rulehunter

import (
	"github.com/vlifesystems/rulehunter/dataset"
	"github.com/vlifesystems/rulehunter/description"
)

func DescribeDataset(
	dataset dataset.Dataset,
) (*description.Description, error) {
	_description := description.New()
	dataset.Rewind()

	for dataset.Next() {
		record, err := dataset.Read()
		if err != nil {
			return _description, err
		}
		_description.NextRecord(record)
	}

	return _description, dataset.Err()
}