package core

import (
	"sort"
	"sync"
	"time"

	"github.com/iguazio/kibini/logger"
)

//
// Object that holds lots of log records and can sort them by time
//

type logRecordSorter []*logRecord

func (lrs logRecordSorter) Len() int           { return len(lrs) }
func (lrs logRecordSorter) Swap(i, j int)      { lrs[i], lrs[j] = lrs[j], lrs[i] }
func (lrs logRecordSorter) Less(i, j int) bool { return lrs[i].WhenUnixNano < lrs[j].WhenUnixNano }

//
// Object which receives log records from many go routines and then after 1 second of inactivity
// flushes them to the writer, sorted
//

type logMerger struct {
	logger                      logging.Logger
	waitGroup                   *sync.WaitGroup
	stopAfterFirstFlush         bool
	inactivityFlushTimeout      time.Duration
	writer                      logWriter
	incomingRecords             chan *logRecord
	pendingRecords              logRecordSorter
	lastPendingRecordReceivedAt time.Time
}

func newLogMerger(logger logging.Logger,
	waitGroup *sync.WaitGroup,
	stopAfterFirstFlush bool,
	inactivityFlushTimeout time.Duration,
	writer logWriter) *logMerger {

	lm := &logMerger{
		logger:                      logger.GetChild("merger"),
		waitGroup:                   waitGroup,
		stopAfterFirstFlush:         stopAfterFirstFlush,
		inactivityFlushTimeout:      inactivityFlushTimeout,
		writer:                      writer,
		incomingRecords:             make(chan *logRecord),
		pendingRecords:              logRecordSorter{},
		lastPendingRecordReceivedAt: time.Now(),
	}

	// increment wait group (will be signaled when we're done)
	waitGroup.Add(1)

	go lm.processIncomingRecords()

	return lm
}

func (lm *logMerger) Write(logRecord *logRecord) error {

	// write the record to the channel
	lm.incomingRecords <- logRecord

	return nil
}

func (lm *logMerger) processIncomingRecords() {
	lm.logger.Debug("Processing incoming records")

	quit := false

	for !quit {

		select {

		// received a new incoming record
		case incomingRecord := <-lm.incomingRecords:

			// write it to the pendingRecords
			lm.pendingRecords = append(lm.pendingRecords, incomingRecord)

			// update the last time we got a record
			lm.lastPendingRecordReceivedAt = time.Now()

		// every 1 second, check if we need to flush anything
		case <-time.After(time.Second):

			// check if there are any pending incoming requests and whether
			// enough time has passed to flush them
			if lm.pendingRecords.Len() != 0 &&
				time.Since(lm.lastPendingRecordReceivedAt) > lm.inactivityFlushTimeout {

				// flush the pending records
				lm.flushPendingRecords()

				// quit if we need to stop after first flush
				quit = lm.stopAfterFirstFlush
			}
		}
	}

	lm.logger.Debug("Done processing incoming records")

	// signal that we're done
	lm.waitGroup.Done()
}

func (lm *logMerger) flushPendingRecords() {
	lm.logger.With(logging.Fields{
		"numPending": lm.pendingRecords.Len(),
	}).Debug("Flushing pending records")

	// start by sorting the pending records by time
	sort.Sort(lm.pendingRecords)

	// now flush them towards the writer
	for _, logRecord := range lm.pendingRecords {
		lm.writer.Write(logRecord)
	}

	// clean out the pending records
	lm.pendingRecords = nil
}
