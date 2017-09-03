// Copyright (C) 2016 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at http://mozilla.org/MPL/2.0/.

// +build !solaris,!darwin solaris,cgo darwin,cgo

package fs

import (
	"context"
	"errors"
	"path/filepath"
	"strings"

	"github.com/zillode/notify"
)

// Notify does not block on sending to channel, so the channel must be buffered.
// The actual number is magic.
const backendBuffer = 500

func (f *BasicFilesystem) Watch(name string, ignore Matcher, ctx context.Context, ignorePerms bool) (<-chan Event, error) {
	absName := filepath.Join(f.URI(), name)

	absShouldIgnore := func(absPath string) bool {
		if !strings.HasPrefix(absPath, absName) {
			panic("bug: Notify backend is processing a change outside of the watched path")
		}
		relPath := f.unrooted(absPath)
		return ignore.ShouldIgnore(relPath)
	}

	outChan := make(chan Event)
	backendChan := make(chan notify.EventInfo, backendBuffer)

	eventMask := subEventMask
	if !ignorePerms {
		eventMask |= permEventMask
	}

	if err := notify.WatchWithFilter(filepath.Join(absName, "..."), backendChan, absShouldIgnore, eventMask); err != nil {
		notify.Stop(backendChan)
		close(backendChan)
		if reachedMaxUserWatches(err) {
			err = errors.New("failed to install inotify handler. Please increase inotify limits, see https://github.com/syncthing/syncthing-inotify#troubleshooting-for-folders-with-many-files-on-linux for more information")
		}
		return nil, err
	}

	go f.watchLoop(absName, backendChan, outChan, ignore, ctx)

	return outChan, nil
}

func (f *BasicFilesystem) watchLoop(absName string, backendChan chan notify.EventInfo, outChan chan<- Event, ignore Matcher, ctx context.Context) {
	for {
		// Detect channel overflow
		if len(backendChan) == backendBuffer {
		outer:
			for {
				select {
				case <-backendChan:
				default:
					break outer
				}
			}
			// When next scheduling a scan, do it on the entire folder as events have been lost.
			outChan <- Event{Name: ".", Type: NonRemove}
		}

		select {
		case ev := <-backendChan:
			if !strings.HasPrefix(ev.Path(), absName) {
				panic("bug: BasicFilesystem watch received event outside of the watched path")
			}
			relPath := f.unrooted(ev.Path())
			if ignore.ShouldIgnore(relPath) {
				continue
			}
			outChan <- Event{Name: relPath, Type: f.eventType(ev.Event())}
		case <-ctx.Done():
			notify.Stop(backendChan)
			return
		}
	}
}

func (f *BasicFilesystem) eventType(notifyType notify.Event) EventType {
	if notifyType&rmEventMask != 0 {
		return Remove
	}
	return NonRemove
}
