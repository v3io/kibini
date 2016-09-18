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
	logger                        logging.Logger
	waitGroup                     *sync.WaitGroup
	stopAfterFirstFlush           bool
	inactivityFlushTimeout        time.Duration
	forceFlushTimeout             time.Duration
	writers                       []logWriter
	incomingRecords               chan *logRecord
	pendingRecords                logRecordSorter
	newestPendingRecordReceivedAt time.Time
	oldestPendingRecordReceivedAt time.Time
}

func newLogMerger(logger logging.Logger,
	waitGroup *sync.WaitGroup,
	stopAfterFirstFlush bool,
	inactivityFlushTimeout time.Duration,
	forceFlushTimeout time.Duration,
	writers []logWriter) *logMerger {

	lm := &logMerger{
		logger:                        logger.GetChild("merger"),
		waitGroup:                     waitGroup,
		stopAfterFirstFlush:           stopAfterFirstFlush,
		inactivityFlushTimeout:        inactivityFlushTimeout,
		forceFlushTimeout:             forceFlushTimeout,
		writers:                       writers,
		incomingRecords:               make(chan *logRecord),
		pendingRecords:                logRecordSorter{},
		newestPendingRecordReceivedAt: time.Now(),
		oldestPendingRecordReceivedAt: time.Now(),
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
			now := time.Now()

			// if we're the first record shoved into the pending records, set the the proper field
			if len(lm.pendingRecords) == 0 {
				lm.oldestPendingRecordReceivedAt = now
			}

			// write it to the pendingRecords
			lm.pendingRecords = append(lm.pendingRecords, incomingRecord)

			// update the last time we got a record
			lm.newestPendingRecordReceivedAt = now

			// check if we need to flush
			lm.checkFlushRequired()

		// if nothing arrives in the queue, after 250ms check if flush is required
		case <-time.After(250 * time.Millisecond):

			// if there are any pending records
			if lm.pendingRecords.Len() != 0 {
				quit = lm.checkFlushRequired() && lm.stopAfterFirstFlush
			}
		}
	}

	lm.logger.Debug("Done processing incoming records")

	// signal that we're done
	lm.waitGroup.Done()
}

func (lm *logMerger) checkFlushRequired() bool {

	// and inactivityFlushTimeout seconds passed since we got the newest pending record
	if time.Since(lm.newestPendingRecordReceivedAt) > lm.inactivityFlushTimeout ||

		// or the oldest pending record is older than the force flush timeout and forceFlushTimeout
		// is enabled (== non-zero)
		(lm.forceFlushTimeout != 0 && time.Since(lm.oldestPendingRecordReceivedAt) > lm.forceFlushTimeout) {

		// flush the pending records
		lm.flushPendingRecords()

		return true
	}

	return false
}

func (lm *logMerger) flushPendingRecords() {

	// start by sorting the pending records by time
	sort.Sort(lm.pendingRecords)

	// now flush them towards the writers
	for _, logRecord := range lm.pendingRecords {
		for _, writer := range lm.writers {
			writer.Write(logRecord)
		}
	}

	// clean out the pending records
	lm.pendingRecords = nil
}
