// Copyright © 2020 Banzai Cloud
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package workflow

import (
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/vmware/govmomi/vim25/progress"
	"go.uber.org/zap"
)

type progressLogger struct {
	taskName string
	logger   *zap.SugaredLogger

	wg sync.WaitGroup

	sink chan chan progress.Report
	done chan struct{}
}

func newProgressLogger(taskName string, logger *zap.SugaredLogger) *progressLogger {
	p := &progressLogger{
		taskName: taskName,
		logger:   logger,

		sink: make(chan chan progress.Report),
		done: make(chan struct{}),
	}

	p.wg.Add(1)

	go p.loopA()

	return p
}

// loopA runs before Sink() has been called.
func (p *progressLogger) loopA() {
	var err error

	defer p.wg.Done()

	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()

	called := false

	for stop := false; !stop; {
		select {
		case ch := <-p.sink:
			err = p.loopB(tick, ch)
			stop = true
			called = true
		case <-p.done:
			stop = true
		case <-tick.C:
			line := fmt.Sprintf("%s", p.taskName)
			p.logger.Debug(line)
		}
	}

	if err != nil && err != io.EOF {
		p.logger.Debug(fmt.Sprintf("%sError: %s\n", p.taskName, err))
	} else if called {
		p.logger.Debug(fmt.Sprintf("%sOK\n", p.taskName))
	}
}

// loopA runs after Sink() has been called.
func (p *progressLogger) loopB(tick *time.Ticker, ch <-chan progress.Report) error {
	var r progress.Report
	var ok bool
	var err error

	for ok = true; ok; {
		select {
		case r, ok = <-ch:
			if !ok {
				break
			}
			err = r.Error()
		case <-tick.C:
			line := fmt.Sprintf("%s", p.taskName)
			if r != nil {
				line += fmt.Sprintf("(%.0f%%", r.Percentage())
				detail := r.Detail()
				if detail != "" {
					line += fmt.Sprintf(", %s", detail)
				}
				line += ")"
			}
			p.logger.Info(line)
		}
	}

	return err
}

func (p *progressLogger) Sink() chan<- progress.Report {
	ch := make(chan progress.Report)
	p.sink <- ch
	return ch
}

func (p *progressLogger) Wait() {
	close(p.done)
	p.wg.Wait()
}
