/*
	rulehuntersrv - A server to find rules in data based on user specified goals
	Copyright (C) 2016 vLife Systems Ltd <http://vlifesystems.com>

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU Affero General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU Affero General Public License for more details.

	You should have received a copy of the GNU Affero General Public License
	along with this program; see the file COPYING.  If not, see
	<http://www.gnu.org/licenses/>.
*/

package html

import (
	"github.com/vlifesystems/rulehuntersrv/config"
	"github.com/vlifesystems/rulehuntersrv/progress"
	"html/template"
	"path/filepath"
	"time"
)

func generateProgressPage(
	config *config.Config,
	progressMonitor *progress.ProgressMonitor,
) error {
	type TplExperiment struct {
		Title    string
		Tags     map[string]string
		Stamp    string
		Filename string
		Status   string
		Msg      string
	}

	type TplData struct {
		Experiments []*TplExperiment
		Html        map[string]template.HTML
	}

	experiments := progressMonitor.GetExperiments()

	tplExperiments := make([]*TplExperiment, len(experiments))

	for i, experiment := range experiments {
		tplExperiments[i] = &TplExperiment{
			experiment.Title,
			makeTagLinks(experiment.Tags),
			experiment.Stamp.Format(time.RFC822),
			experiment.ExperimentFilename,
			experiment.Status.String(),
			experiment.Msg,
		}
	}
	tplData := TplData{tplExperiments, makeHtml("progress")}

	outputFilename := filepath.Join(config.WWWDir, "progress", "index.html")
	return writeTemplate(outputFilename, progressTpl, tplData)
}
